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
	"context"

	"github.com/leinardi/monmux/internal/backend"
)

// Doctor reports what this backend can see. It is read-only from end to end: it
// resolves the binary, asks it for its version and its options, lists the
// connectors and checks whether each one could be written to. It writes nothing.
func (b *Backend) Doctor(ctx context.Context) []backend.Check {
	checks := make([]backend.Check, 0, defaultCheckCount)

	path, err := b.resolvePath()
	if err != nil {
		// The tool is missing or untrusted, but the displays are still worth
		// reporting: whether the monitor is seen at all, and whether its bus
		// could be opened, are exactly what the user needs next.
		checks = append(checks, backend.CheckFailed("ddcutil binary", err))

		return append(checks, b.displayChecks(ctx)...)
	}

	checks = append(checks,
		backend.CheckPassed("ddcutil binary", path),
		backend.FingerprintCheck("ddcutil sha256", path),
	)

	err = b.checkVersion(ctx, path)
	checks = append(checks, backend.CheckResult("ddcutil version", err, "at least 2.2"))

	err = b.checkCapabilities(ctx, path)
	checks = append(
		checks,
		backend.CheckResult("ddcutil options", err, "--edid, --i2c-source-addr, --noverify"),
	)

	return append(checks, b.displayChecks(ctx)...)
}

// defaultCheckCount is a starting size for the check list, not a limit.
const defaultCheckCount = 6

// displayChecks reports one line per connected display.
func (b *Backend) displayChecks(ctx context.Context) []backend.Check {
	displays, err := b.Enumerate(ctx)
	if err != nil {
		return []backend.Check{backend.CheckFailed("displays", err)}
	}

	checks := make([]backend.Check, 0, len(displays)+1)
	checks = append(checks, backend.Check{
		Name:   "displays",
		OK:     len(displays) > 0,
		Detail: backend.DisplayCount(len(displays)),
	})

	for _, display := range displays {
		if !display.Writable {
			checks = append(checks, backend.Check{
				Name:   display.Label,
				OK:     false,
				Detail: display.Status,
			})

			continue
		}

		err = b.Ready(ctx, display)
		checks = append(checks, backend.CheckResult(display.Label, err, display.Status))
	}

	return checks
}
