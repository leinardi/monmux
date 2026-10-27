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

package ddcutil_test

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/ddcutil"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/refusal"
)

// The layout ema987 reported from an RTX 5080 on the NVIDIA proprietary driver:
// the connector has no ddc link, its card and its I2C buses belong to the same
// PCI device, and an AMD iGPU's buses sit under another one.
const (
	nvidiaDevice   = "devices/pci0000:00/0000:00:01.1/0000:01:00.0"
	amdDevice      = "devices/pci0000:00/0000:00:08.1/0000:79:00.0"
	nvidiaConnect  = "card2-HDMI-A-2"
	nvidiaBus      = "i2c-9"
	nvidiaDetect   = "testdata/detect-verbose-3.0.2-nvidia.txt"
	detectCommand  = "detect --verbose"
	setvcpCommand  = "setvcp"
	detectedDetail = "ok (bus i2c-9 found by ddcutil detect)"
)

// unlinkedWorld is a fake system with one NVIDIA connector and no ddc link.
type unlinkedWorld struct {
	root    string
	binary  string
	runner  *exec.Fake
	backend *ddcutil.Backend
}

// newUnlinkedWorld builds that system. detects are what successive `ddcutil
// detect --verbose` runs print, in order; the last one keeps answering.
func newUnlinkedWorld(t *testing.T, busDevice string, detects ...string) *unlinkedWorld {
	t.Helper()

	root := t.TempDir()

	built := &unlinkedWorld{root: root, binary: filepath.Join(root, "bin", ddcutil.Name)}

	connectorDir := filepath.Join(root, nvidiaDevice, "drm", "card2", nvidiaConnect)
	mkdir(t, connectorDir)
	write(t, filepath.Join(connectorDir, "status"), []byte("connected\n"), 0o644)
	write(t, filepath.Join(connectorDir, "edid"), edidFixture(t), 0o644)

	link(t, connectorDir, filepath.Join(root, "class", "drm", nvidiaConnect))

	busDir := filepath.Join(root, busDevice, nvidiaBus)
	mkdir(t, busDir)
	link(t, busDir, filepath.Join(root, "bus", "i2c", "devices", nvidiaBus))

	mkdir(t, filepath.Join(root, "dev"))
	write(t, filepath.Join(root, "dev", nvidiaBus), nil, 0o600)

	mkdir(t, filepath.Dir(built.binary))
	write(t, built.binary, []byte("#!/bin/false\n"), 0o755)

	rules := make([]exec.Rule, 0, len(detects)+3)
	rules = append(rules,
		exec.Rule{Match: "--version", Result: exec.Result{Stdout: versionOutput}},
		exec.Rule{Match: "--help", Result: exec.Result{Stdout: helpOutput}},
	)

	for index, output := range detects {
		rule := exec.Rule{Match: detectCommand, Result: exec.Result{Stdout: output}}
		if index < len(detects)-1 {
			rule.Times = 1
		}

		rules = append(rules, rule)
	}

	built.runner = exec.NewFake(append(rules, exec.Rule{Match: setvcpCommand})...)
	built.backend = ddcutil.New(ddcutil.Options{
		Runner:    built.runner,
		Path:      built.binary,
		SysfsRoot: filepath.Join(root, "class", "drm"),
		DevRoot:   filepath.Join(root, "dev"),
	})

	return built
}

// link makes a symlink, creating its directory.
func link(t *testing.T, target, name string) {
	t.Helper()

	mkdir(t, filepath.Dir(name))

	err := os.Symlink(target, name)
	if err != nil {
		t.Fatalf("linking %s to %s: %v", name, target, err)
	}
}

// nvidiaCapture is ema987's layout, sanitized.
func nvidiaCapture(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(nvidiaDetect)
	if err != nil {
		t.Fatalf("reading %s: %v", nvidiaDetect, err)
	}

	return string(raw)
}

// enumerate runs preflight and enumeration.
func (w *unlinkedWorld) enumerate(t *testing.T) []backend.Display {
	t.Helper()

	err := w.backend.Preflight(t.Context())
	if err != nil {
		t.Fatalf("Preflight() refused: %v", err)
	}

	displays, err := w.backend.Enumerate(t.Context())
	if err != nil || len(displays) != 1 {
		t.Fatalf("Enumerate() = %d displays, %v", len(displays), err)
	}

	return displays
}

