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

// The whole command tree is tested here against a fake backend: no monitor, no
// configuration file and no external binary is involved, and the fake records
// whether anything would have been written.

package main

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/config"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

// errTool stands in for a tool that ran and failed, and errConfig for a
// configuration file monmux will not use.
var (
	errTool   = errors.New("exit status 1")
	errConfig = errors.New("config.yaml: field color not found in type config.Config")
)

// planned is what the fake backend reports it would run. It is shaped like a
// real ddcutil invocation so the dry-run output is worth pinning.
func planned() backend.Command {
	return backend.Command{
		Path: "/usr/bin/ddcutil",
		Args: []string{
			"--edid", "00ffffffffffff001e6dd4770102030401230103",
			"setvcp", "0xF4", "0xD1",
			"--i2c-source-addr=0x50", "--noverify",
		},
		Redact: []int{1},
	}
}

// supported is the tested LG, identified and writable.
func supported() backend.Display {
	return backend.Display{
		Identity: edid.Identity{
			Manufacturer: "GSM",
			ProductCode:  0x77D4,
			SerialNumber: 0x01020304,
			SerialString: "TESTSERIAL01",
			ModelName:    "LG ULTRAWIDE",
		},
		Handle:   "card1-DP-1",
		Label:    "card1-DP-1",
		Writable: true,
		Status:   backend.StatusOK,
	}
}

// hidden is a macOS-shaped display: its handle is private data.
func hidden() backend.Display {
	display := supported()
	display.Handle = "00000000-0000-4000-8000-000000000001"
	display.HandlePrivate = true
	display.Label = "display 1"

	return display
}

// unknown is a monitor the catalog knows nothing about, on a connector with no
// DDC channel.
func unknown() backend.Display {
	return backend.Display{
		Identity: edid.Identity{Manufacturer: "XXX", ProductCode: 0x1234},
		Handle:   "card1-HDMI-A-1",
		Label:    "card1-HDMI-A-1",
		Status:   backend.StatusNoDDCChannel,
	}
}

// checks are what the fake backend's doctor reports.
func checks() []backend.Check {
	return []backend.Check{
		{Name: "ddcutil binary", OK: true, Detail: "/usr/bin/ddcutil"},
		{Name: "card1-HDMI-A-1", OK: false, Detail: "no-ddc-channel"},
	}
}

// world is one invocation of monmux with a fake backend behind it.
type world struct {
	driver *backend.Fake
	config config.Config
	// ConfigErr is returned instead of the configuration.
	configErr error
}

// newWorld returns a system with one supported, writable display.
func newWorld(displays ...backend.Display) *world {
	if len(displays) == 0 {
		displays = []backend.Display{supported()}
	}

	return &world{
		driver: &backend.Fake{
			FakeName: "ddcutil",
			Displays: displays,
			Checks:   checks(),
			Planned:  planned(),
		},
	}
}

// run executes one command line and returns what the user would have seen.
func (w *world) run(args ...string) (stdout, stderr string, code int) {
	out := &strings.Builder{}
	errs := &strings.Builder{}

	code = run(args, out, errs, dependencies{
		Config: func() (config.Config, error) {
			return w.config, w.configErr
		},
		Backend: func(config.Config) (backend.Backend, error) {
			return w.driver, nil
		},
	})

	return out.String(), errs.String(), code
}

// wrote reports whether anything was sent to a monitor.
func (w *world) wrote() bool {
	return len(w.driver.Executed()) > 0
}

// equal compares output against what it must be, byte for byte.
func equal(t *testing.T, what, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("%s =\n%s\nwant\n%s", what, got, want)
	}
}

// --- info ---

