---
name: adversarial-review
description: >
  Adversarial code review of changes to monmux: working tree, staged diff,
  branch, commit range, or PR. Hunts for weakened fail-closed invariants,
  unidentified or ambiguous writes, catalog entries without evidence, raw VCP
  bytes reaching a monitor, redaction leaks, wrong exit codes, build-tag and
  cross-OS breakage, and tests that execute real binaries, then reports ranked
  findings. Use whenever the user asks to review changes, a diff, PR, branch,
  or commit; check work before committing; assess merge readiness; or poke
  holes in an implementation.
---

# Adversarial Review — monmux

Assume the change is wrong until proven right: it hides a bug, weakens a
fail-closed invariant, lets a write happen that the catalog never authorised,
leaks a serial, or drifts from a documented contract. Find the concrete input,
state, or failure point where it fails. Do not praise or restyle the change. A
review with no findings is credible only after active attempts to break the
changed behaviour.

This skill is the entry point for reviewing any change in this repository.
`AGENTS.md` and the project documentation remain the sources of truth; this
skill defines review procedure and reporting.

## 0. The hard rule, during review too

**Reviewing never writes to a monitor.** The `AGENTS.md` prohibition applies to
this skill exactly as it applies to implementation: no `ddcutil setvcp`, no
`ddcutil --i2c-source-addr`, no `m1ddc … set …`, no `monmux switch` without
`--dry-run`, no `i2cset`/`i2ctransfer`, no writes to `/dev/i2c-*`, and no test
or script that executes the real `ddcutil` or `m1ddc` binary.

Read-only verification is allowed: `/sys/class/drm/**`, `ddcutil --version`,
`ddcutil --help`, `ddcutil detect`, `monmux info`, `monmux doctor`, and
`monmux switch … --dry-run`. Anything a reviewer wants proven on real hardware
goes into the report as a command for the human to run, never executed here.

Also read `.agents/skills/go-style-guide/SKILL.md` before judging Go style; the
enforced rules live there, not in your habits.

## 1. Establish the diff

Never review from memory or only from the user's description. Read the actual
diff and determine its intent.

| User intent | Command |
| --- | --- |
| "my work", "before I commit", uncommitted changes | `git status --short`, then `git diff HEAD`; inspect untracked files too |
| staged changes only | `git diff --staged` |
| branch, "this PR", "ready to merge" | determine the base branch (`main`), then `git diff main...HEAD` |
| specific commit range | `git diff <base>..<head>` |
| GitHub PR number | `gh pr view <n>` for intent and metadata, then `gh pr diff <n>` |

Read `git log --oneline` for the reviewed range and any linked issue or PR
body. Code that works but does something other than the stated intent is a
finding.

Read every changed file with enough surrounding context to understand its
contracts. For non-trivial behaviour changes, inspect callers, implementations,
tests, and documentation that depend on the changed symbol. Use symbol and
reference tools rather than assuming all call sites appear in the diff. A
change under a build tag needs its counterpart on the other OS checked too.

## 2. Load project authority

Always read `AGENTS.md`. Load only the additional documentation relevant to the
changed paths:

| Changed area | Read | Review focus |
| --- | --- | --- |
| `internal/catalog` | `docs/compatibility.md`, `docs/adding-a-monitor.md` | evidence for every enabled input, prose/code agreement, no new mechanism without a backend that implements it |
| `internal/policy`, `internal/app` | `docs/architecture.md` | identify-before-write, single unambiguous candidate, refusal reason correctness, no I/O in policy |
| `internal/backend/ddcutil`, `internal/backend/m1ddc` | `docs/backends.md` | version floor, probes, permissions, quirks, `Plan`/`Execute` symmetry, last-moment re-verification |
| `internal/backend` (shared), `toolpath.go`, `check.go` | `docs/security.md`, `docs/backends.md` | tool-path trust check, no backend subpackage imports, doctor check wording |
| `internal/edid` | `docs/architecture.md`, `docs/security.md` | parser bounds, no raw EDID bytes retained, malformed input handling |
| `internal/refusal` | `docs/troubleshooting.md` | every reason has a sentence and a documented fix, message ends with the footer |
| `internal/config`, `cmd/monmux` flags | `docs/configuration.md` | three keys only, unknown keys are an error, flag-over-file precedence |
| `cmd/monmux` output, exit codes | `README.md`, `docs/troubleshooting.md` | redaction by default, exit-code contract, read-only commands worded as diagnostics |
| tests or test infrastructure | `docs/testing.md` | no-exec enforcement, fake runner only, synthetic serials |
| release, packaging, CI | `docs/release.md`, `.github/`, `.mk/` | documented-but-unimplemented versus accidentally changed |

