---
name: go-style-guide
description: >
  Project-specific Go coding rules for monmux. Apply whenever writing,
  editing, or reviewing any .go file in this repository — new functions, new files,
  bug fixes, refactors, test additions. The rules here are enforced by golangci-lint
  (version 2, default: all linters) and the pre-commit hooks. Violations require
  manual fixup after the fact, so internalise them up-front instead. Use this skill
  proactively: consult it before generating Go code, not after lint fails.
---

# Go Style Guide — monmux

Rules derived from `.golangci.yaml` (golangci-lint v2, `default: all`) and verified
against the existing codebase in `cmd/` and `internal/`. §1–§16 follow from the linters
and the build, §17–§19 are review rules the linters cannot check.

golangci-lint is not on `PATH` in every environment; run it through pre-commit
(`pre-commit run golangci-lint-full --all-files`). That lints only the host's backend
(§10): after touching a tagged file or anything shared, also run `make go-lint-cross`,
which lints both. Never commit with `--no-verify` — fix the underlying issue instead.

Linters that are **disabled** in `.golangci.yaml`, so their rules do not apply:

| Disabled linter | Reason |
| --- | --- |
| `exhaustruct`, `exhaustruct_v5` | Requires every struct field to be set; too noisy for short-lived structs |
| `gomodguard` | Replaced by `gomodguard_v2`, which is enabled |
| `gochecknoglobals` | Package-level lookup tables (`kinds`, `grades`, `busPattern`) are intentional |
| `nonamedreturns` | Named returns are allowed |
| `wsl` | Whitespace style is enforced by `gofumpt` instead |

The formatters (`gci`, `gofmt`, `gofumpt`, `goimports`, `golines`) run in the
`golangci-lint-fmt` hook and in CI.

---

## 1. Import grouping

Three groups, separated by blank lines — this is enforced by both `gci` (explicit
`sections: standard, default, prefix(github.com/leinardi/monmux)`) and `goimports`
(`local-prefixes: github.com/leinardi/monmux`) simultaneously, and they must agree:

```go
import (
    // Group 1: stdlib
    "context"
    "errors"
    "fmt"

    // Group 2: third-party (everything that is NOT this module)
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"

    // Group 3: local module (github.com/leinardi/monmux/...)
    "github.com/leinardi/monmux/internal/catalog"
    "github.com/leinardi/monmux/internal/edid"
)
```

Within each group imports are sorted alphabetically. A blank line between groups
is required; no blank lines within a group. Getting this wrong triggers both
`gci` and `goimports`.

---

## 2. Error handling

### 2a. No inline error assignment in `if` (`noinlineerr`)

**Wrong:**

```go
if err := doSomething(); err != nil {
```

**Right:**

```go
err := doSomething()
if err != nil {
```

When a variable is already declared in the same scope, use `=` not `:=` for
the second and later assignments:

```go
err := firstThing()
if err != nil { ... }
err = secondThing() // = not :=
if err != nil { ... }
```

### 2b. Wrap errors with `%w` (`errorlint`)

Always wrap errors so callers can use `errors.Is`/`errors.As`:

```go
return "", fmt.Errorf("locating the home directory: %w", err)
```

The prefix is a short, lowercase phrase naming the operation, not a full
sentence: no capital letters, no trailing period.

Use `errors.Is` and Go 1.26's `errors.AsType[T]` (`errors.AsType[*refusal.Refusal](err)`)
for comparisons, never `==` on error values — `err113` flags that too.

A refusal is the exception to wrapping: pass it up unchanged, because wrapping
would corrupt the message the user sees (the `//nolint:wrapcheck` sites in
`internal/app` say so).

Prefer flat code with early returns; no `else` after a `return` (`revive`'s
`indent-error-flow`).

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

Use `errors.Join`:

```go
var problems []error
for _, kind := range kinds {
    if bare[kind] && numbered[kind] {
        problems = append(problems, fmt.Errorf("%w: %q", ErrMixedNumbering, kind))
    }
}
return errors.Join(problems...)
```

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

The explanation says why the fix does not apply here, not which rule fired (§18).

### On the line before a statement

```go
//nolint:gosec // an EDID product code is 16 bits by definition
ProductCode: uint16(decimal(b.fields[fieldModel])),
```

### Preceding-line — for a function or type declaration

```go
// Ready records the call and returns the scripted error.
//
//nolint:gocritic // hugeParam: the Backend interface passes a Display by value
func (f *Fake) Ready(_ context.Context, _ Display) error {
```

### Multiple linters — comma-separated, no spaces

```go
//nolint:gocyclo,cyclop,gocognit // <why the branches cannot be split>
```

Always name all linters that fire. If `gocyclo` AND `cyclop` both fire for a
complex function, suppress both. Same for `gocyclo`/`cyclop`/`gocognit` when
all three exceed their thresholds.

---

## 4. Complexity limits

| Linter | Threshold | Note |
| --- | --- | --- |
| `gocyclo` | 15 | Cyclomatic complexity |
| `cyclop` | 15 | Same metric, different linter — both fire together |
| `gocognit` | 35 | Cognitive complexity |
| `funlen` | 50 statements | Lines are disabled (`lines: -1`) |

Prefer extracting helpers over suppressing. When suppression is the right call
(e.g., a function that branches over many independent config fields), explain
why in the nolint comment.

These limits apply to test files too: `.golangci.yaml` exempts tests only from
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

`strings.SplitN` is excluded from mnd checks.

Test files (`_test.go`) are fully exempt from `mnd`.

---

## 6. Type aliases

Use `any` instead of `interface{}`. `gofmt` rewrites `interface{}` → `any`
automatically, but write `any` in new code to avoid the formatter changing
your diff.

---

## 7. Struct size (`gocritic hugeParam`)

