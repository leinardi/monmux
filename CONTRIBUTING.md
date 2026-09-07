# Contributing

monmux writes to hardware. The rules below are stricter than a typical Go project's for that reason, and one of them is
absolute — read [The rule](#the-rule) first.

## Prerequisites

- Go 1.26+
- [`pre-commit`](https://pre-commit.com/) — install the hooks once with `make pre-commit-install`
- For running monmux against real hardware: `ddcutil` 2.2+ on Linux, or `m1ddc` on macOS. Neither is needed to build or test.

Development works from either operating system. The macOS backend is cross-compiled and vetted from Linux, and everything that
is not a backend is tag-free and tested on both.

## The rule

**No AI agent — an assistant in an editor, a coding agent, a subagent, a review bot, any tool-driven automation — may run a
command that writes to a monitor.** Only a human runs a writing command, by hand, deliberately.

Forbidden for agents: `ddcutil setvcp …`, any `ddcutil` invocation with `--i2c-source-addr`, `m1ddc … set …`, `monmux switch`
without `--dry-run`, `i2cset`, `i2ctransfer`, any direct write to `/dev/i2c-*`, and any test or script that executes the real
`ddcutil` or `m1ddc` binary.

Allowed for agents, all read-only: reading `/sys/class/drm`, `ddcutil --version`, `ddcutil --help`, `ddcutil detect`,
`monmux info`, `monmux doctor`, `monmux switch … --dry-run`.

**Tests never write to a monitor and never execute any external binary at all.** That is enforced from inside the runner and by
a repository test — see [docs/testing.md](docs/testing.md). Hardware validation is a checklist a human runs, and the result
becomes evidence in the catalog.

## Building and testing

```sh
make go-build          # ./dist/monmux for the host OS
make go-test           # go test -race ./...
make go-vet
make go-build-darwin   # cross-compile the macOS backend from Linux
make go-vet-darwin
make check             # the full pre-commit suite
make check-stage       # the same, on staged files only
```

A single test:

```sh
go test ./internal/edid -run TestParse -v
```

The Makefile pulls shared snippets from `leinardi/make-common@v1` into `.mk/` on first run; `make mk-common-update` refreshes
them. Repo-local targets live in `.mk/cross.mk`.

## The catalog is evidence, not configuration

`internal/catalog/models.yaml` decides which bytes may reach which monitor. Edit it, run `make go-generate` to re-render
`internal/catalog/models_gen.go`, and commit both: the YAML is the readable source of truth and the generated Go is what the
binary compiles. A test fails the build if the two disagree, so every byte is still reviewable in a diff, and the rendering
happens at development time — the shipped binary has no catalog parser and reads no catalog file at run time.

**A model becomes write-enabled when somebody with that monitor in front of them switched it with that value and wrote down what
happened.** Not because the manufacturer is the same, not because the product family is similar, not because a value is
documented somewhere. Evidence is recorded per input, because a model can have one verified input and one that is merely
reported elsewhere.

If you have values but no unit to test them on, contribute the entry disabled, with a source. That is a real contribution: it
stops the next person guessing.

The full procedure, including how to sanitize a fixture and what belongs in the pull request, is in
[docs/adding-a-monitor.md](docs/adding-a-monitor.md).

Related rules that no change may weaken:

- **Never add a raw VCP command.** No code path may take a VCP code or value from a flag, a configuration file, an environment
  variable or any other input. The only bytes that reach a monitor come from a `catalog.Operation`, built by the catalog's
  package-private constructor from a compiled-in entry.
- **Never fall back.** A mechanism is a per-model property. A backend that does not implement one refuses; it does not try
  another mechanism, another VCP code or another value.
- **Never claim more than was done.** A successful write is reported as a command sent, not as a monitor switched.

The six fail-closed invariants are listed in [AGENTS.md](AGENTS.md), and the reasoning behind them is in
[docs/architecture.md](docs/architecture.md).

## Code style

Go style is enforced by `.golangci.yaml` (golangci-lint v2, `default: all`). The conventions are written out in
`.agents/skills/go-style-guide/SKILL.md` — read it before writing Go. A few that come up constantly:

- Only the two backends carry build tags. Everything else compiles for both `GOOS=linux` and `GOOS=darwin`; keep
  OS-independent logic — parsers especially — in tag-free files so it is unit-tested on either host.
- Every Go file starts with the Apache 2.0 header.
- Comments explain why, not what. A comment that restates the code is noise; a comment explaining why a check exists is the
  reason somebody will not delete it in six months.

## Branches, commits and pull requests

- Branch names: `feat/<short-description>`, `fix/<short-description>`, `chore/<short-description>`.
- Commit messages start with a conventional-commit type — `feat:`, `fix:`, `docs:`, `chore:`, `test:`, `refactor:` — followed by
  a short imperative subject. If the change affects what reaches a monitor, say so in the body, explicitly.
- Keep pull requests focused: one logical change each.
- All checks must pass before merge.

## Reporting security issues

Privately, through GitHub's security advisories — see [SECURITY.md](SECURITY.md).
