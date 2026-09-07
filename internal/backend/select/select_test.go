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

package selectbackend_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend/exec"
	selectbackend "github.com/leinardi/monmux/internal/backend/select"
	"github.com/leinardi/monmux/internal/refusal"
)

// The backend is chosen by the operating system and by nothing else. Whatever
// this host is, New either returns a backend for it or refuses; it never
// returns nil with no error, and never falls back to another platform's tool.
func TestNewEitherReturnsABackendOrRefuses(t *testing.T) {
	t.Parallel()

	chosen, err := selectbackend.New()
	if err == nil {
		if chosen == nil {
			t.Fatal("New() returned no backend and no error")
		}

		if chosen.Name() == "" {
			t.Error("the chosen backend has no name")
		}

		return
	}

	if chosen != nil {
		t.Error("New() returned both a backend and an error")
	}

	if !refusal.Is(err, refusal.BackendUnavailable) {
		t.Errorf("error = %v, want backend-unavailable", err)
	}

	if !strings.HasSuffix(err.Error(), "No DDC write was performed.") {
		t.Errorf("the refusal does not state that nothing was written:\n%s", err)
	}
}

// On a platform monmux has no backend for, the refusal says which one it is.
func TestUnsupportedPlatformsAreNamed(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		t.Skipf("%s is a supported platform", runtime.GOOS)
	}

	_, err := selectbackend.New()
	if err == nil {
		t.Fatalf("New() returned a backend on %s", runtime.GOOS)
	}

	if !strings.Contains(err.Error(), runtime.GOOS) {
		t.Errorf("the refusal does not name the platform:\n%s", err)
	}
}

// The concrete backends are wired in by the phases that add them. Until then
// both supported platforms refuse, and this test fails the moment the backend
// package exists but its constructor has not been called from here - so the
// wiring cannot be forgotten.
func TestTheConcreteBackendIsWiredInOnceItExists(t *testing.T) {
	t.Parallel()

	packages := map[string]string{
		"linux":  "internal/backend/ddcutil",
		"darwin": "internal/backend/m1ddc",
	}

	expected, supported := packages[runtime.GOOS]
	if !supported {
		t.Skipf("%s has no backend of its own", runtime.GOOS)
	}

	_, err := os.Stat(filepath.Join(moduleRoot(t), expected))
	if err != nil {
		t.Skipf("%s does not exist yet", expected)
	}

	chosen, err := selectbackend.New()
	if err != nil || chosen == nil {
		t.Fatalf("%s exists but New() still refuses on %s: %v", expected, runtime.GOOS, err)
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

// Options exists so the configuration file can pin a tool path and so tests can
// supply a runner. Passing them must not change which platform is supported.
func TestOptionsDoNotChangeWhichPlatformIsSupported(t *testing.T) {
	t.Parallel()

	withDefaults, defaultErr := selectbackend.New()

	withOptions, optionErr := selectbackend.NewWithOptions(selectbackend.Options{
		Runner:      exec.NewFake(),
		DDCUtilPath: "/usr/bin/ddcutil",
		M1DDCPath:   "/opt/homebrew/bin/m1ddc",
	})

	if (defaultErr == nil) != (optionErr == nil) {
		t.Errorf("options changed the outcome: %v vs %v", defaultErr, optionErr)
	}

	if (withDefaults == nil) != (withOptions == nil) {
		t.Error("options changed whether a backend was returned")
	}
}
