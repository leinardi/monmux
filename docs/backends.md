# Backends

monmux does not speak DDC. It runs an external tool, one per operating system, chosen by `runtime.GOOS` with no override:
`ddcutil` on Linux, `m1ddc` on macOS. Anywhere else, monmux refuses with `backend-unavailable` rather than pretending.

Both backends implement the same interface and share the same rules: the tool path is resolved once and executed directly, a
command is built only by the backend's own planner, the display's identity is re-verified immediately before a write, and the
tool is run exactly once with no retries.

## What both backends check about the tool

The tool path is resolved during preflight, either from `PATH` or from the absolute path in the configuration
(`ddcutil_path`, `m1ddc_path`). The resolved file, with symlinks followed, must be:

- a regular file, and
- not writable by group or other, and
- in a directory that is not writable by group or other.

Anything else is `backend-not-ready`. `monmux doctor` prints the resolved path and its SHA-256. This is a trust boundary and not
a mitigated threat: monmux cannot tell a genuine tool from a correctly-permissioned replacement — see [security.md](security.md).

## Linux: `ddcutil`

### ddcutil requirements

`ddcutil` **2.2 or newer**. That floor is concrete rather than cautious: 2.2.5 is the version monmux was verified against, and
`--i2c-source-addr`, which the LG mechanism needs, does not exist in older releases. Preflight reads `ddcutil --version` and
refuses anything older.

Preflight then reads `ddcutil --help` and requires it to list `--edid`, `--i2c-source-addr` and `--noverify`. It is a capability
probe that costs nothing and touches no monitor: a build without those options fails here rather than in the middle of a write.

### ddcutil enumeration

Displays come from `/sys/class/drm`, not from `ddcutil`. For every connector whose `status` reads `connected`, monmux reads the
`edid` file and resolves the `ddc` symlink to an I²C bus:

| What it finds                   | Result                                                |
| ------------------------------- | ----------------------------------------------------- |
| EDID parses, `ddc` link present | `writable`, status `ok`, handle is the connector name |
| EDID missing or unparsable      | not writable, status `edid-unreadable`                |
| no `ddc` symlink                | not writable, status `no-ddc-channel`                 |

The raw EDID bytes and the bus number are kept inside the backend and never exported. Because enumeration reads only sysfs, `info`
lists your monitors even when `ddcutil` is missing or too old — the checks then say why nothing can be switched.

### ddcutil permissions

`Ready` asks whether `/dev/i2c-N` for that display's bus exists and can be opened for reading and writing by the current user; it
uses `access(2)` and opens nothing. When it cannot, the refusal is `target-not-ready` with the hint: add yourself to the `i2c`
group, or install a udev rule granting access, and log in again. The `i2c-dev` kernel module must be loaded.

This is a per-target check, not a per-tool one: `ddcutil` can be perfectly installed and still be unable to reach one particular
monitor.

### What ddcutil runs

```text
ddcutil --edid <256 hex characters> setvcp 0xF4 0xD1 --i2c-source-addr=0x50 --noverify
ddcutil --edid <256 hex characters> setvcp 0x60 0x0F --noverify
```

The first line is `lg-alt-input`, the second is `vcp-input-source`. They differ only in the VCP code and in the source address:
the standard feature goes to the ordinary DDC/CI address, so it carries no `--i2c-source-addr`.

`--edid` rather than `--bus` is a safety decision. The display is selected by its full EDID, so if the monitor on that bus is no
longer the one monmux identified, `ddcutil` itself refuses. `--noverify` is on both lines, and not only because the LG side
channel has no meaningful read-back: monmux never confirms a switch by reading a monitor at all, and a read of `0x60` is
particularly unreliable — the one model recorded for that mechanism answers with values that are not the ones that select an
input. monmux does not treat the absence of a read-back as a problem to work around.

The EDID hex identifies a physical unit, so it is masked in output unless `--show-serial` is given.

### ddcutil quirks

- The same monitor can report a different EDID product code depending on which input it is displaying. That is why a catalog
  model can carry several identities.
