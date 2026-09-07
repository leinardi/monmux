//go:build darwin

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

// These tests exercise the macOS-only glue: the order the backend does things
// in, and what it stores between calls. Everything they drive is decided by the
// tag-free code tested in decide_test.go, which runs everywhere.
//
// The runner is a fake. No test in this repository executes m1ddc, and the real
// runner refuses to start a process from a test binary at all.
package m1ddc_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/backend/m1ddc"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

const (
	listFixture  = "testdata/display-list-detailed.txt"
	emptyFixture = "testdata/no-external-display.txt"

	testedUUID = "00000000-0000-4000-8000-000000000001"

	// listCommand and setCommand are what the fake runner matches on.
	listCommand = "display list detailed"
	setCommand  = "set input-alt"
)

// errFailed stands in for the error the runner returns for a non-zero exit.
var errFailed = errors.New("m1ddc failed")

// world is a fake system: an m1ddc binary that is never executed, and a runner
// answering from captured output.
type world struct {
	binary  string
	runner  *exec.Fake
	backend *m1ddc.Backend
}

// newWorld builds a system whose m1ddc reports the captured display list.
func newWorld(t *testing.T, rules ...exec.Rule) *world {
	t.Helper()

	if len(rules) == 0 {
		rules = []exec.Rule{
			{Match: listCommand, Result: exec.Result{Stdout: fixture(t, listFixture)}},
			{Match: setCommand, Result: exec.Result{}},
		}
	}

	built := &world{
		binary: filepath.Join(t.TempDir(), "bin", m1ddc.Name),
		runner: exec.NewFake(rules...),
	}

	err := os.MkdirAll(filepath.Dir(built.binary), 0o755)
	if err != nil {
		t.Fatalf("creating %s: %v", filepath.Dir(built.binary), err)
	}

	err = os.WriteFile(built.binary, []byte("#!/bin/false\n"), 0o755)
	if err != nil {
		t.Fatalf("writing %s: %v", built.binary, err)
	}

	// Preflight reports the path with its symlinks resolved, and on macOS a
	// temporary directory lives under /var, which is a link to /private/var.
	// Comparing against the unresolved path would fail for the wrong reason.
	built.binary, err = filepath.EvalSymlinks(built.binary)
	if err != nil {
		t.Fatalf("resolving %s: %v", built.binary, err)
	}

	built.backend = m1ddc.New(m1ddc.Options{Runner: built.runner, Path: built.binary})

	return built
}

// ready runs preflight and enumeration, which every write path depends on.
func (w *world) ready(t *testing.T) []backend.Display {
	t.Helper()

	err := w.backend.Preflight(t.Context())
	if err != nil {
		t.Fatalf("Preflight() refused a good system: %v", err)
	}

	displays, err := w.backend.Enumerate(t.Context())
	if err != nil {
		t.Fatalf("Enumerate() failed: %v", err)
	}

	return displays
}

func fixture(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return string(contents)
}

// operation returns the catalog's USB-C operation for the tested model.
func operation(t *testing.T, identity backend.Display) catalog.Operation {
	t.Helper()

	model, result := catalog.Match(identity.Identity)
	if result != catalog.MatchExact {
		t.Fatalf("match = %s, want exact", result)
	}

	op, enabled := model.Operation(catalog.InputUSBC)
	if !enabled {
		t.Fatalf("%s is no longer enabled for USB-C", model.FullName())
	}

	return op
}

// testedOperation returns the catalog's USB-C operation for the supported model,
// without needing the backend to have enumerated anything.
func testedOperation(t *testing.T) catalog.Operation {
	t.Helper()

	return operation(t, backend.Display{
		Identity: edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D4},
	})
}

func TestEnumerateReportsBothDisplays(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	displays := world.ready(t)
	if len(displays) != 2 {
		t.Fatalf("enumerated %d displays, want 2", len(displays))
	}

	if !displays[0].Writable || displays[1].Writable {
		t.Errorf(
			"writability = %t, %t; want true, false",
			displays[0].Writable,
			displays[1].Writable,
		)
	}

	if displays[0].Handle != testedUUID {
		t.Errorf("handle = %q, want %q", displays[0].Handle, testedUUID)
	}
}

// Everything a write path needs comes from the tool, so nothing may be planned
// or executed before preflight has resolved and probed it.
func TestNothingWorksBeforePreflight(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	_, err := world.backend.Enumerate(t.Context())
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("Enumerate() error = %v, want backend-not-ready", err)
	}

	_, err = world.backend.Plan(backend.Display{Handle: testedUUID}, testedOperation(t))
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("Plan() error = %v, want backend-not-ready", err)
	}

	if len(world.runner.Calls()) != 0 {
		t.Errorf("something ran before preflight: %v", world.runner.Lines())
	}
}

// Plan has no side effects: seeing what would happen must not make it happen.
func TestPlanRunsNothing(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	before := len(world.runner.Calls())

	command, err := world.backend.Plan(displays[0], operation(t, displays[0]))
	if err != nil {
		t.Fatalf("Plan() refused: %v", err)
	}

	if len(world.runner.Calls()) != before {
		t.Errorf("Plan() ran something: %v", world.runner.Lines()[before:])
	}

	if command.Path != world.binary {
		t.Errorf("path = %q, want %q", command.Path, world.binary)
	}
}

