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
	"os"
	"path/filepath"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/ddcutil"
	"github.com/leinardi/monmux/internal/backend/exec"
	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/edid"
)

const (
	connector = "card1-DP-1"
	bus       = "i2c-5"
	fixture   = "../testdata/lg-38wr85qc-w.edid"

	// helpOutput lists the options monmux's plan depends on.
	helpOutput = "  -e, --edid=256 char hex string\n  --i2c-source-addr=source address\n  --noverify\n"
	// versionOutput is what ddcutil 2.2.5 reports.
	versionOutput = "ddcutil 2.2.5\nCopyright (C) 2015-2026 Sanford Rockowitz\n"
)

// world is a fake system: a sysfs tree, a /dev tree, and a ddcutil binary that
// is never executed because the runner is a fake.
type world struct {
	sysfs   string
	dev     string
	binary  string
	runner  *exec.Fake
	backend *ddcutil.Backend
}

// newWorld builds a system with one connected, supported, writable display.
func newWorld(t *testing.T) *world {
	t.Helper()

	root := t.TempDir()

	built := &world{
		sysfs:  filepath.Join(root, "sys"),
		dev:    filepath.Join(root, "dev"),
		binary: filepath.Join(root, "bin", ddcutil.Name),
		runner: exec.NewFake(
			exec.Rule{Match: "--version", Result: exec.Result{Stdout: versionOutput}},
			exec.Rule{Match: "--help", Result: exec.Result{Stdout: helpOutput}},
			exec.Rule{Match: "setvcp", Result: exec.Result{}},
		),
	}

	mkdir(t, filepath.Dir(built.binary))
	write(t, built.binary, []byte("#!/bin/false\n"), 0o755)

	built.addDisplay(t, connector, bus, edidFixture(t))
	built.addBus(t, bus)

	built.backend = ddcutil.New(ddcutil.Options{
		Runner:    built.runner,
		Path:      built.binary,
		SysfsRoot: built.sysfs,
		DevRoot:   built.dev,
	})

	return built
}

// addDisplay adds a connected connector whose ddc link points at a bus.
func (w *world) addDisplay(t *testing.T, name, target string, raw []byte) {
	t.Helper()

	directory := filepath.Join(w.sysfs, name)
	mkdir(t, directory)
	write(t, filepath.Join(directory, "status"), []byte("connected\n"), 0o644)
	write(t, filepath.Join(directory, "edid"), raw, 0o644)

	if target == "" {
		return
	}

	err := os.Symlink("../../../"+target, filepath.Join(directory, "ddc"))
	if err != nil {
		t.Fatalf("linking %s to %s: %v", name, target, err)
	}
}

// addBus adds a readable and writable stand-in for /dev/i2c-N.
func (w *world) addBus(t *testing.T, name string) {
	t.Helper()

	mkdir(t, w.dev)
	write(t, filepath.Join(w.dev, name), nil, 0o600)
}

// ready runs preflight and enumeration, which every write path depends on.
func (w *world) ready(t *testing.T) []backend.Display {
	t.Helper()

	err := w.backend.Preflight(t.Context())
	if err != nil {
		t.Fatalf("Preflight() refused a good system: %v", err)
	}

	displays, err := w.backend.Enumerate(t.Context())
	if err != nil {
		t.Fatalf("Enumerate() failed: %v", err)
	}

	return displays
}

func mkdir(t *testing.T, path string) {
	t.Helper()

	err := os.MkdirAll(path, 0o755)
	if err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
}

func write(t *testing.T, path string, contents []byte, mode os.FileMode) {
	t.Helper()

	err := os.WriteFile(path, contents, mode)
	if err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func edidFixture(t *testing.T) []byte {
	t.Helper()

	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("reading %s: %v", fixture, err)
	}

	return raw
}

// operation returns the catalog's USB-C operation for the tested model.
func operation(t *testing.T) catalog.Operation {
	t.Helper()

	model, match := catalog.Match(edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D3})
	if match != catalog.MatchExact {
		t.Fatalf("the tested model no longer matches: %s", match)
	}

	op, enabled := model.Operation(catalog.InputUSBC)
	if !enabled {
		t.Fatal("the tested model is no longer enabled for USB-C")
	}

	return op
}

// addDisconnected adds a connector with no monitor attached.
func (w *world) addDisconnected(t *testing.T, name string) {
	t.Helper()

	directory := filepath.Join(w.sysfs, name)
	mkdir(t, directory)
	write(t, filepath.Join(directory, "status"), []byte("disconnected\n"), 0o644)
}

// rebuild returns a backend over the same fake system with a different runner or
// a different configured tool path.
func (w *world) rebuild(t *testing.T, runner *exec.Fake, path string) *ddcutil.Backend {
	t.Helper()

	w.runner = runner

	return ddcutil.New(ddcutil.Options{
		Runner:    runner,
		Path:      path,
		SysfsRoot: w.sysfs,
		DevRoot:   w.dev,
	})
}

func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()

	err := os.Chmod(path, mode)
	if err != nil {
		t.Fatalf("changing the mode of %s: %v", path, err)
	}
}
