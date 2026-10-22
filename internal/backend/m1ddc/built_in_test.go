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
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/policy"
	"github.com/leinardi/monmux/internal/refusal"
)

// The two captures of a MacBook with one external monitor, sanitized: an M4 Max
// with the LG, and an earlier Pro/Max-class chip with the Samsung G95NC. The
// IORegistry layout differs between them, which is the point of having both.
const (
	builtInH16G  = "testdata/display-list-built-in-h16g.txt"
	builtInT600x = "testdata/display-list-built-in-t600x.txt"
)

// panelUUID is the synthetic UUID the captures give the built-in panel.
const panelUUID = "00000000-0000-4000-8000-000000000002"

// capture parses one of the built-in captures and returns the panel's record
// and the external display's, whichever order m1ddc listed them in.
func capture(t *testing.T, path string) (panel, external record) {
	t.Helper()

	records, err := parseDisplayList(fixture(t, path))
	if err != nil || len(records) != 2 {
		t.Fatalf("%s parsed to %d records: %v", path, len(records), err)
	}

	if records[0].uuid == panelUUID {
		return records[0], records[1]
	}

	return records[1], records[0]
}

// m1ddc lists the Mac's own panel with a system UUID, so it looked writable.
// It is reported, keeps its handle so a refusal can name its status, and is
// never writable.
func TestTheBuiltInPanelIsReportedAndNotWritable(t *testing.T) {
	t.Parallel()

	for _, path := range []string{builtInH16G, builtInT600x} {
		panel, external := capture(t, path)

		if panel.display.Writable || panel.display.Status != backend.StatusBuiltIn {
			t.Errorf("%s: the panel is writable %t with status %q",
				path, panel.display.Writable, panel.display.Status)
		}

		if panel.display.Handle != panelUUID || panel.display.Identity.Manufacturer != "APP" {
			t.Errorf("%s: the panel is reported as %+v", path, panel.display)
		}

		if !external.display.Writable || external.display.Status != backend.StatusOK {
			t.Errorf("%s: the external display is writable %t with status %q",
				path, external.display.Writable, external.display.Status)
		}
	}
}

// Only a path segment that starts with disp0@ is the panel. dispext0@ is the
// first external display, and is the near miss that matters.
func TestOnlyADisp0SegmentIsTheBuiltInPanel(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"IOService:/AppleARMPE/arm-io/AppleT600xIO/disp0@8A000000/AppleCLCD2":                     true,
		"IOService:/AppleARMPE/arm-io@10F00000/AppleH16GFamilyIO/disp0@0/IOMobileFramebufferShim": true,
		"IOService:/AppleARMPE/arm-io/AppleT600xIO/dispext0@8B000000/AppleCLCD2":                  false,
		"IOService:/AppleARMPE/arm-io/AppleT8103IO/dispext0":                                      false,
		"IOService:/AppleARMPE/arm-io/AppleT600xIO/xdisp0@8A000000/AppleCLCD2":                    false,
		"IOService:/AppleARMPE/arm-io/AppleT600xIO/disp1@8A000000/AppleCLCD2":                     false,
		"":       false,
		"(null)": false,
	}

	for location, want := range cases {
		current := block{fields: map[string]string{fieldLocation: location}}
		if current.builtIn() != want {
			t.Errorf("builtIn(%q) = %t, want %t", location, current.builtIn(), want)
		}
	}
}

// A panel macOS gave no UUID is still a panel, and says so.
func TestABuiltInPanelWithoutAUUIDSaysItIsBuiltIn(t *testing.T) {
	t.Parallel()

	listing := strings.ReplaceAll(fixture(t, builtInT600x), panelUUID, "")

	records, err := parseDisplayList(listing)
	if err != nil {
		t.Fatalf("the listing no longer parses: %v", err)
	}

	if records[0].display.Writable || records[0].display.Status != backend.StatusBuiltIn {
		t.Errorf("the panel is writable %t with status %q",
			records[0].display.Writable, records[0].display.Status)
	}
}

// The reason for the check: with the panel counted, a laptop with one external
// monitor had two writable displays, and --unsafe-model refused with
// multiple-candidates. Without it, the external one is the only target.
func TestUnsafeModelPicksTheExternalDisplayOverTheBuiltInPanel(t *testing.T) {
	t.Parallel()

	for _, path := range []string{builtInH16G, builtInT600x} {
		panel, external := capture(t, path)

		decision, err := policy.Resolve(
			displaysOf([]record{panel, external}),
			policy.Request{Input: catalog.Input("dp"), AssumeModel: "Samsung/G95NC"},
		)
		if err != nil {
			t.Fatalf("%s: Resolve() refused: %v", path, err)
		}

		if decision.Display.Handle != external.uuid || !decision.Assumed {
			t.Errorf(
				"%s: decided on %s, assumed %t",
				path,
				decision.Display.Label,
				decision.Assumed,
			)
		}
	}
}

// m1ddc 1.2.0 does not list the panel on its own - with no external display it
// aborts instead - but if a version ever does, there is nothing to write to,
// and the refusal says which display was seen and why it cannot be written to.
func TestUnsafeModelRefusesTheBuiltInPanelAlone(t *testing.T) {
	t.Parallel()

	panel, _ := capture(t, builtInH16G)

	_, err := policy.Resolve(
		displaysOf([]record{panel}),
		policy.Request{Input: catalog.Input("dp"), AssumeModel: "Samsung/G95NC"},
	)
	if !refusal.Is(err, refusal.DisplayNotWritable) || !strings.Contains(err.Error(), "built-in") {
		t.Errorf("error = %v, want display-not-writable naming the built-in status", err)
	}
}

// Asked to plan for the panel anyway, the backend refuses and names its status,
// rather than handing m1ddc a UUID it found.
func TestThePanelIsNeverAPlanTarget(t *testing.T) {
	t.Parallel()

	panel, external := capture(t, builtInT600x)
	known, statuses := writable([]record{panel, external})

	_, err := lookup(known, statuses, &panel.display)
	if !refusal.Is(err, refusal.TargetNotReady) || !strings.Contains(err.Error(), "built-in") {
		t.Errorf("error = %v, want target-not-ready naming the built-in status", err)
	}
}
