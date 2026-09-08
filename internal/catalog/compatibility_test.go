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

package catalog_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/catalog"
)

// document is the human-readable copy of the catalog.
const document = "docs/compatibility.md"

// columns is how many columns a catalog row has:
// model, identities, input, value, mechanism, enabled, grade, evidence.
const columns = 8

// noIdentities is what the document writes for a model with no EDID fingerprint.
const noIdentities = "none"

// notesHeading opens the part of the document that carries the per-model prose,
// and sourcesLine separates a section's notes from its references.
const (
	notesHeading = "## Notes on the entries"
	sourcesLine  = "Sources:"
)

// autolinked matches a bare URL that markdownlint wrapped in angle brackets.
// That is presentation - the evidence string in the catalog has no brackets - so
// it is undone before comparing, rather than written into the catalog to please
// a formatter.
var autolinked = regexp.MustCompile(`<(https?://[^>]+)>`)

// row is one line of the document's catalog table.
type row struct {
	model      string
	identities string
	input      string
	value      string
	mechanism  string
	enabled    string
	grade      string
	evidence   string
}

// key identifies the row for one input of one model.
func (r *row) key() string {
	return r.model + "/" + r.input
}

// section is what the document says about one model outside the table.
type section struct {
	notes   []string
	sources []string
}

// A user deciding whether to run this on their monitor reads the document; the
// binary reads the catalog. If they disagree, the document is a lie about what
// the program will do to somebody's hardware - so they are compared here rather
// than trusted to be kept in step by hand.
func TestCompatibilityDocumentMatchesTheCatalog(t *testing.T) {
	t.Parallel()

	documented := parse(t)
	recorded := map[string]row{}

	for _, model := range catalog.Models() {
		for _, input := range model.RecordedInputs() {
			expected := describe(t, model, input)
			recorded[expected.key()] = expected

			actual, listed := documented[expected.key()]
			if !listed {
				t.Errorf("%s does not document %s", document, expected.key())

				continue
			}

			if actual != expected {
				t.Errorf(
					"%s documents %s as\n%+v\nbut the catalog records\n%+v",
					document,
					expected.key(),
					actual,
					expected,
				)
			}
		}
	}

	for key := range documented {
		_, exists := recorded[key]
		if !exists {
			t.Errorf("%s documents %s, which is not in the catalog", document, key)
		}
	}

	if len(recorded) == 0 {
		t.Fatal("the catalog records nothing, so this comparison proves nothing")
	}
}

// The notes are where a negative report, a conflict or a quirk is written down,
// and the table has no column for any of them. They are catalog data like the
// values are, so the document has to carry them verbatim rather than carry a
// paraphrase that ages.
func TestCompatibilityDocumentCarriesEveryNoteAndSource(t *testing.T) {
	t.Parallel()

	documented := parseSections(t)
	seen := map[string]bool{}

	for _, model := range catalog.Models() {
		name := model.FullName()
		seen[name] = true

		written, listed := documented[name]
		if !listed {
			t.Errorf("%s has no `### %s` section", document, name)

			continue
		}

		if !slices.Equal(written.notes, model.Notes) {
			t.Errorf(
				"%s notes %s as\n%q\nbut the catalog records\n%q",
				document,
				name,
				written.notes,
				model.Notes,
			)
		}

		if !slices.Equal(written.sources, model.Sources) {
			t.Errorf(
				"%s sources %s as\n%q\nbut the catalog records\n%q",
				document,
				name,
				written.sources,
				model.Sources,
			)
		}
	}

	for name := range documented {
		if !seen[name] {
			t.Errorf("%s has a `### %s` section, which is not in the catalog", document, name)
		}
	}
}

// describe renders what the document must say about one input of one model.
//
//nolint:gocritic // hugeParam: a Model is passed by value everywhere in this package
func describe(t *testing.T, model catalog.Model, input catalog.Input) row {
	t.Helper()

	value, recorded := model.InputValue(input)
	if !recorded {
		t.Fatalf("%s records %s with no value", model.FullName(), input)
	}

	mechanism, recorded := model.InputMechanism(input)
	if !recorded {
		t.Fatalf("%s records %s with no mechanism", model.FullName(), input)
	}

	grade, recorded := model.InputGrade(input)
	if !recorded {
		t.Fatalf("%s records %s with no grade", model.FullName(), input)
	}

	evidence, recorded := model.InputEvidence(input)
	if !recorded {
		t.Fatalf("%s records %s with no evidence", model.FullName(), input)
	}

	enabled := "no"
	if model.WriteEnabled {
		enabled = "yes"
	}

	return row{
		model:      model.FullName(),
		identities: identities(model),
		input:      "`" + input.String() + "`",
		value:      fmt.Sprintf("`0x%02X`", value),
		mechanism:  "`" + mechanism.String() + "`",
		enabled:    enabled,
		grade:      grade.String(),
		evidence:   evidence,
	}
}

