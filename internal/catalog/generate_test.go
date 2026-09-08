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

package catalog_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/catalog"
	"github.com/leinardi/monmux/internal/catalog/internal/generate"
)

// The catalog is written in models.yaml and compiled from models_gen.go. If the
// two disagree, the bytes a reviewer read in the pull request are not the bytes
// the binary would send, so the build fails here rather than shipping.
func TestGeneratedCatalogMatchesTheYAML(t *testing.T) {
	t.Parallel()

	root := moduleRoot(t)

	source, err := os.ReadFile(filepath.Join(root, "internal", "catalog", "models.yaml"))
	if err != nil {
		t.Fatalf("reading models.yaml: %v", err)
	}

	committed, err := os.ReadFile(filepath.Join(root, "internal", "catalog", "models_gen.go"))
	if err != nil {
		t.Fatalf("reading models_gen.go: %v", err)
	}

	rendered, err := generate.Generate(source)
	if err != nil {
		t.Fatalf("models.yaml does not render: %v", err)
	}

	if bytes.Equal(rendered, committed) {
		return
	}

	t.Errorf(
		"models_gen.go is stale: run `make go-generate` and commit the result.\n%s",
		firstDifference(string(committed), string(rendered)),
	)
}

// firstDifference reports the first line where the committed file and a fresh
// rendering diverge, which is enough to see what was edited by hand.
func firstDifference(committed, rendered string) string {
	have := strings.Split(committed, "\n")
	want := strings.Split(rendered, "\n")

	for index := range max(len(have), len(want)) {
		haveLine := lineAt(have, index)
		wantLine := lineAt(want, index)

		if haveLine != wantLine {
			return fmt.Sprintf(
				"line %d:\n  committed: %s\n  rendered:  %s",
				index+1,
				haveLine,
				wantLine,
			)
		}
	}

	return "the files differ only in trailing bytes"
}

// lineAt returns a line, or a marker when the file ended before it.
func lineAt(lines []string, index int) string {
	if index >= len(lines) {
		return "<end of file>"
	}

	return lines[index]
}

// The generator does not import this package, so that a deleted or corrupt
// models_gen.go can still be regenerated. That decoupling costs it the mechanism
// and grade enums, which it mirrors instead. These two tests are where the
// copies are compared: adding a value on one side without the other fails here.
// Input names are not mirrored - both sides import the one grammar - so there is
// nothing to compare for them.
func TestGeneratorKnowsEveryGrade(t *testing.T) {
	t.Parallel()

	graded := make([]string, 0, len(catalog.Grades()))
	for _, grade := range catalog.Grades() {
		graded = append(graded, grade.String())
	}

	if !slices.Equal(graded, generate.Grades()) {
		t.Errorf(
			"the generator accepts grades %v, the enum has %v",
			generate.Grades(),
			graded,
		)
	}
}

func TestGeneratorKnowsEveryMechanism(t *testing.T) {
	t.Parallel()

	mechanisms := make([]string, 0, len(catalog.Mechanisms()))
	for _, mechanism := range catalog.Mechanisms() {
		mechanisms = append(mechanisms, mechanism.String())
	}

	if !slices.Equal(mechanisms, generate.Mechanisms()) {
		t.Errorf(
			"the generator accepts mechanisms %v, the enum has %v",
			generate.Mechanisms(),
			mechanisms,
		)
	}
}
