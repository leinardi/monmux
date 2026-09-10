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

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/catalog/internal/generate"
)

// The fixtures: the same synthetic catalog one release apart.
const (
	previousFixture = "testdata/previous.yaml"
	currentFixture  = "testdata/current.yaml"
)

// enabledEntry is one write-enabled model with two identities and one input. It
// is the starting point of every case below that takes something away again.
const enabledEntry = `models:
  - name: MX100
    vendor: Testco
    identities:
      - manufacturer: TST
        product_code: 0x1000
      - manufacturer: TST
        product_code: 0x1001
    write_enabled: true
    inputs:
      dp:
        mechanism: lg-alt-input
        value: 0xD0
        evidence:
          grade: verified
          date: "2026-01-02"
          tool: "ddcutil 2.2.5"
          note: "switched on the unit"
    notes:
      - "A note."
    sources:
      - https://example.invalid/mx100
`

// disabledEntry is the same model with the flag and one identity taken away and
// nothing added: a release that only gives something up.
const disabledEntry = `models:
  - name: MX100
    vendor: Testco
    identities:
      - manufacturer: TST
        product_code: 0x1000
    write_enabled: false
    inputs:
      dp:
        mechanism: lg-alt-input
        value: 0xD0
        evidence:
          grade: verified
          date: "2026-01-02"
          tool: "ddcutil 2.2.5"
          note: "switched on the unit"
    notes:
      - "A note."
    sources:
      - https://example.invalid/mx100
`

func TestReleaseNotesNameEveryEnablingChange(t *testing.T) {
	t.Parallel()

	rendered, status := release(t, previousFixture, currentFixture)

	if status != "enabled=true\n" {
		t.Errorf("status = %q, want enabled=true", status)
	}

	wanted := []string{
		"## Catalog",
		"**This release changes which bytes monmux can send to a monitor.**",
		"### Models newly write-enabled",
		"- `Testco/RX200` — write-enabled on `hdmi` (`0x11` via `vcp-input-source`)",
		"### Inputs newly enabled",
		"- `Testco/MX100` — `usb-c` (`0xD2` via `lg-alt-input`)",
		"### Values changed on an enabled input",
		"- `Testco/MX100` — `dp` changed from `0xD0` via `lg-alt-input` to `0xD1` via `lg-alt-input`",
		"### Identities added",
		"- `Testco/MX100` — matches `TST/0x1001` \"MX100 REV B\"",
		"- `Testco/RX200` — matches `TST/0x2000`",
		"### Models newly recorded, reachable with `--unsafe-model`",
		"- `Newco/AB10` — recorded on `hdmi2` (`0x12` via `vcp-input-source`)",
		"### Models removed from the catalog",
		"- `Otherco/ZZ900` — removed from the catalog",
		// RX200 kept its value and changed only its grade, so the move is
		// reported where it belongs and not as a changed value.
		"### Evidence, notes and sources",
		"- `Testco/RX200` — changed: the evidence for `hdmi`",
	}

	for _, line := range wanted {
		if !strings.Contains(rendered, line) {
			t.Errorf("the notes do not hold %q; they are:\n%s", line, rendered)
		}
	}
}

func TestAnUnchangedCatalogSaysSo(t *testing.T) {
	t.Parallel()

	rendered, status := release(t, currentFixture, currentFixture)

	if status != "enabled=false\n" {
		t.Errorf("status = %q, want enabled=false", status)
	}

	const wanted = "No catalog change: this release writes exactly what v0.1.0 wrote."

	if !strings.Contains(rendered, wanted) {
		t.Errorf("the notes do not hold %q; they are:\n%s", wanted, rendered)
	}
}

func TestTheFirstReleaseComparesAgainstAnEmptyCatalog(t *testing.T) {
	t.Parallel()

	empty := filepath.Join(t.TempDir(), "empty.yaml")

	err := os.WriteFile(empty, nil, 0o600)
	if err != nil {
		t.Fatalf("writing the empty catalog: %v", err)
	}

	rendered, status := release(t, empty, currentFixture)

	if status != "enabled=true\n" {
		t.Errorf("status = %q, want enabled=true", status)
	}

	wanted := []string{
		"- `Testco/MX100` — new entry, write-enabled on `dp` (`0xD1` via `lg-alt-input`), " +
			"`usb-c` (`0xD2` via `lg-alt-input`)",
		"- `Newco/AB10` — recorded on `hdmi2` (`0x12` via `vcp-input-source`)",
	}

	for _, line := range wanted {
		if !strings.Contains(rendered, line) {
			t.Errorf("the notes do not hold %q; they are:\n%s", line, rendered)
		}
	}
}

