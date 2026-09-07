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

package catalog

// This file is the catalog itself: the complete set of bytes monmux is willing
// to send to a monitor, with the evidence for each one next to it.
//
// Rules for editing it, enforced by the tests in catalog_test.go and spelled out
// in docs/adding-a-monitor.md:
//
//   - A write-enabled model needs at least one EDID identity. An entry with no
//     identities can never match, and so can never be written to.
//   - Every input carries its own evidence string. "Another model in the same
//     family uses this value" is not evidence.
//   - A value that has not been observed on the model itself, or confirmed for
//     that exact model by a named source, does not go in.

const (
	// vendorLG is the vendor name as it appears in output.
	vendorLG = "LG"
	// pnpLG is LG Electronics' three-letter PNP manufacturer ID in EDID.
	pnpLG = "GSM"

	// ddcutilLGWiki documents the LG side channel and the values a tester
	// confirmed for the 38BR85QC.
	ddcutilLGWiki = "https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors"

	// directTest38WR85QCW is the evidence for the two inputs that were switched,
	// in both directions, on the physical unit this project was built against.
	directTest38WR85QCW = "Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc)"
)

// models is the supported-monitor catalog.
var models = []Model{
	{
		Name:   "38WR85QC-W",
		Vendor: vendorLG,
		// The same physical unit reports 0x77D3 over DisplayPort and 0x77D4
		// over USB-C, which is why a model carries several identities.
		Identities: []Identity{
			{Manufacturer: pnpLG, ProductCode: 0x77D3},
			{Manufacturer: pnpLG, ProductCode: 0x77D4},
		},
		WriteEnabled: true,
		Inputs: map[Input]inputOp{
			InputDP: {
				mechanism: MechanismLGAltInput,
				value:     0xD0,
				evidence:  directTest38WR85QCW + ": switched from USB-C to DisplayPort",
			},
			InputUSBC: {
				mechanism: MechanismLGAltInput,
				value:     0xD1,
				evidence:  directTest38WR85QCW + ": switched from DisplayPort to USB-C",
			},
			// HDMI is deliberately absent: the values were never tested on this
			// unit, and another LG model using them is not evidence for this one.
		},
		Sources: []string{
			"Direct observation on the tested unit: EDID GSM/0x77D3 (DisplayPort) and GSM/0x77D4 (USB-C), model string LG ULTRAWIDE",
			ddcutilLGWiki,
		},
	},
	{
		Name:   "38BR85QC",
		Vendor: vendorLG,
		// No identities: no EDID fingerprint for this model has been collected,
		// so it can never match and monmux can never write to it. The mapping is
		// recorded because it is useful to a contributor who owns the monitor.
		Identities:   nil,
		WriteEnabled: false,
		Inputs: map[Input]inputOp{
			InputDP: {
				mechanism: MechanismLGAltInput,
				value:     0xD0,
				evidence:  "Reported by a tester on the ddcutil LG wiki page; not verified here: " + ddcutilLGWiki,
			},
			InputUSBC: {
				mechanism: MechanismLGAltInput,
				value:     0xD1,
				evidence:  "Reported by a tester on the ddcutil LG wiki page; not verified here: " + ddcutilLGWiki,
			},
			InputHDMI1: {
				mechanism: MechanismLGAltInput,
				value:     0x90,
				evidence:  "Reported by a tester on the ddcutil LG wiki page; not verified here: " + ddcutilLGWiki,
			},
			InputHDMI2: {
				mechanism: MechanismLGAltInput,
				value:     0x91,
				evidence:  "Reported by a tester on the ddcutil LG wiki page; not verified here: " + ddcutilLGWiki,
			},
		},
		Sources: []string{ddcutilLGWiki},
	},
}
