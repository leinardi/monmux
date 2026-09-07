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

package refusal_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

const (
	serialString = "TESTSERIAL01"
	serialNumber = 0x01020304
)

func detected() edid.Identity {
	return edid.Identity{
		Manufacturer: "GSM",
		ProductCode:  0x77D3,
		SerialNumber: serialNumber,
		SerialString: serialString,
		ModelName:    "LG ULTRAWIDE",
	}
}

// allReasons is every reason in the enum. Adding one without a sentence should
// fail TestEveryReasonHasItsOwnExplanation.
func allReasons() []refusal.Reason {
	return []refusal.Reason{
		refusal.BackendUnavailable,
		refusal.BackendNotReady,
		refusal.EnumerationFailed,
		refusal.NoDisplays,
		refusal.DisplayNotWritable,
		refusal.UnknownMonitor,
		refusal.AmbiguousCatalog,
		refusal.MultipleCandidates,
		refusal.InputNotEnabled,
		refusal.SerialMismatch,
		refusal.TargetNotReady,
		refusal.IdentityChanged,
		refusal.InvalidOperation,
	}
}

func TestEveryReasonHasItsOwnExplanation(t *testing.T) {
	t.Parallel()

	seen := make(map[string]refusal.Reason, len(allReasons()))

	for _, reason := range allReasons() {
		rendered := refusal.New(reason, "").Error()

		lines := strings.Split(rendered, "\n")
		if len(lines) < 2 {
			t.Fatalf("%s rendered too few lines: %q", reason, rendered)
		}

		explanation := lines[len(lines)-2]
		if explanation == "" || explanation == "The operation was refused." {
			t.Errorf("%s has no explanation of its own: %q", reason, explanation)
		}

		other, duplicate := seen[explanation]
		if duplicate {
			t.Errorf("%s and %s share the explanation %q", reason, other, explanation)
		}

		seen[explanation] = reason
	}
}

func TestEveryRefusalStatesNoWriteHappened(t *testing.T) {
	t.Parallel()

	for _, reason := range allReasons() {
		err := refusal.New(reason, "some detail", detected())

		for _, rendered := range []string{err.Error(), err.Render(true), err.Render(false)} {
			if !strings.HasSuffix(rendered, "No DDC write was performed.") {
				t.Errorf("%s does not end with the no-write footer:\n%s", reason, rendered)
			}

			if !strings.HasPrefix(rendered, "Refusing to switch input.") {
				t.Errorf("%s does not open with the headline:\n%s", reason, rendered)
			}
		}
	}
}

func TestErrorRedactsBothSerials(t *testing.T) {
	t.Parallel()

	err := refusal.New(refusal.UnknownMonitor, "", detected())

	rendered := err.Error()

	if strings.Contains(rendered, serialString) {
		t.Errorf("Error() leaked the serial string:\n%s", rendered)
	}

	if strings.Contains(rendered, fmt.Sprintf("0x%08X", serialNumber)) {
		t.Errorf("Error() leaked the serial number:\n%s", rendered)
	}

	if !strings.Contains(rendered, refusal.RedactionMask) {
		t.Errorf("Error() does not mark the serial as redacted:\n%s", rendered)
	}

	if !strings.Contains(rendered, "GSM") || !strings.Contains(rendered, "0x77D3") {
		t.Errorf("Error() dropped the matching fields:\n%s", rendered)
	}
}

func TestRenderWithShowSerialPrintsBothSerials(t *testing.T) {
	t.Parallel()

	rendered := refusal.New(refusal.SerialMismatch, "", detected()).Render(true)

	if !strings.Contains(rendered, serialString) {
		t.Errorf("Render(true) did not print the serial string:\n%s", rendered)
	}

	if !strings.Contains(rendered, fmt.Sprintf("0x%08X", serialNumber)) {
		t.Errorf("Render(true) did not print the serial number:\n%s", rendered)
	}

	if strings.Contains(rendered, refusal.RedactionMask) {
		t.Errorf("Render(true) still masked something:\n%s", rendered)
	}
}

func TestRenderSaysWhenThereIsNoSerialToShow(t *testing.T) {
	t.Parallel()

	identity := detected()
	identity.SerialNumber = 0
	identity.SerialString = ""

	rendered := refusal.New(refusal.SerialMismatch, "", identity).Render(false)

	if !strings.Contains(rendered, "(none)") {
		t.Errorf("a display without serials should say so:\n%s", rendered)
	}

	if strings.Contains(rendered, refusal.RedactionMask) {
		t.Errorf("nothing was withheld, so nothing should be masked:\n%s", rendered)
	}
}

func TestRenderTemplateShape(t *testing.T) {
	t.Parallel()

	err := refusal.New(refusal.UnknownMonitor, "Requested input: usb-c.", detected())

	want := strings.Join([]string{
		"Refusing to switch input.",
		"",
		"Detected monitor:",
		"  Manufacturer: GSM",
		"  Product ID:   0x77D3",
		"  Model name:   LG ULTRAWIDE",
		"  Serial:       " + refusal.RedactionMask,
		"",
		"No supported-model catalog entry matches this identity.",
		"Requested input: usb-c.",
		"No DDC write was performed.",
	}, "\n")

	if err.Error() != want {
		t.Errorf("Error() =\n%s\n\nwant\n%s", err.Error(), want)
	}
}

func TestRenderWithoutIdentitiesOmitsTheBlock(t *testing.T) {
	t.Parallel()

	rendered := refusal.New(refusal.NoDisplays, "").Error()

	if strings.Contains(rendered, "Detected monitor") {
		t.Errorf("nothing was detected, so no identity block should appear:\n%s", rendered)
	}
}

func TestRenderListsEveryDetectedMonitor(t *testing.T) {
	t.Parallel()

	second := detected()
	second.ProductCode = 0x77D4

	rendered := refusal.New(refusal.MultipleCandidates, "", detected(), second).Error()

	if !strings.Contains(rendered, "Detected monitors:") {
		t.Errorf("two monitors should use the plural header:\n%s", rendered)
	}

	if !strings.Contains(rendered, "0x77D3") || !strings.Contains(rendered, "0x77D4") {
		t.Errorf("both identities should be listed:\n%s", rendered)
	}
}

func TestIsMatchesTheWrappedReason(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("resolving target: %w", refusal.New(refusal.InputNotEnabled, ""))

	if !refusal.Is(err, refusal.InputNotEnabled) {
		t.Error("Is() did not see through the wrapping")
	}

	if refusal.Is(err, refusal.UnknownMonitor) {
		t.Error("Is() matched the wrong reason")
	}

	//nolint:err113 // a plain dynamic error is exactly what this case feeds in
	unrelated := errors.New("unrelated")
	if refusal.Is(unrelated, refusal.InputNotEnabled) {
		t.Error("Is() matched a non-refusal error")
	}
}
