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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/refusal"
)

const (
	// minMajor and minMinor are the oldest ddcutil monmux will use. 2.2 is a
	// concrete floor rather than a guess: it is the series the LG side channel
	// was verified against, and --i2c-source-addr is not in older releases.
	minMajor = 2
	minMinor = 2
)

// requiredOptions are the options monmux's plan depends on. Their presence in
// --help is a read-only capability probe: it costs nothing and it fails early
// rather than in the middle of a write.
var requiredOptions = []string{"--edid", "--i2c-source-addr", "--noverify"}

// versionPattern finds the version in ddcutil's --version output.
var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`)

// Preflight checks the tool: that it is where it should be, that nobody else can
// rewrite it, that it is recent enough, and that it understands the options the
// plan uses. It resolves the absolute path every later call executes directly.
func (b *Backend) Preflight(ctx context.Context) error {
	path, err := b.resolvePath()
	if err != nil {
		return err
	}

	err = b.checkVersion(ctx, path)
	if err != nil {
		return err
	}

	err = b.checkCapabilities(ctx, path)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.path = path

	return nil
}

// resolvePath finds the ddcutil binary and checks that it can be trusted. The
// checks themselves live in internal/backend, because the macOS backend applies
// exactly the same ones and a security check must not exist twice.
func (b *Backend) resolvePath() (string, error) {
	//nolint:wrapcheck // a refusal is passed through unchanged; wrapping would corrupt its message
	return backend.ResolveTool(Name, b.configured, "ddcutil_path")
}

// checkVersion runs ddcutil --version, a read-only call, and refuses anything
// older than the floor.
func (b *Backend) checkVersion(ctx context.Context, path string) error {
	result, err := b.runner.Run(ctx, path, []string{"--version"})
	if err != nil {
		return refusal.New(refusal.BackendNotReady, path+" --version failed: "+err.Error())
	}

	major, minor, err := parseVersion(result.Stdout + result.Stderr)
	if err != nil {
		return err
	}

	if major < minMajor || (major == minMajor && minor < minMinor) {
		return refusal.New(
			refusal.BackendNotReady,
			fmt.Sprintf(
				"ddcutil %d.%d is older than the required %d.%d.",
				major,
				minor,
				minMajor,
				minMinor,
			),
		)
	}

	return nil
}

// parseVersion reads the first version number in ddcutil's --version output.
func parseVersion(output string) (major, minor int, err error) {
	match := versionPattern.FindStringSubmatch(output)
	if match == nil {
		return 0, 0, refusal.New(
			refusal.BackendNotReady,
			"Could not read a version number from ddcutil --version.",
		)
	}

	major, err = strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, refusal.New(refusal.BackendNotReady, "Unreadable ddcutil version: "+match[0])
	}

	minor, err = strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, refusal.New(refusal.BackendNotReady, "Unreadable ddcutil version: "+match[0])
	}

	return major, minor, nil
}

// checkCapabilities confirms this build of ddcutil understands the options the
// plan uses, by reading --help. Nothing is written and no monitor is touched.
func (b *Backend) checkCapabilities(ctx context.Context, path string) error {
	result, err := b.runner.Run(ctx, path, []string{"--help"})
	if err != nil {
		return refusal.New(refusal.BackendNotReady, path+" --help failed: "+err.Error())
	}

	help := result.Stdout + result.Stderr

	missing := make([]string, 0, len(requiredOptions))

	for _, option := range requiredOptions {
		if !strings.Contains(help, option) {
			missing = append(missing, option)
		}
	}

	if len(missing) > 0 {
		return refusal.New(
			refusal.BackendNotReady,
			"This ddcutil does not support "+strings.Join(missing, ", ")+".",
		)
	}

	return nil
}
