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

// Package backend describes what a monitor-control backend is, without being
// one. It imports no backend implementation, so the decision layers can depend
// on the vocabulary without depending on ddcutil or m1ddc.
package backend

import "github.com/leinardi/monmux/internal/edid"

// Display is one attached display as a backend sees it.
type Display struct {
	// Identity is the EDID identity monmux matches against the catalog.
	Identity edid.Identity
	// Handle is how the backend addresses this display: the connector name on
	// Linux (card1-DP-1), the display's UUID on macOS.
	Handle string
	// HandlePrivate is true when the handle itself is private data, as the macOS
	// UUID is. Such a handle is masked in output unless --show-serial was given.
	HandlePrivate bool
	// Label is the public human-readable name: the connector name on Linux,
	// "display 1" on macOS. It is always safe to print.
	Label string
	// Writable says whether this display can be written to at all. An unwritable
	// display is still reported by info, but policy can never select it.
	Writable bool
	// Status is why the display is in the state it is in: "ok",
	// "no-ddc-channel", "edid-unreadable", "no-uuid", and so on.
	Status string
}

// Status values a backend reports for a display.
const (
	// StatusOK means the display was identified and can be written to.
	StatusOK = "ok"
	// StatusNoDDCChannel means the display exposes no DDC channel to talk over.
	StatusNoDDCChannel = "no-ddc-channel"
	// StatusEDIDUnreadable means the EDID was missing or would not parse.
	StatusEDIDUnreadable = "edid-unreadable"
	// StatusNoUUID means macOS gave the display no UUID, the only selector there.
	StatusNoUUID = "no-uuid"
)
