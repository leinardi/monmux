---
name: go-style-guide
description: >
  Go coding rules for monmux: the golangci-lint v2 (default: all) settings and how
  to satisfy them, the linux/darwin build-tag layout and cross-OS lint, refusal
  error handling (never wrapped), and the helpers to reuse (tool-path trust check,
  refusals, doctor checks, redaction, catalog lookups, fakes). Use when writing,
  editing or reviewing any .go file in monmux — new code, bug fixes, refactors or
  tests — before generating Go code, not after lint fails.
---

# Go Style Guide — monmux

Rules derived from `.golangci.yaml` (golangci-lint v2, `default: all`) and verified
against the existing codebase in `cmd/` and `internal/`. §1–§12 follow from the linters
and the build, §13–§15 are review rules the linters cannot check.

golangci-lint is not on `PATH` in every environment; run it through pre-commit (see *Lint,
test and cross-OS loop* at the end). That lints only the host's backend (§8): after touching
a tagged file or anything shared, also run `make go-lint-cross`, which lints both. Never
commit with `--no-verify` — fix the underlying issue instead.

Linters that are **disabled** in `.golangci.yaml`, so their rules do not apply:

| Disabled linter | Reason |
| --- | --- |
| `exhaustruct`, `exhaustruct_v5` | Requires every struct field to be set; too noisy for short-lived structs |
| `gomodguard` | Replaced by `gomodguard_v2`, which is enabled |
| `gochecknoglobals` | Package-level lookup tables (`kinds`, `grades`, `busPattern`) are intentional |
| `nonamedreturns` | Named returns are allowed |
| `wsl` | The deprecated v4 linter; its successor `wsl_v5` stays enabled |

The formatters (`gci`, `gofmt`, `gofumpt`, `goimports`, `golines`) run in the
`golangci-lint-fmt` hook and in CI.

---

## 1. Import grouping

Three groups separated by blank lines — stdlib, third-party, then local
`github.com/leinardi/monmux/...` — alphabetical within each group. `gci` and `goimports`
both enforce it and the `golangci-lint-fmt` hook rewrites it.

---

## 2. Error handling

### 2a. No inline error assignment in `if` (`noinlineerr`)

```go
// Wrong
if err := doSomething(); err != nil {

// Right
err := doSomething()
if err != nil {
    return err
}

err = secondThing() // = not := for the second and later assignments
```

### 2b. Wrap errors with `%w` (`errorlint`)

```go
return "", fmt.Errorf("locating the home directory: %w", err)
```

The prefix is a short, lowercase phrase naming the operation, not a full sentence: no
capital letters, no trailing period. Compare with `errors.Is` and the generic
`errors.AsType[T]` (`errors.AsType[*refusal.Refusal](err)`), never `==` on error values.
Prefer flat code with early returns; no `else` after a `return` (`revive`'s
`indent-error-flow`).

A refusal is the exception to wrapping: pass it up unchanged, because wrapping would
corrupt the message the user sees (the `//nolint:wrapcheck` sites in `internal/app` say
so).

### 2c. Errors are sentinels, detail is wrapped (`err113`)

`err113` flags every `errors.New` inside a function body and every `fmt.Errorf`
without a `%w` verb — a static message included. Declare the error once as a
package-level sentinel and attach the runtime detail by wrapping it:

```go
var ErrTooShort = errors.New("edid: shorter than one 128-byte block")

return Identity{}, fmt.Errorf("%w: got %d bytes", ErrTooShort, len(raw))
```

Export a sentinel (`Err…`) when callers need `errors.Is`; unexported is fine for
package-internal use. Exported sentinel messages start with the package name
(`edid: …`, `config: …`, `exec: …`). Suppress with `//nolint:err113` only when no
sentinel can fit, and say why (§3).

### 2d. Aggregating multiple errors

Use `errors.Join` over a slice of wrapped errors, as `input.CheckNumbering` does to report
every problem at once.

