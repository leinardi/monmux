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

// Package app is the orchestration layer: it runs the steps a switch is made of,
// in the one order they are allowed to happen in, and hands the result to the
// CLI to print. It performs no I/O of its own and knows nothing about ddcutil,
// m1ddc, sysfs or the IORegistry - it asks a [backend.Backend].
//
// The order is the design. The tool is checked, the displays are enumerated, the
// pure policy layer decides which display and which value, the target is checked
// for reachability, and only then is a command built - by the backend, from the
// catalog's operation, so that what --dry-run prints and what a real run
// performs are produced by the same function. Every failure before the write is
// a [refusal.Refusal] carrying the reason, and no monitor was touched.
package app

import (
	"context"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/policy"
	"github.com/leinardi/monmux/internal/refusal"
)

// Options are the choices the caller makes about how a switch is performed.
type Options struct {
	// DryRun stops after the command has been built and returns it unexecuted.
	// Nothing is written, and the backend's Execute is never reached.
	DryRun bool
}

// Outcome is what [Switch] did. It carries everything the CLI needs to report
// the result without asking the backend or the catalog anything further.
type Outcome struct {
	// Backend is the name of the backend that handled the request.
	Backend string
	// Display is the display that was selected.
	Display backend.Display
	// Model is the catalog entry the display was identified as.
	Model catalog.Model
	// Input is the input that was requested.
	Input catalog.Input
	// Operation is the mechanism and value the catalog recorded for it.
	Operation catalog.Operation
	// Command is the invocation that was built. It is what a dry run prints,
	// and what a real run performed.
	Command backend.Command
	// DryRun reports that nothing was executed.
	DryRun bool
	// Sent reports that the command was executed and the tool succeeded. It
	// means the input-switch command was sent, not that the monitor switched:
	// nothing here reads back what the display is doing.
	Sent bool
}

// Switch performs an input switch, or explains why it will not.
//
// Every failure up to and including the moment before the tool is executed is a
// [refusal.Refusal], and for all of them nothing was written. Once the tool has
// been executed the situation is different, and [ExecutionError] says so.
func Switch(
	ctx context.Context,
	driver backend.Backend,
	req policy.Request,
	opts Options,
) (Outcome, error) {
	outcome := Outcome{Backend: driver.Name(), Input: req.Input, DryRun: opts.DryRun}

	err := driver.Preflight(ctx)
	if err != nil {
		//nolint:wrapcheck // a backend's refusal is passed through unchanged; wrapping would corrupt its message
		return Outcome{}, err
	}

	displays, err := driver.Enumerate(ctx)
	if err != nil {
		//nolint:wrapcheck // a backend's refusal is passed through unchanged; wrapping would corrupt its message
		return Outcome{}, err
	}

	decision, err := policy.Resolve(displays, req)
	if err != nil {
		//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
		return Outcome{}, err
	}

	outcome.Display = decision.Display
	outcome.Model = decision.Model
	outcome.Operation = decision.Operation

	// Whether the target can be reached is a question about this monitor rather
	// than about the tool, so it is asked after the display has been chosen and
	// before anything is built for it.
	err = driver.Ready(ctx, decision.Display)
	if err != nil {
		//nolint:wrapcheck // a backend's refusal is passed through unchanged; wrapping would corrupt its message
		return Outcome{}, err
	}

	command, err := driver.Plan(decision.Display, decision.Operation)
	if err != nil {
		//nolint:wrapcheck // a backend's refusal is passed through unchanged; wrapping would corrupt its message
		return Outcome{}, err
	}

	outcome.Command = command

	if opts.DryRun {
		return outcome, nil
	}

	// Execute is handed the operation rather than the command it just showed:
	// the backend rebuilds the invocation itself, through the same planner, and
	// re-verifies the display's identity before writing.
	err = driver.Execute(ctx, decision.Display, decision.Operation)
	if err != nil {
		return Outcome{}, executionFailure(driver.Name(), err)
	}

	outcome.Sent = true

	return outcome, nil
}

// executionFailure classifies what came back from the write.
//
// The backends already draw the one distinction that matters: a tool that never
// started is a refusal, because nothing can have been written, while a tool that
// ran and failed is not, because the write may have gone out. This turns the
// second case into an error that says so rather than letting it read as one more
// refusal.
func executionFailure(name string, err error) error {
	// The check is deliberately on the error itself rather than anywhere in its
	// chain: an error that merely wraps a refusal came from something that ran,
	// and reporting it as a refusal would claim nothing was written. When in
	// doubt, say the write status is unknown.
	//
	declined, refused := err.(*refusal.Refusal)
	if refused {
		return declined
	}

	return &ExecutionError{Backend: name, Err: err}
}
