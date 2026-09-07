# Configuration

monmux works with no configuration at all. The file exists for two things: pinning to one physical monitor, and pointing at a
tool that is not on `PATH`.

Nothing in it can widen what monmux is willing to do. Which monitors and which inputs are write-enabled is decided by the
catalog, which is Go source compiled into the binary, and the backend is chosen by the operating system with no override. A
configuration file can narrow a request or name a different binary, and that is all.

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

| Flag            | Command  | Effect                                                                   |
| --------------- | -------- | ------------------------------------------------------------------------ |
| `--serial <s>`  | `switch` | Pin this invocation to one unit, with the semantics above.               |
| `--dry-run`     | `switch` | Print the exact command that would run, and run nothing.                 |
| `--json`        | `info`   | Print the report as JSON.                                                |
| `--show-serial` | all      | Print serials, display UUIDs and raw EDID hex instead of redacting them. |

## Precedence

`--serial` overrides `serial:` in the file. The file is a standing preference; the flag is what you are asking for right now.
There is no flag for the tool paths: a path that changes per invocation is a scripting problem, not a configuration one.

Nothing else has a precedence question. The backend comes from the operating system, and the catalog comes from the binary.
