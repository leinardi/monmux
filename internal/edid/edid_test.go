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

package edid_test

import (
	"errors"
	"os"
	"testing"

	"github.com/leinardi/monmux/internal/edid"
)

// fixture is a real LG 38WR85QC-W EDID with both serials replaced by the
// synthetic values documented in the plan, and the checksum recomputed.
const fixture = "testdata/lg-38wr85qc-w.edid"

const (
	wantManufacturer = "GSM"
	wantProductCode  = 0x77D3
	wantModelName    = "LG ULTRAWIDE"
	wantSerialNumber = 0x01020304
	wantSerialString = "TESTSERIAL01"
)

func readFixture(t *testing.T) []byte {
	t.Helper()

	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("reading %s: %v", fixture, err)
	}

	return raw
}

func TestParseFixture(t *testing.T) {
	t.Parallel()

	identity, err := edid.Parse(readFixture(t))
	if err != nil {
		t.Fatalf("Parse() returned an unexpected error: %v", err)
	}

	if identity.Manufacturer != wantManufacturer {
		t.Errorf("Manufacturer = %q, want %q", identity.Manufacturer, wantManufacturer)
	}

	if identity.ProductCode != wantProductCode {
		t.Errorf("ProductCode = %#04X, want %#04X", identity.ProductCode, wantProductCode)
	}

	// Exact, not "contains": descriptor 0xFC pads with spaces and terminates
	// with 0x0A, and a catalog entry that pins a model name compares this string
	// against what the macOS backend produces for the same unit. A stray space
	// would make the same monitor match on one operating system and not on the
	// other.
	if identity.ModelName != wantModelName {
		t.Errorf("ModelName = %q, want %q", identity.ModelName, wantModelName)
	}

	if identity.SerialNumber != wantSerialNumber {
		t.Errorf("SerialNumber = %#08X, want %#08X", identity.SerialNumber, wantSerialNumber)
	}

	if identity.SerialString != wantSerialString {
		t.Errorf("SerialString = %q, want %q", identity.SerialString, wantSerialString)
	}
}

func TestParseIgnoresExtensionBlocks(t *testing.T) {
	t.Parallel()

	raw := readFixture(t)
	// Anything past block 0 must not affect the result, garbage included.
	extended := append(append([]byte{}, raw...), make([]byte, 128)...)

	identity, err := edid.Parse(extended)
	if err != nil {
		t.Fatalf("Parse() returned an unexpected error: %v", err)
	}

	if identity.ProductCode != wantProductCode {
		t.Errorf("ProductCode = %#04X, want %#04X", identity.ProductCode, wantProductCode)
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	t.Parallel()

	valid := readFixture(t)

	truncated := valid[:64]

	badHeader := append([]byte{}, valid...)
	badHeader[1] = 0x00

	badChecksum := append([]byte{}, valid...)
	badChecksum[127]++

	badManufacturer := append([]byte{}, valid...)
	badManufacturer[8] = 0x00
	badManufacturer[9] = 0x00
	badManufacturer[127] = checksumFor(badManufacturer)

	tests := map[string]struct {
		raw  []byte
		want error
	}{
		"truncated":        {raw: truncated, want: edid.ErrTooShort},
		"bad header":       {raw: badHeader, want: edid.ErrHeader},
		"bad checksum":     {raw: badChecksum, want: edid.ErrChecksum},
		"bad manufacturer": {raw: badManufacturer, want: edid.ErrManufacturer},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := edid.Parse(test.raw)
			if !errors.Is(err, test.want) {
				t.Errorf("Parse() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestRedactedZeroesBothSerials(t *testing.T) {
	t.Parallel()

	identity, err := edid.Parse(readFixture(t))
	if err != nil {
		t.Fatalf("Parse() returned an unexpected error: %v", err)
	}

	redacted := identity.Redacted()

	if redacted.SerialNumber != 0 {
		t.Errorf("SerialNumber = %#08X, want 0", redacted.SerialNumber)
	}

	if redacted.SerialString != "" {
		t.Errorf("SerialString = %q, want empty", redacted.SerialString)
	}

	if redacted.Manufacturer != identity.Manufacturer ||
		redacted.ProductCode != identity.ProductCode {
		t.Errorf("Redacted() changed the matching fields: %+v", redacted)
	}

	if redacted.ModelName != identity.ModelName {
		t.Errorf("ModelName = %q, want %q", redacted.ModelName, identity.ModelName)
	}

	if !identity.HasSerial() {
		t.Error("HasSerial() = false on the fixture, want true")
	}

	if redacted.HasSerial() {
		t.Error("HasSerial() = true on a redacted identity, want false")
	}
}

// checksumFor returns the byte that makes block 0 sum to zero modulo 256.
func checksumFor(block []byte) byte {
	var sum uint8
	for _, b := range block[:127] {
		sum += b
	}

	return -sum
}
