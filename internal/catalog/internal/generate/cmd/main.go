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

// Command cmd renders the supported-monitor catalog from models.yaml into Go
// source. It is invoked by the //go:generate directive in internal/catalog, and
// through `make go-generate`. It is a development-time tool: it is not part of
// the monmux binary and never touches a monitor.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/leinardi/monmux/internal/catalog/internal/generate"
)

// generatedFileMode is the permission the generated Go file is written with.
const generatedFileMode = 0o644

func main() {
	input := flag.String("in", "models.yaml", "the catalog file to read")
	output := flag.String("out", "models_gen.go", "the Go file to write")

	flag.Parse()

	err := run(*input, *output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalog generator: %v\n", err)
		os.Exit(1)
	}
}

// run reads the catalog file, renders it, and writes the result.
func run(input, output string) error {
	source, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("reading %s: %w", input, err)
	}

	rendered, err := generate.Generate(source)
	if err != nil {
		return fmt.Errorf("rendering %s: %w", input, err)
	}

	//nolint:gosec // G703: -out is a path a developer passes to a build-time tool; it never runs in the shipped binary
	err = os.WriteFile(output, rendered, generatedFileMode)
	if err != nil {
		return fmt.Errorf("writing %s: %w", output, err)
	}

	return nil
}
