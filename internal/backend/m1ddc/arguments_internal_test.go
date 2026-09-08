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
	"slices"
	"testing"

	"github.com/leinardi/monmux/internal/catalog"
)

// The end-to-end golden test in decide_test.go pins what the tested model's
// operation produces. This one pins the argv for a value the catalog does not
// currently enable, which no exported constructor could build an Operation for -
// rendering argv is not the same power as making a backend run it.
func TestArgumentsAreTheSetInvocation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mechanism catalog.Mechanism
		value     uint16
		want      []string
	}{
		"a value that fits in a byte": {
			mechanism: catalog.MechanismLGAltInput,
			value:     0xD1,
			want:      []string{"set", "input-alt", "209"},
		},
		// m1ddc's prepareDDCWrite takes a 16-bit value and splits it into the
		// SH/SL pair itself, so 0x1D1 is handed over as 465 rather than clipped.
		"a value that does not": {
			mechanism: catalog.MechanismLGAltInput,
			value:     0x1D1,
			want:      []string{"set", "input-alt", "465"},
		},
		"a mechanism this backend does not implement": {
			mechanism: catalog.Mechanism("invented"),
			value:     0xD1,
			want:      nil,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := arguments(testCase.mechanism, testCase.value)
			if !slices.Equal(got, testCase.want) {
				t.Errorf("arguments() = %q, want %q", got, testCase.want)
			}
		})
	}
}
