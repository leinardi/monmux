# Release

> **Status: design only. None of this is implemented.** There is no release pipeline, no CI workflow and no published
> artifact yet; monmux is built from source with `make go-build`. This page is the plan, written down so the shape is agreed
> before anything is automated, and so nobody assumes a release exists because a document mentions one.

## What a release should produce

- Static binaries for `linux/amd64`, `linux/arm64` and `darwin/arm64`.
    - `darwin/amd64` is deliberately absent: the macOS backend wraps `m1ddc`, which requires Apple Silicon. Building an Intel
    binary that can never work would be a promise monmux cannot keep.
- A `checksums.txt` covering every artifact, so a downloaded binary can be verified. `monmux doctor` already prints the SHA-256
  of the tool it runs; the same idea applies to monmux itself.
- `.deb` and `.rpm` packages for the Linux binaries, via [nfpm](https://nfpm.goreleaser.com/), declaring a dependency on
  `ddcutil` (>= 2.2).
- A Homebrew formula in a tap, declaring a dependency on `m1ddc`. The README currently says Homebrew arrives with the release
  pipeline; this is that promise.
- Release notes generated from the commit history, with a section that names any change to the catalog explicitly. A release
  that changes which bytes reach which monitor must say so at the top.

## goreleaser

`goreleaser` builds the matrix, the checksums, the packages and the tap from one configuration file. The build needs
`CGO_ENABLED=0` and the same `-ldflags` the Makefile already uses, so that `monmux version` reports the tag, the commit and the
build date rather than `dev`.

```text
builds:      linux/amd64, linux/arm64, darwin/arm64; CGO_ENABLED=0
ldflags:     -s -w -X main.version={{.Version}} -X main.commit={{.FullCommit}} -X main.date={{.Date}}
archives:    tar.gz, with LICENSE and README.md
checksum:    checksums.txt (sha256)
nfpms:       deb + rpm, dependency ddcutil >= 2.2
brews:       tap formula, dependency m1ddc
```

The version variables are already wired: `cmd/monmux/version.go` declares `version`, `commit` and `date`, and `.mk/go.mk` fills
them through `GO_LDFLAGS`.

## CI jobs to port

The jobs come from the reference repository this project's tooling was taken from, minus everything container-related — monmux
has no Dockerfile, no image and no `hadolint` or `dclint` hooks.

| Job           | What it does                                                                                                                                   |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `pre-commit`  | Runs the whole hook suite on every pull request, the same one `make check` runs — on a Linux **and** a macOS runner, see below.                |
| `test`        | `go test -race ./...`, on both runners: a backend's build-tagged tests only ever execute on their own OS.                                      |
| `cross`       | `make go-build-cross`, `make go-vet-cross` and `make go-lint-cross`, which cover both OS backends from either runner.                          |
| `govulncheck` | Vulnerability scan of the module.                                                                                                              |
| `release`     | `goreleaser release` on a tag, publishing the artifacts and updating the tap.                                                                  |

Three things are worth doing deliberately:

- **Run `pre-commit` and `test` on both a Linux and a macOS runner.** Each backend sits behind a build tag, so a linter running
  on one OS does not analyse the other's files at all, and three real findings in `internal/backend/m1ddc` sat unreported until
  the first `make check` was run on a Mac. The `cross` job narrows the gap but does not close it: `make go-lint-cross` depends
  on `golangci-lint` being built with a Go no older than the toolchain, and aborts in `GOROOT` — reporting nothing about this
  repository — when it is not, which is a bad thing to have as the only line of defence. Running each linter natively on its own
  OS has no such dependency. The macOS runner is also the only place the darwin-tagged unit tests execute; it needs no `m1ddc`
  installed, because the suite executes no external binary — [testing.md](testing.md).
- **Warm the pre-commit cache.** The hook suite installs golangci-lint, markdownlint and the rest on first run; caching
  `~/.cache/pre-commit` keyed on `.pre-commit-config.yaml` turns a slow job into a fast one.
- **Do not add a job that runs monmux against hardware.** There is no monitor in CI, and the rule that no automation writes to a
  monitor is not suspended because the automation is called a workflow. Hardware validation is a human checklist —
  [testing.md](testing.md).

## Versioning

Semantic versioning, with one project-specific rule: **any change to `internal/catalog/models.yaml` that enables a model or an
input is at least a minor release, and is called out in the release notes.** Somebody deciding whether to upgrade needs to know
that the new version is willing to write something the old one refused.

## Before the first release

- [ ] A CI workflow, with the jobs above, running `pre-commit` and `test` on both a Linux and a macOS runner.
- [ ] A `goreleaser` configuration, verified with `goreleaser release --snapshot --clean`.
- [ ] A tap repository for the Homebrew formula.
- [ ] Hardware validation completed and recorded for every model the release enables.
- [ ] `SECURITY.md` updated: it currently says only `main` is supported, which stops being true the moment there is a tag.
