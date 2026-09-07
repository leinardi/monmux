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

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/refusal"
)

// Preflight checks the tool: that it is where it should be, that nobody else can
// rewrite it, and that it answers `display list detailed` in a way this backend
// understands. It resolves the absolute path every later call executes directly.
//
// m1ddc has no --version and no --help capability probe worth reading, so the
// display list is the probe. It is read-only, and it is also the command every
// other code path here depends on parsing, so a tool whose output changed is
// caught before a switch rather than during one.
//
// A Mac with nothing attached passes: m1ddc answers "No external display found"
// with a non-zero status, which is not a broken tool. Enumerate then reports no
// displays and policy refuses with no-displays, which is the accurate reason.
func (b *Backend) Preflight(ctx context.Context) error {
	path, err := b.resolvePath()
	if err != nil {
		return err
	}

	_, err = b.list(ctx, refusal.BackendNotReady, path)
	if err != nil {
		return err
	}

	b.rememberPath(path)

	return nil
}

// resolvePath finds the m1ddc binary and checks that it can be trusted. The
// checks themselves live in internal/backend, because the Linux backend applies
// exactly the same ones and a security check must not exist twice.
func (b *Backend) resolvePath() (string, error) {
	//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
	return backend.ResolveTool(Name, b.configured, "m1ddc_path")
}
