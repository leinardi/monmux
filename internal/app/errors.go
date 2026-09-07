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

package app

// UnknownWriteStatus is the sentence every [ExecutionError] ends with. It is the
// counterpart of a refusal's "No DDC write was performed.", and it is deliberately
// not that sentence: the tool ran, so monmux does not know what reached the
// monitor, and claiming otherwise would be the one lie this project must not tell.
const UnknownWriteStatus = "Whether the input-switch command reached the monitor is unknown."

// ExecutionError reports that the backend's tool was executed and failed.
//
// This is the one failure that is not a refusal. A refusal means nothing was
// written; this means monmux started a program that talks to the monitor and
// that program reported failure, so the write may or may not have gone out. The
// CLI maps it to its own exit code and prints [UnknownWriteStatus].
type ExecutionError struct {
	// Backend is the name of the backend whose tool failed.
	Backend string
	// Err is what the backend reported.
	Err error
}

// Error renders the failure, ending with the sentence about the write status.
func (e *ExecutionError) Error() string {
	return e.Backend + " failed: " + e.Err.Error() + ". " + UnknownWriteStatus
}

// Unwrap returns the backend's error.
func (e *ExecutionError) Unwrap() error {
	return e.Err
}
