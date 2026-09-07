//go:build linux

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

package ddcutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/ddcutil"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/refusal"
)

func TestPreflightAcceptsATrustedRecentDdcutil(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	err := world.backend.Preflight(t.Context())
	if err != nil {
		t.Fatalf("Preflight() refused a good system: %v", err)
	}

	// Both probes are read-only.
	lines := world.runner.Lines()
	if len(lines) != 2 {
		t.Fatalf("preflight ran %v, want --version and --help only", lines)
	}

	for _, line := range lines {
		if strings.Contains(line, "setvcp") {
			t.Errorf("preflight wrote something: %s", line)
		}
	}
}

func TestPreflightRefusals(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		arrange func(t *testing.T, world *world) *ddcutil.Backend
		detail  string
	}{
		"version below the floor": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				return world.rebuild(t, exec.NewFake(
					exec.Rule{Match: "--version", Result: exec.Result{Stdout: "ddcutil 2.1.4\n"}},
					exec.Rule{Match: "--help", Result: exec.Result{Stdout: helpOutput}},
				), world.binary)
			},
			detail: "older than the required 2.2",
		},
		"unreadable version": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				return world.rebuild(t, exec.NewFake(
					exec.Rule{Match: "--version", Result: exec.Result{Stdout: "ddcutil\n"}},
					exec.Rule{Match: "--help", Result: exec.Result{Stdout: helpOutput}},
				), world.binary)
			},
			detail: "version number",
		},
		"missing capability": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				return world.rebuild(t, exec.NewFake(
					exec.Rule{Match: "--version", Result: exec.Result{Stdout: versionOutput}},
					exec.Rule{Match: "--help", Result: exec.Result{Stdout: "  --noverify\n"}},
				), world.binary)
			},
			detail: "--i2c-source-addr",
		},
		"binary anyone can rewrite": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				chmod(t, world.binary, 0o777)

				return world.backend
			},
			detail: "group- or world-writable",
		},
		"binary in a directory anyone can rewrite": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				chmod(t, filepath.Dir(world.binary), 0o777)

				return world.backend
			},
			detail: "group- or world-writable",
		},
		"symlink into a directory anyone can rewrite": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				exposed := filepath.Join(t.TempDir(), "exposed")
				mkdir(t, exposed)

				target := filepath.Join(exposed, ddcutil.Name)
				write(t, target, []byte("#!/bin/false\n"), 0o755)
				chmod(t, exposed, 0o777)

				link := filepath.Join(filepath.Dir(world.binary), "linked-ddcutil")

				err := os.Symlink(target, link)
				if err != nil {
					t.Fatalf("linking %s to %s: %v", link, target, err)
				}

				return world.rebuild(t, world.runner, link)
			},
			detail: "group- or world-writable",
		},
		"configured path is not absolute": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				return world.rebuild(t, world.runner, "bin/ddcutil")
			},
			detail: "must be an absolute path",
		},
		"configured path is a directory": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				return world.rebuild(t, world.runner, filepath.Dir(world.binary))
			},
			detail: "not a regular file",
		},
		"configured path does not exist": {
			arrange: func(t *testing.T, world *world) *ddcutil.Backend {
				t.Helper()

				return world.rebuild(
					t,
					world.runner,
					filepath.Join(world.sysfs, "nowhere", "ddcutil"),
				)
			},
			detail: "cannot be inspected",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			world := newWorld(t)
			under := test.arrange(t, world)

			err := under.Preflight(t.Context())
			if !refusal.Is(err, refusal.BackendNotReady) {
				t.Fatalf("error = %v, want backend-not-ready", err)
			}

			if !strings.Contains(err.Error(), test.detail) {
				t.Errorf("message does not mention %q:\n%s", test.detail, err)
			}
		})
	}
}

func TestEnumerateReportsWhatItCannotWriteTo(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	// A connector with no ddc link, one whose EDID will not parse, and a
	// disconnected one that must not appear at all.
	world.addDisplay(t, "card1-DP-2", "", edidFixture(t))
	world.addDisplay(t, "card1-HDMI-A-1", bus, []byte("not an edid"))
	world.addDisconnected(t, "card1-DP-3")

	displays := world.ready(t)

	want := map[string]struct {
		writable bool
		status   string
	}{
		connector:        {writable: true, status: backend.StatusOK},
		"card1-DP-2":     {writable: false, status: backend.StatusNoDDCChannel},
		"card1-HDMI-A-1": {writable: false, status: backend.StatusEDIDUnreadable},
	}

	if len(displays) != len(want) {
		t.Fatalf("enumerated %d displays, want %d: %+v", len(displays), len(want), displays)
	}

	for _, display := range displays {
		expected, known := want[display.Handle]
		if !known {
			t.Errorf("unexpected display %q", display.Handle)

			continue
		}

		if display.Writable != expected.writable || display.Status != expected.status {
			t.Errorf(
				"%s: writable=%t status=%q, want %t and %q",
				display.Handle,
				display.Writable,
				display.Status,
				expected.writable,
				expected.status,
			)
		}

		if display.HandlePrivate {
			t.Errorf("%s: a Linux connector name is not private data", display.Handle)
		}
	}
}

