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
	"fmt"
	"strconv"
	"strings"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/refusal"
)

// This file holds every decision the backend takes: what to run, whether the
// tool answered acceptably, whether the display is still the one that was
// identified, and what an execution failure means. None of it needs macOS, so
// none of it is behind a build tag and all of it is tested on the Linux
// development host. The macOS-only files are the glue that performs the I/O.

// listArgs asks m1ddc what is attached. It is the only command monmux runs that
// is not a switch, and it reads nothing from the monitor over DDC: the display
// list comes from the IORegistry and CoreGraphics.
//
// `get input` and `get input-alt` do talk to the monitor, and monmux never runs
// them: no decision here depends on what the display says its current input is.
var listArgs = []string{displaySubcommand, "list", "detailed"}

// displaySubcommand is m1ddc's verb for everything that names a display.
const displaySubcommand = "display"

// trustedPrefixes are where a package manager puts m1ddc on macOS. A binary
// found elsewhere still runs - it may be a deliberate build - but doctor says so.
var trustedPrefixes = []string{"/opt/homebrew/", "/usr/local/"}

// plan builds the invocation. Plan and Execute both go through it, which is what
// makes what --dry-run prints the same thing a real run performs.
//
// The display is selected by its UUID. m1ddc's default identification method is
// uuid, and the parser only ever produces a UUID-shaped handle, so the value
// here can never be taken for the list index m1ddc also accepts in that
// position - which is the point, because that index moves when a monitor is
// plugged in.
//
// input-alt is the LG side channel, and its argument is decimal: the catalog
// records 0xD1, and m1ddc is handed 209.
func plan(path, uuid string, operation catalog.Operation) backend.Command {
	return backend.Command{
		Path: path,
		Args: append(
			[]string{displaySubcommand, uuid},
			arguments(operation.Mechanism(), operation.Value())...,
		),
		// The UUID identifies the physical unit.
		Redact: []int{1},
	}
}

// arguments builds the set invocation for one mechanism and one value. It is
// split out of [plan] so that a test can pin the exact argv for a value without
// an exported way to build an [catalog.Operation]: rendering argv is not the
// same power as making a backend run it, and only the catalog has the latter.
//
// m1ddc takes the value in decimal and sends both halves of the SH/SL pair, so
// a value wider than a byte needs no special handling here.
//
// A mechanism this backend does not implement yields nil rather than a guess.
// [plan] is reached only after [validate] has accepted the mechanism, so nil is
// unreachable there; it is the shape of "never fall back" rather than a branch
// that runs.
func arguments(mechanism catalog.Mechanism, value uint16) []string {
	switch mechanism {
	case catalog.MechanismLGAltInput:
		return []string{"set", "input-alt", strconv.FormatUint(uint64(value), 10)}
	default:
		return nil
	}
}

// validate is the check both Plan and Execute perform before anything else, so
// an operation this backend cannot perform is refused as invalid-operation
// whatever else is wrong - the same reason, in the same order, as the Linux
// backend gives for the same inputs.
func validate(operation catalog.Operation) error {
	//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
	return backend.ValidateOperation(operation, catalog.MechanismLGAltInput)
}

// planFor builds the invocation for a display the backend enumerated. It is the
// only place a command is built, so Plan cannot show one thing and Execute run
// another, and it validates the operation again rather than trusting its caller
// to have done it.
func planFor(path string, known *record, operation catalog.Operation) (backend.Command, error) {
	err := validate(operation)
	if err != nil {
		return backend.Command{}, err
	}

	return plan(path, known.uuid, operation), nil
}

