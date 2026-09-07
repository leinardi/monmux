//go:build darwin

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
	"github.com/leinardi/monmux/internal/backend/m1ddc"
)

// newBackend returns the m1ddc backend, which is the only one macOS has.
//
//nolint:ireturn // this is the constructor the Backend contract exists for
func newBackend(opts Options) (backend.Backend, error) {
	return m1ddc.New(m1ddc.Options{
		Runner: opts.Runner,
		Path:   opts.M1DDCPath,
	}), nil
}
