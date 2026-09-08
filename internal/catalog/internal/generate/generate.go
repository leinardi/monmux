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

// Package generate renders the supported-monitor catalog from models.yaml into
// the Go source the binary compiles. It runs at development time only: the
// finished monmux binary contains no catalog parser and reads no catalog file.
//
// It is deliberately a leaf package. It does not import internal/catalog, which
// contains the file this package writes; if it did, deleting or corrupting the
// generated file would stop the generator compiling and there would be no way
// back. The table of accepted mechanisms below is therefore its own, and a test
// in internal/catalog checks it against the enum it mirrors. Input names are not
// mirrored: their grammar lives in internal/catalog/internal/input, which is a
// leaf too, so both sides can import the one copy.
package generate

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/leinardi/monmux/internal/catalog/internal/input"
)

// Errors reported by [Parse] and [Validate]. Every rule has its own sentinel so
// a caller, and a test, can tell which one an entry broke.
var (
	// ErrMalformed reports YAML that does not decode into the schema, including
	// an unknown key and an out-of-range value.
	ErrMalformed = errors.New("generate: the catalog file is malformed")
	// ErrEmptyCatalog reports a document that records no models at all.
	ErrEmptyCatalog = errors.New("generate: the catalog records no models")
	// ErrBlank reports a field that is empty or only whitespace.
	ErrBlank = errors.New("generate: blank field")
	// ErrMissing reports a numeric field the file never wrote. It is its own
	// rule because the zero value of a byte is a byte: a product code or a value
	// left out would otherwise decode as 0x00 and be written to a monitor,
	// which is exactly the guess the catalog exists to prevent.
	ErrMissing = errors.New("generate: missing value")
	// ErrDuplicateModel reports two entries with the same model name.
	ErrDuplicateModel = errors.New("generate: duplicate model name")
	// ErrOutOfOrder reports two neighboring entries the file writes the wrong
	// way round. The catalog is rendered in file order, so the file order is the
	// order of the generated table and of the documentation; fixing it here is
	// what keeps a new entry landing next to its family instead of at the end.
	ErrOutOfOrder = errors.New("generate: models are out of catalog order")
	// ErrDuplicateIdentity reports one EDID fingerprint claimed by two models,
	// which would make every match ambiguous.
	ErrDuplicateIdentity = errors.New("generate: duplicate identity")
	// ErrManufacturer reports a PNP ID that is not three uppercase letters.
	ErrManufacturer = errors.New("generate: manufacturer must be three uppercase letters")
	// ErrModelName reports a pinned model name that could not have come out of
	// an EDID descriptor: too long, untrimmed, or holding something other than
	// printable ASCII.
	ErrModelName = errors.New("generate: bad pinned model name")
	// ErrIdentityRequired reports a write-enabled model with no identity. It
	// could never match, so it must not claim to be enabled.
	ErrIdentityRequired = errors.New("generate: a write-enabled model needs at least one identity")
	// ErrIdentityForbidden reports a model that is not write-enabled but records
	// an identity. Matching does not consult the flag, so such an entry would
	// match a real display and then refuse late instead of never matching.
	ErrIdentityForbidden = errors.New(
		"generate: a model that is not write-enabled must record no identity",
	)
	// ErrNoInputs reports a model that documents no input.
	ErrNoInputs = errors.New("generate: the model records no input")
	// ErrNoSources reports a model with no reference backing it.
	ErrNoSources = errors.New("generate: the model records no source")
	// ErrUnknownInput reports an input key that is not a well-formed name: a
	// known connector kind and an optional port number.
	ErrUnknownInput = errors.New("generate: unknown input")
	// ErrMixedNumbering reports a model that writes one connector kind both bare
	// and numbered, which leaves it undecided which port the bare name means.
	ErrMixedNumbering = errors.New("generate: a connector kind is written both bare and numbered")
	// ErrUnknownMechanism reports a mechanism no backend implements.
	ErrUnknownMechanism = errors.New("generate: unknown mechanism")
	// ErrUnknownGrade reports an evidence grade outside the closed list.
	ErrUnknownGrade = errors.New("generate: unknown evidence grade")
	// ErrUnverifiedEnabled reports a write-enabled model whose evidence for some
	// input is anything but a direct test on the unit. Enabling a value nobody
	// ran on that monitor is the one mistake this catalog exists to prevent, so
	// it is a build failure rather than a review note.
	ErrUnverifiedEnabled = errors.New(
		"generate: a write-enabled model needs verified evidence on every input",
	)
	// ErrBadURL reports a reference that is not an https URL.
	ErrBadURL = errors.New("generate: a URL must start with https://")
	// ErrBadNote reports a per-model note that says nothing.
	ErrBadNote = errors.New("generate: blank note")
	// ErrBadText reports text that cannot be rendered where the documentation
	// puts it: a Markdown table cell, or a single bullet.
	ErrBadText = errors.New("generate: text holds a pipe or a line break")
)

