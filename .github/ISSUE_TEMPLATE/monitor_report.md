---
name: Monitor report
about: Report what a monitor did, so its catalog entry can be recorded or enabled
title: "Monitor report: <VENDOR> <MODEL>"
labels: ["catalog"]
assignees: []
---
<!--
This is how a model gets into the catalog, and how a recorded one becomes
write-enabled. Read docs/adding-a-monitor.md first:
https://github.com/leinardi/monmux/blob/main/docs/adding-a-monitor.md

Two things to know before you start:

- A value only becomes `grade: verified` — the grade a write-enabled model needs
  on every input — when the person reporting it ran it on that unit themselves.
  A value copied from another model, another firmware or another issue thread is
  not evidence, however plausible it looks.
- Start every attempt with `--dry-run`, and have the monitor's OSD within reach:
  a wrong value can leave a monitor on an input with no signal.

monmux redacts serial numbers, serial strings, raw EDID hex and macOS display
UUIDs by default. Do not paste `--show-serial` output.
-->

## The monitor

* Vendor and model, exactly as printed on the unit: `?`
* Firmware version, if the OSD shows one: `?`
* Is it already in `docs/compatibility.md`? If so, under which entry? `?`

## Environment

* `monmux version`: `?`
* Operating system and version: `?`
* Linux: `ddcutil --version`: `?` — macOS: `brew info m1ddc` (the installed version): `?`

## What monmux sees

```text
<monmux info --json output; it redacts by default>
```

## What you ran, per input

<!--
One block per input you tried. The `Command:` line is what `--dry-run` printed;
paste it as it was printed, redaction included.
-->

### Input: `?`

* Catalog entry assumed with `--unsafe-model`: `?`
* Dry-run `Command:` line:

```text
```

* What the monitor did: <!-- switched / did nothing / went dark / switched to a different input -->
* Did it come back, and how?
* Date of the test: `?`

## Coverage

* [ ] I ran this on Linux
* [ ] I ran this on macOS
* [ ] I own this monitor and ran every value above on it myself
* [ ] I have read [docs/adding-a-monitor.md](https://github.com/leinardi/monmux/blob/main/docs/adding-a-monitor.md)

## Anything else

<!-- Negative results are worth as much as positive ones: an input that did nothing, a value that
worked once and not again, a firmware that changed the behavior. -->

*