func TestTakingSomethingAwayIsNotAnEnablingChange(t *testing.T) {
	t.Parallel()

	comparison := compare(parse(t, enabledEntry), parse(t, disabledEntry))

	if comparison.enabling() {
		t.Error("disabling a model and dropping an identity was reported as enabling")
	}

	rendered := comparison.markdown("v0.1.0")

	wanted := []string{
		"Nothing became sendable in this release.",
		"### Identities removed",
		"- `Testco/MX100` — no longer matches `TST/0x1001`",
		"### Models no longer write-enabled",
		"- `Testco/MX100` — no longer write-enabled",
	}

	for _, line := range wanted {
		if !strings.Contains(rendered, line) {
			t.Errorf("the notes do not hold %q; they are:\n%s", line, rendered)
		}
	}
}

// Recording an input on a model that is not write-enabled adds a value the
// previous binary could not send at all: catalog.Model.UnsafeOperation ignores
// write_enabled, so --unsafe-model sends exactly this. It is an enabling change,
// and a release carrying it is never a patch.
func TestARecordedInputIsEnablingBecauseOfTheOverride(t *testing.T) {
	t.Parallel()

	after := strings.Replace(disabledEntry, `    notes:`, `      hdmi:
        mechanism: vcp-input-source
        value: 0x11
        evidence:
          grade: reported
          by: "somebody"
          tool: "ddcutil"
          url: https://example.invalid/mx100
    notes:`, 1)

	comparison := compare(parse(t, disabledEntry), parse(t, after))

	if !comparison.enabling() {
		t.Error(
			"an input recorded on a model that is not write-enabled was not reported as enabling",
		)
	}

	rendered := comparison.markdown("v0.1.0")

	wanted := []string{
		"### Inputs newly recorded, reachable with `--unsafe-model`",
		"- `Testco/MX100` — `hdmi` (`0x11` via `vcp-input-source`)",
	}

	for _, line := range wanted {
		if !strings.Contains(rendered, line) {
			t.Errorf("the notes do not hold %q; they are:\n%s", line, rendered)
		}
	}
}

// An input arriving in the same release that disabled the model must still be
// named: the model line says what is no longer written, which would otherwise
// read as though nothing new could be sent.
func TestAnInputAddedWhileTheModelWasDisabledIsStillReported(t *testing.T) {
	t.Parallel()

	after := strings.Replace(disabledEntry, `    notes:`, `      hdmi:
        mechanism: vcp-input-source
        value: 0x11
        evidence:
          grade: reported
          by: "somebody"
          tool: "ddcutil"
          url: https://example.invalid/mx100
    notes:`, 1)

	comparison := compare(parse(t, enabledEntry), parse(t, after))

	if !comparison.enabling() {
		t.Error("an input added while the model was disabled was not reported as enabling")
	}

	rendered := comparison.markdown("v0.1.0")

	wanted := []string{
		"### Inputs newly recorded, reachable with `--unsafe-model`",
		"- `Testco/MX100` — `hdmi` (`0x11` via `vcp-input-source`)",
		"### Models no longer write-enabled",
	}

	for _, line := range wanted {
		if !strings.Contains(rendered, line) {
			t.Errorf("the notes do not hold %q; they are:\n%s", line, rendered)
		}
	}
}

func TestAValueChangedOnARecordedInputIsEnabling(t *testing.T) {
	t.Parallel()

	after := strings.Replace(disabledEntry, "value: 0xD0", "value: 0xD3", 1)

	comparison := compare(parse(t, disabledEntry), parse(t, after))

	if !comparison.enabling() {
		t.Error("a changed value on a recorded input was not reported as enabling")
	}

	rendered := comparison.markdown("v0.1.0")

	const wanted = "- `Testco/MX100` — `dp` changed from `0xD0` via `lg-alt-input` to " +
		"`0xD3` via `lg-alt-input`"

	if !strings.Contains(rendered, "### Values changed on a recorded input") ||
		!strings.Contains(rendered, wanted) {
		t.Errorf("the notes do not hold the changed recorded value; they are:\n%s", rendered)
	}
}

