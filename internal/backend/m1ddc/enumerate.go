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

package m1ddc

import (
	"context"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/refusal"
)

// Enumerate lists the attached displays. Displays monmux cannot write to are
// listed too, with Writable false and a Status saying why: info reports them,
// and policy refuses to select them.
//
// The order is m1ddc's own, because that is what its numbering means: the label
// "display 2" is only true while the list is in the order m1ddc printed it.
func (b *Backend) Enumerate(ctx context.Context) ([]backend.Display, error) {
	path, err := b.toolPath()
	if err != nil {
		return nil, err
	}

	records, err := b.list(ctx, refusal.EnumerationFailed, path)
	if err != nil {
		return nil, err
	}

	b.remember(writable(records))

	return displays(records), nil
}
