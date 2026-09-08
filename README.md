# monmux

Switch supported monitors between video inputs, from the command line, on Linux and macOS.

`monmux` is fail-closed by design. It writes to a monitor only when that monitor is positively identified, from its EDID, as a
model in the built-in supported-monitor catalog, **and** the requested input is explicitly enabled for that model with recorded
evidence that the value was tested on real hardware. Everything else — an unknown monitor, two candidates, an input nobody has
verified — is refused without a write, and every refusal says so in as many words. The one documented exception is the
`--unsafe-model` override below, which has to be typed on the command line it applies to.

It wraps `ddcutil` on Linux and `m1ddc` on macOS; there is no native I²C or IOKit backend. One monitor model is verified so far.

## Supported monitors

One model is verified on hardware: the LG 38WR85QC-W, on `dp` and `usb-c`. Another 70 models from twelve vendors are recorded
from public reports, every one of them disabled and without an EDID fingerprint, so monmux can never match them, and will not
write to them unless you take the identification off yourself with [`--unsafe-model`](#the---unsafe-model-override). They are
written down for the contributor who owns one — enabling a row means testing it on a real unit and recording the evidence, see
[docs/adding-a-monitor.md](docs/adding-a-monitor.md). The full list, with the evidence and the conflicts behind
every value, is in [docs/compatibility.md](docs/compatibility.md).

Two mechanisms are implemented: `lg-alt-input`, the LG side channel, and `vcp-input-source`, the standard `VCP 0x60` Input
Source feature. Only the first has ever reached a monitor; the second has one recorded model, disabled, and enabling it would
be the first hardware run of that path.

## Install

Download a binary from the [releases page](https://github.com/leinardi/monmux/releases), or build from source:

```sh
go install github.com/leinardi/monmux/cmd/monmux@latest
```

A Homebrew formula will arrive together with the release pipeline.

### Prerequisites

**Linux** — `ddcutil` 2.2 or newer (2.2.5 is the version monmux was verified against; the `--i2c-source-addr` option it needs
does not exist in older releases), and permission to open the monitor's `/dev/i2c-N` for reading and writing. On most
distributions that means installing `ddcutil`, loading the `i2c-dev` module, and adding yourself to the `i2c` group or
installing the udev rule that ships with `ddcutil`. Log in again afterwards.

**macOS** — [`m1ddc`](https://github.com/waydabber/m1ddc), which requires Apple Silicon:

```sh
brew install m1ddc
```

`monmux doctor` reports which of these are missing.

## Usage

```sh
monmux info                 # what is attached, and what monmux would do with it
monmux info --json          # the same, for scripts
monmux doctor               # can monmux reach the monitors at all?

monmux switch usb-c --dry-run   # print the exact command, run nothing
monmux switch usb-c             # send it
monmux switch dp --serial ABC123456789   # pick one of two identical monitors

monmux catalog list             # every model in this binary's catalog
monmux catalog list aoc         # filter by vendor, model, or vendor/name
monmux catalog list --verbose   # with the value and evidence grade of every input
monmux catalog show AOC/Q27P1B  # one entry in full, evidence included
```

`monmux catalog` reads nothing outside the binary: no backend, no configuration file, no monitor. It is the offline view of
[docs/compatibility.md](docs/compatibility.md), for the build you are actually running.

Serial numbers, macOS display UUIDs and raw EDID hex are redacted in every output by default, so what monmux prints is safe to
paste into a bug report. `--show-serial` prints them verbatim, and is the only way to see them.

A successful switch reports exactly what happened:

```text
Input-switch command sent (USB-C, 0xD1) to LG 38WR85QC-W via ddcutil. Switch not independently confirmed.
```

The second sentence is not hedging. monmux never reads a monitor back to confirm a switch — the LG side channel has no reliable
read-back, and a read of `VCP 0x60` is no better — so "the command was sent" is the strongest true statement available; monmux
never claims a monitor switched.

### The `--unsafe-model` override

Seventy of the seventy-one entries are recorded from somebody else's report and are not write-enabled, so `monmux switch`
refuses them. `--unsafe-model` is the deliberate way past that, for the person who owns the monitor and is trying to produce
the evidence that would enable it:

```sh
monmux switch hdmi --unsafe-model AOC/Q27P1B --dry-run   # always start here
monmux switch hdmi --unsafe-model AOC/Q27P1B
```

It treats the attached display as the catalog entry you name, instead of identifying it from its EDID, and it skips the
write-enabled gate. It weakens those two things and nothing else: the value still comes from the compiled-in catalog, so there
is still no way to send a VCP code or a value of your own; an input the named entry does not record is still refused; monmux
still refuses when more than one attached display is writable, so pin with `--serial` rather than letting it guess; the
identity is still re-verified immediately before the write; and no configuration key can arm the flag — it is per invocation,
on purpose. A warning naming the display, the assumed model and the value goes to stderr before anything is written, and the
result line says the identification was bypassed.

A wrong value can leave a monitor on an input with no signal, so have the monitor's OSD within reach, and read
[docs/adding-a-monitor.md](docs/adding-a-monitor.md) before you record what happened.

## Exit codes

| Code | Meaning                                                                                                                        |
| ---- | ------------------------------------------------------------------------------------------------------------------------------ |
| `0`  | The input-switch command was sent, or a read-only command succeeded.                                                           |
| `1`  | The external tool ran and failed, or the request could not be made at all. The message says whether a write may have happened. |
| `2`  | monmux refused. No DDC write was performed.                                                                                    |

Only exit `2` carries the promise that nothing was written.

## Documentation

- [Architecture](docs/architecture.md) — how a switch is decided, and what stops it going wrong.
- [Compatibility](docs/compatibility.md) — the catalog, with the evidence for every value.
- [Configuration](docs/configuration.md) — the configuration file, the flags, and which wins.
- [Troubleshooting](docs/troubleshooting.md) — every refusal reason, and what to do about it.
- [Backends](docs/backends.md) — `ddcutil` and `m1ddc` specifics: versions, permissions, quirks.
- [Adding a monitor](docs/adding-a-monitor.md) — the procedure for enabling a model or an input.
- [Security](docs/security.md) — threat model, mitigations, trust boundaries, and what monmux never does.
- [Testing](docs/testing.md) — why no test ever runs an external binary, and the human hardware checklist.
- [Release](docs/release.md) — the release pipeline, as a design. Not implemented yet.
- [Requirements](docs/requirements.md) — the source document this project was built from.
- [Contributing](CONTRIBUTING.md) — prerequisites, workflow, and the catalog evidence rule.
- [AGENTS.md](AGENTS.md) — repository conventions, including the rule that no AI agent may write to a monitor.

## Licence

Apache License 2.0 — see [LICENSE](LICENSE).
