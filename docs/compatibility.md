# Compatibility

This page is the human-readable copy of the supported-monitor catalog in `internal/catalog/models.go`. The catalog is what the
binary uses; this page is checked against it by `TestCompatibilityDocumentMatchesTheCatalog`, so the two cannot drift apart
without failing the build.

One model is verified so far. A model is listed here whether or not monmux will write to it, because "we know about this monitor
and deliberately will not touch it" is information a reader needs.

## What the columns mean

- **Identities** — the EDID fingerprints, as `manufacturer/product code`, that match this model. A model can have several: the
  tested LG reports a different product code depending on which input it is currently displaying. A model with no identity
  recorded can never match a display, and so can never be written to.
- **Input** — the symbolic input, as typed on the command line.
- **Value** — the byte written for that input.
- **Mechanism** — how it is written. `lg-alt-input` is the LG side channel: source address `0x50`, VCP `0xF4`, no verification.
  It is a per-model property and never a fallback.
- **Enabled** — whether monmux will actually perform this switch. A row with `no` is recorded for reference only.
- **Evidence** — why anyone believes that value does that thing on that model. This is the column that decides whether a row is
  enabled.

## Catalog

| Model         | Identities             | Input   | Value  | Mechanism      | Enabled | Evidence                                                                                                                                               |
| ------------- | ---------------------- | ------- | ------ | -------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| LG 38WR85QC-W | GSM/0x77D3, GSM/0x77D4 | `dp`    | `0xD0` | `lg-alt-input` | yes     | Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc): switched from USB-C to DisplayPort                                             |
| LG 38WR85QC-W | GSM/0x77D3, GSM/0x77D4 | `usb-c` | `0xD1` | `lg-alt-input` | yes     | Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc): switched from DisplayPort to USB-C                                             |
| LG 38BR85QC   | none                   | `dp`    | `0xD0` | `lg-alt-input` | no      | Reported by a tester on the ddcutil LG wiki page; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC   | none                   | `usb-c` | `0xD1` | `lg-alt-input` | no      | Reported by a tester on the ddcutil LG wiki page; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC   | none                   | `hdmi1` | `0x90` | `lg-alt-input` | no      | Reported by a tester on the ddcutil LG wiki page; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC   | none                   | `hdmi2` | `0x91` | `lg-alt-input` | no      | Reported by a tester on the ddcutil LG wiki page; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |

## Notes on the entries

### LG 38WR85QC-W

Two identities, because the same physical unit reports `GSM/0x77D3` while displaying DisplayPort and `GSM/0x77D4` while
displaying USB-C. Both were observed directly; the model string is `LG ULTRAWIDE`, which is not used for matching.

`hdmi1` and `hdmi2` are deliberately absent rather than disabled. The values were never tested on this unit, and another LG
model using them is not evidence for this one. Asking for them is refused with `input-not-enabled`.

Sources:

- Direct observation on the tested unit: EDID `GSM/0x77D3` (DisplayPort) and `GSM/0x77D4` (USB-C), model string `LG ULTRAWIDE`.
- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 38BR85QC

Recorded, disabled, and unable to match anything: no EDID fingerprint for it has been collected. The values come from the
ddcutil wiki and nobody involved in this project has a unit to test them on. They are written down because they are useful to a
contributor who does — see the contributor guide for what turns a row like this into an enabled one.

Source: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
