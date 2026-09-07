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

package app

import (
	"context"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
)

// Report is what monmux can see. It is the whole of what `info` and `doctor`
// print, and producing it writes nothing.
type Report struct {
	// Backend is the name of the backend for this operating system.
	Backend string
	// Checks are the backend's diagnostics. They are filled in even when the
	// report also comes back with an error, because a backend that cannot
	// enumerate is exactly when the user needs to see them.
	Checks []backend.Check
	// Displays is one entry per attached display, writable or not.
	Displays []DisplayReport
}

// DisplayReport is one display, together with what the catalog says about it.
type DisplayReport struct {
	// Display is what the backend reported.
	Display backend.Display
	// Model is the catalog entry it matched. It is meaningful only when Match
	// is [catalog.MatchExact]; otherwise it is the zero value, whose name is
	// blank, and nothing should be printed from it.
	Model catalog.Model
	// Match says whether the identity matched the catalog exactly, not at all,
	// or ambiguously.
	Match catalog.MatchResult
	// EnabledInputs are the inputs monmux would be willing to switch this
	// display to. It is empty for a display it will not write to, whether
	// because the model is unknown, the model is not write-enabled, or the
	// display itself is not writable.
	EnabledInputs []catalog.Input
}

// Info reports what monmux can see, and never writes.
//
// Unwritable displays are included, with the status saying why: "this monitor is
// attached and monmux will not switch it" is the answer the user came for, and
// leaving it out would look like the display was not there at all.
//
// Nothing here short-circuits. A report that came back with an error still
// carries everything that could be collected, and that is the point of the
// function: the state a user most needs `info` for is the broken one. On Linux a
// missing or too-old ddcutil stops no enumeration at all - the displays come
// from sysfs - so the monitors are listed next to the check that explains why
// none of them can be switched yet. Where a backend genuinely cannot enumerate
// without its tool, it refuses on its own and the display list is simply empty.
//
// The checks and the display list are two independent snapshots of the hardware:
// Doctor asks the system itself, and the enumeration that follows asks again.
// For a read-only report that is fine, and it is why `info` may cost a backend
// more than one look at the machine.
func Info(ctx context.Context, driver backend.Backend) (Report, error) {
	report := Report{Backend: driver.Name()}

	// Preflight comes first because on some platforms nothing can be
	// enumerated until the tool has been resolved. Doctor runs either way: it
	// is read-only, and its whole purpose is to explain a failure like this one.
	preflightErr := driver.Preflight(ctx)

	report.Checks = driver.Doctor(ctx)

	displays, enumerateErr := driver.Enumerate(ctx)

	report.Displays = make([]DisplayReport, 0, len(displays))

	for _, display := range displays {
		report.Displays = append(report.Displays, describe(display))
	}

	// A tool that is not usable is the root cause when both failed, so it is
	// the one reported.
	if preflightErr != nil {
		//nolint:wrapcheck // a backend's refusal is passed through unchanged; wrapping would corrupt its message
		return report, preflightErr
	}

	if enumerateErr != nil {
		//nolint:wrapcheck // a backend's refusal is passed through unchanged; wrapping would corrupt its message
		return report, enumerateErr
	}

	return report, nil
}

// describe matches one display against the catalog.
//
//nolint:gocritic // hugeParam: a Display is passed by value everywhere else too
func describe(display backend.Display) DisplayReport {
	model, result := catalog.Match(display.Identity)

	reported := DisplayReport{Display: display, Model: model, Match: result}

	// A display monmux cannot write to has no enabled inputs, whatever the
	// catalog says about the model: reporting inputs for it would read as an
	// offer monmux would refuse.
	if result == catalog.MatchExact && display.Writable {
		reported.EnabledInputs = model.EnabledInputs()
	}

	return reported
}
