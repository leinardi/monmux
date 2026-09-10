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
	"fmt"
	"io"
	"strings"

	"github.com/leinardi/monmux/internal/app"
	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

// noModel is what is printed where a catalog entry would be for a monitor that
// has none.
const noModel = "not in the catalog"

// noneListed is what is printed where a list would be, when it is empty.
const noneListed = "none"

// printer writes lines and remembers the first failure, so the rendering code
// reads as the output it produces instead of as error handling.
type printer struct {
	out io.Writer
	err error
}

// linef writes one formatted line.
func (p *printer) linef(format string, args ...any) {
	if p.err != nil {
		return
	}

	_, p.err = fmt.Fprintf(p.out, format+"\n", args...)
}

// renderReport prints what monmux can see.
func renderReport(out io.Writer, report *app.Report, showSerial bool) error {
	page := &printer{out: out}

	page.linef("Backend: %s", report.Backend)

	for index := range report.Displays {
		page.linef("")
		renderDisplay(page, &report.Displays[index], showSerial)
	}

	page.linef("")
	renderChecks(page, report.Checks)

	return page.err
}

// renderDisplay prints one display, whether or not monmux would write to it.
func renderDisplay(page *printer, reported *app.DisplayReport, showSerial bool) {
	display := reported.Display

	page.linef("Display %s", display.Label)
	page.linef("  Handle:  %s", handle(&display, showSerial))
	page.linef("  Monitor: %s", monitor(display.Identity))
	page.linef("  Serial:  %s", serial(display.Identity, showSerial))
	page.linef("  Status:  %s", status(&display))
	page.linef("  Model:   %s", model(reported))
	page.linef("  Inputs:  %s", inputs(reported.EnabledInputs))
}

// renderChecks prints the backend's diagnostics.
func renderChecks(page *printer, checks []backend.Check) {
	page.linef("Checks:")

	if len(checks) == 0 {
		page.linef("  (none reported)")

		return
	}

	for _, check := range checks {
		page.linef("  %-4s %s: %s", verdict(check.OK), check.Name, check.Detail)
	}
}

// renderOutcome prints what a switch did, or would have done.
//
// The success sentence is the one required by the requirements document: it says
// the command was sent, and it says in the same breath that the switch was not
// confirmed. monmux never reads back what a monitor is doing, and the wording
// must not let a reader believe otherwise.
func renderOutcome(out io.Writer, outcome *app.Outcome, showSerial bool) error {
	page := &printer{out: out}

	if !outcome.DryRun {
		page.linef(
			"Input-switch command sent (%s, 0x%02X) to %s via %s%s. "+
				"Switch not independently confirmed.",
			outcome.Input.Label(),
			outcome.Operation.Value(),
			outcome.Model.FullName(),
			outcome.Backend,
			bypassed(outcome.Assumed),
		)

		return page.err
	}

	page.linef("Dry run: nothing was sent.")
	page.linef(
		"  Display: %s (%s)%s",
		outcome.Display.Label,
		outcome.Model.FullName(),
		bypassed(outcome.Assumed),
	)
	page.linef("  Input:   %s (0x%02X)", outcome.Input.Label(), outcome.Operation.Value())
	page.linef("  Command: %s", outcome.Command.Render(showSerial))
	page.linef("No DDC write was performed.")

	return page.err
}

// renderUnsafeWarning prints the banner for a switch that bypasses
// identification.
//
// It goes to stderr once the display has been selected and before anything is
// planned or written, so the user is told which monitor is about to receive an
// unverified value - in a dry run too, where none is. There is no prompt: the
// flag's name and this banner are the guard, and a prompt would break scripting,
// which is what the flag is for.
func renderUnsafeWarning(
	out io.Writer,
	display *backend.Display,
	assumed *catalog.Model,
	input catalog.Input,
) error {
	page := &printer{out: out}

	page.linef("WARNING: identification bypassed by --unsafe-model.")
	page.linef("  Display: %s (%s)", display.Label, monitor(display.Identity))
	page.linef(
		"  Assumed: %s (write-enabled: %s, evidence: %s)",
		assumed.FullName(),
		yesNo(assumed.WriteEnabled),
		assumedGrade(assumed, input),
	)
	page.linef("  Input:   %s", assumedInput(assumed, input))
	page.linef("The display above was not identified, and the value was never verified on")
	page.linef("this model by this project. A wrong value can leave the monitor on an input")
	page.linef("with no signal; recover with the monitor's own OSD or by unplugging the")
	page.linef("other inputs.")

	return page.err
}

// bypassed is what the outcome line adds when the display was never identified,
// so terminal scrollback shows this was not an ordinary switch.
func bypassed(assumed bool) string {
	if !assumed {
		return ""
	}

	return " (identification bypassed)"
}

// assumedGrade renders the evidence behind the value the assumed model records
// for this input.
func assumedGrade(assumed *catalog.Model, input catalog.Input) string {
	grade, recorded := assumed.InputGrade(input)
	if !recorded {
		return noneListed
	}

	return grade.String()
}

