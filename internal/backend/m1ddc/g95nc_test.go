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
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/policy"
	"github.com/leinardi/monmux/internal/refusal"
)

// g95ncListing is a `display list detailed` entry shaped like ema987's capture
// of the Samsung G95NC, with the synthetic UUID and serial in place of the real
// ones, reporting the given product code in decimal and in hex as m1ddc does.
func g95ncListing(productCode uint16) string {
	return strings.Join([]string{
		"[1] Odyssey G95NC (" + testedUUID + ")",
		" - Product name:  Odyssey G95NC",
		" - Manufacturer:  SAM",
		" - AN Serial:     TESTSERIAL01",
		" - Vendor:        19501 (0x4c2d)",
		fmt.Sprintf(" - Model:         %d (0x%04x)", productCode, productCode),
		" - Serial:        16909060 (0x01020304)",
		" - Display ID:    4",
		" - System UUID:   " + testedUUID,
		" - EDID UUID:     " + testedUUID,
		" - IO Location:   IOService:/AppleARMPE/arm-io/AppleT600xIO/dispext1@8B000000/AppleCLCD2",
		" - Adapter:       14595",
		"",
	}, "\n")
}

// g95ncRecords parses that listing.
func g95ncRecords(t *testing.T, productCode uint16) []record {
	t.Helper()

	records, err := parseDisplayList(g95ncListing(productCode))
	if err != nil || len(records) != 1 {
		t.Fatalf("the G95NC listing parsed to %d records: %v", len(records), err)
	}

	return records
}

// displaysOf is what Enumerate would report for the records.
func displaysOf(records []record) []backend.Display {
	displays := make([]backend.Display, 0, len(records))
	for index := range records {
		displays = append(displays, records[index].display)
	}

	return displays
}

// The end-to-end golden test for vcp-input-source through m1ddc, the path the
// G95NC was verified on: parse the listing, identify, decide and plan, and pin
// the exact arguments. m1ddc takes the value in decimal: 15, 17, 18 and 50 are
// 0x0F, 0x11, 0x12 and 0x32.
func TestGoldenCommandForTheG95NC(t *testing.T) {
	t.Parallel()

	values := map[catalog.Input]string{"dp": "15", "hdmi1": "17", "hdmi2": "18", "hdmi3": "50"}

	for _, productCode := range []uint16{0x7476, 0x73D0} {
		for input, value := range values {
			t.Run(fmt.Sprintf("0x%04X/%s", productCode, input), func(t *testing.T) {
				t.Parallel()

				records := g95ncRecords(t, productCode)

				decision, err := policy.Resolve(displaysOf(records), policy.Request{Input: input})
				if err != nil {
					t.Fatalf("Resolve() refused: %v", err)
				}

				if decision.Model.FullName() != "Samsung G95NC" || decision.Assumed {
					t.Fatalf("decided %s, assumed %t", decision.Model.FullName(), decision.Assumed)
				}

				command, err := planFor(binary, &records[0], decision.Operation)
				if err != nil {
					t.Fatalf("planFor() refused: %v", err)
				}

				want := []string{"display", testedUUID, "set", "input", value}
				if command.Path != binary || !slices.Equal(command.Args, want) {
					t.Errorf("planned %s %q, want %s %q", command.Path, command.Args, binary, want)
				}

				if strings.Contains(command.String(), testedUUID) {
					t.Errorf("the default rendering leaked the UUID:\n%s", command)
				}
			})
		}
	}
}

// With Picture-by-Picture on, the G95NC reports a product code the catalog
// refuses, and the decision stops there.
func TestTheG95NCInPictureByPictureIsRefused(t *testing.T) {
	t.Parallel()

	_, err := policy.Resolve(
		displaysOf(g95ncRecords(t, 0x747A)),
		policy.Request{Input: "dp"},
	)
	if !refusal.Is(err, refusal.UnsupportedMode) {
		t.Errorf("error = %v, want unsupported-mode", err)
	}
}
