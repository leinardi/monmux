# Troubleshooting

Start with:

```sh
monmux doctor
monmux info
```

Both are read-only, and both redact serials by default, so their output is safe to paste into an issue.

## Exit codes

| Code | Meaning                                                                                        |
| ---- | ---------------------------------------------------------------------------------------------- |
| `0`  | The input-switch command was sent, or a read-only command succeeded.                           |
| `1`  | The tool ran and failed, or the request could not be made. Whether a write happened is stated. |
| `2`  | monmux refused. No DDC write was performed.                                                    |

Only exit `2` promises that nothing was written. A refusal always ends with `No DDC write was performed.`, and that sentence is
load-bearing: monmux never prints it about a run that reached the tool.

## Refusals

Every refusal names a reason. They are listed here in roughly the order the steps of a switch happen in.

### `backend-unavailable`

monmux has no backend for this operating system. It supports Linux (`ddcutil`) and macOS (`m1ddc`); anywhere else it refuses
rather than guessing.

### `backend-not-ready`

The external tool is missing, too old, missing an option monmux needs, untrusted, or it failed to start. The detail says which.

- **Not found on `PATH`** — install `ddcutil` (Linux) or `m1ddc` (macOS), or set `ddcutil_path` / `m1ddc_path` in the
  configuration to an absolute path. See [configuration.md](configuration.md).
- **Older than 2.2** — `ddcutil` needs to be at least 2.2; `--i2c-source-addr` does not exist before it. Upgrade.
- **Does not support `--edid`, `--i2c-source-addr`, `--noverify`** — an unusual build. Install a stock `ddcutil` 2.2+.
- **Group- or world-writable, or in a writable directory** — monmux will not run a binary that somebody else can rewrite. Fix
  the permissions, or point at a different copy.
- **Must be an absolute path** — a configured tool path has to be absolute.
- **Did not start** — the file exists but could not be executed. Check that it is executable and built for this machine.

Nothing was written in any of these cases; monmux never got as far as trying.

### `enumeration-failed`

The displays could not be listed at all. On Linux that means `/sys/class/drm` could not be read, which is unusual and points at
a container or a very restricted sandbox. On macOS it means `m1ddc display list detailed` failed or printed something monmux
does not recognise — if you upgraded `m1ddc` recently, its output format may have changed; please open an issue.

### `no-displays`

Nothing is attached, as far as the backend can tell. On macOS this is exactly what `m1ddc` reports on a Mac with no external
display. On Linux, check that the monitor is on and that `monmux doctor` lists a connected connector.

### `display-not-writable`

The display monmux would have written to cannot be written to. `monmux info` gives the reason in its status:

- `no-ddc-channel` — the connector exposes no `ddc` link. Common on ports wired through some docks and adapters, and on
  DisplayPort MST hubs. Try connecting the monitor directly.
- `edid-unreadable` — the EDID could not be read or did not parse. Often a cable or an adapter; try another one.
- `no-uuid` — macOS gave the display no system UUID, which is the only way monmux can address it there.

With `--unsafe-model` this is also what an unidentified display is refused with, naming each display and its status: the
catalog is not consulted on that path, so the answer that is actually true of the display is that monmux cannot reach it.

### `unknown-monitor`

The display was identified, and it is not in the catalog. This is monmux working as designed: it will not send a
manufacturer-specific value to a monitor nobody has tested it on.

`monmux info` shows the identity it read. If you own the monitor and want to add it, see
[adding-a-monitor.md](adding-a-monitor.md) — and note that "it is the same brand as one that works" is not evidence.

### `ambiguous-catalog`

Two catalog entries claim the same EDID identity. That is a bug in the catalog rather than anything about your setup; an
invariant test is supposed to make it impossible. Please open an issue with the `monmux info` output.

### `multiple-candidates`

More than one attached display could be the one you meant, and monmux will not pick for you. Pin the one you mean:

```sh
monmux info --show-serial          # read the alphanumeric serial
monmux switch usb-c --serial ABC123456789
```

Or set `serial:` in the configuration file if it is always the same monitor.

With `--unsafe-model`, this refusal counts the displays monmux could **write** to, matched or not, because identification is
exactly what you bypassed — so it can appear where a normal switch would have been unambiguous. The detail names them:

```text
Writable: card1-DP-1, card1-HDMI-A-1. An assumed model identifies nothing, so monmux will not choose between them; pin one with --serial.
```

Pinning is the answer there too. An override that picked the first display would send an unverified value to whichever monitor
happened to be listed first.

### `input-not-enabled`

The monitor is supported, but that input is not enabled for it — nobody has tested that value on that model. `monmux info` lists
the inputs that are enabled. `hdmi1` and `hdmi2` are not enabled for any model today.