func TestInfoJSONIsRedactedByDefault(t *testing.T) {
	t.Parallel()

	world := newWorld(hidden(), unknown())

	stdout, stderr, code := world.run("info", "--json")
	if code != exitSent || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	want := `{
  "backend": "ddcutil",
  "displays": [
    {
      "label": "display 1",
      "handle": "` + refusal.RedactionMask + `",
      "handlePrivate": true,
      "writable": true,
      "status": "ok",
      "identity": {
        "manufacturer": "GSM",
        "productCode": 30676,
        "serialNumber": 0,
        "serialString": "",
        "modelName": "LG ULTRAWIDE"
      },
      "match": "exact",
      "model": "LG 38WR85QC-W",
      "enabledInputs": [
        "dp",
        "usb-c"
      ]
    },
    {
      "label": "card1-HDMI-A-1",
      "handle": "card1-HDMI-A-1",
      "handlePrivate": false,
      "writable": false,
      "status": "no-ddc-channel",
      "identity": {
        "manufacturer": "XXX",
        "productCode": 4660,
        "serialNumber": 0,
        "serialString": "",
        "modelName": ""
      },
      "match": "none",
      "model": "",
      "enabledInputs": []
    }
  ],
  "checks": [
    {
      "name": "ddcutil binary",
      "ok": true,
      "detail": "/usr/bin/ddcutil"
    },
    {
      "name": "card1-HDMI-A-1",
      "ok": false,
      "detail": "no-ddc-channel"
    }
  ]
}
`

	equal(t, "info --json", stdout, want)

	if world.wrote() {
		t.Error("info wrote to a monitor")
	}
}

func TestInfoJSONPrintsPrivateDataOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	world := newWorld(hidden())

	stdout, _, code := world.run("info", "--json", "--show-serial")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	for _, wanted := range []string{
		`"handle": "00000000-0000-4000-8000-000000000001"`,
		`"serialNumber": 16909060`,
		`"serialString": "TESTSERIAL01"`,
	} {
		if !strings.Contains(stdout, wanted) {
			t.Errorf("--show-serial did not print %s:\n%s", wanted, stdout)
		}
	}
}

func TestInfoTextListsEveryDisplay(t *testing.T) {
	t.Parallel()

	world := newWorld(supported(), unknown())

	stdout, _, code := world.run("info")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	want := `Backend: ddcutil

Display card1-DP-1
  Handle:  card1-DP-1
  Monitor: GSM 0x77D4 (LG ULTRAWIDE)
  Serial:  ` + refusal.RedactionMask + `
  Status:  ok (writable)
  Model:   LG 38WR85QC-W
  Inputs:  dp, usb-c

Display card1-HDMI-A-1
  Handle:  card1-HDMI-A-1
  Monitor: XXX 0x1234
  Serial:  none
  Status:  no-ddc-channel (not writable)
  Model:   not in the catalog (none)
  Inputs:  none

Checks:
  ok   ddcutil binary: /usr/bin/ddcutil
  FAIL card1-HDMI-A-1: no-ddc-channel
`

	equal(t, "info", stdout, want)
}

// A read-only command must not be answered with the wording of a refused write.
func TestInfoReportsWhatItCanWhenTheToolIsUnusable(t *testing.T) {
	t.Parallel()

	world := newWorld()
	world.driver.PreflightErr = refusal.New(
		refusal.BackendNotReady,
		"ddcutil was not found on PATH",
	)

	stdout, stderr, code := world.run("info")
	if code != exitRefused {
		t.Errorf("exit = %d, want %d", code, exitRefused)
	}

	if !strings.Contains(stdout, "Display card1-DP-1") {
		t.Errorf("the displays were not reported:\n%s", stdout)
	}

	want := "Not ready: backend-not-ready: ddcutil was not found on PATH\n"
	equal(t, "stderr", stderr, want)

	if strings.Contains(stderr, "No DDC write was performed") {
		t.Errorf("a read-only command talked about writes:\n%s", stderr)
	}
}

// --- switch ---

func TestSwitchReportsSentAndNotConfirmed(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("switch", "usb-c")
	if code != exitSent || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	want := "Input-switch command sent (USB-C, 0xD1) to LG 38WR85QC-W via ddcutil. " +
		"Switch not independently confirmed.\n"

	equal(t, "switch", stdout, want)

	if !world.wrote() {
		t.Error("nothing was sent")
	}
}

