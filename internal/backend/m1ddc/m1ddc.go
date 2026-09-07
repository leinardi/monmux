//go:build darwin

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

package m1ddc

import (
	"context"
	"sync"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/refusal"
)

// Options configures the backend. The zero value talks to the real system.
type Options struct {
	// Runner runs m1ddc. A nil Runner means the real one.
	Runner exec.Runner
	// Path optionally pins the m1ddc binary to an absolute path, from the
	// m1ddc_path configuration key, instead of resolving it from PATH.
	Path string
}

// Backend is the m1ddc backend.
type Backend struct {
	runner     exec.Runner
	configured string

	mu       sync.Mutex
	path     string
	known    map[string]record
	statuses map[string]string
}

// New returns an m1ddc backend. It performs no I/O: Preflight does that.
func New(opts Options) *Backend {
	runner := opts.Runner
	if runner == nil {
		runner = exec.NewRunner()
	}

	return &Backend{
		runner:     runner,
		configured: opts.Path,
		known:      map[string]record{},
		statuses:   map[string]string{},
	}
}

// Name returns the backend's name.
func (*Backend) Name() string {
	return Name
}

// Ready reports whether this display can be reached right now, and on macOS it
// always can: there is no per-target permission model to check, no device node
// to open and no group to belong to. Whether the display can be written to at
// all was already decided during enumeration, and is carried by Writable.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (*Backend) Ready(_ context.Context, _ backend.Display) error {
	return nil
}

// Plan returns the invocation Execute would run, without running it.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (b *Backend) Plan(
	display backend.Display,
	operation catalog.Operation,
) (backend.Command, error) {
	err := validate(operation)
	if err != nil {
		return backend.Command{}, err
	}

	path, err := b.toolPath()
	if err != nil {
		return backend.Command{}, err
	}

	known, err := b.recordFor(&display)
	if err != nil {
		return backend.Command{}, err
	}

	return planFor(path, &known, operation)
}

// Execute performs the switch. It re-asks m1ddc what is attached, confirms the
// UUID still names the display that was identified, rebuilds the invocation
// through the same planner Plan uses, and runs m1ddc once.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (b *Backend) Execute(
	ctx context.Context,
	display backend.Display,
	operation catalog.Operation,
) error {
	err := validate(operation)
	if err != nil {
		return err
	}

	path, err := b.toolPath()
	if err != nil {
		return err
	}

	known, err := b.recordFor(&display)
	if err != nil {
		return err
	}

	command, err := planFor(path, &known, operation)
	if err != nil {
		return err
	}

	current, err := b.list(ctx, refusal.IdentityChanged, path)
	if err != nil {
		return err
	}

	err = verify(current, &display)
	if err != nil {
		return err
	}

	result, err := b.runner.Run(ctx, command.Path, command.Args)

	return executionOutcome(&display, result, err)
}

// list runs `m1ddc display list detailed` and interprets the answer.
func (b *Backend) list(ctx context.Context, reason refusal.Reason, path string) ([]record, error) {
	result, err := b.runner.Run(ctx, path, listArgs)

	return interpretList(reason, result, err)
}

// toolPath returns the path Preflight resolved.
func (b *Backend) toolPath() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.path == "" {
		return "", refusal.New(
			refusal.BackendNotReady,
			Name+" has not been checked yet; preflight must run first.",
		)
	}

	return b.path, nil
}

// recordFor returns what Enumerate learned about a display.
func (b *Backend) recordFor(display *backend.Display) (record, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return lookup(b.known, b.statuses, display)
}

// remember stores what Enumerate learned.
func (b *Backend) remember(known map[string]record, statuses map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.known = known
	b.statuses = statuses
}

// rememberPath stores the tool path Preflight resolved.
func (b *Backend) rememberPath(path string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.path = path
}
