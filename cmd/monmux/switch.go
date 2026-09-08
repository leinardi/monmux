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
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/policy"
)

// newSwitchCmd returns the `switch` subcommand, the only one that writes.
func newSwitchCmd(state *cli) *cobra.Command {
	var (
		serial string
		dryRun bool
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
			"are attached; it matches the alphanumeric serial only.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: inputNames(),
		RunE: func(command *cobra.Command, args []string) error {
			return state.switchInput(command, args[0], serial, dryRun)
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

	return command
}

// switchInput performs one switch.
//
// The input is parsed before anything else happens: an argument that is not a
// well-formed connector name is a mistake in the request rather than something
// for the backend to discover, and nothing should be started for it. A name that
// is well formed but not enabled for the matched model is a different thing, and
// it is refused later, by the policy, with nothing written.
func (c *cli) switchInput(command *cobra.Command, name, serial string, dryRun bool) error {
	input, err := catalog.ParseInput(name)
	if err != nil {
		return fmt.Errorf(
			"%w (a connector kind - %s - optionally followed by a port number, e.g. hdmi2; "+
				"run \"monmux info\" to see the inputs of the attached monitor)",
			err,
			strings.Join(kindNames(), ", "),
		)
	}

	driver, configured, err := c.open()
	if err != nil {
		return err
	}

	// The flag wins over the file: the file is a standing preference, and the
	// flag is what the user is asking for right now.
	pinned := configured.Serial
	if serial != "" {
		pinned = serial
	}

	outcome, err := app.Switch(
		command.Context(),
		driver,
		policy.Request{Input: input, Serial: pinned},
		app.Options{DryRun: dryRun},
	)
	if err != nil {
		//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
		return err
	}

	return renderOutcome(command.OutOrStdout(), &outcome, c.showSerial)
}