// The mechanisms. They mirror catalog.Mechanisms().
const (
	mechanismLGAltInput  = "lg-alt-input"
	mechanismInputSource = "vcp-input-source"
)

// mechanismOrder is every mechanism a backend implements. It mirrors
// catalog.Mechanisms().
var mechanismOrder = []string{mechanismLGAltInput, mechanismInputSource}

// mechanismNames maps a mechanism name to the Go constant that names it.
var mechanismNames = map[string]string{
	mechanismLGAltInput:  "MechanismLGAltInput",
	mechanismInputSource: "MechanismInputSource",
}

// mechanismPhrases maps a mechanism name to the way the evidence sentence
// names it. It is prose, not a value: nothing here reaches a monitor.
var mechanismPhrases = map[string]string{
	mechanismLGAltInput:  "the LG side channel (source address 0x50, VCP 0xF4)",
	mechanismInputSource: "the standard Input Source feature (VCP 0x60)",
}

// The evidence grades. They mirror catalog.Grades(), which a test compares.
const (
	gradeVerified   = "verified"
	gradeDocumented = "documented"
	gradeReported   = "reported"
	gradeQuoted     = "quoted"
)

// gradeOrder is every grade, strongest first.
var gradeOrder = []string{gradeVerified, gradeDocumented, gradeReported, gradeQuoted}

// gradeNames maps a grade to the Go constant that names it.
var gradeNames = map[string]string{
	gradeVerified:   "GradeVerified",
	gradeDocumented: "GradeDocumented",
	gradeReported:   "GradeReported",
	gradeQuoted:     "GradeQuoted",
}

// The evidence fields, by the key the catalog file writes them under.
const (
	fieldBy   = "by"
	fieldTool = "tool"
	fieldDate = "date"
	fieldURL  = "url"
	fieldNote = "note"
)

// renderedFields is every evidence field the documentation writes into a table
// cell, and so every one the text rule applies to.
var renderedFields = []string{fieldBy, fieldTool, fieldDate, fieldURL, fieldNote}

// requiredEvidence is what each grade must supply for its sentence to be
// composable. A grade whose fields are missing renders a sentence with holes in
// it, which is worse than no entry, so it is rejected instead.
var requiredEvidence = map[string][]string{
	gradeVerified:   {fieldDate, fieldTool, fieldNote},
	gradeDocumented: {fieldBy, fieldURL},
	gradeReported:   {fieldBy, fieldTool, fieldURL},
	gradeQuoted:     {fieldBy, fieldTool, fieldURL},
}

// urlScheme is the only scheme a recorded reference may use.
const urlScheme = "https://"

// manufacturerLength is the length of an EDID PNP manufacturer ID.
const manufacturerLength = 3

// modelNameLength is the most an EDID descriptor can hold: 13 bytes of text.
const modelNameLength = 13

// The bounds of printable ASCII, which is all an EDID descriptor carries.
const (
	firstPrintable = 0x20
	lastPrintable  = 0x7E
)

// Mechanisms returns the mechanism names this generator accepts, in enum order.
func Mechanisms() []string {
	return slices.Clone(mechanismOrder)
}

// Grades returns the evidence grades this generator accepts, strongest first.
func Grades() []string {
	return slices.Clone(gradeOrder)
}

