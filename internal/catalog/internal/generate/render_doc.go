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

package generate

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Errors reported by [RenderDocument] and [GenerateDocument].
var (
	// ErrMarkers reports a document whose generated region cannot be located:
	// a marker missing, repeated, or in the wrong order. Splicing anyway would
	// either drop hand-written prose or append a second copy of the catalog, so
	// nothing is written at all.
	ErrMarkers = errors.New("generate: the document does not delimit one generated region")
	// ErrMarkerInContent reports catalog text that holds one of the markers.
	// Writing it out would move the boundary, and the next run would splice the
	// wrong region, so it is refused before it reaches the file.
	ErrMarkerInContent = errors.New("generate: catalog text holds a section marker")
)

// The comments that delimit the generated region of the document. Everything
// between them is rendered from models.yaml; everything outside is prose a human
// writes, and this package copies it through byte for byte.
const (
	beginMarker = "<!-- BEGIN GENERATED: edit internal/catalog/models.yaml, then run make go-generate -->"
	endMarker   = "<!-- END GENERATED -->"
)

// The headings of the two generated sections.
const (
	catalogHeading = "## Catalog"
	notesHeading   = "## Notes on the entries"
)

// The table, as Markdown. The column meanings are explained in the hand-written
// part of the document, above the generated region.
const (
	tableHeader    = "| Model | Identities | Input | Value | Mechanism | Enabled | Grade | Evidence |"
	tableSeparator = "| --- | --- | --- | --- | --- | --- | --- | --- |"
)

// sourcesHeading introduces a model's references, and is the line the test that
// reads the document back splits a section on.
const sourcesHeading = "Sources:"

// noIdentities is what the Identities column says for an entry that records no
// EDID fingerprint, and so can never match a display.
const noIdentities = "none"

// The schemes a bare URL in the catalog may use. Markdown renders such a URL as
// text, so the document wraps it in an autolink; see [autolink].
var urlSchemes = []string{"https://", "http://"}

// GenerateDocument parses a catalog file, renders the two generated sections,
// and splices them into the document, returning the whole new file. Everything
// outside the markers is returned unchanged, so the intro, the column glossary
// and the closing sections are the human's to write.
//
// It returns an error and no output rather than a partial document: a splice
// that cannot find its region must leave the file alone.
func GenerateDocument(source, existing []byte) ([]byte, error) {
	document, err := Parse(source)
	if err != nil {
		return nil, err
	}

	section, err := RenderDocument(document)
	if err != nil {
		return nil, err
	}

	return splice(existing, section)
}

// RenderDocument renders the generated region of docs/compatibility.md: the
// catalog table, then one notes-and-sources section per model. The lines it
// returns go between the markers, exclusive of both.
//
// It validates first, for the same reason [Render] does: the renderers read the
// product code and the value through pointers, and [Document.Validate] is what
// guarantees those are not nil.
func RenderDocument(document Document) ([]string, error) {
	err := document.Validate()
	if err != nil {
		return nil, err
	}

	table, err := renderTable(document)
	if err != nil {
		return nil, err
	}

	// A leading blank keeps a blank line between the begin marker and the first
	// heading, which is what Markdown wants around a heading.
	lines := slices.Concat([]string{""}, table, renderNotes(document))

	err = checkMarkers(lines)
	if err != nil {
		return nil, err
	}

	return lines, nil
}

// renderTable renders the catalog table: one row per model per recorded input,
// models in file order and inputs in catalog order.
func renderTable(document Document) ([]string, error) {
	lines := []string{catalogHeading, "", tableHeader, tableSeparator}

	for index := range document.Models {
		model := &document.Models[index]

		rows, err := renderRows(model)
		if err != nil {
			return nil, err
		}

		lines = append(lines, rows...)
	}

	return append(lines, ""), nil
}

// renderRows renders one model's rows. Every cell is composed from the same
// fields the generated Go is composed from, so the table cannot describe a value
// the binary does not carry.
func renderRows(model *Model) ([]string, error) {
	names, err := sortedInputNames(model)
	if err != nil {
		return nil, err
	}

	enabled := "no"
	if model.WriteEnabled {
		enabled = "yes"
	}

	identities := renderIdentityList(model.Identities)
	rows := make([]string, 0, len(names))

	for _, name := range names {
		entry := model.Inputs[name.String()]

		evidence, evidenceErr := renderEvidence(model.Name, name, &entry)
		if evidenceErr != nil {
			return nil, evidenceErr
		}

		rows = append(rows, fmt.Sprintf(
			"| %s | %s | `%s` | `0x%02X` | `%s` | %s | %s | %s |",
			model.FullName(),
			identities,
			name,
			*entry.Value,
			entry.Mechanism,
			enabled,
			entry.Evidence.Grade,
			autolink(evidence),
		))
	}

	return rows, nil
}

