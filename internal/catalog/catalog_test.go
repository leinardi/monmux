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
	"errors"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
)

// --- invariants: these guard the catalog itself, not the lookup code ---

func TestWriteEnabledModelsHaveAtLeastOneIdentity(t *testing.T) {
	t.Parallel()

	for _, model := range catalog.Models() {
		if model.WriteEnabled && len(model.Identities) == 0 {
			t.Errorf("%s is write-enabled but has no EDID identity to match on", model.FullName())
		}
	}
}

func TestIdentitiesAreUniqueAcrossModels(t *testing.T) {
	t.Parallel()

	owner := map[catalog.Identity]string{}

	for _, model := range catalog.Models() {
		for _, identity := range model.Identities {
			previous, taken := owner[identity]
			if taken {
				t.Errorf("%s is claimed by both %s and %s", identity, previous, model.FullName())
			}

			owner[identity] = model.FullName()
		}
	}
}

func TestEveryInputOfAWriteEnabledModelHasEvidence(t *testing.T) {
	t.Parallel()

	for _, model := range catalog.Models() {
		if !model.WriteEnabled {
			continue
		}

		for _, input := range model.RecordedInputs() {
			evidence, ok := model.InputEvidence(input)
			if !ok || strings.TrimSpace(evidence) == "" {
				t.Errorf("%s enables %s with no evidence", model.FullName(), input)
			}
		}
	}
}

func TestDisabledModelsNeverMatch(t *testing.T) {
	t.Parallel()

	for _, model := range catalog.Models() {
		if model.WriteEnabled {
			continue
		}

		for _, identity := range model.Identities {
			probe := edid.Identity{
				Manufacturer: identity.Manufacturer,
				ProductCode:  identity.ProductCode,
			}

			_, result := catalog.Match(probe)
			if result != catalog.MatchNone {
				t.Errorf(
					"%s is not write-enabled but %s matched it (%s)",
					model.FullName(),
					identity,
					result,
				)
			}
		}
	}
}

func TestDisabledModelsEnableNoInput(t *testing.T) {
	t.Parallel()

	for _, model := range catalog.Models() {
		if model.WriteEnabled {
			continue
		}

		if len(model.EnabledInputs()) != 0 {
			t.Errorf(
				"%s is not write-enabled but enables %v",
				model.FullName(),
				model.EnabledInputs(),
			)
		}

		for _, input := range model.RecordedInputs() {
			_, ok := model.Operation(input)
			if ok {
				t.Errorf(
					"%s is not write-enabled but yielded an operation for %s",
					model.FullName(),
					input,
				)
			}
		}
	}
}

func TestZeroOperationIsInvalid(t *testing.T) {
	t.Parallel()

	var zero catalog.Operation

	if zero.Valid() {
		t.Error("the zero Operation reports itself valid")
	}

	if zero.Value() != 0 || zero.Mechanism() != "" {
		t.Errorf("the zero Operation carries data: %+v", zero)
	}

	if zero.String() != "invalid operation" {
		t.Errorf("String() = %q, want %q", zero.String(), "invalid operation")
	}
}

func TestEveryRecordedInputIsInTheEnum(t *testing.T) {
	t.Parallel()

	known := map[catalog.Input]bool{}
	for _, input := range catalog.Inputs() {
		known[input] = true
	}

	for _, model := range catalog.Models() {
		for input := range model.Inputs {
			if !known[input] {
				t.Errorf("%s records the unknown input %q", model.FullName(), input)
			}
		}
	}
}

func TestEveryRecordedInputUsesAnImplementedMechanism(t *testing.T) {
	t.Parallel()

	implemented := map[catalog.Mechanism]bool{}
	for _, mechanism := range catalog.Mechanisms() {
		implemented[mechanism] = true
	}

	if len(implemented) == 0 {
		t.Fatal("Mechanisms() is empty, so nothing can ever be switched")
	}

	for _, model := range catalog.Models() {
		for _, input := range model.RecordedInputs() {
			mechanism, ok := model.InputMechanism(input)
			if !ok || !implemented[mechanism] || !mechanism.Known() {
				t.Errorf(
					"%s uses mechanism %q for %s, which no backend implements",
					model.FullName(),
					mechanism,
					input,
				)
			}
		}
	}
}

// The catalog is the one place the bytes are decided, so what Models() and
// Match() hand out must not be a way back into it.
func TestReturnedModelsCannotMutateTheCatalog(t *testing.T) {
	t.Parallel()

	tested := edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D3}

	before, result := catalog.Match(tested)
	if result != catalog.MatchExact {
		t.Fatalf("GSM/0x77D3 match = %s, want exact", result)
	}

	handedOut := catalog.Models()

	var enabled, disabled *catalog.Model

	for index := range handedOut {
		if handedOut[index].WriteEnabled {
			enabled = &handedOut[index]
		} else {
			disabled = &handedOut[index]
		}
	}

	if enabled == nil || disabled == nil {
		t.Fatal("the catalog no longer has both a write-enabled and a disabled entry")
	}

	// Repoint an identity, graft an unverified input from the disabled entry
	// onto the write-enabled one, drop a verified input, and rewrite a source.
	enabled.Identities[0].ProductCode = 0x1234
	enabled.Inputs[catalog.InputHDMI1] = disabled.Inputs[catalog.InputHDMI1]
	delete(enabled.Inputs, catalog.InputDP)
	enabled.Sources[0] = "invented"

	after, result := catalog.Match(tested)
	if result != catalog.MatchExact {
		t.Fatalf("after mutation, GSM/0x77D3 match = %s, want exact", result)
	}

	_, hdmiEnabled := after.Operation(catalog.InputHDMI1)
	if hdmiEnabled {
		t.Error("an unverified HDMI value was grafted onto the write-enabled model")
	}

	op, dpEnabled := after.Operation(catalog.InputDP)
	if !dpEnabled || op.Value() != 0xD0 {
		t.Errorf("the verified DisplayPort value was lost: enabled=%t op=%s", dpEnabled, op)
	}

	if after.Identities[0] != before.Identities[0] {
		t.Errorf(
			"an identity was repointed: %s, want %s",
			after.Identities[0],
			before.Identities[0],
		)
	}

	if after.Sources[0] != before.Sources[0] {
		t.Errorf("a source was rewritten: %q", after.Sources[0])
	}
}