### 2e. Ignoring errors explicitly

When an error return genuinely cannot be acted upon (printing to stdout or
stderr), assign it to the blank identifier:

```go
_, _ = fmt.Fprintln(stderr, rendered)
```

---

## 3. `nolint` directives

`nolintlint` enforces three things:

- **Specific**: name every linter — no bare `//nolint`
- **Explanation required**: every directive needs `// reason`
- **No unused**: remove directives when the code no longer triggers that linter

The explanation says why the fix does not apply here, not which rule fired (§14). Put the
directive on the line before the statement or declaration it covers, or at the end of a
single-line statement:

```go
//nolint:gosec // an EDID product code is 16 bits by definition
ProductCode: uint16(decimal(b.fields[fieldModel])),

//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (f *Fake) Ready(_ context.Context, _ Display) error {
```

Multiple linters are comma-separated with no spaces. Name all that fire: `gocyclo` and
`cyclop` measure the same thing, and `gocognit` often joins them.

---

## 4. Complexity limits

| Linter | Threshold | Note |
| --- | --- | --- |
| `gocyclo` | 15 | Cyclomatic complexity |
| `cyclop` | 15 | Same metric, different linter — both fire together |
| `gocognit` | 35 | Cognitive complexity |
| `funlen` | 50 statements | Lines are disabled (`lines: -1`) |

Prefer extracting helpers over suppressing; when suppression is the right call, the
nolint comment says why. These limits apply to test files too: `.golangci.yaml` exempts tests only from
`dupl`, `goconst`, `lll`, `mnd`, `paralleltest`, `testpackage` and `varnamelen`.

---

## 5. Magic numbers (`mnd`)

Numbers 0, 1, 2, 3 are allowed everywhere. Any other literal integer/float in
an `argument`, `case`, `condition`, or `return` position needs a named constant.
Any literal used more than once, or that needs explaining, is a constant too:

```go
const (
    // blockSize is the length of EDID block 0, the only block monmux reads.
    blockSize = 128
    // headerLen is the length of the fixed 00 FF 00 header.
    headerLen = 8
)
```

`strings.SplitN` is excluded from mnd checks. Test files are fully exempt from `mnd`.

---

## 6. Struct size (`gocritic hugeParam`)

Structs over ~80 bytes passed by value trigger `hugeParam`. Pass by pointer — or suppress
when an interface fixes the signature (the `Backend` interface passes a `Display` by value;
§3). The same applies to `rangeValCopy`: iterate large slices by index and take a pointer
(`item := &items[idx]`).

---

## 7. Forbidden packages (`depguard`)

| Forbidden | Use instead |
| --- | --- |
| `github.com/pkg/errors` (rule `forbidden`) | stdlib `errors` + `fmt.Errorf(...%w...)` |
| `github.com/instana/testify` (rule `forbidden`) | `github.com/stretchr/testify` |

---

## 8. Build tags

monmux is cross-platform by default. Only the two OS backends are tagged, and
the tag is the very first line of the file (before the copyright block):

```go
//go:build linux

/*
 * Copyright ...
 */
```

- `//go:build linux` — `internal/backend/ddcutil/` (the ddcutil backend) and the
  Linux half of `internal/backend/select/`.
- `//go:build darwin` — `internal/backend/m1ddc/` (the m1ddc backend) and the
  macOS half of `internal/backend/select/`.
- `//go:build !linux && !darwin` — `internal/backend/select/select_other.go`, the
  fallback that keeps the module compiling (and refusing) on any other OS.
- Everything else — `cmd/`, `internal/{app,catalog,config,edid,policy,refusal,backend}`
  and `internal/backend/exec` — carries **no** build tag and must compile for both
  GOOS values.

Keep OS-independent logic out of tagged files so it can be tested on either host:
the m1ddc output parser lives in a tag-free `parse.go` for exactly this reason.
Test files mirror the build tag of the code they test.

