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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/leinardi/monmux/internal/catalog"
)

// errUnknownModel reports a name no catalog entry carries.
var errUnknownModel = errors.New("no catalog entry is named")

// catalogColumns are the column titles of `monmux catalog list`.
var catalogColumns = []string{"VENDOR", "MODEL", "WRITE", "MECHANISM", "INPUTS"}

// columnGap separates two columns of a rendered table.
const columnGap = "  "

// newCatalogCmd returns the `catalog` command group, which answers from the
// binary alone: the catalog is compiled in, so nothing here opens a backend,
// reads the configuration file or touches a monitor.
func newCatalogCmd() *cobra.Command {
	group := &cobra.Command{
		Use:   "catalog",
		Short: "Show the built-in supported-monitor catalog",
		Long: "Prints what monmux knows about monitors, from the catalog compiled into this\n" +
			"binary. Nothing here looks at the attached displays: it is the same table as\n" +
			"docs/compatibility.md, offline and for this exact build.\n\n" +
			"A model is listed whether or not monmux may write to it. Only a write-enabled\n" +
			"entry can be switched; the rest record what somebody else reported, with the\n" +
			"evidence, and are refused.",
	}

	group.AddCommand(newCatalogListCmd(), newCatalogShowCmd())

	return group
}

// newCatalogListCmd returns `catalog list`.
func newCatalogListCmd() *cobra.Command {
	var (
		asJSON  bool
		verbose bool
	)

	command := &cobra.Command{
		Use:   "list [filter]",
		Short: "List every catalog entry",
		Long: "Lists every model in the catalog, in catalog order, with whether monmux may\n" +
			"write to it, how its inputs are switched, and which inputs are recorded for\n" +
			"it. A recorded input is not necessarily a writable one: that needs the model\n" +
			"to be write-enabled.\n\n" +
			"The optional filter is matched, case-insensitively, as a substring of the\n" +
			"vendor, the model name, and \"vendor/name\".",
		Args: cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			filter := ""
			if len(args) == 1 {
				filter = args[0]
			}

			return listCatalog(command.OutOrStdout(), filter, verbose, asJSON)
		},
	}

	command.Flags().BoolVar(&asJSON, "json", false, "print the catalog as JSON")
	command.Flags().BoolVar(
		&verbose,
		"verbose",
		false,
		"add the value and the evidence grade of every recorded input",
	)

	return command
}

// newCatalogShowCmd returns `catalog show`.
func newCatalogShowCmd() *cobra.Command {
	var asJSON bool

	command := &cobra.Command{
		Use:   "show <vendor/name>",
		Short: "Print everything the catalog records about one model",
		Long: "Prints one catalog entry in full: its EDID identities, every recorded input\n" +
			"with its mechanism, value, evidence grade and the evidence itself, and the\n" +
			"per-model notes and sources.\n\n" +
			"The name is \"vendor/name\" or \"vendor name\", case-insensitively, as printed\n" +
			"by \"monmux catalog list\".",
		Args: cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return showCatalog(command.OutOrStdout(), args[0], asJSON)
		},
	}

	command.Flags().BoolVar(&asJSON, "json", false, "print the entry as JSON")

	return command
}

// listCatalog prints the catalog, filtered or whole.
func listCatalog(out io.Writer, filter string, verbose, asJSON bool) error {
	found := filterModels(catalog.Models(), filter)

	if asJSON {
		return encodeJSON(out, asJSONCatalog(found))
	}

	return renderCatalogList(out, found, verbose)
}

// showCatalog prints one entry.
//
// An unknown name is a mistake in the request rather than a refused write: it is
// an ordinary error, so it exits 1 and says nothing about monitors.
func showCatalog(out io.Writer, name string, asJSON bool) error {
	entry, ok := catalog.Find(name)
	if !ok {
		return fmt.Errorf(
			"%w %q (run \"monmux catalog list\" to see every entry)",
			errUnknownModel,
			name,
		)
	}

	if asJSON {
		return encodeJSON(out, asJSONCatalogModel(&entry))
	}

	return renderCatalogModel(out, &entry)
}

// filterModels keeps the entries whose vendor, name or "vendor/name" contains
// the filter, case-insensitively. An empty filter keeps everything.
func filterModels(entries []catalog.Model, filter string) []catalog.Model {
	if filter == "" {
		return entries
	}

	wanted := strings.ToLower(filter)

	found := make([]catalog.Model, 0, len(entries))

	for index := range entries {
		entry := &entries[index]

		vendor := strings.ToLower(entry.Vendor)
		name := strings.ToLower(entry.Name)

		if strings.Contains(vendor, wanted) ||
			strings.Contains(name, wanted) ||
			strings.Contains(vendor+"/"+name, wanted) {
			found = append(found, *entry)
		}
	}

	return found
}

