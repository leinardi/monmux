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

// Package edid parses block 0 of an EDID into the small identity monmux matches
// against its catalog. It is deliberately cross-platform and does no I/O: a
// backend reads the bytes and hands them over.
//
// The parsed [Identity] never carries the raw EDID bytes. Raw bytes stay inside
// the backend that read them, both because they are the backend's business and
// because they contain the monitor's serial number.
package edid

import (
	"errors"
	"fmt"
	"strings"
)

// Errors returned by [Parse]. They are matched with errors.Is by callers that
// want to tell a malformed EDID from an absent one.
var (
	// ErrTooShort reports an EDID buffer smaller than one 128-byte block.
	ErrTooShort = errors.New("edid: shorter than one 128-byte block")
	// ErrHeader reports a missing or corrupt EDID block-0 header.
	ErrHeader = errors.New("edid: invalid block 0 header")
	// ErrChecksum reports a block-0 checksum that does not sum to zero.
	ErrChecksum = errors.New("edid: invalid block 0 checksum")
	// ErrManufacturer reports a PNP manufacturer ID that is not three letters.
	ErrManufacturer = errors.New("edid: invalid PNP manufacturer ID")
)

const (
	// blockSize is the length of EDID block 0, the only block monmux reads.
	blockSize = 128

	// headerLen is the length of the fixed 00 FF 00 header.
	headerLen = 8

	// manufacturerOffset is the first byte of the big-endian PNP manufacturer ID.
	manufacturerOffset = 8
	// productCodeOffset is the first byte of the little-endian product code.
	productCodeOffset = 10
	// serialNumberOffset is the first byte of the little-endian serial number.
	serialNumberOffset = 12

	// descriptorLen is the length of one of the four block-0 descriptors.
	descriptorLen = 18
	// descriptorTextOffset is where a text descriptor's 13 payload bytes start.
	descriptorTextOffset = 5
	// descriptorTagOffset is where a descriptor's type tag sits.
	descriptorTagOffset = 3

	// tagModelName is the descriptor tag holding the monitor's model name.
	tagModelName = 0xFC
	// tagSerialString is the descriptor tag holding the alphanumeric serial.
	tagSerialString = 0xFF

	// descriptorTerminator ends a text descriptor shorter than 13 bytes.
	descriptorTerminator = '\n'

	// letterBits is the width of one packed letter in the manufacturer ID.
	letterBits = 5
	// letterMask selects one packed letter from the manufacturer ID.
	letterMask = 0x1F
	// manufacturerLetters is the number of letters in a PNP manufacturer ID.
	manufacturerLetters = 3
)

// header is the fixed byte sequence every EDID block 0 starts with.
var header = [headerLen]byte{0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00}

// descriptorOffsets are the starting offsets of the four block-0 descriptors.
var descriptorOffsets = [...]int{54, 72, 90, 108}

// Identity is what monmux matches against its catalog, plus the two fields that
// identify a physical unit rather than a model. It carries no raw EDID bytes.
type Identity struct {
	// Manufacturer is the three-letter PNP ID, e.g. "GSM" for LG.
	Manufacturer string `json:"manufacturer"`
	// ProductCode is the vendor's model code, e.g. 0x77D3.
	ProductCode uint16 `json:"productCode"`
	// SerialNumber is the numeric serial. It is never used for pinning.
	SerialNumber uint32 `json:"serialNumber"`
	// SerialString is the alphanumeric serial from descriptor 0xFF, if present.
	// This is the only field --serial pins against.
	SerialString string `json:"serialString"`
	// ModelName is the human-readable name from descriptor 0xFC, if present.
	ModelName string `json:"modelName"`
}

// Redacted returns a copy with both serial fields zeroed. Every code path that
// prints an identity goes through this unless the user passed --show-serial.
func (i Identity) Redacted() Identity {
	i.SerialNumber = 0
	i.SerialString = ""

	return i
}

