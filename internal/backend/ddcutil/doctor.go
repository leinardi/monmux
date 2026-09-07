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
	"errors"
	"strconv"
	"strings"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/refusal"
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
		checks = append(checks, failed("ddcutil binary", err))

		return append(checks, b.displayChecks(ctx)...)
	}

	checks = append(checks,
		backend.Check{Name: "ddcutil binary", OK: true, Detail: path},
		fingerprintCheck(path),
	)

	err = b.checkVersion(ctx, path)
	checks = append(checks, result("ddcutil version", err, "at least 2.2"))

	err = b.checkCapabilities(ctx, path)
	checks = append(checks, result("ddcutil options", err, "--edid, --i2c-source-addr, --noverify"))

	return append(checks, b.displayChecks(ctx)...)
}

// defaultCheckCount is a starting size for the check list, not a limit.
const defaultCheckCount = 6

// displayChecks reports one line per connected display.
func (b *Backend) displayChecks(ctx context.Context) []backend.Check {
	displays, err := b.Enumerate(ctx)
	if err != nil {
		return []backend.Check{failed("displays", err)}
	}

	checks := make([]backend.Check, 0, len(displays)+1)
	checks = append(checks, backend.Check{
		Name:   "displays",
		OK:     len(displays) > 0,
		Detail: plural(len(displays)),
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
		checks = append(checks, result(display.Label, err, display.Status))
	}

	return checks
}

// result turns an error into a check, keeping the detail for the passing case.
func result(name string, err error, detail string) backend.Check {
	if err != nil {
		return failed(name, err)
	}

	return backend.Check{Name: name, OK: true, Detail: detail}
}

// failed renders a check that did not pass.
//
// Almost every error here is a refusal, whose rendered form opens with the
// headline "Refusing to switch input." - true, but useless as a diagnostic. The
// reason and the detail are what the user needs, so they are what doctor shows.
// Identities are never printed here at all.
func failed(name string, err error) backend.Check {
	var declined *refusal.Refusal

	if errors.As(err, &declined) {
		detail := declined.Reason.String()
		if declined.Detail != "" {
			detail += ": " + firstLine(declined.Detail)
		}

		return backend.Check{Name: name, OK: false, Detail: detail}
	}

	return backend.Check{Name: name, OK: false, Detail: firstLine(err.Error())}
}

// fingerprintCheck reports the SHA-256 of the binary that would be run.
func fingerprintCheck(path string) backend.Check {
	sum, err := fingerprint(path)
	if err != nil {
		return failed("ddcutil sha256", err)
	}

	return backend.Check{Name: "ddcutil sha256", OK: true, Detail: sum}
}

// plural renders the display count.
func plural(count int) string {
	if count == 1 {
		return "1 connected display"
	}

	return itoa(count) + " connected displays"
}

// itoa is strconv.Itoa under a shorter name, kept here so doctor.go reads as
// prose.
func itoa(value int) string {
	return strconv.Itoa(value)
}

// firstLine returns the first line of a multi-line message, for the one-line
// shape doctor output has.
func firstLine(message string) string {
	before, _, ok := strings.Cut(message, "\n")
	if !ok {
		return message
	}

	return before
}
