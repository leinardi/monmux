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

package selectbackend

import (
	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/refusal"
)

// TODO(phase 7): construct the ddcutil backend here.
//
// newBackend returns the ddcutil backend. Until that backend lands, Linux has
// no backend compiled in and monmux refuses rather than pretending otherwise.
//
//nolint:ireturn // this is the constructor the Backend contract exists for
func newBackend(_ Options) (backend.Backend, error) {
	return nil, refusal.New(
		refusal.BackendUnavailable,
		"No ddcutil backend is compiled into this build.",
	)
}