func TestEnumerateFailsLoudlyWhenSysfsIsMissing(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	under := ddcutil.New(ddcutil.Options{
		Runner:    world.runner,
		Path:      world.binary,
		SysfsRoot: filepath.Join(world.sysfs, "nowhere"),
		DevRoot:   world.dev,
	})

	_, err := under.Enumerate(t.Context())
	if !refusal.Is(err, refusal.EnumerationFailed) {
		t.Errorf("error = %v, want enumeration-failed", err)
	}
}

func TestReadyRefusesWhenTheBusIsMissing(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	err := os.Remove(filepath.Join(world.dev, bus))
	if err != nil {
		t.Fatalf("removing the bus: %v", err)
	}

	err = world.backend.Ready(t.Context(), displays[0])
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Errorf("error = %v, want target-not-ready", err)
	}
}

func TestReadyRefusesWhenTheBusCannotBeOpened(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root can open anything, so this check proves nothing here")
	}

	world := newWorld(t)
	displays := world.ready(t)

	chmod(t, filepath.Join(world.dev, bus), 0o000)

	err := world.backend.Ready(t.Context(), displays[0])
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Fatalf("error = %v, want target-not-ready", err)
	}

	if !strings.Contains(err.Error(), "i2c group") {
		t.Errorf("the refusal does not say how to fix it:\n%s", err)
	}
}

func TestReadyAcceptsAnOpenableBus(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	err := world.backend.Ready(t.Context(), displays[0])
	if err != nil {
		t.Errorf("Ready() refused a usable bus: %v", err)
	}
}

func TestDoctorReportsThePathAndFingerprintWithoutWriting(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	checks := world.backend.Doctor(t.Context())

	seen := map[string]backend.Check{}
	for _, check := range checks {
		seen[check.Name] = check
	}

	binary, reported := seen["ddcutil binary"]
	if !reported || binary.Detail != world.binary {
		t.Errorf("doctor did not report the binary path: %+v", binary)
	}

	sum, reported := seen["ddcutil sha256"]
	if !reported || len(sum.Detail) != 64 {
		t.Errorf("doctor did not report a sha256: %+v", sum)
	}

	display, reported := seen[connector]
	if !reported || !display.OK {
		t.Errorf("doctor did not report the display as usable: %+v", display)
	}

	for _, line := range world.runner.Lines() {
		if strings.Contains(line, "setvcp") {
			t.Errorf("doctor wrote something: %s", line)
		}
	}
}

// The most likely diagnostic scenario is a missing or unusable ddcutil, and the
// user needs to be told which. A doctor that only says "refused" is useless.
func TestDoctorNamesWhatFailedAndKeepsGoing(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	under := world.rebuild(t, world.runner, filepath.Join(world.sysfs, "nowhere", "ddcutil"))

	checks := under.Doctor(t.Context())

	seen := map[string]backend.Check{}
	for _, check := range checks {
		seen[check.Name] = check
	}

	binary, reported := seen["ddcutil binary"]
	if !reported || binary.OK {
		t.Fatalf("doctor did not report the missing binary: %+v", binary)
	}

	if strings.Contains(binary.Detail, "Refusing to switch input") {
		t.Errorf("doctor reported the refusal headline instead of the cause: %q", binary.Detail)
	}

	if !strings.Contains(binary.Detail, "backend-not-ready") ||
		!strings.Contains(binary.Detail, "cannot be inspected") {
		t.Errorf("doctor did not say why the binary is unusable: %q", binary.Detail)
	}

	// The displays do not depend on the tool, and are exactly what the user
	// needs to see next.
	display, reported := seen[connector]
	if !reported {
		t.Errorf("doctor stopped at the binary and never reported the displays: %+v", checks)
	}

	if !display.OK {
		t.Errorf("the display is reachable but was reported as failing: %+v", display)
	}
}

// A failing target check must point at the target, with the hint that fixes it.
func TestDoctorReportsAnUnreachableBus(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	world.ready(t)

	err := os.Remove(filepath.Join(world.dev, bus))
	if err != nil {
		t.Fatalf("removing the bus: %v", err)
	}

	checks := world.backend.Doctor(t.Context())

	for _, check := range checks {
		if check.Name != connector {
			continue
		}

		if check.OK {
			t.Fatalf("doctor called an unreachable display usable: %+v", check)
		}

		if !strings.Contains(check.Detail, "target-not-ready") {
			t.Errorf("doctor did not name the reason: %q", check.Detail)
		}

		return
	}

	t.Errorf("doctor never mentioned %s: %+v", connector, checks)
}

// An enumerated display monmux cannot write to is a different fact from a
// display it never saw, and the refusal has to say which.
func TestUnwritableDisplaysAreRefusedAsSuch(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	world.addDisplay(t, "card1-DP-2", "", edidFixture(t))

	displays := world.ready(t)

	var unwritable backend.Display

	for _, display := range displays {
		if display.Handle == "card1-DP-2" {
			unwritable = display
		}
	}

	if unwritable.Handle == "" {
		t.Fatal("the unwritable display was not enumerated")
	}

	err := world.backend.Ready(t.Context(), unwritable)
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Fatalf("Ready() error = %v, want target-not-ready", err)
	}

	if !strings.Contains(err.Error(), "was enumerated but cannot be written to") {
		t.Errorf("the refusal claims the display was never seen:\n%s", err)
	}

	if !strings.Contains(err.Error(), backend.StatusNoDDCChannel) {
		t.Errorf("the refusal does not report the status:\n%s", err)
	}

	_, err = world.backend.Plan(unwritable, operation(t))
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Errorf("Plan() error = %v, want target-not-ready", err)
	}
}
