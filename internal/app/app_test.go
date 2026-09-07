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
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/app"
	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/policy"
	"github.com/leinardi/monmux/internal/refusal"
)

// errTool stands in for whatever a backend reports when its tool ran and failed.
var errTool = errors.New("exit status 1")

// supported is the tested LG, identified and writable.
func supported() backend.Display {
	return backend.Display{
		Identity: edid.Identity{
			Manufacturer: "GSM",
			ProductCode:  0x77D4,
			SerialNumber: 0x01020304,
			SerialString: "TESTSERIAL01",
			ModelName:    "LG ULTRAWIDE",
		},
		Handle:   "card1-DP-1",
		Label:    "card1-DP-1",
		Writable: true,
		Status:   backend.StatusOK,
	}
}

// unsupported is a monitor the catalog knows nothing about.
func unsupported() backend.Display {
	return backend.Display{
		Identity: edid.Identity{Manufacturer: "XXX", ProductCode: 0x1234},
		Handle:   "card1-HDMI-A-1",
		Label:    "card1-HDMI-A-1",
		Writable: true,
		Status:   backend.StatusOK,
	}
}

// unwritable is the tested LG on a connector with no DDC channel.
func unwritable() backend.Display {
	display := supported()
	display.Handle = "card1-DP-2"
	display.Label = "card1-DP-2"
	display.Writable = false
	display.Status = backend.StatusNoDDCChannel

	return display
}

// request is what the user asked for in most of these tests.
func request() policy.Request {
	return policy.Request{Input: catalog.InputUSBC}
}

// fake returns a backend that reports the given displays and does everything it
// is asked.
func fake(displays ...backend.Display) *backend.Fake {
	return &backend.Fake{FakeName: "ddcutil", Displays: displays}
}

func TestSwitchRunsEveryStepInOrderAndSendsOnce(t *testing.T) {
	t.Parallel()

	driver := fake(supported(), unsupported())

	outcome, err := app.Switch(t.Context(), driver, request(), app.Options{})
	if err != nil {
		t.Fatalf("Switch() refused a supported display: %v", err)
	}

	want := []string{
		backend.CallPreflight,
		backend.CallEnumerate,
		backend.CallReady,
		backend.CallPlan,
		backend.CallExecute,
	}

	if !slices.Equal(driver.Calls(), want) {
		t.Errorf("calls = %v, want %v", driver.Calls(), want)
	}

	if !outcome.Sent || outcome.DryRun {
		t.Errorf("outcome = %+v, want it sent and not a dry run", outcome)
	}

	// Everything the CLI needs to say what happened, without asking again.
	if outcome.Backend != "ddcutil" || outcome.Input != catalog.InputUSBC {
		t.Errorf("outcome = %+v", outcome)
	}

	if outcome.Model.FullName() != "LG 38WR85QC-W" {
		t.Errorf("model = %q, want LG 38WR85QC-W", outcome.Model.FullName())
	}

	if outcome.Display.Handle != supported().Handle {
		t.Errorf("switched %q, want %q", outcome.Display.Handle, supported().Handle)
	}

	if outcome.Operation.Value() != 0xD1 {
		t.Errorf("value = 0x%02X, want 0xD1", outcome.Operation.Value())
	}

	executed := driver.Executed()
	if len(executed) != 1 || executed[0] != outcome.Operation {
		t.Errorf("executed %v, want exactly the operation the catalog produced", executed)
	}

	// What was written to matters as much as what was written: the unsupported
	// display was enumerated alongside the supported one.
	targets := driver.Targets()
	if len(targets) != 1 || targets[0].Handle != supported().Handle {
		t.Errorf("wrote to %v, want only %q", targets, supported().Handle)
	}

	if outcome.Command.Path == "" {
		t.Error("the outcome carries no command, so nothing can be printed")
	}
}