---

## 9. Comments and `godox`

- `FIXME` is flagged by `godox`. `TODO` is allowed.
- gocritic's `whyNoLint` check is disabled, but every `//nolint` still needs an
  explanation (`require-explanation`).
- Every exported function, type, and variable has a doc comment beginning with
  the symbol name (`// Redacted returns a copy with both serial fields zeroed.`).
- Unexported symbols get one when their purpose is not obvious from the name.
- Inline comments explain *why*, not *what* (§14).
- Do not add doc comments or comment scaffolding to code you did not otherwise
  change, e.g. as a side effect of a bug fix.

---

## 10. Variable naming (`varnamelen`)

`varnamelen` flags a name shorter than 3 characters whose last use is more than 5 lines
from its declaration (defaults: `min-name-length: 3`, `max-distance: 5`). Test files are
exempt.

- **Receivers are exempt**: `(b *Backend)`, `(f *Fake)`, `(m Model)` are fine.
- **Parameters are checked like locals.** A one-letter parameter passes in a three-line
  function and is flagged as soon as the body grows, so name them from the start:

  ```go
  // Wrong
  func ResolveTool(b, c, k string) (string, error)

  // Right
  func ResolveTool(binary, configured, configKey string) (string, error)
  ```

Local variables follow the same distance rule; rename them to reflect their type or role.

---

## 11. `modernize` — no pointer-boxing helpers

The `modernize` linter (`newexpr` check) flags any function whose sole purpose is to return
a pointer to its argument — the generic `func ptr[T any](v T) *T` included — at the
declaration and at every call site. The Go version in `go.mod` lets `new` take an
expression: write `new(true)` or `new(int64(5))`. Taking the address of a local is fine too.

---

## 12. Other thresholds

| Rule | Setting | What to do |
| --- | --- | --- |
| `any` (`gofmt` rewrite rule) | `interface{}` → `any` | Write `any` in new code so the formatter does not change your diff |
| `lll` | 140 characters | `golines` wraps automatically; test files are exempt |
| `dupl` | 100 tokens | Extract shared logic into a helper; test files are exempt |
| `govet` shadow | enabled | Use distinct names instead of re-declaring an outer variable with `:=` |
| `goconst` | 3+ occurrences, length ≥ 2 | Extract to a named constant; test files are exempt |

---

## 13. Reuse before writing

Every helper below exists so the hand-written version of it is written once. Before adding a
path check, a refusal, a doctor check or a fake, check whether one of these already answers the
question — and if it nearly does, extend it rather than forking it.

| Need | Use | Not |
| --- | --- | --- |
| Deciding a tool binary is trusted enough to run | `backend.ResolveTool` (`internal/backend/toolpath.go`), shared by both backends | a per-backend `exec.LookPath` and permission check: two copies of a security check drift |
| Refusing a write | `refusal.New(reason, detail, detected...)`; `refusal.Is(err, reason)` to test for one | a plain error on a path that promises nothing was written |
| A line of `doctor` output | `backend.CheckPassed`, `CheckResult`, `CheckFailed`, `FingerprintCheck`, `DisplayCount`, `FirstLine` (`internal/backend/check.go`) | a `Check{…}` literal or wording of its own in one backend |
| Printing an identity | `edid.Identity.Redacted()` unless `--show-serial` was passed | zeroing the serial fields by hand |
| Looking a model up | `catalog.Match` (by EDID identity), `catalog.Find` (by name, `--unsafe-model` only), `catalog.Models()` — all return copies | a loop over the entries at the call site |
| Knowing whether a tool may have run | `exec.Started(result, err)` | inspecting the error at the call site |
| Running a tool in a test | `exec.NewFake(rules...)` (`internal/backend/exec/fake.go`) | `os/exec` or the real runner — a repo test fails the build |
| Driving the app or the CLI in a test | `backend.Fake` (`internal/backend/fake.go`) | a second fake backend |

