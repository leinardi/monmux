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

// Package m1ddc is the macOS backend. It wraps the m1ddc binary and never speaks
// DDC itself.
//
// The backend proper is built only for macOS, because the tool it drives exists
// only there. Parsing that tool's output is not a macOS problem, though, and it
// is where a backend is most likely to be quietly wrong - so the parser and
// every decision taken from it live in files with no build tag, and are unit
// tested on the Linux development host as well as on macOS.
package m1ddc

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/leinardi/monmux/internal/backend"
	"github.com/leinardi/monmux/internal/edid"
)

// Name is the backend's name, and the binary it wraps.
const Name = "m1ddc"

// The labels below are the ones m1ddc prints for `display list detailed`. They
// were confirmed on 2026-09-07 against waydabber/m1ddc: first the README, for
// the command and the selector syntax, and then the source it is generated from
// - sources/m1ddc.m (printDisplayInfos, and the no-display path) and
// sources/ioregistry.m (selectDisplay, getDisplayIdentifier) - because the
// README documents the commands rather than the exact output. What is pinned
// here is what the code prints:
//
//	[1] LG ULTRAWIDE (00000000-0000-4000-8000-000000000001)
//	 - Product name:  LG ULTRAWIDE
//	 - Manufacturer:  GSM
//	 - AN Serial:     TESTSERIAL01
//	 - Vendor:        7789 (0x1e6d)
//	 - Model:         30676 (0x77d4)
//	 - Serial:        16909060 (0x01020304)
//	 - Display ID:    1
//	 - System UUID:   00000000-0000-4000-8000-000000000001
//	 - EDID UUID:     00000000-0000-4000-8000-000000000001
//	 - IO Location:   IOService:/AppleARMPE/arm-io/dispext0
//	 - Adapter:       4294967295
//
// Vendor, Model and Serial come from CGDisplayVendorNumber, CGDisplayModelNumber
// and CGDisplaySerialNumber, which are the same three EDID fields the Linux
// backend reads out of sysfs. That is what lets one catalog serve both.
const (
	fieldProductName  = "Product name"
	fieldManufacturer = "Manufacturer"
	fieldSerialString = "AN Serial"
	fieldVendor       = "Vendor"
	fieldModel        = "Model"
	fieldSerialNumber = "Serial"
	fieldUUID         = "System UUID"
)

// noDisplayMessage is what m1ddc writes, with a non-zero exit status, when the
// Mac has no external display attached. It is pinned from sources/m1ddc.m, where
// the string is "No external display found, aborting" followed by EXIT_FAILURE;
// only the stable part of it is matched.
//
// It matters because that combination is not a malfunction: it is the honest
// answer to "what is attached", so preflight accepts it and enumeration reports
// no displays, rather than the backend declaring itself broken.
const noDisplayMessage = "No external display found"

// absentField is what m1ddc prints, through %@, for a value the IORegistry did
// not supply.
const absentField = "(null)"

// ErrUnparsable reports output that does not look like m1ddc's display list.
var ErrUnparsable = errors.New("m1ddc: unrecognized display list output")

// headerPattern matches the "[1] Product name (uuid)" line that opens a
// display's block and carries the number the user sees.
var headerPattern = regexp.MustCompile(`^\[(\d+)]\s+(.*)\s+\(([^)]*)\)\s*$`)

// fieldPattern matches one " - Label:  value" detail line.
var fieldPattern = regexp.MustCompile(`^\s*-\s+([^:]+):\s*(.*?)\s*$`)

// decimalPattern matches the decimal half of a field m1ddc prints twice, as
// "30676 (0x77d4)".
var decimalPattern = regexp.MustCompile(`^(\d+)`)

// uuidPattern is the shape a system UUID has.
//
// It is enforced rather than assumed, because this string becomes m1ddc's
// display selector and m1ddc accepts a list index in the same position. A value
// that is not UUID-shaped is therefore not used at all: addressing a monitor by
// its position in a list that changes when something is plugged in is the
// failure the UUID-only rule exists to prevent.
var uuidPattern = regexp.MustCompile(
	`^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$`,
)

// pnpPattern matches a three-letter PNP manufacturer identifier.
var pnpPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// record is one display as m1ddc described it, together with the [backend.Display]
// the rest of monmux sees. The record is what Execute re-verifies against.
type record struct {
	// display is what Enumerate reports for this entry.
	display backend.Display
	// number is m1ddc's list position, which is what the public label says.
	number int
	// uuid is the display's system UUID, and the only selector monmux uses. It
	// is empty for a display macOS gave no UUID, which is why such a display
	// cannot be written to.
	uuid string
}

// block is the raw text of one display: its header and its detail fields.
type block struct {
	number int
	name   string
	uuid   string
	fields map[string]string
}

// parseDisplayList turns `m1ddc display list detailed` output into records.
//
// Output with no display block at all is an error rather than an empty list: the
// caller has already separated out m1ddc's "nothing is attached" answer, so
// reaching here with nothing recognizable means the output was not what this
// parser was written against, and guessing would be worse than refusing.
func parseDisplayList(output string) ([]record, error) {
	blocks := split(output)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("%w: no display block found", ErrUnparsable)
	}

	records := make([]record, 0, len(blocks))

	for _, current := range blocks {
		records = append(records, current.record())
	}

	return records, nil
}