// HasSerial reports whether the display exposed either serial field. It stays
// true on a redacted copy's original, so callers can say "redacted" rather than
// "absent" without holding on to the unredacted value.
func (i Identity) HasSerial() bool {
	return i.SerialNumber != 0 || i.SerialString != ""
}

// Parse reads block 0 of an EDID and returns the identity it describes. Any
// trailing extension blocks are ignored. It validates the header and the block-0
// checksum, so a truncated or garbled read is reported rather than matched.
func Parse(raw []byte) (Identity, error) {
	if len(raw) < blockSize {
		return Identity{}, fmt.Errorf("%w: got %d bytes", ErrTooShort, len(raw))
	}

	block := raw[:blockSize]

	if [headerLen]byte(block[:headerLen]) != header {
		return Identity{}, ErrHeader
	}

	err := verifyChecksum(block)
	if err != nil {
		return Identity{}, err
	}

	manufacturer, err := parseManufacturer(block)
	if err != nil {
		return Identity{}, err
	}

	identity := Identity{
		Manufacturer: manufacturer,
		ProductCode:  uint16(block[productCodeOffset]) | uint16(block[productCodeOffset+1])<<8,
		SerialNumber: parseSerialNumber(block),
	}

	identity.ModelName = descriptorText(block, tagModelName)
	identity.SerialString = descriptorText(block, tagSerialString)

	return identity, nil
}

// verifyChecksum reports whether the 128 bytes of block 0 sum to zero modulo 256.
func verifyChecksum(block []byte) error {
	var sum uint8
	for _, b := range block {
		sum += b
	}

	if sum != 0 {
		return fmt.Errorf("%w: bytes sum to 0x%02X, want 0x00", ErrChecksum, sum)
	}

	return nil
}

// parseManufacturer decodes the three five-bit letters packed big-endian into
// bytes 8 and 9. "GSM" is LG's PNP ID; the letters are 1-26 for A-Z.
func parseManufacturer(block []byte) (string, error) {
	packed := uint16(block[manufacturerOffset])<<8 | uint16(block[manufacturerOffset+1])

	letters := make([]byte, 0, manufacturerLetters)

	for position := manufacturerLetters - 1; position >= 0; position-- {
		value := (packed >> (letterBits * position)) & letterMask
		if value < 1 || value > 'Z'-'A'+1 {
			return "", fmt.Errorf("%w: 0x%04X", ErrManufacturer, packed)
		}

		letters = append(letters, byte('A'+value-1))
	}

	return string(letters), nil
}

// parseSerialNumber decodes the little-endian numeric serial at bytes 12-15.
func parseSerialNumber(block []byte) uint32 {
	return uint32(block[serialNumberOffset]) |
		uint32(block[serialNumberOffset+1])<<8 |
		uint32(block[serialNumberOffset+2])<<16 |
		uint32(block[serialNumberOffset+3])<<24
}

// descriptorText returns the text of the first descriptor carrying the given tag,
// or "" when block 0 has no such descriptor.
func descriptorText(block []byte, tag byte) string {
	for _, offset := range descriptorOffsets {
		descriptor := block[offset : offset+descriptorLen]
		if !isTextDescriptor(descriptor) || descriptor[descriptorTagOffset] != tag {
			continue
		}

		text := descriptor[descriptorTextOffset:]

		end := len(text)
		for index, b := range text {
			if b == descriptorTerminator {
				end = index

				break
			}
		}

		return strings.TrimRight(string(text[:end]), " ")
	}

	return ""
}

// isTextDescriptor reports whether a descriptor is a display descriptor rather
// than a detailed timing block: bytes 0-2 and 4 are zero, and byte 3 is the tag.
func isTextDescriptor(descriptor []byte) bool {
	return descriptor[0] == 0 &&
		descriptor[1] == 0 &&
		descriptor[2] == 0 &&
		descriptor[4] == 0
}
