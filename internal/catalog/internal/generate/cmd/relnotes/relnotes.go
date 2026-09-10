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
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/leinardi/monmux/internal/catalog/internal/generate"
	"github.com/leinardi/monmux/internal/catalog/internal/input"
)

// heading is the title of the whole section, which goes at the top of the
// release notes.
const heading = "## Catalog"

// The two opening sentences. The first one is the reason this tool exists: a
// release that sends something the previous one did not has to say so before
// anything else, and generated commit-message notes cannot know that.
const (
	enablingLead = "**This release changes which bytes monmux can send to a monitor.** " +
		"Read this section before upgrading. A recorded entry counts even when it is not write-enabled: " +
		"`--unsafe-model` reaches it."
	recordedLead = "Nothing became sendable in this release. The catalog only gave something up, " +
		"or moved the record behind a value it already carried."
)

// unchangedFormat is what the section says when the two catalogs are identical.
// Silence would be indistinguishable from a step that did not run.
const unchangedFormat = "No catalog change: this release writes exactly what %s wrote.\n"

// section is one heading of the catalog notes, with its bullets.
//
// enabling marks the sections that make a release at least a minor one: they
// list bytes this binary can send and the previous one could not.
//
// That includes an entry which is merely recorded, with write_enabled false.
// Such an entry is never matched and never written to by a normal switch, but
// catalog.Model.UnsafeOperation ignores the flag, so `--unsafe-model` sends
// exactly those values - recording one adds a value the previous binary had no
// way to send at all. The classification errs towards enabling for the same
// reason the rest of the project errs towards refusing: the cost of calling a
// patch a minor is a version number, and the cost of the reverse is a release
// that sends something new to a monitor without saying so.
type section struct {
	title    string
	enabling bool
	lines    []string
}

// notes is the whole comparison of two catalog files.
type notes struct {
	sections []section
}

// enabling reports whether the comparison found a change that enables a model
// or an input. The release workflow reads it: a patch bump becomes a minor.
func (n notes) enabling() bool {
	for _, part := range n.sections {
		if part.enabling && len(part.lines) > 0 {
			return true
		}
	}

	return false
}

// markdown renders the section. previous is how the notes name the release
// being compared against, and is only used when nothing changed.
func (n notes) markdown(previous string) string {
	var out strings.Builder

	out.WriteString(heading + "\n\n")

	written := false

	for _, part := range n.sections {
		if len(part.lines) == 0 {
			continue
		}

		if !written {
			out.WriteString(n.lead() + "\n\n")

			written = true
		}

		fmt.Fprintf(&out, "### %s\n\n", part.title)

		for _, line := range part.lines {
			out.WriteString(line + "\n")
		}

		out.WriteString("\n")
	}

	if !written {
		fmt.Fprintf(&out, unchangedFormat, previous)
	}

	return out.String()
}

// lead is the sentence under the heading.
func (n notes) lead() string {
	if n.enabling() {
		return enablingLead
	}

	return recordedLead
}

// changes accumulates the comparison, one slice per section, so that the order
// of the sections is decided in one place and the walk does not have to know it.
type changes struct {
	writeEnabled      []string
	enabledInputs     []string
	changedValues     []string
	addedIdentities   []string
	recordedModels    []string
	recordedInputs    []string
	changedRecorded   []string
	removedIdentities []string
	noLongerEnabled   []string
	removedInputs     []string
	removedModels     []string
	records           []string
}

// sections is the order the notes are printed in: everything that widens what
// monmux can send first - what it writes by itself, then what `--unsafe-model`
// reaches - and everything it gives up or merely documents after.
func (c *changes) sections() []section {
	return []section{
		{title: "Models newly write-enabled", enabling: true, lines: c.writeEnabled},
		{title: "Inputs newly enabled", enabling: true, lines: c.enabledInputs},
		{title: "Values changed on an enabled input", enabling: true, lines: c.changedValues},
		{title: "Identities added", enabling: true, lines: c.addedIdentities},
		{
			title:    "Models newly recorded, reachable with `--unsafe-model`",
			enabling: true,
			lines:    c.recordedModels,
		},
		{
			title:    "Inputs newly recorded, reachable with `--unsafe-model`",
			enabling: true,
			lines:    c.recordedInputs,
		},
		{title: "Values changed on a recorded input", enabling: true, lines: c.changedRecorded},
		{title: "Identities removed", enabling: false, lines: c.removedIdentities},
		{title: "Models no longer write-enabled", enabling: false, lines: c.noLongerEnabled},
		{title: "Inputs no longer recorded", enabling: false, lines: c.removedInputs},
		{title: "Models removed from the catalog", enabling: false, lines: c.removedModels},
		{title: "Evidence, notes and sources", enabling: false, lines: c.records},
	}
}

