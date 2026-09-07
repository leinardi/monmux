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

package exec_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/leinardi/monmux/internal/backend/exec"
)

// The guard is the reason tests can never switch a monitor by accident. This
// test asks the real runner to run something harmless and asserts that it
// refuses purely because it is running inside a test binary.
func TestTheRealRunnerRefusesToRunInsideATest(t *testing.T) {
	t.Parallel()

	result, err := exec.NewRunner().Run(t.Context(), "/bin/true", nil)

	if !errors.Is(err, exec.ErrExecutionDisabled) {
		t.Fatalf("the real runner executed from a test: err = %v", err)
	}

	if !strings.Contains(err.Error(), "test binary") {
		t.Errorf("error does not say why: %v", err)
	}

	if result.Stdout != "" || result.Stderr != "" || result.ExitCode != -1 {
		t.Errorf("a refused run produced output: %+v", result)
	}
}

func TestExecutionDisabledExplainsItself(t *testing.T) {
	t.Parallel()

	err := exec.ExecutionDisabled()
	if !errors.Is(err, exec.ErrExecutionDisabled) {
		t.Fatalf("ExecutionDisabled() = %v inside a test", err)
	}
}

func TestStartedDistinguishesNeverRanFromExitedNonZero(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		result exec.Result
		err    error
		want   bool
	}{
		"clean run": {
			result: exec.Result{ExitCode: 0},
			err:    nil,
			want:   true,
		},
		"execution refused": {
			result: exec.Result{ExitCode: -1},
			err:    exec.ErrExecutionDisabled,
			want:   false,
		},
		"never started": {
			result: exec.Result{ExitCode: -1},
			err:    context.DeadlineExceeded,
			want:   false,
		},
		"exited non-zero": {
			result: exec.Result{ExitCode: 2},
			err:    context.DeadlineExceeded,
			want:   true,
		},
		// A tool that completed while the context was being canceled reports
		// an error with a real exit status. Its write may well have gone out,
		// so it must not be reported as "nothing happened".
		"canceled after completing": {
			result: exec.Result{ExitCode: 0},
			err:    context.Canceled,
			want:   true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			started := exec.Started(test.result, test.err)
			if started != test.want {
				t.Errorf(
					"Started(%+v, %v) = %t, want %t",
					test.result,
					test.err,
					started,
					test.want,
				)
			}
		})
	}
}

func TestFakeRecordsAndAnswers(t *testing.T) {
	t.Parallel()

	fake := exec.NewFake(
		exec.Rule{Match: "--version", Result: exec.Result{Stdout: "ddcutil 2.2.5"}},
		exec.Rule{Match: "detect", Err: context.Canceled},
	)

	result, err := fake.Run(t.Context(), "/usr/bin/ddcutil", []string{"--version"})
	if err != nil {
		t.Fatalf("fake returned %v", err)
	}

	if result.Stdout != "ddcutil 2.2.5" {
		t.Errorf("stdout = %q", result.Stdout)
	}

	_, err = fake.Run(t.Context(), "/usr/bin/ddcutil", []string{"detect"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("scripted error = %v, want context.Canceled", err)
	}

	_, err = fake.Run(t.Context(), "/usr/bin/ddcutil", []string{"setvcp"})
	if !errors.Is(err, exec.ErrUnexpectedCommand) {
		t.Errorf("an unscripted command returned %v", err)
	}

	want := []string{
		"/usr/bin/ddcutil --version",
		"/usr/bin/ddcutil detect",
		"/usr/bin/ddcutil setvcp",
	}

	lines := fake.Lines()
	if len(lines) != len(want) {
		t.Fatalf("recorded %d calls, want %d: %v", len(lines), len(want), lines)
	}

	for index, line := range lines {
		if line != want[index] {
			t.Errorf("call %d = %q, want %q", index, line, want[index])
		}
	}
}

func TestFakeCopiesTheArgumentsItRecords(t *testing.T) {
	t.Parallel()

	fake := exec.NewFake(exec.Rule{})

	args := []string{"setvcp", "0xF4"}

	_, err := fake.Run(t.Context(), "/usr/bin/ddcutil", args)
	if err != nil {
		t.Fatalf("fake returned %v", err)
	}

	args[1] = "0x60"

	if fake.Calls()[0].Args[1] != "0xF4" {
		t.Error("the fake kept a reference to the caller's slice")
	}
}

// The same command must be able to answer differently on a second call: a
// backend re-reads a display's identity inside Execute, and a test needs that
// second read to disagree with the first.
func TestFakeCanAnswerTheSameCommandTwiceDifferently(t *testing.T) {
	t.Parallel()

	fake := exec.NewFake(
		exec.Rule{Match: "display list", Result: exec.Result{Stdout: "first"}, Times: 1},
		exec.Rule{Match: "display list", Result: exec.Result{Stdout: "second"}},
	)

	want := []string{"first", "second", "second"}

	for _, expected := range want {
		result, err := fake.Run(t.Context(), "/opt/homebrew/bin/m1ddc", []string{"display", "list"})
		if err != nil {
			t.Fatalf("fake returned %v", err)
		}

		if result.Stdout != expected {
			t.Errorf("stdout = %q, want %q", result.Stdout, expected)
		}
	}
}

func TestFakeFuncAnswersEveryCall(t *testing.T) {
	t.Parallel()

	fake := exec.NewFake(exec.Rule{Match: "never", Err: exec.ErrUnexpectedCommand})
	fake.Func = func(call exec.Call) (exec.Result, error) {
		return exec.Result{Stdout: call.Line()}, nil
	}

	result, err := fake.Run(t.Context(), "/usr/bin/ddcutil", []string{"detect"})
	if err != nil {
		t.Fatalf("fake returned %v", err)
	}

	if result.Stdout != "/usr/bin/ddcutil detect" {
		t.Errorf("stdout = %q", result.Stdout)
	}
}