No matching document does not mean lighter review. Apply `AGENTS.md`, the
invariants below, and the general adversarial passes.

## 3. Repository invariants

Check these whenever affected, directly or indirectly. Weakening any of them is
normally a blocker.

- **Identify before writing.** A write requires an exact catalog match on the
  parsed EDID identity. Unknown, ambiguous, or multiple candidate monitors are
  refused — never guessed at, never defaulted, never "best match". A model with
  no identities must never be matched.
- **Enabled inputs only.** An input is writable only if the matched model
  explicitly enables it, with recorded evidence. Check that a new or edited
  entry in `internal/catalog/models.yaml` carries evidence, that
  `internal/catalog/models_gen.go` was regenerated from it (never hand-edited),
  and that `docs/compatibility.md` was updated in the same commit — a test
  compares each pair.
- **Re-verify at the last moment.** `Execute` must re-read the identity (sysfs
  EDID and bus on Linux, `display list detailed` on macOS) and refuse with
  `identity-changed` if anything moved since enumeration. Removing, caching, or
  short-circuiting that re-read is a blocker.
- **Refuse loudly, write never.** Every refusal path returns a
  `refusal.Refusal` whose rendered message ends with
  `No DDC write was performed.` and exits with code 2. A path that reports a
  refusal after the tool may have run is a false promise: only exit code 1 may
  leave the write status unknown, and it must say so.
- **Never fall back.** If the chosen mechanism is not implemented by the
  backend, refuse with `invalid-operation`. No trying another mechanism,
  another VCP code, another value, or a retry loop around a failed write.
- **Redact by default.** Serial numbers, serial strings, raw EDID hex and macOS
  UUIDs are masked unless `--show-serial` is passed. This covers refusal
  messages, `info`, `doctor`, dry-run command rendering, error strings, and any
  new output.
- **No raw VCP command, ever.** No code path may take a VCP code or value from
  a flag, config file, environment variable, or any other input. The only bytes
  that reach a monitor come from a `catalog.Operation` built by the catalog's
  package-private constructor from a compiled-in table entry. A new exported
  `Operation` constructor, a settable `Value`, or a mechanism parsed from user
  input is a blocker.
- **Backends only execute a `Command` produced by `Plan`.** `Command` is
  returned for display only and is never accepted as input by any method.
  `Execute` takes the `catalog.Operation` and rebuilds the invocation through
  the same private planner, so what `--dry-run` prints is what a real run
  executes. Any method that accepts a path, argv, or `Command` from a caller
  breaks this.
- **Tool paths are trust-checked once, in one place.** Both backends resolve
  through `backend.ResolveTool`: PATH or an absolute configured path, symlinks
  followed, plain file only, and refused if the file or its directory is
  writable by group or other. A second copy of this check, or a backend
  bypassing it, is a finding.
- **Tests never execute an external binary.** Unit tests drive only the
  recording fake `Runner`. The real runner refuses when `testing.Testing()` is
  true or `MONMUX_NO_EXEC=1` is set. A repo test fails if any `_test.go`
  imports `os/exec` or references the real runner constructor — a change that
  relaxes either guard, or that adds a `//nolint` over it, is a blocker.
- **Layering holds.** `internal/policy` is pure decision logic with no I/O.
  `internal/backend` imports no backend subpackage. `internal/edid` carries no
  raw EDID bytes out of the parser. `internal/app` orchestrates and never
  formats for the terminal; `cmd/monmux` owns flags, output and exit codes.
- **Only backends carry build tags.** Everything else must compile for both
  `GOOS=linux` and `GOOS=darwin`. OS-independent logic — parsers especially —
  belongs in tag-free files so it is unit-testable on either host. Logic moved
  into a tagged file loses its Linux-side test coverage; say so.
- **Fixtures are synthetic.** Serials and UUIDs in `testdata/` come from the
  fixed allowlist a test enforces. A real serial in a fixture, a commit, or a
  doc example is a blocker.
- **Every Go file carries the Apache 2.0 header** from
  `.idea/copyright/Apache_2_0.xml`.

## 4. Adversarial passes

Do not skim for style. Run each pass with "how can this fail?" framing:

- **Identification:** malformed or truncated EDID, missing descriptor, absent
  or non-alphanumeric serial, two attached displays matching the same model,
  one matching and one not, an identity that matches two catalog entries, a
  display that appears between enumeration and execute, a `--serial` pin that
  matches nothing or matches more than one.
- **Refusal correctness:** for each new or reordered branch, confirm the reason
  is the accurate one, that the identities attached to it are the right ones,
  and that the branch really precedes any execution. Look for a refusal
  constructed after `Run` was called, and for an error that merely wraps a
  refusal being reported as one.
