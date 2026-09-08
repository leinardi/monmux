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

package policy_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/policy"
	"github.com/leinardi/monmux/internal/refusal"
)

const (
	serialA = "TESTSERIAL01"
	serialB = "TESTSERIAL02"
)

// supported is the tested LG 38WR85QC-W as it appears over DisplayPort.
func supported() backend.Display {
	return backend.Display{
		Identity: edid.Identity{
			Manufacturer: "GSM",
			ProductCode:  0x77D3,
			SerialNumber: 0x01020304,
			SerialString: serialA,
			ModelName:    "LG ULTRAWIDE",
		},
		Handle:   "card1-DP-1",
		Label:    "card1-DP-1",
		Writable: true,
		Status:   backend.StatusOK,
	}
}

// unknown is a monitor no catalog entry claims.
func unknown() backend.Display {
	return backend.Display{
		Identity: edid.Identity{Manufacturer: "DEL", ProductCode: 0x1234, SerialString: serialB},
		Handle:   "card1-DP-2",
		Label:    "card1-DP-2",
		Writable: true,
		Status:   backend.StatusOK,
	}
}

func TestResolveSwitchesASupportedDisplay(t *testing.T) {
	t.Parallel()

	decision, err := policy.Resolve(
		[]backend.Display{supported()},
		policy.Request{Input: catalog.Input("usb-c")},
	)
	if err != nil {
		t.Fatalf("Resolve() refused a supported display: %v", err)
	}

	if !decision.Operation.Valid() {
		t.Error("the decision carries an invalid operation")
	}

	if decision.Operation.Value() != 0xD1 {
		t.Errorf("value = 0x%02X, want 0xD1", decision.Operation.Value())
	}

	if decision.Operation.Mechanism() != catalog.MechanismLGAltInput {
		t.Errorf(
			"mechanism = %q, want %q",
			decision.Operation.Mechanism(),
			catalog.MechanismLGAltInput,
		)
	}

	if decision.Display.Handle != "card1-DP-1" {
		t.Errorf("handle = %q, want card1-DP-1", decision.Display.Handle)
	}

	if decision.Model.Name != "38WR85QC-W" {
		t.Errorf("model = %q, want 38WR85QC-W", decision.Model.Name)
	}
}

func TestResolveIgnoresUnrelatedDisplays(t *testing.T) {
	t.Parallel()

	displays := []backend.Display{unknown(), supported()}

	decision, err := policy.Resolve(displays, policy.Request{Input: catalog.Input("dp")})
	if err != nil {
		t.Fatalf("Resolve() refused: %v", err)
	}

	if decision.Display.Handle != "card1-DP-1" {
		t.Errorf("picked %q, want the supported display", decision.Display.Handle)
	}

	if decision.Operation.Value() != 0xD0 {
		t.Errorf("value = 0x%02X, want 0xD0", decision.Operation.Value())
	}
}