// renderCatalogList prints the table and the summary underneath it.
func renderCatalogList(out io.Writer, found []catalog.Model, verbose bool) error {
	page := &printer{out: out}

	if len(found) == 0 {
		page.linef("No catalog entry matches that filter.")

		return page.err
	}

	rows := make([][]string, 0, len(found)+1)
	rows = append(rows, catalogColumns)

	writeEnabled := 0

	for index := range found {
		entry := &found[index]

		if entry.WriteEnabled {
			writeEnabled++
		}

		rows = append(rows, []string{
			entry.Vendor,
			entry.Name,
			yesNo(entry.WriteEnabled),
			strings.Join(modelMechanisms(entry), ", "),
			modelInputs(entry, verbose),
		})
	}

	widths := columnWidths(rows)
	for _, row := range rows {
		page.linef("%s", renderRow(row, widths))
	}

	page.linef("")
	page.linef("%s, %d write-enabled.", plural(len(found), "model"), writeEnabled)

	return page.err
}

// renderCatalogModel prints one entry in full.
func renderCatalogModel(out io.Writer, entry *catalog.Model) error {
	page := &printer{out: out}

	page.linef("%s", entry.FullName())
	page.linef("  Write-enabled: %s", yesNo(entry.WriteEnabled))
	page.linef("  Identities:    %s", modelIdentities(entry))

	renderModelInputs(page, entry)
	renderModelList(page, "Notes", entry.Notes)
	renderModelList(page, "Sources", entry.Sources)

	if !entry.WriteEnabled {
		page.linef("")
		page.linef("Not write-enabled: \"monmux switch\" refuses this model, because nothing")
		page.linef("recorded here was verified by this project. To send a recorded value")
		page.linef(
			"anyway, add --unsafe-model %s (with --dry-run first).",
			entry.Vendor+"/"+entry.Name,
		)
		page.linef("See docs/adding-a-monitor.md for what it takes to write-enable an entry.")
	}

	return page.err
}

// renderModelInputs prints the recorded inputs, one per line, with the evidence
// behind each of them.
func renderModelInputs(page *printer, entry *catalog.Model) {
	recorded := entry.RecordedInputs()
	if len(recorded) == 0 {
		page.linef("  Inputs:        %s", noneListed)

		return
	}

	page.linef("  Inputs:")

	rows := make([][]string, 0, len(recorded))

	for _, input := range recorded {
		mechanism, _ := entry.InputMechanism(input)
		value, _ := entry.InputValue(input)
		grade, _ := entry.InputGrade(input)
		evidence, _ := entry.InputEvidence(input)

		rows = append(rows, []string{
			input.String(),
			mechanism.String(),
			fmt.Sprintf("0x%02X", value),
			grade.String(),
			evidence,
		})
	}

	widths := columnWidths(rows)
	for _, row := range rows {
		page.linef("    %s", renderRow(row, widths))
	}
}

// renderModelList prints a titled list of prose lines, and nothing at all when
// there are none.
func renderModelList(page *printer, title string, lines []string) {
	if len(lines) == 0 {
		return
	}

	page.linef("  %s:", title)

	for _, line := range lines {
		page.linef("    - %s", line)
	}
}

