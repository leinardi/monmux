# Compatibility

This page is the human-readable copy of the supported-monitor catalog in `internal/catalog/models.yaml`. The catalog is what the
binary uses. The two sections between the `BEGIN GENERATED` and `END GENERATED` markers — the table and the per-model notes —
are rendered from that file by `make go-generate`: edit the catalog, regenerate, and commit both. Everything outside the markers
is written by hand. Three tests check the result, so the page and the catalog cannot drift apart without failing the build:
`TestCompatibilityDocumentMatchesTheGenerator` compares this file against a fresh rendering, and
`TestCompatibilityDocumentMatchesTheCatalog` and `TestCompatibilityDocumentCarriesEveryNoteAndSource` read it back and compare
it against the compiled catalog, which is the copy the binary actually uses.

One model is verified so far. Everything else in the table is a report somebody else published: recorded because it is useful
to a contributor who owns that monitor, disabled, and without an EDID fingerprint, so it can never match a display and can
never be written to. A model is listed here whether or not monmux will write to it, because "we know about this monitor and
deliberately will not touch it" is information a reader needs.

monmux implements two mechanisms. `lg-alt-input` is the LG side channel, which is why most entries are LGs; `vcp-input-source`
is the standard Input Source feature, `VCP 0x60`, and has one recorded model, disabled. "Standard" is a name, not a promise:
which values a model accepts, and whether it accepts the write at all, are still per-model facts with per-input evidence.

## What the columns mean

- **Identities** — the EDID fingerprints, as `manufacturer/product code`, that match this model. A model can have several: the
  tested LG reports a different product code depending on which input it is currently displaying. A model with no identity
  recorded can never match a display, and so can never be written to. A fingerprint may also pin the EDID model name, written
  after the code as `GSM/0x7707 "LG HDR 4K"`; that is only used where two models share a reused product code, and no entry
  needs it today.
- **Input** — the symbolic input, as typed on the command line: a connector kind (`dp`, `hdmi`, `usb-c`, `dvi`, `vga`,
  `thunderbolt`) plus an optional port number, as in `hdmi2`. A kind is written bare when the model has one port of it and
  numbered when it has several, never both.
- **Value** — the value written for that input, up to 16 bits. A SetVCP carries an SH/SL pair; most recorded values fit in the low byte, and one does not.
- **Mechanism** — how it is written. `lg-alt-input` is the LG side channel: source address `0x50`, VCP `0xF4`, no verification.
  `vcp-input-source` is the standard Input Source feature: the ordinary source address, VCP `0x60`, no verification. A mechanism
  is a per-model property and never a fallback — a backend that does not implement a model's mechanism refuses rather than
  trying the other one.
- **Enabled** — whether monmux will actually perform this switch. A row with `no` is recorded for reference only.
- **Grade** — how strong the evidence is. `verified` is a direct test on the unit by this project and is the only grade that
  may be enabled; `documented` is the manufacturer's own documentation with no field report behind it; `reported` is somebody
  reporting that switching that named input with that value worked; `quoted` is a report that quotes the values and says they
  work, without saying which inputs were tried individually.
- **Evidence** — why anyone believes that value does that thing on that model, composed from the fields the catalog records so
  that every row of one grade reads the same way.

<!-- BEGIN GENERATED: edit internal/catalog/models.yaml, then run make go-generate -->

## Catalog