func TestResolveRefusals(t *testing.T) {
	t.Parallel()

	unwritable := supported()
	unwritable.Writable = false
	unwritable.Status = backend.StatusNoDDCChannel

	second := supported()
	second.Handle = "card1-DP-3"
	second.Label = "card1-DP-3"
	second.Identity.ProductCode = 0x77D4
	second.Identity.SerialString = serialB

	noSerial := supported()
	noSerial.Identity.SerialString = ""

	tests := map[string]struct {
		displays []backend.Display
		request  policy.Request
		want     refusal.Reason
		detail   string
	}{
		"no displays at all": {
			displays: nil,
			request:  policy.Request{Input: catalog.Input("dp")},
			want:     refusal.NoDisplays,
		},
		"unknown monitor": {
			displays: []backend.Display{unknown()},
			request:  policy.Request{Input: catalog.Input("dp")},
			want:     refusal.UnknownMonitor,
		},
		"supported but not writable": {
			displays: []backend.Display{unwritable},
			request:  policy.Request{Input: catalog.Input("dp")},
			want:     refusal.DisplayNotWritable,
			detail:   backend.StatusNoDDCChannel,
		},
		"two supported displays": {
			displays: []backend.Display{supported(), second},
			request:  policy.Request{Input: catalog.Input("dp")},
			want:     refusal.MultipleCandidates,
		},
		"input without evidence": {
			displays: []backend.Display{supported()},
			request:  policy.Request{Input: catalog.Input("hdmi1")},
			want:     refusal.InputNotEnabled,
			detail:   "hdmi1",
		},
		"input outside the enum": {
			displays: []backend.Display{supported()},
			request:  policy.Request{Input: catalog.Input("vga")},
			want:     refusal.InputNotEnabled,
		},
		"serial not attached": {
			displays: []backend.Display{supported()},
			request:  policy.Request{Input: catalog.Input("dp"), Serial: "NOTHERE"},
			want:     refusal.SerialMismatch,
		},
		"pinning a display with no serial": {
			displays: []backend.Display{noSerial},
			request:  policy.Request{Input: catalog.Input("dp"), Serial: serialA},
			want:     refusal.SerialMismatch,
			detail:   policy.NoSerialDetail,
		},
		"one display cannot be pinned, another can": {
			// The supported monitor exposes no alphanumeric serial while an
			// unrelated one does. Saying only "not attached" would send the
			// user chasing a typo instead of telling them that pinning is
			// unavailable for that unit.
			displays: []backend.Display{noSerial, unknown()},
			request:  policy.Request{Input: catalog.Input("dp"), Serial: serialA},
			want:     refusal.SerialMismatch,
			detail:   noSerial.Label + ": " + policy.NoSerialDetail,
		},
		"every display exposes a serial, none is the pinned one": {
			displays: []backend.Display{supported(), unknown()},
			request:  policy.Request{Input: catalog.Input("dp"), Serial: "NOTHERE"},
			want:     refusal.SerialMismatch,
			detail:   "No attached display carries the requested serial.",
		},
		"a blank pin is an unusable pin, not an absent one": {
			displays: []backend.Display{supported()},
			request:  policy.Request{Input: catalog.Input("dp"), Serial: "   "},
			want:     refusal.SerialMismatch,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := policy.Resolve(test.displays, test.request)
			if err == nil {
				t.Fatal("Resolve() allowed a write it should have refused")
			}

			if !refusal.Is(err, test.want) {
				t.Errorf("reason = %v, want %s", err, test.want)
			}

			if test.detail != "" && !strings.Contains(err.Error(), test.detail) {
				t.Errorf("message does not mention %q:\n%s", test.detail, err)
			}

			if !strings.HasSuffix(err.Error(), "No DDC write was performed.") {
				t.Errorf("refusal does not state that nothing was written:\n%s", err)
			}
		})
	}
}

func TestSerialPinSelectsTheRequestedUnit(t *testing.T) {
	t.Parallel()

	second := supported()
	second.Handle = "card1-DP-3"
	second.Label = "card1-DP-3"
	second.Identity.ProductCode = 0x77D4
	second.Identity.SerialString = serialB

	decision, err := policy.Resolve([]backend.Display{supported(), second}, policy.Request{
		Input:  catalog.Input("dp"),
		Serial: serialB,
	})
	if err != nil {
		t.Fatalf("Resolve() refused a pinned display: %v", err)
	}

	if decision.Display.Handle != "card1-DP-3" {
		t.Errorf("pinned the wrong unit: %q", decision.Display.Handle)
	}
}

func TestSerialPinTrimsSurroundingWhitespaceOnBothSides(t *testing.T) {
	t.Parallel()

	padded := supported()
	padded.Identity.SerialString = "  " + serialA + "\n"

	_, err := policy.Resolve([]backend.Display{padded}, policy.Request{
		Input:  catalog.Input("dp"),
		Serial: "\t" + serialA + " ",
	})
	if err != nil {
		t.Errorf("Resolve() refused a serial that differs only by padding: %v", err)
	}
}

func TestSerialPinIsCaseSensitive(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve([]backend.Display{supported()}, policy.Request{
		Input:  catalog.Input("dp"),
		Serial: strings.ToLower(serialA),
	})
	if !refusal.Is(err, refusal.SerialMismatch) {
		t.Errorf("a lowercased serial matched: %v", err)
	}
}

func TestSerialPinNeverUsesTheNumericSerial(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve([]backend.Display{supported()}, policy.Request{
		Input:  catalog.Input("dp"),
		Serial: "16909060", // 0x01020304 in decimal
	})
	if !refusal.Is(err, refusal.SerialMismatch) {
		t.Errorf("the numeric serial was accepted as a pin: %v", err)
	}
}

