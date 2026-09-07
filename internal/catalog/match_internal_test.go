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
				InputDP: {mechanism: MechanismLGAltInput, value: 0xD0, evidence: "test"},
			},
		},
		{
			Name:         "second",
			Vendor:       "LG",
			Identities:   []Identity{{Manufacturer: "GSM", ProductCode: 0x77D3}},
			WriteEnabled: true,
			Inputs: map[Input]inputOp{
				InputDP: {mechanism: MechanismLGAltInput, value: 0xD1, evidence: "test"},
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
				InputDP: {mechanism: MechanismLGAltInput, value: 0xD0, evidence: "test"},
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
			InputDP: {mechanism: Mechanism("not-implemented"), value: 0xD0, evidence: "test"},
		},
	}

	op, ok := model.Operation(InputDP)
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
