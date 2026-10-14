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

package docs_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/test/docs"
	"github.com/leinardi/monmux/test/internal/reporoot"
)

// skippedDirs are directories that hold no Markdown the project authors: git's own, build output
// and local tool state. They are skipped wherever they appear.
var skippedDirs = []string{
	".git",
	".idea",
	".cache",
	"dist",
	"dist-notes",
	"node_modules",
	"vendor",
}

// TestRelativeLinksResolve checks every relative link and fragment in the project's Markdown.
func TestRelativeLinksResolve(t *testing.T) {
	t.Parallel()

	root := reporoot.Find(t)
	checker := docs.NewChecker(root)

	files := markdownFiles(t, root)

	// A walk that finds nothing would pass while checking nothing.
	for _, required := range []string{"README.md", "AGENTS.md", "docs/architecture.md"} {
		if !slices.Contains(files, required) {
			t.Fatalf("the walk did not reach %s; it found %v", required, files)
		}
	}

	for _, rel := range files {
		problems, err := checker.CheckFile(rel)
		if err != nil {
			t.Fatal(err)
		}

		for _, problem := range problems {
			t.Error(problem)
		}
	}
}

// TestCheckerFindsBrokenLinks proves the link check is not blind, on a scratch tree.
func TestCheckerFindsBrokenLinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixture(t, dir, "a.md", strings.Join([]string{
		"# Title",
		"## 1. Section — `monmux switch`",
		"[ok](b.md#heading-two) [ok](#1-section--monmux-switch) [ok](https://example.com/missing)",
		"[broken file](missing.md) [broken anchor](b.md#nope) `[in code](missing-too.md)`",
		"```text",
		"[in a fence](missing-three.md)",
		"```",
		"[outside](../outside.md)",
	}, "\n"))
	writeFixture(t, dir, "b.md", "# Heading Two\n")

	problems, err := docs.NewChecker(dir).CheckFile("a.md")
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(problems))
	for _, problem := range problems {
		got = append(got, problem.String())
	}

	want := []string{
		`a.md:4: link "missing.md": missing.md does not exist`,
		`a.md:4: link "b.md#nope": b.md has no heading or anchor #nope`,
		`a.md:8: link "../outside.md" leaves the repository`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

// TestSlug pins the GitHub slug rules the anchors follow.
func TestSlug(t *testing.T) {
	t.Parallel()

	for heading, want := range map[string]string{
		"HARD RULE: no monitor writes by any AI agent": "hard-rule-no-monitor-writes-by-any-ai-agent",
		"Linux, on DisplayPort":                        "linux-on-displayport",
		"Hardware validation — for a human, by hand":   "hardware-validation--for-a-human-by-hand",
		"`--unsafe-model`":                             "--unsafe-model",
		"**Bold** and _emphasis_ with snake_case":      "bold-and-emphasis-with-snake_case",
		"[Linked](other.md) heading":                   "linked-heading",
		"Ünïcode Lëtters Stay":                         "ünïcode-lëtters-stay",
	} {
		if got := docs.Slug(heading); got != want {
			t.Errorf("Slug(%q) = %q, want %q", heading, got, want)
		}
	}
}

// markdownFiles walks the repository for Markdown, as sorted slash-separated paths relative to
// root. The walk does not follow symlinks, so .claude/skills (a link to .agents/skills) is read
// once, under .agents.
func markdownFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if path != root && slices.Contains(skippedDirs, entry.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".md") {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return fmt.Errorf("relative path of %s: %w", path, relErr)
			}

			files = append(files, filepath.ToSlash(rel))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walk the repository: %v", err)
	}

	slices.Sort(files)

	return files
}

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600)
	if err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