// A dry run must reach the point where the command exists and stop there.
func TestDryRunBuildsTheCommandAndWritesNothing(t *testing.T) {
	t.Parallel()

	driver := fake(supported())

	outcome, err := app.Switch(t.Context(), driver, request(), app.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Switch() refused a dry run: %v", err)
	}

	want := []string{
		backend.CallPreflight,
		backend.CallEnumerate,
		backend.CallReady,
		backend.CallPlan,
	}

	if !slices.Equal(driver.Calls(), want) {
		t.Errorf("calls = %v, want %v", driver.Calls(), want)
	}

	if outcome.Sent || !outcome.DryRun {
		t.Errorf("outcome = %+v, want a dry run that was not sent", outcome)
	}

	if outcome.Command.Path == "" {
		t.Error("a dry run produced no command to print")
	}

	if len(driver.Executed()) != 0 {
		t.Errorf("a dry run executed %v", driver.Executed())
	}
}

// Every way a switch can be refused, and the one thing they all have in common:
// no write was attempted.
func TestSwitchRefusesWithTheReasonOfTheLayerThatRefused(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		driver *backend.Fake
		req    policy.Request
		want   refusal.Reason
	}{
		"the tool is not usable": {
			driver: &backend.Fake{
				PreflightErr: refusal.New(refusal.BackendNotReady, "no ddcutil"),
			},
			req:  request(),
			want: refusal.BackendNotReady,
		},
		"the displays cannot be listed": {
			driver: &backend.Fake{
				EnumerateErr: refusal.New(refusal.EnumerationFailed, "no sysfs"),
			},
			req:  request(),
			want: refusal.EnumerationFailed,
		},
		"nothing is attached": {
			driver: fake(),
			req:    request(),
			want:   refusal.NoDisplays,
		},
		"the monitor is not in the catalog": {
			driver: fake(unsupported()),
			req:    request(),
			want:   refusal.UnknownMonitor,
		},
		"the only supported display cannot be written to": {
			driver: fake(unwritable()),
			req:    request(),
			want:   refusal.DisplayNotWritable,
		},
		"two supported displays and no pin": {
			driver: fake(supported(), supported()),
			req:    request(),
			want:   refusal.MultipleCandidates,
		},
		"the serial pin matches nothing": {
			driver: fake(supported()),
			req:    policy.Request{Input: catalog.InputUSBC, Serial: "OTHERSERIAL"},
			want:   refusal.SerialMismatch,
		},
		"the input was never verified on this model": {
			driver: fake(supported()),
			req:    policy.Request{Input: catalog.InputHDMI1},
			want:   refusal.InputNotEnabled,
		},
		"the target cannot be reached": {
			driver: &backend.Fake{
				Displays: []backend.Display{supported()},
				ReadyErr: refusal.New(refusal.TargetNotReady, "/dev/i2c-5"),
			},
			req:  request(),
			want: refusal.TargetNotReady,
		},
		"the command cannot be built": {
			driver: &backend.Fake{
				Displays: []backend.Display{supported()},
				PlanErr:  refusal.New(refusal.InvalidOperation, "unknown mechanism"),
			},
			req:  request(),
			want: refusal.InvalidOperation,
		},
		// The identity changed between enumeration and the write. The backend
		// refuses, and that refusal must reach the caller unchanged.
		"the display is no longer the one that was identified": {
			driver: &backend.Fake{
				Displays:   []backend.Display{supported()},
				ExecuteErr: refusal.New(refusal.IdentityChanged, "the EDID changed"),
			},
			req:  request(),
			want: refusal.IdentityChanged,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outcome, err := app.Switch(t.Context(), testCase.driver, testCase.req, app.Options{})
			if !refusal.Is(err, testCase.want) {
				t.Fatalf("error = %v, want %s", err, testCase.want)
			}

			if outcome.Sent {
				t.Error("a refused switch reported itself sent")
			}

			var failed *app.ExecutionError
			if errors.As(err, &failed) {
				t.Errorf("a refusal was reported as an execution failure: %v", err)
			}
		})
	}
}

