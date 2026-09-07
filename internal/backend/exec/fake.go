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

package exec

import (
	"context"
	"errors"
	"strings"
	"sync"
)

// ErrUnexpectedCommand is returned by a [Fake] asked to run something no rule
// covers. Tests fail loudly on an unplanned invocation rather than silently
// receiving empty output.
var ErrUnexpectedCommand = errors.New("exec: the fake runner has no rule for this command")

// Rule tells a [Fake] what to answer for commands whose rendered form contains
// Match. An empty Match matches everything.
type Rule struct {
	// Match is a substring of "<path> <args...>".
	Match string
	// Result is what the fake returns for a matching command.
	Result Result
	// Err is the error the fake returns alongside Result.
	Err error
	// Times limits how often this rule answers. Zero means no limit; a rule
	// with Times 1 answers once and then lets the next matching rule take over,
	// which is how a test makes the same command return something different the
	// second time - the identity-changed case, for instance.
	Times int
}

// Call is one invocation a [Fake] recorded.
type Call struct {
	// Path is the program the caller asked for.
	Path string
	// Args are the arguments it was given.
	Args []string
}

// Line renders the call the way a rule matches against it.
func (c Call) Line() string {
	return strings.Join(append([]string{c.Path}, c.Args...), " ")
}

// Fake is a Runner that records what it was asked to run and answers from a
// script. It never starts a process, which is what makes it the only runner
// tests use.
type Fake struct {
	// Func, when set, answers every call and the rules are not consulted. It is
	// for tests whose answer depends on the arguments rather than on a fixed
	// script.
	Func func(Call) (Result, error)

	mu    sync.Mutex
	rules []Rule
	used  []int
	calls []Call
}

// NewFake returns a fake runner answering according to rules, in order: the
// first rule whose Match is contained in the command line wins.
func NewFake(rules ...Rule) *Fake {
	return &Fake{rules: rules, used: make([]int, len(rules))}
}

// Run records the invocation and answers from the script.
func (f *Fake) Run(_ context.Context, path string, args []string) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	call := Call{Path: path, Args: append([]string(nil), args...)}
	f.calls = append(f.calls, call)

	if f.Func != nil {
		return f.Func(call)
	}

	line := call.Line()

	for index, rule := range f.rules {
		if rule.Match != "" && !strings.Contains(line, rule.Match) {
			continue
		}

		if rule.Times > 0 && f.used[index] >= rule.Times {
			continue
		}

		f.used[index]++

		return rule.Result, rule.Err
	}

	return Result{ExitCode: -1}, ErrUnexpectedCommand
}

// Calls returns everything the fake was asked to run, in order.
func (f *Fake) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]Call(nil), f.calls...)
}

// Lines returns the rendered command line of every recorded call.
func (f *Fake) Lines() []string {
	calls := f.Calls()

	lines := make([]string, 0, len(calls))
	for _, call := range calls {
		lines = append(lines, call.Line())
	}

	return lines
}