func TestDryRunRedactsPrivateArgumentsByDefault(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("switch", "usb-c", "--dry-run")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	want := `Dry run: nothing was sent.
  Display: card1-DP-1 (LG 38WR85QC-W)
  Input:   USB-C (0xD1)
  Command: /usr/bin/ddcutil --edid ` + refusal.RedactionMask +
		" setvcp 0xF4 0xD1 --i2c-source-addr=0x50 --noverify\n" +
		"No DDC write was performed.\n"

	equal(t, "switch --dry-run", stdout, want)

	if world.wrote() {
		t.Error("a dry run wrote to a monitor")
	}
}

func TestDryRunPrintsTheCommandVerbatimWhenAsked(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("switch", "usb-c", "--dry-run", "--show-serial")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	want := `Dry run: nothing was sent.
  Display: card1-DP-1 (LG 38WR85QC-W)
  Input:   USB-C (0xD1)
  Command: /usr/bin/ddcutil --edid 00ffffffffffff001e6dd4770102030401230103 setvcp 0xF4 0xD1 --i2c-source-addr=0x50 --noverify
No DDC write was performed.
`

	equal(t, "switch --dry-run --show-serial", stdout, want)

	if world.wrote() {
		t.Error("a dry run wrote to a monitor")
	}
}

func TestARefusalIsPrintedInFullAndExitsTwo(t *testing.T) {
	t.Parallel()

	world := newWorld(unknown())

	stdout, stderr, code := world.run("switch", "usb-c")
	if code != exitRefused {
		t.Errorf("exit = %d, want %d", code, exitRefused)
	}

	if stdout != "" {
		t.Errorf("a refused switch printed to stdout:\n%s", stdout)
	}

	if !strings.HasPrefix(stderr, "Refusing to switch input.") {
		t.Errorf("the refusal template was not used:\n%s", stderr)
	}

	if !strings.HasSuffix(stderr, "No DDC write was performed.\n") {
		t.Errorf("the refusal does not end with the promise:\n%s", stderr)
	}

	if world.wrote() {
		t.Error("a refused switch wrote to a monitor")
	}
}

func TestARefusalRedactsSerialsUnlessAsked(t *testing.T) {
	t.Parallel()

	world := newWorld(unknown(), supported(), supported())

	_, stderr, _ := world.run("switch", "usb-c")
	if strings.Contains(stderr, "TESTSERIAL01") {
		t.Errorf("a refusal leaked a serial:\n%s", stderr)
	}

	_, verbose, _ := world.run("switch", "usb-c", "--show-serial")
	if !strings.Contains(verbose, "TESTSERIAL01") {
		t.Errorf("--show-serial did not print the serial:\n%s", verbose)
	}
}

// The one failure that is not a refusal: the tool ran, so what reached the
// monitor is unknown, and the exit code must not say "nothing was written".
func TestAToolThatRanAndFailedExitsOne(t *testing.T) {
	t.Parallel()

	world := newWorld()
	world.driver.ExecuteErr = errTool

	_, stderr, code := world.run("switch", "usb-c")
	if code != exitToolError {
		t.Errorf("exit = %d, want %d", code, exitToolError)
	}

	if !strings.Contains(stderr, "unknown") {
		t.Errorf("the message does not say the write status is unknown:\n%s", stderr)
	}

	if strings.Contains(stderr, "No DDC write was performed") {
		t.Errorf("the message claims nothing was written:\n%s", stderr)
	}
}

// An argument that is not a well-formed connector name is a mistake in the
// request, and nothing should be started for it.
func TestAnUnknownInputStartsNothing(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"hdmi01", "scart"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			world := newWorld()

			_, stderr, code := world.run("switch", name)
			if code != exitToolError {
				t.Errorf("exit = %d, want %d", code, exitToolError)
			}

			if !strings.Contains(stderr, "monmux info") {
				t.Errorf("the error does not point at monmux info:\n%s", stderr)
			}

			if len(world.driver.Calls()) != 0 {
				t.Errorf("an unknown input still called the backend: %v", world.driver.Calls())
			}
		})
	}
}

