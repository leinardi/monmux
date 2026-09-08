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

package backend

import (
	"context"
	"sync"

	"github.com/leinardi/monmux/internal/catalog"
)

// The names of the calls a [Fake] records, in the order a switch makes them.
const (
	CallPreflight = "preflight"
	CallEnumerate = "enumerate"
	CallReady     = "ready"
	CallPlan      = "plan"
	CallExecute   = "execute"
	CallDoctor    = "doctor"
)

// Fake is a [Backend] that records what it was asked to do and answers from a
// script. It touches no monitor and runs no program at all, which is what makes
// it the only backend the orchestration and CLI tests use.
//
// It lives here rather than in a test file because more than one package needs
// it, exactly as [github.com/leinardi/monmux/internal/backend/exec.Fake] does.
type Fake struct {
	// FakeName is what Name reports.
	FakeName string
	// Displays is what Enumerate reports.
	Displays []Display
	// Checks is what Doctor reports.
	Checks []Check
	// Planned, when its Path is set, is what Plan returns. Otherwise Plan
	// builds a command from the display and the operation.
	Planned Command

	// The errors each method returns instead of succeeding.
	PreflightErr error
	EnumerateErr error
	ReadyErr     error
	PlanErr      error
	ExecuteErr   error

	mu       sync.Mutex
	calls    []string
	executed []catalog.Operation
	targets  []Display
}

// Name returns the backend's name.
func (f *Fake) Name() string {
	if f.FakeName == "" {
		return "fake"
	}

	return f.FakeName
}

// Preflight records the call and returns the scripted error.
func (f *Fake) Preflight(_ context.Context) error {
	f.record(CallPreflight)

	return f.PreflightErr
}

// Enumerate records the call and returns the scripted displays.
func (f *Fake) Enumerate(_ context.Context) ([]Display, error) {
	f.record(CallEnumerate)

	if f.EnumerateErr != nil {
		return nil, f.EnumerateErr
	}

	return append([]Display(nil), f.Displays...), nil
}

// Ready records the call and returns the scripted error.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (f *Fake) Ready(_ context.Context, _ Display) error {
	f.record(CallReady)

	return f.ReadyErr
}

// Plan records the call and returns the scripted command. It validates the
// operation exactly as a real backend does, so a caller that hands it something
// the catalog did not produce is refused here too. The fake stands in for a
// backend that implements everything, so it validates against the whole enum
// rather than a list of its own: a test driving any catalog entry through it
// still gets the forged-operation refusal, and adding a mechanism needs no edit
// here.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (f *Fake) Plan(display Display, operation catalog.Operation) (Command, error) {
	f.record(CallPlan)

	err := ValidateOperation(operation, catalog.Mechanisms()...)
	if err != nil {
		return Command{}, err
	}

	if f.PlanErr != nil {
		return Command{}, f.PlanErr
	}

	return f.command(&display, operation), nil
}

// Execute records the call, the operation it was given, and returns the
// scripted error.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (f *Fake) Execute(_ context.Context, display Display, operation catalog.Operation) error {
	err := ValidateOperation(operation, catalog.Mechanisms()...)
	if err != nil {
		f.record(CallExecute)

		return err
	}

	f.mu.Lock()
	f.calls = append(f.calls, CallExecute)
	f.executed = append(f.executed, operation)
	f.targets = append(f.targets, display)
	f.mu.Unlock()

	return f.ExecuteErr
}

// Doctor records the call and returns the scripted checks.
func (f *Fake) Doctor(_ context.Context) []Check {
	f.record(CallDoctor)

	return append([]Check(nil), f.Checks...)
}

// Calls returns the methods that were called, in order.
func (f *Fake) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]string(nil), f.calls...)
}

// Executed returns the operations Execute was given. An empty result means no
// write was even attempted, which is what most of these tests are about.
func (f *Fake) Executed() []catalog.Operation {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]catalog.Operation(nil), f.executed...)
}

// Targets returns the displays Execute was asked to write to.
func (f *Fake) Targets() []Display {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]Display(nil), f.targets...)
}

// command builds a plausible invocation for a display and an operation.
func (f *Fake) command(display *Display, operation catalog.Operation) Command {
	if f.Planned.Path != "" {
		return f.Planned
	}

	return Command{
		Path:   "/usr/bin/" + f.Name(),
		Args:   []string{display.Handle, operation.String()},
		Redact: []int{0},
	}
}

// record notes one call.
func (f *Fake) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, call)
}
