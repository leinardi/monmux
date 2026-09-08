# Testing

## The rule

**No AI agent — an assistant in an editor, a coding agent, a subagent, a review bot, any tool-driven automation — may run a
command that writes to a monitor.** Only a human runs a writing command, by hand, deliberately.

**Tests never write to a monitor, and never execute any external binary at all.** Not `ddcutil`, not `m1ddc`, not a stub script,
not `--version`. Every test in this repository runs against a fake.

Both halves of that are enforced in code, not left to discipline.

## How the no-exec rule is enforced

`internal/backend/exec` is the only package in monmux that starts a process, and the real runner refuses to:

```go
func ExecutionDisabled() error {
    if testing.Testing() {
        return fmt.Errorf("%w: running under a test binary", ErrExecutionDisabled)
    }

    if os.Getenv(NoExecEnv) == "1" {
        return fmt.Errorf("%w: %s=1 is set", ErrExecutionDisabled, NoExecEnv)
    }

    return nil
}
```

The check is inside `Run`, not at construction time, so no code path can hold a runner that was built before the check. A test
that constructs the real runner and asks it to run something gets an error, not a subprocess. `MONMUX_NO_EXEC=1` is the same
guard for scripts and one-off runs, and it is useful for looking at monmux's behaviour on a machine you would rather it did not
touch.

From the outside, `TestNoTestFileCanStartAProcess` walks every `_test.go` in the repository and fails the build if one imports
`os/exec` or so much as names the real runner's constructor. Two files are allowed to name the constructor — the guard's own
test and the runner's — and only while they still assert that execution is refused.

## What the tests are

**Unit tests, everywhere.** The pure layers — `edid`, `catalog`, `policy`, `refusal`, `config` — are tested directly. The
backends are tested through a recording fake runner that answers from captured output and records what it was asked to run.

**Golden command tests.** Each backend has a test pinning the exact arguments it produces for the catalog's model, byte for
byte:

```text
ddcutil --edid <256 hex> setvcp 0xF4 0xD1 --i2c-source-addr=0x50 --noverify
m1ddc display <UUID> set input-alt 209
```

These exist so that a refactor cannot quietly change what reaches a monitor. If the bytes change, the test fails and somebody
has to say why in a commit message.

**Golden CLI output.** `info`, `info --json`, `doctor`, both dry-run renderings and the success sentence are compared exactly,
including the redaction. Output is an interface too.

**Invariant tests.** The catalog's rules are tested rather than trusted: a write-enabled model has at least one identity, an
identity belongs to one model, every recorded input has evidence, a disabled model never matches, the zero operation is
invalid. `TestCompatibilityDocumentMatchesTheCatalog` compares `docs/compatibility.md` against the catalog itself, and
`TestGeneratedCatalogMatchesTheYAML` compares the catalog against `internal/catalog/models.yaml`, so a hand-edit of the
generated Go fails the build instead of shipping. The `go-test-repo-mod` pre-commit hook is filtered to Go files and `go.mod`,
which would let a commit that edits only the catalog file skip that test; `.pre-commit-config.yaml` widens the filter to include
`models.yaml` so an edit without a regeneration cannot be committed. The generator has its own tests for every rule it enforces, and
`TestGeneratorKnowsEveryInputAndMechanism` keeps its copy of the input and mechanism enums in step with the real ones.

**Structural tests.** A reflection test walks the `Backend` interface and fails if any method could be handed a `Command`,
however deeply wrapped — that is what makes a command unforgeable.

**Cross-compilation.** Only one of the two backends is visible to the host toolchain at a time, so `make go-build-cross` and
`make go-vet-cross` run it over `GOOS=linux` and then `GOOS=darwin`, type-checking each backend's build-tagged tests too. Both
are named outright rather than derived as "the OS the host is not": `go env GOOS` reports the target rather than the host, and
`go-build`/`go-vet` set no `GOOS` of their own, so any value derived from one of them has a case where both aim at the same OS
and a backend is left checked by nothing at all. A backend's parser and decisions carry no build tag, so they run in
`make go-test` like everything else.

`make go-lint-cross` does the same for the linter, and it is the one that matters: lint is where the two backends diverge most,
and three findings in `internal/backend/m1ddc` went unreported for as long as the linter only ever ran on Linux. `make check`
still covers the host's backend alone, so run the cross target before pushing a change to either one.