---

## 14. Comments carry rationale; history goes in the commit

A comment says **why the code is the way it is** — the constraint, the failure it avoids, the
alternative that was rejected and what broke. It does not narrate what changed, when, or at whose
request. That belongs in the commit body, where `git log` and `git blame` can find it and where it
does not rot as the code moves.

```go
// Bad — history in the code.
// Moved here from the ddcutil backend after the m1ddc backend needed it too.

// Good — rationale in the code.
// Both backends share it: this is a security check, and two copies of a
// security check drift.
func ResolveTool(binary, configured, configKey string) (string, error) {
```

The same rule is what makes `//nolint` explanations useful: say why the fix does not apply here,
not that the linter complained.

---

## 15. Waiting in tests

Tests drive `exec.Fake` and `backend.Fake` synchronously: no `time.Sleep`, no goroutines,
nothing to wait for. If a test has to wait, decide what kind of wait it is *before* writing it:

- **Positive eventual** ("something will have happened") — never a sleep: wait on the signal, or
  poll with a deadline and fail naming the condition that never held.
- **Negative assertion** ("nothing happens"), **real elapsed window** (the duration is under test),
  **ordering barrier with no quiescence signal** — a sleep, with a comment saying which of these it
  is; name the duration as a constant or a multiple of the interval under test.

A sleep whose comment says "give X time to Y" where Y is observable, and a sleep added to make a
flaky test pass, are always wrong.

---

## What to avoid

- `pkg/errors` (§7).
- `log.Fatal`, or `os.Exit` anywhere but a `main` function: `main()` calls `os.Exit(run(...))`
  exactly once so deferred cleanup always runs.
- `os/exec` in a test, or anything else that could run a real `ddcutil` or `m1ddc` (§13).
- OS-independent logic in a build-tagged file (§8).
- `interface{}` (§12) and pointer-boxing helpers (§11).
- Designing for hypothetical requirements: no configurability, abstractions or helpers for
  features that do not exist yet.
- Skipping or suppressing pre-commit hooks (`--no-verify`).
- Adding comments to code you did not change (§9).

---

## Quick checklist before submitting Go code

- [ ] Imports in 3 groups: stdlib / third-party / local, alphabetical within each
- [ ] No `if err := f(); err != nil` — split to two lines
- [ ] All errors wrapped with `%w`, refusals passed up unchanged
- [ ] No `errors.New` or `%w`-less `fmt.Errorf` in a function body: wrap a package-level sentinel
- [ ] `any` not `interface{}`; `new(expr)`, not a pointer-boxing helper
- [ ] Numbers other than 0–3 extracted to named constants (non-test code)
- [ ] Each `//nolint` names specific linters and explains why the fix does not apply
- [ ] No `FIXME` comments
- [ ] `//go:build linux` / `//go:build darwin` first line in backend-specific files; `make go-lint-cross`
      after touching one
- [ ] Function statement count ≤ 50, test files included
- [ ] No shadowed variables
- [ ] Checked §13 for an existing helper before writing a new one
- [ ] Comments say why, not what changed — history is in the commit body (§14)
- [ ] No `time.Sleep` in tests (§15)

## Lint, test and cross-OS loop

1. Run `pre-commit run golangci-lint-fmt --files <changed .go files>` and
   `pre-commit run golangci-lint-full --files <changed .go files>` (or `--all-files`).
2. Run `make go-vet` and `make go-test`. After editing `internal/catalog/models.yaml`, run
   `make go-generate` first and commit what it renders.
3. After touching a build-tagged file or anything shared, run `make go-build-cross`,
   `make go-vet-cross` and `make go-lint-cross`: step 1 lints only the host's backend.
4. Fix each report and re-run from step 1 until all of them are clean.
5. Check `git status`: the formatter hook rewrites files in place, so review and keep its changes.
