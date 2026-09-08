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

package backend_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

func planned() backend.Command {
	return backend.Command{
		Path:   "/usr/bin/ddcutil",
		Args:   []string{"--edid", "00ffffffffffff00", "setvcp", "0xF4", "0xD1"},
		Redact: []int{1},
	}
}

func TestCommandMasksPrivateArgumentsByDefault(t *testing.T) {
	t.Parallel()

	rendered := planned().String()

	if strings.Contains(rendered, "00ffffffffffff00") {
		t.Errorf("the raw EDID was printed:\n%s", rendered)
	}

	if !strings.Contains(rendered, refusal.RedactionMask) {
		t.Errorf("the redacted argument is not marked as such:\n%s", rendered)
	}

	if !strings.Contains(rendered, "setvcp 0xF4 0xD1") {
		t.Errorf("redaction swallowed the rest of the command:\n%s", rendered)
	}
}

func TestCommandPrintsEverythingWhenAsked(t *testing.T) {
	t.Parallel()

	rendered := planned().Render(true)

	want := "/usr/bin/ddcutil --edid 00ffffffffffff00 setvcp 0xF4 0xD1"
	if rendered != want {
		t.Errorf("Render(true) =\n%s\nwant\n%s", rendered, want)
	}
}

// Command is an output of Plan, never an input. Nothing in the interface may
// accept one, or a caller could hand a backend an executable of its choosing.
func TestNoBackendMethodAcceptsACommand(t *testing.T) {
	t.Parallel()

	backendType := reflect.TypeFor[backend.Backend]()
	commandType := reflect.TypeFor[backend.Command]()

	for method := range backendType.Methods() {
		for argument := range method.Type.NumIn() {
			parameter := method.Type.In(argument)
			if reaches(parameter, commandType, map[reflect.Type]bool{}) {
				t.Errorf(
					"%s can be handed a Command through argument %d (%s)",
					method.Name,
					argument,
					parameter,
				)
			}
		}
	}
}

// reaches reports whether a value of type from can carry a value of type target,
// however deeply it is wrapped. A parameter that merely contains a Command would
// defeat the invariant just as surely as one that is a Command.
func reaches(from, target reflect.Type, seen map[reflect.Type]bool) bool {
	if from == target {
		return true
	}

	if seen[from] {
		return false
	}

	seen[from] = true

	kind := from.Kind()

	if kind == reflect.Pointer || kind == reflect.Slice || kind == reflect.Array ||
		kind == reflect.Chan {
		return reaches(from.Elem(), target, seen)
	}

	if kind == reflect.Map {
		return reaches(from.Key(), target, seen) || reaches(from.Elem(), target, seen)
	}

	if kind == reflect.Struct {
		for field := range from.Fields() {
			if reaches(field.Type, target, seen) {
				return true
			}
		}

		return false
	}

	if kind == reflect.Func {
		for in := range from.Ins() {
			if reaches(in, target, seen) {
				return true
			}
		}

		return false
	}

	return false
}

// The walk is the thing actually enforcing the invariant, so it is itself tested
// against types that do carry a Command and types that do not.
func TestReachesFindsAWrappedCommand(t *testing.T) {
	t.Parallel()

	commandType := reflect.TypeFor[backend.Command]()

	type wrapper struct {
		Commands []backend.Command
	}

	carriers := []any{
		backend.Command{},
		&backend.Command{},
		[]backend.Command{},
		map[string]backend.Command{},
		wrapper{},
		func(backend.Command) {},
	}

	for _, carrier := range carriers {
		carrierType := reflect.TypeOf(carrier)
		if !reaches(carrierType, commandType, map[reflect.Type]bool{}) {
			t.Errorf("%s carries a Command but the walk missed it", carrierType)
		}
	}

	innocents := []any{"", 0, []string{}, struct{ Path string }{}}

	for _, innocent := range innocents {
		innocentType := reflect.TypeOf(innocent)
		if reaches(innocentType, commandType, map[reflect.Type]bool{}) {
			t.Errorf("%s carries no Command but the walk claimed it does", innocentType)
		}
	}
}

// Execute must be given the operation, not the plan, so that it rebuilds the
// invocation itself.
func TestExecuteTakesAnOperation(t *testing.T) {
	t.Parallel()

	backendType := reflect.TypeFor[backend.Backend]()
	operationType := reflect.TypeFor[catalog.Operation]()

	method, found := backendType.MethodByName("Execute")
	if !found {
		t.Fatal("Backend has no Execute method")
	}

	takesOperation := false

	for in := range method.Type.Ins() {
		if in == operationType {
			takesOperation = true
		}
	}

	if !takesOperation {
		t.Error("Execute does not take a catalog.Operation")
	}
}

func TestValidateOperationRejectsTheZeroValue(t *testing.T) {
	t.Parallel()

	var zero catalog.Operation

	err := backend.ValidateOperation(zero, catalog.MechanismLGAltInput)
	if !refusal.Is(err, refusal.InvalidOperation) {
		t.Errorf("error = %v, want invalid-operation", err)
	}
}

func TestValidateOperationRejectsAnUnimplementedMechanism(t *testing.T) {
	t.Parallel()

	model, result := catalog.Match(edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D3})
	if result != catalog.MatchExact {
		t.Fatalf("the tested model no longer matches: %s", result)
	}

	operation, enabled := model.Operation(catalog.Input("dp"))
	if !enabled {
		t.Fatal("the tested model is no longer enabled for DisplayPort")
	}

	err := backend.ValidateOperation(operation, catalog.Mechanism("something-else"))
	if !refusal.Is(err, refusal.InvalidOperation) {
		t.Errorf("error = %v, want invalid-operation", err)
	}

	err = backend.ValidateOperation(operation, catalog.MechanismLGAltInput)
	if err != nil {
		t.Errorf("a catalog operation with an implemented mechanism was rejected: %v", err)
	}
}
