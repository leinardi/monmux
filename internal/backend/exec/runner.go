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

// Package exec is the only place in monmux that runs an external program.
//
// Everything a backend does goes through a [Runner], which exists so that tests
// can observe invocations without performing them. The real runner enforces that
// from the inside: it refuses to execute anything while a test binary is running,
// or when MONMUX_NO_EXEC is set. A test cannot switch a monitor by accident, even
// if it constructs the real runner and asks it to.
package exec

import (
	"context"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"testing"
)

// NoExecEnv disables execution when set to "1". The guard inside the runner is
// the real protection; this is a belt for scripts and one-off runs.
const NoExecEnv = "MONMUX_NO_EXEC"

// ErrExecutionDisabled is returned instead of running anything when execution is
// disabled. It is not a tool failure: nothing was started.
var ErrExecutionDisabled = errors.New("exec: execution is disabled")

// Result is what an external tool produced.
type Result struct {
	// Stdout is the tool's standard output.
	Stdout string
	// Stderr is the tool's standard error.
	Stderr string
	// ExitCode is the tool's exit status, or -1 when it never started.
	ExitCode int
}

// Runner runs an external program. The path is always absolute and already
// resolved: a runner performs no PATH lookup of its own.
type Runner interface {
	Run(ctx context.Context, path string, args []string) (Result, error)
}

// realRunner is the only implementation that starts a process.
type realRunner struct{}

// NewRunner returns the runner that actually executes programs. It refuses to
// run anything from a test binary or when MONMUX_NO_EXEC=1.
//
//nolint:ireturn // callers depend on the Runner contract, never on which one they hold
func NewRunner() Runner {
	return realRunner{}
}

// ExecutionDisabled reports whether this process may execute external programs
// at all, and why not when it may not.
func ExecutionDisabled() error {
	if testing.Testing() {
		return fmt.Errorf("%w: running under a test binary", ErrExecutionDisabled)
	}

	if os.Getenv(NoExecEnv) == "1" {
		return fmt.Errorf("%w: %s=1 is set", ErrExecutionDisabled, NoExecEnv)
	}

	return nil
}

// Run executes path with args and collects its output. The guard is checked here
// rather than at construction time, so no code path can hold a runner that was
// built before the check.
func (realRunner) Run(ctx context.Context, path string, args []string) (Result, error) {
	err := ExecutionDisabled()
	if err != nil {
		return Result{ExitCode: -1}, err
	}

	command := osexec.CommandContext(ctx, path, args...)

	var stdout, stderr strings.Builder

	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Run()

	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(command),
	}

	if err != nil {
		return result, fmt.Errorf("running %s: %w", path, err)
	}

	return result, nil
}

// exitCode returns the process's exit status, or -1 when it never ran.
func exitCode(command *osexec.Cmd) int {
	if command.ProcessState == nil {
		return -1
	}

	return command.ProcessState.ExitCode()
}

// Started reports whether the tool actually ran, which is the difference between
// "nothing was written" and "the write status is unknown".
//
// Not-started is recognized positively - execution refused, the binary could not
// be started, or no process state at all - and everything else counts as
// started. That default is deliberate: os/exec also reports a canceled context
// or a failed output copy for a process that ran to completion, and claiming
// nothing was written when something may have been is the one mistake this
// project must not make.
func Started(result Result, err error) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, ErrExecutionDisabled) {
		return false
	}

	if _, ok := errors.AsType[*osexec.Error](err); ok {
		return false
	}

	return result.ExitCode >= 0
}