- **Write path:** what `Plan` prints versus what `Execute` builds; the
  re-verification comparison (bus, identity fields, which mismatch counts);
  context cancellation mid-write; a tool that started and failed versus one
  that never started; non-zero exit codes; timeouts.
- **Backend parsing:** unexpected `ddcutil detect`/`m1ddc` output, absent
  fields, extra displays, locale or version differences, a version below the
  floor, partially readable sysfs, permission errors on `/dev/i2c-*`. Feed the
  parser the ugliest capture you can construct in `testdata/`.
- **Redaction:** inspect every new format string, error, log line, rendered
  command, doctor check detail and struct `String()` method for a serial, a
  serial string, EDID hex or a macOS UUID reaching output without the
  `--show-serial` gate. Search for identity fields crossing into a message.
- **Config and flags:** unknown key, wrong type, empty file, missing file,
  unreadable file, relative `ddcutil_path`, `XDG_CONFIG_HOME` overrides, flag
  versus file precedence, and whether the documented precedence still holds.
- **Go correctness:** slice and map aliasing (catalog models are cloned for a
  reason), a returned slice sharing a package-level backing array, nil-versus-
  empty conflation, `uint8` truncation, error wrapping that loses
  `errors.As(&refusal.Refusal{})`, `%w` versus `%v`, shadowed errors, ignored
  `Close`, context misuse, and off-by-one in parser bounds checks.
- **Exit codes:** `0` sent, `2` refused with nothing written, `1` the tool ran
  and failed or the request could not be made. Trace every new path to its
  code. A read-only command that fails must read as a diagnostic, not as a
  refused write.
- **Cross-OS:** does the change still build with `GOOS=darwin`? Does a shared
  file reference a Linux-only symbol? Does a `select` package change leave
  `select_other.go` wrong?
- **Contract drift:** compare the implementation with `README.md`, CLI help,
  `docs/architecture.md`, `docs/configuration.md`, `docs/troubleshooting.md`,
  `docs/compatibility.md`, and the commit or PR intent. Flag any undocumented
  flag, output, refusal reason, config key, exit code, or catalog change.
- **Tests:** require behaviour-focused regression coverage for changed
  behaviour. Reject tests that pass against the old code, assert on the fake's
  recorded call instead of the decision that produced it, weaken an existing
  assertion, drop a refusal-reason check, or add a fixture with a real serial.
  A new refusal reason without a test, and a new catalog input without a
  planned-command test, are findings.

Prefer one reproducible defect over ten vague suggestions. If you cannot name
the triggering state and the wrong result or broken invariant, keep
investigating or omit it.

## 5. Verify findings and gates

Use focused tests while investigating (`go test ./internal/<pkg> -run <Name>`),
then run the gate the changed set owes. Verification is read-only by
construction: the test suite executes no external binary, and no gate here
writes to a monitor.

| Diff touched | Run |
| --- | --- |
| any Go source or test | `make go-test`, then `make go-vet` |
| `internal/backend/m1ddc`, `select`, or any shared file | additionally `make go-build-darwin` and `make go-vet-darwin` |
| `go.mod`, `go.sum`, or dependencies | additionally `make go-tidy` |
| `internal/catalog` or `docs/compatibility.md` | `make go-test` — the compatibility test is the gate |
| build, Makefile, `.mk/`, or CI | `make go-build` and `make check` |
| broad change or merge-readiness review | `make check` (pre-commit on all files), plus `make go-test` |
| docs or skill only | `make check` and inspect the rendered content and links |

The suite is fast, isolated and hermetic, so a source change normally warrants
the full suite rather than only targeted tests. A failing gate is a confirmed
finding when the reviewed change caused it. If a gate cannot run, state why and
mark it unverified; never imply it passed.

Hardware behaviour is never a gate you run. If a finding can only be settled on
real hardware, write the exact command for the human to run and mark the
finding unresolved.

## 6. Report

Rank findings by severity, worst first. A write that can happen without an
exact catalog match, a raw VCP path, a lost re-verification, a false "nothing
was written" promise, a serial leak, or a test that can execute a real binary
are normally blockers. Skip pure formatting unless it changes meaning or breaks
a required gate.

For each finding:

```text
<path>:<line> - <severity: blocker | high | medium | low>: <one-line defect>
  Failure: <concrete input/state -> wrong result or broken invariant>
  Fix: <specific corrective change>
```

Put findings first. Then list open questions or assumptions, followed by any
hardware checks the human must run, followed by a one-line verdict: **block**,
**approve with nits**, or **approve**. Include gates actually run and gates not
run. If no findings exist, say so explicitly and briefly name the failure modes
you tried to trigger. Be blunt, but never invent a finding to appear thorough.