func TestUnwritableDisplaysAreNeverSelected(t *testing.T) {
	t.Parallel()

	unwritable := supported()
	unwritable.Writable = false
	unwritable.Status = backend.StatusEDIDUnreadable
	unwritable.Handle = "card1-DP-9"
	unwritable.Label = "card1-DP-9"
	unwritable.Identity.ProductCode = 0x77D4
	unwritable.Identity.SerialString = serialB

	decision, err := policy.Resolve([]backend.Display{unwritable, supported()}, policy.Request{
		Input: catalog.Input("dp"),
	})
	if err != nil {
		t.Fatalf("Resolve() refused although a writable supported display was present: %v", err)
	}

	if decision.Display.Handle != "card1-DP-1" {
		t.Errorf("selected %q, which is not writable", decision.Display.Handle)
	}
}

func TestRefusalsRedactSerialsByDefault(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve(
		[]backend.Display{supported()},
		policy.Request{Input: catalog.Input("hdmi1")},
	)
	if err == nil {
		t.Fatal("Resolve() allowed an input with no evidence")
	}

	if strings.Contains(err.Error(), serialA) {
		t.Errorf("the refusal leaked a serial:\n%s", err)
	}
}

// --- the --unsafe-model assume path ---

func TestAssumeWritesToAnUnidentifiedDisplay(t *testing.T) {
	t.Parallel()

	decision, err := policy.Resolve(
		[]backend.Display{unknown()},
		policy.Request{Input: catalog.Input("hdmi"), AssumeModel: "AOC/Q27P1B"},
	)
	if err != nil {
		t.Fatalf("Resolve() refused an assumed model: %v", err)
	}

	if !decision.Assumed {
		t.Error("the decision does not say identification was bypassed")
	}

	if !decision.Operation.Valid() || decision.Operation.Value() != 0x11 {
		t.Errorf("operation = %s, want vcp-input-source 0x11", decision.Operation)
	}

	if decision.Operation.Mechanism() != catalog.MechanismInputSource {
		t.Errorf("mechanism = %q", decision.Operation.Mechanism())
	}

	if decision.Model.FullName() != "AOC Q27P1B" {
		t.Errorf("model = %q, want AOC Q27P1B", decision.Model.FullName())
	}

	if decision.Display.Handle != "card1-DP-2" {
		t.Errorf("picked %q, want the unidentified display", decision.Display.Handle)
	}
}

// The normal path says nothing about identification, and must keep saying
// nothing: Assumed is set on the assume path only.
func TestAnIdentifiedSwitchIsNotMarkedAssumed(t *testing.T) {
	t.Parallel()

	decision, err := policy.Resolve(
		[]backend.Display{supported()},
		policy.Request{Input: catalog.Input("dp")},
	)
	if err != nil {
		t.Fatalf("Resolve() refused: %v", err)
	}

	if decision.Assumed {
		t.Error("an identified switch claims identification was bypassed")
	}
}

// An override that silently picked the first display would write an unverified
// value to whichever monitor happened to be listed first.
func TestAssumeRefusesMoreThanOneWritableDisplay(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve(
		[]backend.Display{unknown(), supported()},
		policy.Request{Input: catalog.Input("hdmi"), AssumeModel: "AOC/Q27P1B"},
	)
	if !refusal.Is(err, refusal.MultipleCandidates) {
		t.Fatalf("Resolve() returned %v, want multiple-candidates", err)
	}

	var declined *refusal.Refusal
	if !errors.As(err, &declined) {
		t.Fatalf("Resolve() returned %v, want a refusal", err)
	}

	for _, wanted := range []string{"card1-DP-1", "card1-DP-2", "--serial"} {
		if !strings.Contains(declined.Detail, wanted) {
			t.Errorf("the refusal does not mention %q: %q", wanted, declined.Detail)
		}
	}
}

func TestAssumeWritesToTheDisplayThePinSelected(t *testing.T) {
	t.Parallel()

	decision, err := policy.Resolve(
		[]backend.Display{unknown(), supported()},
		policy.Request{
			Input:       catalog.Input("hdmi"),
			Serial:      serialB,
			AssumeModel: "AOC/Q27P1B",
		},
	)
	if err != nil {
		t.Fatalf("Resolve() refused a pinned assumed model: %v", err)
	}

	if decision.Display.Handle != "card1-DP-2" {
		t.Errorf("picked %q, want the pinned display", decision.Display.Handle)
	}

	if decision.Operation.Value() != 0x11 {
		t.Errorf("value = 0x%02X, want 0x11", decision.Operation.Value())
	}
}

