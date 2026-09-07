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

package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/config"
)

func TestParseReadsEveryKey(t *testing.T) {
	t.Parallel()

	document := `serial: TESTSERIAL01
ddcutil_path: /usr/local/bin/ddcutil
m1ddc_path: /opt/homebrew/bin/m1ddc
`

	parsed, err := config.Parse([]byte(document))
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	want := config.Config{
		Serial:      "TESTSERIAL01",
		DDCUtilPath: "/usr/local/bin/ddcutil",
		M1DDCPath:   "/opt/homebrew/bin/m1ddc",
	}

	if parsed != want {
		t.Errorf("config = %+v, want %+v", parsed, want)
	}
}

// A key that is silently ignored would leave the user believing a pin is in
// effect when it is not, and for a serial pin that means writing to a monitor
// they meant to exclude.
func TestParseRefusesAnUnknownKey(t *testing.T) {
	t.Parallel()

	_, err := config.Parse([]byte("serail: TESTSERIAL01\n"))
	if err == nil {
		t.Fatal("a misspelled key was accepted")
	}

	if !errors.Is(err, config.ErrUnreadable) {
		t.Errorf("error = %v, want it to wrap ErrUnreadable", err)
	}

	if !strings.Contains(err.Error(), "serail") {
		t.Errorf("the error does not name the offending key: %v", err)
	}
}

func TestParseAcceptsAnEmptyDocument(t *testing.T) {
	t.Parallel()

	for _, document := range []string{"", "\n", "# nothing but a comment\n"} {
		parsed, err := config.Parse([]byte(document))
		if err != nil {
			t.Errorf("Parse(%q) failed: %v", document, err)
		}

		if parsed != (config.Config{}) {
			t.Errorf("Parse(%q) = %+v, want the zero configuration", document, parsed)
		}
	}
}

// monmux works without a configuration file, so its absence is not an error.
func TestLoadFromTreatsAMissingFileAsNoConfiguration(t *testing.T) {
	t.Parallel()

	parsed, err := config.LoadFrom(filepath.Join(t.TempDir(), config.FileName))
	if err != nil {
		t.Fatalf("a missing file was reported as an error: %v", err)
	}

	if parsed != (config.Config{}) {
		t.Errorf("config = %+v, want the zero configuration", parsed)
	}
}

func TestLoadFromNamesTheFileItCouldNotUse(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), config.FileName)

	err := os.WriteFile(path, []byte("serial: [not, a, string]\n"), 0o600)
	if err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}

	_, err = config.LoadFrom(path)
	if err == nil {
		t.Fatal("an unusable configuration was accepted")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("the error does not name the file: %v", err)
	}
}

// The location is the same on Linux and macOS: a user with both machines should
// not have to remember two.
func TestPathFollowsTheConfigurationHome(t *testing.T) {
	home := t.TempDir()

	t.Setenv(config.XDGConfigHome, home)

	path, err := config.Path()
	if err != nil {
		t.Fatalf("Path() failed: %v", err)
	}

	want := filepath.Join(home, config.Directory, config.FileName)
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	t.Setenv(config.XDGConfigHome, "")
	t.Setenv("HOME", home)

	path, err = config.Path()
	if err != nil {
		t.Fatalf("Path() failed: %v", err)
	}

	want = filepath.Join(home, ".config", config.Directory, config.FileName)
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestLoadReadsTheFileFromTheConfigurationHome(t *testing.T) {
	home := t.TempDir()

	t.Setenv(config.XDGConfigHome, home)

	directory := filepath.Join(home, config.Directory)

	err := os.MkdirAll(directory, 0o755)
	if err != nil {
		t.Fatalf("creating %s: %v", directory, err)
	}

	err = os.WriteFile(
		filepath.Join(directory, config.FileName),
		[]byte("serial: TESTSERIAL01\n"),
		0o600,
	)
	if err != nil {
		t.Fatalf("writing the configuration: %v", err)
	}

	parsed, err := config.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if parsed.Serial != "TESTSERIAL01" {
		t.Errorf("serial = %q, want TESTSERIAL01", parsed.Serial)
	}
}
