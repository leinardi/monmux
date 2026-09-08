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

package m1ddc

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/refusal"
)

// binary stands in for the path Preflight resolved. Nothing ever runs it: these
// tests only build and inspect commands.
const binary = "/opt/homebrew/bin/m1ddc"

// operation returns the catalog's USB-C operation for the tested model.
func operation(t *testing.T) catalog.Operation {
	t.Helper()

	model, result := catalog.Match(listed(t)[0].display.Identity)
	if result != catalog.MatchExact {
		t.Fatalf("match = %s, want exact", result)
	}

	op, enabled := model.Operation(catalog.InputUSBC)
	if !enabled {
		t.Fatalf("%s is no longer enabled for USB-C", model.FullName())
	}

	return op
}

// This is the golden test: these are the exact arguments monmux hands m1ddc for
// the tested LG 38WR85QC-W. If a refactor changes them, it changes what reaches
// a monitor, and that must never happen quietly.
func TestGoldenCommandForTheTestedModel(t *testing.T) {
	t.Parallel()

	records := listed(t)

	command, err := planFor(binary, &records[0], operation(t))
	if err != nil {
		t.Fatalf("planFor() refused: %v", err)
	}

	// 209 is 0xD1: the catalog records the VCP value in hex and m1ddc takes it
	// in decimal.
	wantArgs := []string{"display", testedUUID, "set", "input-alt", "209"}

	if command.Path != binary {
		t.Errorf("path = %q, want %q", command.Path, binary)
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
		t.Errorf("redact = %v, want [1]: the UUID identifies the physical unit", command.Redact)
	}

	if strings.Contains(command.String(), testedUUID) {
		t.Errorf("the default rendering leaked the UUID:\n%s", command)
	}

	if command.Render(true) != binary+" display "+testedUUID+" set input-alt 209" {
		t.Errorf("--show-serial rendering = %q", command.Render(true))
	}
}

func TestPlanRejectsAnOperationTheCatalogDidNotProduce(t *testing.T) {
	t.Parallel()

	var forged catalog.Operation

	records := listed(t)

	_, err := planFor(binary, &records[0], forged)
	if !refusal.Is(err, refusal.InvalidOperation) {
		t.Errorf("error = %v, want invalid-operation", err)
	}
}

func TestPlanRejectsAMechanismThisBackendDoesNotImplement(t *testing.T) {
	t.Parallel()

	// The catalog's only mechanism today is the one this backend implements, so
	// the guard is checked through the interface's own validator, with a
	// mechanism no backend claims.
	err := backend.ValidateOperation(operation(t), catalog.Mechanism("standard-vcp-60"))
	if !refusal.Is(err, refusal.InvalidOperation) {
		t.Errorf("error = %v, want invalid-operation", err)
	}
}

// A Mac with nothing attached is not a broken tool. m1ddc says so with a
// non-zero exit status, and both preflight and enumeration must read that as an
// empty list, so policy refuses with no-displays rather than backend-not-ready.
func TestTheNoDisplayAnswerIsAnEmptyList(t *testing.T) {
	t.Parallel()

	result := exec.Result{Stdout: fixture(t, emptyFixture), ExitCode: 1}

	for _, reason := range []refusal.Reason{refusal.BackendNotReady, refusal.EnumerationFailed} {
		records, err := interpretList(reason, result, fmt.Errorf("exit status 1: %w", errFailed))
		if err != nil {
			t.Errorf("the no-display answer was refused as %s: %v", reason, err)
		}

		if len(records) != 0 {
			t.Errorf("the no-display answer yielded %d displays", len(records))
		}
	}
}

// errFailed stands in for the error the runner returns for a non-zero exit.
var errFailed = errors.New("m1ddc failed")

func TestInterpretListRefusesWithTheCallersReason(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		result exec.Result
		err    error
		reason refusal.Reason
		want   refusal.Reason
	}{
		"a failed run during preflight": {
			result: exec.Result{Stderr: "boom", ExitCode: 2},
			err:    errFailed,
			reason: refusal.BackendNotReady,
			want:   refusal.BackendNotReady,
		},
		"a failed run during enumeration": {
			result: exec.Result{Stderr: "boom", ExitCode: 2},
			err:    errFailed,
			reason: refusal.EnumerationFailed,
			want:   refusal.EnumerationFailed,
		},
		"a failed run before a write": {
			result: exec.Result{Stderr: "boom", ExitCode: 2},
			err:    errFailed,
			reason: refusal.IdentityChanged,
			want:   refusal.IdentityChanged,
		},
		"unrecognized output": {
			result: exec.Result{Stdout: "m1ddc 2.0 says hello\n"},
			err:    nil,
			reason: refusal.BackendNotReady,
			want:   refusal.BackendNotReady,
		},
		// A tool that never started says nothing about the displays, and it is
		// always the same problem whoever asked.
		"a tool that never started": {
			result: exec.Result{ExitCode: -1},
			err:    exec.ErrExecutionDisabled,
			reason: refusal.EnumerationFailed,
			want:   refusal.BackendNotReady,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := interpretList(testCase.reason, testCase.result, testCase.err)
			if !refusal.Is(err, testCase.want) {
				t.Errorf("error = %v, want %s", err, testCase.want)
			}
		})
	}
}

