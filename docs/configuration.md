# Configuration

monmux works with no configuration at all. The file exists for two things: pinning to one physical monitor, and pointing at a
tool that is not on `PATH`.

Nothing in it can widen what monmux is willing to do. Which monitors and which inputs are write-enabled is decided by the
catalog, which is generated into the binary from a file read only at development time, and the backend is chosen by the operating
system with no override. A configuration file can narrow a request or name a different binary, and that is all.

## The file

```text
$XDG_CONFIG_HOME/monmux/config.yaml
```

or, when `XDG_CONFIG_HOME` is not set:

```text
~/.config/monmux/config.yaml
```

The location is the same on Linux and macOS on purpose: somebody with both machines should not have to remember two.

```yaml
serial: ABC123456789
ddcutil_path: /usr/local/bin/ddcutil
m1ddc_path: /opt/homebrew/bin/m1ddc
```

Every key is optional, and a missing file is not an error. **An unknown key is an error.** A typo that was silently ignored
would leave you believing a pin is in effect when it is not, and for a serial pin that means writing to a monitor you meant to
exclude.

## `serial`

Pins every switch to one physical unit. It is an extra restriction on top of the catalog match, never a replacement for it: the
monitor must still be a supported model with the requested input enabled (requirement 9.10).

The semantics are deliberately narrow:

- It matches the **alphanumeric serial string only** — EDID descriptor `0xFF` on Linux, the `AN Serial` field on macOS. The
  numeric serial is never used for pinning: it is not what is printed on the label on the back of the monitor, and two
  different units can carry the same one.
- The comparison is **exact and case-sensitive**, after trimming whitespace from both the configured value and the display's.
- A display that exposes **no alphanumeric serial at all** cannot be pinned. Asking to pin it is refused with `serial-mismatch`
  and the detail `display exposes no alphanumeric serial`, naming the display, so it is clear the pin can never work for that
  unit rather than looking like a typo.

Find the value with `monmux info --show-serial`.

## `ddcutil_path` and `m1ddc_path`

An absolute path to the backend's tool, for a system where it is not on `PATH` or where a specific build must be used. A
relative path is refused with `backend-not-ready`.

The same trust checks apply to a configured path as to one found on `PATH`: the resolved file, with symlinks followed, must be a
regular file, and neither it nor its containing directory may be writable by anyone but its owner. `monmux doctor` prints the
resolved path and its SHA-256.

This is a trust boundary, not a mitigated threat — see [security.md](security.md).

## Flags

| Flag                 | Command                                                     | Effect                                                                        |
| -------------------- | ----------------------------------------------------------- | ----------------------------------------------------------------------------- |
| `--serial <s>`       | `switch`                                                    | Pin this invocation to one unit, with the semantics above.                    |
| `--dry-run`          | `switch`                                                    | Print the exact command that would run, and run nothing.                      |
| `--unsafe-model <m>` | `switch`                                                    | Treat the display as this catalog entry instead of identifying it. See below. |
| `--verbose`          | `catalog list`                                              | Add the value and the evidence grade of every recorded input.                 |
| `--json`             | `switch`, `info`, `catalog list`, `catalog show`, `version` | Print the report as JSON. See [json.md](json.md).                             |
| `--show-serial`      | all                                                         | Print serials, display UUIDs and raw EDID hex instead of redacting them.      |

## `--unsafe-model`, and why it has no key

`--unsafe-model VENDOR/MODEL` treats the attached display as that catalog entry rather than identifying it from its EDID, and
skips the write-enabled gate. It is the one flag that makes monmux write where it would otherwise refuse, and it is
**flag-only, every invocation, on purpose**: there is no configuration key for it and there will not be one.

The reason is the one this whole file opens with. A standing preference is the wrong shape for a weakening: a key in a file
would arm the override for every later `monmux switch`, including the ones you did not think about, and a typo in that key
would be an error you never see. Typing the flag is the point at which you decide to bypass identification, and it applies to
that one command.

What it does not weaken: the value still comes from the compiled-in catalog, so there is still no way to pass a VCP code or a
value; an input the named entry does not record is refused; more than one writable display is refused rather than chosen
between, so pin with `--serial`; and the identity is still re-verified immediately before the write. A name that is not a
catalog entry is a usage error, exit code 1 — `monmux catalog list` prints the names it accepts.

See [security.md](security.md) for what the override moves across the trust boundary, and
[adding-a-monitor.md](adding-a-monitor.md) for what it is for.

## Precedence

`--serial` overrides `serial:` in the file. The file is a standing preference; the flag is what you are asking for right now.
There is no flag for the tool paths: a path that changes per invocation is a scripting problem, not a configuration one, and
there is no key for `--unsafe-model`, for the opposite reason — see above.

Nothing else has a precedence question. The backend comes from the operating system, and the catalog comes from the binary.