// A connector with no ddc link is placed on the one bus ddcutil reports with
// exactly its EDID, and switched like any other: selected by that EDID, after
// the bus is looked up again.
func TestAConnectorWithoutADDCLinkIsPlacedByItsEDID(t *testing.T) {
	t.Parallel()

	world := newUnlinkedWorld(t, nvidiaDevice, nvidiaCapture(t))
	displays := world.enumerate(t)

	display := displays[0]
	if !display.Writable || display.Status != backend.StatusOK {
		t.Fatalf("the display is writable %t with status %q", display.Writable, display.Status)
	}

	err := world.backend.Ready(t.Context(), display)
	if err != nil {
		t.Fatalf("Ready() refused: %v", err)
	}

	command, err := world.backend.Plan(display, operation(t))
	if err != nil {
		t.Fatalf("Plan() refused: %v", err)
	}

	want := []string{
		"--edid", hex.EncodeToString(edidFixture(t)),
		"setvcp", "0xF4", "0xD1", "--i2c-source-addr=0x50", "--noverify",
	}
	if !slices.Equal(command.Args, want) {
		t.Errorf("planned %q, want %q", command.Args, want)
	}

	before := len(world.runner.Lines())

	err = world.backend.Execute(t.Context(), display, operation(t))
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	ran := world.runner.Lines()[before:]
	if len(ran) != 2 || !strings.Contains(ran[0], detectCommand) ||
		ran[1] != command.Render(true) {
		t.Errorf("Execute ran %q, want the bus looked up again and then the planned command", ran)
	}
}

// doctor says where the bus came from, because that is the first thing to know
// if a switch then misbehaves.
func TestDoctorSaysTheBusCameFromDDCUtil(t *testing.T) {
	t.Parallel()

	world := newUnlinkedWorld(t, nvidiaDevice, nvidiaCapture(t))
	world.enumerate(t)

	for _, check := range world.backend.Doctor(t.Context()) {
		if check.Name != nvidiaConnect {
			continue
		}

		if !check.OK || check.Detail != detectedDetail {
			t.Errorf("check = %+v, want ok with %q", check, detectedDetail)
		}

		return
	}

	t.Error("doctor did not report the connector")
}

// Every way the lookup can fail leaves the connector as it was before the
// fallback existed: listed, unwritable, no-ddc-channel.
func TestAnUnplacedConnectorStaysUnwritable(t *testing.T) {
	t.Parallel()

	capture := nvidiaCapture(t)

	cases := map[string]struct {
		busDevice string
		detects   []string
	}{
		"ddcutil cannot run detect": {busDevice: nvidiaDevice},
		"the EDID is on no bus": {
			busDevice: nvidiaDevice,
			detects:   []string{strings.Replace(capture, "+0010   0c", "+0010   0d", 1)},
		},
		"the EDID is on two buses": {
			busDevice: nvidiaDevice,
			detects:   []string{capture + strings.Replace(capture, "/dev/i2c-9", "/dev/i2c-8", 1)},
		},
		"ddcutil names another connector": {
			busDevice: nvidiaDevice,
			detects:   []string{strings.ReplaceAll(capture, nvidiaConnect, "card2-HDMI-A-1")},
		},
		"the bus belongs to another device": {
			busDevice: amdDevice,
			detects:   []string{capture},
		},
	}

	for name, test := range cases {
		world := newUnlinkedWorld(t, test.busDevice, test.detects...)
		displays := world.enumerate(t)

		if displays[0].Writable || displays[0].Status != backend.StatusNoDDCChannel {
			t.Errorf("%s: the display is writable %t with status %q",
				name, displays[0].Writable, displays[0].Status)
		}
	}
}

// The bus is looked up again just before the write. A display ddcutil no longer
// places, or places on another bus, is refused with nothing sent.
func TestExecuteRefusesWhenTheLookupChanged(t *testing.T) {
	t.Parallel()

	capture := nvidiaCapture(t)

	cases := map[string]string{
		"the display moved to another bus": strings.Replace(capture, "/dev/i2c-9", "/dev/i2c-8", 1),
		"the display is gone":              "No displays found.\n",
	}

	for name, later := range cases {
		world := newUnlinkedWorld(t, nvidiaDevice, capture, later)
		displays := world.enumerate(t)

		err := world.backend.Execute(t.Context(), displays[0], operation(t))
		if !refusal.Is(err, refusal.IdentityChanged) {
			t.Errorf("%s: error = %v, want identity-changed", name, err)
		}

		for _, line := range world.runner.Lines() {
			if strings.Contains(line, setvcpCommand) {
				t.Errorf("%s: a write ran: %q", name, line)
			}
		}
	}
}

// A connector with its ddc link never runs detect: the fallback costs nothing
// on a system that does not need it.
func TestALinkedConnectorNeverRunsDetect(t *testing.T) {
	t.Parallel()

	world := newWorld(t)
	world.ready(t)

	for _, line := range world.runner.Lines() {
		if strings.Contains(line, detectCommand) {
			t.Errorf("detect ran on a system with its ddc links: %q", line)
		}
	}
}