// Evidence, notes and sources never reach a monitor, so moving one enables
// nothing - but the notes still have to say the catalog changed, because "no
// catalog change" claims the file is identical.
func TestRecordKeepingChangesAreReportedWithoutEnablingAnything(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		after  string
		wanted string
	}{
		"evidence": {
			after: strings.Replace(
				enabledEntry,
				`"switched on the unit"`,
				`"switched it twice"`,
				1,
			),
			wanted: "- `Testco/MX100` — changed: the evidence for `dp`",
		},
		"notes": {
			after:  strings.Replace(enabledEntry, `"A note."`, `"A different note."`, 1),
			wanted: "- `Testco/MX100` — changed: the notes",
		},
		"sources": {
			after: strings.Replace(
				enabledEntry, "https://example.invalid/mx100", "https://example.invalid/other", 1,
			),
			wanted: "- `Testco/MX100` — changed: the sources",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			comparison := compare(parse(t, enabledEntry), parse(t, test.after))

			if comparison.enabling() {
				t.Errorf("changing the %s was reported as an enabling change", name)
			}

			rendered := comparison.markdown("v0.1.0")

			if !strings.Contains(rendered, "### Evidence, notes and sources") ||
				!strings.Contains(rendered, test.wanted) {
				t.Errorf("the notes do not hold %q; they are:\n%s", test.wanted, rendered)
			}
		})
	}
}

func TestInputsAreListedInCatalogOrder(t *testing.T) {
	t.Parallel()

	const manyInputs = `models:
  - name: MX100
    vendor: Testco
    identities: []
    write_enabled: false
    inputs:
      hdmi2:
        mechanism: vcp-input-source
        value: 0x12
        evidence: {grade: quoted, by: "somebody", tool: "ddcutil", url: https://example.invalid/mx100}
      dp:
        mechanism: vcp-input-source
        value: 0x0F
        evidence: {grade: quoted, by: "somebody", tool: "ddcutil", url: https://example.invalid/mx100}
      hdmi1:
        mechanism: vcp-input-source
        value: 0x11
        evidence: {grade: quoted, by: "somebody", tool: "ddcutil", url: https://example.invalid/mx100}
    sources:
      - https://example.invalid/mx100
`

	document := parse(t, manyInputs)

	rendered := renderInputs(&document.Models[0])
	wanted := "`dp` (`0x0F` via `vcp-input-source`), `hdmi1` (`0x11` via `vcp-input-source`), " +
		"`hdmi2` (`0x12` via `vcp-input-source`)"

	if rendered != wanted {
		t.Errorf("renderInputs() = %q, want %q", rendered, wanted)
	}
}

func TestRunNeedsTwoCatalogFiles(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"none": {},
		"one":  {currentFixture},
		"three": {
			previousFixture, currentFixture, currentFixture,
		},
	}

	for name, paths := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder

			err := run(paths, "v0.1.0", filepath.Join(t.TempDir(), "status"), &out)
			if !errors.Is(err, errUsage) {
				t.Errorf("run(%v) error = %v, want errUsage", paths, err)
			}
		})
	}
}

func TestRunReportsAMalformedCatalog(t *testing.T) {
	t.Parallel()

	broken := filepath.Join(t.TempDir(), "broken.yaml")

	err := os.WriteFile(broken, []byte("models:\n  - unknown_key: 1\n"), 0o600)
	if err != nil {
		t.Fatalf("writing the broken catalog: %v", err)
	}

	var out strings.Builder

	err = run(
		[]string{previousFixture, broken},
		"v0.1.0",
		filepath.Join(t.TempDir(), "status"),
		&out,
	)
	if !errors.Is(err, generate.ErrMalformed) {
		t.Errorf("run() error = %v, want ErrMalformed", err)
	}
}

// release runs the tool over two catalog files the way the workflow does, and
// returns the markdown and the status line.
func release(t *testing.T, previous, current string) (rendered, status string) {
	t.Helper()

	var out strings.Builder

	path := filepath.Join(t.TempDir(), "status")

	err := run([]string{previous, current}, "v0.1.0", path, &out)
	if err != nil {
		t.Fatalf("run(%s, %s): %v", previous, current, err)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the status file: %v", err)
	}

	return out.String(), string(written)
}

// parse decodes an inline catalog, so a case that needs one entry does not need
// a fixture file for it.
func parse(t *testing.T, source string) generate.Document {
	t.Helper()

	document, err := generate.Parse([]byte(source))
	if err != nil {
		t.Fatalf("parsing the catalog: %v", err)
	}

	return document
}
