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

	"github.com/spf13/cobra"
)

// These values are printed by `monmux version` and overridden at build time with
// -ldflags, e.g.:
//
//	go build -ldflags "-X main.version=1.2.3 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%d)"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// newVersionCmd returns the `version` subcommand, which reports the build metadata
// baked in by GO_LDFLAGS.
func newVersionCmd() *cobra.Command {
	var asJSON bool

	command := &cobra.Command{
		Use:   "version",
		Short: "Print the monmux version and build metadata",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if asJSON {
				return writeJSON(command.OutOrStdout(), jsonVersion{
					Version: version,
					Commit:  commit,
					Date:    date,
				}, "version")
			}

			_, err := fmt.Fprintf(
				command.OutOrStdout(),
				"monmux %s (commit %s, built %s)\n",
				version,
				commit,
				date,
			)
			if err != nil {
				return fmt.Errorf("writing version output: %w", err)
			}

			return nil
		},
	}

	command.Flags().BoolVar(&asJSON, "json", false, "print the version as JSON")

	return command
}

// jsonVersion is the whole of `monmux version --json`. It is the output
// contract, kept separate from the build variables on purpose: a variable
// renamed inside monmux must not silently rename itself in somebody's script.
//
// Every field is a plain string, whatever the build put there: an unreleased
// build reports "dev", "none" and "unknown", so a client can tell a development
// binary from a released one without the document changing shape.
type jsonVersion struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}