// compare walks the two catalogs and reports what changed between them. Both
// walks are in file order, which is the order the catalog and the compatibility
// document are written in, so the notes read in the same order as the table.
func compare(previous, current generate.Document) notes {
	var found changes

	before := index(previous)

	for at := range current.Models {
		model := &current.Models[at]

		earlier, existed := before[entryName(model)]
		if !existed {
			found.newModel(model)

			continue
		}

		found.changedModel(earlier, model)
	}

	after := index(current)

	for at := range previous.Models {
		model := &previous.Models[at]

		_, kept := after[entryName(model)]
		if kept {
			continue
		}

		found.removedModels = append(
			found.removedModels,
			bullet(entryName(model), removedDetail(model)),
		)
	}

	return notes{sections: found.sections()}
}

// newModel records an entry the previous catalog did not have at all.
func (c *changes) newModel(model *generate.Model) {
	name := entryName(model)

	if model.WriteEnabled {
		c.writeEnabled = append(
			c.writeEnabled,
			bullet(name, "new entry, write-enabled on "+renderInputs(model)),
		)

		return
	}

	c.recordedModels = append(c.recordedModels, bullet(name, "recorded on "+renderInputs(model)))
}

// changedModel records what moved on an entry both catalogs have.
//
// Every walk runs on every entry. A model that crosses the write-enabled line
// gets one line naming every input it now writes, because crossing that line
// enables all of them at once; the per-input walk still runs next to it, so that
// an input added in the same release as the flag moved is never silently folded
// into that one line.
func (c *changes) changedModel(before, after *generate.Model) {
	name := entryName(after)

	c.identities(name, before, after)
	c.inputs(name, before, after)
	c.evidence(name, before, after)

	switch {
	case !before.WriteEnabled && after.WriteEnabled:
		c.writeEnabled = append(
			c.writeEnabled,
			bullet(name, "write-enabled on "+renderInputs(after)),
		)
	case before.WriteEnabled && !after.WriteEnabled:
		c.noLongerEnabled = append(c.noLongerEnabled, bullet(name, "no longer write-enabled"))
	}
}

// inputs records the per-input changes of an entry both catalogs have. An input
// recorded on a model that is not write-enabled is an enabling change too: the
// value is what `--unsafe-model` sends, and the previous binary did not carry it.
func (c *changes) inputs(name string, before, after *generate.Model) {
	enabled := after.WriteEnabled

	for _, key := range sortedInputs(after.Inputs) {
		now := after.Inputs[key]

		then, existed := before.Inputs[key]
		if existed && sameOperation(&then, &now) {
			continue
		}

		switch {
		case !existed && enabled:
			c.enabledInputs = append(c.enabledInputs, bullet(name, renderInput(key, &now)))
		case !existed:
			c.recordedInputs = append(c.recordedInputs, bullet(name, renderInput(key, &now)))
		case enabled:
			c.changedValues = append(c.changedValues, bullet(name, changedDetail(key, &then, &now)))
		default:
			c.changedRecorded = append(
				c.changedRecorded,
				bullet(name, changedDetail(key, &then, &now)),
			)
		}
	}

	for _, key := range sortedInputs(before.Inputs) {
		_, kept := after.Inputs[key]
		if kept {
			continue
		}

		c.removedInputs = append(c.removedInputs, bullet(name, fmt.Sprintf("`%s`", key)))
	}
}

// evidence records the parts of an entry that say why a value is believed,
// rather than what it is: the per-input evidence, the model notes and the
// sources. None of it ever reaches a monitor, so none of it enables anything -
// but "no catalog change" has to mean the file is identical, not that the bytes
// happen to match, so a moved record is reported rather than swallowed.
//
// Evidence is only compared where the operation stayed put. Where the value or
// the mechanism moved, the evidence moved with it by definition, and the input
// is already named in an enabling section above.
func (c *changes) evidence(name string, before, after *generate.Model) {
	moved := []string{}

	for _, key := range sortedInputs(after.Inputs) {
		now := after.Inputs[key]

		then, existed := before.Inputs[key]
		if !existed || !sameOperation(&then, &now) {
			continue
		}

		if then.Evidence != now.Evidence {
			moved = append(moved, fmt.Sprintf("the evidence for `%s`", key))
		}
	}

	if !slices.Equal(before.Notes, after.Notes) {
		moved = append(moved, "the notes")
	}

	if !slices.Equal(before.Sources, after.Sources) {
		moved = append(moved, "the sources")
	}

	if len(moved) == 0 {
		return
	}

	c.records = append(c.records, bullet(name, "changed: "+strings.Join(moved, ", ")))
}