// The happy path: the display is re-checked, and then exactly one switch is
// sent - the same invocation Plan showed.
func TestExecuteReverifiesAndSendsOneSwitch(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)
	op := operation(t, displays[0])

	planned, err := world.backend.Plan(displays[0], op)
	if err != nil {
		t.Fatalf("Plan() refused: %v", err)
	}

	before := len(world.runner.Calls())

	err = world.backend.Execute(t.Context(), displays[0], op)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	ran := world.runner.Lines()[before:]
	if len(ran) != 2 {
		t.Fatalf("Execute() ran %v, want a re-verification and one switch", ran)
	}

	if !strings.Contains(ran[0], listCommand) {
		t.Errorf("the display was not re-verified before the write: %q", ran[0])
	}

	if ran[1] != planned.Render(true) {
		t.Errorf("executed %q, but --dry-run showed %q", ran[1], planned.Render(true))
	}
}

// The display behind the UUID changed between enumeration and the write. Nothing
// is sent.
func TestExecuteRefusesWhenTheIdentityChanged(t *testing.T) {
	t.Parallel()

	listing := fixture(t, listFixture)
	changed := strings.Replace(listing, "30676 (0x77d4)", "4660 (0x1234)", 1)

	world := newWorld(t,
		// Preflight and Enumerate see the display; the re-verification does not.
		exec.Rule{Match: listCommand, Result: exec.Result{Stdout: listing}, Times: 2},
		exec.Rule{Match: listCommand, Result: exec.Result{Stdout: changed}},
		exec.Rule{Match: setCommand, Result: exec.Result{}},
	)

	displays := world.ready(t)

	err := world.backend.Execute(t.Context(), displays[0], operation(t, displays[0]))
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Fatalf("Execute() error = %v, want identity-changed", err)
	}

	for _, line := range world.runner.Lines() {
		if strings.Contains(line, setCommand) {
			t.Errorf("a switch was sent after the identity changed: %q", line)
		}
	}
}

// A display that was enumerated but cannot be addressed is refused before
// anything runs.
func TestExecuteRefusesAnUnwritableDisplay(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	before := len(world.runner.Calls())

	err := world.backend.Execute(t.Context(), displays[1], operation(t, displays[0]))
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Errorf("Execute() error = %v, want target-not-ready", err)
	}

	if len(world.runner.Calls()) != before {
		t.Errorf("an unwritable display still ran something: %v", world.runner.Lines()[before:])
	}
}

func TestExecuteRejectsAnOperationTheCatalogDidNotProduce(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	before := len(world.runner.Calls())

	err := world.backend.Execute(t.Context(), displays[0], catalog.Operation{})
	if !refusal.Is(err, refusal.InvalidOperation) {
		t.Errorf("Execute() error = %v, want invalid-operation", err)
	}

	if len(world.runner.Calls()) != before {
		t.Errorf("an invalid operation still ran something: %v", world.runner.Lines()[before:])
	}
}

// A Mac with nothing attached is a working backend with no displays, not a
// backend that cannot run.
func TestPreflightAcceptsAMacWithNoDisplays(t *testing.T) {
	t.Parallel()

	world := newWorld(t, exec.Rule{
		Match:  listCommand,
		Result: exec.Result{Stdout: fixture(t, emptyFixture), ExitCode: 1},
		Err:    errFailed,
	})

	err := world.backend.Preflight(t.Context())
	if err != nil {
		t.Fatalf("Preflight() refused a Mac with no displays: %v", err)
	}

	displays, err := world.backend.Enumerate(t.Context())
	if err != nil {
		t.Fatalf("Enumerate() failed: %v", err)
	}

	if len(displays) != 0 {
		t.Errorf("enumerated %d displays, want none", len(displays))
	}
}

func TestPreflightRefusesAToolItCannotTrust(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	err := os.Chmod(world.binary, 0o777)
	if err != nil {
		t.Fatalf("chmod %s: %v", world.binary, err)
	}

	err = world.backend.Preflight(t.Context())
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("a world-writable binary was accepted: %v", err)
	}

	err = os.Remove(world.binary)
	if err != nil {
		t.Fatalf("removing %s: %v", world.binary, err)
	}

	err = world.backend.Preflight(t.Context())
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("a missing binary was accepted: %v", err)
	}
}

func TestPreflightRefusesOutputItDoesNotUnderstand(t *testing.T) {
	t.Parallel()

	world := newWorld(t, exec.Rule{
		Match:  listCommand,
		Result: exec.Result{Stdout: "m1ddc 2.0 says hello\n"},
	})

	err := world.backend.Preflight(t.Context())
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("Preflight() error = %v, want backend-not-ready", err)
	}
}

func TestDoctorReportsTheBinaryAndTheDisplays(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	checks := world.backend.Doctor(t.Context())

	details := map[string]backend.Check{}
	for _, check := range checks {
		details[check.Name] = check
	}

	binary, reported := details[m1ddc.Name+" binary"]
	if !reported || binary.Detail != world.binary {
		t.Errorf("doctor did not report the resolved binary: %+v", checks)
	}

	sum, reported := details[m1ddc.Name+" sha256"]
	if !reported || len(sum.Detail) != 64 {
		t.Errorf("doctor did not report a sha256: %+v", sum)
	}

	// The fake binary is in a temporary directory, which is exactly the case
	// the location note exists for.
	where, reported := details[m1ddc.Name+" location"]
	if !reported || where.OK {
		t.Errorf("doctor did not flag an unexpected install location: %+v", where)
	}

	displays, reported := details["displays"]
	if !reported || !strings.Contains(displays.Detail, "2 connected displays") {
		t.Errorf("doctor did not report the displays: %+v", displays)
	}
}

// Doctor is read-only: it may ask what is attached, but it must never switch
// anything.
func TestDoctorWritesNothing(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	world.backend.Doctor(t.Context())

	for _, line := range world.runner.Lines() {
		if strings.Contains(line, setCommand) {
			t.Errorf("doctor sent a switch: %q", line)
		}
	}
}
