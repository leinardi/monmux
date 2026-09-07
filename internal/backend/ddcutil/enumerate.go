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

package ddcutil

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"syscall"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

const (
	// edidBlockSize is the length of the EDID block monmux reads and compares.
	edidBlockSize = 128
	// connectedStatus is what a connector's status file says when a monitor is
	// attached to it.
	connectedStatus = "connected"
	// i2cGroupHint tells the user how to get access to the bus.
	i2cGroupHint = "Add yourself to the i2c group (or install a udev rule granting access) and log in again."

	// readOK and writeOK are access(2)'s R_OK and W_OK. Go's syscall package
	// does not export them, and asking access(2) is the least invasive way to
	// find out whether the bus could be opened: it opens nothing.
	readOK  = 0x4
	writeOK = 0x2
)

// busPattern matches the I2C bus a connector's ddc link points at.
var busPattern = regexp.MustCompile(`^i2c-\d+$`)

// Enumerate lists the connected displays. Displays monmux cannot write to are
// listed too, with Writable false and a Status saying why: info reports them,
// and policy refuses to select them.
func (b *Backend) Enumerate(_ context.Context) ([]backend.Display, error) {
	entries, err := os.ReadDir(b.sysfsRoot)
	if err != nil {
		return nil, refusal.New(
			refusal.EnumerationFailed,
			"Could not read "+b.sysfsRoot+": "+err.Error(),
		)
	}

	displays := make([]backend.Display, 0, len(entries))
	found := make(map[string]connector, len(entries))
	statuses := make(map[string]string, len(entries))

	for _, entry := range entries {
		name := entry.Name()

		if !connected(filepath.Join(b.sysfsRoot, name)) {
			continue
		}

		display, known, writable := inspect(filepath.Join(b.sysfsRoot, name), name)

		displays = append(displays, display)
		statuses[display.Handle] = display.Status

		if writable {
			found[display.Handle] = known
		}
	}

	slices.SortFunc(displays, func(left, right backend.Display) int {
		return strings.Compare(left.Handle, right.Handle)
	})

	b.remember(found, statuses)

	return displays, nil
}

// connected reports whether a sysfs entry is a DRM connector with a monitor
// attached. Anything without a readable status file is not one.
func connected(directory string) bool {
	status, err := os.ReadFile(filepath.Join(directory, "status"))
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(status)) == connectedStatus
}

// inspect reads one connector. A display whose EDID will not parse, or which
// exposes no DDC channel, comes back unwritable with the reason in Status.
func inspect(directory, name string) (backend.Display, connector, bool) {
	display := backend.Display{
		Handle: name,
		Label:  name,
	}

	raw, err := os.ReadFile(filepath.Join(directory, "edid"))
	if err != nil || len(raw) < edidBlockSize {
		display.Status = backend.StatusEDIDUnreadable

		return display, connector{}, false
	}

	identity, err := edid.Parse(raw)
	if err != nil {
		display.Status = backend.StatusEDIDUnreadable

		return display, connector{}, false
	}

	display.Identity = identity

	bus, err := readBus(directory)
	if err != nil {
		display.Status = backend.StatusNoDDCChannel

		return display, connector{}, false
	}

	display.Writable = true
	display.Status = backend.StatusOK

	return display, connector{edid: bytes.Clone(raw[:edidBlockSize]), bus: bus}, true
}

// readBus follows a connector's ddc link to the I2C bus behind it.
func readBus(directory string) (string, error) {
	target, err := os.Readlink(filepath.Join(directory, "ddc"))
	if err != nil {
		return "", refusal.New(refusal.TargetNotReady, "No ddc link: "+err.Error())
	}

	bus := filepath.Base(target)
	if !busPattern.MatchString(bus) {
		return "", refusal.New(
			refusal.TargetNotReady,
			"The ddc link points at "+bus+", which is not an I2C bus.",
		)
	}

	return bus, nil
}

// Ready reports whether this display's I2C bus can be opened for writing right
// now. It is about the target rather than the tool: ddcutil can be perfectly
// installed and still be unable to reach this monitor.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (b *Backend) Ready(_ context.Context, display backend.Display) error {
	known, err := b.connectorFor(&display)
	if err != nil {
		return err
	}

	device := filepath.Join(b.devRoot, known.bus)

	_, err = os.Stat(device)
	if err != nil {
		return refusal.New(
			refusal.TargetNotReady,
			device+" is not available: "+err.Error(),
			display.Identity,
		)
	}

	err = syscall.Access(device, readOK|writeOK)
	if err != nil {
		return refusal.New(
			refusal.TargetNotReady,
			device+" cannot be opened for reading and writing. "+i2cGroupHint,
			display.Identity,
		)
	}

	return nil
}

// reverify re-reads the connector immediately before a write. If its EDID or its
// bus differs by a single byte from what Enumerate saw, the display is not the
// one that was identified and nothing is sent.
func (b *Backend) reverify(display *backend.Display, known connector) error {
	directory := filepath.Join(b.sysfsRoot, display.Handle)

	raw, err := os.ReadFile(filepath.Join(directory, "edid"))
	if err != nil || len(raw) < edidBlockSize {
		return refusal.New(
			refusal.IdentityChanged,
			"The EDID of "+display.Label+" could no longer be read.",
			display.Identity,
		)
	}

	if !bytes.Equal(raw[:edidBlockSize], known.edid) {
		return refusal.New(
			refusal.IdentityChanged,
			"The EDID of "+display.Label+" changed since it was identified.",
			display.Identity,
		)
	}

	bus, err := readBus(directory)
	if err != nil {
		return refusal.New(
			refusal.IdentityChanged,
			"The DDC channel of "+display.Label+" disappeared since it was identified.",
			display.Identity,
		)
	}

	if bus != known.bus {
		return refusal.New(
			refusal.IdentityChanged,
			"Display "+display.Label+" moved from "+known.bus+" to "+bus+".",
			display.Identity,
		)
	}

	return nil
}