It is also the one target that can fail for reasons outside this repository. When cross-targeting, `golangci-lint` typechecks
the standard library from source using the `go/types` it was built with; if that is older than your toolchain it aborts inside
`GOROOT` and then reports **nothing at all** about this repository — a violation planted in a build-tagged file is reported
nowhere. v2.12.2, built with go1.26.5, does exactly that against a Go 1.27 toolchain; v2.13.2, built with go1.27.0, is clean.

If the target fails inside `GOROOT` rather than inside monmux, your linter is older than your toolchain: update
`.pre-commit-config.yaml`, or skip the target for now. **Never silence that typecheck error with a path exclusion.** It does not
restore the analysis, it only hides the abort, and the run then goes green having checked nothing. Running the linter natively
on each OS in CI is still the durable answer — [release.md](release.md).

## Fixtures

Fixtures are captures of real hardware, so they arrive carrying somebody's serial number. Every identifier is replaced before
the fixture is committed, with the synthetic values documented in `internal/backend/testdata/README.md`:

| Identifier                | Value                                  |
| ------------------------- | -------------------------------------- |
| EDID numeric serial       | `0x01020304`                           |
| EDID serial string (0xFF) | `TESTSERIAL01`                         |
| m1ddc alphanumeric serial | `TESTSERIAL01`                         |
| m1ddc binary serial       | `16909060 (0x01020304)`                |
| macOS display UUID        | `00000000-0000-4000-8000-00000000000N` |

`TestFixturesCarryOnlySyntheticSerials` reads every file under every `testdata` directory in the repository, at any depth, and
fails if any other serial or UUID appears. The values are format-valid rather than obviously fake, because a parser must accept
them exactly as it accepts the real thing.

For an EDID, recompute the block-0 checksum after editing: the 128 bytes must sum to zero modulo 256 or the parser rejects the
fixture. The whitespace hooks skip `testdata/`, so a capture keeps the trailing spaces and missing final newline the real tool
produced.

## Running them

```sh
make go-test          # go test -race ./...
make go-build-cross   # compile-check both OS backends, whatever the host
make go-vet-cross     # cross-vet both, including their build-tagged tests
make go-lint-cross    # lint both, which `make check` does not
make check            # everything pre-commit runs, for the host OS only
```

## Hardware validation — for a human, by hand

Automated tests prove that monmux builds the command it means to. They cannot prove a monitor accepts it. That is done once per
model, by a person, and the result becomes the evidence in the catalog.

Run these yourself. Do not delegate them.

### Linux, on DisplayPort

1. `monmux doctor` — every check passes; the `ddcutil` version is at least 2.2 and the options are present.
2. `monmux info --json` — one display with:
   - `identity.manufacturer` `GSM`, `identity.productCode` `30675` (`0x77D3`)
   - `model` `LG 38WR85QC-W`
   - `enabledInputs` `["dp", "usb-c"]`
   - `writable` `true`, `status` `ok`
3. `monmux switch usb-c --dry-run` — nothing is sent, and the `Command:` line reads exactly:

   ```text
   /usr/bin/ddcutil --edid <redacted; --show-serial to print> setvcp 0xF4 0xD1 --i2c-source-addr=0x50 --noverify
   ```

4. `monmux switch usb-c` — the monitor switches to USB-C. Exit code `0`, and the message says the switch was not independently
   confirmed.

### macOS, from USB-C

Run this later, from the machine on the other input.

1. `monmux doctor` — `m1ddc` is found and the display list parses.
2. `monmux info --json` — one display with `identity.manufacturer` `GSM`, `identity.productCode` `30676` (`0x77D4`, the code the
   unit reports while on USB-C), and a `handle` that is masked by default and is a UUID under `--show-serial`.
3. Capture `m1ddc display list detailed`, sanitize it as above, and commit it as a fixture if it differs from the one in
   `internal/backend/m1ddc/testdata/`.
4. `monmux switch dp --dry-run` — nothing is sent, and the `Command:` line reads
   `/opt/homebrew/bin/m1ddc display <redacted; --show-serial to print> set input-alt 208`.
5. `monmux switch dp` — the monitor switches back to DisplayPort.

### Negative cases, on either platform

1. `monmux switch hdmi1` — refused with `input-not-enabled`, exit code `2`, and nothing was written. HDMI was never tested on
   this unit and must stay refused.
2. `monmux switch usb-c --serial NOTMYSERIAL` — refused with `serial-mismatch`, exit code `2`.

Record what you ran, on what date, on which OS, and what the monitor did. That paragraph is the evidence string in the catalog
— see [adding-a-monitor.md](adding-a-monitor.md).
