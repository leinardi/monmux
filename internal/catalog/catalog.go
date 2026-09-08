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

// Package catalog is the built-in supported-monitor catalog and the only place
// in monmux where the bytes that reach a monitor are decided.
//
// The catalog is written in models.yaml and rendered into models_gen.go, which is
// committed and is what the binary compiles. The rendering happens at development
// time, so the binary carries no catalog parser and reads no catalog file at run
// time: entries are compile-time typed, and adding a model is a reviewable diff of
// literal bytes with recorded evidence next to them, in both files at once. No
// flag, config key or environment variable can introduce a value here.
//
// Edit models.yaml, run `make go-generate`, and commit both files.
// TestGeneratedCatalogMatchesTheYAML fails the build if the two ever disagree.
//
// The types enforce the fail-closed rule structurally. An [Operation] can only be
// built by this package from a catalog entry; its zero value is invalid and every
// backend rejects it. An [Input] is a name, not a value: it selects an entry and
// never carries a byte of its own.
package catalog

//go:generate go run ./internal/generate/cmd -in models.yaml -out models_gen.go

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/leinardi/monmux/internal/catalog/internal/input"
	"github.com/leinardi/monmux/internal/edid"
)

// ErrUnknownInput reports a symbolic input name that is not well formed. It
// wraps [input.ErrUnknownKind] and [input.ErrMalformedNumber], so a caller that
// cares which rule was broken can still tell.
var ErrUnknownInput = errors.New("catalog: unknown input")

// Input is a symbolic, vendor-neutral input name, as typed on the command line:
// a connector kind and an optional port number, e.g. dp, hdmi2, usb-c. It only
// selects a catalog entry; it never carries a VCP code or a value, and users
// never type one (req. 9.5).
type Input string

// Kind is a connector kind, the first half of an [Input]. The set is closed and
// defined in internal/input; a new kind is a Go change, a new port of a known
// kind is only a models.yaml edit.
type Kind = input.Kind

// Kinds returns every connector kind, in the order inputs are listed.
func Kinds() []Kind {
	return input.Kinds()
}

// ParseInput converts a command-line argument into an [Input]. A name that is
// not well formed is an error: there is no "pass it through and hope" path.
// Whether the matched model enables the input is a separate, later question.
func ParseInput(name string) (Input, error) {
	_, err := input.Parse(name)
	if err != nil {
		return "", fmt.Errorf("%w: %q: %w", ErrUnknownInput, name, err)
	}

	return Input(name), nil
}

// String returns the symbolic name, as typed on the command line.
func (i Input) String() string {
	return string(i)
}

// Label returns the human-readable name for messages, e.g. "DisplayPort" or
// "HDMI 2". A name that does not parse is returned as it is, so a diagnostic
// still says what it was handed.
func (i Input) Label() string {
	parsed, err := input.Parse(string(i))
	if err != nil {
		return string(i)
	}

	return parsed.Label()
}

// CompareInputs orders two input names: by connector kind, then by port number,
// with a bare kind before its numbered ports. It is a total order, so sorting is
// transitive even in the presence of a name that does not parse: every parseable
// name sorts before every unparseable one, and unparseable ones sort lexically.
// The compiled catalog cannot hold an unparseable key - a test enforces that -
// but a hand-built [Model] in a test must not be able to make a sort misbehave.
func CompareInputs(first, second Input) int {
	left, leftErr := input.Parse(string(first))
	right, rightErr := input.Parse(string(second))

	switch {
	case leftErr != nil && rightErr != nil:
		return strings.Compare(string(first), string(second))
	case leftErr != nil:
		return 1
	case rightErr != nil:
		return -1
	default:
		return input.Compare(left, right)
	}
}

// KnownInputs returns every input any catalog entry records, de-duplicated and
// in listing order. The CLI completes against it; it is not a promise that any
// of them is writable, which is a per-model question.
func KnownInputs() []Input {
	seen := map[Input]bool{}

	var known []Input

	for _, model := range models {
		for name := range model.Inputs {
			if seen[name] {
				continue
			}

			seen[name] = true

			known = append(known, name)
		}
	}

	slices.SortFunc(known, CompareInputs)

	return known
}

// Mechanism is how a model's input is switched. It is a closed enum with one
// value today, and it is the extension point for other vendors: a second
// mechanism (the standard VCP 0x60 Input Source feature, say) is added only
// together with the first evidenced model that needs it, never speculatively.
//
// The mechanism is a per-model property. It is never a fallback: if a backend
// does not implement a model's mechanism it refuses with invalid-operation
// rather than trying another one (req. 9.7).
type Mechanism string

// MechanismLGAltInput is the LG side channel documented by ddcutil: a SetVCP of
// the manufacturer-specific VCP code 0xF4 sent with DDC/CI source address 0x50,
// with no read-back verification. The value written is model-specific, which is
// exactly why it lives in a per-model catalog entry.
const MechanismLGAltInput Mechanism = "lg-alt-input"

