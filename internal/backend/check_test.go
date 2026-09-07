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

package backend_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/edid"
	"github.com/leinardi/monmux/internal/refusal"
)

// errRead is a plain error, the kind a check reports when nothing refused.
var errRead = errors.New("reading /usr/bin/ddcutil\nsecond line")

// A refusal renders with the headline "Refusing to switch input." - true, but
// useless in a diagnostic list, and it carries identities monmux redacts. What
// doctor shows is the reason and the first line of the detail, and nothing else.
func TestCheckFailedShowsTheReasonAndNoIdentity(t *testing.T) {
	t.Parallel()

	declined := refusal.New(
		refusal.TargetNotReady,
		"/dev/i2c-5 cannot be opened.\nAdd yourself to the i2c group.",
		edid.Identity{Manufacturer: "GSM", ProductCode: 0x77D4, SerialString: "TESTSERIAL01"},
	)

	check := backend.CheckFailed("card1-DP-1", declined)

	if check.OK {
		t.Error("a refusal was reported as a passing check")
	}

	if !strings.HasPrefix(check.Detail, refusal.TargetNotReady.String()+": ") {
		t.Errorf("detail = %q, want it to open with the reason", check.Detail)
	}

	if strings.Contains(check.Detail, "\n") {
		t.Errorf("detail spans more than one line: %q", check.Detail)
	}

	if strings.Contains(check.Detail, "TESTSERIAL01") {
		t.Errorf("detail leaked a serial: %q", check.Detail)
	}
}

func TestCheckFailedUsesAPlainErrorAsItIs(t *testing.T) {
	t.Parallel()

	check := backend.CheckFailed("ddcutil sha256", errRead)

	if check.OK || check.Detail != "reading /usr/bin/ddcutil" {
		t.Errorf("check = %+v", check)
	}
}

func TestCheckResultKeepsTheDetailWhenItPasses(t *testing.T) {
	t.Parallel()

	passed := backend.CheckResult("ddcutil version", nil, "at least 2.2")
	if !passed.OK || passed.Detail != "at least 2.2" {
		t.Errorf("check = %+v", passed)
	}

	failed := backend.CheckResult(
		"ddcutil version",
		refusal.New(refusal.BackendNotReady, "too old"),
		"at least 2.2",
	)
	if failed.OK || !strings.Contains(failed.Detail, "too old") {
		t.Errorf("check = %+v", failed)
	}
}

func TestDisplayCountReadsAsEnglish(t *testing.T) {
	t.Parallel()

	counts := map[int]string{
		0: "0 connected displays",
		1: "1 connected display",
		2: "2 connected displays",
	}

	for count, want := range counts {
		got := backend.DisplayCount(count)
		if got != want {
			t.Errorf("DisplayCount(%d) = %q, want %q", count, got, want)
		}
	}
}

func TestFirstLine(t *testing.T) {
	t.Parallel()

	lines := map[string]string{
		"":                  "",
		"one line":          "one line",
		"first\nsecond":     "first",
		"first\n\nthird":    "first",
		"\nleading newline": "",
	}

	for message, want := range lines {
		got := backend.FirstLine(message)
		if got != want {
			t.Errorf("FirstLine(%q) = %q, want %q", message, got, want)
		}
	}
}