// modelMechanisms are the distinct mechanisms across an entry's recorded inputs,
// in the order the inputs are listed.
func modelMechanisms(entry *catalog.Model) []string {
	var names []string

	for _, input := range entry.RecordedInputs() {
		mechanism, ok := entry.InputMechanism(input)
		if !ok {
			continue
		}

		name := mechanism.String()
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	return names
}

// modelInputs renders an entry's recorded inputs - not its enabled ones, which
// is the point of the listing: a recorded input that is refused is still worth
// seeing. With verbose it carries the value and the evidence grade too.
func modelInputs(entry *catalog.Model, verbose bool) string {
	recorded := entry.RecordedInputs()
	if len(recorded) == 0 {
		return noneListed
	}

	names := make([]string, 0, len(recorded))

	for _, input := range recorded {
		if !verbose {
			names = append(names, input.String())

			continue
		}

		value, _ := entry.InputValue(input)
		grade, _ := entry.InputGrade(input)

		names = append(names, fmt.Sprintf("%s 0x%02X %s", input, value, grade))
	}

	return strings.Join(names, ", ")
}

// modelIdentities renders the EDID fingerprints an entry is recognized by.
func modelIdentities(entry *catalog.Model) string {
	if len(entry.Identities) == 0 {
		return "none (can never match a display)"
	}

	names := make([]string, 0, len(entry.Identities))
	for _, identity := range entry.Identities {
		names = append(names, identity.String())
	}

	return strings.Join(names, ", ")
}

// columnWidths returns the width of every column but the last, which needs none.
func columnWidths(rows [][]string) []int {
	widths := make([]int, len(rows[0]))

	for _, row := range rows {
		for index, cell := range row {
			if len(cell) > widths[index] {
				widths[index] = len(cell)
			}
		}
	}

	return widths
}

// renderRow pads one row to the column widths. The last column is never padded,
// so no line ends in trailing spaces.
func renderRow(row []string, widths []int) string {
	cells := make([]string, 0, len(row))

	for index, cell := range row {
		if index == len(row)-1 {
			cells = append(cells, cell)

			continue
		}

		cells = append(cells, cell+strings.Repeat(" ", widths[index]-len(cell)))
	}

	return strings.TrimRight(strings.Join(cells, columnGap), " ")
}

// yesNo renders a flag the way the table prints it.
func yesNo(enabled bool) string {
	if enabled {
		return "yes"
	}

	return "no"
}

// plural renders a count with its noun, e.g. "1 model" or "71 models".
func plural(total int, noun string) string {
	if total == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", total, noun)
}

// The JSON shapes below are the command's output contract, kept separate from
// the catalog's own types on purpose: a field renamed inside monmux must not
// silently rename itself in somebody's script.
type (
	// jsonCatalog is the whole of `monmux catalog list --json`.
	jsonCatalog struct {
		Models []jsonCatalogModel `json:"models"`
		// Count is how many entries are listed, after any filter.
		Count int `json:"count"`
		// WriteEnabled is how many of those monmux may write to.
		WriteEnabled int `json:"writeEnabled"`
	}

	// jsonCatalogModel is one catalog entry, and the whole of `catalog show --json`.
	jsonCatalogModel struct {
		Vendor       string             `json:"vendor"`
		Name         string             `json:"name"`
		FullName     string             `json:"fullName"`
		WriteEnabled bool               `json:"writeEnabled"`
		Identities   []string           `json:"identities"`
		Mechanisms   []string           `json:"mechanisms"`
		Inputs       []jsonCatalogInput `json:"inputs"`
		Notes        []string           `json:"notes"`
		Sources      []string           `json:"sources"`
	}

	// jsonCatalogInput is one recorded input of one entry, with its evidence.
	jsonCatalogInput struct {
		Name      string `json:"name"`
		Mechanism string `json:"mechanism"`
		// Value is the recorded value; ValueHex is the same value as the
		// documentation writes it.
		Value    uint16 `json:"value"`
		ValueHex string `json:"valueHex"`
		Grade    string `json:"grade"`
		Evidence string `json:"evidence"`
	}
)

// asJSONCatalog converts a listing into the output contract.
func asJSONCatalog(found []catalog.Model) jsonCatalog {
	document := jsonCatalog{
		Models: make([]jsonCatalogModel, 0, len(found)),
	}

	for index := range found {
		entry := &found[index]

		if entry.WriteEnabled {
			document.WriteEnabled++
		}

		document.Models = append(document.Models, asJSONCatalogModel(entry))
	}

	document.Count = len(document.Models)

	return document
}

// asJSONCatalogModel converts one entry into the output contract.
func asJSONCatalogModel(entry *catalog.Model) jsonCatalogModel {
	document := jsonCatalogModel{
		Vendor:       entry.Vendor,
		Name:         entry.Name,
		FullName:     entry.FullName(),
		WriteEnabled: entry.WriteEnabled,
		Identities:   make([]string, 0, len(entry.Identities)),
		Mechanisms:   modelMechanisms(entry),
		Inputs:       make([]jsonCatalogInput, 0, len(entry.Inputs)),
		Notes:        entry.Notes,
		Sources:      entry.Sources,
	}

	if document.Mechanisms == nil {
		document.Mechanisms = []string{}
	}

	if document.Notes == nil {
		document.Notes = []string{}
	}

	if document.Sources == nil {
		document.Sources = []string{}
	}

	for _, identity := range entry.Identities {
		document.Identities = append(document.Identities, identity.String())
	}

	for _, input := range entry.RecordedInputs() {
		mechanism, _ := entry.InputMechanism(input)
		value, _ := entry.InputValue(input)
		grade, _ := entry.InputGrade(input)
		evidence, _ := entry.InputEvidence(input)

		document.Inputs = append(document.Inputs, jsonCatalogInput{
			Name:      input.String(),
			Mechanism: mechanism.String(),
			Value:     value,
			ValueHex:  fmt.Sprintf("0x%02X", value),
			Grade:     grade.String(),
			Evidence:  evidence,
		})
	}

	return document
}

// encodeJSON prints a document.
//
// HTML escaping is turned off for the same reason `monmux info --json` turns it
// off: this output goes to a terminal or to jq, and escaping mangles the very
// strings a reader is looking at.
func encodeJSON(out io.Writer, document any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(document)
	if err != nil {
		return fmt.Errorf("writing the catalog as JSON: %w", err)
	}

	return nil
}
