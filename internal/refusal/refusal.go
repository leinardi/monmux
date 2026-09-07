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

// Package refusal holds the one typed error every monmux layer returns when it
// declines to write to a monitor. There is exactly one such type so that the CLI
// has exactly one thing to recognize: it maps a [Refusal] to exit code 2 and
// prints it with the template from requirement 9.14.
//
// Every rendered refusal ends with "No DDC write was performed." and prints
// identities redacted unless the user explicitly asked for serials.
package refusal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/leinardi/monmux/internal/edid"
)

// Reason is the closed set of reasons monmux refuses to write.
type Reason string

const (
	// BackendUnavailable means this operating system has no backend at all.
	BackendUnavailable Reason = "backend-unavailable"
	// BackendNotReady means the backend tool is missing, too old, or untrusted.
	BackendNotReady Reason = "backend-not-ready"
	// EnumerationFailed means the attached displays could not be listed.
	EnumerationFailed Reason = "enumeration-failed"
	// NoDisplays means enumeration succeeded and found nothing to act on.
	NoDisplays Reason = "no-displays"
	// DisplayNotWritable means the only supported display cannot be written to.
	DisplayNotWritable Reason = "display-not-writable"
	// UnknownMonitor means no catalog entry matches the detected identity.
	UnknownMonitor Reason = "unknown-monitor"
	// AmbiguousCatalog means more than one catalog entry claims the identity.
	AmbiguousCatalog Reason = "ambiguous-catalog"
	// MultipleCandidates means more than one attached display is supported.
	MultipleCandidates Reason = "multiple-candidates"
	// InputNotEnabled means the model has no evidence for the requested input.
	InputNotEnabled Reason = "input-not-enabled"
	// SerialMismatch means no attached display matches the pinned serial.
	SerialMismatch Reason = "serial-mismatch"
	// TargetNotReady means the target's write channel is not usable.
	TargetNotReady Reason = "target-not-ready"
	// IdentityChanged means the display moved between identification and write.
	IdentityChanged Reason = "identity-changed"
	// InvalidOperation means the operation is not one this backend implements.
	InvalidOperation Reason = "invalid-operation"
)

// String returns the reason as it appears in messages and machine-readable output.
func (r Reason) String() string {
	return string(r)
}

// RedactionMask replaces private data in output when --show-serial was not given.
const RedactionMask = "<redacted; --show-serial to print>"

const (
	// headline opens every refusal.
	headline = "Refusing to switch input."
	// footer closes every refusal. Its presence is the safety property.
	footer = "No DDC write was performed."
	// unknownReason is used if a reason ever reaches here without a sentence.
	unknownReason = "The operation was refused."
	// indent prefixes the identity field lines.
	indent = "  "
	// fieldWidth aligns the identity field labels.
	fieldWidth = 14
)

// explanations maps each reason to the sentence shown to the user. A map rather
// than a switch: an unmapped reason degrades to [unknownReason] instead of
// printing nothing at all.
var explanations = map[Reason]string{
	BackendUnavailable: "No monitor-control backend is available for this operating system.",
	BackendNotReady:    "The backend tool is not usable.",
	EnumerationFailed:  "The displays attached to this system could not be enumerated.",
	NoDisplays:         "No display was detected.",
	DisplayNotWritable: "The supported display cannot be written to.",
	UnknownMonitor:     "No supported-model catalog entry matches this identity.",
	AmbiguousCatalog:   "More than one catalog entry claims this identity, so the match is not trustworthy.",
	MultipleCandidates: "More than one attached display matches a supported model; pin one with --serial.",
	InputNotEnabled:    "The requested input is not enabled for this model.",
	SerialMismatch:     "No attached display matches the pinned serial.",
	TargetNotReady:     "The target display cannot be reached over DDC right now.",
	IdentityChanged:    "The display changed between identification and the write.",
	InvalidOperation:   "The requested operation is not one this backend implements.",
}

// Refusal is the typed error returned by every layer that declines to write.
// It is a refusal, not a failure: it states positively that nothing was sent.
//
//nolint:errname // named for what it is, not for the XxxError convention
type Refusal struct {
	// Reason is why the write was refused.
	Reason Reason
	// Detected are the identities involved, if any were read before refusing.
	Detected []edid.Identity
	// Detail adds case-specific context, e.g. which input was requested.
	Detail string
}

// New builds a refusal. The variadic identities are the displays the decision was
// made about, and may be empty when the refusal happened before enumeration.
func New(reason Reason, detail string, detected ...edid.Identity) *Refusal {
	return &Refusal{Reason: reason, Detected: detected, Detail: detail}
}

// Error renders the refusal with serials redacted. This is what ends up in logs
// and in any caller that treats a refusal as a plain error, so it is the safe
// form by construction: the verbose form has to be asked for by name.
func (r *Refusal) Error() string {
	return r.Render(false)
}

// Render returns the full refusal message. Serial numbers and serial strings are
// printed only when showSerial is true; otherwise they are masked.
func (r *Refusal) Render(showSerial bool) string {
	lines := []string{headline}

	if len(r.Detected) > 0 {
		lines = append(lines, "", detectedHeader(len(r.Detected)))

		for index, identity := range r.Detected {
			if index > 0 {
				lines = append(lines, "")
			}

			lines = append(lines, identityLines(identity, showSerial)...)
		}
	}

	lines = append(lines, "", r.explanation())

	if r.Detail != "" {
		lines = append(lines, r.Detail)
	}

	lines = append(lines, footer)

	return strings.Join(lines, "\n")
}

// explanation returns the sentence for this refusal's reason.
func (r *Refusal) explanation() string {
	explanation, ok := explanations[r.Reason]
	if !ok {
		return unknownReason
	}

	return explanation
}

// detectedHeader returns the singular or plural header for the identity block.
func detectedHeader(count int) string {
	if count == 1 {
		return "Detected monitor:"
	}

	return "Detected monitors:"
}

// identityLines renders one identity as indented, aligned label/value lines.
// Fields the EDID did not carry are omitted; the serial line is always present,
// because "redacted" and "the monitor exposes none" are different facts.
func identityLines(identity edid.Identity, showSerial bool) []string {
	hasSerial := identity.HasSerial()
	if !showSerial {
		identity = identity.Redacted()
	}

	lines := []string{
		field("Manufacturer", identity.Manufacturer),
		field("Product ID", fmt.Sprintf("0x%04X", identity.ProductCode)),
	}

	if identity.ModelName != "" {
		lines = append(lines, field("Model name", identity.ModelName))
	}

	return append(lines, field("Serial", serialValue(identity, hasSerial, showSerial)))
}

// serialValue renders the serial line: the value when it was asked for, the mask
// when it was withheld, and "(none)" when the monitor exposes no serial at all.
func serialValue(identity edid.Identity, hasSerial, showSerial bool) string {
	if !hasSerial {
		return "(none)"
	}

	if !showSerial {
		return RedactionMask
	}

	if identity.SerialString == "" {
		return fmt.Sprintf("0x%08X", identity.SerialNumber)
	}

	return fmt.Sprintf("%s (0x%08X)", identity.SerialString, identity.SerialNumber)
}

// field renders one indented, aligned "Label: value" line.
func field(label, value string) string {
	return fmt.Sprintf("%s%-*s%s", indent, fieldWidth, label+":", value)
}

// Is reports whether err is a refusal with the given reason. The CLI uses it to
// pick an exit code; tests use it to assert the refusal path that was taken.
func Is(err error, reason Reason) bool {
	var refusal *Refusal
	if !errors.As(err, &refusal) {
		return false
	}

	return refusal.Reason == reason
}
