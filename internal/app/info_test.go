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

package app_test

import (
	"slices"
	"testing"

	"github.com/leinardi/monmux/internal/app"
	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/refusal"
)

// checks is what a healthy backend's doctor reports.
func checks() []backend.Check {
	return []backend.Check{{Name: "ddcutil binary", OK: true, Detail: "/usr/bin/ddcutil"}}
}

func TestInfoReportsEveryDisplay(t *testing.T) {
	t.Parallel()

	driver := &backend.Fake{
		FakeName: "ddcutil",
		Displays: []backend.Display{supported(), unsupported(), unwritable()},
		Checks:   checks(),
	}

	report, err := app.Info(t.Context(), driver)
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	if report.Backend != "ddcutil" || len(report.Checks) != 1 {
		t.Errorf("report = %+v", report)
	}

	if len(report.Displays) != 3 {
		t.Fatalf("reported %d displays, want 3", len(report.Displays))
	}

	identified := report.Displays[0]
	if identified.Match != catalog.MatchExact ||
		identified.Model.FullName() != "LG 38WR85QC-W" {
		t.Errorf("the supported display was not identified: %+v", identified)
	}

	want := []catalog.Input{catalog.InputDP, catalog.InputUSBC}
	if !slices.Equal(identified.EnabledInputs, want) {
		t.Errorf("enabled inputs = %v, want %v", identified.EnabledInputs, want)
	}

	unknown := report.Displays[1]
	if unknown.Match != catalog.MatchNone || len(unknown.EnabledInputs) != 0 {
		t.Errorf("an unknown monitor was reported as switchable: %+v", unknown)
	}
}

// A display monmux will not write to is still reported, with the status saying
// why. Leaving it out would look like the monitor was not attached at all.
func TestInfoReportsUnwritableDisplaysWithTheirStatus(t *testing.T) {
	t.Parallel()

	driver := &backend.Fake{Displays: []backend.Display{unwritable()}, Checks: checks()}

	report, err := app.Info(t.Context(), driver)
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	if len(report.Displays) != 1 {
		t.Fatalf("reported %d displays, want 1", len(report.Displays))
	}

	reported := report.Displays[0]

	if reported.Display.Writable {
		t.Error("an unwritable display was reported as writable")
	}

	if reported.Display.Status != backend.StatusNoDDCChannel {
		t.Errorf("status = %q, want %q", reported.Display.Status, backend.StatusNoDDCChannel)
	}

	// The model is recognized - that is worth showing - but nothing is offered
	// for a display monmux would refuse to write to.
	if reported.Match != catalog.MatchExact {
		t.Errorf("match = %s, want exact", reported.Match)
	}

	if len(reported.EnabledInputs) != 0 {
		t.Errorf(
			"inputs %v were offered for a display monmux will not write to",
			reported.EnabledInputs,
		)
	}
}

func TestInfoNeverWrites(t *testing.T) {
	t.Parallel()

	driver := &backend.Fake{Displays: []backend.Display{supported()}, Checks: checks()}

	_, err := app.Info(t.Context(), driver)
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	for _, call := range driver.Calls() {
		if call == backend.CallExecute || call == backend.CallPlan {
			t.Errorf("info called %s: %v", call, driver.Calls())
		}
	}

	if len(driver.Executed()) != 0 {
		t.Errorf("info executed %v", driver.Executed())
	}
}

// A report that comes back with an error still carries everything that could be
// collected. That is the point of the command: the state a user most needs
// `info` for is the broken one.
func TestInfoReportsWhatItCanWhenSomethingFailed(t *testing.T) {
	t.Parallel()

	broken := []struct {
		name         string
		driver       *backend.Fake
		want         refusal.Reason
		wantDisplays int
	}{
		// On Linux the displays come from sysfs and a missing or too-old
		// ddcutil stops no enumeration at all, so the monitors are listed next
		// to the check that explains why none of them can be switched yet.
		{
			name: "the tool is not usable but the displays are still visible",
			driver: &backend.Fake{
				Checks:       checks(),
				Displays:     []backend.Display{supported()},
				PreflightErr: refusal.New(refusal.BackendNotReady, "no ddcutil"),
			},
			want:         refusal.BackendNotReady,
			wantDisplays: 1,
		},
		{
			name: "the displays cannot be listed",
			driver: &backend.Fake{
				Checks:       checks(),
				EnumerateErr: refusal.New(refusal.EnumerationFailed, "no sysfs"),
			},
			want:         refusal.EnumerationFailed,
			wantDisplays: 0,
		},
	}

	for _, testCase := range broken {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			report, err := app.Info(t.Context(), testCase.driver)
			if !refusal.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %s", err, testCase.want)
			}

			if len(report.Checks) != 1 {
				t.Errorf("the checks were dropped on the error path: %+v", report)
			}

			if len(report.Displays) != testCase.wantDisplays {
				t.Errorf(
					"reported %d displays on the error path, want %d",
					len(report.Displays),
					testCase.wantDisplays,
				)
			}

			if !slices.Contains(testCase.driver.Calls(), backend.CallDoctor) {
				t.Errorf("doctor was not run: %v", testCase.driver.Calls())
			}

			if !slices.Contains(testCase.driver.Calls(), backend.CallEnumerate) {
				t.Errorf("enumeration was skipped: %v", testCase.driver.Calls())
			}
		})
	}
}
