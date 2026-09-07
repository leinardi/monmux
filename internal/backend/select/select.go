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

// Package selectbackend picks the backend for the operating system monmux is
// running on. It lives at internal/backend/select; the package is named
// selectbackend because select is a Go keyword and cannot name a package.
//
// It is a sibling of internal/backend rather than part of it: the build-tagged
// files here import the concrete backends, so keeping them out of the interface
// package is what stops an import cycle. Nothing chooses a backend by
// configuration - the operating system decides, and there is no override.
package selectbackend

import (
	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/backend/exec"
)

// Options carries what a backend needs that is not implied by the OS.
type Options struct {
	// Runner runs external tools. A nil Runner means the real one.
	Runner exec.Runner
	// DDCUtilPath optionally pins the ddcutil binary to an absolute path from
	// the configuration file, instead of resolving it from PATH.
	DDCUtilPath string
	// M1DDCPath optionally pins the m1ddc binary the same way.
	M1DDCPath string
}

// New returns the backend for this operating system, with the defaults.
//
//nolint:ireturn // the caller works through the Backend contract by design
func New() (backend.Backend, error) {
	return NewWithOptions(Options{})
}

// NewWithOptions returns the backend for this operating system. An operating
// system monmux has no backend for is a refusal, not a fallback.
//
//nolint:ireturn // the caller works through the Backend contract by design
func NewWithOptions(opts Options) (backend.Backend, error) {
	return newBackend(opts)
}
