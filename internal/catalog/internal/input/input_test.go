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

package input_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/leinardi/monmux/internal/catalog/internal/input"
)

func TestParseAcceptsAKindAndAnOptionalNumber(t *testing.T) {
	t.Parallel()

	cases := map[string]input.Name{
		"dp":           {Kind: input.KindDP},
		"hdmi":         {Kind: input.KindHDMI},
		"hdmi1":        {Kind: input.KindHDMI, Number: 1},
		"hdmi2":        {Kind: input.KindHDMI, Number: 2},
		"hdmi10":       {Kind: input.KindHDMI, Number: 10},
		"usb-c":        {Kind: input.KindUSBC},
		"usb-c2":       {Kind: input.KindUSBC, Number: 2},
		"dvi":          {Kind: input.KindDVI},
		"vga":          {Kind: input.KindVGA},
		"thunderbolt":  {Kind: input.KindThunderbolt},
		"thunderbolt2": {Kind: input.KindThunderbolt, Number: 2},
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			parsed, err := input.Parse(name)
			if err != nil {
				t.Fatalf("Parse(%q) returned %v", name, err)
			}

			if parsed != want {
				t.Errorf("Parse(%q) = %+v, want %+v", name, parsed, want)
			}

			if parsed.String() != name {
				t.Errorf("Parse(%q).String() = %q", name, parsed.String())
			}
		})
	}
}

func TestParseRejectsAnythingElse(t *testing.T) {
	t.Parallel()

	cases := map[string]error{
		"":                         input.ErrUnknownKind,
		"DP":                       input.ErrUnknownKind,
		"displayport":              input.ErrUnknownKind,
		"usbc":                     input.ErrUnknownKind,
		"scart":                    input.ErrUnknownKind,
		"1":                        input.ErrUnknownKind,
		"0xD0":                     input.ErrUnknownKind,
		"hdmi-1":                   input.ErrUnknownKind,
		"hdmi 1":                   input.ErrUnknownKind,
		"usb-c-2":                  input.ErrUnknownKind,
		"hdmi0":                    input.ErrMalformedNumber,
		"hdmi01":                   input.ErrMalformedNumber,
		"hdmi1x":                   input.ErrMalformedNumber,
		"hdmi99999999999999999999": input.ErrMalformedNumber,
	}

	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			parsed, err := input.Parse(name)
			if err == nil {
				t.Fatalf("Parse(%q) accepted it as %+v", name, parsed)
			}

			if !errors.Is(err, want) {
				t.Errorf("Parse(%q) error = %v, want %v", name, err, want)
			}
		})
	}
}

func TestLabelsDeriveFromTheParts(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"dp":           "DisplayPort",
		"hdmi":         "HDMI",
		"hdmi3":        "HDMI 3",
		"usb-c":        "USB-C",
		"usb-c2":       "USB-C 2",
		"dvi":          "DVI",
		"vga":          "VGA",
		"thunderbolt2": "Thunderbolt 2",
	}

	for name, want := range cases {
		parsed, err := input.Parse(name)
		if err != nil {
			t.Fatalf("Parse(%q) returned %v", name, err)
		}

		if parsed.Label() != want {
			t.Errorf("Parse(%q).Label() = %q, want %q", name, parsed.Label(), want)
		}
	}
}

func TestEveryKindHasItsOwnLabel(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}

	for _, kind := range input.Kinds() {
		label := kind.Label()
		if label == "" || label == kind.String() {
			t.Errorf("%q has no human-readable label", kind)
		}

		if seen[label] {
			t.Errorf("label %q is used twice", label)
		}

		seen[label] = true
	}
}

func TestCompareOrdersByKindThenByPort(t *testing.T) {
	t.Parallel()

	shuffled := []string{
		"thunderbolt", "hdmi10", "usb-c2", "vga", "hdmi2", "usb-c",
		"usb-c1", "dvi", "dp", "hdmi", "hdmi1",
	}

	want := []string{
		"dp", "hdmi", "hdmi1", "hdmi2", "hdmi10", "usb-c", "usb-c1", "usb-c2",
		"dvi", "vga", "thunderbolt",
	}

	names := make([]input.Name, 0, len(shuffled))

	for _, name := range shuffled {
		parsed, err := input.Parse(name)
		if err != nil {
			t.Fatalf("Parse(%q) returned %v", name, err)
		}

		names = append(names, parsed)
	}

	slices.SortFunc(names, input.Compare)

	got := make([]string, 0, len(names))
	for _, name := range names {
		got = append(got, name.String())
	}

	if !slices.Equal(got, want) {
		t.Errorf("sorted = %v, want %v", got, want)
	}
}

func TestCheckNumberingRejectsAKindWrittenBothWays(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		names   []string
		wantErr bool
	}{
		"bare kinds only":            {names: []string{"dp", "usb-c", "hdmi"}},
		"numbered ports only":        {names: []string{"hdmi1", "hdmi2", "usb-c1"}},
		"a mix across kinds":         {names: []string{"dp", "hdmi1", "hdmi2"}},
		"nothing at all":             {names: nil},
		"one kind bare and numbered": {names: []string{"usb-c", "usb-c2"}, wantErr: true},
		"the numbered one first":     {names: []string{"hdmi2", "hdmi"}, wantErr: true},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			names := make([]input.Name, 0, len(testCase.names))

			for _, raw := range testCase.names {
				parsed, err := input.Parse(raw)
				if err != nil {
					t.Fatalf("Parse(%q) returned %v", raw, err)
				}

				names = append(names, parsed)
			}

			err := input.CheckNumbering(names)

			if testCase.wantErr && !errors.Is(err, input.ErrMixedNumbering) {
				t.Errorf("CheckNumbering(%v) = %v, want ErrMixedNumbering", testCase.names, err)
			}

			if !testCase.wantErr && err != nil {
				t.Errorf("CheckNumbering(%v) = %v, want nil", testCase.names, err)
			}
		})
	}
}

func TestKindsIsACopy(t *testing.T) {
	t.Parallel()

	first := input.Kinds()
	first[0] = "invented"

	if input.Kinds()[0] != input.KindDP {
		t.Error("Kinds() hands out the package's own slice")
	}
}