// renderIdentityList renders the fingerprints the way the Identities column
// lists them. An entry with none says so in a word, because a blank cell reads
// like an omission rather than like the deliberate refusal to match that it is.
func renderIdentityList(identities []Identity) string {
	if len(identities) == 0 {
		return noIdentities
	}

	rendered := make([]string, 0, len(identities))

	for _, identity := range identities {
		// The claim type already renders an identity the way monmux writes one,
		// and is what the duplicate check compares, so the column and the rule
		// cannot disagree about what an identity is called.
		rendered = append(rendered, claim{
			manufacturer: identity.Manufacturer,
			productCode:  *identity.ProductCode,
			modelName:    identity.ModelName,
		}.String())
	}

	return strings.Join(rendered, ", ")
}

// renderNotes renders one subsection per model, carrying the prose and the
// references the catalog file records. Neither is ever an operation: nothing
// here reaches a monitor.
func renderNotes(document Document) []string {
	lines := []string{notesHeading, ""}

	for index := range document.Models {
		model := &document.Models[index]

		lines = append(lines, "### "+model.FullName(), "")

		for _, note := range model.Notes {
			lines = append(lines, "- "+autolink(note))
		}

		lines = append(lines, "", sourcesHeading, "")

		for _, source := range model.Sources {
			lines = append(lines, "- "+autolink(source))
		}

		// The trailing blank separates this section from the next one, and the
		// last one from the end marker.
		lines = append(lines, "")
	}

	return lines
}

// FullName is how the documentation names a model, and how the test that reads
// the document back keys its rows.
func (m *Model) FullName() string {
	return m.Vendor + " " + m.Name
}

// autolink wraps every bare URL in a line in Markdown autolink brackets, which
// is what markdownlint's MD034 fix does. The generated document has to be a
// fixed point of the hooks that run after this generator, or the two rewrite the
// file in turn and neither commit is ever clean.
//
// It works on whitespace-separated tokens, so a URL that is already bracketed,
// or one that is the target of a `[text](url)` link, is left alone: neither
// token starts with a scheme. Splitting on a single space rather than on fields
// keeps any other run of whitespace exactly as the catalog wrote it.
func autolink(line string) string {
	tokens := strings.Split(line, " ")

	for index, token := range tokens {
		if bareURL(token) {
			tokens[index] = "<" + token + ">"
		}
	}

	return strings.Join(tokens, " ")
}

// bareURL reports whether a token is a URL written as plain text.
func bareURL(token string) bool {
	for _, scheme := range urlSchemes {
		if strings.HasPrefix(token, scheme) {
			return true
		}
	}

	return false
}

// checkMarkers refuses rendered text that carries a marker. A note or a source
// holding one would move the boundary of the generated region, and the next run
// would splice over the prose outside it.
func checkMarkers(lines []string) error {
	for _, line := range lines {
		if strings.Contains(line, beginMarker) || strings.Contains(line, endMarker) {
			return fmt.Errorf("%w: %q", ErrMarkerInContent, line)
		}
	}

	return nil
}

// splice replaces the lines between the markers with the rendered section,
// keeping both markers and every byte outside them.
func splice(existing []byte, section []string) ([]byte, error) {
	lines := strings.Split(string(existing), "\n")

	begin, err := locate(lines, beginMarker)
	if err != nil {
		return nil, err
	}

	end, err := locate(lines, endMarker)
	if err != nil {
		return nil, err
	}

	if end < begin {
		return nil, fmt.Errorf("%w: the end marker comes before the begin marker", ErrMarkers)
	}

	spliced := slices.Concat(lines[:begin+1], section, lines[end:])

	return []byte(strings.Join(spliced, "\n")), nil
}

// locate finds the one line that is a marker. Neither zero nor two is a document
// this package will write to: with none there is no region, and with two there is
// no way to tell which pair a human meant.
func locate(lines []string, marker string) (int, error) {
	found := -1

	for index, line := range lines {
		if strings.TrimSpace(line) != marker {
			continue
		}

		if found >= 0 {
			return 0, fmt.Errorf("%w: %q appears more than once", ErrMarkers, marker)
		}

		found = index
	}

	if found < 0 {
		return 0, fmt.Errorf("%w: %q is missing", ErrMarkers, marker)
	}

	return found, nil
}
