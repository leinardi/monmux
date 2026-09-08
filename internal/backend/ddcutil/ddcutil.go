//go:build linux

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

// Package ddcutil is the Linux backend. It wraps the ddcutil binary and never
// speaks I2C itself.
//
// Two things make the write as safe as this design can make it. The display is
// selected by its full 256-character EDID rather than by a bus number, so if the
// monitor on that bus changed since monmux looked, ddcutil itself refuses. And
// Execute re-reads the connector's EDID and its ddc link immediately before
// running, so a display that moved is caught here first.
package ddcutil

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/refusal"
)

const (
	// Name is the backend's name, and the binary it wraps.
	Name = "ddcutil"

	// defaultSysfsRoot is where the kernel exposes DRM connectors.
	defaultSysfsRoot = "/sys/class/drm"
	// defaultDevRoot is where the I2C device nodes live.
	defaultDevRoot = "/dev"
)

// Options configures the backend. The zero value talks to the real system.
type Options struct {
	// Runner runs ddcutil. A nil Runner means the real one.
	Runner exec.Runner
	// Path optionally pins the ddcutil binary to an absolute path, from the
	// ddcutil_path configuration key, instead of resolving it from PATH.
	Path string
	// SysfsRoot overrides /sys/class/drm. Tests use it; nothing else should.
	SysfsRoot string
	// DevRoot overrides /dev. Tests use it; nothing else should.
	DevRoot string
}

// connector is what Enumerate learned about one display and kept to itself: the
// raw EDID bytes and the I2C bus behind its ddc link. Neither is exported.
// The EDID is private data, and the bus is only meaningful together with it.
type connector struct {
	edid []byte
	bus  string
}

// Backend is the ddcutil backend.
type Backend struct {
	runner     exec.Runner
	configured string
	sysfsRoot  string
	devRoot    string

	mu         sync.Mutex
	path       string
	connectors map[string]connector
	statuses   map[string]string
}

// New returns a ddcutil backend. It performs no I/O: Preflight does that.
func New(opts Options) *Backend {
	runner := opts.Runner
	if runner == nil {
		runner = exec.NewRunner()
	}

	sysfsRoot := opts.SysfsRoot
	if sysfsRoot == "" {
		sysfsRoot = defaultSysfsRoot
	}

	devRoot := opts.DevRoot
	if devRoot == "" {
		devRoot = defaultDevRoot
	}

	return &Backend{
		runner:     runner,
		configured: opts.Path,
		sysfsRoot:  sysfsRoot,
		devRoot:    devRoot,
		connectors: map[string]connector{},
		statuses:   map[string]string{},
	}
}

// Name returns the backend's name.
func (*Backend) Name() string {
	return Name
}

// Plan returns the invocation Execute would run, without running it.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (b *Backend) Plan(
	display backend.Display,
	operation catalog.Operation,
) (backend.Command, error) {
	err := backend.ValidateOperation(operation, catalog.MechanismLGAltInput)
	if err != nil {
		//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
		return backend.Command{}, err
	}

	known, path, err := b.state(&display)
	if err != nil {
		return backend.Command{}, err
	}

	return plan(path, known, operation), nil
}

// Execute performs the switch. It re-verifies the display's identity, rebuilds
// the invocation through the same planner Plan uses, and runs ddcutil once.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (b *Backend) Execute(
	ctx context.Context,
	display backend.Display,
	operation catalog.Operation,
) error {
	err := backend.ValidateOperation(operation, catalog.MechanismLGAltInput)
	if err != nil {
		//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
		return err
	}

	known, path, err := b.state(&display)
	if err != nil {
		return err
	}

	err = b.reverify(&display, known)
	if err != nil {
		return err
	}

	command := plan(path, known, operation)

	result, err := b.runner.Run(ctx, command.Path, command.Args)
	if err == nil {
		return nil
	}

	// A tool that never started cannot have written anything, so that is a
	// refusal like any other. A tool that ran and failed is not: the write may
	// have gone out, and saying otherwise would be a lie.
	if !exec.Started(result, err) {
		return refusal.New(
			refusal.BackendNotReady,
			"ddcutil did not start: "+err.Error(),
			display.Identity,
		)
	}

	return fmt.Errorf("ddcutil exited with status %d: %w", result.ExitCode, err)
}

// plan builds the invocation. Plan and Execute both go through it, which is what
// makes --dry-run's output the same thing a real run performs.
//
// The display is selected by its full EDID rather than by its bus: ddcutil then
// refuses on its own if the monitor on that bus is no longer this one.
func plan(path string, known connector, operation catalog.Operation) backend.Command {
	return backend.Command{
		Path: path,
		Args: append(
			[]string{"--edid", hex.EncodeToString(known.edid)},
			arguments(operation.Mechanism(), operation.Value())...,
		),
		// The EDID hex identifies the physical unit, serial included.
		Redact: []int{1},
	}
}

// arguments builds the setvcp invocation for one mechanism and one value. It is
// split out of [plan] so that a test can pin the exact argv for a value without
// an exported way to build an [catalog.Operation]: rendering argv is not the
// same power as making a backend run it, and only the catalog has the latter.
//
// A mechanism this backend does not implement yields nil rather than a guess.
// [plan] is reached only after backend.ValidateOperation has accepted the
// mechanism, so nil is unreachable there; it is the shape of "never fall back"
// rather than a branch that runs.
func arguments(mechanism catalog.Mechanism, value uint16) []string {
	switch mechanism {
	case catalog.MechanismLGAltInput:
		return []string{
			"setvcp",
			fmt.Sprintf("0x%02X", catalog.LGAltInputVCP),
			fmt.Sprintf("0x%02X", value),
			fmt.Sprintf("--i2c-source-addr=0x%02X", catalog.LGAltInputSourceAddr),
			"--noverify",
		}
	default:
		return nil
	}
}

// state returns what the backend knows about a display and the resolved tool
// path, refusing if either is missing. It is what the write paths need; Ready
// asks only about the display, because reaching a monitor is a question about
// the monitor rather than about the tool.
func (b *Backend) state(display *backend.Display) (connector, string, error) {
	path, err := b.toolPath()
	if err != nil {
		return connector{}, "", err
	}

	known, err := b.connectorFor(display)
	if err != nil {
		return connector{}, "", err
	}

	return known, path, nil
}

// toolPath returns the path Preflight resolved.
func (b *Backend) toolPath() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.path == "" {
		return "", refusal.New(
			refusal.BackendNotReady,
			"ddcutil has not been checked yet; preflight must run first.",
		)
	}

	return b.path, nil
}

// connectorFor returns what Enumerate learned about a display.
func (b *Backend) connectorFor(display *backend.Display) (connector, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	known, writable := b.connectors[display.Handle]
	if writable {
		return known, nil
	}

	status, enumerated := b.statuses[display.Handle]
	if enumerated {
		return connector{}, refusal.New(
			refusal.TargetNotReady,
			"Display "+display.Label+" was enumerated but cannot be written to (status: "+status+").",
			display.Identity,
		)
	}

	return connector{}, refusal.New(
		refusal.TargetNotReady,
		"Display "+display.Label+" was not enumerated by this backend.",
		display.Identity,
	)
}

// remember stores what Enumerate learned: the writable connectors it can plan
// for, and the status of every connector it saw, so a later refusal can tell
// "monmux never saw this display" from "monmux saw it and cannot write to it".
func (b *Backend) remember(connectors map[string]connector, statuses map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.connectors = connectors
	b.statuses = statuses
}