// m1ddc's numbering is only true while the list is in the order it printed, so
// what Enumerate reports keeps that order rather than sorting it.
func TestDisplaysKeepTheOrderTheToolPrinted(t *testing.T) {
	t.Parallel()

	records := listed(t)

	listedDisplays := displays(records)
	if len(listedDisplays) != len(records) {
		t.Fatalf("reported %d displays for %d records", len(listedDisplays), len(records))
	}

	for index, display := range listedDisplays {
		if display != records[index].display {
			t.Errorf("display %d is %+v, want %+v", index, display, records[index].display)
		}

		want := "display " + strconv.Itoa(index+1)
		if display.Label != want {
			t.Errorf("display %d is labeled %q, want %q", index, display.Label, want)
		}
	}
}

func TestLookupRefusesDisplaysThatCannotBeWrittenTo(t *testing.T) {
	t.Parallel()

	records := listed(t)
	known, statuses := writable(records)

	if len(known) != 1 {
		t.Fatalf("indexed %d writable displays, want 1", len(known))
	}

	found, err := lookup(known, statuses, &records[0].display)
	if err != nil {
		t.Fatalf("the writable display was refused: %v", err)
	}

	if found.uuid != testedUUID {
		t.Errorf("uuid = %q, want %q", found.uuid, testedUUID)
	}

	// Enumerated, but with no UUID to address it by.
	_, err = lookup(known, statuses, &records[1].display)
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Errorf("error = %v, want target-not-ready", err)
	}

	// Never enumerated at all.
	stranger := backend.Display{Handle: "00000000-0000-4000-8000-000000000009", Label: "display 9"}

	_, err = lookup(known, statuses, &stranger)
	if !refusal.Is(err, refusal.TargetNotReady) {
		t.Errorf("error = %v, want target-not-ready", err)
	}
}

// The check that runs immediately before the write: the UUID must still name
// the display it named when it was identified.
func TestVerifyRefusesADisplayThatIsNoLongerTheSameOne(t *testing.T) {
	t.Parallel()

	records := listed(t)
	target := records[0].display

	err := verify(records, &target)
	if err != nil {
		t.Fatalf("an unchanged display was refused: %v", err)
	}

	changed, err := parseDisplayList(
		strings.Replace(fixture(t, listFixture), "30676 (0x77d4)", "4660 (0x1234)", 1),
	)
	if err != nil {
		t.Fatalf("the altered capture no longer parses: %v", err)
	}

	err = verify(changed, &target)
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Errorf("error = %v, want identity-changed", err)
	}

	// The display is gone entirely: nothing answers to that UUID any more.
	err = verify(changed[1:], &target)
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Errorf("error = %v, want identity-changed", err)
	}

	err = verify(nil, &target)
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Errorf("error = %v, want identity-changed", err)
	}
}

// A display with no UUID must not be matched by an empty handle: that would
// re-verify one monitor and switch another.
func TestVerifyNeverMatchesAnEmptyUUID(t *testing.T) {
	t.Parallel()

	records := listed(t)
	target := records[1].display

	err := verify(records, &target)
	if !refusal.Is(err, refusal.IdentityChanged) {
		t.Errorf("error = %v, want identity-changed", err)
	}
}

// The one distinction monmux must never get wrong: a tool that never started
// wrote nothing, and a tool that ran and failed may have written.
func TestExecutionOutcomeSeparatesNotStartedFromFailed(t *testing.T) {
	t.Parallel()

	target := listed(t)[0].display

	err := executionOutcome(&target, exec.Result{}, nil)
	if err != nil {
		t.Errorf("a successful run reported %v", err)
	}

	err = executionOutcome(&target, exec.Result{ExitCode: -1}, exec.ErrExecutionDisabled)
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Errorf("a tool that never started reported %v, want backend-not-ready", err)
	}

	err = executionOutcome(&target, exec.Result{ExitCode: 3}, errFailed)

	if _, ok := errors.AsType[*refusal.Refusal](err); ok {
		t.Errorf("a tool that ran and failed was reported as a refusal: %v", err)
	}

	if !strings.Contains(err.Error(), "status 3") {
		t.Errorf("error = %v, want it to name the exit status", err)
	}
}

func TestInstalledWhereExpected(t *testing.T) {
	t.Parallel()

	expected := []string{"/opt/homebrew/bin/m1ddc", "/usr/local/bin/m1ddc"}
	for _, path := range expected {
		if !installedWhereExpected(path) {
			t.Errorf("%s is a package manager's location but was not recognized", path)
		}
	}

	unexpected := []string{"/tmp/m1ddc", "/home/someone/bin/m1ddc", "/opt/homebrewery/m1ddc"}
	for _, path := range unexpected {
		if installedWhereExpected(path) {
			t.Errorf("%s was treated as a package manager's location", path)
		}
	}
}
