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
	"slices"
	"testing"

	"github.com/leinardi/monmux/internal/catalog"
)

// The end-to-end golden test in plan_test.go pins what the tested model's
// operation produces. This one pins the argv for a value the catalog does not
// currently enable, which no exported constructor could build an Operation for -
// rendering argv is not the same power as making a backend run it.
func TestArgumentsAreTheSetvcpInvocation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		mechanism catalog.Mechanism
		value     uint16
		want      []string
	}{
		"a value that fits in a byte": {
			mechanism: catalog.MechanismLGAltInput,
			value:     0xD1,
			want: []string{
				"setvcp", "0xF4", "0xD1", "--i2c-source-addr=0x50", "--noverify",
			},
		},
		// A SetVCP carries an SH/SL pair and ddcutil takes 0..65535, so a value
		// wider than a byte is written out in full rather than truncated.
		"a value that does not": {
			mechanism: catalog.MechanismLGAltInput,
			value:     0x1D1,
			want: []string{
				"setvcp", "0xF4", "0x1D1", "--i2c-source-addr=0x50", "--noverify",
			},
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
