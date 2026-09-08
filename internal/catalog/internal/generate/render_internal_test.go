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

package generate

import (
	"errors"
	"testing"

	"github.com/leinardi/monmux/internal/catalog/internal/input"
)

// The evidence sentence is the one thing a reader of the compatibility page
// judges a row by, so every grade has to read the same way whoever wrote the
// entry. These are the four sentences, pinned.
func TestRenderEvidenceComposesOneSentencePerGrade(t *testing.T) {
	t.Parallel()

	value := uint8(0xD0)

	cases := map[string]struct {
		evidence Evidence
		name     string
		want     string
	}{
		"verified": {
			evidence: Evidence{
				Grade: gradeVerified,
				Date:  "2026-09-07",
				Tool:  "Linux (ddcutil) and macOS (m1ddc)",
				Note:  "switched from USB-C to DisplayPort",
			},
			name: "dp",
			want: "Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc): " +
				"switched from USB-C to DisplayPort",
		},
		"documented": {
			evidence: Evidence{
				Grade: gradeDocumented,
				By:    "LG",
				URL:   "https://example.com/manual.pdf",
			},
			name: "hdmi1",
			want: "Documented by LG for the exact TEST-1: HDMI 1 is 0xD0 over the LG side channel " +
				"(source address 0x50, VCP 0xF4); no field report; not verified here: " +
				"https://example.com/manual.pdf",
		},
		"reported": {
			evidence: Evidence{
				Grade: gradeReported,
				By:    "a tester",
				Tool:  "ddcutil",
				URL:   "https://example.com/report",
			},
			name: "usb-c",
			want: "Reported working on the exact TEST-1 by a tester (ddcutil): switching to USB-C " +
				"with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; " +
				"not verified here: https://example.com/report",
		},
		"quoted": {
			evidence: Evidence{
				Grade: gradeQuoted,
				By:    "a tester",
				Tool:  "ddcutil",
				URL:   "https://example.com/report",
			},
			name: "thunderbolt",
			want: "Weaker report on the exact TEST-1 by a tester (ddcutil): the report quotes " +
				"0xD0 for Thunderbolt and says it works, without saying which inputs were tried " +
				"individually; not verified here: https://example.com/report",
		},
		"a note is folded in before the reference": {
			evidence: Evidence{
				Grade: gradeReported,
				By:    "a tester",
				Tool:  "ddcutil",
				URL:   "https://example.com/report",
				Note:  "found by looping over candidate values",
			},
			name: "dp",
			want: "Reported working on the exact TEST-1 by a tester (ddcutil): switching to " +
				"DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) " +
				"succeeded; found by looping over candidate values; not verified here: " +
				"https://example.com/report",
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			parsed, err := input.Parse(testCase.name)
			if err != nil {
				t.Fatalf("parsing %q: %v", testCase.name, err)
			}

			entry := Input{
				Mechanism: "lg-alt-input",
				Value:     &value,
				Evidence:  testCase.evidence,
			}

			got, err := renderEvidence("TEST-1", parsed, &entry)
			if err != nil {
				t.Fatalf("renderEvidence: %v", err)
			}

			if got != testCase.want {
				t.Errorf("renderEvidence =\n%s\nwant\n%s", got, testCase.want)
			}
		})
	}
}

// Validation runs before rendering, so these two are bugs in this package rather
// than in the catalog file. They still must not compose half a sentence.
func TestRenderEvidenceRefusesWhatItCannotName(t *testing.T) {
	t.Parallel()

	value := uint8(0xD0)

	parsed, err := input.Parse("dp")
	if err != nil {
		t.Fatalf("parsing dp: %v", err)
	}

	for name, entry := range map[string]Input{
		"an unknown mechanism has no phrase": {
			Mechanism: "vcp-60",
			Value:     &value,
			Evidence:  Evidence{Grade: gradeReported, By: "a tester", URL: "https://example.com"},
		},
		"an unknown grade has no sentence": {
			Mechanism: "lg-alt-input",
			Value:     &value,
			Evidence:  Evidence{Grade: "hearsay", By: "a tester", URL: "https://example.com"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			rendered, renderErr := renderEvidence("TEST-1", parsed, &entry)
			if renderErr == nil {
				t.Fatalf("%s was rendered as %q", name, rendered)
			}

			if !errors.Is(renderErr, ErrRender) {
				t.Errorf("%s gave %v, want ErrRender", name, renderErr)
			}
		})
	}
}