// identities records the EDID fingerprints an entry gained or lost. An added
// one is an enabling change: it is what makes monmux match a display it
// previously refused as unknown.
func (c *changes) identities(name string, before, after *generate.Model) {
	earlier := renderIdentities(before.Identities)
	later := renderIdentities(after.Identities)

	for _, identity := range later {
		if !slices.Contains(earlier, identity) {
			c.addedIdentities = append(c.addedIdentities, bullet(name, "matches "+identity))
		}
	}

	for _, identity := range earlier {
		if !slices.Contains(later, identity) {
			c.removedIdentities = append(
				c.removedIdentities,
				bullet(name, "no longer matches "+identity),
			)
		}
	}
}

// index maps every entry of a catalog by the name the command line uses for it.
func index(document generate.Document) map[string]*generate.Model {
	models := make(map[string]*generate.Model, len(document.Models))

	for at := range document.Models {
		model := &document.Models[at]
		models[entryName(model)] = model
	}

	return models
}

// entryName is how `monmux catalog show` names an entry: VENDOR/MODEL.
func entryName(model *generate.Model) string {
	return model.Vendor + "/" + model.Name
}

// bullet renders one line of a section.
func bullet(entry, detail string) string {
	return fmt.Sprintf("- `%s` — %s", entry, detail)
}

// removedDetail says whether the entry that went away could be written to.
func removedDetail(model *generate.Model) string {
	if model.WriteEnabled {
		return "removed from the catalog; it was write-enabled"
	}

	return "removed from the catalog"
}

// changedDetail renders one input whose operation moved.
func changedDetail(name string, before, after *generate.Input) string {
	return fmt.Sprintf(
		"`%s` changed from %s to %s",
		name,
		renderOperation(before),
		renderOperation(after),
	)
}

// renderInputs lists every input of a model, in catalog order.
func renderInputs(model *generate.Model) string {
	names := sortedInputs(model.Inputs)
	if len(names) == 0 {
		return "no input"
	}

	rendered := make([]string, 0, len(names))

	for _, name := range names {
		spec := model.Inputs[name]
		rendered = append(rendered, renderInput(name, &spec))
	}

	return strings.Join(rendered, ", ")
}

// renderInput renders one input, its value and its mechanism.
func renderInput(name string, spec *generate.Input) string {
	return fmt.Sprintf("`%s` (%s)", name, renderOperation(spec))
}

// renderOperation names what an input would send. It is prose about the
// catalog, not an operation: nothing here reaches a monitor.
func renderOperation(spec *generate.Input) string {
	return fmt.Sprintf("%s via `%s`", renderValue(spec.Value), spec.Mechanism)
}

// renderValue renders a 16-bit catalog value. A file that never wrote one says
// so rather than rendering the zero a missing value would decode as.
func renderValue(value *uint16) string {
	if value == nil {
		return "no value"
	}

	return fmt.Sprintf("`0x%02X`", *value)
}

// renderIdentities renders every fingerprint of a model. The rendered string is
// also the comparison key: it holds all three fields an identity is made of, so
// two identities are the same one exactly when they read the same.
func renderIdentities(identities []generate.Identity) []string {
	rendered := make([]string, 0, len(identities))

	for at := range identities {
		rendered = append(rendered, renderIdentity(&identities[at]))
	}

	return rendered
}

// renderIdentity renders one fingerprint the way monmux documentation writes it.
func renderIdentity(identity *generate.Identity) string {
	code := "no product code"
	if identity.ProductCode != nil {
		code = fmt.Sprintf("0x%04X", *identity.ProductCode)
	}

	rendered := fmt.Sprintf("`%s/%s`", identity.Manufacturer, code)
	if identity.ModelName == "" {
		return rendered
	}

	return fmt.Sprintf("%s %q", rendered, identity.ModelName)
}

// sameOperation reports whether an input would send exactly what it sent
// before. Only the mechanism and the value are compared: they are what reaches
// a monitor, and a reworded piece of evidence is not a release note.
func sameOperation(before, after *generate.Input) bool {
	return before.Mechanism == after.Mechanism && sameValue(before.Value, after.Value)
}

// sameValue compares two catalog values, either of which may be absent.
func sameValue(before, after *uint16) bool {
	if before == nil || after == nil {
		return before == after
	}

	return *before == *after
}

// sortedInputs orders an entry's input keys the way the catalog orders them.
func sortedInputs(inputs map[string]generate.Input) []string {
	names := slices.Collect(maps.Keys(inputs))
	slices.SortFunc(names, compareInputNames)

	return names
}

// compareInputNames orders two input keys by connector kind and port number. A
// key the grammar does not accept cannot be ordered by it and sorts last; the
// generator rejects such a key, so this only decides what a broken file reads
// like, and it never crashes on one.
func compareInputNames(first, second string) int {
	one, oneErr := input.Parse(first)
	two, twoErr := input.Parse(second)

	switch {
	case oneErr != nil && twoErr != nil:
		return strings.Compare(first, second)
	case oneErr != nil:
		return 1
	case twoErr != nil:
		return -1
	default:
		return input.Compare(one, two)
	}
}
