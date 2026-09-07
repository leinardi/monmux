# monmux

Switch supported monitors between video inputs, from the command line, on Linux and macOS.

`monmux` is fail-closed by design: it writes to a monitor only when that monitor is positively identified as a model in the
built-in supported-monitor catalog **and** the requested input is explicitly enabled for that model. Anything else — an unknown
monitor, an ambiguous match, an input without recorded evidence — is refused without a write. One model is verified so far.

It wraps `ddcutil` on Linux and `m1ddc` on macOS; there is no native I²C/IOKit backend.

> **Status: work in progress.** This README is a placeholder and will be replaced once the CLI is implemented.

## Documentation

- [Requirements](docs/requirements.md) — goal, verified findings, compatibility data and safety requirements.
- [AGENTS.md](AGENTS.md) — repository conventions, including the rule that no AI agent may write to a monitor.

## Licence

Apache License 2.0 — see [LICENSE](LICENSE).
