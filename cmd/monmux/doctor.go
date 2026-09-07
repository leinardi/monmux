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
	"github.com/spf13/cobra"

	"github.com/leinardi/monmux/internal/app"
)

// newDoctorCmd returns the `doctor` subcommand, which writes nothing.
func newDoctorCmd(state *cli) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report whether monmux can reach the monitors at all",
		Long: "Checks the backend's external tool and every attached display, and prints one\n" +
			"line per check. Everything it does is read-only: no monitor is written to and\n" +
			"none is asked what input it is currently on.",
		Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return state.doctor(command)
		},
	}
}

// doctor prints the backend's diagnostics, and then whatever went wrong
// producing them. The checks come first because they are the answer: an error
// here is usually the very thing the checks describe.
func (c *cli) doctor(command *cobra.Command) error {
	driver, _, err := c.open()
	if err != nil {
		return err
	}

	report, infoErr := app.Info(command.Context(), driver)

	page := &printer{out: command.OutOrStdout()}

	page.linef("Backend: %s", report.Backend)
	page.linef("")
	renderChecks(page, report.Checks)

	if page.err != nil {
		return page.err
	}

	if infoErr != nil {
		return &readOnlyError{Err: infoErr}
	}

	return nil
}
