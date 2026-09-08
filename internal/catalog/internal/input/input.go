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

// Package input is the grammar of a symbolic input name: a connector kind and an
// optional port number, e.g. dp, hdmi2, usb-c. It carries no bytes and knows
// nothing about monitors; it only decides which names are well-formed, what they
// are called in output, and how they sort.
//
// It is a leaf, imported by both internal/catalog and the catalog generator, so
// the two agree on the grammar without the generator importing the package whose
// source it writes. The set of kinds is closed and lives here: a new connector
// kind is a Go change, added with the first model that needs it, while a new port
// of a known kind is only a models.yaml edit.
package input

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
)

// Errors reported by [Parse] and [CheckNumbering]. The catalog wraps them under
// its own sentinel, so a caller can match either the general case or this one.
var (
	// ErrUnknownKind reports a name whose connector kind is not in the list.
	ErrUnknownKind = errors.New("input: unknown connector kind")
	// ErrMalformedNumber reports a port number that is not a positive decimal
	// integer without a leading zero.
	ErrMalformedNumber = errors.New("input: malformed port number")
	// ErrMixedNumbering reports a model that writes one kind both bare and
	// numbered, which would leave "hdmi" and "hdmi1" as two names for what may
	// well be one port.
	ErrMixedNumbering = errors.New("input: a kind is written both bare and numbered")
)

// Kind is a connector kind. The set is closed: these are the kinds monmux knows
// how to name, not the ones it can write to, which is a per-model question the
// catalog answers.
type Kind string

const (
	// KindDP is DisplayPort.
	KindDP Kind = "dp"
	// KindHDMI is HDMI.
	KindHDMI Kind = "hdmi"
	// KindUSBC is USB-C (DisplayPort alternate mode).
	KindUSBC Kind = "usb-c"
	// KindDVI is DVI.
	KindDVI Kind = "dvi"
	// KindVGA is VGA.
	KindVGA Kind = "vga"
	// KindThunderbolt is Thunderbolt.
	KindThunderbolt Kind = "thunderbolt"
)

// kinds is every kind, in the order names sort and are listed.
var kinds = []Kind{KindDP, KindHDMI, KindUSBC, KindDVI, KindVGA, KindThunderbolt}

// labels are the human-readable names used in success and refusal messages. A
// port number is appended to these, so "HDMI" becomes "HDMI 2".
var labels = map[Kind]string{
	KindDP:          "DisplayPort",
	KindHDMI:        "HDMI",
	KindUSBC:        "USB-C",
	KindDVI:         "DVI",
	KindVGA:         "VGA",
	KindThunderbolt: "Thunderbolt",
}

// Kinds returns every connector kind, in listing order.
func Kinds() []Kind {
	return slices.Clone(kinds)
}

// Label returns the human-readable name for messages, e.g. "DisplayPort".
func (k Kind) Label() string {
	label, ok := labels[k]
	if !ok {
		return string(k)
	}

	return label
}

// String returns the kind as it is typed on the command line.
func (k Kind) String() string {
	return string(k)
}

// index is the kind's position in the listing order, or len(kinds) for a kind
// that is not in the list at all. Only [Compare] uses it.
func (k Kind) index() int {
	at := slices.Index(kinds, k)
	if at < 0 {
		return len(kinds)
	}

	return at
}

// Name is a parsed input name. Number is 0 for a bare kind, which is how a model
// with a single port of that kind writes it.
type Name struct {
	Kind   Kind
	Number int
}

// Parse turns a command-line argument, or a catalog key, into a [Name]. The
// grammar is exact and case-sensitive: a known kind, optionally followed by a
// decimal port number with no leading zero and no separator.
func Parse(name string) (Name, error) {
	kind, digits := cut(name)

	if !slices.Contains(kinds, Kind(kind)) {
		return Name{}, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}

	if digits == "" {
		return Name{Kind: Kind(kind)}, nil
	}

	if digits[0] == '0' {
		return Name{}, fmt.Errorf("%w: %q has a leading zero", ErrMalformedNumber, digits)
	}

	number, err := strconv.Atoi(digits)
	if err != nil {
		return Name{}, fmt.Errorf("%w: %q: %w", ErrMalformedNumber, digits, err)
	}

	return Name{Kind: Kind(kind), Number: number}, nil
}

// cut splits a name at its first ASCII digit. Everything before is the candidate
// kind and everything after is the candidate number, so anything that is neither
// - a separator, a space, a second run of letters - stays in one half and is
// rejected by the check that half gets.
func cut(name string) (kind, digits string) {
	for at := range len(name) {
		if name[at] >= '0' && name[at] <= '9' {
			return name[:at], name[at:]
		}
	}

	return name, ""
}

// String returns the name as it is typed on the command line.
func (n Name) String() string {
	if n.Number == 0 {
		return n.Kind.String()
	}

	return n.Kind.String() + strconv.Itoa(n.Number)
}

// Label returns the human-readable name for messages, e.g. "HDMI 2".
func (n Name) Label() string {
	if n.Number == 0 {
		return n.Kind.Label()
	}

	return n.Kind.Label() + " " + strconv.Itoa(n.Number)
}

// Compare orders two names by kind and then by port number, so a bare kind comes
// before its numbered ports and hdmi2 comes before hdmi10. It is the order every
// list of inputs is printed in.
func Compare(first, second Name) int {
	byKind := first.Kind.index() - second.Kind.index()
	if byKind != 0 {
		return byKind
	}

	return first.Number - second.Number
}

// CheckNumbering reports a kind that appears both bare and numbered in one
// model. The bare form means "the single port of this kind", so writing it next
// to a numbered one leaves it undecided which port the bare name addresses.
func CheckNumbering(names []Name) error {
	var (
		bare     = map[Kind]bool{}
		numbered = map[Kind]bool{}
		problems []error
	)

	for _, name := range names {
		if name.Number == 0 {
			bare[name.Kind] = true
		} else {
			numbered[name.Kind] = true
		}
	}

	for _, kind := range kinds {
		if bare[kind] && numbered[kind] {
			problems = append(problems, fmt.Errorf("%w: %q", ErrMixedNumbering, kind))
		}
	}

	return errors.Join(problems...)
}