// claim is one entry's assertion that an EDID belongs to it, reduced to
// comparable values so that two entries claiming the same display are detected
// by what they say rather than by where the decoder happened to put it.
type claim struct {
	manufacturer string
	productCode  uint16
	modelName    string
	owner        string
}

// String renders the claim the way monmux documentation writes an identity.
func (c claim) String() string {
	rendered := fmt.Sprintf("%s/0x%04X", c.manufacturer, c.productCode)
	if c.modelName == "" {
		return rendered
	}

	return fmt.Sprintf("%s %q", rendered, c.modelName)
}

// collides reports whether two claims could both match one display.
//
// Two identities collide when the manufacturer and the product code are equal
// and either pins no model name, or both pin the same one. A bare claim next to
// a pinned one is a collision even though the strings differ: the bare one
// matches every display with that code, the pinned one included, so the display
// the pinned entry was written for would become ambiguous. Only two entries that
// both pin, with different names, may share a product code - which is the whole
// point of pinning, since one vendor code is reused across products.
func (c claim) collides(other claim) bool {
	if c.manufacturer != other.manufacturer || c.productCode != other.productCode {
		return false
	}

	return c.modelName == "" || other.modelName == "" || c.modelName == other.modelName
}

// Document is the whole catalog file.
type Document struct {
	Models []Model `yaml:"models"`
}

// Model is one catalog entry, as written in the file.
type Model struct {
	Name       string     `yaml:"name"`
	Vendor     string     `yaml:"vendor"`
	Identities []Identity `yaml:"identities"`
	//nolint:tagliatelle // the file is snake_case, like the configuration file
	WriteEnabled bool             `yaml:"write_enabled"`
	Inputs       map[string]Input `yaml:"inputs"`
	Notes        []string         `yaml:"notes"`
	Sources      []string         `yaml:"sources"`
}

// Identity is one EDID fingerprint a model is recognized by. ProductCode is a
// pointer so that a file which never wrote one is told apart from a file that
// wrote zero; [Document.Validate] rejects the first.
type Identity struct {
	Manufacturer string `yaml:"manufacturer"`
	//nolint:tagliatelle // see Model
	ProductCode *uint16 `yaml:"product_code"`
	// ModelName pins the EDID descriptor 0xFC text. It is optional and left out
	// unless two models have to share a product code, which happens because a
	// vendor reuses one. See [claim.collides].
	//nolint:tagliatelle // see Model
	ModelName string `yaml:"model_name"`
}

// Input is what the file records for one input of one model. Value is 16 bits
// wide because a SetVCP carries an SH/SL pair, and it is a pointer for the same
// reason as [Identity.ProductCode]: a value nobody wrote must not default to
// 0x0000.
type Input struct {
	Mechanism string   `yaml:"mechanism"`
	Value     *uint16  `yaml:"value"`
	Evidence  Evidence `yaml:"evidence"`
}

// Evidence is why anyone believes a value does what the entry says it does. It
// is recorded as fields rather than as a sentence, so that the strength of a
// claim is a value the generator and the tests can check, and so that every
// entry of one grade reads the same way in the generated catalog and in the
// documentation. [renderEvidence] composes the sentence.
type Evidence struct {
	Grade string `yaml:"grade"`
	By    string `yaml:"by"`
	Tool  string `yaml:"tool"`
	Date  string `yaml:"date"`
	URL   string `yaml:"url"`
	Note  string `yaml:"note"`
}

// field returns one evidence field by its file key, so the per-grade table of
// required fields can be checked without a switch per grade.
func (e *Evidence) field(name string) string {
	switch name {
	case fieldBy:
		return e.By
	case fieldTool:
		return e.Tool
	case fieldDate:
		return e.Date
	case fieldURL:
		return e.URL
	case fieldNote:
		return e.Note
	default:
		return ""
	}
}

// Generate parses, validates and renders a catalog file in one step. It is the
// only entry point the command and the drift test use. The validation happens
// inside [Render], so there is no order of calls that renders an entry the
// catalog rules reject.
func Generate(source []byte) ([]byte, error) {
	document, err := Parse(source)
	if err != nil {
		return nil, err
	}

	return Render(document)
}

