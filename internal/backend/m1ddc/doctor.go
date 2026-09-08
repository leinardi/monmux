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
	"strings"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/refusal"
)

// defaultCheckCount is a starting size for the check list, not a limit.
const defaultCheckCount = 6

// Doctor reports what this backend can see. It is read-only from end to end: it
// resolves the binary, fingerprints it and asks it once what is attached. It
// writes nothing, and it never asks a monitor what its current input is.
func (b *Backend) Doctor(ctx context.Context) []backend.Check {
	checks := make([]backend.Check, 0, defaultCheckCount)

	path, err := b.resolvePath()
	if err != nil {
		// Without the binary there is nothing left to report: unlike the Linux
		// backend, everything monmux knows about a display on macOS comes from
		// the tool itself.
		return append(checks, backend.CheckFailed(Name+" binary", err))
	}

	checks = append(checks,
		backend.CheckPassed(Name+" binary", path),
		backend.FingerprintCheck(Name+" sha256", path),
		location(path),
	)

	records, err := b.list(ctx, refusal.BackendNotReady, path)
	if err != nil {
		return append(checks, backend.CheckFailed("displays", err))
	}

	checks = append(checks, backend.Check{
		Name:   "displays",
		OK:     len(records) > 0,
		Detail: backend.DisplayCount(len(records)),
	})

	for index := range records {
		current := &records[index].display
		checks = append(checks, backend.Check{
			Name:   current.Label,
			OK:     current.Writable,
			Detail: current.Status,
		})
	}

	return checks
}

// location reports whether m1ddc is installed where a package manager puts it.
//
// This is a note, not a verdict. A binary somewhere else may well be a
// deliberate local build, so monmux still runs it; but an unexpected location is
// the first thing worth knowing when the tool behaves strangely, and monmux
// cannot tell a genuine m1ddc from a correctly-permissioned replacement.
func location(path string) backend.Check {
	if installedWhereExpected(path) {
		return backend.CheckPassed(Name+" location", "under "+strings.Join(trustedPrefixes, " or "))
	}

	return backend.Check{
		Name:   Name + " location",
		OK:     false,
		Detail: "not under " + strings.Join(trustedPrefixes, " or ") + "; monmux will still run it",
	}
}
