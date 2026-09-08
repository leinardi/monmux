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
// source, and into the generated sections of the compatibility document. It is
// invoked by the //go:generate directive in internal/catalog, and through
// `make go-generate`. It is a development-time tool: it is not part of the
// monmux binary and never touches a monitor.
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
	document := flag.String("doc", "", "the compatibility document to splice into, if any")

	flag.Parse()

	err := run(*input, *output, *document)
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalog generator: %v\n", err)
		os.Exit(1)
	}
}

// run reads the catalog file, renders it, and writes the results. The document
// is optional: with no -doc, this renders the Go file alone.
func run(input, output, document string) error {
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

	if document == "" {
		return nil
	}

	return spliceDocument(source, document)
}

// spliceDocument rewrites the generated sections of the compatibility document
// from the same catalog file the Go source came from. It reads the document
// first because everything outside the generated region is prose a human writes,
// and this only replaces what lies between the markers.
func spliceDocument(source []byte, document string) error {
	existing, err := os.ReadFile(document)
	if err != nil {
		return fmt.Errorf("reading %s: %w", document, err)
	}

	spliced, err := generate.GenerateDocument(source, existing)
	if err != nil {
		return fmt.Errorf("rendering %s: %w", document, err)
	}

	//nolint:gosec // G703: -doc is a path a developer passes to a build-time tool; it never runs in the shipped binary
	err = os.WriteFile(document, spliced, generatedFileMode)
	if err != nil {
		return fmt.Errorf("writing %s: %w", document, err)
	}

	return nil
}