// Parse decodes a catalog file. An unknown key is an error rather than
// something ignored, so a misspelled field cannot silently drop an entry.
func Parse(source []byte) (Document, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(source))
	decoder.KnownFields(true)

	var document Document

	err := decoder.Decode(&document)
	if err != nil {
		return Document{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}

	// A YAML stream can hold several documents. Decoding only the first would
	// silently ignore everything after a `---`, so a second entry list could sit
	// in the file, be reviewed, and never reach the binary. An empty trailing
	// document carries nothing and is allowed; one with content is not.
	for {
		var trailing yaml.Node

		err = decoder.Decode(&trailing)
		if errors.Is(err, io.EOF) {
			return document, nil
		}

		if err != nil {
			return Document{}, fmt.Errorf("%w: %w", ErrMalformed, err)
		}

		if !emptyDocument(&trailing) {
			return Document{}, fmt.Errorf(
				"%w: the file must hold exactly one YAML document", ErrMalformed,
			)
		}
	}
}

// Validate applies every rule the catalog must satisfy, reporting all breakages
// at once so a contributor sees the whole list rather than the first line of it.
func (d Document) Validate() error {
	if len(d.Models) == 0 {
		return ErrEmptyCatalog
	}

	var (
		problems []error
		names    = map[string]bool{}
		claimed  []claim
	)

	for index := range d.Models {
		model := &d.Models[index]

		problems = append(problems, model.validate()...)

		// Sortedness is a property of every neighboring pair, so comparing each
		// entry with the one before it reports every inversion and names the two
		// entries a contributor has to swap.
		if index > 0 {
			previous := &d.Models[index-1]
			if compareModels(previous, model) > 0 {
				problems = append(problems, fmt.Errorf(
					"%w: %s %q must come before %s %q",
					ErrOutOfOrder, model.Vendor, model.Name, previous.Vendor, previous.Name,
				))
			}
		}

		if names[model.Name] {
			problems = append(problems, fmt.Errorf("%w: %q", ErrDuplicateModel, model.Name))
		}

		names[model.Name] = true

		for _, identity := range model.Identities {
			// A missing product code is reported by validateIdentities; there is
			// nothing to compare until the file supplies one.
			if identity.ProductCode == nil {
				continue
			}

			current := claim{
				manufacturer: identity.Manufacturer,
				productCode:  *identity.ProductCode,
				modelName:    identity.ModelName,
				owner:        model.Name,
			}

			// Pairwise rather than a map lookup, because the rule is not
			// equality: a bare claim collides with a pinned one.
			for _, previous := range claimed {
				if !previous.collides(current) {
					continue
				}

				problems = append(problems, fmt.Errorf(
					"%w: %s and %s could both match one display, for %q and %q",
					ErrDuplicateIdentity,
					previous,
					current,
					previous.owner,
					current.owner,
				))
			}

			claimed = append(claimed, current)
		}
	}

	return errors.Join(problems...)
}

// compareModels is the order the catalog file is written in: the write-enabled
// models first, then by vendor, then by model name. Enabled first because the
// first question the compatibility table answers is which monitors monmux will
// actually write to; vendor and name after it because a contributor looking for
// a model, or for the place to add one, looks for its family.
func compareModels(first, second *Model) int {
	// Reversed, because a write-enabled entry sorts before one that is not.
	if order := cmp.Compare(rank(second.WriteEnabled), rank(first.WriteEnabled)); order != 0 {
		return order
	}

	if order := compareText(first.Vendor, second.Vendor); order != 0 {
		return order
	}

	return compareText(first.Name, second.Name)
}

// rank turns a flag into something [cmp.Compare] can order.
func rank(flag bool) int {
	if flag {
		return 1
	}

	return 0
}

// compareText orders two catalog fields the way a reader scanning the file
// would: case-insensitively, since `LC49G95T` and `Dark Matter 40776` sit in one
// list and a capital letter is not a section break. Equal-but-for-case strings
// fall back to the byte order so that the result is a total order and the check
// cannot depend on which of two entries the file wrote first.
func compareText(first, second string) int {
	if folded := cmp.Compare(strings.ToLower(first), strings.ToLower(second)); folded != 0 {
		return folded
	}

	return cmp.Compare(first, second)
}

