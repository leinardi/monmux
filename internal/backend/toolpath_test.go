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

// The tool-path trust check is the one thing both backends share and neither
// owns, and the backends themselves are build-tagged, so it is tested here: this
// file has no build tag and runs on every platform monmux is developed on.

package backend_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/refusal"
)

// tool is the binary the checks are applied to. It is never executed.
const tool = "pretend-ddcutil"

// place writes an executable file inside a directory nobody else can write to,
// and returns the path with its symlinks resolved - which is the form
// ResolveTool reports, and the form a caller must compare against.
func place(t *testing.T, directory string) string {
	t.Helper()

	err := os.MkdirAll(directory, 0o755)
	if err != nil {
		t.Fatalf("creating %s: %v", directory, err)
	}

	path := filepath.Join(directory, tool)

	err = os.WriteFile(path, []byte("#!/bin/false\n"), 0o600)
	if err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	// Executable, and writable by nobody but its owner: what a package manager
	// leaves behind, and what the checks are meant to accept.
	chmod(t, path, 0o755)

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolving %s: %v", path, err)
	}

	return resolved
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()

	err := os.Chmod(path, mode)
	if err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

func TestResolveToolAcceptsATrustedBinary(t *testing.T) {
	t.Parallel()

	want := place(t, filepath.Join(t.TempDir(), "bin"))

	got, err := backend.ResolveTool(tool, want, "ddcutil_path")
	if err != nil {
		t.Fatalf("a trusted binary was refused: %v", err)
	}

	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

// A configured path is the only way to override PATH, and it must be absolute:
// a relative one would depend on where monmux happened to be started from.
func TestResolveToolRequiresAnAbsoluteConfiguredPath(t *testing.T) {
	t.Parallel()

	_, err := backend.ResolveTool(tool, "bin/"+tool, "ddcutil_path")
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Fatalf("error = %v, want backend-not-ready", err)
	}

	if !strings.Contains(err.Error(), "ddcutil_path") {
		t.Errorf("the refusal does not name the configuration key: %v", err)
	}
}

func TestResolveToolRefusesAnUntrustedBinary(t *testing.T) {
	t.Parallel()

	cases := map[string]func(t *testing.T) string{
		"the file does not exist": func(t *testing.T) string {
			t.Helper()

			return filepath.Join(t.TempDir(), tool)
		},
		"the path is a directory": func(t *testing.T) string {
			t.Helper()

			return t.TempDir()
		},
		"the file is group-writable": func(t *testing.T) string {
			t.Helper()

			path := place(t, filepath.Join(t.TempDir(), "bin"))
			chmod(t, path, 0o775)

			return path
		},
		"the file is world-writable": func(t *testing.T) string {
			t.Helper()

			path := place(t, filepath.Join(t.TempDir(), "bin"))
			chmod(t, path, 0o757)

			return path
		},
		"the directory is group-writable": func(t *testing.T) string {
			t.Helper()

			path := place(t, filepath.Join(t.TempDir(), "bin"))
			chmod(t, filepath.Dir(path), 0o775)

			return path
		},
		// The case the symlink resolution exists for: the path itself looks
		// safe, and it points into a directory anyone can rewrite.
		"a safe path links into a writable directory": func(t *testing.T) string {
			t.Helper()

			root := t.TempDir()
			target := place(t, filepath.Join(root, "writable"))
			chmod(t, filepath.Dir(target), 0o777)

			link := filepath.Join(root, tool)

			err := os.Symlink(target, link)
			if err != nil {
				t.Fatalf("linking %s: %v", link, err)
			}

			return link
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := backend.ResolveTool(tool, build(t), "ddcutil_path")
			if !refusal.Is(err, refusal.BackendNotReady) {
				t.Errorf("error = %v, want backend-not-ready", err)
			}
		})
	}
}

// With nothing configured the binary comes from PATH, and the same trust checks
// apply to whatever is found there.
func TestResolveToolFindsTheBinaryOnPath(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "bin")
	want := place(t, directory)

	t.Setenv("PATH", directory)

	got, err := backend.ResolveTool(tool, "", "ddcutil_path")
	if err != nil {
		t.Fatalf("a trusted binary on PATH was refused: %v", err)
	}

	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}

	chmod(t, want, 0o777)

	_, err = backend.ResolveTool(tool, "", "ddcutil_path")
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("a world-writable binary on PATH was accepted: %v", err)
	}
}

func TestResolveToolRefusesABinaryThatIsNotOnPath(t *testing.T) {
	t.Setenv("PATH", filepath.Join(t.TempDir(), "empty"))

	_, err := backend.ResolveTool(tool, "", "ddcutil_path")
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("error = %v, want backend-not-ready", err)
	}
}

func TestFingerprintIdentifiesTheFile(t *testing.T) {
	t.Parallel()

	path := place(t, filepath.Join(t.TempDir(), "bin"))

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	sum := sha256.Sum256(contents)

	got, err := backend.Fingerprint(path)
	if err != nil {
		t.Fatalf("Fingerprint() failed: %v", err)
	}

	if got != hex.EncodeToString(sum[:]) {
		t.Errorf("fingerprint = %q, want %q", got, hex.EncodeToString(sum[:]))
	}

	_, err = backend.Fingerprint(filepath.Join(t.TempDir(), "absent"))
	if err == nil {
		t.Error("a missing file was fingerprinted")
	}
}
