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

package toolchain_test

import (
	"fmt"
	"regexp"
	"slices"
	"testing"

	"github.com/leinardi/monmux/test/internal/reporoot"
)

// The actions whose inputs carry a shared pin.
const (
	setupGo       = "actions/setup-go"
	golangciLintA = "golangci/golangci-lint-action"
	// linterFromHook is how a golangci-lint-action step takes its version from the step that reads
	// the golangci-lint rev out of .pre-commit-config.yaml (ci.yaml and release.yaml).
	linterFromHook = "${{ steps.linter.outputs.version }}"
)

var (
	goDirective  = regexp.MustCompile(`(?m)^go (\d+\.\d+)(?:\.\d+)?$`)
	goPrereq     = regexp.MustCompile(`(?m)^- Go (\d+\.\d+)\+$`)
	hookLinterRe = regexp.MustCompile(
		`repo: https://github\.com/golangci/golangci-lint\s+rev:\s*(v[\d.]+)`,
	)
	exactTag = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
)

// TestGoLineIsOneValue: the go directive and the prerequisite CONTRIBUTING.md states name the same
// Go line. Every setup-go step reads it from go.mod (TestActionStepsArePinned), so go.mod is the
// one place CI's toolchain comes from.
func TestGoLineIsOneValue(t *testing.T) {
	t.Parallel()

	lines := map[string]string{}

	for file, pattern := range map[string]*regexp.Regexp{
		"go.mod":          goDirective,
		"CONTRIBUTING.md": goPrereq,
	} {
		match := pattern.FindStringSubmatch(reporoot.ReadFile(t, file))
		if match == nil {
			t.Errorf("%s does not state the Go line (%s)", file, pattern)

			continue
		}

		lines[file] = match[1]
	}

	if lines["go.mod"] != lines["CONTRIBUTING.md"] {
		t.Errorf("the Go line is not one value: %v", lines)
	}
}

// TestActionStepsArePinned: no setup-go step leaves its version to the action's default or writes
// one of its own, and every golangci-lint-action step installs the golangci-lint the pre-commit
// hook pins, either through the step that reads it or as the same exact tag.
func TestActionStepsArePinned(t *testing.T) {
	t.Parallel()

	hook := hookLinterRe.FindStringSubmatch(reporoot.ReadFile(t, ".pre-commit-config.yaml"))
	if hook == nil {
		t.Fatal(".pre-commit-config.yaml pins no golangci-lint rev")
	}

	steps := githubActionSteps(t)

	if !slices.ContainsFunc(steps, func(step actionStep) bool { return step.action == setupGo }) {
		t.Fatal("no setup-go step found under .github; the rule would check nothing")
	}

	for _, problem := range pinProblems(steps, hook[1]) {
		t.Error(problem)
	}
}

// TestActionPinRules is the negative control of the pin rules: pinned steps pass, and each
// unpinned form is a problem.
func TestActionPinRules(t *testing.T) {
	t.Parallel()

	steps, err := parseActionSteps("pinned.yaml", []byte(`jobs:
  lint:
    steps:
      - uses: actions/setup-go@0123456789abcdef0123456789abcdef01234567 # v7.0.0
        with:
          go-version-file: go.mod
          check-latest: true
      - uses: golangci/golangci-lint-action@0123456789abcdef0123456789abcdef01234567 # v9.3.0
        with:
          version: ${{ steps.linter.outputs.version }}
      - uses: golangci/golangci-lint-action@0123456789abcdef0123456789abcdef01234567 # v9.3.0
        with:
          version: v2.14.0
`))
	if err != nil {
		t.Fatal(err)
	}

	problems := pinProblems(steps, "v2.14.0")
	if len(problems) != 0 {
		t.Errorf("pinned steps reported: %q", problems)
	}

	steps, err = parseActionSteps("unpinned.yaml", []byte(`runs:
  using: composite
  steps:
    - uses: actions/setup-go@0123456789abcdef0123456789abcdef01234567
      with:
        go-version: "1.26"
    - uses: actions/setup-go@0123456789abcdef0123456789abcdef01234567
    - uses: actions/setup-go@0123456789abcdef0123456789abcdef01234567
      with:
        go-version-file: .go-version
    - uses: golangci/golangci-lint-action@0123456789abcdef0123456789abcdef01234567
      with:
        version: latest
    - uses: golangci/golangci-lint-action@0123456789abcdef0123456789abcdef01234567
      with:
        version: v2.13.0
    - uses: golangci/golangci-lint-action@0123456789abcdef0123456789abcdef01234567
    - uses: some/other-action@0123456789abcdef0123456789abcdef01234567
      with:
        go-version: "1.27"
`))
	if err != nil {
		t.Fatal(err)
	}

	// setup-go writing a version also fails the go.mod rule, and none of the three setup-go steps
	// sets check-latest, hence three problems more.
	const want = 11

	problems = pinProblems(steps, "v2.14.0")
	if len(problems) != want {
		t.Errorf("unpinned steps: want %d problems, got %d: %q", want, len(problems), problems)
	}
}

// pinProblems applies the pin rules to steps: no step writes a go-version, setup-go reads go.mod
// and sets check-latest (go.mod names only the 1.26 line, and the patch cached on the runner image
// can lag the newest release), and golangci-lint-action installs hookRev, by the hook-reading step
// or as that exact tag.
func pinProblems(steps []actionStep, hookRev string) []string {
	var problems []string

	for _, step := range steps {
		if step.with["go-version"] != "" {
			problems = append(problems, fmt.Sprintf(
				"%s: %s gets go-version %q; the Go line comes from go.mod",
				step.file, step.action, step.with["go-version"]))
		}

		if step.action == setupGo && step.with["go-version-file"] != "go.mod" {
			problems = append(problems, step.file+": setup-go must read go-version-file: go.mod")
		}

		if step.action == setupGo && step.with["check-latest"] != "true" {
			problems = append(problems, step.file+": setup-go must set check-latest: true")
		}

		if step.action != golangciLintA {
			continue
		}

		version := step.with["version"]

		switch {
		case version == linterFromHook:
		case exactTag.MatchString(version) && version == hookRev:
		default:
			problems = append(problems, fmt.Sprintf(
				"%s: golangci-lint-action installs %q; want %s or the hook's rev %s",
				step.file, version, linterFromHook, hookRev))
		}
	}

	return problems
}
