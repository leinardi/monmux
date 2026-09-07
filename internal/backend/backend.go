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
	"slices"
	"strings"

	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/refusal"
)

// Command is one external tool invocation, exactly as it would be run.
//
// A Command is an output, never an input. [Backend.Plan] returns one so the CLI
// can show what a switch would do; no method on [Backend] accepts one. That is
// what makes it unforgeable: [Backend.Execute] is given a [catalog.Operation]
// and rebuilds the invocation through the same private planner Plan uses, so no
// caller can hand a backend an arbitrary executable or arguments, and what
// --dry-run prints is what a real run executes.
type Command struct {
	// Path is the absolute path of the tool, resolved during Preflight. It is
	// executed directly: there is no PATH lookup at execution time.
	Path string
	// Args are the arguments, excluding the program name.
	Args []string
	// Redact holds the indexes into Args that carry private data, such as the
	// raw EDID hex on Linux or the display UUID on macOS. Those arguments are
	// masked in output unless --show-serial was given.
	Redact []int
}

// Render returns the command as a single line. Arguments listed in Redact are
// masked unless showSerial is true.
func (c Command) Render(showSerial bool) string {
	parts := make([]string, 0, len(c.Args)+1)
	parts = append(parts, c.Path)

	for index, arg := range c.Args {
		if !showSerial && c.isRedacted(index) {
			parts = append(parts, refusal.RedactionMask)

			continue
		}

		parts = append(parts, arg)
	}

	return strings.Join(parts, " ")
}

// String renders the command with private arguments masked.
func (c Command) String() string {
	return c.Render(false)
}

// isRedacted reports whether the argument at index carries private data.
func (c Command) isRedacted(index int) bool {
	return slices.Contains(c.Redact, index)
}

// Check is one diagnostic result reported by [Backend.Doctor].
type Check struct {
	// Name is what was checked, e.g. "ddcutil version".
	Name string
	// OK reports whether the check passed.
	OK bool
	// Detail explains the result, and is safe to print.
	Detail string
}

// Backend is a monitor-control backend: a wrapper around one external tool.
//
// The split between Plan and Execute is deliberate. Plan has no side effects and
// exists so a user can see the exact invocation before anything happens. Execute
// takes the operation rather than the plan, re-verifies that the display is
// still the one that was identified, and runs the tool exactly once. Nothing in
// this interface accepts a [Command].
type Backend interface {
	// Name is the backend's name, e.g. "ddcutil".
	Name() string

	// Preflight checks the tool itself: that the binary is present, trusted,
	// recent enough, and supports the options monmux needs. It resolves the
	// absolute path used by every later call. Failures are refusals with
	// reason backend-not-ready.
	Preflight(ctx context.Context) error

	// Enumerate lists the attached displays, including ones that cannot be
	// written to; those come back with Writable false and a Status saying why.
	Enumerate(ctx context.Context) ([]Display, error)

	// Ready checks the target itself rather than the tool: whether this
	// particular display can be reached right now, for example whether the
	// user may open its /dev/i2c-N. Failures are refusals with reason
	// target-not-ready.
	Ready(ctx context.Context, display Display) error

	// Plan returns the invocation Execute would run, without running it and
	// without any other side effect. It rejects an invalid operation and any
	// mechanism this backend does not implement, with invalid-operation.
	Plan(display Display, operation catalog.Operation) (Command, error)

	// Execute performs the switch: it validates the operation, re-verifies the
	// display's identity, rebuilds the invocation through the same planner Plan
	// uses, and runs it once. It never retries.
	Execute(ctx context.Context, display Display, operation catalog.Operation) error

	// Doctor reports what the backend can see, read-only.
	Doctor(ctx context.Context) []Check
}

// ValidateOperation is the check every backend performs before planning or
// executing: the operation must have come from the catalog, and its mechanism
// must be one this backend implements. There is no fallback to another
// mechanism, another VCP code or another value.
func ValidateOperation(operation catalog.Operation, implemented ...catalog.Mechanism) error {
	if !operation.Valid() {
		return refusal.New(refusal.InvalidOperation, "The operation did not come from the catalog.")
	}

	if slices.Contains(implemented, operation.Mechanism()) {
		return nil
	}

	return refusal.New(
		refusal.InvalidOperation,
		"Mechanism "+operation.Mechanism().String()+" is not implemented by this backend.",
	)
}
