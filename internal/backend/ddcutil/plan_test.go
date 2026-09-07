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
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/refusal"
)

// This is the golden test: these are the exact bytes monmux sends to the tested
// LG 38WR85QC-W. If a refactor changes them, it changes what reaches a monitor,
// and that must never happen quietly.
func TestGoldenCommandForTheTestedModel(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	command, err := world.backend.Plan(displays[0], operation(t))
	if err != nil {
		t.Fatalf("Plan() refused: %v", err)
	}

	wantArgs := []string{
		"--edid", hex.EncodeToString(edidFixture(t)),
		"setvcp",
		"0xF4",
		"0xD1",
		"--i2c-source-addr=0x50",
		"--noverify",
	}

	if command.Path != world.binary {
		t.Errorf("path = %q, want %q", command.Path, world.binary)
	}

	if len(command.Args) != len(wantArgs) {
		t.Fatalf("args = %q, want %q", command.Args, wantArgs)
	}

	for index, arg := range command.Args {
		if arg != wantArgs[index] {
			t.Errorf("arg %d = %q, want %q", index, arg, wantArgs[index])
		}
	}

	if len(command.Redact) != 1 || command.Redact[0] != 1 {
		t.Errorf("redact = %v, want [1]: the EDID hex identifies the physical unit", command.Redact)
	}

	if strings.Contains(command.String(), "00ffffffffffff00") {
		t.Errorf("the default rendering leaked the EDID:\n%s", command)
	}
}

// Plan has no side effects: seeing what would happen must not make it happen.
func TestPlanRunsNothing(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	before := len(world.runner.Calls())

	_, err := world.backend.Plan(displays[0], operation(t))
	if err != nil {
		t.Fatalf("Plan() refused: %v", err)
	}

	if len(world.runner.Calls()) != before {
		t.Errorf("Plan() ran something: %v", world.runner.Lines()[before:])
	}
}

func TestPlanRejectsAnOperationTheCatalogDidNotProduce(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	var forged catalog.Operation

	_, err := world.backend.Plan(displays[0], forged)
	if !refusal.Is(err, refusal.InvalidOperation) {
		t.Errorf("error = %v, want invalid-operation", err)
	}

	err = world.backend.Execute(t.Context(), displays[0], forged)
	if !refusal.Is(err, refusal.InvalidOperation) {
		t.Errorf("Execute() error = %v, want invalid-operation", err)
	}

	if len(world.runner.Lines()) > 2 {
		t.Errorf("an invalid operation still ran something: %v", world.runner.Lines())
	}
}

func TestPlanRefusesBeforePreflight(t *testing.T) {
	t.Parallel()

	world := newWorld(t)

	_, err := world.backend.Plan(backend.Display{Handle: connector, Label: connector}, operation(t))
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("error = %v, want backend-not-ready", err)
	}
}

func TestPlanRefusesADisplayItNeverEnumerated(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	world.ready(t)

	stranger := backend.Display{Handle: "card9-DP-9", Label: "card9-DP-9", Writable: true}

	_, err := world.backend.Plan(stranger, operation(t))
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Errorf("error = %v, want target-not-ready", err)
	}
}

// The whole point of Plan and Execute sharing a planner: what is shown is what
// is run.
func TestExecuteRunsExactlyWhatPlanShowed(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	command, err := world.backend.Plan(displays[0], operation(t))
	if err != nil {
		t.Fatalf("Plan() refused: %v", err)
	}

	err = world.backend.Execute(t.Context(), displays[0], operation(t))
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	calls := world.runner.Calls()

	last := calls[len(calls)-1]
	if last.Line() != command.Render(true) {
		t.Errorf("ran\n%s\nbut planned\n%s", last.Line(), command.Render(true))
	}
}

func TestExecuteRunsOnceAndNeverRetries(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	err := world.backend.Execute(t.Context(), displays[0], operation(t))
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	writes := 0

	for _, line := range world.runner.Lines() {
		if strings.Contains(line, "setvcp") {
			writes++
		}
	}

	if writes != 1 {
		t.Errorf("setvcp ran %d times, want exactly 1: %v", writes, world.runner.Lines())
	}
}

// A display that moved, or that was swapped for another monitor, must not be
// written to - and the check happens before anything is sent.
func TestExecuteRefusesWhenTheEDIDChanged(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	swapped := edidFixture(t)
	swapped[11]++ // a different product code
	swapped[127] = checksum(swapped)

	write(t, filepath.Join(world.sysfs, connector, "edid"), swapped, 0o644)

	err := world.backend.Execute(t.Context(), displays[0], operation(t))
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Errorf("error = %v, want identity-changed", err)
	}

	assertNoWrite(t, world)
}

func TestExecuteRefusesWhenTheDisplayMovedBus(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	link := filepath.Join(world.sysfs, connector, "ddc")

	err := os.Remove(link)
	if err != nil {
		t.Fatalf("removing %s: %v", link, err)
	}

	err = os.Symlink("../../../i2c-9", link)
	if err != nil {
		t.Fatalf("relinking %s: %v", link, err)
	}

	err = world.backend.Execute(t.Context(), displays[0], operation(t))
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Errorf("error = %v, want identity-changed", err)
	}

	assertNoWrite(t, world)
}

func TestExecuteRefusesWhenTheDDCChannelDisappeared(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	displays := world.ready(t)

	err := os.Remove(filepath.Join(world.sysfs, connector, "ddc"))
	if err != nil {
		t.Fatalf("removing the ddc link: %v", err)
	}

	err = world.backend.Execute(t.Context(), displays[0], operation(t))
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Errorf("error = %v, want identity-changed", err)
	}

	assertNoWrite(t, world)
}

// assertNoWrite fails if anything that could switch an input was run.
func assertNoWrite(t *testing.T, world *world) {
	t.Helper()

	for _, line := range world.runner.Lines() {
		if strings.Contains(line, "setvcp") {
			t.Errorf("a write was attempted: %s", line)
		}
	}
}

// checksum returns the byte that makes an EDID block sum to zero modulo 256.
func checksum(block []byte) byte {
	var sum uint8
	for _, b := range block[:127] {
		sum += b
	}

	return -sum
}
