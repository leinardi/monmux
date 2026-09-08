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

monmux implements two mechanisms. `lg-alt-input` is the LG side channel, and every model recorded with it is an LG;
`vcp-input-source` is the standard Input Source feature, `VCP 0x60`, and is what every other vendor here uses, plus three LGs
whose reports went through it. Every `vcp-input-source` row is disabled, and no monitor has ever been switched by monmux over
it. "Standard" is a name, not a promise: which values a model accepts, and whether it accepts the write at all, are still
per-model facts with per-input evidence, and the values recorded under it range from the specification's 0x0F/0x11 to 0x05,
0x09, 0x13 and 0x00.

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

The table is ordered the way the catalog file is written: the models monmux will write to first, then by vendor, then by model
name. The generator enforces that order on the file, so the two cannot drift apart.

<!-- BEGIN GENERATED: edit internal/catalog/models.yaml, then run make go-generate -->

## Catalog

| Model | Identities | Input | Value | Mechanism | Enabled | Grade | Evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| LG 38WR85QC-W | GSM/0x77D3, GSM/0x77D4 | `dp` | `0xD0` | `lg-alt-input` | yes | verified | Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc): switched from USB-C to DisplayPort |
| LG 38WR85QC-W | GSM/0x77D3, GSM/0x77D4 | `usb-c` | `0xD1` | `lg-alt-input` | yes | verified | Direct test on the unit, 2026-09-07, Linux (ddcutil) and macOS (m1ddc): switched from DisplayPort to USB-C |
| Alienware AW2725DF | none | `dp1` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact AW2725DF by markhagemann (display-switch issue 157) (display-switch 1.4.0 on Linux): switching to DisplayPort 1 with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/haimgel/display-switch/issues/157> |
| Alienware AW2725DF | none | `dp2` | `0x13` | `vcp-input-source` | no | reported | Reported working on the exact AW2725DF by markhagemann (display-switch issue 157) (display-switch 1.4.0 on Linux): switching to DisplayPort 2 with 0x13 over the standard Input Source feature (VCP 0x60) succeeded; the capabilities string lists 0x13 as an unrecognised value; 0x10 does not switch; not verified here: <https://github.com/haimgel/display-switch/issues/157> |
| Alienware AW2725DF | none | `hdmi` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact AW2725DF by markhagemann (display-switch issue 157) (display-switch 1.4.0 on Linux): switching to HDMI with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; used as the working alternative before 0x13 was found; not verified here: <https://github.com/haimgel/display-switch/issues/157> |
| AOC AG493UCX | none | `dp1` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact AG493UCX by Prydon9 (ddcutil issue 314) (ddcutil on Debian 11, sent from an Intel NUC on HDMI-1): switching to DisplayPort 1 with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; switched from HDMI-1 and back repeatedly, also while USB-C was displayed; not verified here: <https://github.com/rockowitz/ddcutil/issues/314> |
| AOC AG493UCX | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact AG493UCX by Prydon9 (ddcutil issue 314) (ddcutil on Debian 11, sent from an Intel NUC on HDMI-1): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/314> |
| AOC Q27P1B | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact Q27P1B by denilsonsa (ddcutil issue 385) (ddcutil 2.1.3 on Manjaro Linux, sent over DisplayPort): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; never read back; 0x305 is read while DisplayPort is displayed; not verified here: <https://github.com/rockowitz/ddcutil/issues/385> |
| AOC Q27P1B | none | `hdmi` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact Q27P1B by denilsonsa (ddcutil issue 385) (ddcutil 2.1.3 on Manjaro Linux, sent over DisplayPort): switching to HDMI with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; never read back; 0x300 is read after the switch; not verified here: <https://github.com/rockowitz/ddcutil/issues/385> |
| AOC Q27P1B | none | `dvi` | `0x03` | `vcp-input-source` | no | reported | Reported working on the exact Q27P1B by denilsonsa (ddcutil issue 385) (ddcutil 2.1.3 on Manjaro Linux, sent over DisplayPort): switching to DVI with 0x03 over the standard Input Source feature (VCP 0x60) succeeded; never read back; 0x300 is read after the switch; not verified here: <https://github.com/rockowitz/ddcutil/issues/385> |
| AOC Q27P1B | none | `vga` | `0x01` | `vcp-input-source` | no | reported | Reported working on the exact Q27P1B by denilsonsa (ddcutil issue 385) (ddcutil 2.1.3 on Manjaro Linux, sent over DisplayPort): switching to VGA with 0x01 over the standard Input Source feature (VCP 0x60) succeeded; reads back 0x01 once the switch completes; not verified here: <https://github.com/rockowitz/ddcutil/issues/385> |
| AOC U2790B | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact U2790B by Evgeni Golov (die-welt.net, Building a simple KVM switch for 30 EUR) (ddcutil on Linux, run from a udev rule on keyboard hot-plug): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; VCP 0x60 read 0x0F while DisplayPort was displayed before the test; not verified here: <https://github.com/evgeni/die-welt.net/blob/devel/posts/2021/01/building-a-simple-kvm-switch-for-30eur.md> |
| AOC U2790B | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact U2790B by Evgeni Golov (die-welt.net, Building a simple KVM switch for 30 EUR) (ddcutil on Linux, run from a udev rule on keyboard hot-plug): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/evgeni/die-welt.net/blob/devel/posts/2021/01/building-a-simple-kvm-switch-for-30eur.md> |
| AOC U27N3R | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact U27N3R by zhufeng (ddcutil issue 580) (ddcutil on an Ubuntu 24.04.3 live CD (Lenovo laptop)): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/580> |
| AOC U27N3R | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact U27N3R by zhufeng (ddcutil issue 580) (ddcutil on an Ubuntu 24.04.3 live CD (Lenovo laptop)): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/580> |
| AOC U27N3R | none | `hdmi2` | `0x12` | `vcp-input-source` | no | reported | Reported working on the exact U27N3R by zhufeng (ddcutil issue 580) (ddcutil on an Ubuntu 24.04.3 live CD (Lenovo laptop)): switching to HDMI 2 with 0x12 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/580> |
| AOC U27U2DS | none | `dp` | `0x0F` | `vcp-input-source` | no | quoted | Weaker report on the exact U27U2DS by Ding998 (ddcctl issue 67) (not stated): the report quotes 0x0F for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/kfix/ddcctl/issues/67> |
| AOC U27U2DS | none | `hdmi1` | `0x11` | `vcp-input-source` | no | quoted | Weaker report on the exact U27U2DS by Ding998 (ddcctl issue 67) (not stated): the report quotes 0x11 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/kfix/ddcctl/issues/67> |
| AOC U27U2DS | none | `hdmi2` | `0x12` | `vcp-input-source` | no | quoted | Weaker report on the exact U27U2DS by Ding998 (ddcctl issue 67) (not stated): the report quotes 0x12 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/kfix/ddcctl/issues/67> |
| Asus PA328Q | none | `dp1` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact PA328Q by Mahmud Ridwan (hjr265.me, Switch Monitor Input from Linux Command Line) (ddcutil on Linux): switching to DisplayPort 1 with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/hjr265/hjr265.me/blob/master/content/blog/switch-monitor-input-from-linux-command-line.md> |
| Asus PA328Q | none | `dp2` | `0x10` | `vcp-input-source` | no | reported | Reported working on the exact PA328Q by Mahmud Ridwan (hjr265.me, Switch Monitor Input from Linux Command Line) (ddcutil on Linux): switching to DisplayPort 2 with 0x10 over the standard Input Source feature (VCP 0x60) succeeded; the Mini DisplayPort connector; not verified here: <https://github.com/hjr265/hjr265.me/blob/master/content/blog/switch-monitor-input-from-linux-command-line.md> |
| Asus VG279Q1A | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact VG279Q1A by wbeuil (ddcctl issue 97) (ddcctl on a MacBook Pro 16 through a CalDigit TS3 Plus dock on HDMI): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/97> |
| BenQ PD3220U | none | `thunderbolt` | `0x14` | `vcp-input-source` | no | reported | Reported working on the exact PD3220U by eljobe (BetterDisplay discussion 2903) (BetterDisplay on two MacBook Pros): switching to Thunderbolt with 0x14 over the standard Input Source feature (VCP 0x60) succeeded; the entry BetterDisplay labels Other 2; not verified here: <https://github.com/waydabber/BetterDisplay/discussions/2903> |
| BenQ PD3226G | none | `thunderbolt` | `0x13` | `vcp-input-source` | no | reported | Reported working on the exact PD3226G by manzoorwanijk (BetterDisplay discussion 5647) (BetterDisplay 4.3.5 on an Apple Silicon Mac over HDMI): switching to Thunderbolt with 0x13 over the standard Input Source feature (VCP 0x60) succeeded; the entry BetterDisplay labels HDMI 3; not verified here: <https://github.com/waydabber/BetterDisplay/discussions/5647> |
| Dell AW3425DW | none | `dp` | `0x0F` | `vcp-input-source` | no | quoted | Weaker report on the exact AW3425DW by a contributor on the ddcutil Dell wiki page (ddcutil): the report quotes 0x0F for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Dell> |
| Dell AW3425DW | none | `hdmi1` | `0x11` | `vcp-input-source` | no | quoted | Weaker report on the exact AW3425DW by a contributor on the ddcutil Dell wiki page (ddcutil): the report quotes 0x11 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Dell> |
| Dell AW3425DW | none | `hdmi2` | `0x12` | `vcp-input-source` | no | quoted | Weaker report on the exact AW3425DW by a contributor on the ddcutil Dell wiki page (ddcutil): the report quotes 0x12 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Dell> |
| Dell P2720DC | none | `hdmi` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact P2720DC by nichcuta (display-switch issue 86) (display-switch on Windows 10 over a DisplayPort-to-HDMI cable): switching to HDMI with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/haimgel/display-switch/issues/86> |
| Dell P2720DC | none | `usb-c` | `0x1B` | `vcp-input-source` | no | reported | Reported working on the exact P2720DC by nichcuta (display-switch issue 86) (display-switch on a MacBook Pro (Catalina) over USB-C): switching to USB-C with 0x1B over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/haimgel/display-switch/issues/86> |
| Dell S3423DWC | none | `hdmi1` | `0x11` | `vcp-input-source` | no | quoted | Weaker report on the exact S3423DWC by idanizi (dell-monitor-switch README) (m1ddc set input on Apple Silicon over USB-C): the report quotes 0x11 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/idanizi/dell-monitor-switch> |
| Dell S3423DWC | none | `hdmi2` | `0x12` | `vcp-input-source` | no | quoted | Weaker report on the exact S3423DWC by idanizi (dell-monitor-switch README) (m1ddc set input on Apple Silicon over USB-C): the report quotes 0x12 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/idanizi/dell-monitor-switch> |
| Dell S3423DWC | none | `usb-c` | `0x1B` | `vcp-input-source` | no | quoted | Weaker report on the exact S3423DWC by idanizi (dell-monitor-switch README) (m1ddc set input on Apple Silicon over USB-C): the report quotes 0x1B for USB-C and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/idanizi/dell-monitor-switch> |
| Dell U2412M | none | `dvi` | `0x03` | `vcp-input-source` | no | reported | Reported working on the exact U2412M by pranavanmaru (ddcctl issue 103) (ddcctl on macOS over DisplayPort): switching to DVI with 0x03 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/103> |
| Dell U2720Q | none | `usb-c` | `0x1B` | `vcp-input-source` | no | reported | Reported working on the exact U2720Q by nmostafavi (display-switch issue 6) (display-switch on Windows (dxva2 SetVCPFeature)): switching to USB-C with 0x1B over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/haimgel/display-switch/issues/6> |
| Dell U2723QE | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact U2723QE by tjk213 (tk-dotfiles, swap-sources.sh) (m1ddc set input 15 on an M1 MacBook driving two U2723QE): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; the script comments that the display switches as desired even when m1ddc then reports a DDC communication failure; not verified here: <https://github.com/tjk213/tk-dotfiles/blob/main/core/swap-sources.sh> |
| Dell U2724DE | none | `usb-c` | `0x19` | `vcp-input-source` | no | reported | Reported working on the exact U2724DE by manzoorwanijk (BetterDisplay discussion 5647) (BetterDisplay 4.3.5 on an Apple Silicon Mac): switching to USB-C with 0x19 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/waydabber/BetterDisplay/discussions/5647> |
| Dell U3219Q | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact U3219Q by hawktang (ddcctl issue 120) (ddcctl on a MacBook Pro 2017): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/120> |
| Dell U3421WE | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact U3421WE by Jean-Charles Quillet (blog post: How to use ddcutil to switch input of a Dell screen) (ddcutil on NixOS, a toggle script between the two inputs): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/jecaro/jeancharles.quillet/blob/master/posts/2021-08-20-How-to-use-ddcutil-to-switch-input-of-a-Dell-screen.md> |
| Dell U3421WE | none | `usb-c` | `0x1B` | `vcp-input-source` | no | reported | Reported working on the exact U3421WE by Jean-Charles Quillet (blog post: How to use ddcutil to switch input of a Dell screen) (ddcutil on NixOS, a toggle script between the two inputs): switching to USB-C with 0x1B over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/jecaro/jeancharles.quillet/blob/master/posts/2021-08-20-How-to-use-ddcutil-to-switch-input-of-a-Dell-screen.md> |
| Dell U3818DW | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact U3818DW by tsarath (ddcctl issue 76) (ddcctl on a Mac mini 2018 over DisplayPort): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/76> |
| Dell U3818DW | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact U3818DW by tsarath (ddcctl issue 76) (ddcctl on a Mac mini 2018 over DisplayPort): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/76> |
| Dell U3818DW | none | `hdmi2` | `0x12` | `vcp-input-source` | no | reported | Reported working on the exact U3818DW by tsarath (ddcctl issue 76) (ddcctl on a Mac mini 2018 over DisplayPort): switching to HDMI 2 with 0x12 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/76> |
| Dell U3818DW | none | `usb-c` | `0x1B` | `vcp-input-source` | no | reported | Reported working on the exact U3818DW by hdansou (ddcctl issue 76) (ddcctl on macOS): switching to USB-C with 0x1B over the standard Input Source feature (VCP 0x60) succeeded; switched to USB-C and back to DisplayPort with 15; not verified here: <https://github.com/kfix/ddcctl/issues/76> |
| HKC G27M7Pro | none | `dp` | `0x07` | `vcp-input-source` | no | quoted | Weaker report on the exact G27M7Pro by Star-ZER0 (Twinkle Tray issue 1156) (Twinkle Tray custom VCP on Windows): the report quotes 0x07 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/xanderfrangos/twinkle-tray/issues/1156> |
| HKC G27M7Pro | none | `hdmi1` | `0x05` | `vcp-input-source` | no | quoted | Weaker report on the exact G27M7Pro by Star-ZER0 (Twinkle Tray issue 1156) (Twinkle Tray custom VCP on Windows): the report quotes 0x05 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/xanderfrangos/twinkle-tray/issues/1156> |
| HKC G27M7Pro | none | `hdmi2` | `0x06` | `vcp-input-source` | no | quoted | Weaker report on the exact G27M7Pro by Star-ZER0 (Twinkle Tray issue 1156) (Twinkle Tray custom VCP on Windows): the report quotes 0x06 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/xanderfrangos/twinkle-tray/issues/1156> |
| HKC G27M7Pro | none | `usb-c` | `0x08` | `vcp-input-source` | no | quoted | Weaker report on the exact G27M7Pro by Star-ZER0 (Twinkle Tray issue 1156) (Twinkle Tray custom VCP on Windows): the report quotes 0x08 for USB-C and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/xanderfrangos/twinkle-tray/issues/1156> |
| HP Z27n G2 | none | `dp2` | `0x13` | `vcp-input-source` | no | quoted | Weaker report on the exact Z27n G2 by a contributor on the ddcutil HP wiki page (ddcutil): the report quotes 0x13 for DisplayPort 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/wiki/HP> |
| LG 27BN88Q-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27BN88Q-B by bansheerubber (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1646752889> |
| LG 27BN88Q-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27BN88Q-B by bansheerubber (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1646752889> |
| LG 27GL83A-B | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact 27GL83A-B by lauhayden and nichcuta (display-switch issue 86) (display-switch on Windows and macOS, and ddcutil setvcp 60, sent from the HDMI 1 side): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; the switch happens, then the monitor shows an "Out of Range" overlay until the input is reselected in the OSD; not verified here: <https://github.com/haimgel/display-switch/issues/86> |
| LG 27GL83A-B | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact 27GL83A-B by lauhayden and nichcuta (display-switch issue 86) (display-switch on Windows and macOS, and ddcutil setvcp 60, sent from the DisplayPort side): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/haimgel/display-switch/issues/86> |
| LG 27GP850-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27GP850-B by kaleb422 (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595> |
| LG 27GP850-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27GP850-B by kaleb422 (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595> |
| LG 27GP850-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27GP850-B by kaleb422 (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595> |
| LG 27UK500-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UK500-B by francis36012 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UK500-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UK500-B by francis36012 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UK500-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UK500-B by francis36012 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UL550-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UL550-W by titou10titou10 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UL550-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UL550-W by titou10titou10 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UL550-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UL550-W by titou10titou10 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-W by the tester who added it to the ddcutil LG wiki page (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the report switched from the USB-C input; the wiki row records the model as confirmed without values, which that tester quotes in the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011> |
| LG 27UN850-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-W by the tester who added it to the ddcutil LG wiki page (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the report switched from the USB-C input; the wiki row records the model as confirmed without values, which that tester quotes in the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011> |
| LG 27UN850-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-W by the tester who added it to the ddcutil LG wiki page (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the report switched from the USB-C input; the wiki row records the model as confirmed without values, which that tester quotes in the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011> |
| LG 27UN850-WY | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-WY | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-WY | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UN850-WY | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 27UN850-WY by ccrxf (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UP850-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP850-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP850-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP850-W | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 27UP850-W by Prototyped (ddcutil 2.0, --i2c-source-addr=x50): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1694629234> |
| LG 27UP85NP-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27UP85NP-W by edror12 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27UP85NP-W | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 27UP85NP-W by edror12 (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27US500-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 27US500-W by Fidelxyz (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27US500-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 27US500-W by Fidelxyz (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 27US500-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 27US500-W by Fidelxyz (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 28MQ780-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 28MQ780-B by amildahl (Windows, ADL): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195> |
| LG 28MQ780-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 28MQ780-B by amildahl (Windows, ADL): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195> |
| LG 28MQ780-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 28MQ780-B by amildahl (Windows, ADL): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195> |
| LG 29U531A | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 29U531A by tinkererkzy (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29U531A | none | `hdmi` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 29U531A by tinkererkzy (ddcutil): switching to HDMI with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29U531A | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 29U531A by tinkererkzy (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29UM69G | none | `dp` | `0xC0` | `lg-alt-input` | no | reported | Reported working on the exact 29UM69G by jonpas (ddcutil): switching to DisplayPort with 0xC0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788> |
| LG 29UM69G | none | `hdmi` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 29UM69G by jonpas (ddcutil): switching to HDMI with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788> |
| LG 29UM69G | none | `usb-c` | `0xE0` | `lg-alt-input` | no | reported | Reported working on the exact 29UM69G by jonpas (ddcutil): switching to USB-C with 0xE0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788> |
| LG 29WN600 | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 29WN600 by iamSlightlyWind (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29WN600 | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 29WN600 by iamSlightlyWind (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 29WN600 | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 29WN600 by iamSlightlyWind (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32BL95U-W | none | `dp` | `0xD0` | `lg-alt-input` | no | documented | Documented by LG for the exact 32BL95U-W: DisplayPort is 0xD0 over the LG side channel (source address 0x50, VCP 0xF4); no field report; the service manual's Screen adjust command table, row 13, maps Input Select, command F4, to this value; not verified here: <https://research.encompass.com/ZEN/sm/32BL95UW.pdf> |
| LG 32BL95U-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | documented | Documented by LG for the exact 32BL95U-W: HDMI 1 is 0x90 over the LG side channel (source address 0x50, VCP 0xF4); no field report; the service manual's Screen adjust command table, row 13, maps Input Select, command F4, to this value; not verified here: <https://research.encompass.com/ZEN/sm/32BL95UW.pdf> |
| LG 32BL95U-W | none | `thunderbolt` | `0xD2` | `lg-alt-input` | no | documented | Documented by LG for the exact 32BL95U-W: Thunderbolt is 0xD2 over the LG side channel (source address 0x50, VCP 0xF4); no field report; the service manual's Screen adjust command table, row 13, maps Input Select, command F4, to this value; not verified here: <https://research.encompass.com/ZEN/sm/32BL95UW.pdf> |
| LG 32GP750-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GP750-B by chris-pcguy (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32GP750-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GP750-B by chris-pcguy (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32GP750-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GP750-B by chris-pcguy (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 32GP83B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GP83B by Gilgame24 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333> |
| LG 32GP83B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GP83B by Gilgame24 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333> |
| LG 32GP83B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GP83B by Gilgame24 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333> |
| LG 32GP850-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GP850-B by erenard (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145> |
| LG 32GP850-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GP850-B by erenard (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145> |
| LG 32GP850-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GP850-B by erenard (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145> |
| LG 32GR93U-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32GR93U-B by gzougianos (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727> |
| LG 32GR93U-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32GR93U-B by gzougianos (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727> |
| LG 32GR93U-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32GR93U-B by gzougianos (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727> |
| LG 32QN650-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32QN650-B by agspoon (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046> |
| LG 32QN650-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32QN650-B by agspoon (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046> |
| LG 32QN650-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32QN650-B by agspoon (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046> |
| LG 32U990A | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 32U990A by pyang2045 (m1ddc input-alt on macOS): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/pyang2045/streamdeck-display-knob> |
| LG 32U990A | none | `hdmi` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 32U990A by pyang2045 (m1ddc input-alt on macOS): the report quotes 0x90 for HDMI and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/pyang2045/streamdeck-display-knob> |
| LG 32U990A | none | `thunderbolt` | `0xD2` | `lg-alt-input` | no | quoted | Weaker report on the exact 32U990A by pyang2045 (m1ddc input-alt on macOS): the report quotes 0xD2 for Thunderbolt and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/pyang2045/streamdeck-display-knob> |
| LG 32UD99-W | none | `dp` | `0xE0` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to DisplayPort with 0xE0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; found by looping over candidate values; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636997772> |
| LG 32UD99-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636761366> |
| LG 32UD99-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636761366> |
| LG 32UD99-W | none | `usb-c` | `0xC0` | `lg-alt-input` | no | reported | Reported working on the exact 32UD99-W by andrewgodman (Lunar on macOS): switching to USB-C with 0xC0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; found by looping over candidate values; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636997772> |
| LG 32UN880-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 32UN880-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 32UN880-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 32UN880-B | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 32UN880-B by two reporters (Windows, NVAPI): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567> |
| LG 32UP83AK-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 32UP83AK-W by 5uck1ess (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/2#issuecomment-3221830027> |
| LG 34GS95QE | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 34GS95QE by Vib0 (Windows, NVAPI): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302> |
| LG 34GS95QE | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 34GS95QE by Vib0 (Windows, NVAPI): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302> |
| LG 34GS95QE | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 34GS95QE by Vib0 (Windows, NVAPI): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302> |
| LG 34U650A-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay on macOS): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 34U650A-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay on macOS): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 34U650A-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay on macOS): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 34U650A-B | none | `usb-c` | `0x1D1` | `lg-alt-input` | no | reported | Reported working on the exact 34U650A-B by mikecarlton (BetterDisplay ddcAlt, macOS): switching to USB-C with 0x1D1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; found by looping over values; the listed 0xD2 got no response; 0x1D1 is 465 decimal; not verified here: <https://github.com/waydabber/BetterDisplay/issues/4853> |
| LG 34UC98-W | none | `hdmi1` | `0x11` | `vcp-input-source` | no | quoted | Weaker report on the exact 34UC98-W by nathang21 (ddcctl issue 67) (ddcctl on macOS): the report quotes 0x11 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/kfix/ddcctl/issues/67> |
| LG 34UC98-W | none | `hdmi2` | `0x12` | `vcp-input-source` | no | quoted | Weaker report on the exact 34UC98-W by nathang21 (ddcctl issue 67) (ddcctl on macOS): the report quotes 0x12 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/kfix/ddcctl/issues/67> |
| LG 34UC98-W | none | `thunderbolt` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact 34UC98-W by nathang21 (ddcctl issue 67) (ddcctl on macOS): switching to Thunderbolt with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/67> |
| LG 34UM88C-P | none | `dp` | `0x00` | `vcp-input-source` | no | quoted | Weaker report on the exact 34UM88C-P by inkhey (ddcutil discussion 331) (ddcutil 1.2.2 on Pop!_OS 22.04, in PBP and standard mode): the report quotes 0x00 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-7152259> |
| LG 34UM88C-P | none | `hdmi1` | `0x01` | `vcp-input-source` | no | quoted | Weaker report on the exact 34UM88C-P by inkhey (ddcutil discussion 331) (ddcutil 1.2.2 on Pop!_OS 22.04, in PBP and standard mode): the report quotes 0x01 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-7152259> |
| LG 34UM88C-P | none | `hdmi2` | `0x10` | `vcp-input-source` | no | reported | Reported working on the exact 34UM88C-P by inkhey (ddcutil discussion 331) (ddcutil 1.2.2 on Pop!_OS 22.04, in PBP and standard mode): switching to HDMI 2 with 0x10 over the standard Input Source feature (VCP 0x60) succeeded; the command shown is `ddcutil setvcp 60 0x10`; not verified here: <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-7152259> |
| LG 34WN650-W | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN650-W by wigust (ddcutil 2.1.2): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750> |
| LG 34WN650-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN650-W by wigust (ddcutil 2.1.2): the report quotes 0x90 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750> |
| LG 34WN650-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN650-W by wigust (ddcutil 2.1.2): the report quotes 0x91 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750> |
| LG 34WN750-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 34WN750-B by permezel (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 34WN750-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 34WN750-B by permezel (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 34WN750-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 34WN750-B by permezel (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 34WN780 | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN780 by piaverous (Windows, NVAPI): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317> |
| LG 34WN780 | none | `hdmi1` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN780 by piaverous (Windows, NVAPI): the report quotes 0x90 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317> |
| LG 34WN780 | none | `hdmi2` | `0x91` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN780 by piaverous (Windows, NVAPI): the report quotes 0x91 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317> |
| LG 34WN80C-B | none | `dp` | `0xD0` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): the report quotes 0xD0 for DisplayPort and says it works, without saying which inputs were tried individually; the wiki entry says only that the script works perfectly and does not quote the values, which come from the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 34WN80C-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): the report quotes 0x90 for HDMI 1 and says it works, without saying which inputs were tried individually; the wiki entry says only that the script works perfectly and does not quote the values, which come from the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 34WN80C-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | quoted | Weaker report on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): the report quotes 0x91 for HDMI 2 and says it works, without saying which inputs were tried individually; the wiki entry says only that the script works perfectly and does not quote the values, which come from the linked comment; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 34WN80C-B | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 34WN80C-B by nicklan (a Python USB-HID script, not ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; the commenter states that "0xd1 is actually the usb-c input" on this monitor; not verified here: <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752> |
| LG 38BR85QC | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 38BR85QC | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 38BR85QC by a tester on the ddcutil LG wiki page (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40U990A-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 40U990A-W by titou10titou10 (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40U990A-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 40U990A-W by titou10titou10 (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40U990A-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 40U990A-W by titou10titou10 (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95C-W | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95C-W by fblaese (ddcutil): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 40WP95X | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 40WP95X by stepahin (Windows, NVAPI): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5> |
| LG 45GX950A-B | none | `dp` | `0xD0` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to DisplayPort with 0xD0 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 45GX950A-B | none | `hdmi1` | `0x90` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to HDMI 1 with 0x90 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 45GX950A-B | none | `hdmi2` | `0x91` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to HDMI 2 with 0x91 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| LG 45GX950A-B | none | `usb-c` | `0xD1` | `lg-alt-input` | no | reported | Reported working on the exact 45GX950A-B by tyvsmith (ddcutil on Linux): switching to USB-C with 0xD1 over the LG side channel (source address 0x50, VCP 0xF4) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors> |
| Monoprice Dark Matter 40776 | none | `dp1` | `0x07` | `vcp-input-source` | no | reported | Reported working on the exact Dark Matter 40776 by ethack (ddcutil issue 157) (ddcutil 0.9.9 on Pop!_OS 20.04): switching to DisplayPort 1 with 0x07 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/157> |
| Monoprice Dark Matter 40776 | none | `dp2` | `0x08` | `vcp-input-source` | no | quoted | Weaker report on the exact Dark Matter 40776 by ethack (ddcutil issue 157) (ddcutil 0.9.9 on Pop!_OS 20.04): the report quotes 0x08 for DisplayPort 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/157> |
| Monoprice Dark Matter 40776 | none | `hdmi1` | `0x05` | `vcp-input-source` | no | quoted | Weaker report on the exact Dark Matter 40776 by ethack (ddcutil issue 157) (ddcutil 0.9.9 on Pop!_OS 20.04): the report quotes 0x05 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/157> |
| Monoprice Dark Matter 40776 | none | `hdmi2` | `0x06` | `vcp-input-source` | no | reported | Reported working on the exact Dark Matter 40776 by ethack (ddcutil issue 157) (ddcutil 0.9.9 on Pop!_OS 20.04): switching to HDMI 2 with 0x06 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/157> |
| MSI MPG 321URX QD-OLED | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact MPG 321URX QD-OLED by Alex Plescan (alexplescan.com, KVM post) (m1ddc set input 15 on Apple Silicon, sent from the USB-C side): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/alexpls/alexplescan.com/blob/master/content/posts/2025/kvm/index.md> |
| MSI MPG 321URX QD-OLED | none | `usb-c` | `0x10` | `vcp-input-source` | no | reported | Reported working on the exact MPG 321URX QD-OLED by Alex Plescan (alexplescan.com, KVM post) (ddcutil setvcp 0x60 0x10 on Linux (KDE), sent from the DisplayPort side): switching to USB-C with 0x10 over the standard Input Source feature (VCP 0x60) succeeded; the monitor lists this value as DisplayPort-2; not verified here: <https://github.com/alexpls/alexplescan.com/blob/master/content/posts/2025/kvm/index.md> |
| Philips 328P6VUBREB/11 | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact 328P6VUBREB/11 by sattoke (ddcutil issue 162) (ddcutil 1.0.0-dev (commit 943f8e7) on a Raspberry Pi 4): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/162> |
| Philips 328P6VUBREB/11 | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact 328P6VUBREB/11 by sattoke (ddcutil issue 162) (ddcutil 1.0.0-dev (commit 943f8e7) on a Raspberry Pi 4): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/162> |
| Philips 328P6VUBREB/11 | none | `hdmi2` | `0x12` | `vcp-input-source` | no | reported | Reported working on the exact 328P6VUBREB/11 by sattoke (ddcutil issue 162) (ddcutil 1.0.0-dev (commit 943f8e7) on a Raspberry Pi 4): switching to HDMI 2 with 0x12 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/162> |
| Philips 328P6VUBREB/11 | none | `usb-c` | `0x13` | `vcp-input-source` | no | reported | Reported working on the exact 328P6VUBREB/11 by sattoke (ddcutil issue 162) (ddcutil 1.0.0-dev (commit 943f8e7) on a Raspberry Pi 4): switching to USB-C with 0x13 over the standard Input Source feature (VCP 0x60) succeeded; reads back 0x00 while USB-C is displayed; not verified here: <https://github.com/rockowitz/ddcutil/issues/162> |
| Philips 49M2C8900L | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact 49M2C8900L by marclloydjolly (evnia-mac README) (m1ddc set input on an M1 Pro over DisplayPort): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/marclloydjolly/evnia-mac> |
| Philips 49M2C8900L | none | `hdmi1` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact 49M2C8900L by marclloydjolly (evnia-mac README) (m1ddc set input on an M1 Pro over DisplayPort): switching to HDMI 1 with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/marclloydjolly/evnia-mac> |
| Philips 49M2C8900L | none | `usb-c` | `0x15` | `vcp-input-source` | no | reported | Reported working on the exact 49M2C8900L by marclloydjolly (evnia-mac README) (m1ddc set input on an M1 Pro over DisplayPort): switching to USB-C with 0x15 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/marclloydjolly/evnia-mac> |
| Samsung C49RG94SSU | none | `dp1` | `0x09` | `vcp-input-source` | no | reported | Reported working on the exact C49RG94SSU by philipp1992 (ddcutil issue 226) (ddcutil on Linux): switching to DisplayPort 1 with 0x09 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/226> |
| Samsung G95NC | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact G95NC by ema987 (ddcutil issue 397) (ddcutil on Linux, firmware M-C9557GGPA-1007.0): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/397> |
| Samsung G95NC | none | `hdmi1` | `0x05` | `vcp-input-source` | no | reported | Reported working on the exact G95NC by ema987 (ddcutil issue 397) (ddcutil on Linux, firmware M-C9557GGPA-1007.0): switching to HDMI 1 with 0x05 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/397> |
| Samsung G95NC | none | `hdmi2` | `0x06` | `vcp-input-source` | no | reported | Reported working on the exact G95NC by ema987 (ddcutil issue 397) (ddcutil on Linux, firmware M-C9557GGPA-1007.0): switching to HDMI 2 with 0x06 over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/rockowitz/ddcutil/issues/397> |
| Samsung LC49G95T | none | `dp1` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact LC49G95T by DimpiM/monitor-switch (hardware-findings.md) (ddcutil 2.2.0 on a Raspberry Pi Zero 2 W, sent from the HDMI input): switching to DisplayPort 1 with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; reading VCP 0x60 back afterwards gives 0x03, which is not the value that selects the input; not verified here: <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md> |
| Samsung LC49G95T | none | `dp2` | `0x10` | `vcp-input-source` | no | reported | Reported working on the exact LC49G95T by DimpiM/monitor-switch (hardware-findings.md) (ddcutil 2.2.0 on a Raspberry Pi Zero 2 W, sent from the HDMI input): switching to DisplayPort 2 with 0x10 over the standard Input Source feature (VCP 0x60) succeeded; reading VCP 0x60 back afterwards gives 0x04, which is not the value that selects the input; not verified here: <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md> |
| Samsung LC49G95T | none | `hdmi` | `0x11` | `vcp-input-source` | no | reported | Reported working on the exact LC49G95T by DimpiM/monitor-switch (hardware-findings.md) (ddcutil 2.2.0 on a Raspberry Pi Zero 2 W, sent from the HDMI input): switching to HDMI with 0x11 over the standard Input Source feature (VCP 0x60) succeeded; reading VCP 0x60 back afterwards gives 0x01, which is not the value that selects the input; not verified here: <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md> |
| Samsung LS27A800U | none | `hdmi` | `0x05` | `vcp-input-source` | no | reported | Reported working on the exact LS27A800U by unai-ndz (ddcutil issue 395) (ddcutil on Linux): switching to HDMI with 0x05 over the standard Input Source feature (VCP 0x60) succeeded; 0x11 blanks the screen for a second and returns to the previous input; not verified here: <https://github.com/rockowitz/ddcutil/issues/395> |
| Samsung LS32A70 | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact LS32A70 by ehdoyle (ddcutil issue 343) (ddcutil 0.9.8 on Ubuntu 20.04): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; reads back 0x06, not 0x0F; not verified here: <https://github.com/rockowitz/ddcutil/issues/343> |
| Samsung LS32A70 | none | `hdmi` | `0x05` | `vcp-input-source` | no | reported | Reported working on the exact LS32A70 by ehdoyle (ddcutil issue 343) (ddcutil 0.9.8 on Ubuntu 20.04): switching to HDMI with 0x05 over the standard Input Source feature (VCP 0x60) succeeded; reads back 0x05; not verified here: <https://github.com/rockowitz/ddcutil/issues/343> |
| Samsung LU28R55 | none | `dp` | `0x0F` | `vcp-input-source` | no | reported | Reported working on the exact LU28R55 by Lewiscowles1986 (ddcctl issue 127) (ddcctl on macOS): switching to DisplayPort with 0x0F over the standard Input Source feature (VCP 0x60) succeeded; not verified here: <https://github.com/kfix/ddcctl/issues/127> |
| Samsung U28H750UQNXZA | none | `dp` | `0x0F` | `vcp-input-source` | no | quoted | Weaker report on the exact U28H750UQNXZA by pfps (ddcutil issue 185) (ddcutil on Linux, sent from HDMI-1): the report quotes 0x0F for DisplayPort and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/185> |
| Samsung U28H750UQNXZA | none | `hdmi1` | `0x05` | `vcp-input-source` | no | quoted | Weaker report on the exact U28H750UQNXZA by pfps (ddcutil issue 185) (ddcutil on Linux, sent from HDMI-1): the report quotes 0x05 for HDMI 1 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/185> |
| Samsung U28H750UQNXZA | none | `hdmi2` | `0x06` | `vcp-input-source` | no | quoted | Weaker report on the exact U28H750UQNXZA by pfps (ddcutil issue 185) (ddcutil on Linux, sent from HDMI-1): the report quotes 0x06 for HDMI 2 and says it works, without saying which inputs were tried individually; not verified here: <https://github.com/rockowitz/ddcutil/issues/185> |

## Notes on the entries

### LG 38WR85QC-W

- Two identities, because the same physical unit reports `GSM/0x77D3` while displaying DisplayPort and `GSM/0x77D4` while displaying USB-C. Both were observed directly; the model string is `LG ULTRAWIDE`, which is not used for matching.
- `hdmi1` and `hdmi2` are deliberately absent rather than disabled. The values were never tested on this unit, and another LG model using them is not evidence for this one. Asking for them is refused with `input-not-enabled`. Several sources say the HDMI inputs of a 38WR85QC-W respond to `0x90` and `0x91`; because this model is write-enabled, recording them would enable a write, so they stay out until somebody runs them on this unit by hand, the way [testing.md](testing.md) describes.

Sources:

- Direct observation on the tested unit: EDID GSM/0x77D3 (DisplayPort) and GSM/0x77D4 (USB-C), model string LG ULTRAWIDE
- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### Alienware AW2725DF

- A Dell brand. display-switch's own `DisplayPort2` name sends 0x10, which reports success while the monitor stays on DP 1; the second DisplayPort answers to 0x13.

Sources:

- <https://github.com/haimgel/display-switch/issues/157>

### AOC AG493UCX

- Marketed as the AGON AG493UCX. USB-C cannot be selected: 0x13 from the capabilities string and 0x02, which is what VCP 0x60 reads while USB-C is displayed, both do nothing. The reporter's workaround is to switch to an unconnected input so the monitor falls back to USB-C on its own.
- HDMI-2 and DP-2 were not tried; 0x12 and 0x10 come from the capabilities string only and are not recorded. The monitor accepts commands on an input that is not displayed.
- EDID, as text only: manufacturer `AOC`, model string `AG493UG7R4`, product code 18736 (0x4930).

Sources:

- <https://github.com/rockowitz/ddcutil/issues/314>

### AOC Q27P1B

- The reporter had two units, both connected over DisplayPort, and not enough machines to feed every input at once; each value was written and the monitor observed switching to that input. 0x02 and 0x04 also switch to VGA and DVI respectively; 0x10 and 0x12 do nothing. The capabilities string declares only 0x01 and 0x03.
- Reading VCP 0x60 back returns a 16-bit status (0x300 to 0x305) that never equals the value written, and the monitor does not answer DDC for several seconds while switching. Neither matters to monmux, which never reads a monitor to confirm a switch.
- EDID, as text only: manufacturer `AOC`, model string `Q27P1B`, product code 9985 (0x2701), taken from the `AOC-Q27P1B-9985.mccs` file name in the report.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/385>

### AOC U2790B

- Written up on 2021-01-18. HDMI-2 (0x12 in the capabilities string) was not tried.

Sources:

- <https://github.com/evgeni/die-welt.net/blob/devel/posts/2021/01/building-a-simple-kvm-switch-for-30eur.md>

### AOC U27N3R

- The report says the three values work fine to switch to. USB-C cannot be selected: VCP 0x60 reads 0x00 or 0x02 (the latter with ddcutil 2.2.0-dev on a Debian 13 live CD) while USB-C is displayed and writing either does nothing. With the OSD input set to Auto, switching to the unconnected DisplayPort makes the monitor fall through to USB-C (its input order is HDMI1, HDMI2, DP, USB-C).

Sources:

- <https://github.com/rockowitz/ddcutil/issues/580>

### AOC U27U2DS

- USB-C cannot be selected: the value 0 the monitor reports for it does nothing when written. With the OSD input on Auto and nothing on DisplayPort, switching to DisplayPort makes the monitor fall through to USB-C.

Sources:

- <https://github.com/kfix/ddcctl/issues/67>

### Asus PA328Q

- Written up on 2023-10-24. The three HDMI inputs answer to 0x11, 0x12 and 0x13 per the capabilities string and the author mapped them by trial, but the mapping is not stated, so they are left out. ddcutil cannot parse this monitor's capabilities string.

Sources:

- <https://github.com/hjr265/hjr265.me/blob/master/content/blog/switch-monitor-input-from-linux-command-line.md>

### Asus VG279Q1A

- Switching to HDMI 1 works while DisplayPort is displayed. Switching back to DisplayPort with 0x0F blinks the screen and returns to HDMI 1, so DisplayPort is not recorded; the reporter's workaround is switching to the unconnected HDMI 2, from which the monitor falls back to DisplayPort.

Sources:

- <https://github.com/kfix/ddcctl/issues/97>

### BenQ PD3220U

- The Thunderbolt input answers to 20 (0x14), which no tool labels as such. The report also switches to DisplayPort through BetterDisplay's DisplayPort 1 entry, but never states the value that entry sends, so DisplayPort is not recorded. HDMI was not reported.

Sources:

- <https://github.com/waydabber/BetterDisplay/discussions/2903>

### BenQ PD3226G

- The capabilities string is `60(0F 11 13)`; 0x19, which BetterDisplay sends for USB-C, is silently ignored. DisplayPort and HDMI were not reported switched.

Sources:

- <https://github.com/waydabber/BetterDisplay/discussions/5647>

### Dell AW3425DW

- An Alienware model listed on the ddcutil Dell wiki page with `ddcutil setvcp 60 0x0F` as the example for DisplayPort. The page also documents PiP and PbP layouts on VCP 0xE9 and game presets on VCP 0xF0, which are out of scope.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Dell>

### Dell P2720DC

- Quirk: on the second switch cycle the monitor sometimes goes to soft power-off and reports "No source detected on USB-C"; the reporter recovers by switching back to HDMI and then to USB-C again, with a delay between the two. monmux does one write and never retries.

Sources:

- <https://github.com/haimgel/display-switch/issues/86>

### Dell S3423DWC

- The README says "tested on Dell S3423DWC", ships one script per input (27, 17, 18) and says `set input` works reliably; it calls the values standard Dell values and warns that reads return garbage. No input is described being switched individually.

Sources:

- <https://github.com/idanizi/dell-monitor-switch>

### Dell U2412M

- Switching from DisplayPort to DVI works. Switching back with 15 (0x0F), sent from the DisplayPort side while DVI is displayed, does nothing; whether 0x0F works when sent from the displayed input is not reported, so DisplayPort is not recorded.

Sources:

- <https://github.com/kfix/ddcctl/issues/103>

### Dell U2720Q

- The value was found with NirSoft ControlMyMonitor and then used by display-switch: the report says it is working great. DisplayPort and HDMI were not reported.

Sources:

- <https://github.com/haimgel/display-switch/issues/6>

### Dell U2723QE

- Only the DisplayPort switch is described as observed. The same script sends `setvcp 0x60 0x1b` for USB-C on Linux, and homer0's ddc-switcher bridge configures 27 for USB-C and 17 for HDMI on two U2723QE, but neither source says those switches happened, so USB-C and HDMI are not recorded.
- After a switch, m1ddc may exit non-zero because its read-back fails once the input is gone, and the script switches its displays from the highest m1ddc index down because switching one display can make the next one fail.

Sources:

- <https://github.com/tjk213/tk-dotfiles/blob/main/core/swap-sources.sh>
- <https://github.com/homer0/ddc-switcher/blob/main/m1ddc-bridge.sh>

### Dell U2724DE

- The capabilities string is `60(19 0F 11)`, so USB-C is 0x19 here rather than the 0x1B most Dell reports use. DisplayPort and HDMI were not reported switched.

Sources:

- <https://github.com/waydabber/BetterDisplay/discussions/5647>

### Dell U3219Q

- Mentioned as the working monitor in a report about an ASUS VG27A that does not switch. HDMI and USB-C were not reported.

Sources:

- <https://github.com/kfix/ddcctl/issues/120>

### Dell U3421WE

- A second user, ManTreff, drives USB-C on the same model with `setvcp 60 0x1b` from the DellDisplayManagerLite README. The two reports give different EDID product codes, 41349 (0xA185) and 41345 (0xA181); text only, neither is an identity.
- The two HDMI inputs (0x11 and 0x12 in the capabilities string) were not tried.

Sources:

- <https://github.com/jecaro/jeancharles.quillet/blob/master/posts/2021-08-20-How-to-use-ddcutil-to-switch-input-of-a-Dell-screen.md>
- <https://github.com/ManTreff/DellDisplayManagerLite>

### Dell U3818DW

- ddcctl writes the standard VCP 0x60 feature. aryoda reports the same 0x1B for USB-C from the capabilities string of another U3818DW in ddcutil issue 70, and the ddcutil Dell wiki page lists the model; neither is a switch test and neither adds a value.

Sources:

- <https://github.com/kfix/ddcctl/issues/76>
- <https://github.com/rockowitz/ddcutil/issues/70>

### HKC G27M7Pro

- The report says 5, 6, 7 and 8 select HDMI-1, HDMI-2, DisplayPort and Type-C, and that the 15 to 18 in the capabilities string do nothing.

Sources:

- <https://github.com/xanderfrangos/twinkle-tray/issues/1156>

### HP Z27n G2

- The wiki entry says the second DisplayPort input is 0x13 instead of the standard 0x10, that the capabilities string reports it accurately, and that after a switch VCP 0x60 keeps reading the old input until the new source carries a signal. The first DisplayPort, HDMI and USB-C are not given.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/HP>

### LG 27BN88Q-B

- Listed on the ddcutil LG wiki page by bansheerubber, who also wrote the values out in ddcutil issue #100: "successfully switching between HDMI 1 (0x90) and display port 1 (0xD0)". Only those two inputs are recorded; no report covers the others.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1646752889>

### LG 27GL83A-B

- Driven by the standard Input Source feature. Three quirks: the monitor answers DDC only on the input currently displayed; switching from HDMI 1 to DisplayPort brings up an "Out of Range" overlay on the new input, whether or not FreeSync is on; and the OSD keeps showing the old input after a DDC switch. ddccontrol-db issue 134 reports the same stale OSD from ddccontrol and from ControlMyMonitor on Windows, and closed it as monitor firmware behaviour.
- The numbers come from nichcuta, who runs the same model with 17 and 15 from a Windows PC on DisplayPort and a Mac on HDMI and reports it working once the OSD input is left on DisplayPort. lauhayden used display-switch's `Hdmi1` and `Displayport1` names, which the tool maps to 0x11 and 0x0F, and `ddcutil setvcp 60` without quoting the value.

Sources:

- <https://github.com/haimgel/display-switch/issues/86>
- <https://github.com/ddccontrol/ddccontrol-db/issues/134>

### LG 27GP850-B

- Same contested family as the 32GP850-B. kaleb422 reports the three values working over NVAPI on Windows and ships them in the NVapi-write-value-to-monitor README, while other owners of GP850 units — including a 27GP850P-B and a unit on firmware 3.06 — see only a flicker, and an LG firmware list names GP850 as unsupported. Recorded, disabled, with the disagreement written down.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2106185595>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/2>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695120912>

### LG 27UK500-B

- Test result contributed by francis36012 to the ddcutil LG wiki page. No EDID fingerprint was published with it.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 27UL550-W

- Test result contributed by titou10titou10 to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 27UN850-W

- The wiki row says "confirmed" and gives no values. The values come from the same tester's comment on ddcutil issue #100, which reports switching from the USB-C input to the HDMI1, HDMI2 and DP1 inputs with x0090, x0091 and x00d0. USB-C is not recorded: the report switches away from it, never to it.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1627345011>

### LG 27UN850-WY

- Test result contributed by ccrxf to the ddcutil LG wiki page. The name is kept exactly as the source writes it and is not merged with the separate 27UN850-W entry below, which comes from a different report.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

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

### LG 28MQ780-B

- USB-C is deliberately not recorded. amildahl tested 0xD1, while shinyquagsire23's lg_display_manager — which keys off the model string `28MQ780` — another commenter on that gist with a 28MQ780-B, and a blog write-up all use 0xD2. Two values for one input, on the same model, is exactly the case the catalog refuses to guess at, so the input is left out until somebody tests both.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2524850195>
- <https://gist.github.com/shinyquagsire23/f6b2adef253c6c3ab557a4852bf3abad>

### LG 29U531A

- Test result contributed by tinkererkzy to the ddcutil LG wiki page. A single HDMI port is reported, so the input is written bare.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 29UM69G

- One of the two entries whose values are not the usual 0x90/0x91/0xD0/0xD1 set: DisplayPort is 0xC0 and USB-C is 0xE0. The evidence for all three is jonpas' report, on the wiki and in ddcutil issue #100. The same author's `i3/monitor.sh` is listed as corroboration and no more: it defines the USB-C value but only ever invokes the HDMI and DisplayPort ones, so it does not show 0xE0 being sent. That file is not pinned to a commit, because no revision of it has been quoted anywhere this entry could cite.
- A single HDMI port is reported, so the input is written bare. A 2016 unit badged 29UM69G-B (EDID `GSM/0x76FA`) did not switch at all, which is recorded here as a warning rather than as a separate entry: there is no schema slot for a negative result. The last source below is that report.
- The standard Input Source feature is a known dead end on this model: the ddcutil LG wiki page lists the 29UM69G among displays where `setvcp 60` flashes the screen, sometimes opens the OSD and never sets the value, and `getvcp 60` always reads 0x00.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1695779788>
- <https://github.com/jonpas/dotfiles/blob/master/i3/monitor.sh> - the same three values; the script defines the USB-C value but only ever invokes the HDMI and DisplayPort ones. Not pinned to a commit: no revision of it has been quoted anywhere this entry could cite
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2259286211>
- <https://github.com/rockowitz/ddcutil/wiki/LG>

### LG 29WN600

- Test result contributed by iamSlightlyWind to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 32BL95U-W

- The one entry documented by the manufacturer rather than by a user. The service manual's cover says `MODEL : 32BL95U`; the marketed name with its suffix is established by cross-reference, from the manual's file name `32BL95UW.pdf` and LG's own product page, which lists `32BL95U-W.AUB` as the only 32BL95U variant. Thunderbolt is the first use of that connector kind in the catalog.
- The EDID product IDs the manual assigns — 0x7706 for HDMI, 0x7707 for DisplayPort, 0x7722 for Thunderbolt, model name `LG HDR 4K` — are not recorded as identities, because a model that is not write-enabled must record none; they would also collide with other LG models, which is discussed at the end of this page.
- The spec table's `User Model Name 32UL950` gets no entry of its own and no merge: an alias in a manual is not a report that a 32UL950 switches inputs.

Sources:

- <https://research.encompass.com/ZEN/sm/32BL95UW.pdf>
- <https://www.lg.com/us/support/product/lg-32BL95U-W>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1435477786>

### LG 32GP750-B

- Test result contributed by chris-pcguy to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 32GP83B

- Reported by Gilgame24 in ddcutil discussion 331.

Sources:

- <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-16088333>

### LG 32GP850-B

- Reported working by erenard, on the wiki and in ddcutil issue #100, on a unit manufactured in week 12 of 2023. This one is contested: a later report on the NVapi-write-value-to-monitor tracker describes a 32GP850-B that only flickers and does not change input, and an LG firmware list quoted in ddcutil issue #100 names "GP850" as explicitly unsupported. The row stays disabled and without an identity, so nothing is written to either kind of unit; whoever enables it will have to say which firmware they tested.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925342145>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/1#issuecomment-2110539219>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1444483801>

### LG 32GR93U-B

- gzougianos reports the "exact same numbers" as the NVapi-write-value-to-monitor README working over NVAPI. An m1ddc issue separately reports 208 and 144 working through BetterDisplay's LG alternate input on this model, which is the same two values in decimal.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2244679727>
- <https://github.com/waydabber/m1ddc/issues/48>

### LG 32QN650-B

- The wiki row (agspoon) gives no values; the two linked comments on ddcutil issue #100 quote x0090, x0091 and x00d0 for this model.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1626872046>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1634924423>

### LG 32U990A

- The weakest entry here. `lg.sh` in pyang2045/streamdeck-display-knob is described as "Manual control of the LG UltraFine 32U990A" and sends m1ddc `input-alt` 144, 208 and 210, but the script does not say whether it was run successfully. A single HDMI port is reported, so the input is written bare.

Sources:

- <https://github.com/pyang2045/streamdeck-display-knob>

### LG 32UD99-W

- The second entry with unusual values: DisplayPort is 0xE0 and USB-C is 0xC0, the reverse pairing of the 29UM69G. The two comments carry different halves of the result and the rows are cited accordingly: the first one confirms only the HDMI values, and DisplayPort and USB-C appear in the follow-up, where they were found by looping over candidate values with Lunar on macOS. Worth extra care when someone verifies this model on hardware.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636761366>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1636997772>

### LG 32UN880-B

- Reported independently on the NVapi-write-value-to-monitor tracker (switching from DisplayPort to USB-C, HDMI1, HDMI2 and DP), in ddcutil issue #100, and by way of the amdddc-windows tool in ddcutil issue #612, which uses 0x90.
- One quirk is worth repeating to anyone who verifies it: after switching to USB-C, the monitor stopped accepting commands sent from the DisplayPort and HDMI sides.

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2835626567>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/8>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2142184960>
- <https://github.com/rockowitz/ddcutil/issues/612>

### LG 32UP83AK-W

- The reporter says 0xD0 "works without any issues" and that they switch to HDMI and DisplayPort, but never writes the HDMI value down, so only DisplayPort is recorded.

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/2#issuecomment-3221830027>
- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/3#issuecomment-3224103812>

### LG 34GS95QE

- Vib0 reports the three values as "Tested". A BetterDisplay discussion separately shows a 34GS95QE-B working through the LG alternate input, but the values there appear only in a screenshot, so that report corroborates the mechanism rather than the numbers.

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5#issuecomment-2812426302>
- <https://github.com/waydabber/BetterDisplay/discussions/4246>

### LG 34U650A-B

- The report gives the values in decimal — ddcAlt 144, 145 and 208 "work correctly". The USB-C value is the one entry in the catalog that does not fit in a byte: 210 got no response, and 465 did.

Sources:

- <https://github.com/waydabber/BetterDisplay/issues/4853>
- <https://github.com/waydabber/BetterDisplay/discussions/4883>

### LG 34UC98-W

- Driven by the standard Input Source feature; ddcctl writes VCP 0x60. The Thunderbolt input answers to 15, the value the specification gives DisplayPort-1; the DisplayPort connector itself could not be selected with any of the many values between 0 and 18 the reporter tried. The second Thunderbolt port is an output for daisy-chaining.

Sources:

- <https://github.com/kfix/ddcctl/issues/67>

### LG 34UM88C-P

- The one LG entry driven by the standard Input Source feature rather than the side channel; nobody has reported this model on VCP 0xF4. The capabilities string declares 0x11, 0x12, 0x0F and 0x10, VCP 0x60 read 0x00 (which ddcutil labels an invalid value) while DisplayPort was displayed, and the reporter says the read is wrong but the switch works.
- EDID, as text only: manufacturer `GSM`, model string `LG ULTRAWIDE`, product code 23266 (0x5AE2). Not an identity.

Sources:

- <https://github.com/rockowitz/ddcutil/discussions/331#discussioncomment-7152259>

### LG 34WN650-W

- Weaker evidence: the report quotes erenard's three values, says "Works" with ddcutil 2.1.2, and does not break the result down per input.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1925579750>

### LG 34WN750-B

- Test result contributed by permezel to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 34WN780

- Weaker evidence: the report quotes the three NVAPI values for this model and says "Worked well", without saying which it tried.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-2453464317>

### LG 34WN80C-B

- Two evidence strengths in one entry. USB-C is explicit: the linked comment names the input and the value. The other three come from the value table in the referenced gist, which the wiki contributor reports as working without quoting the numbers.
- Neither report went through ddcutil. The monitor was driven by the Python USB-HID script in that gist, which is why the evidence strings say so: the mechanism is the same LG side channel, but nothing here says the values survive a `ddcutil setvcp`. The gist is cited unpinned, because no revision of it is quoted in the report.
- The standard Input Source feature is a known dead end on this model: the ddcutil LG wiki page lists the 34WN80C-B among displays where `setvcp 60` flashes the screen, sometimes opens the OSD and never sets the value, and `getvcp 60` always reads 0x00.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>
- <https://github.com/rockowitz/ddcutil/issues/100#issuecomment-1542859752>
- <https://gist.github.com/shinyquagsire23/f6b2adef253c6c3ab557a4852bf3abad>
- <https://github.com/rockowitz/ddcutil/wiki/LG>

### LG 38BR85QC

- Recorded, disabled, and unable to match anything: no EDID fingerprint for it has been collected. The values come from the ddcutil wiki and nobody involved in this project has a unit to test them on. They are written down because they are useful to a contributor who does — see the contributor guide for what turns a row like this into an enabled one.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 40U990A-W

- Test result contributed by titou10titou10 to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 40WP95C-W

- Test result contributed by fblaese to the ddcutil LG wiki page.

Sources:

- <https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

### LG 40WP95X

- Only USB-C is reported: "This is how switching to USB-C input works".

Sources:

- <https://github.com/kaleb422/NVapi-write-value-to-monitor/issues/5>

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

### Monoprice Dark Matter 40776

- 40776 is the Monoprice product number. The reporter switched between HDMI 2 and DisplayPort 1, the two inputs with a computer attached; HDMI 1 and DisplayPort 2 are listed as working values without being shown switched, hence the two grades. The read-back equals the value written. The capabilities string declares 0x11, 0x12, 0x0F and 0x10, none of which work.
- The monitor accepts `setvcp` and answers `getvcp` only on the input currently displayed. Recorded on the ddcutil Monoprice wiki page from this report.
- EDID, as text only: manufacturer `LHC`, model string `34CHR`, product code 52.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/157>
- <https://github.com/rockowitz/ddcutil/wiki/Monoprice>

### MSI MPG 321URX QD-OLED

- The post writes the model as MSI MPG 321URX; both commands are bound to hotkeys and used daily. The built-in KVM follows the video input. The two HDMI inputs were not reported.

Sources:

- <https://github.com/alexpls/alexplescan.com/blob/master/content/posts/2025/kvm/index.md>

### Philips 328P6VUBREB/11

- Each write was followed by a read that showed the new input. Quirk that matters to monmux: the monitor answers VCP version 3.0 on feature 0xDF while its capabilities string says 2.2, and ddcutil then treats 0x60 as a table feature and rejects `setvcp 0x60 0x12` with "Invalid hex value" until `--mccs 2.2` or a user-defined feature file forces 2.2. monmux passes neither, so on a ddcutil that still behaves this way the command fails before anything is written. The ddcutil Philips wiki page carries the same workaround.
- EDID, as text only: manufacturer `PHL`, model string `PHL 328P6VU`, product code 2343 (0x0927).

Sources:

- <https://github.com/rockowitz/ddcutil/issues/162>
- <https://github.com/rockowitz/ddcutil/wiki/Phillips>

### Philips 49M2C8900L

- An Evnia model. The README says switching to USB-C (21) and HDMI 1 (17) and back was live-tested, and that the monitor keeps answering DDC on an input that is not displayed. USB-C is 0x15, not the 0x1B that Dell-derived guides assume.
- The README is the only report and reads as generated text; treat it as one uncorroborated claim.

Sources:

- <https://github.com/marclloydjolly/evnia-mac>

### Samsung C49RG94SSU

- A CRG9; the report gives no regional suffix. DP-2 cannot be selected from any input: philipp1992 tried every value from 0 to 30 and pwhelan, on a C49RG9 in the same thread, brute-forced every 16-bit value. Switching to HDMI works but the value is not stated. 0x09 is Tuner-1 in the MCCS specification.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/226>

### Samsung G95NC

- The Odyssey Neo G9 57 inch; the thread names model codes LS57CG952NUXEN and LS57CG952NNXZA. jimmy-tr33 in the same thread confirms HDMI-1 and HDMI-2 switch with 0x05 and 0x06.
- Conflict, recorded and not resolved: mwd102, on firmware 1009.2 and working from the Samsung Windows application, lists HDMI-1 as 0x11 and HDMI-2 as 0x12. HDMI-3 reads back 0x01 but writing 0x01 does not switch, and the ddc-mqtt project configures 7 for it; HDMI-3 is left out. PBP and PIP use vendor codes 0xE2 and 0xE3, which ema987 found absent on firmware 1007.0 and present on 1009.2; out of scope either way.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/397>
- <https://github.com/mwd102/g95nc-ddc>
- <https://github.com/moimart/ddc-mqtt>

### Samsung LC49G95T

- The first record of the `vcp-input-source` mechanism, and the first entry that is not an LG. It is disabled and carries no identity like every other unverified entry, so that backend path has never reached a monitor: enabling this model would be the first hardware run of the mechanism, and belongs in the [testing.md](testing.md) checklist rather than in a routine catalog flip.
- The values the monitor reports back are not the values that select an input. After a switch, reading VCP 0x60 gives 0x03 for DP1, 0x04 for DP2 and 0x01 for HDMI, and writing those back does not switch. That is one reason monmux never confirms a switch by reading a monitor.
- DDC/CI answers only on the HDMI input; the DisplayPort inputs do not expose slave address 0x37 at all, so the switch has to be sent from HDMI.
- Switching to an input with no signal wedges the monitor's DDC engine until a link reset or a trip through the OSD. Never switch blind: whoever verifies this model needs the target input already connected.
- The capabilities string declares Input Source values the monitor does not have, so it is not a source of values for this model.
- EDID, as text only: manufacturer `SAM`, model name `LC49G95T`. No product code has been published, which is the other reason the entry records no identity.
- Corroboration from macOS: in display-switch issue 43 kcorey switches an LC49G95TSSUXEN between `Hdmi1` (0x11) and `DisplayPort1` (0x0F) from a Mac connected through an HDMI adapter, and in BetterDisplay issue 2476 an Odyssey G9 on an M2 Air over DisplayPort does nothing, which is the DisplayPort silence above.

Sources:

- <https://github.com/DimpiM/monitor-switch/blob/main/docs/hardware-findings.md>
- <https://github.com/DimpiM/monitor-switch/blob/main/service/profiles/samsung-lc49g95t.yaml>
- <https://github.com/haimgel/display-switch/issues/43>
- <https://github.com/waydabber/BetterDisplay/issues/2476>

### Samsung LS27A800U

- The name is the EDID model string; the report gives no regional suffix. VCP 0x60 reads 0x06 while HDMI is displayed. USB-C cannot be selected by any value, per hostmit in BetterDisplay issue 4221, who also reports DisplayPort switching in BetterDisplay without stating the value.
- EDID, as text only: manufacturer `SAM`, model string `LS27A800U`, product code 29092 (0x71A4) from the report and 0x71A1 from the BetterDisplay export; two units, two codes.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/395>
- <https://github.com/waydabber/BetterDisplay/issues/4221>

### Samsung LS32A70

- The name is the EDID model string; the report gives no regional suffix. The capabilities string declares VGA (0x01) and DVI (0x03), which the monitor does not have.
- Conflict, recorded and not resolved: the ddcutil Samsung wiki page, written from this report, says DisplayPort needs 0x06 to set and reads back 0x0F, which is the reverse of what the report itself shows. The value here follows the report, which quotes the commands.
- EDID, as text only: manufacturer `SAM`, model string `LS32A70`.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/343>
- <https://github.com/rockowitz/ddcutil/wiki/Samsung>

### Samsung LU28R55

- The name is as the report writes it; the full model code (LU28R550UQ with a regional suffix) is not given. The two HDMI inputs cannot be selected: every value from 0 to 36 and 99 flickers the screen and returns to the source.

Sources:

- <https://github.com/kfix/ddcctl/issues/127>

### Samsung U28H750UQNXZA

- The report says input switching with ddcutil works from a laptop on HDMI-1 and that the real values are 5, 6 and 15 where the capabilities string claims 0x11, 0x12 and 0x0F; it does not say which inputs were switched to. On HDMI-2 the monitor answered no DDC at all in that setup. The larger U32H750 shows the same three values on the ddcutil Samsung wiki page, as reads.

Sources:

- <https://github.com/rockowitz/ddcutil/issues/185>
- <https://github.com/rockowitz/ddcutil/wiki/Samsung>

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
