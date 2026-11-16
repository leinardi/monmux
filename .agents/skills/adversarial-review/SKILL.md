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

This skill routes to the project's authorities rather than replacing them: `AGENTS.md`, the
documentation and `go-style-guide` own the rules.

## Hard rule: reviewing never writes to a monitor

The HARD RULE in `AGENTS.md` applies to review exactly as it applies to implementation: run only
the commands it lists as allowed, and no test or script that executes the real `ddcutil` or
`m1ddc`. Anything a reviewer wants proven on real hardware goes into the report as a command for
the human to run, never executed here.

## Scope

When invoked directly: review the scope the user gives (`gh pr view <n>` for intent, then `gh pr
diff <n>`); with none, the uncommitted work (`git diff HEAD` plus untracked files, which no diff
shows); on a clean tree, `git diff main...HEAD`. Never review from a description of the change.

Read the stated **intent** (`git log --oneline` for the range, any linked issue or PR body): a
change that does something other than what it claims is a finding. Read every changed file in
full, not just the hunks, and for non-trivial changes the callers, implementations, tests and
docs of what changed, found with a reference search, not assumed from the diff: a signature or
behaviour change is only safe if every call site agrees. A change under a build tag needs its
counterpart on the other OS checked too.

## Authority (path → authority)

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
| `test/**` meta-tests, `scripts/check-*.sh` and their `*.test.sh` | the comment heading each test | a meta-test that can no longer fail is a finding (see below) |
| release, packaging, CI | `docs/release.md`, `.github/`, `.mk/` | documented-but-unimplemented versus accidentally changed |

A meta-test guards a claim about the repository rather than the product: the Go line and the
golangci-lint pin agree everywhere (`test/toolchain`), no gate passes on nothing, every Markdown
link resolves (`test/docs`), and the licence-header script refuses what it says
it refuses (their shell tests). Try to make the guarded thing go wrong and see whether the test
still notices. A test that passes because its input is empty, its walk found no files, its
allowlist entry outlived the line it excused, or its rule never ran is a false pass.

No matching document (bash, Makefile, plain YAML, Markdown)? Fall back to `AGENTS.md`, the
invariants and passes below. **Same rigor** — an unmatched area is not a lighter
review.

## Repository invariants

`AGENTS.md`'s fail-closed invariants 1–6 and its two blast-radius rules (*Never add a raw VCP
command*, *Backends only execute a `Command` produced by `Plan`*) are the checklist: read them
there, not from memory. Weakening any of them is normally **critical**. What each looks like in
a diff:

- **The `--unsafe-model` exception.** It is the only way past invariants 1 and 2: the EDID is not
  consulted and `write_enabled` is ignored (`Model.UnsafeOperation`, reached only from the
  `policy.assume` path). It still refuses when no display is writable, refuses more than one
  writable display with `multiple-candidates` (pin one with `--serial`), refuses an input the named
  entry does not record (`input-not-enabled`), takes the value from the compiled-in catalog, and
  still re-verifies at `Execute`. No configuration key or environment variable can arm it. A second
  caller of `UnsafeOperation`, a non-flag way to set `Request.AssumeModel`, or an assume path that
  picks among writable displays is critical.
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
- **Commit type matches whether it ships** (`AGENTS.md`'s release table, *Commit messages*).
  `svu` derives the release from Conventional Commits, so a change users should receive typed
  `chore`, `docs`, `refactor` or `perf` never ships, a stray `!` cuts a major, and a word ending
  in `feat` or `fix` right before a colon anywhere in a subject counts as that type. A release
  that adds or moves a byte monmux can send is at least a minor (the catalog rule), never a
  patch. PRs land as merge commits, so every commit in the range counts, not just the title
  (`docs/release.md`).

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

## Adversarial passes

Beyond the generic passes (logic, nil and empty input, aliasing, error handling, concurrency,
resources, security), run these with a "how would I make this fail" framing, and name one
concrete failing input per candidate rather than asserting "looks fine".

- **Contract drift:** a public signature, flag, config key, JSON field, refusal reason, exit
  code, output format or catalog entry changed without updating every consumer and the docs
  (*Docs drift* below).
- **Tests:** a test that passes against the *old* code, that asserts on a fake's recorded calls
  instead of the decision that produced them, or that was weakened or deleted to make the change
  pass is a finding. A bug fix with no regression test is a gap worth flagging.
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
- **monmux test gaps:** a dropped refusal-reason check, a fixture with a real serial, a new
  refusal reason without a test, and a new catalog input without a
  planned-command test are findings.

## Always-on passes

The passes above are shaped by the diff. These run on **every** review, whatever changed.

### (a) What reaches a monitor

Ask the one question the HARD RULE and the invariants are built on: does this change create a
new place where something from outside monmux — a flag, a config key, an environment variable,
a tool's output — becomes a byte sent to a monitor, a write decision, or a program that gets
executed? If it does, walk the identify-before-writing, enabled-inputs, raw-VCP and
`Plan`/`Execute` invariants against it.

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
silent, and no hook counts them: the review is where one is accepted or not. So demand the
attempt: for a complexity or length rule, was the obvious extraction tried and what broke? For
`wrapcheck`, why is wrapping wrong here? For `varnamelen`, why does the name have to be short? A
suppression whose reason
comment restates the rule instead of explaining why the fix does not apply is a finding, and so is
a suppression with no reason comment at all. The same holds for a new exclusion in
`.golangci.yaml` — and an exclusion, enable or setting there that matches nothing in this repo
(copied from another project) is a finding too.

### (d) Cross-file duplication

Before accepting a new helper, search for the one that already exists — in `internal/**` and
`cmd/**`, by *behaviour*, not by the name the author chose. The *Reuse before writing* table in
`go-style-guide` lists the helpers that already exist (the tool-path trust check, refusals, doctor
checks, redaction, catalog lookups, the fakes). Two implementations of the same rule drift apart,
and the one the reviewer did not read is the one that keeps the bug.

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

## Gates

Do not re-run the repository's gates (`make go-test`, `make go-vet`, `make check`, the `*-cross`
targets, `make audit-deps`, `make verify-scripts`). The implementer is expected to run them and CI
runs them again, but do not assume either happened: report the verification evidence you actually
have (under adversarial-review-loop, `verification.txt` when supplied, which is the implementer's
claim and not proof; when invoked directly, the CI status or gate output in the PR or session) and
what is missing, such as a backend or shared-file change with no cross-OS build or lint reported.
Judge what is missing against `AGENTS.md`'s *Validation matrix*: every changed path owes the checks
its row names. A failure in that evidence caused by the change is a confirmed finding. Never say
that a check you did not see passed, and a skipped test (`SKIP` in `-v` output) is not a pass. When
invoked directly, one focused test (`go test ./internal/<pkg> -run <Name>`) may be run to confirm a
single suspected finding, never a full suite or gate; the suite executes no external binary. A
candidate that is neither reproduced nor traced end to end is dropped: a vague "consider" is not a
finding.

Hardware behaviour is never something you run. A finding that can only be settled on real
hardware gets the exact command for the human to run and is marked unresolved.

## Report

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
run, then a one-line verdict: **block**, **approve with nits**, or **approve**, and one line:
`Verification evidence: seen … / missing …`. If you found nothing, state what you
tried to break so the "no findings" is credible. Be blunt; do not soften a real defect to be
polite, and do not invent findings to look thorough.