// assumedInput renders the input with the value and mechanism the assumed model
// records for it. An input it does not record has no value to print, and the
// switch is refused with input-not-enabled a moment later.
func assumedInput(assumed *catalog.Model, input catalog.Input) string {
	value, recorded := assumed.InputValue(input)

	mechanism, known := assumed.InputMechanism(input)
	if !recorded || !known {
		return input.Label() + " (not recorded for this model)"
	}

	return fmt.Sprintf("%s (0x%02X, %s)", input.Label(), value, mechanism)
}

// modelNames are the catalog entries as "Vendor/Name". It is what --unsafe-model
// completes against: being in the catalog is not a promise that monmux may write
// to the model, which is exactly what that flag overrides.
func modelNames() []string {
	entries := catalog.Models()

	names := make([]string, 0, len(entries))
	for index := range entries {
		names = append(names, entries[index].Vendor+"/"+entries[index].Name)
	}

	return names
}

// handle renders the backend's address for a display, masking it when it is
// private data - as the macOS UUID is.
func handle(display *backend.Display, showSerial bool) string {
	if display.HandlePrivate && !showSerial {
		return refusal.RedactionMask
	}

	return display.Handle
}

// monitor renders the identity the catalog is matched on.
func monitor(identity edid.Identity) string {
	rendered := fmt.Sprintf("%s 0x%04X", identity.Manufacturer, identity.ProductCode)
	if identity.ModelName != "" {
		rendered += " (" + identity.ModelName + ")"
	}

	return rendered
}

// serial renders the serial fields, redacted unless the user asked for them.
func serial(identity edid.Identity, showSerial bool) string {
	if !identity.HasSerial() {
		return noneListed
	}

	if !showSerial {
		return refusal.RedactionMask
	}

	if identity.SerialString == "" {
		return fmt.Sprintf("0x%08X", identity.SerialNumber)
	}

	return fmt.Sprintf("%s (0x%08X)", identity.SerialString, identity.SerialNumber)
}

// status renders whether monmux would write to this display, and why not.
func status(display *backend.Display) string {
	if display.Writable {
		return display.Status + " (writable)"
	}

	return display.Status + " (not writable)"
}

// model renders the catalog entry, if there is one.
func model(reported *app.DisplayReport) string {
	if reported.Match != catalog.MatchExact {
		return noModel + " (" + reported.Match.String() + ")"
	}

	return reported.Model.FullName()
}

// inputs renders the inputs monmux is willing to switch a display to.
func inputs(enabled []catalog.Input) string {
	if len(enabled) == 0 {
		return noneListed
	}

	names := make([]string, 0, len(enabled))
	for _, input := range enabled {
		names = append(names, input.String())
	}

	return strings.Join(names, ", ")
}

// firstLine returns the first line of a message, for the one-line diagnostics.
func firstLine(message string) string {
	before, _, found := strings.Cut(message, "\n")
	if !found {
		return message
	}

	return before
}

// verdict renders a check's result.
func verdict(ok bool) string {
	if ok {
		return "ok"
	}

	return "FAIL"
}

// inputNames are the inputs any catalog entry records, as typed on the command
// line. It is what completion offers: a name being recorded says nothing about
// whether the attached monitor enables it, which only `monmux info` can say.
func inputNames() []string {
	known := catalog.KnownInputs()

	names := make([]string, 0, len(known))
	for _, input := range known {
		names = append(names, input.String())
	}

	return names
}

// kindNames are the connector kinds an input name can start with.
func kindNames() []string {
	kinds := catalog.Kinds()

	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, kind.String())
	}

	return names
}

// The switch outcomes, as `monmux switch --json` names them. They are part of
// the output contract, and each one is paired with exactly one write status.
const (
	// outcomeSent means the command was sent. The switch itself is not
	// independently confirmed: nothing here reads back what the monitor does.
	outcomeSent = "sent"
	// outcomeDryRun means the command was built and not executed.
	outcomeDryRun = "dry-run"
	// outcomeRefused means monmux declined, and nothing was written.
	outcomeRefused = "refused"
	// outcomeFailed means the request could not be made, or the tool ran and
	// failed.
	outcomeFailed = "failed"
)

// The write statuses. Only these three exist, and "none" is a promise: it is
// made for a refusal and for a dry run, the two cases where monmux knows
// nothing left the process.
const (
	writeStatusSent    = "sent"
	writeStatusNone    = "none"
	writeStatusUnknown = "unknown"
)

// degradedOutcome is the error text for a switch that reported neither a send
// nor a dry run. It cannot happen through app.Switch; if it ever does, the
// document says the write status is unknown rather than promising anything.
const degradedOutcome = "the switch reported neither a send nor a dry run"

