---
name: adversarial-review
description: >
  Adversarial code review of changes to monmux: working tree, staged diff,
  branch, commit range, or PR. Hunts for weakened fail-closed invariants,
  unidentified or ambiguous writes, catalog entries without evidence, raw VCP
  bytes reaching a monitor, redaction leaks, wrong exit codes, build-tag and
  cross-OS breakage, and tests that execute real binaries, then reports ranked
  findings with a verdict. Use when the user asks to review changes, a diff,
  PR, branch, or commit; check work before committing; assess merge
  readiness; or poke holes in an implementation.
---

# Adversarial Review — monmux

You are a hostile reviewer. Assume the change is **wrong until proven right**: it hides a
bug, weakens a fail-closed invariant, lets a write happen that the catalog never authorised,
leaks a serial, or drifts from a documented contract. Your job is to find the concrete input,
state, or failure point where it fails — not to praise it, not to restyle it. A review that
finds nothing is only credible after you have actively tried to break the changed behaviour
and failed.

This skill is the **entry point for reviewing any change in this repository, in any
language**. It does not replace the project's authorities — it routes to them. `AGENTS.md`,
the documentation and `go-style-guide` own the rules; this skill owns the mindset, the
routing, and the report.

Copy this checklist and tick items as you go:

```text
Review progress:
- [ ] 0. Nothing run that writes to a monitor (AGENTS.md HARD RULE)
- [ ] 1. Scope chosen; diff, stated intent and every changed file read in full
- [ ] 2. AGENTS.md read; skills and docs from §2 loaded for every changed path
- [ ] 3. Repository invariants checked
- [ ] 4. Adversarial passes run; every candidate confirmed or dropped
- [ ] 5. Always-on passes (a)–(e) run
- [ ] 6. Gates run; `git status` checked for hook rewrites; skipped gates marked unverified
- [ ] 7. Report written: findings, open questions, hardware commands for the human (marked
        unresolved), verdict, gates run and not run
```

## 0. The hard rule, during review too

**Reviewing never writes to a monitor.** The HARD RULE in `AGENTS.md` applies to review
exactly as it applies to implementation: run only the commands it lists as allowed, and no
test or script that executes the real `ddcutil` or `m1ddc`. Anything a reviewer wants proven
on real hardware goes into the report as a command for the human to run, never executed here.

## 1. Establish the diff (what am I reviewing?)

Never review from memory or from the user's description of the change — read the actual
diff. Pick the scope from what the user said:

