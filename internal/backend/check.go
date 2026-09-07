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
	"errors"
	"strconv"
	"strings"

	"github.com/leinardi/monmux/internal/refusal"
)

// The helpers below render a [Check]. Every backend's Doctor produces the same
// shape of report, so they live here rather than once per backend: doctor output
// is what a user pastes into a bug report, and two backends whose diagnostics
// disagree in wording are two backends nobody can compare.

// CheckPassed renders a check that passed.
func CheckPassed(name, detail string) Check {
	return Check{Name: name, OK: true, Detail: detail}
}

// CheckResult turns an error into a check, keeping detail for the passing case.
func CheckResult(name string, err error, detail string) Check {
	if err != nil {
		return CheckFailed(name, err)
	}

	return CheckPassed(name, detail)
}

// CheckFailed renders a check that did not pass.
//
// Almost every error here is a refusal, whose rendered form opens with the
// headline "Refusing to switch input." - true, but useless as a diagnostic. The
// reason and the detail are what the user needs, so they are what doctor shows.
// Identities are never printed here at all.
func CheckFailed(name string, err error) Check {
	var declined *refusal.Refusal

	if errors.As(err, &declined) {
		detail := declined.Reason.String()
		if declined.Detail != "" {
			detail += ": " + FirstLine(declined.Detail)
		}

		return Check{Name: name, OK: false, Detail: detail}
	}

	return Check{Name: name, OK: false, Detail: FirstLine(err.Error())}
}

// FingerprintCheck reports the SHA-256 of the binary a backend would run, so a
// bug report says which build was involved.
func FingerprintCheck(name, path string) Check {
	sum, err := Fingerprint(path)
	if err != nil {
		return CheckFailed(name, err)
	}

	return CheckPassed(name, sum)
}

// DisplayCount renders how many displays were found.
func DisplayCount(count int) string {
	if count == 1 {
		return "1 connected display"
	}

	return strconv.Itoa(count) + " connected displays"
}

// FirstLine returns the first line of a multi-line message, for the one-line
// shape doctor output has.
func FirstLine(message string) string {
	before, _, found := strings.Cut(message, "\n")
	if !found {
		return message
	}

	return before
}