// interpretList reads one `display list detailed` invocation.
//
// Preflight, Enumerate and the pre-write re-verification all go through it, so
// they cannot disagree about what the tool's answer meant; they differ only in
// the refusal reason a failure carries.
//
// An empty list is a legitimate answer. m1ddc reports "No external display
// found" with a non-zero exit status, and that is not a broken tool: it is a Mac
// with nothing plugged in, which policy should report as no-displays rather than
// as a backend that cannot run.
func interpretList(reason refusal.Reason, result exec.Result, err error) ([]record, error) {
	output := result.Stdout + result.Stderr

	if err != nil {
		// A tool that never started says nothing about the displays, and it is
		// always the same problem: the backend is not usable at all.
		if !exec.Started(result, err) {
			return nil, refusal.New(
				refusal.BackendNotReady,
				Name+" did not start: "+err.Error(),
			)
		}

		if noDisplaysReported(output) {
			return []record{}, nil
		}

		return nil, refusal.New(
			reason,
			Name+" "+strings.Join(listArgs, " ")+" failed: "+err.Error(),
		)
	}

	records, err := parseDisplayList(output)
	if err != nil {
		return nil, refusal.New(
			reason,
			Name+" "+strings.Join(listArgs, " ")+" produced output monmux does not recognize.",
		)
	}

	return records, nil
}

// displays returns what Enumerate reports for a set of records.
func displays(records []record) []backend.Display {
	listed := make([]backend.Display, 0, len(records))
	for index := range records {
		listed = append(listed, records[index].display)
	}

	return listed
}

// writable indexes the records that can be written to, by their handle, and the
// status of every record seen. Keeping both is what lets a later refusal tell
// "monmux never saw this display" from "monmux saw it and cannot write to it".
func writable(records []record) (known map[string]record, statuses map[string]string) {
	known = make(map[string]record, len(records))
	statuses = make(map[string]string, len(records))

	for index := range records {
		current := records[index]
		statuses[current.display.Handle] = current.display.Status

		if current.display.Writable {
			known[current.display.Handle] = current
		}
	}

	return known, statuses
}

// lookup returns what Enumerate learned about a display, refusing for one that
// cannot be written to or was never seen.
func lookup(
	known map[string]record,
	statuses map[string]string,
	display *backend.Display,
) (record, error) {
	current, ok := known[display.Handle]
	if ok {
		return current, nil
	}

	status, enumerated := statuses[display.Handle]
	if enumerated {
		return record{}, refusal.New(
			refusal.TargetNotReady,
			"Display "+display.Label+" was enumerated but cannot be written to (status: "+
				status+").",
			display.Identity,
		)
	}

	return record{}, refusal.New(
		refusal.TargetNotReady,
		"Display "+display.Label+" was not enumerated by this backend.",
		display.Identity,
	)
}

// verify re-checks, immediately before a write, that the UUID still names the
// display it named when it was identified.
//
// The UUID is stable for a given panel, so a UUID that now reports a different
// identity means the displays moved underneath monmux - and the input switch
// would land on the wrong monitor. Nothing is sent.
func verify(records []record, display *backend.Display) error {
	for index := range records {
		current := records[index]
		if current.uuid == "" || current.uuid != display.Handle {
			continue
		}

		if current.display.Identity != display.Identity {
			return refusal.New(
				refusal.IdentityChanged,
				"The display behind "+display.Label+
					" is no longer the one that was identified.",
				display.Identity,
			)
		}

		return nil
	}

	return refusal.New(
		refusal.IdentityChanged,
		"Display "+display.Label+" is no longer attached under the UUID it was identified by.",
		display.Identity,
	)
}

// executionOutcome says what a finished m1ddc run means.
//
// A tool that never started cannot have written anything, so that is a refusal
// like any other. A tool that ran and failed is not: the write may have gone
// out, and saying otherwise would be a lie.
func executionOutcome(display *backend.Display, result exec.Result, err error) error {
	if err == nil {
		return nil
	}

	if !exec.Started(result, err) {
		return refusal.New(
			refusal.BackendNotReady,
			Name+" did not start: "+err.Error(),
			display.Identity,
		)
	}

	return fmt.Errorf("%s exited with status %d: %w", Name, result.ExitCode, err)
}

// installedWhereExpected reports whether the binary sits where a package manager
// would have put it.
func installedWhereExpected(path string) bool {
	for _, prefix := range trustedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