Structs passed by value that are over ~80 bytes trigger `hugeParam`. Pass by
pointer instead — or add `//nolint:gocritic // <interface constraint reason>`
when the signature is fixed by an interface (the `Backend` interface passes a
`Display` by value).

The same applies to `rangeValCopy`: iterate large slices by index and take a
pointer (`item := &items[idx]`).

---

## 8. Line length (`lll`)

Max 140 characters. `golines` wraps automatically, but try to stay within
bounds when writing new code — especially long function signatures and struct
tags. Test files are exempt.

---

## 9. Forbidden packages (`depguard`)

| Forbidden | Use instead |
| --- | --- |
| `github.com/pkg/errors` (rule `forbidden`) | stdlib `errors` + `fmt.Errorf(...%w...)` |
| `github.com/instana/testify` (rule `forbidden`) | `github.com/stretchr/testify` |

---

## 10. Build tags

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

## 11. Comments and `godox`

- `FIXME` is flagged by `godox`. Do not leave `FIXME` comments in committed code.
- `TODO` is allowed.
- Comment style: gocritic's `whyNoLint` check is disabled, but all `//nolint`
  directives still need an explanation per nolintlint's `require-explanation` setting.

Doc comments:

- Every exported function, type, and variable has a doc comment beginning with
  the symbol name (`// Redacted returns a copy with both serial fields zeroed.`).
- Unexported symbols get one when their purpose is not obvious from the name.
- Inline comments explain *why*, not *what* (§18).
- Do not add doc comments or comment scaffolding to code you did not otherwise
  change, e.g. as a side effect of a bug fix.

---

## 12. Duplication (`dupl`)

Avoid copy-pasting blocks longer than ~100 tokens. Extract shared logic into a
helper. Test files are exempt from `dupl`.

---

## 13. Shadowing (`govet shadow`)

`govet` shadow detection is enabled. Avoid re-declaring variables with `:=`
when they shadow an outer-scope variable. Prefer distinct names or
restructuring to avoid shadows.

---

## 14. Variable naming (`varnamelen`)

Short variable names are fine in tight scopes (loop indices `i`, `k`, map
values `v`). `varnamelen` flags a name shorter than 3 characters whose last use
is more than 5 lines from its declaration (its defaults: `min-name-length: 3`,
`max-distance: 5`). Test files are exempt.

**Specific rules that bite most often:**

- **Receivers are exempt**: `(b *Backend)`, `(f *Fake)`, `(m Model)` — all fine.
- **Parameters are checked like locals.** A one-letter parameter passes in a
  three-line function and is flagged as soon as the body grows, so give
  parameters ≥ 3-char descriptive names from the start:

  ```go
  // Wrong — 'b', 'c', 'k' are too short for params
  func Parse(b []byte) (Identity, error)
  func ResolveTool(b, c, k string) (string, error)

  // Right
  func Parse(raw []byte) (Identity, error)
  func ResolveTool(binary, configured, configKey string) (string, error)
  ```

- **Local variables** follow the same distance rule: a variable named `c` that
  is still used more than 5 lines later is flagged; rename it to reflect its type
  or role.

Rule of thumb: if the name alone doesn't tell you what the variable holds,
make it longer.

---

## 15. `modernize` — no pointer-boxing helpers

The `modernize` linter (`newexpr` check) flags any function whose sole purpose
is to return a pointer to its argument — the generic `func ptr[T any](v T) *T`
included — at the declaration and at every call site. Go 1.26's `new` takes an
expression, so no helper is needed:

```go
// Wrong — flagged twice
func boolPtr(b bool) *bool { return &b }
cases := []struct{ enable *bool }{{enable: boolPtr(true)}}

// Right
cases := []struct{ enable *bool }{
    {enable: new(true)},
    {enable: new(false)},
    {enable: nil},
}
```

Taking the address of a local (`val := computeSomething()`, then `&val`) is fine
too, and reads better when the value is computed or used more than once.

---

## 16. Constant strings (`goconst`)

String literals appearing 3+ times with length ≥ 2 should be extracted to a
named constant. Test files are exempt.

---

## 17. Reuse before writing

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

## 18. Comments carry rationale; history goes in the commit

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

## 19. Waiting in tests

monmux tests have no `time.Sleep`, no goroutines and nothing to wait for: they drive
`exec.Fake` and `backend.Fake` synchronously. Keep it that way. If a test ever has to wait, decide
what kind of wait it is *before* writing it:

- **Positive eventual** ("something will have happened") — never a sleep: wait on the signal, or
  poll with a deadline and fail naming the condition that never held.
- **Negative assertion** ("nothing happens"), **real elapsed window** (the duration is under test),
  **ordering barrier with no quiescence signal** — a sleep, with a comment saying which of these it
  is; name the duration as a constant or a multiple of the interval under test.

A sleep whose comment says "give X time to Y" where Y is observable, and a sleep added to make a
flaky test pass, are always wrong.

---

## What to avoid

- `pkg/errors` (§9).
- `log.Fatal`, or `os.Exit` anywhere but a `main` function: `main()` calls `os.Exit(run(...))`
  exactly once so deferred cleanup always runs.
- `os/exec` in a test, or anything else that could run a real `ddcutil` or `m1ddc` (§17).
- OS-independent logic in a build-tagged file (§10).
- `interface{}` (§6) and pointer-boxing helpers (§15).
- Designing for hypothetical requirements: no configurability, abstractions or helpers for
  features that do not exist yet.
- Skipping or suppressing pre-commit hooks (`--no-verify`).
- Adding comments to code you did not change (§11).

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
- [ ] Checked §17 for an existing helper before writing a new one
- [ ] Comments say why, not what changed — history is in the commit body (§18)
- [ ] No `time.Sleep` in tests (§19)