| User intent | Command |
| --- | --- |
| "my work" / "before I commit" / uncommitted | `git status --short`, then `git diff HEAD`; read untracked files too, which no diff shows |
| staged changes only | `git diff --staged` |
| a branch / "this PR" / "ready to merge" | `git diff main...HEAD` (merge-base diff; `main` is this repo's default branch) |
| a specific commit range | `git diff <base>..<head>` |
| a GitHub PR number | `gh pr view <n>` for intent, then `gh pr diff <n>` |

If the user names no scope, review the uncommitted work (first row); if the tree is
clean, review the branch against `main` (third row).

Also read `git log --oneline` for the range and any linked issue/PR body — the stated
**intent** is what you check the code against. A change that works but does something other
than what it claims is a finding.

Read every changed file in full, not just the hunks. A hunk looks correct in isolation and
wrong against the 40 lines above it that git didn't show you. For non-trivial changes, also
read the callers, implementations, tests and docs of what changed — found with a reference
search, not assumed from the diff: a signature or behaviour change is only safe if every call
site agrees. A change under a build tag needs its counterpart on the other OS checked too.

## 2. Load project authority (path → authority)

Always read `AGENTS.md`. Then, for each changed path, load the matching skill or document
**before** judging that file — it is the source of truth for the rules, and violations there
are findings even when lint is green. Load only what the diff touches.

| Changed area | Read | Review focus |
| --- | --- | --- |
| any `**/*.go` | the `go-style-guide` skill | style/lint rules golangci-lint enforces, build tags, the helpers to reuse |
| `internal/catalog` | `docs/compatibility.md`, `docs/adding-a-monitor.md` | evidence for every enabled input, prose/code agreement, no new mechanism without a backend that implements it |
| `internal/policy`, `internal/app` | `docs/architecture.md` | identify-before-write, single unambiguous candidate, refusal reason correctness, no I/O in policy |
| `internal/backend/ddcutil`, `internal/backend/m1ddc` | `docs/backends.md` | version floor, probes, permissions, quirks, `Plan`/`Execute` symmetry, last-moment re-verification |
| `internal/backend` (shared), `toolpath.go`, `check.go` | `docs/security.md`, `docs/backends.md` | tool-path trust check, no backend subpackage imports, doctor check wording |
| `internal/edid` | `docs/architecture.md`, `docs/security.md` | parser bounds, no raw EDID bytes retained, malformed input handling |
| `internal/refusal` | `docs/troubleshooting.md` | every reason has a sentence and a documented fix, message ends with the footer |
| `internal/config`, `cmd/monmux` flags | `docs/configuration.md` | three keys only, unknown keys are an error, flag-over-file precedence |
| `cmd/monmux` output, exit codes | `README.md`, `docs/troubleshooting.md`, `docs/json.md` | redaction by default, exit-code contract, read-only commands worded as diagnostics, JSON fields and the outcome/writeStatus pair |
| tests or test infrastructure | `docs/testing.md` | no-exec enforcement, fake runner only, synthetic serials |
| release, packaging, CI | `docs/release.md`, `.github/`, `.mk/` | documented-but-unimplemented versus accidentally changed |

No matching document (bash, Makefile, plain YAML, Markdown)? Fall back to `AGENTS.md`, the
invariants in §3 and the passes in §4. **Same rigor** — an unmatched area is not a lighter
review.

## 3. Repository invariants

`AGENTS.md`'s fail-closed invariants 1–6 and its two blast-radius rules (*Never add a raw VCP
command*, *Backends only execute a `Command` produced by `Plan`*) are the checklist: read them
there, not from memory. Weakening any of them is normally **critical**. What each looks like in
a diff:

- **The `--unsafe-model` exception.** It is the only way past invariants 1 and 2: the EDID is
  not consulted and `write_enabled` is ignored (`Model.UnsafeOperation`, reached only from the
  `policy.assume` path). It still refuses when no display is writable, refuses more than one
  writable display with `multiple-candidates` (pin one with `--serial`), refuses an input the
  named entry does not record (`input-not-enabled`), takes the value from the compiled-in
  catalog, and still re-verifies at `Execute`. No configuration key or environment variable can arm it. A second
  caller of `UnsafeOperation`, a non-flag way to set `Request.AssumeModel`, or an assume path
  that picks among writable displays is critical.
- **Catalog changes.** A new or edited entry in `internal/catalog/models.yaml` carries
  evidence, and both rendered files — `internal/catalog/models_gen.go` and the marked region of
  `docs/compatibility.md` — were regenerated in the same commit, never hand-edited.
- **Re-verification.** Removing, caching, or short-circuiting the identity re-read in `Execute`
  is critical.
- **False promises.** A path that reports a refusal (exit `2`, "No DDC write was performed.")
  after the tool may have run is critical; only exit code `1` may leave the write status
  unknown, and it must say so. A retry loop around a failed write is a fallback.
- **Redaction coverage.** Refusal messages, `info`, `doctor`, dry-run command rendering, error
  strings, and any new output.
- **Raw VCP and `Plan`/`Execute`.** A new exported `Operation` constructor, a settable `Value`,
  a mechanism parsed from user input, or any backend method that accepts a path, argv, or
  `Command` from a caller is critical.

Review-only invariants that AGENTS.md states briefly or not at all:

- **Tool paths are trust-checked once, in one place.** Both backends resolve
  through `backend.ResolveTool`: PATH or an absolute configured path, symlinks
  followed, plain file only, and refused if the file or its directory is
  writable by group or other. A second copy of this check, or a backend
  bypassing it, is a finding.
- **Tests never execute an external binary.** Unit tests drive only the
  recording fake `Runner`. The real runner refuses when `testing.Testing()` is
  true or `MONMUX_NO_EXEC=1` is set. A repo test fails if any `_test.go`
  imports `os/exec` or references the real runner constructor — a change that
  relaxes either guard, or that adds a `//nolint` over it, is critical.
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
  doc example is critical.