// --- the tested unit: the values here are what actually reaches the monitor ---

func TestTestedModelIsWriteEnabledForItsTwoVerifiedInputs(t *testing.T) {
	t.Parallel()

	model, result := catalog.Match(edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D3})
	if result != catalog.MatchExact {
		t.Fatalf("GSM/0x77D3 match = %s, want exact", result)
	}

	if model.FullName() != "LG 38WR85QC-W" {
		t.Fatalf("matched %s, want LG 38WR85QC-W", model.FullName())
	}

	want := map[catalog.Input]uint8{catalog.InputDP: 0xD0, catalog.InputUSBC: 0xD1}

	for input, value := range want {
		op, ok := model.Operation(input)
		if !ok {
			t.Fatalf("%s is not enabled for %s", model.FullName(), input)
		}

		if !op.Valid() {
			t.Errorf("%s yielded an invalid operation for %s", model.FullName(), input)
		}

		if op.Value() != value {
			t.Errorf("%s value = 0x%02X, want 0x%02X", input, op.Value(), value)
		}

		if op.Mechanism() != catalog.MechanismLGAltInput {
			t.Errorf(
				"%s mechanism = %q, want %q",
				input,
				op.Mechanism(),
				catalog.MechanismLGAltInput,
			)
		}
	}
}

func TestBothIdentitiesOfTheTestedModelMatchIt(t *testing.T) {
	t.Parallel()

	for _, code := range []uint16{0x77D3, 0x77D4} {
		model, result := catalog.Match(edid.Identity{Manufacturer: "GSM", ProductCode: code})
		if result != catalog.MatchExact {
			t.Errorf("GSM/0x%04X match = %s, want exact", code, result)

			continue
		}

		if model.Name != "38WR85QC-W" {
			t.Errorf("GSM/0x%04X matched %s, want 38WR85QC-W", code, model.FullName())
		}
	}
}

func TestUntestedInputsAreNotEnabled(t *testing.T) {
	t.Parallel()

	model, _ := catalog.Match(edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D3})

	for _, input := range []catalog.Input{catalog.InputHDMI1, catalog.InputHDMI2} {
		_, ok := model.Operation(input)
		if ok {
			t.Errorf(
				"%s is enabled for %s, which was never tested on this unit",
				model.FullName(),
				input,
			)
		}
	}
}

// --- lookup ---

func TestMatchReportsUnknownMonitors(t *testing.T) {
	t.Parallel()

	unknown := []edid.Identity{
		{Manufacturer: "GSM", ProductCode: 0x0001},
		{Manufacturer: "DEL", ProductCode: 0x77D3},
		{},
	}

	for _, identity := range unknown {
		model, result := catalog.Match(identity)
		if result != catalog.MatchNone {
			t.Errorf(
				"%s/0x%04X match = %s, want none",
				identity.Manufacturer,
				identity.ProductCode,
				result,
			)
		}

		if model.Name != "" {
			t.Errorf("a failed match returned %s", model.FullName())
		}
	}
}

func TestMatchIgnoresSerialsAndModelName(t *testing.T) {
	t.Parallel()

	identity := edid.Identity{
		Manufacturer: "GSM",
		ProductCode:  0x77D3,
		SerialNumber: 0x01020304,
		SerialString: "TESTSERIAL01",
		ModelName:    "LG ULTRAWIDE",
	}

	_, withSerials := catalog.Match(identity)

	_, redacted := catalog.Match(identity.Redacted())

	if withSerials != redacted || withSerials != catalog.MatchExact {
		t.Errorf("match depends on the serial fields: %s vs %s", withSerials, redacted)
	}
}

// --- the input enum ---

func TestParseInputAcceptsExactlyTheEnum(t *testing.T) {
	t.Parallel()

	for _, input := range catalog.Inputs() {
		parsed, err := catalog.ParseInput(input.String())
		if err != nil {
			t.Errorf("ParseInput(%q) returned %v", input, err)
		}

		if parsed != input {
			t.Errorf("ParseInput(%q) = %q", input, parsed)
		}
	}

	for _, name := range []string{"", "DP", "displayport", "hdmi", "usbc", "0xD0"} {
		_, err := catalog.ParseInput(name)
		if !errors.Is(err, catalog.ErrUnknownInput) {
			t.Errorf("ParseInput(%q) error = %v, want ErrUnknownInput", name, err)
		}
	}
}

func TestEveryInputHasItsOwnLabel(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}

	for _, input := range catalog.Inputs() {
		label := input.Label()
		if label == "" || label == input.String() {
			t.Errorf("%q has no human-readable label", input)
		}

		if seen[label] {
			t.Errorf("label %q is used twice", label)
		}

		seen[label] = true
	}
}