// A name that is well formed but not enabled for the matched model is the other
// half of the same story: it is a refusal, not a usage error, and it promises
// that nothing was written.
func TestAParseableInputTheModelDoesNotEnableIsRefused(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"hdmi", "hdmi1", "hdmi3"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			world := newWorld()

			_, stderr, code := world.run("switch", name)
			if code != exitRefused {
				t.Errorf("exit = %d, want %d", code, exitRefused)
			}

			if !strings.Contains(stderr, "The requested input is not enabled for this model.") {
				t.Errorf("the refusal is not input-not-enabled:\n%s", stderr)
			}

			if !strings.Contains(stderr, "Requested "+name+" on LG 38WR85QC-W") {
				t.Errorf("the refusal does not name the input asked for:\n%s", stderr)
			}

			if !strings.HasSuffix(stderr, "No DDC write was performed.\n") {
				t.Errorf("the refusal does not end with the promise:\n%s", stderr)
			}

			if world.wrote() {
				t.Error("a refused switch wrote to a monitor")
			}
		})
	}
}

// Completion offers every input the catalog records, in listing order and once
// each. It is built from the catalog, so adding a port to models.yaml needs no
// edit here.
func TestCompletionOffersEveryRecordedInput(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("__complete", "switch", "")
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")

	// cobra ends the list with a line naming the shell directive, which is not
	// a candidate.
	if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], ":") {
		lines = lines[:len(lines)-1]
	}

	want := make([]string, 0, len(catalog.KnownInputs()))
	for _, input := range catalog.KnownInputs() {
		want = append(want, input.String())
	}

	if !slices.Equal(lines, want) {
		t.Errorf("completion offered %v, want %v", lines, want)
	}
}

// --- configuration ---

func TestTheSerialFlagWinsOverTheConfiguredPin(t *testing.T) {
	t.Parallel()

	other := supported()
	other.Handle = "card1-DP-2"
	other.Label = "card1-DP-2"
	other.Identity.SerialString = "TESTSERIAL02"

	world := newWorld(supported(), other)
	world.config = config.Config{Serial: "TESTSERIAL01"}

	_, stderr, code := world.run("switch", "usb-c", "--serial", "TESTSERIAL02")
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	targets := world.driver.Targets()
	if len(targets) != 1 || targets[0].Handle != other.Handle {
		t.Errorf("wrote to %v, want only the pinned %q", targets, other.Handle)
	}
}

func TestTheConfiguredPinIsUsedWhenNoFlagIsGiven(t *testing.T) {
	t.Parallel()

	other := supported()
	other.Handle = "card1-DP-2"
	other.Label = "card1-DP-2"
	other.Identity.SerialString = "TESTSERIAL02"

	world := newWorld(supported(), other)
	world.config = config.Config{Serial: "TESTSERIAL02"}

	_, stderr, code := world.run("switch", "usb-c")
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	targets := world.driver.Targets()
	if len(targets) != 1 || targets[0].Handle != other.Handle {
		t.Errorf("wrote to %v, want only the pinned %q", targets, other.Handle)
	}
}

func TestABrokenConfigurationStartsNothing(t *testing.T) {
	t.Parallel()

	world := newWorld()
	world.configErr = errConfig

	_, stderr, code := world.run("switch", "usb-c")
	if code != exitToolError {
		t.Errorf("exit = %d, want %d", code, exitToolError)
	}

	if !strings.Contains(stderr, "color") {
		t.Errorf("the configuration error was not reported:\n%s", stderr)
	}

	if len(world.driver.Calls()) != 0 {
		t.Errorf("a broken configuration still called the backend: %v", world.driver.Calls())
	}
}

// --- the read-only commands ---

func TestDoctorPrintsTheChecksAndWritesNothing(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("doctor")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	want := `Backend: ddcutil

Checks:
  ok   ddcutil binary: /usr/bin/ddcutil
  FAIL card1-HDMI-A-1: no-ddc-channel
`

	equal(t, "doctor", stdout, want)

	if world.wrote() {
		t.Error("doctor wrote to a monitor")
	}
}

func TestVersionPrintsTheBuildMetadata(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("version")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	if !strings.HasPrefix(stdout, "monmux ") {
		t.Errorf("version printed %q", stdout)
	}
}

func TestCompletionIsAvailable(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("completion", "bash")
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	if !strings.Contains(stdout, "monmux") {
		t.Error("the completion script does not mention monmux")
	}
}