// A refusal means nothing was written, so no refused path may have reached the
// tool - except the identity check, which the backend performs inside Execute.
func TestARefusedSwitchNeverWrites(t *testing.T) {
	t.Parallel()

	drivers := []*backend.Fake{
		{PreflightErr: refusal.New(refusal.BackendNotReady, "no ddcutil")},
		fake(),
		fake(unsupported()),
		fake(unwritable()),
		{
			Displays: []backend.Display{supported()},
			ReadyErr: refusal.New(refusal.TargetNotReady, "/dev/i2c-5"),
		},
	}

	for _, driver := range drivers {
		_, err := app.Switch(t.Context(), driver, request(), app.Options{})
		if err == nil {
			t.Fatal("a broken system was not refused")
		}

		if slices.Contains(driver.Calls(), backend.CallExecute) {
			t.Errorf("a refused switch still called Execute: %v", driver.Calls())
		}
	}
}

// The one distinction the CLI's exit codes rest on: a tool that never started
// wrote nothing and stays a refusal, while a tool that ran and failed leaves the
// write status unknown and must say so.
func TestAToolThatRanAndFailedIsNotARefusal(t *testing.T) {
	t.Parallel()

	driver := &backend.Fake{Displays: []backend.Display{supported()}, ExecuteErr: errTool}

	outcome, err := app.Switch(t.Context(), driver, request(), app.Options{})

	var failed *app.ExecutionError
	if !errors.As(err, &failed) {
		t.Fatalf("error = %v, want an ExecutionError", err)
	}

	if outcome.Sent {
		t.Error("a failed execution reported itself sent")
	}

	var declined *refusal.Refusal
	if errors.As(err, &declined) {
		t.Errorf("a failed execution was reported as a refusal: %v", err)
	}

	if !errors.Is(err, errTool) {
		t.Errorf("the backend's error was not preserved: %v", err)
	}

	if !strings.Contains(err.Error(), app.UnknownWriteStatus) {
		t.Errorf("the message does not say the write status is unknown:\n%s", err)
	}

	if strings.Contains(err.Error(), "No DDC write was performed") {
		t.Errorf("the message claims no write was performed:\n%s", err)
	}
}

// A tool that never started is a refusal, because nothing can have been written.
func TestAToolThatNeverStartedStaysARefusal(t *testing.T) {
	t.Parallel()

	driver := &backend.Fake{
		Displays:   []backend.Display{supported()},
		ExecuteErr: refusal.New(refusal.BackendNotReady, "ddcutil did not start"),
	}

	_, err := app.Switch(t.Context(), driver, request(), app.Options{})
	if !refusal.Is(err, refusal.BackendNotReady) {
		t.Fatalf("error = %v, want backend-not-ready", err)
	}

	if !strings.Contains(err.Error(), "No DDC write was performed") {
		t.Errorf("the refusal does not say that nothing was written:\n%s", err)
	}
}

// A serial pin selects one physical unit out of several identical ones.
func TestASerialPinSelectsTheUnitItNames(t *testing.T) {
	t.Parallel()

	other := supported()
	other.Handle = "card1-DP-3"
	other.Label = "card1-DP-3"
	other.Identity.SerialString = "TESTSERIAL02"

	driver := fake(supported(), other)

	outcome, err := app.Switch(
		t.Context(),
		driver,
		policy.Request{Input: catalog.InputUSBC, Serial: "TESTSERIAL02"},
		app.Options{},
	)
	if err != nil {
		t.Fatalf("Switch() refused a pinned display: %v", err)
	}

	if outcome.Display.Handle != other.Handle {
		t.Errorf("switched %q, want %q", outcome.Display.Handle, other.Handle)
	}

	// The whole point of pinning is which of two identical units is written to,
	// so the assertion has to be about what Execute was handed, not about what
	// the outcome says was decided.
	targets := driver.Targets()
	if len(targets) != 1 || targets[0].Handle != other.Handle {
		t.Errorf("wrote to %v, want only the pinned %q", targets, other.Handle)
	}

	if targets[0].Identity.SerialString != "TESTSERIAL02" {
		t.Errorf("wrote to the unit with serial %q", targets[0].Identity.SerialString)
	}
}
