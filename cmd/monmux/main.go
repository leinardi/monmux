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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// exitFailure is the exit code used while the command tree is still a stub. The
// full exit-code contract (0 sent, 2 refused, 1 tool error) arrives with the
// switch command.
const exitFailure = 1

// newRootCmd builds the monmux command tree.
func newRootCmd() *cobra.Command {
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

	root.AddCommand(newVersionCmd())

	return root
}

func main() {
	err := newRootCmd().Execute()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "Error:", err)

		os.Exit(exitFailure)
	}
}
