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
	"regexp"
	"strings"
	"testing"

	"github.com/leinardi/monmux/test/internal/reporoot"
)

// vacuousFlags turn a gate with nothing to check, or a failing one, into a pass. None of them may
// appear where gates are defined, unless allowedVacuous names the exact line and says why.
var vacuousFlags = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"|| true", regexp.MustCompile(`\|\|\s*(true\b|:(\s|;|$)|exit\s+0\b)`)},
	{"make's - recipe prefix", regexp.MustCompile(`^\t[@+]*-`)},
	{".IGNORE", regexp.MustCompile(`^\.IGNORE\b`)},
	{
		"make --ignore-errors",
		regexp.MustCompile(`\$\(MAKE\).*\s(-[a-zA-Z]*i[a-zA-Z]*|--ignore-errors)\b`),
	},
	{"continue-on-error", regexp.MustCompile(`continue-on-error`)},
	{
		"if: false",
		regexp.MustCompile(`\bif:\s*(["']?false["']?\s*$|["']?\$\{\{\s*false\s*\}\}["']?)`),
	},
}

// allowedVacuous holds the lines, by file and trimmed text, where a flag is the logic rather than
// a bypass. Each entry must still match a line: a stale one fails, so the list cannot quietly
// outlive the code it excuses.
var allowedVacuous = map[string]map[string]string{
	".github/workflows/release.yaml": {
		// grep exits 1 when no tag matches, which under pipefail would end the step; an empty
		// tag list is a valid answer here, and the callers test for it.
		`{ grep -E "$RELEASE_TAG_REGEX" || true; } | sort -V | tail -n1`:                        "no release tag yet",
		`git tag --points-at HEAD | { grep -E "$RELEASE_TAG_REGEX" || true; } | grep -qxF "$1"`: "HEAD carries no release tag",
		// HEAD^ does not exist on a root commit; the baseline is then empty, which is handled.
		`PREVIOUS=$({ git tag --merged "HEAD^" 2>/dev/null || true; } | highest_release)`: "root commit",
		// The remote has no release tag yet, so the pipeline finds nothing; checked right after.
		`| tail -n1 || true)`: "no remote release tag yet",
	},
}

// TestNoVacuousFlags scans the files that define gates: the Makefile and its fragments, the
// pre-commit configuration and every YAML file under .github, the workflows included.
func TestNoVacuousFlags(t *testing.T) {
	t.Parallel()

	fragments := makeFragments(t)
	workflows := githubYAML(t)

	files := make([]string, 0, 2+len(fragments)+len(workflows))
	files = append(files, "Makefile", ".pre-commit-config.yaml")
	files = append(files, fragments...)
	files = append(files, workflows...)

	used := map[string]map[string]bool{}

	for _, rel := range files {
		allowed := allowedVacuous[rel]

		for _, hit := range vacuousHits(reporoot.ReadFile(t, rel)) {
			if _, ok := allowed[hit.text]; ok {
				if used[rel] == nil {
					used[rel] = map[string]bool{}
				}

				used[rel][hit.text] = true

				continue
			}

			t.Errorf("%s:%d: %s: %s", rel, hit.line, hit.flag, hit.text)
		}
	}

	for rel, lines := range allowedVacuous {
		for line := range lines {
			if !used[rel][line] {
				t.Errorf(
					"%s: the allowed line %q is gone; remove it from allowedVacuous",
					rel,
					line,
				)
			}
		}
	}
}

// TestVacuousFlagsAreRecognized is the negative control: each flag, written as a gate would carry
// it, is found.
func TestVacuousFlagsAreRecognized(t *testing.T) {
	t.Parallel()

	for _, sample := range []string{
		"\tgo vet ./... || true",
		"\tgo vet ./... ||:",
		"\tgo vet ./... || exit 0",
		"\t-go vet ./...",
		"\t@-go vet ./...",
		".IGNORE: go-lint",
		"\t$(MAKE) -i go-test",
		"\t$(MAKE) --ignore-errors go-test",
		"        continue-on-error: true",
		"    if: false",
		"    if: ${{ false }}",
		"    if: 'false'",
		"    if: \"${{ false }}\"",
	} {
		if len(vacuousHits(sample)) == 0 {
			t.Errorf("no vacuous flag found in %q", sample)
		}
	}
}

type vacuousHit struct {
	line int
	flag string
	text string
}

func vacuousHits(content string) []vacuousHit {
	var hits []vacuousHit

	for index, line := range strings.Split(content, "\n") {
		for _, flag := range vacuousFlags {
			if flag.pattern.MatchString(line) {
				hits = append(hits, vacuousHit{index + 1, flag.name, strings.TrimSpace(line)})
			}
		}
	}

	return hits
}

// TestVacuousHitLinesAreOneBased pins the line numbers the failures print.
func TestVacuousHitLinesAreOneBased(t *testing.T) {
	t.Parallel()

	hits := vacuousHits("ok\n\tgo vet ./... || true\n")
	if len(hits) != 1 || hits[0].line != 2 {
		t.Errorf("hits = %v, want one on line 2", hits)
	}
}