// mechanisms is every mechanism monmux implements. A value outside this set can
// never become a valid [Operation], so a catalog entry that names an unknown
// mechanism is inert rather than dangerous.
var mechanisms = []Mechanism{MechanismLGAltInput}

// Mechanisms returns every implemented mechanism.
func Mechanisms() []Mechanism {
	return slices.Clone(mechanisms)
}

// Known reports whether this mechanism is one monmux implements.
func (m Mechanism) Known() bool {
	return slices.Contains(mechanisms, m)
}

const (
	// LGAltInputSourceAddr is the DDC/CI source-address byte the LG side channel
	// uses. It is not a request to touch EDID storage, despite 0x50 also being
	// the usual I2C address of an EDID EEPROM.
	LGAltInputSourceAddr = 0x50

	// LGAltInputVCP is the manufacturer-specific VCP code for the LG side
	// channel's input-switch command.
	LGAltInputVCP = 0xF4
)

// String returns the mechanism name as it appears in output and documentation.
func (m Mechanism) String() string {
	return string(m)
}

// Grade is how strong the evidence for one input is. It is a closed enum, and
// it is what the rule "monmux only writes what somebody ran on that unit" is
// written in: a write-enabled model carries [GradeVerified] on every input, and
// the generator refuses the catalog file otherwise.
type Grade string

const (
	// GradeVerified is a direct test on the unit itself, by this project.
	GradeVerified Grade = "verified"
	// GradeDocumented is the manufacturer's own documentation, with no field
	// report behind it.
	GradeDocumented Grade = "documented"
	// GradeReported is somebody reporting that switching that named input with
	// that value worked.
	GradeReported Grade = "reported"
	// GradeQuoted is the weakest: a report that quotes the values and says they
	// work, without saying which inputs were tried individually.
	GradeQuoted Grade = "quoted"
)

// grades is every grade, strongest first. It is the order the documentation
// lists them in, and the generator mirrors it.
var grades = []Grade{GradeVerified, GradeDocumented, GradeReported, GradeQuoted}

// Grades returns every evidence grade, strongest first.
func Grades() []Grade {
	return slices.Clone(grades)
}

// Known reports whether this grade is one the catalog defines.
func (g Grade) Known() bool {
	return slices.Contains(grades, g)
}

// String returns the grade as it appears in output and documentation.
func (g Grade) String() string {
	return string(g)
}

// Operation is a single, fully decided write: which mechanism, and which value.
// It is deliberately opaque. The fields are unexported and the constructor is
// package-private, so the only way to obtain a valid Operation is to look one up
// in this catalog. The zero value is invalid, and every backend rejects it.
type Operation struct {
	mechanism Mechanism
	value     uint8
	valid     bool
}

// newOperation builds a valid operation. It is package-private on purpose: no
// caller outside the catalog may invent one. An unknown mechanism yields the
// invalid zero value rather than something a backend might try to run.
func newOperation(mechanism Mechanism, value uint8) Operation {
	if !mechanism.Known() {
		return Operation{}
	}

	return Operation{mechanism: mechanism, value: value, valid: true}
}

// Mechanism returns the mechanism the backend must implement to run this.
func (o Operation) Mechanism() Mechanism {
	return o.mechanism
}

// Value returns the model-specific value to write.
func (o Operation) Value() uint8 {
	return o.value
}

// Valid reports whether this operation came from a catalog entry. A backend that
// is handed an invalid operation refuses with invalid-operation.
func (o Operation) Valid() bool {
	return o.valid
}

// String renders the operation for dry-run and diagnostic output.
func (o Operation) String() string {
	if !o.valid {
		return "invalid operation"
	}

	return fmt.Sprintf("%s 0x%02X", o.mechanism, o.value)
}

// Identity is one EDID fingerprint a model is recognized by. One physical model
// can expose several: the tested 38WR85QC-W reports a different product code
// over DisplayPort than over USB-C.
type Identity struct {
	// Manufacturer is the three-letter PNP ID, e.g. "GSM".
	Manufacturer string
	// ProductCode is the EDID product code, e.g. 0x77D3.
	ProductCode uint16
}

// String renders the fingerprint the way monmux documentation writes it.
func (i Identity) String() string {
	return fmt.Sprintf("%s/0x%04X", i.Manufacturer, i.ProductCode)
}

// inputOp is what the catalog records for one input of one model: how to switch,
// what value to write, and the evidence for that value. Evidence is per input,
// not per model, because a model can have one verified input and one that is
// merely reported elsewhere.
type inputOp struct {
	mechanism Mechanism
	value     uint8
	grade     Grade
	evidence  string
}

// Model is one catalog entry.
type Model struct {
	// Name is the model name, e.g. "38WR85QC-W".
	Name string
	// Vendor is the manufacturer's name, e.g. "LG".
	Vendor string
	// Identities are the EDID fingerprints this model is recognized by. A model
	// with no identities can never match, and so can never be written to.
	Identities []Identity
	// WriteEnabled says whether monmux may write to this model at all. An entry
	// can be documented without being trusted.
	WriteEnabled bool
	// Inputs are the inputs recorded for this model, with their evidence.
	Inputs map[Input]inputOp
	// Notes are per-model prose: negative reports, conflicts, aliases and
	// quirks. They are documentation and never an [Operation], so nothing
	// written here can reach a monitor.
	Notes []string
	// Sources are model-level references backing the entry.
	Sources []string
}