| Model | Identities | Input | Value | Mechanism | Enabled | Grade | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| LG 38WR85QC-W | GSM/0x77D3, GSM/0x77D4 | `dp` | `0xD0` | `lg-alt-input` | yes | verified | Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc): switched from USB-C to DisplayPort |
| LG 38WR85QC-W | GSM/0x77D3, GSM/0x77D4 | `usb-c` | `0xD1` | `lg-alt-input` | yes | verified | Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc): switched from DisplayPort to USB-C |
| LG 38BR85QC | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27BN88Q-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27BN88Q-B by bansheerubber (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1646752889> |
| LG 27BN88Q-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27BN88Q-B by bansheerubber (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1646752889> |
| LG 27UK500-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UK500-B by francis36012 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UK500-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UK500-B by francis36012 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UK500-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UK500-B by francis36012 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UL550-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UL550-W by titou10titou10 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UL550-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UL550-W by titou10titou10 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UL550-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UL550-W by titou10titou10 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-WY | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-WY | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-WY | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-WY | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-W by the tester who added it to the ddcutil LG wiki page (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the report switched from the USB-C input; the wiki row records the model as confirmed without values, which that tester quotes in the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011> |
| LG 27UN850-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-W by the tester who added it to the ddcutil LG wiki page (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the report switched from the USB-C input; the wiki row records the model as confirmed without values, which that tester quotes in the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011> |
| LG 27UN850-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-W by the tester who added it to the ddcutil LG wiki page (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the report switched from the USB-C input; the wiki row records the model as confirmed without values, which that tester quotes in the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011> |
| LG 27UP850-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP850-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP850-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP850-W | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP85NP-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UP85NP-W by edror12 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UP85NP-W | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 27UP85NP-W by edror12 (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27US500-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27US500-W by Fidelxyz (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27US500-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27US500-W by Fidelxyz (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27US500-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27US500-W by Fidelxyz (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29UM69G | none | `dp` | `0xC0` | `lg-alt-input` | no | reported | Reported working on the exact 29UM69G by jonpas (ddcutil): switching to DisplayPort with 0xC0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788> |
| LG 29UM69G | none | `hdmi` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 29UM69G by jonpas (ddcutil): switching to HDMI with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788> |
| LG 29UM69G | none | `usb-c` | `0xE0` | `lg-alt-input` | no | reported | Reported working on the exact 29UM69G by jonpas (ddcutil): switching to USB-C with 0xE0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788> |
| LG 29WN600 | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 29WN600 by iamSlightlyWind (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29WN600 | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 29WN600 by iamSlightlyWind (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29WN600 | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 29WN600 by iamSlightlyWind (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29U531A | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 29U531A by tinkererkzy (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29U531A | none | `hdmi` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 29U531A by tinkererkzy (ddcutil): switching to HDMI with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29U531A | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 29U531A by tinkererkzy (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32GP750-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GP750-B by chris-pcguy (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32GP750-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GP750-B by chris-pcguy (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32GP750-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GP750-B by chris-pcguy (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32GP850-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GP850-B by erenard (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145> |
| LG 32GP850-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GP850-B by erenard (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145> |
| LG 32GP850-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GP850-B by erenard (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145> |
| LG 32QN650-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32QN650-B by agspoon (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046> |
| LG 32QN650-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32QN650-B by agspoon (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046> |
| LG 32QN650-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32QN650-B by agspoon (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046> |
| LG 32UD99-W | none | `dp` | `0xE0` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to DisplayPort with 0xE0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; found by looping over candidate values; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636997772> |
| LG 32UD99-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636761366> |
| LG 32UD99-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636761366> |
| LG 32UD99-W | none | `usb-c` | `0xC0` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to USB-C with 0xC0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; found by looping over candidate values; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636997772> |
| LG 34WN750-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 34WN750-B by permezel (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 34WN750-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 34WN750-B by permezel (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 34WN750-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 34WN750-B by permezel (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 34WN80C-B | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; the wiki entry says only that the script works perfectly and does not quote the values, which come from the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 34WN80C-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): the report quotes 0x90 for HDMI 1 and says it works, without saying which inputs were tried individually; the wiki entry says only that the script works perfectly and does not quote the values, which come from the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 34WN80C-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): the report quotes 0x91 for HDMI 2 and says it works, without saying which inputs were tried individually; the wiki entry says only that the script works perfectly and does not quote the values, which come from the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 34WN80C-B | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the commenter states that "0xd1 is actually the usb-c input" on this monitor; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 40U990A-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 40U990A-W by titou10titou10 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40U990A-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 40U990A-W by titou10titou10 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40U990A-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 40U990A-W by titou10titou10 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 45GX950A-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 45GX950A-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 45GX950A-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 45GX950A-B | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27GP850-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27GP850-B by kaleb422 (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595> |
| LG 27GP850-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27GP850-B by kaleb422 (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595> |
| LG 27GP850-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27GP850-B by kaleb422 (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595> |
| LG 32UN880-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 32UN880-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 32UN880-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 32UN880-B | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 34WN780 | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN780 by piaverous (Windows, NVAPI): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317> |
| LG 34WN780 | none | `hdmi1` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN780 by piaverous (Windows, NVAPI): the report quotes 0x90 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317> |
| LG 34WN780 | none | `hdmi2` | `0x91` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN780 by piaverous (Windows, NVAPI): the report quotes 0x91 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317> |
| LG 34WN650-W | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN650-W by wigust (ddcutil 2.1.2): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750> |
| LG 34WN650-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN650-W by wigust (ddcutil 2.1.2): the report quotes 0x90 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750> |
| LG 34WN650-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN650-W by wigust (ddcutil 2.1.2): the report quotes 0x91 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750> |
| LG 28MQ780-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 28MQ780-B by amildahl (Windows, ADL): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195> |
| LG 28MQ780-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 28MQ780-B by amildahl (Windows, ADL): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195> |
| LG 28MQ780-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 28MQ780-B by amildahl (Windows, ADL): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195> |
| LG 32GR93U-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GR93U-B by gzougianos (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727> |
| LG 32GR93U-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GR93U-B by gzougianos (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727> |
| LG 32GR93U-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GR93U-B by gzougianos (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727> |
| LG 32GP83B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GP83B by Gilgame24 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333> |
| LG 32GP83B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GP83B by Gilgame24 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333> |
| LG 32GP83B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GP83B by Gilgame24 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333> |
| LG 34U650A-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay on macOS): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 34U650A-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay on macOS): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 34U650A-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay on macOS): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 34U650A-B | none | `usb-c` | `0x1D1` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay ddcAlt, macOS): switching to USB-C with 0x1D1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; found by looping over values; the listed 0xD2 got no response; 0x1D1 is 465 decimal; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 40WP95X | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95X by stepahin (Windows, NVAPI): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5> |
| LG 34GS95QE | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 34GS95QE by Vib0 (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302> |
| LG 34GS95QE | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 34GS95QE by Vib0 (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302> |
| LG 34GS95QE | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 34GS95QE by Vib0 (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302> |
| LG 32UP83AK-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32UP83AK-W by 5uck1ess (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/2#issuecomment-3221830027> |
| LG 32BL95U-W | none | `dp` | `0xD0` | `lg-alt-input` | no | documented | Documented by LG for the exact 32BL95U-W: DisplayPort is 0xD0 over the LG side channel (source address 0x50, VCP 0xF4); no field report; the service manual's Screen adjust command table, row 13, maps Input Select, command F4, to this value; not verified here: <https://research.encompass.com/ZEN/sm/32BL95UW.pdf> |
| LG 32BL95U-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | documented | Documented by LG for the exact 32BL95U-W: HDMI 1 is 0x90 over the LG side channel (source address 0x50, VCP 0xF4); no field report; the service manual's Screen adjust command table, row 13, maps Input Select, command F4, to this value; not verified here: <https://research.encompass.com/ZEN/sm/32BL95UW.pdf> |
| LG 32BL95U-W | none | `thunderbolt` | `0xD2` | `lg-alt-input` | no | documented | Documented by LG for the exact 32BL95U-W: Thunderbolt is 0xD2 over the LG side channel (source address 0x50, VCP 0xF4); no field report; the service manual's Screen adjust command table, row 13, maps Input Select, command F4, to this value; not verified here: <https://research.encompass.com/ZEN/sm/32BL95UW.pdf> |
| LG 32U990A | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 32U990A by pyang2045 (m1ddc input-alt on macOS): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/pyang2045/streamdeck-display-knob> |
| LG 32U990A | none | `hdmi` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 32U990A by pyang2045 (m1ddc input-alt on macOS): the report quotes 0x90 for HDMI and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/pyang2045/streamdeck-display-knob> |
| LG 32U990A | none | `thunderbolt` | `0xD2` | `lg-alt-input` | no | quoted | Weaker report on the exact 32U990A by pyang2045 (m1ddc input-alt on macOS): the report quotes 0xD2 for Thunderbolt and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/pyang2045/streamdeck-display-knob> |
| Samsung LC49G95T | none | `dp1` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact LC49G95T by DimpiM/monitor-switch (hardware-findings.md) (ddcutil 2.2.0 on a Raspberry Pi Zero 2 W, sent from the HDMI input): switching to DisplayPort 1 with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; reading VCP 0x60 back afterwards gives 0x03, which is not the value that selects the input; not verified here: <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md> |
| Samsung LC49G95T | none | `dp2` | `0x10` | `vcp-input-source` | no | reported | Reported working on the exact LC49G95T by DimpiM/monitor-switch (hardware-findings.md) (ddcutil 2.2.0 on a Raspberry Pi Zero 2 W, sent from the HDMI input): switching to DisplayPort 2 with 0x10 over the standard Input Source feature (VCP 0x60) succeeded; reading VCP 0x60 back afterwards gives 0x04, which is not the value that selects the input; not verified here: <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md> |
| Samsung LC49G95T | none | `hdmi` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact LC49G95T by DimpiM/monitor-switch (hardware-findings.md) (ddcutil 2.2.0 on a Raspberry Pi Zero 2 W, sent from the HDMI input): switching to HDMI with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; reading VCP 0x60 back afterwards gives 0x01, which is not the value that selects the input; not verified here: <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md> |

## Notes on the entries

### LG 38WR85QC-W

- Two identities, because the same physical unit reports `GSM/0x77D3` while displaying DisplayPort and `GSM/0x77D4` while displaying USB-C. Both were observed directly; the model string is `LG ULTRAWIDE`, which is not used for matching.
- `hdmi1` and `hdmi2` are deliberately absent rather than disabled. The values were never tested on this unit, and another LG model using them is not evidence for this one. Asking for them is refused with `input-not-enabled`. Several sources say the HDMI inputs of a 38WR85QC-W respond to `0x90` and `0x91`; because this model is write-enabled, recording them would enable a write, so they stay out until somebody runs them on this unit by hand, the way [testing.md](testing.md) describes.

Sources:

- Direct observation on the tested unit: EDID GSM/0x77D3 (DisplayPort) and GSM/0x77D4 (USB-C), model string LG ULTRAWIDE
- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 38BR85QC

- Recorded, disabled, and unable to match anything: no EDID fingerprint for it has been collected. The values come from the ddcutil wiki and nobody involved in this project has a unit to test them on. They are written down because they are useful to a contributor who does — see the contributor guide for what turns a row like this into an enabled one.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 27BN88Q-B

- Listed on the ddcutil LG wiki page by bansheerubber, who also wrote the values out in ddcutil issue #100: "successfully switching between HDMI 1 (0x90) and display port 1 (0xD0)". Only those two inputs are recorded; no report covers the others.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1646752889>

### LG 27UK500-B

- Test result contributed by francis36012 to the ddcutil LG wiki page. No EDID fingerprint was published with it.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 27UL550-W

- Test result contributed by titou10titou10 to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 27UN850-WY

- Test result contributed by ccrxf to the ddcutil LG wiki page. The name is kept exactly as the source writes it and is not merged with the separate 27UN850-W entry below, which comes from a different report.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 27UN850-W

- The wiki row says "confirmed" and gives no values. The values come from the same tester's comment on ddcutil issue #100, which reports switching from the USB-C input to the HDMI1, HDMI2 and DP1 inputs with x0090, x0091 and x00d0. USB-C is not recorded: the report switches away from it, never to it.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011>

### LG 27UP850-W

- Listed on the ddcutil LG wiki page by Prototyped, who published the full table in ddcutil issue #100, taken with ddcutil 2.0 and `--i2c-source-addr=x50`.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234>

### LG 27UP85NP-W

- Test result contributed by edror12 to the ddcutil LG wiki page. Only DisplayPort and USB-C were reported.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 27US500-W

- Test result contributed by Fidelxyz to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 29UM69G

- One of the two entries whose values are not the usual 0x90/0x91/0xD0/0xD1 set: DisplayPort is 0xC0 and USB-C is 0xE0. The evidence for all three is jonpas' report, on the wiki and in ddcutil issue #100. The same author's `i3/monitor.sh` is listed as corroboration and no more: it defines the USB-C value but only ever invokes the HDMI and DisplayPort ones, so it does not show 0xE0 being sent. That file is not pinned to a commit, because no revision of it has been quoted anywhere this entry could cite.
- A single HDMI port is reported, so the input is written bare. A 2016 unit badged 29UM69G-B (EDID `GSM/0x76FA`) did not switch at all, which is recorded here as a warning rather than as a separate entry: there is no schema slot for a negative result. The last source below is that report.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788>
- <https://github.com/jonpas/dotfiles/blob/master/i3/monitor.sh> - the same three values; the script defines the USB-C value but only ever invokes the HDMI and DisplayPort ones. Not pinned to a commit: no revision of it has been quoted anywhere this entry could cite
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2259286211>

### LG 29WN600

- Test result contributed by iamSlightlyWind to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 29U531A

- Test result contributed by tinkererkzy to the ddcutil LG wiki page. A single HDMI port is reported, so the input is written bare.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 32GP750-B

- Test result contributed by chris-pcguy to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 32GP850-B

- Reported working by erenard, on the wiki and in ddcutil issue #100, on a unit manufactured in week 12 of 2023. This one is contested: a later report on the NVapi-write-value-to-monitor tracker describes a 32GP850-B that only flickers and does not change input, and an LG firmware list quoted in ddcutil issue #100 names "GP850" as explicitly unsupported. The row stays disabled and without an identity, so nothing is written to either kind of unit; whoever enables it will have to say which firmware they tested.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/1#issuecomment-2110539219>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1444483801>

### LG 32QN650-B

- The wiki row (agspoon) gives no values; the two linked comments on ddcutil issue #100 quote x0090, x0091 and x00d0 for this model.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1634924423>

### LG 32UD99-W

- The second entry with unusual values: DisplayPort is 0xE0 and USB-C is 0xC0, the reverse pairing of the 29UM69G. The two comments carry different halves of the result and the rows are cited accordingly: the first one confirms only the HDMI values, and DisplayPort and USB-C appear in the follow-up, where they were found by looping over candidate values with Lunar on macOS. Worth extra care when someone verifies this model on hardware.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636761366>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636997772>

### LG 34WN750-B

- Test result contributed by permezel to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 34WN80C-B

- Two evidence strengths in one entry. USB-C is explicit: the linked comment names the input and the value. The other three come from the value table in the referenced gist, which the wiki contributor reports as working without quoting the numbers.
- Neither report went through ddcutil. The monitor was driven by the Python USB-HID script in that gist, which is why the evidence strings say so: the mechanism is the same LG side channel, but nothing here says the values survive a `ddcutil setvcp`. The gist is cited unpinned, because no revision of it is quoted in the report.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752>
- <https://gist.github.com/shinyquagsire23/f6b2adef253c6c3ab557a4852bf3abad>

### LG 40U990A-W

- Test result contributed by titou10titou10 to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 40WP95C-W

- Test result contributed by fblaese to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 45GX950A-B

- The best-corroborated of the new entries: the wiki result, a Stream Deck plugin whose `ddcutil.py` drives these values on Linux, a Windows NVAPI script that copies them from the wiki, and a Windows ADL application whose author says it is "only tested on 45GX950A-B".
- Two things do not agree. A MonitorControl issue lists read-back values of 15 for USB-C and 209 for DP2, but those are reads through `m1ddc get input-alt`, not writes, so they are not evidence about what a write does. And a BetterDisplay discussion reports a 45GX950A manufactured in week 9 of 2025 that would not leave HDMI 1 by any DDC path on macOS.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/tyvsmith/streamcontroller-lg-monitor-control>
- <https://github.com/meer-cha/lg-input-switch>
- <https://github.com/phillip9933/LGInputSwitch>
- <https://github.com/MonitorControl/MonitorControl/issues/1872>
- <https://github.com/waydabber/BetterDisplay/discussions/5353>

### LG 27GP850-B

- Same contested family as the 32GP850-B. kaleb422 reports the three values working over NVAPI on Windows and ships them in the NVapi-write-value-to-monitor README, while other owners of GP850 units — including a 27GP850P-B and a unit on firmware 3.06 — see only a flicker, and an LG firmware list names GP850 as unsupported. Recorded, disabled, with the disagreement written down.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/2>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695120912>

### LG 32UN880-B

- Reported independently on the NVapi-write-value-to-monitor tracker (switching from DisplayPort to USB-C, HDMI1, HDMI2 and DP), in ddcutil issue #100, and by way of the amdddc-windows tool in ddcutil issue #612, which uses 0x90.
- One quirk is worth repeating to anyone who verifies it: after switching to USB-C, the monitor stopped accepting commands sent from the DisplayPort and HDMI sides.

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/8>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2142184960>
- <https://github.com/rockowitz/ddcutil/issues/612>

### LG 34WN780

- Weaker evidence: the report quotes the three NVAPI values for this model and says "Worked well", without saying which it tried.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317>

### LG 34WN650-W

- Weaker evidence: the report quotes erenard's three values, says "Works" with ddcutil 2.1.2, and does not break the result down per input.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750>

### LG 28MQ780-B

- USB-C is deliberately not recorded. amildahl tested 0xD1, while shinyquagsire23's lg_display_manager — which keys off the model string `28MQ780` — another commenter on that gist with a 28MQ780-B, and a blog write-up all use 0xD2. Two values for one input, on the same model, is exactly the case the catalog refuses to guess at, so the input is left out until somebody tests both.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195>
- <https://gist.github.com/shinyquagsire23/f6b2adef253c6c3ab557a4852bf3abad>

### LG 32GR93U-B

- gzougianos reports the "exact same numbers" as the NVapi-write-value-to-monitor README working over NVAPI. An m1ddc issue separately reports 208 and 144 working through BetterDisplay's LG alternate input on this model, which is the same two values in decimal.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727>
- <https://github.com/waydabber/m1ddc/issues/48>

### LG 32GP83B

- Reported by Gilgame24 in ddcutil discussion 331.

Sources:

- <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333>

### LG 34U650A-B

- The report gives the values in decimal — ddcAlt 144, 145 and 208 "work correctly". The USB-C value is the one entry in the catalog that does not fit in a byte: 210 got no response, and 465 did.

Sources:

- <https://github.com/waydabber/BetterDisplay/issues/4853>
- <https://github.com/waydabber/BetterDisplay/discussions/4883>

### LG 40WP95X

- Only USB-C is reported: "This is how switching to USB-C input works".

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5>

### LG 34GS95QE

- Vib0 reports the three values as "Tested". A BetterDisplay discussion separately shows a 34GS95QE-B working through the LG alternate input, but the values there appear only in a screenshot, so that report corroborates the mechanism rather than the numbers.

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302>
- <https://github.com/waydabber/BetterDisplay/discussions/4246>

### LG 32UP83AK-W

- The reporter says 0xD0 "works without any issues" and that they switch to HDMI and DisplayPort, but never writes the HDMI value down, so only DisplayPort is recorded.

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/2#issuecomment-3221830027>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/3#issuecomment-3224103812>

### LG 32BL95U-W

- The one entry documented by the manufacturer rather than by a user. The service manual's cover says `MODEL : 32BL95U`; the marketed name with its suffix is established by cross-reference, from the manual's file name `32BL95UW.pdf` and LG's own product page, which lists `32BL95U-W.AUB` as the only 32BL95U variant. Thunderbolt is the first use of that connector kind in the catalog.
- The EDID product IDs the manual assigns — 0x7706 for HDMI, 0x7707 for DisplayPort, 0x7722 for Thunderbolt, model name `LG HDR 4K` — are not recorded as identities, because a model that is not write-enabled must record none; they would also collide with other LG models, which is discussed at the end of this page.
- The spec table's `User Model Name 32UL950` gets no entry of its own and no merge: an alias in a manual is not a report that a 32UL950 switches inputs.

Sources:

- <https://research.encompass.com/ZEN/sm/32BL95UW.pdf>
- <https://www.lg.com/us/support/product/lg-32BL95U-W>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1435477786>

### LG 32U990A

- The weakest entry here. `lg.sh` in pyang2045/streamdeck-display-knob is described as "Manual control of the LG UltraFine 32U990A" and sends m1ddc `input-alt` 144, 208 and 210, but the script does not say whether it was run successfully. A single HDMI port is reported, so the input is written bare.

Sources:

- <https://github.com/pyang2045/streamdeck-display-knob>

### Samsung LC49G95T

- The first record of the `vcp-input-source` mechanism, and the first entry that is not an LG. It is disabled and carries no identity like every other unverified entry, so that backend path has never reached a monitor: enabling this model would be the first hardware run of the mechanism, and belongs in the [testing.md](testing.md) checklist rather than in a routine catalog flip.
- The values the monitor reports back are not the values that select an input. After a switch, reading VCP 0x60 gives 0x03 for DP1, 0x04 for DP2 and 0x01 for HDMI, and writing those back does not switch. That is one reason monmux never confirms a switch by reading a monitor.
- DDC/CI answers only on the HDMI input; the DisplayPort inputs do not expose slave address 0x37 at all, so the switch has to be sent from HDMI.
- Switching to an input with no signal wedges the monitor's DDC engine until a link reset or a trip through the OSD. Never switch blind: whoever verifies this model needs the target input already connected.
- The capabilities string declares Input Source values the monitor does not have, so it is not a source of values for this model.
- EDID, as text only: manufacturer `SAM`, model name `LC49G95T`. No product code has been published, which is the other reason the entry records no identity.

Sources:

- <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md>
- <https://github.com/DimpiM/monitor-switch/blob/main/service/profiles/samsung-lc49g95t.yaml>

<!-- END GENERATED -->

## What is deliberately not here

The table records what monmux could be taught to do. Several things people have published cannot be written down in it at all,
and the reasons are worth stating, because "it is not in the table" is otherwise indistinguishable from "nobody looked".

**A report with no named input.** `vcp-input-source` now exists, so a `VCP 0x60` report is representable — the Samsung Odyssey
G9 is the first one recorded. What is still left out is the bulk of what people publish about `0x60`: ddccontrol-db and the
reports about the LG 34UM88C-P, 32GK650F-B, 29UM57-P and 27UD88-W give values without saying which socket each one lit up on
the unit that was tested, and a value with no input name has nothing to key an entry on. The rules below have not moved: a
mechanism existing is not evidence that a model uses it.

**More than one write.** A 34WP75C is reported switching to USB-C with `0xD1`, but only when the command is sent twice: the
first write bounces back to DisplayPort. monmux performs exactly one write, so a row for it would not reproduce the reported
switch.

**An input named only by its value.** "0x90 works" says which byte was sent, not which socket lit up. The 35WN75C-B, 27BP85UN,
27GR83Q-B, 34GP63A-B, 32UN880P-B, 34WN650, 32UN650K, 32QN55T and 39GX950B reports stop there, so there is nothing to key an
entry on.

**A conflict with no way to resolve it.** The 34GL750 is reported working over the side channel in one place and reported to
need `0x60` in another; it is omitted entirely rather than recorded on a coin flip. The USB-C value of the 28MQ780-B is
omitted for the same reason, and its entry says so.

**A source nobody else can open.** A blog post reports a 27GP95RP switching to DisplayPort with `0xD0` through
amdddc-windows on Windows. No stable link to it could be established, and the only citable page about that model, in ddcutil
issue #100, is about the 27GP95R — a different model name — and reports that it did not respond at all. An entry whose bytes
rest on a citation a reader cannot check is worse than no entry, so there is none.

**A negative result.** There is no slot for "this model is known not to switch". The 27GP850P-B, 27GN850-B, 27GN950, 34GN850-B,
38WN95C-W, 43UN700-B, 27UL600, 27GP95R, 40BP95C-W, the 2016 29UM69G-B and one 2025 45GX950A have all been reported as not
switching over this channel, and one 38WN95CP-W was bricked by an unrelated DDC write. Those reports are written into the notes
above where they touch an entry, and nowhere else. Treat this page as "no evidence of a problem", never as "known good".

**Not a model.** "LG HDR WQHD" and "LG UltraFine" are EDID model strings shared by many products. A report that names only one
of those, or only a serial number, cannot be attached to a model.

## EDID product codes are reused

LG assigns the same EDID product code to more than one product. ddccontrol-db maps `GSM 0x7706` to the 27UK850-W family and
`GSM 0x7707` to the 32UD99 and the 27UN880-B, while the 32BL95U service manual assigns `0x7706`, `0x7707` and `0x7722` to that
model's HDMI, DisplayPort and Thunderbolt inputs.

The catalog requires one identity to match one model, and the matcher refuses a display that several models claim: an ambiguous
match must never turn into a write. An identity may therefore pin the EDID model name — descriptor `0xFC` on Linux, the
`Product name` m1ddc prints on macOS — and two identities collide only when the manufacturer and the product code are equal
**and** either pins no name, or both pin the same one. So two models may share a reused code, provided both pin, with different
names.

A bare identity beside a pinned one with the same code is rejected, because the bare one matches every display with that code
and would shadow the pinned entry. That makes bringing a real EDID for one of these models a change to the other entry as well
as to yours: read [adding-a-monitor.md](adding-a-monitor.md) before collecting one, and say so in the pull request rather than
working around it. No entry pins a name today.
