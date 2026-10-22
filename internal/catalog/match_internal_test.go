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

// LG assigns one EDID product code to several products, so the catalog has to be
// able to hold two models behind one code. The pinned model name is what tells
// them apart, and a display that reports no model name matches neither: the
// fail-closed answer, not a coin flip.
//
// This is the internal test because it builds entries by hand; the real catalog
// pins no name today, and the generator refuses a bare identity next to a pinned
// one so that a bare entry can never shadow a pinned one.
func TestSharedProductCodeIsDisambiguatedByModelName(t *testing.T) {
	t.Parallel()

	entries := []Model{
		{
			Name:   "first",
			Vendor: "LG",
			Identities: []Identity{
				{Manufacturer: "GSM", ProductCode: 0x7707, ModelName: "LG HDR 4K"},
			},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				"dp": {mechanism: MechanismLGAltInput, value: 0xD0, evidence: "test"},
			},
		},
		{
			Name:   "second",
			Vendor: "LG",
			Identities: []Identity{
				{Manufacturer: "GSM", ProductCode: 0x7707, ModelName: "LG ULTRAFINE"},
			},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				"dp": {mechanism: MechanismLGAltInput, value: 0xD1, evidence: "test"},
			},
		},
	}

	for name, want := range map[string]string{"LG HDR 4K": "first", "LG ULTRAFINE": "second"} {
		model, result := matchIn(
			edid.Identity{Manufacturer: "GSM", ProductCode: 0x7707, ModelName: name},
			entries,
		)

		if result != MatchExact || model.Name != want {
			t.Errorf("%q gave %s for %q, want exact for %q", name, result, model.Name, want)
		}
	}

	for _, name := range []string{"", "LG SOMETHING ELSE"} {
		model, result := matchIn(
			edid.Identity{Manufacturer: "GSM", ProductCode: 0x7707, ModelName: name},
			entries,
		)

		if result != MatchNone {
			t.Errorf("model name %q gave %s for %s, want none", name, result, model.FullName())
		}
	}
}

// The pinned name is compared against whatever the backend put in the identity,
// and the two backends fill that field from different places: EDID descriptor
// 0xFC on Linux, m1ddc's "Product name" on macOS. They must agree, or an entry
// would match on one operating system and not on the other. The parsers are
// tested where they live; this is the catalog end of the same rule.
func TestAPinnedNameMatchesTheSameStringFromEitherBackend(t *testing.T) {
	t.Parallel()

	entries := []Model{
		{
			Name:   "pinned",
			Vendor: "LG",
			Identities: []Identity{
				{Manufacturer: "GSM", ProductCode: 0x77D3, ModelName: "LG ULTRAWIDE"},
			},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				"dp": {mechanism: MechanismLGAltInput, value: 0xD0, evidence: "test"},
			},
		},
	}

	// What internal/edid produces from descriptor 0xFC, and what
	// internal/backend/m1ddc produces from `display list detailed`: the same
	// string, with the serial fields the two backends fill differently.
	fromLinux := edid.Identity{
		Manufacturer: "GSM",
		ProductCode:  0x77D3,
		SerialNumber: 0x01020304,
		SerialString: "TESTSERIAL01",
		ModelName:    "LG ULTRAWIDE",
	}
	fromMacOS := edid.Identity{
		Manufacturer: "GSM",
		ProductCode:  0x77D3,
		SerialNumber: 0x01020304,
		ModelName:    "LG ULTRAWIDE",
	}

	for name, identity := range map[string]edid.Identity{
		"linux": fromLinux,
		"macos": fromMacOS,
	} {
		model, result := matchIn(identity, entries)
		if result != MatchExact || model.Name != "pinned" {
			t.Errorf("%s gave %s for %s, want exact", name, result, model.FullName())
		}
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

// refusingEntries is one write-enabled entry with an allowed identity and a
// refused one, pinned so the model name has to agree, and a second entry that
// claims nothing.
func refusingEntries() []Model {
	return []Model{
		{
			Name:       "first",
			Vendor:     "ACME",
			Identities: []Identity{{Manufacturer: "ACM", ProductCode: 0x0001}},
			RefusedIdentities: []RefusedIdentity{{
				Identity: Identity{Manufacturer: "ACM", ProductCode: 0x0002, ModelName: "ACME PBP"},
				Reason:   "Picture-by-picture is on.",
			}},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				"dp": {mechanism: MechanismInputSource, value: 0x0F, evidence: "test"},
			},
		},
		{Name: "second", Vendor: "ACME"},
	}
}

// A refused identity is recognized - the model is named - and refused, under
// the same comparison as an allowed one, the pinned model name included.
func TestARefusedIdentityMatchesAndIsRefused(t *testing.T) {
	t.Parallel()

	refused := edid.Identity{Manufacturer: "ACM", ProductCode: 0x0002, ModelName: "ACME PBP"}

	model, result := matchIn(refused, refusingEntries())
	if result != MatchExact || model.Name != "first" {
		t.Fatalf("match = %s for %q, want exact for first", result, model.Name)
	}

	reason, isRefused := model.IdentityRefusal(refused)
	if !isRefused || reason != "Picture-by-picture is on." {
		t.Errorf("refusal = %q, %t", reason, isRefused)
	}

	allowed := edid.Identity{Manufacturer: "ACM", ProductCode: 0x0001}

	_, isRefused = model.IdentityRefusal(allowed)
	if isRefused {
		t.Error("the allowed identity is refused")
	}

	// The pinned name is part of the fingerprint: without it the display is
	// not this mode at all, and so not this model.
	unnamed := edid.Identity{Manufacturer: "ACM", ProductCode: 0x0002}

	_, result = matchIn(unnamed, refusingEntries())
	if result != MatchNone {
		t.Errorf("an unpinned display matched a pinned refused identity: %s", result)
	}
}

// One display claimed by two entries is ambiguous whether either claim allows a
// write or refuses one.
func TestARefusedIdentityClaimedTwiceIsAmbiguous(t *testing.T) {
	t.Parallel()

	entries := refusingEntries()
	entries[1].Identities = []Identity{{Manufacturer: "ACM", ProductCode: 0x0002}}

	_, result := matchIn(
		edid.Identity{Manufacturer: "ACM", ProductCode: 0x0002, ModelName: "ACME PBP"},
		entries,
	)
	if result != MatchAmbiguous {
		t.Errorf("match = %s, want ambiguous", result)
	}
}

// The refused identities a caller is handed are a copy: dropping one must not
// make the compiled-in entry write to that mode.
func TestRefusedIdentitiesAreClonedWithTheModel(t *testing.T) {
	t.Parallel()

	entries := refusingEntries()

	handedOut := cloneModel(entries[0])
	handedOut.RefusedIdentities[0].Identity.ProductCode = 0x0003

	if entries[0].RefusedIdentities[0].Identity.ProductCode != 0x0002 {
		t.Error("mutating a handed-out model changed the entry's refused identity")
	}
}
