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

// Command relnotes compares two versions of the supported-monitor catalog and
// prints the section that goes at the top of the release notes: what the
// release newly writes, and what it only records. The release workflow passes
// the result to goreleaser as the release header, and reads the enabled=
// line to enforce the rule that a release which enables a model or an input is
// never a patch.
//
// It is a development-time tool, like the generator it sits next to. It reads
// two files, writes markdown and touches no monitor. It imports
// internal/catalog/internal/generate for the schema and never imports
// internal/catalog, for the reason that package's generator documents.
//
// Usage:
//
//	relnotes [-previous v0.1.0] [-status path] <previous models.yaml> <current models.yaml>
//
// An empty previous file is an empty catalog, which is what the first release
// compares against.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/leinardi/monmux/internal/catalog/internal/generate"
)

// statusFileMode is the permission the status file is created with when it does
// not exist yet. In the release workflow it always does: it is $GITHUB_OUTPUT.
const statusFileMode = 0o644

// expectedArguments is the number of catalog files the tool compares.
const expectedArguments = 2

// errUsage reports the wrong number of positional arguments.
var errUsage = errors.New("relnotes: expected two catalog files")

func main() {
	previous := flag.String(
		"previous", "the previous release", "how the notes name the release being compared against",
	)
	status := flag.String(
		"status", "", "file to append the enabled=true or enabled=false line to; stderr when empty",
	)

	flag.Parse()

	err := run(flag.Args(), *previous, *status, os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalog release notes: %v\n", err)
		os.Exit(1)
	}
}

// run compares the two catalog files named on the command line, writes the
// markdown to out, and reports whether the release enables anything.
func run(paths []string, previous, status string, out io.Writer) error {
	if len(paths) != expectedArguments {
		return fmt.Errorf(
			"%w: usage: relnotes [flags] <previous models.yaml> <current models.yaml>", errUsage,
		)
	}

	before, err := load(paths[0])
	if err != nil {
		return err
	}

	after, err := load(paths[1])
	if err != nil {
		return err
	}

	comparison := compare(before, after)

	_, err = io.WriteString(out, comparison.markdown(previous))
	if err != nil {
		return fmt.Errorf("writing the release notes: %w", err)
	}

	return writeStatus(status, comparison.enabling())
}

// load reads and decodes one catalog file. A file with nothing in it is an
// empty catalog rather than an error: the first release has no previous tag to
// read a models.yaml out of, and the workflow hands an empty file over for it.
func load(path string) (generate.Document, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return generate.Document{}, fmt.Errorf("reading %s: %w", path, err)
	}

	if strings.TrimSpace(string(source)) == "" {
		return generate.Document{}, nil
	}

	document, err := generate.Parse(source)
	if err != nil {
		return generate.Document{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	return document, nil
}

// writeStatus reports whether the release enables a model or an input, as one
// line the shell can read. It is appended rather than written over: the file
// the workflow names is $GITHUB_OUTPUT, which every step of the job shares.
func writeStatus(path string, enabling bool) error {
	line := fmt.Sprintf("enabled=%t\n", enabling)

	if path == "" {
		_, err := fmt.Fprint(os.Stderr, line)
		if err != nil {
			return fmt.Errorf("reporting the status: %w", err)
		}

		return nil
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, statusFileMode)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}

	_, err = file.WriteString(line)
	if err != nil {
		_ = file.Close()

		return fmt.Errorf("writing %s: %w", path, err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}

	return nil
}
