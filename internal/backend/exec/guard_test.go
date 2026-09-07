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

package exec_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// monmux tests never execute an external binary. The runner enforces that from
// the inside, and this test enforces it from the outside: no test file in the
// repository may reach for the process-starting machinery at all.
//
// The two files below are the guard's own test surface and may name the
// constructor they are guarding. Nothing else may, and no file anywhere is
// exempt from the import check.
var constructorExemptions = map[string]string{
	"internal/backend/exec/guard_test.go":  "this test",
	"internal/backend/exec/runner_test.go": "asserts that the real runner refuses to run",
}

// forbiddenImport is the package that starts processes.
const forbiddenImport = "os/exec"

// realRunnerConstructor is the only function that returns a runner which starts
// processes.
const realRunnerConstructor = "NewRunner"

func TestNoTestFileCanStartAProcess(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	checked := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolving %s against the module root: %w", path, err)
		}

		relative = filepath.ToSlash(relative)

		checked++

		inspect(t, path, relative)

		return nil
	})
	if err != nil {
		t.Fatalf("walking the repository: %v", err)
	}

	if checked == 0 {
		t.Fatal("no test files were checked, so this guard proves nothing")
	}
}

// inspect fails the test if one test file could start a process.
func inspect(t *testing.T, path, relative string) {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", relative, err)
	}

	parsed, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parsing %s: %v", relative, err)
	}

	for _, spec := range parsed.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("reading the imports of %s: %v", relative, err)
		}

		if imported == forbiddenImport {
			t.Errorf(
				"%s imports %s; tests never execute an external binary",
				relative,
				forbiddenImport,
			)
		}
	}

	reason, exempt := constructorExemptions[relative]
	if exempt {
		if !strings.Contains(string(source), "ErrExecutionDisabled") {
			t.Errorf(
				"%s may name %s only as %q, but it no longer asserts that execution is refused",
				relative,
				realRunnerConstructor,
				reason,
			)
		}

		return
	}

	if strings.Contains(string(source), realRunnerConstructor) {
		t.Errorf(
			"%s references %s; tests use the recording fake runner",
			relative,
			realRunnerConstructor,
		)
	}
}

// moduleRoot walks up from the test's directory to the directory holding go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()

	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("locating the working directory: %v", err)
	}

	for {
		_, err := os.Stat(filepath.Join(directory, "go.mod"))
		if err == nil {
			return directory
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("no go.mod found above the test's directory")
		}

		directory = parent
	}
}
