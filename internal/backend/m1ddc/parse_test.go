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

// The parser and every decision taken from it are tested here, in an internal
// test package and with no build tag, so they run on the Linux development host
// as well as on macOS. Only the glue that performs I/O is macOS-only.

package m1ddc

import (
	"errors"
	"os"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
)

const (
	// listFixture is a sanitized capture of `m1ddc display list detailed`.
	listFixture = "testdata/display-list-detailed.txt"
	// emptyFixture is a capture of what m1ddc says with nothing attached.
	emptyFixture = "testdata/no-external-display.txt"

	// testedUUID is the synthetic UUID of the supported display in the capture.
	testedUUID = "00000000-0000-4000-8000-000000000001"
)

// fixture reads a capture.
func fixture(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return string(contents)
}

// listed parses the capture, which almost every test starts from.
func listed(t *testing.T) []record {
	t.Helper()

	records, err := parseDisplayList(fixture(t, listFixture))
	if err != nil {
		t.Fatalf("the captured display list no longer parses: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("parsed %d displays from the capture, want 2", len(records))
	}

	return records
}

func TestTheCapturedListYieldsTheSupportedDisplay(t *testing.T) {
	t.Parallel()

	first := listed(t)[0].display

	want := backend.Display{
		Identity: edid.Identity{
			Manufacturer: "GSM",
			ProductCode:  0x77D4,
			SerialNumber: 0x01020304,
			SerialString: "TESTSERIAL01",
			ModelName:    "LG ULTRAWIDE",
		},
		Handle:        testedUUID,
		HandlePrivate: true,
		Label:         "display 1",
		Writable:      true,
		Status:        backend.StatusOK,
	}

	if first != want {
		t.Errorf("display =\n%+v\nwant\n%+v", first, want)
	}
}

// The catalog is matched on the same three EDID fields on both operating
// systems. If m1ddc's numbers were mapped onto the wrong ones, this is where it
// shows: the tested monitor would stop being recognized.
func TestTheCapturedDisplayMatchesTheCatalog(t *testing.T) {
	t.Parallel()

	model, result := catalog.Match(listed(t)[0].display.Identity)
	if result != catalog.MatchExact {
		t.Fatalf("match = %s, want exact", result)
	}

	if model.FullName() != "LG 38WR85QC-W" {
		t.Errorf("matched %s, want LG 38WR85QC-W", model.FullName())
	}
}

// A display macOS gave no UUID cannot be addressed at all: the list position
// would address it, but it changes when a monitor is plugged in.
func TestADisplayWithoutAUUIDIsReportedAndNotWritable(t *testing.T) {
	t.Parallel()

	second := listed(t)[1]

	if second.uuid != "" || second.display.Handle != "" {
		t.Errorf("a display with no UUID got the handle %q", second.display.Handle)
	}

	if second.display.Writable {
		t.Error("a display with no UUID is writable")
	}

	if second.display.Status != backend.StatusNoUUID {
		t.Errorf("status = %q, want %q", second.display.Status, backend.StatusNoUUID)
	}

	if second.display.Label != "display 2" {
		t.Errorf("label = %q, want %q", second.display.Label, "display 2")
	}
}

// (null) is m1ddc's marker for a field the IORegistry did not supply, and it is
// an absent value rather than a serial reading "(null)".
// The handle becomes m1ddc's selector, and m1ddc accepts a list index in the
// same position. Anything that is not a UUID must therefore leave the display
// unwritable rather than be passed through.
func TestOnlyAUUIDShapedHandleIsAccepted(t *testing.T) {
	t.Parallel()

	rejected := []string{"1", "2", "(null)", "", "not-a-uuid", testedUUID + "extra"}

	for _, candidate := range rejected {
		parsed := block{
			number: 1,
			uuid:   candidate,
			fields: map[string]string{fieldUUID: candidate},
		}

		got := parsed.record()
		if got.uuid != "" || got.display.Handle != "" {
			t.Errorf("%q was accepted as a selector", candidate)
		}

		if got.display.Writable || got.display.Status != backend.StatusNoUUID {
			t.Errorf(
				"%q left the display writable=%t status=%q",
				candidate,
				got.display.Writable,
				got.display.Status,
			)
		}
	}

	accepted := block{
		number: 1,
		uuid:   testedUUID,
		fields: map[string]string{},
	}

	if accepted.record().uuid != testedUUID {
		t.Errorf("a UUID printed only in the header was not accepted")
	}
}

func TestAbsentFieldsStayAbsent(t *testing.T) {
	t.Parallel()

	second := listed(t)[1].display.Identity

	if second.SerialString != "" {
		t.Errorf("serial string = %q, want it empty", second.SerialString)
	}

	// The alphanumeric serial is the only one a --serial pin matches on, so a
	// display without one is a display that cannot be pinned - which policy
	// says plainly, and which it can only say if the field is empty here.
	if second.Manufacturer != "XXX" {
		t.Errorf("manufacturer = %q, want %q", second.Manufacturer, "XXX")
	}
}

// The manufacturer string comes from the IORegistry and is sometimes missing.
// The packed vendor number is the same value EDID carries, so it is what the
// catalog match falls back to - matching on a wrong one would mean matching a
// different vendor's monitor.
func TestManufacturerFallsBackToTheVendorNumber(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		manufacturer string
		vendor       string
		want         string
	}{
		"the reported identifier is used": {"GSM", "7789 (0x1e6d)", "GSM"},
		"absent falls back to the number": {"(null)", "7789 (0x1e6d)", "GSM"},
		"empty falls back to the number":  {"", "7789 (0x1e6d)", "GSM"},
		"a non-PNP string is decoded":     {"LG Electronics", "7789 (0x1e6d)", "GSM"},
		"neither yields nothing":          {"(null)", "0 (0x0000)", ""},
		"an undecodable number yields nothing": {
			"(null)", "65535 (0xffff)", "",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			parsed := block{fields: map[string]string{
				fieldManufacturer: testCase.manufacturer,
				fieldVendor:       testCase.vendor,
			}}

			got := parsed.manufacturer()
			if got != testCase.want {
				t.Errorf("manufacturer = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestOutputWithNoDisplayBlockIsAnError(t *testing.T) {
	t.Parallel()

	for _, output := range []string{"", "usage: m1ddc ...\n", " - Product name:  orphan\n"} {
		_, err := parseDisplayList(output)
		if !errors.Is(err, ErrUnparsable) {
			t.Errorf("parseDisplayList(%q) error = %v, want ErrUnparsable", output, err)
		}
	}
}

// The exact message is pinned in a constant, and the capture is what pins the
// constant to the tool.
func TestTheNoDisplayMessageMatchesTheCapture(t *testing.T) {
	t.Parallel()

	if !noDisplaysReported(fixture(t, emptyFixture)) {
		t.Errorf("m1ddc's no-display answer is no longer recognized by %q", noDisplayMessage)
	}

	if noDisplaysReported(fixture(t, listFixture)) {
		t.Error("a display list was mistaken for the no-display answer")
	}
}
