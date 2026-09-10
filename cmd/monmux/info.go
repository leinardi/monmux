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
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/leinardi/monmux/internal/app"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
)

// newInfoCmd returns the `info` subcommand, which writes nothing.
func newInfoCmd(state *cli) *cobra.Command {
	var asJSON bool

	command := &cobra.Command{
		Use:   "info",
		Short: "List the attached displays and what monmux would do with them",
		Long: "Lists every attached display, whether monmux would write to it and why not,\n" +
			"which catalog entry it matched, which inputs are enabled for it, and the\n" +
			"backend's own diagnostics. Nothing is written and no monitor is asked what\n" +
			"input it is currently on.\n\n" +
			"Serial numbers and other private identifiers are redacted unless\n" +
			"--show-serial is given.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return state.info(command, asJSON)
		},
	}

	command.Flags().BoolVar(&asJSON, "json", false, "print the report as JSON")

	return command
}

// info prints the report, and then whatever went wrong producing it.
//
// The report is printed even when something failed, because it is the diagnostic
// for exactly those failures: a missing ddcutil does not stop Linux from listing
// the monitors, and seeing them next to the check that explains why none of them
// can be switched is the whole point of the command.
func (c *cli) info(command *cobra.Command, asJSON bool) error {
	driver, _, err := c.open()
	if err != nil {
		return err
	}

	report, infoErr := app.Info(command.Context(), driver)

	if asJSON {
		err = writeJSON(command.OutOrStdout(), asJSONReport(&report, c.showSerial), "report")
	} else {
		err = renderReport(command.OutOrStdout(), &report, c.showSerial)
	}

	if err != nil {
		return err
	}

	if infoErr != nil {
		return &readOnlyError{Err: infoErr}
	}

	return nil
}

// The JSON shapes below are the command's output contract, kept separate from
// the internal types on purpose: a field renamed inside monmux must not silently
// rename itself in somebody's script.
type (
	// jsonReport is the whole of `monmux info --json`.
	jsonReport struct {
		Backend  string        `json:"backend"`
		Displays []jsonDisplay `json:"displays"`
		Checks   []jsonCheck   `json:"checks"`
	}

	// jsonDisplay is one attached display, writable or not.
	jsonDisplay struct {
		Label string `json:"label"`
		// Handle is masked when it is private data, unless --show-serial.
		Handle        string        `json:"handle"`
		HandlePrivate bool          `json:"handlePrivate"`
		Writable      bool          `json:"writable"`
		Status        string        `json:"status"`
		Identity      edid.Identity `json:"identity"`
		Match         string        `json:"match"`
		Model         string        `json:"model"`
		EnabledInputs []string      `json:"enabledInputs"`
	}

	// jsonCheck is one line of the backend's diagnostics.
	jsonCheck struct {
		Name   string `json:"name"`
		OK     bool   `json:"ok"`
		Detail string `json:"detail"`
	}
)

// writeJSON prints one output document as JSON. The subject names what is being
// written, so a failure says which command's output could not be produced.
//
// HTML escaping is turned off: this output goes to a terminal or to jq, and
// nothing here is going into a web page. Left on, it would turn the redaction
// mask into "\u003credacted…\u003e", which is unreadable for the one field a
// user is most likely to be looking at.
func writeJSON(out io.Writer, document any, subject string) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(document)
	if err != nil {
		return fmt.Errorf("writing the %s as JSON: %w", subject, err)
	}

	return nil
}

// asJSONReport converts the report into the output contract, redacting whatever
// the user did not ask to see.
func asJSONReport(report *app.Report, showSerial bool) jsonReport {
	document := jsonReport{
		Backend:  report.Backend,
		Displays: make([]jsonDisplay, 0, len(report.Displays)),
		Checks:   make([]jsonCheck, 0, len(report.Checks)),
	}

	for index := range report.Displays {
		reported := &report.Displays[index]
		display := reported.Display

		identity := display.Identity
		if !showSerial {
			identity = identity.Redacted()
		}

		document.Displays = append(document.Displays, jsonDisplay{
			Label:         display.Label,
			Handle:        handle(&display, showSerial),
			HandlePrivate: display.HandlePrivate,
			Writable:      display.Writable,
			Status:        display.Status,
			Identity:      identity,
			Match:         reported.Match.String(),
			Model:         jsonModel(reported),
			EnabledInputs: jsonInputs(reported.EnabledInputs),
		})
	}

	for _, check := range report.Checks {
		document.Checks = append(document.Checks, jsonCheck{
			Name:   check.Name,
			OK:     check.OK,
			Detail: check.Detail,
		})
	}

	return document
}

// jsonModel is the matched model's name, or nothing at all when there is none.
// A blank name is the honest answer: the zero catalog entry has no name to print.
func jsonModel(reported *app.DisplayReport) string {
	if reported.Match != catalog.MatchExact {
		return ""
	}

	return reported.Model.FullName()
}

// jsonInputs renders the enabled inputs as their command-line names.
func jsonInputs(enabled []catalog.Input) []string {
	names := make([]string, 0, len(enabled))
	for _, input := range enabled {
		names = append(names, input.String())
	}

	return names
}