// identities renders a model's EDID fingerprints the way the document lists them.
//
//nolint:gocritic // hugeParam: see describe
func identities(model catalog.Model) string {
	if len(model.Identities) == 0 {
		return noIdentities
	}

	rendered := make([]string, 0, len(model.Identities))
	for _, identity := range model.Identities {
		rendered = append(rendered, identity.String())
	}

	return strings.Join(rendered, ", ")
}

// parse reads the catalog table out of the document.
func parse(t *testing.T) map[string]row {
	t.Helper()

	rows := map[string]row{}

	for line := range strings.SplitSeq(contents(t), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}

		cells := split(trimmed)
		if len(cells) != columns {
			continue
		}

		// The header and its separator are not entries.
		if cells[0] == "Model" || strings.HasPrefix(cells[0], "---") {
			continue
		}

		parsed := row{
			model:      cells[0],
			identities: cells[1],
			input:      cells[2],
			value:      cells[3],
			mechanism:  cells[4],
			enabled:    cells[5],
			grade:      cells[6],
			evidence:   cells[7],
		}

		previous, duplicated := rows[parsed.key()]
		if duplicated {
			t.Errorf("%s lists %s twice: %+v and %+v", document, parsed.key(), previous, parsed)
		}

		rows[parsed.key()] = parsed
	}

	if len(rows) == 0 {
		t.Fatalf("%s has no catalog table, so this comparison proves nothing", document)
	}

	return rows
}

// sectionParser accumulates the document's per-model prose, one section at a
// time. It is a type rather than a nest of closures so that "what a line does"
// and "what a section is" can be read separately.
type sectionParser struct {
	sections map[string]section
	name     string
	current  section
	sourced  bool
	bullet   []string
}

// flush ends the bullet being read, if any, and files it under the half of the
// section it belongs to.
func (p *sectionParser) flush() {
	if len(p.bullet) == 0 {
		return
	}

	text := autolinked.ReplaceAllString(strings.Join(p.bullet, " "), "$1")
	if p.sourced {
		p.current.sources = append(p.current.sources, text)
	} else {
		p.current.notes = append(p.current.notes, text)
	}

	p.bullet = nil
}

// end files the section being read, if any, and starts an empty one.
func (p *sectionParser) end(t *testing.T) {
	t.Helper()

	p.flush()

	if p.name != "" {
		_, duplicated := p.sections[p.name]
		if duplicated {
			t.Errorf("%s has two `### %s` sections", document, p.name)
		}

		p.sections[p.name] = p.current
	}

	p.name, p.current, p.sourced = "", section{}, false
}

// read takes one line of a section.
func (p *sectionParser) read(t *testing.T, trimmed string) {
	t.Helper()

	switch {
	case strings.HasPrefix(trimmed, "### "):
		p.end(t)

		p.name = strings.TrimPrefix(trimmed, "### ")
	case p.name == "":
	case trimmed == sourcesLine:
		p.flush()

		p.sourced = true
	case strings.HasPrefix(trimmed, "- "):
		p.flush()

		p.bullet = []string{strings.TrimPrefix(trimmed, "- ")}
	case trimmed == "":
		p.flush()
	case len(p.bullet) > 0:
		// A bullet may be wrapped over several lines, which are joined by a
		// single space, because that is how a long note is written by hand.
		p.bullet = append(p.bullet, trimmed)
	}
}

// parseSections reads the per-model prose out of the document. A section runs
// from its `### ` heading to the next heading of any level; the bullets before
// the "Sources:" line are its notes and the ones after it are its references.
func parseSections(t *testing.T) map[string]section {
	t.Helper()

	parser := &sectionParser{sections: map[string]section{}}
	inNotes := false

	for line := range strings.SplitSeq(contents(t), "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case trimmed == notesHeading:
			inNotes = true
		case !inNotes:
		case strings.HasPrefix(trimmed, "## "):
			parser.end(t)

			inNotes = false
		default:
			parser.read(t, trimmed)
		}
	}

	parser.end(t)

	if len(parser.sections) == 0 {
		t.Fatalf("%s has no per-model sections, so this comparison proves nothing", document)
	}

	return parser.sections
}

// split cuts one table line into its cells.
func split(line string) []string {
	fields := strings.Split(strings.Trim(line, "|"), "|")

	cells := make([]string, 0, len(fields))
	for _, field := range fields {
		cells = append(cells, autolinked.ReplaceAllString(strings.TrimSpace(field), "$1"))
	}

	return cells
}

// contents reads the document.
func contents(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(moduleRoot(t), document))
	if err != nil {
		t.Fatalf("reading %s: %v", document, err)
	}

	return string(raw)
}

// moduleRoot walks up from the test's directory to the directory holding go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()

	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("locating the working directory: %v", err)
	}

	for {
		_, err := os.Stat(filepath.Join(directory, "go.mod"))
		if err == nil {
			return directory
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("no go.mod found above the test's directory")
		}

		directory = parent
	}
}