// split cuts the output into one block per display. A detail line before any
// header, or a header that does not parse, is ignored rather than guessed at.
func split(output string) []block {
	blocks := []block{}

	var current *block

	for line := range strings.SplitSeq(output, "\n") {
		header := headerPattern.FindStringSubmatch(line)
		if header != nil {
			number, err := strconv.Atoi(header[1])
			if err != nil {
				continue
			}

			blocks = append(blocks, block{
				number: number,
				name:   strings.TrimSpace(header[2]),
				uuid:   strings.TrimSpace(header[3]),
				fields: map[string]string{},
			})

			current = &blocks[len(blocks)-1]

			continue
		}

		if current == nil {
			continue
		}

		field := fieldPattern.FindStringSubmatch(line)
		if field != nil {
			current.fields[strings.TrimSpace(field[1])] = strings.TrimSpace(field[2])
		}
	}

	return blocks
}

// record turns one parsed block into a record.
//
// A display with no usable UUID is reported and never written to. The UUID is
// the only selector monmux gives m1ddc: the list position would also address a
// display, but it changes when a monitor is plugged in or wakes up, so using it
// would mean sending an input switch to whichever display happened to be second.
func (b block) record() record {
	uuid := b.uuid
	if detailed, ok := b.fields[fieldUUID]; ok && detailed != "" {
		uuid = detailed
	}

	// Anything that is not a UUID - m1ddc's "(null)", an empty field, or a
	// value from some future version of the tool - leaves the display with no
	// handle, and so unwritable.
	if !uuidPattern.MatchString(uuid) {
		uuid = ""
	}

	display := backend.Display{
		Identity:      b.identity(),
		Handle:        uuid,
		HandlePrivate: true,
		Label:         "display " + strconv.Itoa(b.number),
		Writable:      uuid != "",
		Status:        backend.StatusOK,
	}

	if !display.Writable {
		display.Status = backend.StatusNoUUID
	}

	return record{display: display, number: b.number, uuid: uuid}
}

// identity maps m1ddc's fields onto the identity the catalog is matched on.
func (b block) identity() edid.Identity {
	name := present(b.fields[fieldProductName])
	if name == "" {
		name = present(b.name)
	}

	return edid.Identity{
		Manufacturer: b.manufacturer(),
		//nolint:gosec // an EDID product code is 16 bits by definition
		ProductCode: uint16(decimal(b.fields[fieldModel])),
		//nolint:gosec // an EDID serial number is 32 bits by definition
		SerialNumber: uint32(decimal(b.fields[fieldSerialNumber])),
		SerialString: present(b.fields[fieldSerialString]),
		ModelName:    name,
	}
}

// manufacturer returns the three-letter PNP identifier the catalog matches on.
//
// m1ddc prints a Manufacturer string, but it is the IORegistry's ManufacturerID
// and is sometimes absent. Vendor is the same packed number EDID carries, so it
// is decoded whenever the string is not a plain three-letter code: a wrong value
// here would mean matching the catalog entry of a different vendor's monitor.
func (b block) manufacturer() string {
	reported := present(b.fields[fieldManufacturer])
	if pnpPattern.MatchString(reported) {
		return reported
	}

	//nolint:gosec // a packed PNP identifier is 16 bits by definition
	return decodeVendor(uint16(decimal(b.fields[fieldVendor])))
}

const (
	// letterBits is the width of one packed letter in a PNP identifier.
	letterBits = 5
	// letterMask selects one packed letter.
	letterMask = 0x1F
	// pnpLetters is how many letters a PNP identifier has.
	pnpLetters = 3
	// lastLetter is the highest value a packed letter may hold, 'Z'.
	lastLetter = 'Z' - 'A' + 1
)

// decodeVendor turns the packed EDID vendor number into its three letters, and
// returns nothing at all for a number that does not decode to three letters:
// half a manufacturer identifier is worse than none.
func decodeVendor(packed uint16) string {
	letters := make([]byte, 0, pnpLetters)

	for position := pnpLetters - 1; position >= 0; position-- {
		value := (packed >> (letterBits * position)) & letterMask
		if value < 1 || value > lastLetter {
			return ""
		}

		letters = append(letters, byte('A'+value-1))
	}

	return string(letters)
}

// decimal reads the decimal half of a field m1ddc prints as "30676 (0x77d4)".
func decimal(field string) uint64 {
	match := decimalPattern.FindStringSubmatch(strings.TrimSpace(field))
	if match == nil {
		return 0
	}

	value, err := strconv.ParseUint(match[1], 10, 64)
	if err != nil {
		return 0
	}

	return value
}

// present returns a field's value, or nothing when m1ddc printed its marker for
// a value the IORegistry did not supply. An absent field stays absent instead of
// becoming the literal string "(null)".
func present(value string) string {
	if value == absentField {
		return ""
	}

	return value
}

// noDisplaysReported reports whether output is m1ddc's "nothing is attached"
// answer rather than a failure.
func noDisplaysReported(output string) bool {
	return strings.Contains(output, noDisplayMessage)
}
