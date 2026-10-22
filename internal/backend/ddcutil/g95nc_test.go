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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"testing"

	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/policy"
	"github.com/leinardi/monmux/internal/refusal"
)

// The EDID block-0 layout the G95NC stand-in is written into.
const (
	descriptorStart  = 54
	descriptorLength = 18
	checksumOffset   = 127
	modelNameTag     = 0xFC
)

// The Samsung G95NC's product codes, as ema987 captured them.
const (
	g95ncOverDP   = 0x7476
	g95ncOverHDMI = 0x73D0
	g95ncPBP      = 0x747A
)

// g95ncEDID is the tested LG's EDID rewritten to identify as a Samsung G95NC
// reporting productCode: the manufacturer, the product code and the model name
// changed and the checksum recomputed. Nobody has published a G95NC EDID, and
// the serials stay the fixture's synthetic ones.
func g95ncEDID(t *testing.T, productCode uint16) []byte {
	t.Helper()

	raw := edidFixture(t)

	// "SAM" packed as three 5-bit letters.
	raw[8], raw[9] = 0x4C, 0x2D
	binary.LittleEndian.PutUint16(raw[10:12], productCode)

	for offset := descriptorStart; offset < checksumOffset; offset += descriptorLength {
		if raw[offset] == 0 && raw[offset+1] == 0 && raw[offset+3] == modelNameTag {
			copy(raw[offset+5:offset+descriptorLength], "Odyssey G95NC")
		}
	}

	var sum byte
	for _, value := range raw[:checksumOffset] {
		sum += value
	}

	raw[checksumOffset] = -sum

	return raw
}

// g95ncWorld is the fake system with its one display swapped for a G95NC.
func g95ncWorld(t *testing.T, productCode uint16) (built *world, raw []byte) {
	t.Helper()

	raw = g95ncEDID(t, productCode)

	built = newWorld(t)
	write(t, filepath.Join(built.sysfs, connector, "edid"), raw, 0o644)

	return built, raw
}

// The end-to-end golden test for vcp-input-source, which could not exist until
// a model using it was write-enabled: enumerate a G95NC, identify it, decide,
// plan and execute, and pin the exact arguments at the end. It runs for both
// identities the model is matched by, because the input a machine sits on
// decides which one the monitor reports.
func TestGoldenCommandForTheG95NC(t *testing.T) {
	t.Parallel()

	values := map[catalog.Input]string{
		"dp":    "0x0F",
		"hdmi1": "0x11",
		"hdmi2": "0x12",
		"hdmi3": "0x32",
	}

	for _, productCode := range []uint16{g95ncOverDP, g95ncOverHDMI} {
		for input, value := range values {
			t.Run(fmt.Sprintf("0x%04X/%s", productCode, input), func(t *testing.T) {
				t.Parallel()

				world, raw := g95ncWorld(t, productCode)
				displays := world.ready(t)

				decision, err := policy.Resolve(displays, policy.Request{Input: input})
				if err != nil {
					t.Fatalf("Resolve() refused: %v", err)
				}

				if decision.Model.FullName() != "Samsung G95NC" || decision.Assumed {
					t.Fatalf("decided %s, assumed %t", decision.Model.FullName(), decision.Assumed)
				}

				command, err := world.backend.Plan(decision.Display, decision.Operation)
				if err != nil {
					t.Fatalf("Plan() refused: %v", err)
				}

				want := []string{
					"--edid", hex.EncodeToString(raw), "setvcp", "0x60", value, "--noverify",
				}
				if command.Path != world.binary || !slices.Equal(command.Args, want) {
					t.Errorf(
						"planned %s %q, want %s %q",
						command.Path,
						command.Args,
						world.binary,
						want,
					)
				}

				if len(command.Redact) != 1 || command.Redact[0] != 1 {
					t.Errorf(
						"redact = %v, want [1]: the EDID hex identifies the unit",
						command.Redact,
					)
				}

				err = world.backend.Execute(t.Context(), decision.Display, decision.Operation)
				if err != nil {
					t.Fatalf("Execute() failed: %v", err)
				}

				calls := world.runner.Calls()
				if last := calls[len(calls)-1]; last.Line() != command.Render(true) {
					t.Errorf("ran\n%s\nbut planned\n%s", last.Line(), command.Render(true))
				}
			})
		}
	}
}

// With Picture-by-Picture on, the G95NC reports a product code the catalog
// refuses. The display is identified, refused for its mode, and nothing is run.
func TestTheG95NCInPictureByPictureIsRefused(t *testing.T) {
	t.Parallel()

	world, _ := g95ncWorld(t, g95ncPBP)
	displays := world.ready(t)
	before := len(world.runner.Calls())

	_, err := policy.Resolve(displays, policy.Request{Input: "hdmi1"})
	if !refusal.Is(err, refusal.UnsupportedMode) {
		t.Fatalf("error = %v, want unsupported-mode", err)
	}

	if len(world.runner.Calls()) != before {
		t.Errorf("the refusal ran something: %v", world.runner.Lines()[before:])
	}
}