// validate applies the rules that concern one entry on its own.
func (m *Model) validate() []error {
	var problems []error

	if strings.TrimSpace(m.Name) == "" {
		problems = append(problems, fmt.Errorf("%w: a model has a blank name", ErrBlank))
	}

	if strings.TrimSpace(m.Vendor) == "" {
		problems = append(problems, fmt.Errorf("%w: %q has a blank vendor", ErrBlank, m.Name))
	}

	// The two together are a table cell and a Markdown heading in the generated
	// documentation, so a pipe would shift every column of that row and a line
	// break would split the heading in half.
	if !renderable(m.Name) || !renderable(m.Vendor) {
		problems = append(problems, fmt.Errorf(
			"%w: %q has an unrenderable name or vendor", ErrBadText, m.Name,
		))
	}

	problems = append(problems, m.validateIdentities()...)
	problems = append(problems, m.validateInputs()...)
	problems = append(problems, m.validateNotes()...)
	problems = append(problems, m.validateSources()...)

	return problems
}

// validateIdentities checks the fingerprints, including the two rules that tie
// them to WriteEnabled.
func (m *Model) validateIdentities() []error {
	var problems []error

	if m.WriteEnabled && len(m.Identities) == 0 {
		problems = append(problems, fmt.Errorf("%w: %q", ErrIdentityRequired, m.Name))
	}

	if !m.WriteEnabled && len(m.Identities) != 0 {
		problems = append(problems, fmt.Errorf("%w: %q", ErrIdentityForbidden, m.Name))
	}

	for _, identity := range m.Identities {
		if !validManufacturer(identity.Manufacturer) {
			problems = append(problems, fmt.Errorf(
				"%w: %q has %q", ErrManufacturer, m.Name, identity.Manufacturer,
			))
		}

		if identity.ProductCode == nil {
			problems = append(problems, fmt.Errorf(
				"%w: %q has an identity with no product code", ErrMissing, m.Name,
			))
		}

		if !validModelName(identity.ModelName) {
			problems = append(problems, fmt.Errorf(
				"%w: %q pins %q", ErrModelName, m.Name, identity.ModelName,
			))
		}

		// Printable ASCII still admits the cell separator, and the Identities
		// column of the document is a table cell like any other.
		if !renderable(identity.ModelName) {
			problems = append(problems, fmt.Errorf(
				"%w: %q pins an unrenderable model name", ErrBadText, m.Name,
			))
		}
	}

	return problems
}

// validateInputs checks that every recorded input is a well-formed name switched
// by a mechanism a backend implements, with its own evidence, and that no
// connector kind is written both bare and numbered.
func (m *Model) validateInputs() []error {
	var problems []error

	if len(m.Inputs) == 0 {
		problems = append(problems, fmt.Errorf("%w: %q", ErrNoInputs, m.Name))
	}

	parsed := make([]input.Name, 0, len(m.Inputs))

	for _, name := range slices.Sorted(maps.Keys(m.Inputs)) {
		entry := m.Inputs[name]

		candidate, err := input.Parse(name)
		if err != nil {
			problems = append(
				problems,
				fmt.Errorf("%w: %q records %q: %w", ErrUnknownInput, m.Name, name, err),
			)
		} else {
			parsed = append(parsed, candidate)
		}

		if !slices.Contains(mechanismOrder, entry.Mechanism) {
			problems = append(problems, fmt.Errorf(
				"%w: %q records %q for %q", ErrUnknownMechanism, m.Name, entry.Mechanism, name,
			))
		}

		if entry.Value == nil {
			problems = append(
				problems,
				fmt.Errorf("%w: %q records %q with no value", ErrMissing, m.Name, name),
			)
		}

		problems = append(problems, m.validateEvidence(name, &entry)...)
	}

	err := input.CheckNumbering(parsed)
	if err != nil {
		problems = append(problems, fmt.Errorf("%w: %q: %w", ErrMixedNumbering, m.Name, err))
	}

	return problems
}

