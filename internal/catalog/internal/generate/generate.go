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
	// ErrDuplicateIdentity reports one EDID fingerprint claimed by two models,
	// which would make every match ambiguous.
	ErrDuplicateIdentity = errors.New("generate: duplicate identity")
	// ErrManufacturer reports a PNP ID that is not three uppercase letters.
	ErrManufacturer = errors.New("generate: manufacturer must be three uppercase letters")
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
)

// mechanismOrder is every mechanism a backend implements. It mirrors
// catalog.Mechanisms().
var mechanismOrder = []string{"lg-alt-input"}

// mechanismNames maps a mechanism name to the Go constant that names it.
var mechanismNames = map[string]string{
	"lg-alt-input": "MechanismLGAltInput",
}

// manufacturerLength is the length of an EDID PNP manufacturer ID.
const manufacturerLength = 3

// Mechanisms returns the mechanism names this generator accepts, in enum order.
func Mechanisms() []string {
	return slices.Clone(mechanismOrder)
}

// fingerprint is an [Identity] reduced to comparable values, so that two entries
// claiming the same EDID are detected by what they say rather than by where the
// decoder happened to put it.
type fingerprint struct {
	manufacturer string
	productCode  uint16
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
	Sources      []string         `yaml:"sources"`
}

// Identity is one EDID fingerprint a model is recognized by. ProductCode is a
// pointer so that a file which never wrote one is told apart from a file that
// wrote zero; [Document.Validate] rejects the first.
type Identity struct {
	Manufacturer string `yaml:"manufacturer"`
	//nolint:tagliatelle // see Model
	ProductCode *uint16 `yaml:"product_code"`
}

// Input is what the file records for one input of one model. Value is a pointer
// for the same reason as [Identity.ProductCode]: a byte nobody wrote must not
// default to 0x00.
type Input struct {
	Mechanism string `yaml:"mechanism"`
	Value     *uint8 `yaml:"value"`
	Evidence  string `yaml:"evidence"`
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
		owners   = map[fingerprint]string{}
	)

	for index := range d.Models {
		model := &d.Models[index]

		problems = append(problems, model.validate()...)

		if names[model.Name] {
			problems = append(problems, fmt.Errorf("%w: %q", ErrDuplicateModel, model.Name))
		}

		names[model.Name] = true

		for _, identity := range model.Identities {
			// A missing product code is reported by validateIdentities; there is
			// no fingerprint to compare until the file supplies one.
			if identity.ProductCode == nil {
				continue
			}

			current := fingerprint{
				manufacturer: identity.Manufacturer,
				productCode:  *identity.ProductCode,
			}

			previous, taken := owners[current]
			if taken {
				problems = append(problems, fmt.Errorf(
					"%w: %s/0x%04X is claimed by both %q and %q",
					ErrDuplicateIdentity,
					current.manufacturer,
					current.productCode,
					previous,
					model.Name,
				))
			}

			owners[current] = model.Name
		}
	}

	return errors.Join(problems...)
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

	problems = append(problems, m.validateIdentities()...)
	problems = append(problems, m.validateInputs()...)
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

		if strings.TrimSpace(entry.Evidence) == "" {
			problems = append(
				problems,
				fmt.Errorf("%w: %q records %q with no evidence", ErrBlank, m.Name, name),
			)
		}
	}

	err := input.CheckNumbering(parsed)
	if err != nil {
		problems = append(problems, fmt.Errorf("%w: %q: %w", ErrMixedNumbering, m.Name, err))
	}

	return problems
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