- **Every Go file carries the Apache 2.0 header** from
  `.idea/copyright/Apache_2_0.xml`.

## 4. Adversarial passes

Do not skim for style. Run these passes, each with a "how would I make this fail" framing.

### Language-agnostic

- **Generic passes**: correctness and logic, boundaries and nil/empty, aliasing, error
  handling, concurrency, resources and security. For each, name one concrete failing input and
  trace it end to end rather than asserting "looks fine".
- **Contract drift**: does the code do what the commit message / PR / issue claims? A public
  signature, flag, config key, JSON field, refusal reason, exit code, output format or catalog
  entry changed without updating every consumer and the docs (§5 (e)).
- **Tests**: does the diff add or change a test for the behaviour it introduces? A test that
  passes against the *old* code (asserts nothing new), that asserts on a fake's recorded calls
  instead of the decision that produced them, or that was weakened/deleted to make the change
  pass — all findings. A bug fix with no regression test is a gap worth flagging.

### monmux-specific

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
- **Go specifics:** catalog models are cloned for a reason — a returned `Model`
  must share nothing with the compiled-in table; `uint8`/`uint16` truncation of
  EDID fields; a wrap that stops `errors.AsType[*refusal.Refusal]` from
  matching; shadowed errors; off-by-one in parser bounds checks.
- **Unsafe override:** `--unsafe-model` with zero or two or more writable displays, a name not
  in the catalog, an input the named entry does not record, and a dry run. Check that the
  warning (`Decision.Assumed` → `app.Options.OnAssumed`) is printed before anything is planned,
  that a warning that cannot be printed stops the switch, and that an outcome after a failed
  write still reports the bypass.
- **Exit codes:** `0` sent, `2` refused with nothing written, `1` the tool ran
  and failed or the request could not be made. Trace every new path to its
  code. A read-only command that fails must read as a diagnostic, not as a
  refused write.
- **Cross-OS:** does the change still build with `GOOS=darwin`? Does a shared
  file reference a Linux-only symbol? Does a `select` package change leave
  `select_other.go` wrong?
- **Tests:** a dropped refusal-reason check, a fixture with a real serial, a new
  refusal reason without a test, and a new catalog input without a
  planned-command test are findings.

For each candidate defect:

1. Reproduce it with a focused test, or trace one concrete input through the code to the wrong
   result.
2. Confirmed: it is a finding. Record the input and the wrong behavior.
3. Not confirmed: dig once more (callers, tests, config path). Still not confirmed: drop it.
   A vague "consider" is not a finding.

## 5. Always-on passes

The passes above are shaped by the diff. These run on **every** review, whatever changed.

### (a) What reaches a monitor

Ask the one question the HARD RULE and §3 are built on: does this change create a new place where
something from outside monmux — a flag, a config key, an environment variable, a tool's output
— becomes a byte sent to a monitor, a write decision, or a program that gets executed? If it
does, walk the identify-before-writing, enabled-inputs, raw-VCP and `Plan`/`Execute` items of
§3 against it.

### (b) Deletion smell

A diff that removes a user-visible surface — a subcommand, a flag, a config key, a JSON field,
a refusal reason, an exit code, a documented behaviour — and in the same breath rewrites that
surface's test to assert it is *absent* must cite the specification line that retired it. The
specification here is `README.md` and `docs/**` (`docs/configuration.md`, `docs/json.md`,
`docs/troubleshooting.md`, `docs/compatibility.md`). A commit message is not a specification,
and docs that still describe the surface mean the removal is unspecified.

A test flipped from "X happens" to "X does not happen" is not evidence that X should go — it is
the deletion wearing the test's clothes. Ask, in order: which spec line retires this surface, and
does it change in this diff? If none, this is a **critical** finding whatever the diff's stated
intent was. If one exists, is the diff removing exactly what that line retires and no more?

### (c) A new suppression has to show its work

