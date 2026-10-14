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

// Package reporoot finds the repository root for the meta-tests under test/. They read the
// repository's own files -- workflows, make fragments, Markdown -- and never start a process: no
// test in monmux does (docs/testing.md).
package reporoot

import (
	"os"
	"path/filepath"
	"testing"
)

// Find returns the repository root, found as the nearest ancestor of the working directory that
// holds go.mod.
func Find(tb testing.TB) string {
	tb.Helper()

	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("reporoot: working directory: %v", err)
	}

	for {
		_, statErr := os.Stat(filepath.Join(dir, "go.mod"))
		if statErr == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("reporoot: no go.mod above the working directory")
		}

		dir = parent
	}
}

// ReadFile reads a file by its slash-separated path relative to the repository root.
func ReadFile(tb testing.TB, rel string) string {
	tb.Helper()

	data, err := os.ReadFile(filepath.Join(Find(tb), filepath.FromSlash(rel)))
	if err != nil {
		tb.Fatalf("read %s: %v", rel, err)
	}

	return string(data)
}
