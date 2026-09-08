# AGENTS.md

## HARD RULE: no monitor writes by any AI agent

**No AI agent — main session, subagent, reviewer, or any tool-driven automation — may run a command that writes to a monitor.**
Only the human user runs writing commands, by hand, when they decide to. This applies during implementation *and* during review.

Forbidden for agents:

- `ddcutil setvcp …`, and any `ddcutil` invocation with `--i2c-source-addr`
- `m1ddc … set …`
- `monmux switch` without `--dry-run`
- `i2cset`, `i2ctransfer`, or any direct write to `/dev/i2c-*`
- any test or script that executes the real `ddcutil` / `m1ddc` binaries

Allowed for agents (read-only):

- reading `/sys/class/drm/**`
- `ddcutil --version`, `ddcutil --help`, `ddcutil detect`
- `monmux info`, `monmux doctor`, `monmux switch … --dry-run`

**Tests never perform DDC writes and never execute any external binary at all.** Unit tests use only the recording fake `Runner`.
This is enforced from inside the real runner: it refuses to execute when `testing.Testing()` is true (which covers `make go-test`, the
pre-commit `go-test-repo-mod` hook, and IDE runs) or when `MONMUX_NO_EXEC=1` is set (a belt for scripts). A repo test also fails if any
`_test.go` imports `os/exec` or references the real runner constructor.

Hardware validation is a checklist the human runs; agents prepare the commands and wait.

## What this is

`monmux` is a cross-platform Go CLI that switches **supported monitors** between video inputs, with a fail-closed policy: a write happens
only when the attached monitor is positively identified as a model in the built-in supported-monitor catalog *and* the requested input is
explicitly enabled for that model. One model is verified so far (LG 38WR85QC-W); nothing in the code, CLI, or docs is vendor-specific
except the catalog entry and the mechanism it uses.

It wraps `ddcutil` on Linux and `m1ddc` on macOS, permanently. There is no native I²C/IOKit backend and there never will be one.

Requirements: [`docs/requirements.md`](docs/requirements.md) — the source document, kept as written.

Documentation, and what each page is for:

| Page                                                   | What it answers                                                          |
| ------------------------------------------------------ | ------------------------------------------------------------------------ |
| [`README.md`](README.md)                               | What monmux is, how to install it, how to run it.                        |
| [`docs/architecture.md`](docs/architecture.md)         | How a switch is decided, and the five things that make it safe.          |
| [`docs/backends.md`](docs/backends.md)                 | ddcutil and m1ddc specifics: version floor, probes, permissions, quirks. |
| [`docs/compatibility.md`](docs/compatibility.md)       | The catalog, in prose. Checked against the code by a test.               |
| [`docs/adding-a-monitor.md`](docs/adding-a-monitor.md) | The procedure for enabling a model or an input.                          |
| [`docs/configuration.md`](docs/configuration.md)       | The configuration file, the flags, and which wins.                       |
| [`docs/security.md`](docs/security.md)                 | Threat model, mitigations, trust boundaries, what monmux never does.     |
| [`docs/testing.md`](docs/testing.md)                   | How the no-exec rule is enforced, and the human hardware checklist.      |
| [`docs/troubleshooting.md`](docs/troubleshooting.md)   | Every refusal reason and its fix.                                        |
| [`docs/release.md`](docs/release.md)                   | The release pipeline, as a design. Not implemented.                      |
| [`CONTRIBUTING.md`](CONTRIBUTING.md)                   | Prerequisites, workflow, and the catalog evidence rule.                  |

A change to the catalog must update [`docs/compatibility.md`](docs/compatibility.md) and re-render `models_gen.go` in the same
commit: one test compares the document against the catalog and another compares the catalog against its YAML, and both fail the build
if they disagree.

## Common commands

```bash
make go-build          # build ./dist/monmux for the host OS
make go-test           # CGO_ENABLED=1 go test -race ./...
make go-vet
make go-generate       # re-render internal/catalog/models_gen.go from models.yaml
make go-build-cross    # compile-check both OS backends (GOOS=linux and GOOS=darwin), whatever the host
make go-vet-cross      # go vet both, their build-tagged tests included
make go-lint-cross     # lint both; `make check` only ever lints the host's backend
make go-tidy           # go mod tidy + go mod verify
make check             # pre-commit on all files
make check-stage       # pre-commit on the staging area only
```

Single test:

```bash
go test ./internal/edid -run TestParse -v
```

The Makefile pulls shared snippets from `leinardi/make-common@v1` into `.mk/` on first run. To refresh: `make mk-common-update`.
Repo-local make targets live in `.mk/cross.mk`.

## Package map

- `cmd/monmux` — cobra commands: `info`, `switch`, `doctor`, `version`, `completion`. Owns flags, config loading, output formatting, and exit codes.
- `internal/refusal` — the one typed refusal error used by every layer, with the reason enum and the rendered message.
- `internal/edid` — EDID block-0 parser producing an `Identity`. Carries no raw EDID bytes.
- `internal/catalog` — the supported-monitor catalog, the `Input` name (a connector kind plus an optional port number), the `Kind`
  and `Mechanism` enums, and the opaque `Operation`. The catalog is written in `models.yaml` and rendered into the committed
  `models_gen.go` by `make go-generate`; a test fails the build if the two disagree. `internal/catalog/internal/generate` is that
  renderer, and it deliberately does not import `internal/catalog`, so a deleted or corrupt `models_gen.go` can still be
  regenerated.