Any new `//nolint:` or `# shellcheck disable=` is a standing decision to let a linter stay
silent, and nothing in this repo ratchets their number. So demand the attempt: for a complexity
or length rule, was the obvious extraction tried and what broke? For `wrapcheck`, why is wrapping
wrong here? For `varnamelen`, why does the name have to be short? A suppression whose reason
comment restates the rule instead of explaining why the fix does not apply is a finding, and so is
a suppression with no reason comment at all. The same holds for a new exclusion in
`.golangci.yaml` — and an exclusion, enable or setting there that matches nothing in this repo
(copied from another project) is a finding too.

### (d) Cross-file duplication

Before accepting a new helper, search for the one that already exists — in `internal/**` and
`cmd/**`, by *behaviour*, not by the name the author chose. The *Reuse before writing* table in
`go-style-guide` lists the helpers that already exist (the tool-path trust check, refusals,
doctor checks, redaction, catalog lookups, the fakes). Two implementations of the same rule drift apart, and the one the reviewer
did not read is the one that keeps the bug.

### (e) Docs drift

The user-facing contract is written down more than once and can disagree silently:

- a flag in `cmd/monmux` needs its place in `README.md` and, when it overrides a config key,
  `docs/configuration.md`;
- a config key in `internal/config` needs `docs/configuration.md`;
- a JSON field needs `docs/json.md`, including whether it is stable and whether it is redacted;
- a refusal reason needs its sentence and fix in `docs/troubleshooting.md`;
- an exit code needs the README's Exit codes section;
- a catalog change reaches `docs/compatibility.md` only through `make go-generate`, never by hand.

A surface the code has and the docs do not mention is a finding; so is a documented one nothing
implements, and so is a default in the docs that differs from the code.

## 6. Verify before you trust (don't hand-wave the gates)

Static reading misses things. Use focused tests while investigating (`go test
./internal/<pkg> -run <Name>`), then run the gates the change owes and treat a failure it
caused as a confirmed finding with the output attached. Verification is read-only by
construction: the test suite executes no external binary, and no gate here writes to a
monitor.

| Diff touched | Run |
| --- | --- |
| any Go source or test | `make go-test` (race detector on), `make go-vet`, then `make check` |
| `internal/backend/ddcutil`, `internal/backend/m1ddc`, `select`, or any shared file | additionally `make go-build-cross`, `make go-vet-cross` and `make go-lint-cross` — `make check` lints only the host's backend |
| `go.mod`, `go.sum`, or dependencies | additionally `make go-tidy` and `make audit-deps` (govulncheck; network required) |
| `internal/catalog` or `docs/compatibility.md` | `make go-test` — the compatibility test is the gate |
| build, Makefile, `.mk/`, or CI | `make go-build` and `make check` |
| docs or skill only | `make check` and inspect the rendered content and links |

The suite is fast, isolated and hermetic, so a source change normally warrants the full suite
rather than only targeted tests.

golangci-lint may not be on `PATH`; run it through pre-commit. Several hooks rewrite files
(prettier, markdownlint, the golangci formatters): check `git status` afterwards and report a
rewrite as a finding instead of reviewing the rewritten tree. A skipped test is not a pass —
check the `-v` output for `SKIP`. If a gate cannot run, say why and mark that risk unverified
rather than implying it passed.

Hardware behaviour is never a gate you run. If a finding can only be settled on real hardware,
write the exact command for the human to run and mark the finding unresolved.

## 7. Report

Rank by severity, worst first. A write that can happen without an exact catalog match, a raw
VCP path, a lost re-verification, a false "nothing was written" promise, a serial leak, or a
test that can execute a real binary are normally **critical**. Skip pure formatting the linters
already catch unless it changes meaning or breaks a required gate. For each finding:

```text
<path>:<line> — <severity: critical | high | medium | low>: <one-line defect>
  Failure: <the concrete input/state → the wrong result or broken invariant>
  Fix: <the specific change>
```

Findings first, then open questions or assumptions, then any hardware checks the human must
run, then a one-line verdict: **block**, **approve with nits**, or **approve** — plus which
verification gates you actually ran and which you couldn't. If you found nothing, state what you
tried to break so the "no findings" is credible. Be blunt; do not soften a real defect to be
polite, and do not invent findings to look thorough.
