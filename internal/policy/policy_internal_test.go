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

package policy

import (
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

func writableDisplay() backend.Display {
	return backend.Display{
		Identity: edid.Identity{
			Manufacturer: "GSM",
			ProductCode:  0x77D3,
			SerialString: "TESTSERIAL01",
		},
		Handle:   "card1-DP-1",
		Label:    "card1-DP-1",
		Writable: true,
		Status:   backend.StatusOK,
	}
}

// A catalog whose entries disagree about who owns an identity is a bug in the
// catalog, not a choice for policy to make. It must refuse rather than pick.
func TestResolveRefusesAnAmbiguousCatalogMatch(t *testing.T) {
	t.Parallel()

	ambiguous := func(edid.Identity) (catalog.Model, catalog.MatchResult) {
		return catalog.Model{}, catalog.MatchAmbiguous
	}

	_, err := resolve(
		[]backend.Display{writableDisplay()},
		Request{Input: catalog.Input("dp")},
		ambiguous,
	)
	if !refusal.Is(err, refusal.AmbiguousCatalog) {
		t.Errorf("error = %v, want ambiguous-catalog", err)
	}
}

// An ambiguous match must not be rescued by another display that happens to
// match cleanly: the catalog is untrustworthy until a human fixes it.
func TestAnAmbiguousMatchStopsTheWholeResolution(t *testing.T) {
	t.Parallel()

	second := writableDisplay()
	second.Handle = "card1-DP-2"
	second.Label = "card1-DP-2"
	second.Identity.ProductCode = 0x77D4

	calls := 0
	firstIsAmbiguous := func(identity edid.Identity) (catalog.Model, catalog.MatchResult) {
		calls++

		if identity.ProductCode == 0x77D3 {
			return catalog.Model{}, catalog.MatchAmbiguous
		}

		return catalog.Match(identity)
	}

	_, err := resolve(
		[]backend.Display{writableDisplay(), second},
		Request{Input: catalog.Input("dp")},
		firstIsAmbiguous,
	)

	if !refusal.Is(err, refusal.AmbiguousCatalog) {
		t.Errorf("error = %v, want ambiguous-catalog", err)
	}

	if calls != 1 {
		t.Errorf("kept matching after an ambiguous result: %d lookups", calls)
	}
}

// An unwritable display is still a display the catalog can disagree about, and
// a catalog collision must not be reported as an unknown monitor.
func TestAnAmbiguousMatchOnAnUnwritableDisplayIsStillAmbiguous(t *testing.T) {
	t.Parallel()

	unwritable := writableDisplay()
	unwritable.Writable = false
	unwritable.Status = backend.StatusNoDDCChannel

	ambiguous := func(edid.Identity) (catalog.Model, catalog.MatchResult) {
		return catalog.Model{}, catalog.MatchAmbiguous
	}

	_, err := resolve([]backend.Display{unwritable}, Request{Input: catalog.Input("dp")}, ambiguous)
	if !refusal.Is(err, refusal.AmbiguousCatalog) {
		t.Errorf("error = %v, want ambiguous-catalog", err)
	}
}