This refusal is not a limitation to work around. Enabling an input means testing it on a real unit and recording the evidence.

With `--unsafe-model` the write-enabled gate is already bypassed, so this refusal means something narrower: the entry you named
does not **record** that input at all, and there is therefore no value to send. The detail lists what it does record, rather
than what it enables, which for an entry that is not write-enabled would be nothing:

```text
Requested usb-c on AOC Q27P1B; recorded inputs: dp, hdmi, dvi, vga.
```

`monmux catalog show AOC/Q27P1B` prints the same list with the evidence behind every value.

### `serial-mismatch`

A serial pin was requested and no attached display matches it.

- **Check the value** with `monmux info --show-serial`. The comparison is exact and case-sensitive, after trimming whitespace.
- **It matches the alphanumeric serial only** — the string in EDID descriptor `0xFF`, or macOS's `AN Serial`. The numeric serial
  is never used for pinning.
- **The detail `display exposes no alphanumeric serial`** means that unit reports no such serial at all, so it cannot be pinned.
  That is a property of the monitor, not a typo on your part; use a different way to disambiguate, or nothing at all if it is
  the only supported display.

### `target-not-ready`

The tool is fine and the display is recognised, but this particular monitor cannot be reached right now.

On Linux this is almost always permissions on `/dev/i2c-N`:

```sh
sudo modprobe i2c-dev
sudo usermod -aG i2c "$USER"      # then log out and back in
```

`ddcutil` also ships a udev rule that grants access; installing it is the alternative. `monmux doctor` says which display is
affected.

### `identity-changed`

Between listing the displays and writing, the display stopped being the one monmux identified — the EDID changed, the `ddc` link
moved to another bus, or on macOS the UUID no longer carries the same identity. Nothing was written.

Usually something was replugged, or a monitor woke up, mid-command. Run it again. If it happens repeatedly, a dock or an MST hub
is probably rebinding buses underneath you, and `monmux info` run twice in a row will show it.

### `invalid-operation`

The backend was handed an operation it will not perform: one that did not come from the catalog, or one whose mechanism this
backend does not implement. Seeing this from a normal command line is a bug — please open an issue.

## Exit code 1

### The tool ran and failed

```text
ddcutil failed: exit status 1. Whether the input-switch command reached the monitor is unknown.
```

This is the one outcome monmux cannot characterise. The tool started, so a write may have gone out; monmux will not claim
otherwise. Run `monmux doctor`, check what the monitor is displaying, and try again if appropriate. monmux never retries on its
own.

### The configuration file could not be used

```text
Error: ~/.config/monmux/config.yaml: config: the configuration file cannot be read: yaml: unmarshal errors:
  line 1: field colour not found in type config.Config
```

An unknown key is an error on purpose: a silently ignored typo would leave you believing a pin is in effect when it is not. Fix
the key — the valid ones are `serial`, `ddcutil_path` and `m1ddc_path`.

### An unknown input

```text
Error: catalog: unknown input: "scart": input: unknown connector kind: "scart" (a connector kind - dp, hdmi, usb-c, dvi, vga,
thunderbolt - optionally followed by a port number, e.g. hdmi2; run "monmux info" to see the inputs of the attached monitor)
```

An input name is a connector kind and, optionally, a port number: a positive decimal integer with no leading zero and no
separator. `dp`, `hdmi2` and `usb-c` are names; `HDMI`, `hdmi0`, `hdmi01`, `hdmi-1` and `scart` are not. This is a usage error,
exit code 1, and nothing is started for it.

### An unknown `--unsafe-model` name

```text
Error: no catalog entry is named "AOC/Q27P1C" (run "monmux catalog list" to see every entry)
```

The override names a catalog entry, so a name that is not one is a usage error, exit code 1, and nothing is started for it.
The accepted spellings are `Vendor/Name` and `Vendor Name`, case-insensitively — `monmux catalog list` prints them, and shell
completion offers them.

A name that is well formed but not enabled for the monitor you have is a different thing: it is refused with
`input-not-enabled` and exit code 2, and `monmux info` lists the inputs the attached monitor is enabled for. Symbolic inputs
are the only thing `switch` accepts either way. There is no way to pass a raw VCP code or value, by design.

## Reporting an issue

Include:

- `monmux doctor` and `monmux info --json`, without `--show-serial` (the default output is already redacted).
- The exact command you ran and its exit code.
- The output of `ddcutil --version` or `brew info m1ddc`.
- Your OS and desktop environment, and whether the monitor is connected directly or through a dock or hub.

For security issues, use GitHub's security advisories rather than a public issue — see [security.md](security.md).
