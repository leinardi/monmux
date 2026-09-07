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

// Package policy decides whether a switch may happen, and to what. It is pure:
// it does no I/O, talks to no tool, and touches no monitor. It is handed the
// displays a backend enumerated and the user's request, and it either produces a
// [Decision] or refuses.
//
// The reasons it can give are display-level only: nothing here knows whether a
// tool is installed or whether a target is reachable. Those readiness questions
// belong to internal/app, which asks the backend.
package policy

import (
	"strings"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

// NoSerialDetail is the detail reported when a pin was requested but the display
// exposes no alphanumeric serial to pin against. Pinning is simply unavailable
// for such a unit, and the documentation says so.
const NoSerialDetail = "display exposes no alphanumeric serial"

// Request is what the user asked for.
type Request struct {
	// Input is the symbolic input to switch to.
	Input catalog.Input
	// Serial optionally pins the request to one physical unit. It matches the
	// alphanumeric serial string only, never the numeric serial: exact,
	// case-sensitive, after trimming surrounding whitespace on both sides.
	Serial string
}

// Decision is a switch monmux is willing to perform. Holding one means the
// display was identified, the model is write-enabled, and the input is enabled
// for that model with recorded evidence.
type Decision struct {
	// Display is the display to write to.
	Display backend.Display
	// Model is the catalog entry it was identified as.
	Model catalog.Model
	// Operation is the mechanism and value to write. It is always valid.
	Operation catalog.Operation
}

// matcher looks an identity up in the catalog. It exists as a seam so the
// ambiguous-catalog branch can be tested: the real catalog must never contain a
// duplicated identity, and a test enforces that. Production always passes
// [catalog.Match].
type matcher func(edid.Identity) (catalog.Model, catalog.MatchResult)

// candidate is a writable display that matched exactly one catalog entry.
type candidate struct {
	display backend.Display
	model   catalog.Model
}

// Resolve picks the single display to write to, or explains why it will not.
//
// The order is deliberate: a serial pin narrows first, so a user with two
// monitors gets "that serial is not here" rather than "pick one"; unwritable
// displays are then dropped, because they can be reported but never selected;
// what remains is matched against the catalog; and only then is the requested
// input checked against the matched model.
func Resolve(displays []backend.Display, req Request) (Decision, error) {
	return resolve(displays, req, catalog.Match)
}

// resolve is Resolve against an arbitrary catalog lookup.
func resolve(displays []backend.Display, req Request, match matcher) (Decision, error) {
	if len(displays) == 0 {
		return Decision{}, refusal.New(refusal.NoDisplays, "")
	}

	pinned := strings.TrimSpace(req.Serial)
	if req.Serial != "" && pinned == "" {
		return Decision{}, refusal.New(
			refusal.SerialMismatch,
			"The requested serial is blank.",
			identities(displays)...,
		)
	}

	selected := displays

	if pinned != "" {
		matched, err := pin(displays, pinned)
		if err != nil {
			return Decision{}, err
		}

		selected = matched
	}

	writable, unwritable := partition(selected)

	candidates, err := supported(writable, match)
	if err != nil {
		return Decision{}, err
	}

	switch {
	case len(candidates) == 0:
		return Decision{}, nothingToWriteTo(selected, unwritable, match)
	case len(candidates) > 1:
		return Decision{}, refusal.New(
			refusal.MultipleCandidates,
			"Matched: "+labels(candidates)+".",
			identitiesOf(candidates)...,
		)
	}

	return decide(&candidates[0], req.Input)
}

// pin narrows the displays to the one whose alphanumeric serial matches. The
// numeric serial is never consulted: it is not what a user reads off a label,
// and two different units can carry the same one.
func pin(displays []backend.Display, wanted string) ([]backend.Display, error) {
	matched := make([]backend.Display, 0, 1)
	unpinnable := make([]string, 0, len(displays))

	for _, display := range displays {
		serial := strings.TrimSpace(display.Identity.SerialString)
		if serial == "" {
			unpinnable = append(unpinnable, display.Label)
		}

		if serial != "" && serial == wanted {
			matched = append(matched, display)
		}
	}

	if len(matched) == 0 {
		// Naming the units that cannot be pinned at all matters more than the
		// generic sentence: the user would otherwise assume a typo and retry a
		// pin that can never work for that monitor.
		detail := "No attached display carries the requested serial."
		if len(unpinnable) > 0 {
			detail = strings.Join(unpinnable, ", ") + ": " + NoSerialDetail
		}

		return nil, refusal.New(refusal.SerialMismatch, detail, identities(displays)...)
	}

	return matched, nil
}

// partition splits displays into those that can be written to and those that
// cannot. Unwritable displays are kept so a refusal can name them.
func partition(displays []backend.Display) (writable, unwritable []backend.Display) {
	for _, display := range displays {
		if display.Writable {
			writable = append(writable, display)

			continue
		}

		unwritable = append(unwritable, display)
	}

	return writable, unwritable
}

// supported returns the writable displays that match exactly one catalog entry.
// An identity claimed by more than one entry is a catalog bug, and it refuses
// rather than picking one.
func supported(displays []backend.Display, match matcher) ([]candidate, error) {
	candidates := make([]candidate, 0, len(displays))

	for _, display := range displays {
		model, result := match(display.Identity)

		switch result {
		case catalog.MatchExact:
			candidates = append(candidates, candidate{display: display, model: model})
		case catalog.MatchAmbiguous:
			return nil, refusal.New(
				refusal.AmbiguousCatalog,
				"Display: "+display.Label+".",
				display.Identity,
			)
		case catalog.MatchNone:
		}
	}

	return candidates, nil
}

// nothingToWriteTo explains why no display survived. An unwritable display that
// would otherwise have been the target is the interesting case: the monitor is
// supported, monmux simply cannot reach it.
func nothingToWriteTo(selected, unwritable []backend.Display, match matcher) error {
	for _, display := range unwritable {
		_, result := match(display.Identity)

		switch result {
		case catalog.MatchExact:
			return refusal.New(
				refusal.DisplayNotWritable,
				"Display "+display.Label+" is supported but not writable (status: "+display.Status+").",
				display.Identity,
			)
		case catalog.MatchAmbiguous:
			// A catalog collision is never quietly absorbed, not even on a
			// display monmux could not have written to anyway.
			return refusal.New(
				refusal.AmbiguousCatalog,
				"Display: "+display.Label+".",
				display.Identity,
			)
		case catalog.MatchNone:
		}
	}

	return refusal.New(refusal.UnknownMonitor, "", identities(selected)...)
}

// decide checks the requested input against the matched model. This is the last
// gate: an input with no recorded evidence for this model is refused, and no
// value from another model or another input is ever substituted.
func decide(target *candidate, input catalog.Input) (Decision, error) {
	operation, enabled := target.model.Operation(input)
	if !enabled {
		return Decision{}, refusal.New(
			refusal.InputNotEnabled,
			inputDetail(&target.model, input),
			target.display.Identity,
		)
	}

	return Decision{
		Display:   target.display,
		Model:     target.model,
		Operation: operation,
	}, nil
}

// inputDetail says which input was asked for and which ones this model has.
func inputDetail(model *catalog.Model, input catalog.Input) string {
	detail := "Requested " + input.String() + " on " + model.FullName() + "; enabled inputs: "

	enabled := model.EnabledInputs()
	if len(enabled) == 0 {
		return detail + "none."
	}

	names := make([]string, 0, len(enabled))
	for _, enabledInput := range enabled {
		names = append(names, enabledInput.String())
	}

	return detail + strings.Join(names, ", ") + "."
}

// identities returns the identity of every display, for the refusal message.
func identities(displays []backend.Display) []edid.Identity {
	found := make([]edid.Identity, 0, len(displays))
	for _, display := range displays {
		found = append(found, display.Identity)
	}

	return found
}

// identitiesOf returns the identity of every candidate.
func identitiesOf(candidates []candidate) []edid.Identity {
	found := make([]edid.Identity, 0, len(candidates))
	for index := range candidates {
		found = append(found, candidates[index].display.Identity)
	}

	return found
}

// labels lists the candidates by their public labels, never by their handles:
// a handle can be private data.
func labels(candidates []candidate) string {
	names := make([]string, 0, len(candidates))
	for index := range candidates {
		target := &candidates[index]
		names = append(names, target.display.Label+" ("+target.model.FullName()+")")
	}

	return strings.Join(names, ", ")
}
