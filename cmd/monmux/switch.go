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

package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/leinardi/monmux/internal/app"
	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/policy"
)

// newSwitchCmd returns the `switch` subcommand, the only one that writes.
func newSwitchCmd(state *cli) *cobra.Command {
	var (
		serial      string
		unsafeModel string
		dryRun      bool
		asJSON      bool
	)

	command := &cobra.Command{
		Use:   "switch <input>",
		Short: "Switch the identified monitor to an input",
		Long: "Switches the attached monitor to the given input.\n\n" +
			"An input is a connector kind - " + strings.Join(kindNames(), ", ") + " -\n" +
			"optionally followed by a port number, as in hdmi2. Run \"monmux info\" to see\n" +
			"the inputs the attached monitor is enabled for.\n\n" +
			"The write happens only when exactly one attached display is positively\n" +
			"identified as a catalog model that is write-enabled for that input. Anything\n" +
			"else is refused, and a refusal means nothing was written.\n\n" +
			"With --dry-run the exact command that would run is printed and nothing is\n" +
			"executed. Use --serial to pick one physical unit when two identical monitors\n" +
			"are attached; it matches the alphanumeric serial only.\n\n" +
			"--unsafe-model is the one way past that. It treats the attached display as\n" +
			"the catalog entry you name, without identifying it and without the\n" +
			"write-enabled gate, and it sends a value nobody verified on your unit. The\n" +
			"entry is named VENDOR/MODEL, as \"monmux catalog list\" prints it, e.g.\n" +
			exampleModelName + ". Run it with --dry-run first, and have the monitor's OSD\n" +
			"within reach.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: inputNames(),
		RunE: func(command *cobra.Command, args []string) error {
			return state.switchInput(command, args[0], serial, unsafeModel, dryRun, asJSON)
		},
	}

	command.Flags().StringVar(
		&serial,
		"serial",
		"",
		"only switch the display whose alphanumeric serial is this, exactly",
	)
	command.Flags().BoolVar(
		&dryRun,
		"dry-run",
		false,
		"print the command that would run, and run nothing",
	)
	command.Flags().BoolVar(
		&asJSON,
		"json",
		false,
		"print the outcome as JSON, whatever it is; the exit code is unchanged",
	)
	command.Flags().StringVar(
		&unsafeModel,
		"unsafe-model",
		"",
		"DANGEROUS: treat the display as this catalog model, named VENDOR/MODEL as "+
			"\"monmux catalog list\" prints it, instead of identifying it; bypasses EDID "+
			"matching and the write-enabled gate",
	)

	// Completion offers every catalog entry, write-enabled or not: what the flag
	// is for is the entries monmux would otherwise refuse. The error is ignored
	// because the only way to get one is to name a flag that does not exist.
	_ = command.RegisterFlagCompletionFunc(
		"unsafe-model",
		func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return modelNames(), cobra.ShellCompDirectiveNoFileComp
		},
	)

	return command
}

// switchInput performs one switch and reports it, as prose or as JSON.
//
// With --json exactly one document is printed on stdout for every outcome, and
// the exit code is unchanged: the code stays the authority for "the write status
// is unknown", and the document is the authority for telling a dry run from a
// send, which the code cannot. Nothing is printed twice - a failure that has
// been reported as a document is not also rendered as prose on stderr.
func (c *cli) switchInput(
	command *cobra.Command,
	name, serial, unsafeModel string,
	dryRun, asJSON bool,
) error {
	outcome, backendName, err := c.performSwitch(command, name, serial, unsafeModel, dryRun)

	if !asJSON {
		if err != nil {
			return err
		}

		return renderOutcome(command.OutOrStdout(), &outcome, c.showSerial)
	}

	document := asJSONSwitch(&outcome, backendName, err, c.showSerial)

	writeErr := writeJSON(command.OutOrStdout(), document, "outcome")
	if writeErr != nil {
		return writeErr
	}

	if err != nil {
		// The document is the whole report, so the message must not be printed
		// a second time as prose on stderr. Only the exit code survives, and it
		// is the one the text path would have used: a refusal still exits 2.
		return &renderedError{code: exitCode(err)}
	}

	return nil
}

// performSwitch does the switch itself and reports what came of it, printing
// nothing but the --unsafe-model warning. Its caller decides how the result is
// rendered, so the text and JSON renderings cannot disagree about what happened.
//
// The input is parsed before anything else happens: an argument that is not a
// well-formed connector name is a mistake in the request rather than something
// for the backend to discover, and nothing should be started for it. A name that
// is well formed but not enabled for the matched model is a different thing, and
// it is refused later, by the policy, with nothing written.
//
// The returned backend name is the backend that handled the request, and is
// empty when the failure happened before one could be opened.
func (c *cli) performSwitch(
	command *cobra.Command,
	name, serial, unsafeModel string,
	dryRun bool,
) (app.Outcome, string, error) {
	input, err := catalog.ParseInput(name)
	if err != nil {
		return app.Outcome{}, "", fmt.Errorf(
			"%w (a connector kind - %s - optionally followed by a port number, e.g. hdmi2; "+
				"run \"monmux info\" to see the inputs of the attached monitor)",
			err,
			strings.Join(kindNames(), ", "),
		)
	}

	// The override names a catalog entry, so a name that is not one is a mistake
	// in the request, the same as an input name that does not parse: it is an
	// argument error rather than a refusal, and nothing is started for it.
	if unsafeModel != "" {
		_, known := catalog.Find(unsafeModel)
		if !known {
			return app.Outcome{}, "", unknownModel(unsafeModel)
		}
	}

	driver, configured, err := c.open()
	if err != nil {
		return app.Outcome{}, "", err
	}

	// The flag wins over the file: the file is a standing preference, and the
	// flag is what the user is asking for right now.
	pinned := configured.Serial
	if serial != "" {
		pinned = serial
	}

	// The warning is printed from inside the switch, once the display has been
	// selected and before anything is planned or written, so it can name the
	// monitor that is about to receive an unverified value. A dry run prints it
	// too. Note what is not consulted anywhere here: the configuration file has
	// no key for the override, so only this invocation's flag can have armed it.
	// bypassed records that the policy actually assumed a model, which is what
	// the callback firing means. It is not the same as the flag being set: a
	// request that never got that far bypassed no identification, and a request
	// that got past it did - including when the write then failed, which is the
	// one outcome where an unverified value may have reached the monitor.
	bypassed := false

	opts := app.Options{DryRun: dryRun}
	if unsafeModel != "" {
		opts.OnAssumed = func(
			display *backend.Display,
			model *catalog.Model,
			name catalog.Input,
		) error {
			bypassed = true

			return renderUnsafeWarning(command.ErrOrStderr(), display, model, name)
		}
	}

	backendName := driver.Name()

	outcome, err := app.Switch(
		command.Context(),
		driver,
		policy.Request{Input: input, Serial: pinned, AssumeModel: unsafeModel},
		opts,
	)
	if err != nil {
		// app.Switch zeroes the outcome on every error path, so what the run is
		// still known to have done is put back here.
		//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
		return app.Outcome{Assumed: bypassed}, backendName, err
	}

	return outcome, backendName, nil
}