// validateEvidence checks one input's evidence: a grade from the closed list,
// every field that grade's sentence needs, and - for a write-enabled model - the
// rule that only a direct test on the unit may become a write.
func (m *Model) validateEvidence(name string, entry *Input) []error {
	var problems []error

	evidence := &entry.Evidence

	required, known := requiredEvidence[evidence.Grade]
	if !known {
		return append(problems, fmt.Errorf(
			"%w: %q records %q with grade %q", ErrUnknownGrade, m.Name, name, evidence.Grade,
		))
	}

	if m.WriteEnabled && evidence.Grade != gradeVerified {
		problems = append(problems, fmt.Errorf(
			"%w: %q enables %q on %s evidence",
			ErrUnverifiedEnabled, m.Name, name, evidence.Grade,
		))
	}

	for _, field := range required {
		if strings.TrimSpace(evidence.field(field)) == "" {
			problems = append(problems, fmt.Errorf(
				"%w: %q records %q with grade %q and no %s",
				ErrBlank, m.Name, name, evidence.Grade, field,
			))
		}
	}

	if evidence.URL != "" && !strings.HasPrefix(evidence.URL, urlScheme) {
		problems = append(problems, fmt.Errorf(
			"%w: %q records %q with %q", ErrBadURL, m.Name, name, evidence.URL,
		))
	}

	for _, field := range renderedFields {
		if renderable(evidence.field(field)) {
			continue
		}

		problems = append(problems, fmt.Errorf(
			"%w: %q records %q with an unrenderable %s", ErrBadText, m.Name, name, field,
		))
	}

	return problems
}

// validateNotes checks the per-model prose. A note is documentation, never an
// operation, but it is rendered as one Markdown bullet, so it must fit on one.
func (m *Model) validateNotes() []error {
	var problems []error

	for _, note := range m.Notes {
		if strings.TrimSpace(note) == "" {
			problems = append(problems, fmt.Errorf("%w: %q has a blank note", ErrBadNote, m.Name))

			continue
		}

		if !renderable(note) {
			problems = append(problems, fmt.Errorf(
				"%w: %q has an unrenderable note", ErrBadText, m.Name,
			))
		}
	}

	return problems
}

// unrenderable is what may not appear in text the documentation puts in a table
// cell or in one bullet: the cell separator, and any line break.
const unrenderable = "|\n\r"

// renderable reports whether text survives being written into the document.
func renderable(text string) bool {
	return !strings.ContainsAny(text, unrenderable)
}

// validateSources checks that the entry says where its values came from.
func (m *Model) validateSources() []error {
	var problems []error

	if len(m.Sources) == 0 {
		problems = append(problems, fmt.Errorf("%w: %q", ErrNoSources, m.Name))
	}

	for _, source := range m.Sources {
		if strings.TrimSpace(source) == "" {
			problems = append(problems, fmt.Errorf("%w: %q has a blank source", ErrBlank, m.Name))

			continue
		}

		if !renderable(source) {
			problems = append(problems, fmt.Errorf(
				"%w: %q has an unrenderable source", ErrBadText, m.Name,
			))
		}
	}

	return problems
}

// nullTag is the YAML tag of a document, or a node, that holds nothing.
const nullTag = "!!null"

// emptyDocument reports whether a decoded document carries nothing. A file may
// open or close with a `---` marker, which yields a document whose only child is
// null. That is punctuation, not a second catalog, so it is allowed through.
func emptyDocument(node *yaml.Node) bool {
	if len(node.Content) == 0 {
		return true
	}

	return len(node.Content) == 1 && node.Content[0].Tag == nullTag
}

// validModelName reports whether a pinned name could have come out of an EDID
// descriptor. An empty name is not pinned at all and is always fine.
func validModelName(name string) bool {
	if name == "" {
		return true
	}

	if name != strings.TrimSpace(name) || len(name) > modelNameLength {
		return false
	}

	for index := range len(name) {
		if name[index] < firstPrintable || name[index] > lastPrintable {
			return false
		}
	}

	return true
}

// validManufacturer reports whether a PNP ID is three uppercase ASCII letters.
func validManufacturer(code string) bool {
	if len(code) != manufacturerLength {
		return false
	}

	for _, letter := range code {
		if letter < 'A' || letter > 'Z' {
			return false
		}
	}

	return true
}