// The JSON shapes below are `monmux switch --json`'s output contract, kept
// separate from the internal types on purpose: a field renamed inside monmux
// must not silently rename itself in somebody's script.
type (
	// jsonSwitch is the whole of one `monmux switch --json` run. Exactly one
	// document is printed for every run that reaches the handler, whatever the
	// outcome and whatever the exit code.
	jsonSwitch struct {
		// Outcome and WriteStatus are always present, and always paired:
		// sent/sent, dry-run/none, refused/none, failed/unknown.
		Outcome     string `json:"outcome"`
		WriteStatus string `json:"writeStatus"`
		// Backend is the backend that handled the request, absent when the
		// failure happened before one could be opened.
		Backend string `json:"backend,omitempty"`
		// Display, Model, Input, InputLabel and Operation are present only when
		// a decision was reached, which is to say for sent and dry-run.
		Display    *jsonSwitchDisplay `json:"display,omitempty"`
		Model      string             `json:"model,omitempty"`
		Input      string             `json:"input,omitempty"`
		InputLabel string             `json:"inputLabel,omitempty"`
		Operation  *jsonOperation     `json:"operation,omitempty"`
		// Command is the invocation, redacted unless --show-serial.
		Command string `json:"command,omitempty"`
		// Assumed reports that identification was bypassed with --unsafe-model.
		Assumed bool `json:"assumed"`
		// Refusal is present only when Outcome is refused.
		Refusal *jsonRefusal `json:"refusal,omitempty"`
		// Error is present only when Outcome is failed.
		Error string `json:"error,omitempty"`
	}

	// jsonSwitchDisplay is the display that was written to, or would have been.
	jsonSwitchDisplay struct {
		Label string `json:"label"`
		// Handle is masked when it is private data, unless --show-serial.
		Handle string `json:"handle"`
	}

	// jsonOperation is the mechanism and the value the catalog recorded.
	jsonOperation struct {
		Mechanism string `json:"mechanism"`
		// Value is the recorded value; ValueHex is the same value as the
		// documentation writes it.
		Value    uint16 `json:"value"`
		ValueHex string `json:"valueHex"`
	}

	// jsonRefusal is why monmux declined, with the identities it saw.
	jsonRefusal struct {
		Reason      string `json:"reason"`
		Explanation string `json:"explanation"`
		Detail      string `json:"detail"`
		// Detected are the identities the decision was made about, redacted
		// unless --show-serial.
		Detected []edid.Identity `json:"detected"`
	}
)

// asJSONSwitch converts one switch into the output contract.
//
// The pair it writes is the whole point of the document: the exit code cannot
// tell a dry run from a real send, and this can, so the pair is derived from
// what the outcome positively reports rather than from the absence of an error.
// A state app.Switch cannot produce degrades to failed/unknown, because
// "unknown" is the one write status that promises nothing.
func asJSONSwitch(
	outcome *app.Outcome,
	backendName string,
	err error,
	showSerial bool,
) jsonSwitch {
	document := jsonSwitch{
		Backend: backendName,
		// Assumed is what happened, not what was asked for: a run that failed
		// before the policy chose a display bypassed no identification, whatever
		// --unsafe-model was set to.
		Assumed: outcome.Assumed,
		Outcome: outcomeFailed,
		// A failure that reached here may have written: only a refusal, and a
		// dry run that never executed anything, may say otherwise.
		WriteStatus: writeStatusUnknown,
	}

	declined, refused := errors.AsType[*refusal.Refusal](err)

	switch {
	case err != nil && refused:
		document.Outcome = outcomeRefused
		document.WriteStatus = writeStatusNone
		document.Refusal = asJSONRefusal(declined, showSerial)
	case err != nil:
		document.Error = err.Error()
	case outcome.Sent:
		document.Outcome = outcomeSent
		document.WriteStatus = writeStatusSent

		fillJSONDecision(&document, outcome, showSerial)
	case outcome.DryRun:
		document.Outcome = outcomeDryRun
		document.WriteStatus = writeStatusNone

		fillJSONDecision(&document, outcome, showSerial)
	default:
		// Unreachable through app.Switch, which reports one or the other. The
		// document still has to be well formed, and a failed one carries its
		// text in error.
		document.Error = degradedOutcome
	}

	return document
}

// fillJSONDecision adds the fields that exist only once a display, a model and
// an operation have been settled on.
func fillJSONDecision(document *jsonSwitch, outcome *app.Outcome, showSerial bool) {
	document.Display = &jsonSwitchDisplay{
		Label:  outcome.Display.Label,
		Handle: handle(&outcome.Display, showSerial),
	}
	document.Model = outcome.Model.FullName()
	document.Input = outcome.Input.String()
	document.InputLabel = outcome.Input.Label()
	document.Operation = &jsonOperation{
		Mechanism: outcome.Operation.Mechanism().String(),
		Value:     outcome.Operation.Value(),
		ValueHex:  fmt.Sprintf("0x%02X", outcome.Operation.Value()),
	}
	document.Command = outcome.Command.Render(showSerial)
}

// asJSONRefusal converts a refusal into the output contract, carrying the same
// sentence the rendered message would have shown and redacting the identities
// unless the user asked for them.
func asJSONRefusal(declined *refusal.Refusal, showSerial bool) *jsonRefusal {
	document := &jsonRefusal{
		Reason:      declined.Reason.String(),
		Explanation: declined.Explanation(),
		Detail:      declined.Detail,
		Detected:    make([]edid.Identity, 0, len(declined.Detected)),
	}

	for _, identity := range declined.Detected {
		if !showSerial {
			identity = identity.Redacted()
		}

		document.Detected = append(document.Detected, identity)
	}

	return document
}