// FullName returns "LG 38WR85QC-W".
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) FullName() string {
	return m.Vendor + " " + m.Name
}

// Operation returns the operation for an input of this model. The second result
// is false when the model is not write-enabled or has no entry for that input;
// there is no fallback and no default.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) Operation(input Input) (Operation, bool) {
	op, ok := m.Inputs[input]
	if !ok || !m.WriteEnabled {
		return Operation{}, false
	}

	operation := newOperation(op.mechanism, op.value)
	if !operation.Valid() {
		return Operation{}, false
	}

	return operation, true
}

// RecordedInputs returns every input this entry documents, in listing order,
// whether or not the model is write-enabled.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) RecordedInputs() []Input {
	return slices.SortedFunc(maps.Keys(m.Inputs), CompareInputs)
}

// EnabledInputs returns the inputs monmux may actually switch this model to. For
// a model that is not write-enabled, that is none of them.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) EnabledInputs() []Input {
	if !m.WriteEnabled {
		return nil
	}

	return m.RecordedInputs()
}

// InputValue returns the recorded value for an input, whether or not the model
// is write-enabled. Documentation checks use it; nothing that writes does.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) InputValue(input Input) (uint8, bool) {
	op, ok := m.Inputs[input]
	if !ok {
		return 0, false
	}

	return op.value, true
}

// InputEvidence returns the evidence recorded for an input.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) InputEvidence(input Input) (string, bool) {
	op, ok := m.Inputs[input]
	if !ok {
		return "", false
	}

	return op.evidence, true
}

// InputGrade returns the grade of the evidence recorded for an input.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) InputGrade(input Input) (Grade, bool) {
	op, ok := m.Inputs[input]
	if !ok {
		return "", false
	}

	return op.grade, true
}

// InputMechanism returns the mechanism recorded for an input.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) InputMechanism(input Input) (Mechanism, bool) {
	op, ok := m.Inputs[input]
	if !ok {
		return "", false
	}

	return op.mechanism, true
}

// Matches reports whether an EDID identity is one of this model's fingerprints.
//
//nolint:gocritic // hugeParam: a value receiver keeps Model usable where it is not addressable
func (m Model) Matches(identity edid.Identity) bool {
	for _, known := range m.Identities {
		if known.Manufacturer == identity.Manufacturer &&
			known.ProductCode == identity.ProductCode {
			return true
		}
	}

	return false
}

// MatchResult is the outcome of looking an identity up in the catalog.
type MatchResult int

const (
	// MatchNone means no entry claims this identity: an unknown monitor.
	MatchNone MatchResult = iota
	// MatchExact means exactly one entry claims it.
	MatchExact
	// MatchAmbiguous means more than one entry claims it, which is a catalog
	// bug rather than a user problem. It is fail-closed all the same.
	MatchAmbiguous
)

// String renders the match result for diagnostics.
func (r MatchResult) String() string {
	switch r {
	case MatchNone:
		return "none"
	case MatchExact:
		return "exact"
	case MatchAmbiguous:
		return "ambiguous"
	default:
		return "unknown"
	}
}

// Match looks an EDID identity up in the catalog. On anything but [MatchExact]
// the returned model is the zero value and must not be used.
func Match(identity edid.Identity) (Model, MatchResult) {
	return matchIn(identity, models)
}

// matchIn is Match against an arbitrary set of entries, so the ambiguous branch
// can be tested with a catalog that has the collision the real one must not have.
func matchIn(identity edid.Identity, entries []Model) (Model, MatchResult) {
	var (
		found Model
		count int
	)

	for _, model := range entries {
		if !model.Matches(identity) {
			continue
		}

		count++

		if count == 1 {
			found = model
		}
	}

	switch {
	case count == 0:
		return Model{}, MatchNone
	case count > 1:
		return Model{}, MatchAmbiguous
	default:
		return cloneModel(found), MatchExact
	}
}

// cloneModel returns a copy that shares nothing with the compiled-in catalog:
// the identity and source slices and the input map are all copied, so a caller
// cannot repoint an entry at another monitor or graft an unverified value onto
// a write-enabled model.
//
//nolint:gocritic // hugeParam: cloning is exactly what this does; a pointer would defeat it
func cloneModel(model Model) Model {
	model.Identities = slices.Clone(model.Identities)
	model.Notes = slices.Clone(model.Notes)
	model.Sources = slices.Clone(model.Sources)
	model.Inputs = maps.Clone(model.Inputs)

	return model
}

// Models returns every catalog entry, in catalog order. Both the slice and the
// entries are copies: mutating anything reachable from the result cannot change
// what a later [Match] returns.
func Models() []Model {
	clones := make([]Model, 0, len(models))
	for _, model := range models {
		clones = append(clones, cloneModel(model))
	}

	return clones
}
