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
func (b *Backend) Enumerate(ctx context.Context) ([]backend.Display, error) {
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
	unlinked := map[int]connector{}

	for _, entry := range entries {
		name := entry.Name()

		if !connected(filepath.Join(b.sysfsRoot, name)) {
			continue
		}

		display, known, writable := inspect(filepath.Join(b.sysfsRoot, name), name)

		if !writable && known.edid != nil {
			unlinked[len(displays)] = known
		}

		displays = append(displays, display)

		if writable {
			found[display.Handle] = known
		}
	}

	b.resolveUnlinked(ctx, displays, unlinked, found)

	for index := range displays {
		statuses[displays[index].Handle] = displays[index].Status
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
// exposes no DDC channel, comes back unwritable with the reason in Status. One
// with a good EDID and no ddc link also returns that EDID, so the caller can
// look its bus up with ddcutil instead.
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

		return display, connector{edid: bytes.Clone(raw[:edidBlockSize])}, false
	}

	display.Writable = true
	display.Status = backend.StatusOK

	return display, connector{edid: bytes.Clone(raw[:edidBlockSize]), bus: bus}, true
}

// resolveUnlinked finds the bus of every connector that has a good EDID and no
// ddc link, by asking ddcutil which bus carries that exact EDID. It runs only
// when such a connector exists and preflight has accepted ddcutil, so a system
// whose connectors all have their link never pays for it. A connector it cannot
// place stays as it was: unwritable, with status no-ddc-channel.
func (b *Backend) resolveUnlinked(
	ctx context.Context,
	displays []backend.Display,
	unlinked map[int]connector,
	found map[string]connector,
) {
	if len(unlinked) == 0 {
		return
	}

	blocks, ok := b.detect(ctx)
	if !ok {
		return
	}

	for index, known := range unlinked {
		display := &displays[index]

		bus, placed := b.placeBus(blocks, known.edid, display.Handle)
		if !placed {
			continue
		}

		display.Writable = true
		display.Status = backend.StatusOK
		found[display.Handle] = connector{edid: known.edid, bus: bus, detected: true}
	}
}

// detect runs `ddcutil detect --verbose` and parses it. Nothing is concluded
// from a run that failed: the connectors it would have placed stay unwritable.
func (b *Backend) detect(ctx context.Context) ([]detectedBus, bool) {
	path, err := b.toolPath()
	if err != nil {
		return nil, false
	}

	result, err := b.runner.Run(ctx, path, detectArgs)
	if err != nil {
		return nil, false
	}

	return parseDetect(result.Stdout), true
}

// placeBus returns the bus ddcutil reports for an EDID, once it has also been
// checked against sysfs: the bus has to belong to the same PCI device as the
// connector's card. ddcutil's own association is a second opinion; this one is
// the kernel's, and either disagreeing leaves the connector unplaced.
func (b *Backend) placeBus(blocks []detectedBus, block []byte, name string) (string, bool) {
	bus, ok := busByEDID(blocks, block, name)
	if !ok || !busPattern.MatchString(bus) {
		return "", false
	}

	connectorPath, err := filepath.EvalSymlinks(filepath.Join(b.sysfsRoot, name))
	if err != nil {
		return "", false
	}

	device, _, cut := strings.Cut(
		connectorPath,
		string(filepath.Separator)+"drm"+string(filepath.Separator),
	)
	if !cut {
		return "", false
	}

	busPath, err := filepath.EvalSymlinks(filepath.Join(b.i2cRoot, bus))
	if err != nil || !strings.HasPrefix(busPath, device+string(filepath.Separator)) {
		return "", false
	}

	return bus, true
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
// one that was identified and nothing is sent. A bus that came from ddcutil is
// looked up again the same way, rather than trusted from the earlier run.
func (b *Backend) reverify(ctx context.Context, display *backend.Display, known connector) error {
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

	bus, err := b.currentBus(ctx, directory, display.Handle, known)
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

// currentBus is the bus a connector is on now, found the way Enumerate found it:
// through the ddc link, or through ddcutil for a connector that has none.
func (b *Backend) currentBus(
	ctx context.Context,
	directory, name string,
	known connector,
) (string, error) {
	if !known.detected {
		return readBus(directory)
	}

	blocks, ok := b.detect(ctx)
	if !ok {
		return "", refusal.New(refusal.TargetNotReady, "ddcutil detect failed.")
	}

	bus, placed := b.placeBus(blocks, known.edid, name)
	if !placed {
		return "", refusal.New(refusal.TargetNotReady, "ddcutil no longer places the display.")
	}

	return bus, nil
}
