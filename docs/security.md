# Security

monmux writes to hardware. The failure that matters is not a crash: it is sending a value a monitor was never verified to accept,
to a monitor that is not the one monmux thought it was addressing. Every design decision below exists for that reason.

There is prior art for the consequences. A monitor that stops displaying anything after an unexpected manufacturer-specific
write is not recoverable by software, which is why the default in this project is always to refuse.

## Threat model

### Wrong values

**Risk.** A value that means "switch to USB-C" on one model means something else on another, and manufacturer-specific
registers are not a documented, portable interface.

**Mitigation.** A value can only reach a monitor if it is written in the catalog next to evidence that someone tested it on that
exact model. The catalog is `internal/catalog/models.yaml`, rendered by `make go-generate` into the committed
`internal/catalog/models_gen.go` that the binary compiles. That rendering is a development-time step, so the shipped binary has
no catalog parser and reads no catalog file at run time; both files appear in the same diff, and a test fails the build if they
disagree, so the bytes a reviewer approved are the bytes that ship. The generator refuses a file that leaves a product code or
a value out rather than defaulting it to zero, refuses a key it does not know, and refuses a second YAML document, so no value
reaches the generated catalog that the file did not spell out. `catalog.Operation` has unexported fields and a package-private constructor, so no value that is
not in the catalog can exist as an operation at all; the zero value is invalid and every backend rejects it. The CLI accepts
symbolic inputs only — a connector kind such as `dp` or `hdmi`, optionally numbered such as `hdmi2` — and has no flag that takes
a VCP code or a raw value (requirement 9.5).
An input nobody has tested on a model is not enabled for it, and asking for it is refused.

### Stale identity, and a bus that moved

**Risk.** Displays are enumerated at one moment and written at another. In between, a monitor can be unplugged, another plugged
in, or an I²C bus rebound to a different connector — so the display monmux identified is not the one that receives the write.

**Mitigation.** The identity is re-verified immediately before the write. On Linux the connector's EDID is re-read and compared
byte for byte with what enumeration saw, and the `ddc` link is re-resolved to the same bus; on macOS the display list is
re-read and the UUID must still carry the same identity. Any difference is `identity-changed`, and nothing is sent. The Linux
write then selects the display by its full 256-character EDID rather than by a bus number, so `ddcutil` itself refuses if the
monitor on that bus changed in the remaining instant.

### Display reordering

**Risk.** "Display 2" is a position in a list, and the list changes when a monitor is plugged in or wakes up. Addressing a
monitor by its position means eventually addressing a different monitor.

**Mitigation.** A list position is never used as a selector. On Linux the handle is the DRM connector name; on macOS it is the
display's system UUID, and monmux checks that what it was given is actually UUID-shaped before using it — `m1ddc` accepts a list
index in the same argument position, so a non-UUID value there would silently mean "whichever display is second". A display with
no usable UUID is reported with status `no-uuid` and is never written to. Enumeration order is never proof of identity
(requirement 9.9).

### Forged commands

**Risk.** If any part of monmux could hand a backend an arbitrary program and argument list, every other check would be
decoration.

**Mitigation.** `Backend.Plan` returns a `Command`; no method on the `Backend` interface accepts one, and a reflection test
enforces that no method ever will. `Execute` takes the `catalog.Operation` and rebuilds the invocation through the same private
planner `Plan` used, so what `--dry-run` shows is what a real run performs, and neither can be substituted for something else.
The tool path is resolved once during preflight and executed directly, with no `PATH` lookup at execution time.

### Configuration as an injection vector

**Risk.** A configuration file is user data that a program acts on. If it could enable a monitor, enable an input, choose a
mechanism or pick a backend, then editing a YAML file would be enough to write anything anywhere.

**Mitigation.** It cannot do any of those things. The file has three keys — a serial to pin to, and the two tool paths. The
catalog is compiled in, and `models.yaml` is a build input that a running monmux never opens; the backend is chosen by the
operating system with no override. An unknown key is an error rather than
something ignored, so a typo cannot silently disable a pin the user believes is protecting them.

### Serial and UUID leakage

**Risk.** Diagnostic output ends up in bug reports and pastebins, and a monitor's serial number and a Mac's display UUID are
identifying data.

**Mitigation.** Every output redacts them by default: `info`, `info --json`, `doctor`, refusal messages, and the command printed
by `--dry-run`, whose EDID hex identifies a physical unit just as precisely as a serial does. `--show-serial` prints them
verbatim, and is the only way to see them. Test fixtures carry synthetic identifiers only, and a test fails the build if any
other value appears in one.

## Trust boundaries

Two things monmux does **not** defend against, stated plainly rather than listed as mitigated:

**The tool it runs.** monmux resolves `ddcutil` or `m1ddc` from `PATH`, or from the absolute path in the configuration, and
checks that the resolved file is a regular file that neither it nor its directory allows anyone but its owner to rewrite. That
is all it can check. **monmux cannot tell a genuine `ddcutil` from a maliciously replaced one with correct ownership and
permissions.** Anyone who can write to that binary — or to a directory on your `PATH` ahead of it — can make monmux run whatever
they like. `monmux doctor` prints the resolved path and its SHA-256 so the binary can be identified; comparing it against a
known-good value is the user's job, not monmux's.

**The kernel and the tool's own behaviour.** monmux does not open `/dev/i2c-*` and does not speak DDC. What the external tool
puts on the wire once it is invoked, and what a driver does with it, is outside monmux's control.

## What monmux never does

- Never writes to a monitor it has not positively identified as a catalog model, or for an input that model does not enable.
- Never exposes a raw VCP code or value to the user (9.5).
- Never probes: no trying values to see what happens, no scanning VCP codes, no brute-forcing source addresses. All discovery
  is read-only (9.6).
- Never falls back to standard Input Source `VCP 0x60` because a monitor happens to read it (9.7).
- Never decides anything from `get input` or `get input-alt` (9.8).
- Never retries a write, and never loops until the picture changes (9.11).
- Never claims a monitor switched. It reports that a command was sent, and says in the same sentence that the switch was not
  independently confirmed (9.12).

## The rule for contributors and agents

**No AI agent — an assistant in an editor, a coding agent, a subagent, a review bot, any tool-driven automation — may run a
command that writes to a monitor.** Only a human runs a writing command, by hand, deliberately.

For agents this means: no `ddcutil setvcp`, no `ddcutil` invocation with `--i2c-source-addr`, no `m1ddc … set …`, no
`monmux switch` without `--dry-run`, no `i2cset`, no `i2ctransfer`, no writes to `/dev/i2c-*`, and no test or script that
executes the real `ddcutil` or `m1ddc` binary at all. Read-only work is fine: reading `/sys/class/drm`, `ddcutil --version`,
`ddcutil detect`, `monmux info`, `monmux doctor`, `monmux switch … --dry-run`.

The rule is enforced in code as well as written down. The real runner refuses to start any process while a test binary is
running or when `MONMUX_NO_EXEC=1` is set, and a repository test fails the build if any test file imports `os/exec` or so much
as names the real runner's constructor. Hardware validation for a new model is done by a human, and the evidence is recorded in
the catalog in the same commit.

## Reporting a problem

Please report security issues privately through GitHub's [security advisories](https://github.com/leinardi/monmux/security/advisories/new)
rather than in a public issue. A report that includes `monmux info` output is welcome; run it without `--show-serial` unless the
serial is the point, since the default output is already redacted.
