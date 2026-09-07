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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/leinardi/monmux/internal/refusal"
)

// writableByOthers are the write bits for group and other. A tool anyone can
// rewrite is not a tool monmux is willing to run.
const writableByOthers os.FileMode = 0o022

// ResolveTool finds the binary a backend wraps and decides whether it can be
// trusted enough to run. Both backends share it: this is a security check, and
// two copies of a security check drift.
//
// It resolves from PATH, or from an absolute path given in the configuration,
// follows symlinks, and refuses anything that is not a plain file or that either
// it or its directory lets somebody other than its owner rewrite.
//
// What it does not do is tell you the binary is genuine. PATH, and a configured
// path, are a trust boundary rather than a threat monmux mitigates: a ddcutil
// replaced with correct ownership and permissions passes every check here.
func ResolveTool(binary, configured, configKey string) (string, error) {
	candidate := configured

	if candidate == "" {
		found, err := exec.LookPath(binary)
		if err != nil {
			return "", refusal.New(
				refusal.BackendNotReady,
				binary+" was not found on PATH: "+err.Error(),
			)
		}

		candidate = found
	} else if !filepath.IsAbs(candidate) {
		return "", refusal.New(
			refusal.BackendNotReady,
			configKey+" must be an absolute path, and "+candidate+" is not.",
		)
	}

	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return "", refusal.New(
			refusal.BackendNotReady,
			"Could not resolve "+candidate+": "+err.Error(),
		)
	}

	// A symlink from a safe directory into a writable one is exactly the case
	// these checks exist for, so judge the file the link actually resolves to.
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", refusal.New(
			refusal.BackendNotReady,
			absolute+" cannot be inspected: "+err.Error(),
		)
	}

	err = trusted(resolved)
	if err != nil {
		return "", err
	}

	return resolved, nil
}

// trusted refuses a binary that is not a plain file, or that either it or its
// directory allows anyone but its owner to rewrite.
func trusted(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return refusal.New(refusal.BackendNotReady, path+" cannot be inspected: "+err.Error())
	}

	if !info.Mode().IsRegular() {
		return refusal.New(refusal.BackendNotReady, path+" is not a regular file.")
	}

	if info.Mode().Perm()&writableByOthers != 0 {
		return refusal.New(refusal.BackendNotReady, path+" is group- or world-writable.")
	}

	directory := filepath.Dir(path)

	parent, err := os.Stat(directory)
	if err != nil {
		return refusal.New(refusal.BackendNotReady, directory+" cannot be inspected: "+err.Error())
	}

	if parent.Mode().Perm()&writableByOthers != 0 {
		return refusal.New(refusal.BackendNotReady, directory+" is group- or world-writable.")
	}

	return nil
}

// Fingerprint returns the SHA-256 of a file, so doctor can show which binary it
// is about to run.
func Fingerprint(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	sum := sha256.Sum256(contents)

	return hex.EncodeToString(sum[:]), nil
}
