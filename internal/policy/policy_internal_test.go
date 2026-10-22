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
	"errors"
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

// refusedCode is a product code the tested LG does not report, used as a mode
// of it that the catalog refuses.
const refusedCode = 0x77FF

// refusedReason is what that mode is refused with.
const refusedReason = "Picture-by-picture is on. Turn it off in the OSD."

// withRefusedMode is a matcher over the real write-enabled LG with one refused
// identity grafted onto a copy of it, so the refusal is tested against an entry
// that would otherwise switch.
func withRefusedMode(t *testing.T) matcher {
	t.Helper()

	model, found := catalog.Find("LG/38WR85QC-W")
	if !found {
		t.Fatal("the tested LG is not in the catalog")
	}

	model.RefusedIdentities = append(model.RefusedIdentities, catalog.RefusedIdentity{
		Identity: catalog.Identity{Manufacturer: "GSM", ProductCode: refusedCode},
		Reason:   refusedReason,
	})

	return func(identity edid.Identity) (catalog.Model, catalog.MatchResult) {
		if !model.Matches(identity) {
			return catalog.Model{}, catalog.MatchNone
		}

		return model, catalog.MatchExact
	}
}

// refusedDisplay is the LG reporting the refused product code.
func refusedDisplay() backend.Display {
	display := writableDisplay()
	display.Identity.ProductCode = refusedCode

	return display
}

// A refused identity names its model and is refused for its mode, with the
// catalog's reason as the detail, whatever input was asked for - one the model
// enables included.
func TestARefusedIdentityIsRefusedForItsMode(t *testing.T) {
	t.Parallel()

	for _, input := range []catalog.Input{"dp", "usb-c", "hdmi1"} {
		_, err := resolve(
			[]backend.Display{refusedDisplay()},
			Request{Input: input},
			withRefusedMode(t),
		)
		if !refusal.Is(err, refusal.UnsupportedMode) {
			t.Fatalf("%s: error = %v, want unsupported-mode", input, err)
		}

		declined, _ := errors.AsType[*refusal.Refusal](err)
		if declined.Detail != refusedReason {
			t.Errorf("%s: detail = %q, want the catalog's reason", input, declined.Detail)
		}

		if len(declined.Detected) != 1 || declined.Detected[0].ProductCode != refusedCode {
			t.Errorf("%s: detected = %v, want the refused display", input, declined.Detected)
		}
	}
}

// The refusal belongs to the identity, not to the model: the same entry reached
// through an identity it allows still switches.
func TestAnAllowedIdentityOfAModelWithARefusedOneStillDecides(t *testing.T) {
	t.Parallel()

	decision, err := resolve(
		[]backend.Display{writableDisplay()},
		Request{Input: catalog.Input("dp")},
		withRefusedMode(t),
	)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if decision.Operation.Value() != 0xD0 || decision.Assumed {
		t.Errorf("decision = %+v", decision)
	}
}

// A recognized monitor is still one the user may have meant, so a refused
// display next to an allowed one is two candidates rather than a quiet pick of
// the other.
func TestARefusedDisplayStillCountsAsACandidate(t *testing.T) {
	t.Parallel()

	other := writableDisplay()
	other.Handle, other.Label = "card1-DP-2", "card1-DP-2"
	other.Identity.SerialString = "TESTSERIAL02"

	_, err := resolve(
		[]backend.Display{refusedDisplay(), other},
		Request{Input: catalog.Input("dp")},
		withRefusedMode(t),
	)
	if !refusal.Is(err, refusal.MultipleCandidates) {
		t.Errorf("error = %v, want multiple-candidates", err)
	}

	// Pinning the allowed one switches it; pinning the refused one refuses.
	_, err = resolve(
		[]backend.Display{refusedDisplay(), other},
		Request{Input: catalog.Input("dp"), Serial: "TESTSERIAL02"},
		withRefusedMode(t),
	)
	if err != nil {
		t.Errorf("pinning the allowed display: %v", err)
	}

	_, err = resolve(
		[]backend.Display{refusedDisplay(), other},
		Request{Input: catalog.Input("dp"), Serial: "TESTSERIAL01"},
		withRefusedMode(t),
	)
	if !refusal.Is(err, refusal.UnsupportedMode) {
		t.Errorf("pinning the refused display: error = %v, want unsupported-mode", err)
	}
}
