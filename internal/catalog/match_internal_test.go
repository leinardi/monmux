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

package catalog

import (
	"slices"
	"testing"

	"github.com/leinardi/monmux/internal/edid"
)

// The real catalog cannot contain a duplicated identity — a test in
// catalog_test.go enforces that — so the ambiguous branch is exercised here,
// against a deliberately colliding set of entries.
func TestMatchInRefusesAnAmbiguousCatalog(t *testing.T) {
	t.Parallel()

	colliding := []Model{
		{
			Name:         "first",
			Vendor:       "LG",
			Identities:   []Identity{{Manufacturer: "GSM", ProductCode: 0x77D3}},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				"dp": {mechanism: MechanismLGAltInput, value: 0xD0, evidence: "test"},
			},
		},
		{
			Name:         "second",
			Vendor:       "LG",
			Identities:   []Identity{{Manufacturer: "GSM", ProductCode: 0x77D3}},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				"dp": {mechanism: MechanismLGAltInput, value: 0xD1, evidence: "test"},
			},
		},
	}

	model, result := matchIn(edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D3}, colliding)

	if result != MatchAmbiguous {
		t.Errorf("match = %s, want ambiguous", result)
	}

	if model.Name != "" || model.WriteEnabled || len(model.Inputs) != 0 {
		t.Errorf("an ambiguous match returned a usable model: %+v", model)
	}
}

func TestMatchInStillResolvesAUniqueIdentity(t *testing.T) {
	t.Parallel()

	entries := []Model{
		{
			Name:         "first",
			Vendor:       "LG",
			Identities:   []Identity{{Manufacturer: "GSM", ProductCode: 0x77D3}},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				"dp": {mechanism: MechanismLGAltInput, value: 0xD0, evidence: "test"},
			},
		},
		{
			Name:       "second",
			Vendor:     "LG",
			Identities: []Identity{{Manufacturer: "GSM", ProductCode: 0x77D4}},
		},
	}

	model, result := matchIn(edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D3}, entries)

	if result != MatchExact || model.Name != "first" {
		t.Errorf("match = %s for %s, want exact for first", result, model.Name)
	}
}

// An entry naming a mechanism no backend implements must not produce something
// a backend could be asked to run.
func TestUnknownMechanismYieldsNoOperation(t *testing.T) {
	t.Parallel()

	model := Model{
		Name:         "invented",
		Vendor:       "ACME",
		Identities:   []Identity{{Manufacturer: "ACM", ProductCode: 0x0001}},
		WriteEnabled: true,
		Inputs: map[Input]inputOp{
			"dp": {mechanism: Mechanism("not-implemented"), value: 0xD0, evidence: "test"},
		},
	}

	op, ok := model.Operation("dp")
	if ok {
		t.Errorf("an unimplemented mechanism yielded an operation: %s", op)
	}

	if op.Valid() {
		t.Errorf("an unimplemented mechanism yielded a valid operation: %s", op)
	}

	if newOperation("", 0xD0).Valid() {
		t.Error("the empty mechanism produced a valid operation")
	}
}

// The order inputs are listed in is the order they are compared and printed in,
// and it must not be the order a map happens to iterate in. This is the internal
// test because it builds an entry by hand, with keys the real catalog does not
// have.
func TestRecordedInputsSortByKindThenPort(t *testing.T) {
	t.Parallel()

	model := Model{
		Name:   "TEST-1",
		Vendor: "ACME",
		Inputs: map[Input]inputOp{
			"hdmi10":      {mechanism: MechanismLGAltInput, value: 0x90, evidence: "test"},
			"hdmi2":       {mechanism: MechanismLGAltInput, value: 0x91, evidence: "test"},
			"usb-c":       {mechanism: MechanismLGAltInput, value: 0xD1, evidence: "test"},
			"dp":          {mechanism: MechanismLGAltInput, value: 0xD0, evidence: "test"},
			"thunderbolt": {mechanism: MechanismLGAltInput, value: 0xD2, evidence: "test"},
		},
	}

	want := []Input{"dp", "hdmi2", "hdmi10", "usb-c", "thunderbolt"}

	got := model.RecordedInputs()
	if !slices.Equal(got, want) {
		t.Errorf("RecordedInputs() = %v, want %v", got, want)
	}
}

// CompareInputs has to be a total order even for a name that does not parse, or
// a sort over a hand-built entry could misbehave rather than merely order it
// oddly. The compiled catalog cannot hold such a key - TestEveryRecordedInputParses
// enforces that - so this is the only place one exists.
func TestCompareInputsIsTotalEvenForNamesThatDoNotParse(t *testing.T) {
	t.Parallel()

	unsorted := []Input{"zzz", "hdmi2", "aaa", "dp", "usb-c"}
	want := []Input{"dp", "hdmi2", "usb-c", "aaa", "zzz"}

	sorted := slices.Clone(unsorted)
	slices.SortFunc(sorted, CompareInputs)

	if !slices.Equal(sorted, want) {
		t.Errorf("sorted = %v, want %v", sorted, want)
	}

	if CompareInputs("dp", "dp") != 0 || CompareInputs("aaa", "aaa") != 0 {
		t.Error("CompareInputs does not report a name equal to itself")
	}
}
