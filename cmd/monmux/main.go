/*
 * Copyright 2026 Roberto Leinardi.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package main is the monmux command-line entry point. monmux switches supported
// monitors between video inputs, and refuses to write anything unless the attached
// monitor is positively identified as a model in the built-in catalog and the
// requested input is explicitly enabled for it.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/leinardi/monmux/internal/backend"
	selectbackend "github.com/leinardi/monmux/internal/backend/select"
	"github.com/leinardi/monmux/internal/config"
	"github.com/leinardi/monmux/internal/refusal"
)

// The exit codes are part of the interface: a script needs to tell "the command
// was sent" from "monmux declined to send it" from "the tool ran and failed",
// and the third case is the only one where a write may have happened.
const (
	// exitSent means the input-switch command was sent, or a read-only command
	// succeeded.
	exitSent = 0
	// exitToolError means the external tool failed, or monmux could not be
	// asked to do the thing at all. Whether a write happened is stated in the
	// message.
	exitToolError = 1
	// exitRefused means monmux refused. No DDC write was performed.
	exitRefused = 2
)

// dependencies are the things the command tree reaches outside itself for. They
// are injected so the tests can drive the whole CLI against a fake backend
// without a monitor, a configuration file or an external binary anywhere.
type dependencies struct {
	// Config loads the configuration file.
	Config func() (config.Config, error)
	// Backend returns the backend for this operating system.
	Backend func(config.Config) (backend.Backend, error)
}

// production is what the real command uses.
func production() dependencies {
	return dependencies{
		Config: config.Load,
		Backend: func(configured config.Config) (backend.Backend, error) {
			return selectbackend.NewWithOptions(selectbackend.Options{
				DDCUtilPath: configured.DDCUtilPath,
				M1DDCPath:   configured.M1DDCPath,
			})
		},
	}
}

// cli is the state every subcommand shares.
type cli struct {
	deps dependencies
	// showSerial prints serial numbers, display UUIDs and raw EDID hex instead
	// of masking them. It is off by default: monmux's normal output is safe to
	// paste into a bug report.
	showSerial bool
}

// open loads the configuration and returns the backend for this system.
//
//nolint:ireturn // the CLI works through the Backend contract by design
func (c *cli) open() (backend.Backend, config.Config, error) {
	configured, err := c.deps.Config()
	if err != nil {
		return nil, config.Config{}, err
	}

	driver, err := c.deps.Backend(configured)
	if err != nil {
		return nil, config.Config{}, err
	}

	return driver, configured, nil
}

// newRootCmd builds the monmux command tree.
func newRootCmd(state *cli) *cobra.Command {
	root := &cobra.Command{
		Use:   "monmux",
		Short: "Switch supported monitors between video inputs",
		Long: "monmux switches supported monitors between video inputs.\n\n" +
			"It writes only when the attached monitor is positively identified as a model in\n" +
			"the built-in supported-monitor catalog and the requested input is explicitly\n" +
			"enabled for that model. Everything else is refused without a write.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(
		&state.showSerial,
		"show-serial",
		false,
		"print serials, UUIDs and raw EDID hex instead of redacting them",
	)

	root.AddCommand(
		newInfoCmd(state),
		newSwitchCmd(state),
		newCatalogCmd(),
		newDoctorCmd(state),
		newVersionCmd(),
	)

	return root
}

// run executes one invocation and returns the process's exit code. main is a
// wrapper around it so that the tests can run the same code with their own
// arguments, streams and dependencies.
func run(args []string, stdout, stderr io.Writer, deps dependencies) int {
	state := &cli{deps: deps}

	root := newRootCmd(state)
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := root.Execute()
	if err == nil {
		return exitSent
	}

	_, _ = fmt.Fprintln(stderr, message(err, state.showSerial))

	return exitCode(err)
}

// readOnlyError marks a failure raised by a command that never intended to
// write anything. It still wraps whatever went wrong, so the exit code is
// unchanged; what changes is the wording, because "Refusing to switch input. …
// No DDC write was performed." is a strange thing to tell somebody who only
// asked what is attached.
type readOnlyError struct {
	// Err is the failure as the layer below reported it.
	Err error
}

// Error renders the underlying failure.
func (f *readOnlyError) Error() string {
	return f.Err.Error()
}

// Unwrap returns the underlying failure, so the exit code still comes from it.
func (f *readOnlyError) Unwrap() error {
	return f.Err
}

// message renders a failure for the user.
//
// A refusal renders itself: the reason, what was detected, and the closing
// promise that no DDC write was performed. Its identities are redacted unless
// the user asked for them.
func message(err error, showSerial bool) string {
	if readOnly, ok := errors.AsType[*readOnlyError](err); ok {
		return diagnostic(readOnly.Err)
	}

	if declined, ok := errors.AsType[*refusal.Refusal](err); ok {
		return declined.Render(showSerial)
	}

	return "Error: " + err.Error()
}

// diagnostic renders a failure of a read-only command in one line. What is wrong
// has already been printed in full by the checks above it; this says which of
// them stopped the command, and it never claims anything about a write.
func diagnostic(err error) string {
	var declined *refusal.Refusal
	if !errors.As(err, &declined) {
		return "Error: " + err.Error()
	}

	rendered := "Not ready: " + declined.Reason.String()
	if declined.Detail != "" {
		rendered += ": " + firstLine(declined.Detail)
	}

	return rendered
}

// exitCode maps a failure onto the process's exit status.
func exitCode(err error) int {
	if _, ok := errors.AsType[*refusal.Refusal](err); ok {
		return exitRefused
	}

	// Everything else - the tool ran and failed, the configuration is broken,
	// the arguments were wrong - is exit 1. Only a refusal carries the promise
	// that nothing was written, so only a refusal gets the code that means it,
	// and an ExecutionError, whose whole point is that the write status is
	// unknown, must never be mistaken for one.
	return exitToolError
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, production()))
}
