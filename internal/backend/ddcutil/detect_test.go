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

// The `ddcutil detect --verbose` parser and the match on it are tested here, in
// an internal test package and with no build tag, so they run on any host.

package ddcutil

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"
)

// The captures, sanitized: the synthetic LG EDID in place of the real one, on
// an AMD card under ddcutil 2.2.5 and on an NVIDIA card under 3.0.2, where the
// kernel gives the connector no ddc link.
const (
	detectAMD    = "testdata/detect-verbose-2.2.5-amdgpu.txt"
	detectNVIDIA = "testdata/detect-verbose-3.0.2-nvidia.txt"
	// syntheticEDID is the EDID both captures carry.
	syntheticEDID = "../testdata/lg-38wr85qc-w.edid"
)

// readFixture reads one capture.
func readFixture(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	return string(contents)
}

// blockZero is the synthetic EDID's block 0.
func blockZero(t *testing.T) []byte {
	t.Helper()

	return []byte(readFixture(t, syntheticEDID))[:edidBlockSize]
}

// The fallback runs one read-only command, and only that one.
func TestDetectRunsOnlyTheReadOnlyListing(t *testing.T) {
	t.Parallel()

	if !slices.Equal(detectArgs, []string{"detect", "--verbose"}) {
		t.Errorf("detectArgs = %q", detectArgs)
	}
}

// Both versions' output yields the bus, the connector and the whole of block 0,
// whatever the indent.
func TestParseDetectReadsBothCaptures(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ bus, connector string }{
		detectAMD:    {bus: "i2c-10", connector: "card1-DP-1"},
		detectNVIDIA: {bus: "i2c-9", connector: "card2-HDMI-A-2"},
	}

	for path, want := range cases {
		blocks := parseDetect(readFixture(t, path))
		if len(blocks) != 1 {
			t.Fatalf("%s: parsed %d blocks, want 1", path, len(blocks))
		}

		got := blocks[0]
		if !got.valid || got.bus != want.bus || got.connector != want.connector {
			t.Errorf("%s: parsed %+v", path, got)
		}

		if !bytes.Equal(got.edid, blockZero(t)) {
			t.Errorf("%s: the EDID is not block 0 of the fixture: %x", path, got.edid)
		}
	}
}

// A dump that is short, out of order or not hex is no EDID at all: half an EDID
// compared with a whole one would prove nothing.
func TestParseDetectDropsAnUntrustworthyDump(t *testing.T) {
	t.Parallel()

	capture := readFixture(t, detectNVIDIA)
	rows := strings.Split(capture, "\n")

	var row0010 string

	for _, line := range rows {
		if strings.HasPrefix(line, "+0010") {
			row0010 = line
		}
	}

	cases := map[string]string{
		"a row missing":   strings.Replace(capture, row0010+"\n", "", 1),
		"rows swapped":    strings.Replace(capture, "+0000", "+0010", 1),
		"a row cut short": strings.Replace(capture, row0010, row0010[:20], 1),
	}

	for name, output := range cases {
		blocks := parseDetect(output)
		if len(blocks) != 1 || blocks[0].edid != nil {
			t.Errorf("%s: kept the EDID %x", name, blocks[0].edid)
		}
	}
}

// A bus ddcutil cannot talk DDC on is parsed, so it still counts when the same
// EDID appears twice, but is never a bus to write on.
func TestParseDetectMarksAnInvalidDisplay(t *testing.T) {
	t.Parallel()

	output := strings.Replace(readFixture(t, detectNVIDIA), "Display 1", "Invalid display", 1)

	blocks := parseDetect(output)
	if len(blocks) != 1 || blocks[0].valid || blocks[0].edid == nil {
		t.Errorf("parsed %+v", blocks)
	}
}

func TestBusByEDIDAcceptsOnlyOneUnambiguousMatch(t *testing.T) {
	t.Parallel()

	edid := blockZero(t)
	capture := readFixture(t, detectNVIDIA)

	other := bytes.Clone(edid)
	other[12] ^= 0xFF

	cases := map[string]struct {
		output, connector string
		want              string
	}{
		"the matching bus": {
			output: capture, connector: "card2-HDMI-A-2", want: "i2c-9",
		},
		"ddcutil names another connector": {
			output: capture, connector: "card2-HDMI-A-1",
		},
		"the EDID on two buses": {
			output:    capture + strings.Replace(capture, "/dev/i2c-9", "/dev/i2c-8", 1),
			connector: "card2-HDMI-A-2",
		},
		"the EDID only on a bus ddcutil cannot talk on": {
			output:    strings.Replace(capture, "Display 1", "Invalid display", 1),
			connector: "card2-HDMI-A-2",
		},
		"nothing at all": {
			output: "No displays found.\n", connector: "card2-HDMI-A-2",
		},
	}

	for name, test := range cases {
		bus, ok := busByEDID(parseDetect(test.output), edid, test.connector)
		if bus != test.want || ok != (test.want != "") {
			t.Errorf("%s: busByEDID = %q, %t, want %q", name, bus, ok, test.want)
		}
	}

	_, ok := busByEDID(parseDetect(capture), other, "card2-HDMI-A-2")
	if ok {
		t.Error("a different EDID was placed")
	}
}
