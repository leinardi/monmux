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
	"bytes"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
)

// This file reads `ddcutil detect --verbose`, which is how monmux finds the I2C
// bus of a connector the kernel gave no ddc link - every connector of the NVIDIA
// proprietary driver, for one. It carries no build tag, like the m1ddc parser,
// so it is tested wherever the tests run.
//
// The output was pinned from ddcutil 2.2.5 and 3.0.2, which print the same
// lines with a different indent. One block per bus that carries an EDID:
//
//	Display 1
//	   I2C bus:  /dev/i2c-9
//	      DRM_connector:                             card2-HDMI-A-2
//	      ...
//	   EDID hex dump:
//	        +0          +4          +8          +c            0   4   8   c
//	+0000   00 ff 00 4c 2d d0 73 04 03 02 01   ........L-.s....
//	...
//	+0070   ...
//
// A block opens with an unindented header: "Display N" for a bus ddcutil can
// talk DDC on, and "Invalid display" or another phrase for one it cannot.

// detectArgs is the read-only invocation that lists every bus with an EDID.
var detectArgs = []string{"detect", "--verbose"}

// The lines of `ddcutil detect --verbose` the parser reads.
var (
	// detectHeaderPattern is a block header: any unindented line. It is checked
	// after detectDumpPattern, because a hex dump line is unindented too.
	detectHeaderPattern = regexp.MustCompile(`^\S`)
	// detectValidPattern is the header of a bus ddcutil can talk DDC on.
	detectValidPattern = regexp.MustCompile(`^Display \d+\s*$`)
	// detectBusPattern names the block's bus.
	detectBusPattern = regexp.MustCompile(`^\s+I2C bus:\s+/dev/(i2c-\d+)\s*$`)
	// detectConnectorPattern names the DRM connector ddcutil associates with
	// the bus.
	detectConnectorPattern = regexp.MustCompile(`^\s+DRM_connector:\s+(\S+)\s*$`)
	// detectDumpPattern is one row of the EDID hex dump: an offset and sixteen
	// bytes, then their ASCII rendering, which is ignored.
	detectDumpPattern = regexp.MustCompile(`^\+([0-9a-f]{4})\s+((?:[0-9a-f]{2} ){15}[0-9a-f]{2})`)
)

const (
	// edidBlockSize is the length of the EDID block monmux reads and compares.
	edidBlockSize = 128
	// dumpRowSize is how many bytes one row of the hex dump carries.
	dumpRowSize = 16
)

// detectedBus is one block of `ddcutil detect --verbose`.
type detectedBus struct {
	// valid is true for a "Display N" block: a bus ddcutil can talk DDC on.
	valid bool
	// bus is the I2C bus, e.g. "i2c-9".
	bus string
	// connector is the DRM connector ddcutil names for the bus, if any.
	connector string
	// edid is EDID block 0, or nil when the dump was missing, out of order or
	// short: half an EDID compared with a whole one would prove nothing.
	edid []byte
	// broken records a dump that cannot be trusted.
	broken bool
}

// parseDetect reads `ddcutil detect --verbose` into one entry per block. Lines
// before the first header, and lines it does not know, are ignored.
func parseDetect(output string) []detectedBus {
	var blocks []detectedBus

	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimRight(line, "\r")

		if row := detectDumpPattern.FindStringSubmatch(line); row != nil {
			if len(blocks) > 0 {
				blocks[len(blocks)-1].addRow(row[1], row[2])
			}

			continue
		}

		// A row of the dump that does not read as one is a broken dump, not
		// the header of a new block.
		if strings.HasPrefix(line, "+") {
			if len(blocks) > 0 {
				blocks[len(blocks)-1].broken = true
			}

			continue
		}

		if detectHeaderPattern.MatchString(line) {
			blocks = append(blocks, detectedBus{valid: detectValidPattern.MatchString(line)})

			continue
		}

		if len(blocks) == 0 {
			continue
		}

		current := &blocks[len(blocks)-1]

		if match := detectBusPattern.FindStringSubmatch(line); match != nil {
			current.bus = match[1]
		}

		if match := detectConnectorPattern.FindStringSubmatch(line); match != nil {
			current.connector = match[1]
		}
	}

	for index := range blocks {
		if blocks[index].broken || len(blocks[index].edid) != edidBlockSize {
			blocks[index].edid = nil
		}
	}

	return blocks
}

// addRow appends one row of the hex dump. Rows have to arrive in order, from
// offset zero; anything else marks the dump broken. Rows past block 0 are not
// kept, because block 0 is what monmux compares.
func (d *detectedBus) addRow(offset, values string) {
	position, err := strconv.ParseUint(offset, 16, 16)
	if err != nil || (int(position) != len(d.edid) && int(position) < edidBlockSize) {
		d.broken = true

		return
	}

	if int(position) >= edidBlockSize {
		return
	}

	row, err := hex.DecodeString(strings.ReplaceAll(values, " ", ""))
	if err != nil || len(row) != dumpRowSize {
		d.broken = true

		return
	}

	d.edid = append(d.edid, row...)
}

// busByEDID returns the bus whose EDID block 0 is exactly the connector's.
//
// Every condition fails closed. The EDID has to appear on exactly one bus,
// because two buses carrying it - a mirrored or MST setup - leave no way to tell
// which one the connector is. That bus has to be one ddcutil can talk DDC on.
// And ddcutil has to name the same connector for it: ddcutil makes its own
// association, and the two agreeing is a second opinion rather than a guess.
func busByEDID(blocks []detectedBus, edid []byte, name string) (string, bool) {
	var (
		found *detectedBus
		count int
	)

	for index := range blocks {
		if blocks[index].edid != nil && bytes.Equal(blocks[index].edid, edid) {
			count++
			found = &blocks[index]
		}
	}

	if count != 1 || !found.valid || found.connector != name || found.bus == "" {
		return "", false
	}

	return found.bus, true
}