// The override skips the write-enabled gate, not the catalog: an input the
// assumed entry never recorded has no value, and none is invented for it.
func TestAssumeRefusesAnInputTheModelDoesNotRecord(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve(
		[]backend.Display{unknown()},
		policy.Request{Input: catalog.Input("usb-c"), AssumeModel: "AOC/Q27P1B"},
	)
	if !refusal.Is(err, refusal.InputNotEnabled) {
		t.Fatalf("Resolve() returned %v, want input-not-enabled", err)
	}

	var declined *refusal.Refusal
	if !errors.As(err, &declined) {
		t.Fatalf("Resolve() returned %v, want a refusal", err)
	}

	// The message lists what the entry records, not what it enables, which for a
	// model that is not write-enabled would be nothing at all.
	for _, wanted := range []string{"recorded inputs:", "dp", "hdmi", "dvi", "vga"} {
		if !strings.Contains(declined.Detail, wanted) {
			t.Errorf("the refusal does not mention %q: %q", wanted, declined.Detail)
		}
	}
}

// The override reaches a display monmux would not have identified; it does not
// reach one it cannot write to at all.
func TestAssumeStillRefusesAnUnwritableDisplay(t *testing.T) {
	t.Parallel()

	unwritable := supported()
	unwritable.Writable = false
	unwritable.Status = backend.StatusNoDDCChannel

	_, err := policy.Resolve(
		[]backend.Display{unwritable},
		policy.Request{Input: catalog.Input("hdmi"), AssumeModel: "AOC/Q27P1B"},
	)
	if !refusal.Is(err, refusal.DisplayNotWritable) {
		t.Fatalf("Resolve() returned %v, want display-not-writable", err)
	}

	// Identification is what the override bypassed, so an unmatched display is
	// refused for the reason that is actually true of it: monmux cannot write to
	// it. Telling the user no catalog entry matches would send them back to the
	// flag they already used.
	unidentified := unknown()
	unidentified.Writable = false
	unidentified.Status = backend.StatusNoDDCChannel

	_, err = policy.Resolve(
		[]backend.Display{unidentified},
		policy.Request{Input: catalog.Input("hdmi"), AssumeModel: "AOC/Q27P1B"},
	)
	if !refusal.Is(err, refusal.DisplayNotWritable) {
		t.Fatalf("Resolve() returned %v, want display-not-writable", err)
	}

	var declined *refusal.Refusal
	if !errors.As(err, &declined) {
		t.Fatalf("Resolve() returned %v, want a refusal", err)
	}

	for _, wanted := range []string{"card1-DP-2", backend.StatusNoDDCChannel} {
		if !strings.Contains(declined.Detail, wanted) {
			t.Errorf("the refusal does not mention %q: %q", wanted, declined.Detail)
		}
	}
}

func TestAssumeRefusesWithNoDisplaysAtAll(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve(
		nil,
		policy.Request{Input: catalog.Input("hdmi"), AssumeModel: "AOC/Q27P1B"},
	)
	if !refusal.Is(err, refusal.NoDisplays) {
		t.Fatalf("Resolve() returned %v, want no-displays", err)
	}
}

// A name that is not a catalog entry is an argument error rather than a refusal,
// exactly as an unparseable input name is: the request could not be made. The
// CLI rejects it before starting anything; this is the belt behind that.
func TestAssumeRejectsAModelThatIsNotInTheCatalog(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve(
		[]backend.Display{unknown()},
		policy.Request{Input: catalog.Input("hdmi"), AssumeModel: "Acme/Nothing"},
	)
	if !errors.Is(err, policy.ErrUnknownModel) {
		t.Fatalf("Resolve() returned %v, want an unknown-model error", err)
	}

	if _, ok := errors.AsType[*refusal.Refusal](err); ok {
		t.Error("a bad argument was reported as a refusal")
	}

	if !strings.Contains(err.Error(), "monmux catalog list") {
		t.Errorf("the error does not point at the listing: %v", err)
	}
}