- `internal/catalog/internal/input` — the grammar of an input name: the closed list of connector kinds, the parser, the labels
  and the ordering. A shared leaf, imported by both `internal/catalog` and the generator, so the two agree on what a name is
  without the generator importing the package whose source it writes.
- `internal/policy` — pure decision logic: displays plus request in, `Decision` or refusal out. No I/O.
- `internal/backend` — the `Backend` interface plus the `Display`, `Command` and `Check` types, the shared tool-path trust check
  (`toolpath.go`), the shared doctor check helpers (`check.go`) and the `Fake` backend the app and CLI tests drive. Imports no
  backend subpackage.
- `internal/backend/exec` — the `Runner` interface, the real runner (with the no-exec guard) and the recording fake.
- `internal/backend/select` — picks the backend by `runtime.GOOS`.
- `internal/backend/ddcutil` — Linux backend (`//go:build linux`).
- `internal/backend/m1ddc` — macOS backend (`//go:build darwin`), with the output parser and every decision taken from it in
  tag-free `parse.go` and `decide.go`, unit-tested on Linux.
- `internal/app` — orchestration: preflight, enumerate, resolve, ready, plan, execute; and `Info`, which never writes.
- `internal/config` — the three-key configuration file (`serial`, `ddcutil_path`, `m1ddc_path`). Unknown keys are an error.

## Fail-closed invariants

Do not weaken any of these. They are the reason the tool exists.

1. **Identify before writing.** A write requires an exact catalog match on the parsed EDID identity — manufacturer and product code,
   plus the EDID model name where an identity pins one, which is how two models sharing a reused product code are told apart.
   Unknown, ambiguous, or multiple candidate monitors are refused — never guessed at, never defaulted.
2. **Enabled inputs only.** An input is writable only if the matched model explicitly enables it, with recorded evidence. A model with no
   identities is never matched and therefore never written to.
3. **Re-verify at the last moment.** `Execute` re-reads the identity (sysfs EDID and bus on Linux, `display list detailed` on macOS) and
   refuses with `identity-changed` if anything moved since enumeration.
4. **Refuse loudly, write never.** Every refusal path returns a `refusal.Refusal` whose message ends with `No DDC write was performed.`
   and exits with code 2. Only code 1 (tool/exec error) may leave the write status unknown, and it says so explicitly.
5. **Never fall back.** If the chosen mechanism is not implemented by the backend, refuse with `invalid-operation`. Do not try another
   mechanism, another VCP code, or another value.
6. **Redact by default.** Serial numbers, serial strings, raw EDID hex and macOS UUIDs are masked in all output unless `--show-serial`
   is passed.

## Two rules that keep the blast radius small

**Never add a raw VCP command.** No code path may take a VCP code or value from a flag, a config file, an environment variable, or any
other input. The only bytes that reach a monitor come from a `catalog.Operation` built by the catalog's package-private constructor from
a compiled-in table entry with recorded evidence. Adding an input or a model means editing `internal/catalog/models.yaml`, supplying
the evidence, and running `make go-generate` to re-render `models_gen.go` — commit both — see
[`docs/adding-a-monitor.md`](docs/adding-a-monitor.md). A new connector *kind* is the one exception: the kinds are a closed list
in `internal/catalog/internal/input`, so a monitor with an input nobody has named yet is a Go change there, added with the first
model that needs it. A new *port* of a kind that already exists (`hdmi3`, `usb-c2`) is only a YAML edit. The YAML is a build input read at development time only: the binary contains no
catalog parser and reads no catalog file at run time.

**Backends only execute a `Command` produced by `Plan`.** `Command` is returned by `Plan` for display only and is never accepted as
input by any method. `Execute` takes the `catalog.Operation`, not a `Command`, and rebuilds the invocation through the same private
planner that `Plan` uses, so no caller can hand a backend an arbitrary executable or arguments. What `--dry-run` prints is what a real
run executes, by construction.

## Conventions worth knowing

- Go style is enforced by `.golangci.yaml` (golangci-lint v2, `default: all`). The rules are written out in
  `.agents/skills/go-style-guide/SKILL.md`; read it before writing Go.
- Only the two backends carry build tags. Everything else must compile for both `GOOS=linux` and `GOOS=darwin`; keep OS-independent
  logic (parsers especially) in tag-free files so it can be unit-tested on either host.
- Version strings (`version`, `commit`, `date`) live in `cmd/monmux/version.go` and are filled by `-ldflags -X main.version=...` from
  `GO_LDFLAGS` in `.mk/go.mk`.
- Every Go file starts with the Apache 2.0 header from `.idea/copyright/Apache_2_0.xml`.
- Test fixtures contain synthetic serials and UUIDs only, from a fixed allowlist that a test enforces. Never commit a real serial.
  The whitespace pre-commit hooks skip `testdata/`, so a capture keeps the bytes the real tool produced.
- Exit codes are part of the interface: `0` sent, `2` refused with nothing written, `1` the tool ran and failed or the request
  could not be made. Only a refusal may promise that nothing was written.
- A read-only command that fails is worded as a diagnostic, not as a refused write: `info` and `doctor` print their report and
  then one line naming the check that stopped them.
