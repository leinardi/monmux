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

// Package toolchain_test pins the repository's tooling: the Go line and the golangci-lint version
// stay one value wherever they are written, and no gate carries a flag that makes it pass on
// nothing. It reads files only; like every test in monmux, it starts no process.
package toolchain_test

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/leinardi/monmux/test/internal/reporoot"
)

// githubYAML lists every YAML file under .github, as slash-separated paths relative to the
// repository root.
func githubYAML(t *testing.T) []string {
	t.Helper()

	root := reporoot.Find(t)

	var files []string

	err := filepath.WalkDir(
		filepath.Join(root, ".github"),
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !entry.IsDir() &&
				(strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return fmt.Errorf("relative path of %s: %w", path, relErr)
				}

				files = append(files, filepath.ToSlash(rel))
			}

			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("no YAML found under .github; every scan over it would check nothing")
	}

	return files
}

// makeFragments lists the .mk files directly under .mk, relative to the repository root.
func makeFragments(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(reporoot.Find(t), ".mk"))
	if err != nil {
		t.Fatalf("list .mk: %v", err)
	}

	var files []string

	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".mk") {
			files = append(files, ".mk/"+entry.Name())
		}
	}

	if len(files) == 0 {
		t.Fatal("no make fragments found under .mk; the scan would check nothing")
	}

	return files
}

// actionStep is one step, in a workflow or a composite action, that uses an action.
type actionStep struct {
	file string
	// action is the uses: value without its @ref.
	action string
	with   map[string]string
}

// githubActionSteps parses every YAML file under .github for steps that use an action.
func githubActionSteps(t *testing.T) []actionStep {
	t.Helper()

	var steps []actionStep

	for _, file := range githubYAML(t) {
		parsed, err := parseActionSteps(file, []byte(reporoot.ReadFile(t, file)))
		if err != nil {
			t.Fatal(err)
		}

		steps = append(steps, parsed...)
	}

	return steps
}

// parseActionSteps finds every mapping with a uses: key anywhere in the document, so job steps and
// composite action steps alike.
func parseActionSteps(file string, content []byte) ([]actionStep, error) {
	var document any

	err := yaml.Unmarshal(content, &document)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}

	var steps []actionStep

	var walk func(node any)

	walk = func(node any) {
		switch value := node.(type) {
		case map[string]any:
			if uses, ok := value["uses"].(string); ok {
				step := actionStep{
					file:   file,
					action: strings.SplitN(uses, "@", 2)[0],
					with:   map[string]string{},
				}

				inputs, _ := value["with"].(map[string]any)
				for key, input := range inputs {
					step.with[key] = fmt.Sprint(input)
				}

				steps = append(steps, step)
			}

			for _, child := range value {
				walk(child)
			}
		case []any:
			for _, child := range value {
				walk(child)
			}
		}
	}

	walk(document)

	return steps, nil
}
