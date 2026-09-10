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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
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

func TestVersionJSONDecodes(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("version", "--json")
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	var document struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}

	err := json.Unmarshal([]byte(stdout), &document)
	if err != nil {
		t.Fatalf("version --json did not decode: %v", err)
	}

	if document.Version != version || document.Commit != commit || document.Date != date {
		t.Errorf(
			"version --json = %+v, want %q/%q/%q",
			document,
			version,
			commit,
			date,
		)
	}
}

func TestVersionJSONReplacesTheProseOutput(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("version", "--json")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	if strings.Contains(stdout, "monmux ") {
		t.Errorf("version --json printed prose too:\n%s", stdout)
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

// --- catalog ---

// catalogRows returns the model rows of `monmux catalog list`, without the
// header, the blank line or the summary, plus the summary itself.
func catalogRows(t *testing.T, stdout string) (rows []string, summary string) {
	t.Helper()

	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("catalog list printed only:\n%s", stdout)
	}

	if !strings.HasPrefix(lines[0], "VENDOR") {
		t.Errorf("the first line is not the header: %q", lines[0])
	}

	if lines[len(lines)-2] != "" {
		t.Errorf("the summary is not preceded by a blank line: %q", lines[len(lines)-2])
	}

	return lines[1 : len(lines)-2], lines[len(lines)-1]
}

// writeEnabledCount is how many catalog entries monmux may write to.
func writeEnabledCount(entries []catalog.Model) int {
	count := 0

	for index := range entries {
		if entries[index].WriteEnabled {
			count++
		}
	}

	return count
}

func TestCatalogListIsInCatalogOrder(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("catalog", "list")
	if code != exitSent || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	entries := catalog.Models()

	rows, summary := catalogRows(t, stdout)
	if len(rows) != len(entries) {
		t.Fatalf("catalog list printed %d models, want %d", len(rows), len(entries))
	}

	for index := range entries {
		entry := &entries[index]

		if !strings.HasPrefix(rows[index], entry.Vendor) ||
			!strings.Contains(rows[index], entry.Name) {
			t.Fatalf("row %d is %q, want %s %s", index, rows[index], entry.Vendor, entry.Name)
		}
	}

	want := fmt.Sprintf("%d models, %d write-enabled.", len(entries), writeEnabledCount(entries))
	equal(t, "the summary", summary, want)

	if world.wrote() {
		t.Error("catalog list wrote to a monitor")
	}
}

func TestCatalogListFiltersCaseInsensitively(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("catalog", "list", "aOc")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	wanted := 0

	for _, entry := range catalog.Models() {
		if strings.EqualFold(entry.Vendor, "AOC") {
			wanted++
		}
	}

	rows, _ := catalogRows(t, stdout)
	if len(rows) != wanted {
		t.Fatalf("the filter kept %d models, want %d:\n%s", len(rows), wanted, stdout)
	}

	for _, row := range rows {
		if !strings.HasPrefix(row, "AOC") {
			t.Errorf("the filter kept %q", row)
		}
	}
}

func TestCatalogListVerboseCarriesValueAndGrade(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("catalog", "list", "Q27P1B", "--verbose")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	rows, _ := catalogRows(t, stdout)
	if len(rows) != 1 {
		t.Fatalf("the filter kept %d models, want 1:\n%s", len(rows), stdout)
	}

	if !strings.Contains(rows[0], "dp 0x0F reported") ||
		!strings.Contains(rows[0], "vga 0x01 reported") {
		t.Errorf("--verbose printed %q", rows[0])
	}
}

func TestCatalogListJSONDecodes(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("catalog", "list", "--json")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	var document struct {
		Models []struct {
			Vendor       string   `json:"vendor"`
			Name         string   `json:"name"`
			FullName     string   `json:"fullName"`
			WriteEnabled bool     `json:"writeEnabled"`
			Identities   []string `json:"identities"`
			Mechanisms   []string `json:"mechanisms"`
			Inputs       []struct {
				Name      string `json:"name"`
				Mechanism string `json:"mechanism"`
				Value     uint16 `json:"value"`
				ValueHex  string `json:"valueHex"`
				Grade     string `json:"grade"`
				Evidence  string `json:"evidence"`
			} `json:"inputs"`
		} `json:"models"`
		Count        int `json:"count"`
		WriteEnabled int `json:"writeEnabled"`
	}

	err := json.Unmarshal([]byte(stdout), &document)
	if err != nil {
		t.Fatalf("catalog list --json did not decode: %v", err)
	}

	entries := catalog.Models()
	if document.Count != len(entries) || len(document.Models) != len(entries) {
		t.Fatalf(
			"count = %d, models = %d, want %d",
			document.Count,
			len(document.Models),
			len(entries),
		)
	}

	if document.WriteEnabled != writeEnabledCount(entries) {
		t.Errorf("writeEnabled = %d, want %d", document.WriteEnabled, writeEnabledCount(entries))
	}

	first := document.Models[0]
	if first.Vendor != entries[0].Vendor || first.Name != entries[0].Name ||
		first.FullName != entries[0].FullName() || !first.WriteEnabled {
		t.Errorf("the first entry decoded as %+v", first)
	}

	if len(first.Inputs) == 0 || first.Inputs[0].ValueHex != "0xD0" ||
		first.Inputs[0].Grade != "verified" {
		t.Errorf("the first entry's inputs decoded as %+v", first.Inputs)
	}
}

func TestCatalogShowPrintsAWriteEnabledEntry(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("catalog", "show", "lg/38WR85QC-W")
	if code != exitSent || stderr != "" {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	for _, wanted := range []string{
		"LG 38WR85QC-W\n",
		"  Write-enabled: yes\n",
		"  Identities:    GSM/0x77D3, GSM/0x77D4\n",
		"    dp     lg-alt-input  0xD0  verified  Direct test on the unit",
		"    usb-c  lg-alt-input  0xD1  verified  Direct test on the unit",
		"  Notes:\n",
		"  Sources:\n",
	} {
		if !strings.Contains(stdout, wanted) {
			t.Errorf("catalog show did not print %q:\n%s", wanted, stdout)
		}
	}

	if strings.Contains(stdout, "--unsafe-model") {
		t.Error("a write-enabled entry pointed at the override")
	}

	if world.wrote() {
		t.Error("catalog show wrote to a monitor")
	}
}

func TestCatalogShowPointsARecordedOnlyEntryAtTheOverride(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, _, code := world.run("catalog", "show", "AOC Q27P1B")
	if code != exitSent {
		t.Fatalf("exit = %d", code)
	}

	for _, wanted := range []string{
		"AOC Q27P1B\n",
		"  Write-enabled: no\n",
		"  Identities:    none (can never match a display)\n",
		"    vga   vcp-input-source  0x01  reported  Reported working",
		"--unsafe-model AOC/Q27P1B",
		"docs/adding-a-monitor.md",
	} {
		if !strings.Contains(stdout, wanted) {
			t.Errorf("catalog show did not print %q:\n%s", wanted, stdout)
		}
	}
}

func TestCatalogShowOfAnUnknownNameIsAnError(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("catalog", "show", "Acme/Nothing")
	if code != exitToolError {
		t.Fatalf("exit = %d, want %d", code, exitToolError)
	}

	if stdout != "" {
		t.Errorf("an unknown model printed %q", stdout)
	}

	if !strings.Contains(stderr, "monmux catalog list") {
		t.Errorf("the error does not point at the listing: %q", stderr)
	}

	if !strings.Contains(stderr, "VENDOR/MODEL") {
		t.Errorf("the error does not say what a name looks like: %q", stderr)
	}

	if strings.Contains(stderr, "No DDC write was performed") {
		t.Errorf("a read-only command talked about writes: %q", stderr)
	}
}

// The model name alone is the obvious thing to type and is not a name, so the
// error has to say which name to type instead.
func TestCatalogShowOfAModelNameSuggestsTheFullName(t *testing.T) {
	t.Parallel()

	world := newWorld()

	_, stderr, code := world.run("catalog", "show", "38WR85QC-W")
	if code != exitToolError {
		t.Fatalf("exit = %d, want %d", code, exitToolError)
	}

	for _, wanted := range []string{
		`no catalog entry is named "38WR85QC-W"`,
		`did you mean "LG/38WR85QC-W"?`,
	} {
		if !strings.Contains(stderr, wanted) {
			t.Errorf("the error does not say %q: %q", wanted, stderr)
		}
	}
}

// A name that resembles nothing gets the rule and an example instead of a guess.
func TestCatalogShowOfANameLikeNothingExplainsTheForm(t *testing.T) {
	t.Parallel()

	world := newWorld()

	_, stderr, code := world.run("catalog", "show", "nothing-like-a-model")
	if code != exitToolError {
		t.Fatalf("exit = %d, want %d", code, exitToolError)
	}

	if strings.Contains(stderr, "did you mean") {
		t.Errorf("a name matching nothing was guessed at: %q", stderr)
	}

	for _, wanted := range []string{"VENDOR/MODEL", "LG/38WR85QC-W", "monmux catalog list"} {
		if !strings.Contains(stderr, wanted) {
			t.Errorf("the error does not say %q: %q", wanted, stderr)
		}
	}
}

// --- switch --unsafe-model ---

// strange is a writable display no catalog entry claims: the case the override
// exists for.
func strange() backend.Display {
	return backend.Display{
		Identity: edid.Identity{Manufacturer: "XXX", ProductCode: 0x2701},
		Handle:   "card1-HDMI-A-1",
		Label:    "card1-HDMI-A-1",
		Writable: true,
		Status:   backend.StatusOK,
	}
}

// The same setup without the flag is the control: identification is what the
// override bypasses, and it refuses on its own.
func TestSwitchRefusesAnUnidentifiedDisplayWithoutTheOverride(t *testing.T) {
	t.Parallel()

	world := newWorld(strange())

	_, stderr, code := world.run("switch", "hdmi", "--dry-run")
	if code != exitRefused {
		t.Fatalf("exit = %d, want %d", code, exitRefused)
	}

	if !strings.Contains(stderr, "No supported-model catalog entry matches this identity.") {
		t.Errorf("the refusal is not unknown-monitor: %q", stderr)
	}

	if !strings.Contains(stderr, "No DDC write was performed.") {
		t.Errorf("the refusal does not promise nothing was written: %q", stderr)
	}

	if world.wrote() {
		t.Error("a refused switch wrote to a monitor")
	}
}

func TestSwitchWithUnsafeModelWarnsOnStderrAndDryRuns(t *testing.T) {
	t.Parallel()

	world := newWorld(strange())

	stdout, stderr, code := world.run(
		"switch", "hdmi", "--unsafe-model", "AOC/Q27P1B", "--dry-run",
	)
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	for _, wanted := range []string{
		"WARNING: identification bypassed by --unsafe-model.",
		"Display: card1-HDMI-A-1 (XXX 0x2701)",
		"Assumed: AOC Q27P1B (write-enabled: no, evidence: reported)",
		"Input:   HDMI (0x11, vcp-input-source)",
		"never verified",
		"OSD",
	} {
		if !strings.Contains(stderr, wanted) {
			t.Errorf("the warning does not say %q:\n%s", wanted, stderr)
		}
	}

	if !strings.Contains(stdout, "Input:   HDMI (0x11)") {
		t.Errorf("the dry run does not carry the recorded value:\n%s", stdout)
	}

	if !strings.Contains(stdout, "Display: card1-HDMI-A-1 (AOC Q27P1B) (identification bypassed)") {
		t.Errorf("the dry run does not say the display was never identified:\n%s", stdout)
	}

	if !strings.Contains(stdout, "No DDC write was performed.") {
		t.Errorf("the dry run does not promise nothing was written:\n%s", stdout)
	}

	if world.wrote() {
		t.Error("a dry run wrote to a monitor")
	}
}

func TestSwitchWithUnsafeModelSaysSoOnTheSuccessLine(t *testing.T) {
	t.Parallel()

	world := newWorld(strange())

	stdout, stderr, code := world.run("switch", "hdmi", "--unsafe-model", "aoc q27p1b")
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	if !strings.Contains(stderr, "WARNING: identification bypassed") {
		t.Errorf("a real run printed no warning:\n%s", stderr)
	}

	// The banner has to name the monitor that is about to be written to, and a
	// real run has no dry-run block to say it instead.
	if !strings.Contains(stderr, "Display: card1-HDMI-A-1 (XXX 0x2701)") {
		t.Errorf("the warning does not name the display:\n%s", stderr)
	}

	want := "Input-switch command sent (HDMI, 0x11) to AOC Q27P1B via ddcutil " +
		"(identification bypassed). Switch not independently confirmed.\n"
	equal(t, "switch --unsafe-model", stdout, want)

	executed := world.driver.Executed()
	if len(executed) != 1 {
		t.Fatalf("the backend executed %d operations, want 1", len(executed))
	}

	if executed[0].Value() != 0x11 || executed[0].Mechanism() != catalog.MechanismInputSource {
		t.Errorf("the backend was handed %s", executed[0])
	}
}

// A normal switch must keep reading as one: the note belongs to the override.
func TestAnIdentifiedSwitchSaysNothingAboutBypassing(t *testing.T) {
	t.Parallel()

	world := newWorld()

	stdout, stderr, code := world.run("switch", "usb-c")
	if code != exitSent {
		t.Fatalf("exit = %d, stderr = %q", code, stderr)
	}

	if strings.Contains(stdout, "identification bypassed") || stderr != "" {
		t.Errorf("an identified switch mentioned the override:\n%s\n%s", stdout, stderr)
	}
}

func TestSwitchWithAnUnknownUnsafeModelIsAnArgumentError(t *testing.T) {
	t.Parallel()

	world := newWorld(strange())

	stdout, stderr, code := world.run(
		"switch", "hdmi", "--unsafe-model", "Acme/Nothing", "--dry-run",
	)
	if code != exitToolError {
		t.Fatalf("exit = %d, want %d", code, exitToolError)
	}

	if stdout != "" {
		t.Errorf("an unknown model printed %q", stdout)
	}

	if !strings.Contains(stderr, "monmux catalog list") {
		t.Errorf("the error does not point at the listing: %q", stderr)
	}

	if strings.Contains(stderr, "WARNING: identification bypassed") {
		t.Errorf("the warning was printed for a name that is not a model: %q", stderr)
	}

	if world.driver.Calls() != nil {
		t.Errorf("an argument error still reached the backend: %v", world.driver.Calls())
	}

	if world.wrote() {
		t.Error("an argument error wrote to a monitor")
	}
}

// The override takes the same names as "catalog show", and mistyping one is the
// same usage error with the same suggestion.
func TestSwitchWithAModelNameSuggestsTheFullName(t *testing.T) {
	t.Parallel()

	world := newWorld(strange())

	_, stderr, code := world.run("switch", "hdmi", "--unsafe-model", "Q27P1B", "--dry-run")
	if code != exitToolError {
		t.Fatalf("exit = %d, want %d", code, exitToolError)
	}

	if !strings.Contains(stderr, `did you mean "AOC/Q27P1B"?`) {
		t.Errorf("the error does not suggest the full name: %q", stderr)
	}

	if world.wrote() {
		t.Error("an argument error wrote to a monitor")
	}
}

// The override is a flag, every invocation: no configuration key arms it.
func TestTheConfigurationFileCannotArmTheOverride(t *testing.T) {
	t.Parallel()

	fields := reflect.VisibleFields(reflect.TypeFor[config.Config]())
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field.Name), "unsafe") ||
			strings.Contains(strings.ToLower(field.Name), "model") {
			t.Errorf("config.Config carries %q, which could arm the override", field.Name)
		}
	}
}
