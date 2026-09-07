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
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/catalog"
)

// document is the human-readable copy of the catalog.
const document = "docs/compatibility.md"

// columns is how many columns a catalog row has:
// model, identities, input, value, mechanism, enabled, evidence.
const columns = 7

// noIdentities is what the document writes for a model with no EDID fingerprint.
const noIdentities = "none"

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
	evidence   string
}

// key identifies the row for one input of one model.
func (r *row) key() string {
	return r.model + "/" + r.input
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

	contents, err := os.ReadFile(filepath.Join(moduleRoot(t), document))
	if err != nil {
		t.Fatalf("reading %s: %v", document, err)
	}

	rows := map[string]row{}

	for line := range strings.SplitSeq(string(contents), "\n") {
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
			evidence:   cells[6],
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

// split cuts one table line into its cells.
func split(line string) []string {
	fields := strings.Split(strings.Trim(line, "|"), "|")

	cells := make([]string, 0, len(fields))
	for _, field := range fields {
		cells = append(cells, autolinked.ReplaceAllString(strings.TrimSpace(field), "$1"))
	}

	return cells
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
