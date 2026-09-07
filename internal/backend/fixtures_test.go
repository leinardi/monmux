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

package backend_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/edid"
)

// Fixtures are captures of real hardware, so they arrive carrying somebody's
// serial number. The rule is written down in testdata/README.md; this is what
// enforces it. Only these values may appear.
const (
	syntheticSerialNumber = 0x01020304
	syntheticSerialString = "TESTSERIAL01"
	syntheticBinarySerial = "16909060"
)

// allowedUUID is the shape every fixture UUID must have: the documented
// synthetic one, differing only in its final digit.
var allowedUUID = regexp.MustCompile(`^0{8}-0000-4000-8000-0{11}\d$`)

// uuidLike finds anything shaped like a UUID, so a real one cannot hide in a
// fixture unnoticed.
var uuidLike = regexp.MustCompile(
	`[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}`,
)

// serialLine matches a text fixture line that reports a serial.
var serialLine = regexp.MustCompile(`(?i)serial[^:]*:\s*(.+)`)

func TestFixturesCarryOnlySyntheticSerials(t *testing.T) {
	t.Parallel()

	// Every testdata directory in the module, not just this package's: a real
	// serial must not be able to slip in anywhere.
	root := moduleRoot(t)

	fixtures := []string{}
	directories := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}

			if entry.Name() == "testdata" {
				directories++
			}

			return nil
		}

		// Membership is decided by the whole path, not by the immediate
		// parent: a fixture nested inside testdata/ is still a fixture.
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolving %s against the module root: %w", path, err)
		}

		inTestdata := slices.Contains(
			strings.Split(relative, string(filepath.Separator)),
			"testdata",
		)
		if !inTestdata || entry.Name() == "README.md" {
			return nil
		}

		fixtures = append(fixtures, path)

		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	if directories == 0 || len(fixtures) == 0 {
		t.Fatalf(
			"inspected %d fixtures in %d testdata directories; this guard proves nothing",
			len(fixtures),
			directories,
		)
	}

	for _, path := range fixtures {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		if strings.HasSuffix(path, ".edid") {
			checkEDID(t, path, contents)

			continue
		}

		checkText(t, path, string(contents))
	}
}

// checkEDID parses a binary EDID fixture and asserts both of its serials are the
// synthetic ones.
func checkEDID(t *testing.T, path string, contents []byte) {
	t.Helper()

	identity, err := edid.Parse(contents)
	if err != nil {
		t.Errorf("%s does not parse as an EDID: %v", path, err)

		return
	}

	if identity.SerialNumber != syntheticSerialNumber {
		t.Errorf(
			"%s carries the numeric serial %#08X, want %#08X",
			path,
			identity.SerialNumber,
			syntheticSerialNumber,
		)
	}

	if identity.SerialString != syntheticSerialString {
		t.Errorf(
			"%s carries the serial string %q, want %q",
			path,
			identity.SerialString,
			syntheticSerialString,
		)
	}
}

// checkText asserts that every serial and every UUID in a text fixture is one of
// the documented synthetic values.
func checkText(t *testing.T, path, contents string) {
	t.Helper()

	for _, uuid := range uuidLike.FindAllString(contents, -1) {
		if !allowedUUID.MatchString(uuid) {
			t.Errorf("%s carries the UUID %s, which is not the synthetic one", path, uuid)
		}
	}

	for line := range strings.SplitSeq(contents, "\n") {
		match := serialLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		value := strings.TrimSpace(match[1])
		if value == syntheticSerialString || value == syntheticBinarySerial {
			continue
		}

		t.Errorf("%s reports the serial %q, which is not one of the synthetic values", path, value)
	}
}

// moduleRoot walks up to the directory holding go.mod.
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
