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

package generate_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/catalog/internal/generate"
)

// The markers, repeated here rather than exported, so a test fails if somebody
// changes the strings the committed document already carries.
const (
	begin = "<!-- BEGIN GENERATED: edit internal/catalog/models.yaml, then run make go-generate -->"
	end   = "<!-- END GENERATED -->"
)

// oneModel is the smallest catalog that renders: one disabled model, one input,
// one note, one source.
const oneModel = `models:
  - name: TESTER-1
    vendor: Acme
    identities: []
    write_enabled: false
    inputs:
      hdmi2:
        mechanism: vcp-input-source
        value: 0x11
        evidence:
          grade: reported
          by: "somebody"
          tool: "ddcutil"
          url: https://example.invalid/report
    notes:
      - "A note that links [testing.md](testing.md) and nothing else."
    sources:
      - https://example.invalid/source
`

// surround wraps a rendered region in the prose a real document has around it,
// including prose that looks like the things the splice looks for.
func surround(body string) string {
	return strings.Join([]string{
		"# Compatibility",
		"",
		"## What the columns mean",
		"",
		"- **Model** — a | pipe | in hand-written prose.",
		"",
		begin,
		body,
		end,
		"",
		"## What is deliberately not here",
		"",
		"Prose the generator must not touch.",
		"",
	}, "\n")
}

// render is the whole pipeline, for a test that only cares about the output.
func render(t *testing.T, source, existing string) string {
	t.Helper()

	spliced, err := generate.GenerateDocument([]byte(source), []byte(existing))
	if err != nil {
		t.Fatalf("GenerateDocument: %v", err)
	}

	return string(spliced)
}

// Everything outside the markers is the human's, and is copied through byte for
// byte - including prose that holds a pipe or a heading of its own.
func TestGenerateDocumentKeepsEveryByteOutsideTheMarkers(t *testing.T) {
	t.Parallel()

	existing := surround("stale content the generator replaces")
	spliced := render(t, oneModel, existing)

	before, _, found := strings.Cut(existing, begin)
	if !found {
		t.Fatal("the fixture lost its begin marker")
	}

	if !strings.HasPrefix(spliced, before+begin) {
		t.Errorf("the prose above the region changed:\n%s", spliced)
	}

	_, after, found := strings.Cut(existing, end)
	if !found {
		t.Fatal("the fixture lost its end marker")
	}

	if !strings.HasSuffix(spliced, end+after) {
		t.Errorf("the prose below the region changed:\n%s", spliced)
	}

	if strings.Contains(spliced, "stale content") {
		t.Error("the old region survived the splice")
	}
}

// Running the generator twice must produce the same file, or every commit after
// the first would carry a diff nobody asked for.
func TestGenerateDocumentIsIdempotent(t *testing.T) {
	t.Parallel()

	once := render(t, oneModel, surround("stale"))

	twice := render(t, oneModel, once)
	if once != twice {
		t.Errorf("a second run changed the file:\n%s\n---\n%s", once, twice)
	}
}

// A bare URL is wrapped the way markdownlint's fix would wrap it, so the
// generated file is a fixed point of the hooks that run after the generator. A
// URL that is already a Markdown link is not a bare URL and stays as it is.
func TestGenerateDocumentAutolinksBareURLsOnly(t *testing.T) {
	t.Parallel()

	spliced := render(t, oneModel, surround(""))

	for _, want := range []string{
		"- <https://example.invalid/source>",
		"not verified here: <https://example.invalid/report>",
	} {
		if !strings.Contains(spliced, want) {
			t.Errorf("no autolink for %q in:\n%s", want, spliced)
		}
	}

	if !strings.Contains(spliced, "links [testing.md](testing.md) and") {
		t.Errorf("a Markdown link was rewritten:\n%s", spliced)
	}
}

// A source that is a URL followed by prose autolinks the URL and leaves the rest
// of the sentence alone, which is what the catalog already records for one entry.
func TestGenerateDocumentAutolinksAURLFollowedByProse(t *testing.T) {
	t.Parallel()

	source := strings.Replace(
		oneModel,
		"      - https://example.invalid/source\n",
		"      - \"https://example.invalid/source - what it says\"\n",
		1,
	)

	spliced := render(t, source, surround(""))

	want := "- <https://example.invalid/source> - what it says"
	if !strings.Contains(spliced, want) {
		t.Errorf("want %q in:\n%s", want, spliced)
	}
}