- `ddcutil detect` is not used for enumeration. It is read-only and useful for humans, but sysfs is the more direct answer and
  it needs no tool at all.

## macOS: `m1ddc`

### m1ddc requirements

[`m1ddc`](https://github.com/waydabber/m1ddc), which requires Apple Silicon:

```sh
brew install m1ddc
```

`m1ddc` has no `--version` and no options worth probing, so preflight uses the display list itself as the probe. It is
read-only, and it is the command every other code path depends on parsing — so a tool whose output has changed is caught before
a switch rather than during one.

A Mac with no external display attached passes preflight. `m1ddc` answers `No external display found, aborting` with a non-zero
exit status, which is not a broken tool: enumeration then reports no displays and monmux refuses with `no-displays`, which is
the accurate reason.

`doctor` warns when the binary is not under `/opt/homebrew` or `/usr/local`. It is a note, not a verdict — a local build is a
legitimate reason — but an unexpected location is the first thing worth knowing when the tool behaves strangely.

### m1ddc enumeration

From `m1ddc display list detailed`. The fields it prints map onto the same identity the Linux backend reads out of an EDID:
`Vendor`, `Model` and `Serial` are `CGDisplayVendorNumber`, `CGDisplayModelNumber` and `CGDisplaySerialNumber`, which are the
EDID manufacturer ID, product code and serial number. That is what lets one catalog serve both platforms.

The manufacturer string comes from the IORegistry and is sometimes missing; monmux uses it only when it is a plain three-letter
PNP identifier and decodes the packed vendor number otherwise.

The **system UUID is the only selector**. `m1ddc` also accepts a list index in the same argument position, and that index changes
when a monitor is plugged in or wakes up, so monmux checks that the handle it holds is actually UUID-shaped before using it. A
display with no usable UUID is reported with status `no-uuid` and is never written to.

Because the UUID identifies a physical unit, it is masked in output unless `--show-serial` is given.

### m1ddc permissions

There is nothing to check. macOS has no per-target permission model here, so `Ready` succeeds for every display; whether a
display can be written to at all was already decided during enumeration.

### What m1ddc runs

```text
m1ddc display <system UUID> set input-alt 209
m1ddc display <system UUID> set input 15
```

`input-alt` is `lg-alt-input`: the same LG side channel `ddcutil` reaches with `--i2c-source-addr=0x50`. `input` is
`vcp-input-source`, the standard `VCP 0x60` feature. Neither reads the monitor back. The value is decimal on both: the catalog
records `0xD1` and `m1ddc` is handed `209`.

`m1ddc` takes a 16-bit value and splits the SH/SL pair itself, so the one catalog value that does not fit in a byte needs no
special handling — `0x1D1` is handed over as `465`.

Before that runs, the display list is re-read and the UUID must still carry the same identity, or the switch is refused as
`identity-changed`.

### m1ddc quirks

- `get input` and `get input-alt` are never used. Current-input reads on the tested hardware were stale while switching still
  worked, and there is an upstream report of the same pattern. No decision monmux makes depends on what a monitor says it is
  displaying.
- `m1ddc`'s help lists USB-C as `210` among its common values. The unit in the catalog switches to USB-C on `0xD1` (209), which
  is what was tested on it. The catalog is the evidence; the help text is a general hint.
- Both the no-display message and the field labels are pinned in constants and covered by fixtures, so an upstream change to
  either fails a test rather than silently changing behaviour.

## Adding a backend

A third backend would implement `backend.Backend` in its own package under `internal/backend/`, build-tagged for its platform,
and be constructed from `internal/backend/select`. The rules it inherits are not optional: it must reject any mechanism it does
not implement, it must re-verify identity immediately before writing, its `Plan` must be the only place a command is built, and
it must run its tool exactly once.

Keep the parsing and the decisions in files with **no build tag**, as the macOS backend does. Parsing another program's output is
where a backend is most likely to be quietly wrong, and it is the part that has nothing to do with the operating system — so it
should be unit-tested everywhere, including on a machine that cannot run the tool at all.