// Models keep the order of the catalog file - here vendor order, which puts
// SECOND before FIRST - and inputs are sorted by kind and then by port, so the
// table order is decided by the grammar rather than by Go's map iteration.
func TestRenderDocumentOrdersModelsAndInputs(t *testing.T) {
	t.Parallel()

	source := `models:
  - name: SECOND
    vendor: Acme
    identities: []
    write_enabled: false
    inputs:
      usb-c:
        mechanism: lg-alt-input
        value: 0xD1
        evidence: {grade: reported, by: "x", tool: "y", url: https://example.invalid/a}
      hdmi2:
        mechanism: lg-alt-input
        value: 0x91
        evidence: {grade: reported, by: "x", tool: "y", url: https://example.invalid/a}
      dp:
        mechanism: lg-alt-input
        value: 0xD0
        evidence: {grade: reported, by: "x", tool: "y", url: https://example.invalid/a}
      hdmi1:
        mechanism: lg-alt-input
        value: 0x90
        evidence: {grade: reported, by: "x", tool: "y", url: https://example.invalid/a}
    notes: ["second"]
    sources: [https://example.invalid/a]
  - name: FIRST
    vendor: Bravo
    identities: []
    write_enabled: false
    inputs:
      dp:
        mechanism: lg-alt-input
        value: 0xD0
        evidence: {grade: reported, by: "x", tool: "y", url: https://example.invalid/a}
    notes: ["first"]
    sources: [https://example.invalid/a]
`

	document, err := generate.Parse([]byte(source))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	lines, err := generate.RenderDocument(document)
	if err != nil {
		t.Fatalf("RenderDocument: %v", err)
	}

	var inputs []string

	for _, line := range lines {
		if !strings.HasPrefix(line, "| Acme SECOND |") {
			continue
		}

		inputs = append(inputs, strings.Split(line, " | ")[2])
	}

	want := "`dp` `hdmi1` `hdmi2` `usb-c`"
	if strings.Join(inputs, " ") != want {
		t.Errorf("inputs are in order %v, want %s", inputs, want)
	}

	second := indexOfPrefix(lines, "### Acme SECOND")
	first := indexOfPrefix(lines, "### Bravo FIRST")

	if second < 0 || first < 0 || second > first {
		t.Errorf("the notes sections are in the wrong order: %d and %d", second, first)
	}
}

// indexOfPrefix reports where a line starting with a prefix is, or -1.
func indexOfPrefix(lines []string, prefix string) int {
	for index, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return index
		}
	}

	return -1
}

// A model with no fingerprint says so in a word. A blank cell would read like an
// omission rather than like an entry that can never match a display.
func TestRenderDocumentNamesTheAbsenceOfAnIdentity(t *testing.T) {
	t.Parallel()

	spliced := render(t, oneModel, surround(""))

	if !strings.Contains(spliced, "| Acme TESTER-1 | none | `hdmi2` | `0x11` |") {
		t.Errorf("no row with an explicit absent identity in:\n%s", spliced)
	}
}

// A document the splice cannot read is left alone: no output, an error naming
// the marker. Appending, or guessing at the region, would either duplicate the
// catalog or eat the prose around it.
func TestGenerateDocumentRefusesADocumentItCannotSplice(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"no begin marker":  "# Doc\n\n" + end + "\n",
		"no end marker":    "# Doc\n\n" + begin + "\n",
		"neither marker":   "# Doc\n\nnothing to splice into\n",
		"two begin":        begin + "\nbody\n" + begin + "\n" + end + "\n",
		"two end":          begin + "\nbody\n" + end + "\n" + end + "\n",
		"reversed markers": end + "\nbody\n" + begin + "\n",
	}

	for name, existing := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			spliced, err := generate.GenerateDocument([]byte(oneModel), []byte(existing))
			if !errors.Is(err, generate.ErrMarkers) {
				t.Fatalf("error is %v, want ErrMarkers", err)
			}

			if spliced != nil {
				t.Errorf("a refused splice returned %d bytes", len(spliced))
			}
		})
	}
}

// Catalog text that carries a marker would move the boundary of the generated
// region, and the next run would splice over the prose outside it. Every field
// that reaches the document is covered by the one check, so this walks them.
func TestRenderDocumentRefusesAMarkerInCatalogText(t *testing.T) {
	t.Parallel()

	cases := map[string]struct{ from, to string }{
		"a note":       {"A note that links", end + " A note that links"},
		"a source":     {"https://example.invalid/source", "see " + end},
		"a model name": {"name: TESTER-1", "name: " + end},
		"a vendor":     {"vendor: Acme", "vendor: " + end},
		"evidence":     {`by: "somebody"`, `by: "` + begin + `"`},
	}

	for name, replacement := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			source := strings.Replace(oneModel, replacement.from, replacement.to, 1)
			if source == oneModel {
				t.Fatalf("the fixture does not contain %q", replacement.from)
			}

			_, err := generate.GenerateDocument([]byte(source), []byte(surround("")))
			if !errors.Is(err, generate.ErrMarkerInContent) {
				t.Fatalf("error is %v, want ErrMarkerInContent", err)
			}
		})
	}
}

// A catalog the rules reject is never rendered, so a document is never written
// from one. This is the same order of checks [Render] makes, for the same reason.
func TestRenderDocumentValidatesBeforeItRenders(t *testing.T) {
	t.Parallel()

	source := strings.Replace(oneModel, "        value: 0x11\n", "", 1)

	_, err := generate.GenerateDocument([]byte(source), []byte(surround("")))
	if !errors.Is(err, generate.ErrMissing) {
		t.Fatalf("error is %v, want ErrMissing", err)
	}
}
