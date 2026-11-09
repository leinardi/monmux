# Release

How a monmux release is built, what it produces, and what to do when one fails halfway. The pipeline
is [`.github/workflows/release.yaml`](../.github/workflows/release.yaml) and
[`.goreleaser.yaml`](../.goreleaser.yaml); this page is why it is shaped that way.

One rule frames all of it: **no job runs monmux against hardware.** There is no monitor in CI, and
the rule that no automation writes to a monitor is not suspended because the automation is called a
workflow. Nothing in either workflow executes `ddcutil`, `m1ddc` or `monmux switch`; the test suite
refuses to start a process by construction, so a runner needs neither tool installed. Hardware
validation is a human checklist — [testing.md](testing.md).

## What a release produces

- Static binaries for `linux/amd64`, `linux/arm64` and `darwin/arm64`, each with `-trimpath`,
  `CGO_ENABLED=0` and the version, commit and build date compiled in, so `monmux version` reports
  the tag rather than `dev`.
  - `darwin/amd64` is deliberately absent: the macOS backend wraps `m1ddc`, which requires Apple
    Silicon. Building an Intel binary that can never work would be a promise monmux cannot keep.
- A `tar.gz` per target, carrying the binary, `LICENSE` and `README.md`.
- `checksums.txt` covering every artifact, the packages included, and `checksums.txt.sigstore.json`:
  a keyless [cosign](https://docs.sigstore.dev/) signature made with the release workflow's own OIDC
  identity. There is no private key anywhere. `monmux doctor` already prints the SHA-256 of the tool
  it runs; the same idea applies to monmux itself.
- A [build provenance
  attestation](https://docs.github.com/actions/security-guides/using-artifact-attestations) over the
  archives, the packages and the checksum file, verifiable with `gh attestation verify`.
- `.deb` and `.rpm` packages, built by [nfpm](https://nfpm.goreleaser.com/), published to
  [Cloudsmith](https://cloudsmith.io/~leinardi/repos/monmux/) as well as attached to the GitHub
  release.
- A [Homebrew **cask**](https://docs.brew.sh/Cask-Cookbook) in `leinardi/homebrew-tap`, depending on
  the `m1ddc` formula.
- Release notes: goreleaser's grouped changelog, under a header generated from
  `internal/catalog/models.yaml` itself, which names every catalog change the release makes.

### A cask, not a formula

goreleaser's `brews:` was soft-deprecated in v2.10 and hard-deprecated in v2.16; `homebrew_casks:`
replaced it. That suits monmux: a cask is macOS-only by nature, which is the only platform the tap
needs to serve, and it takes `dependencies: [{formula: m1ddc}]` with no OS guard.

It has one consequence worth stating plainly, because it is a security decision rather than a
packaging detail: Homebrew quarantines everything a cask downloads, and monmux's binaries carry no
Apple Developer ID signature, so without intervention Gatekeeper refuses to run them at all. The
cask therefore strips `com.apple.quarantine` in a `postflight_steps` block. It is written as a
`custom_block` because goreleaser's `hooks` still render Homebrew's deprecated Ruby `postflight`
block, which makes every `brew` command warn. What the strip bypasses, and how to install without
it, is in [security.md](security.md#the-install-path).

### The package dependency is unversioned

The packages declare `Depends: ddcutil` and `Requires: ddcutil` with **no version**, and the package
description names the 2.2 floor. This is deliberate, and it is a trade, so both sides are written
down.

monmux needs `ddcutil` 2.2. Distributions today ship (as of September 2026): Debian 12 `1.4.1`,
Debian 13 `2.2.0`, Ubuntu 24.04 `1.4.1`, Ubuntu 26.04 `2.2.5`, Fedora 42 `2.1.4`, Fedora 43 `2.2.1`,
Arch and Tumbleweed `2.2.7`, openSUSE Leap 15.6 `1.4.1`.

- A hard floor would be honest at install time, but `apt` would refuse the package outright on
  Debian 12 and Ubuntu 24.04 — the two most common targets — including for somebody who built
  `ddcutil` 2.2 into `/usr/local`, which `dpkg` cannot see. There is no way past that short of
  `--force-depends`.
- The unversioned dependency installs everywhere. `monmux info` still works, because enumeration
  reads sysfs and needs no tool at all, and anything that needs the tool refuses with
  `backend-not-ready` naming the version it found and the version it needs. `monmux doctor` says the
  same thing before you try. A user is never left with a mystery, only with a version to upgrade,
  and the README says up front that Debian 12 and Ubuntu 24.04 ship one that is too old.

The floor is enforced where it can be checked properly — in preflight, against the binary that is
actually installed — rather than by a package manager guessing from a version string.

## How a release runs

One `workflow_dispatch` on `main`, with two inputs: `version` (optional, `X.Y.Z` or `vX.Y.Z`) and
`dry_run` (boolean).

**One release runs at a time.** The workflow takes a `release` concurrency group with
`cancel-in-progress: false`, so a second dispatch queues behind the first rather than racing it. Two
runs in flight together would each check their version against the same latest tag, both pass, and
then publish in whatever order they finished — a lower version landing last would take `latest` with
it. Nothing is ever cancelled either: a run killed between the tag push and the Cloudsmith upload is
the one state this pipeline has to be recovered from by hand.

### Dependabot releases

A merge of a Dependabot pull request releases the next patch on its own, with no dispatch, when
everything since the last release is Dependabot's and at least one of its commits is a `fix(deps)`
(a Go module update). The workflow also runs on every push to `main`: its `dependabot` job calls
gh-reusable-workflows' `dependabot-release` with `ecosystems: gomod`, which checks from the API
alone that every unreleased commit is Dependabot's own (committed by GitHub) or the merge of its
pull request, that the change only moves action pins and the versions of those ecosystems' files,
and that CI passed on the push. Then the release runs as a dispatch would, with that patch as an
explicit version. Anything else is a green no-op with a notice: a `chore(deps)` merge (actions,
pre-commit, development dependencies) ships nothing, and a human commit waiting to be released
leaves the release to a dispatch.

A push that merges a Dependabot pull request takes the same `release` group as a dispatch, so the
two never publish at once and a lower version never moves `latest` after a higher one. Every other
push runs in a group of its own and only skips the `dependabot` job. GitHub keeps one pending run
per group: a Dependabot merge that arrives while a release runs and a dispatch waits replaces that
dispatch, which shows as cancelled; dispatch it again.

The job order exists because **tags in this repository are immutable**. The "Immutable tags" ruleset
covers `~ALL` tags except `v[0-9]` and `v[1-9][0-9]`, blocks update, deletion and non-fast-forward,
and has no bypass actors, so the familiar "delete the tag on failure" cleanup step cannot work here
— it would fail every time. Everything that can fail without a tag therefore happens before the tag
exists, and everything after it is idempotent.

1. **Refuse anything but `main`.** `workflow_dispatch` lets the caller pick any branch, and the
   release job holds `contents: write`, an OIDC identity and the tap token — dispatched from a side
   branch it would tag that branch's tree and publish it. The `guard` job fails, rather than skips,
   when `github.ref` is not `refs/heads/main`, and `ci-green` and the release job wait on it, so
   such a run is red and nothing is built or published (only the read-only `audit` runs anyway).
   The ruleset cannot express this: tag protection stops a tag from *moving*, not from being created
   on the wrong commit. The second line of defence is the `release` environment — add a deployment
   branch rule limiting it to `main`, and required reviewers if a release should need a second pair
   of eyes.
2. **Require CI green, and audit again.** The release does not repeat what CI already proved. The
   `ci-green` job runs `require-ci-green` from
   [`leinardi/gh-composite-actions`](https://github.com/leinardi/gh-composite-actions): `ci.yaml`'s
   `ci-result` must have passed on the push of this exact commit to `main`, in that run's newest
   attempt, or the release fails without building anything. Only the time-sensitive check runs
   again: the `audit` job runs `make audit-deps` (govulncheck), because advisories are published
   every day and the dependencies CI scanned may have one now. It needs nothing and nothing needs
   it, so it runs beside the build; step 8 waits for it. [What CI proves for a
   release](#what-ci-proves-for-a-release) lists what moved where.
3. **Resolve the version**, with gh-composite-actions' `release-version`. Only strict
   `vMAJOR.MINOR.PATCH` tags are releases, `^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`
   (`RELEASE_TAG_REGEX` in the workflow is the same regex), with no leading zeros, no pre-release and
   no build metadata. An explicit input must match it. Otherwise the highest strict tag reachable
   from `HEAD` is the current version, and `svu next` is given that exact tag as `--tag.pattern`,
   never a glob (a glob such as `v[0-9]*.[0-9]*.[0-9]*` also matches `v1.2.3-rc.1`). svu is
   installed (`setup-svu`) only when the version is derived, so a version typed by hand releases
   even while GitHub's release downloads fail. With no strict tag reachable, a run with no version
   fails and asks for one. `svu` prints the current version unchanged, and exits 0, when nothing
   since that tag is a `feat`, a `fix` or a breaking change, so equality is the "nothing to bump"
   case and it fails with *"No feat/fix or breaking-change commit since `<tag>`, so there is nothing
   to bump. Re-run with an explicit version to force one."* — unless that tag is on `HEAD` itself,
   which is a re-run of a run that derived its version.

   The version must then be **higher than every version already released**, whether it was derived
   or typed. Tags here are immutable and the release is marked latest, so publishing a `v1.1.1`
   after `v1.2.0` would create a permanent lower tag and hand everyone following the "latest" link
   an older binary. The only version a run may reuse is the one it is recovering, and only while it
   is still the highest release: goreleaser marks every release it publishes as latest and pushes
   its cask to the tap, so re-running an older version would hand both an older binary.
   `release-version` allows such a recovery, as the sibling repositories do; here a step right after
   it refuses one. A branch whose history does not contain the
   highest release — for example one tagged on a merge commit that was later rewritten — derives
   from an older base and so fails that check; the failure names the out-of-history tag, and an
   explicit version higher than it is the way out.

   **A recovery is exactly this version already sitting on `HEAD`** — not "`HEAD` carries some tag".
   The wider test would misread a release cut from a commit that is already tagged with the
   *previous* version, and would then read the catalog baseline from one release further back. Every
   tag lookup in the step filters the full tag list through `RELEASE_TAG_REGEX` rather than taking
   the nearest tag, because `latest` is a separate pointer that the ruleset deliberately leaves
   mutable.

   The baseline for the notes is the previous release tag (`release-version`'s `previous`), read from
   `HEAD^` on a recovery, since the tag being released is on `HEAD` already and describing from
   there would compare the catalog against itself.
4. **Write the catalog notes**, and apply the catalog version rule below.
5. **Check the tag.** If `refs/tags/<version>` already exists on the remote, the run continues only
   when it points at `HEAD` — a re-run recovering a failed publish. Pointing anywhere else is a hard
   failure, and so is a `git ls-remote` that fails: only an answer that names no such ref reads as
   absent.
6. **Tag locally.** `git tag -a`, and nothing is pushed. goreleaser only checks that the tag exists
   locally on `HEAD` (`git describe --exact-match`), never on the remote, so building against a
   local-only tag is supported. That is also why the changelog uses `use: git`: the GitHub compare
   API would 404 on a tag it cannot see.
7. **`goreleaser check`**, then **`goreleaser release --clean --skip=publish`**: builds, archives,
   packages, checksums and signs, with nothing pushed anywhere. A configuration error, a build error
   or a cosign error stops the run here, with no tag on the remote and nothing to clean up.
8. **Require the audit, then gate the tag.** `require-job-success` waits for this run's `audit` job
   and fails unless it passed in the current attempt. Then the highest released version is read
   again, from the **remote** this time: concurrency serializes this workflow, but a tag can arrive
   from outside it, and the check in step 3 ran several minutes of building ago. A version that is
   no longer the highest fails here, with nothing pushed. The `git ls-remote` is its own assignment
   under `set -e`, so a failed lookup stops the step instead of reading as "no tags". A dry run
   makes both checks, so it fails wherever a real run would.
9. **A dry run stops here**, having written the release notes and the artifact list to the job
   summary and uploaded `dist/` as a workflow artifact. This is the review step: the version is
   resolved, the notes are rendered and everything is built, without a tag, a release, a cask commit
   or a Cloudsmith upload.
10. **`git push origin <version>`** — the point of no return. The checkout keeps no credentials
    (`persist-credentials: false`); the token reaches this one `git` command as a header.
11. **`goreleaser release --clean`** again: it rebuilds and publishes — creating the GitHub release
    with `target_commitish: {{ .FullCommit }}`, uploading every artifact and pushing the cask to the
    tap. Rebuilding is safe because the build is reproducible: everything stamped into an artifact
    comes from the commit rather than from the clock — `.Tag`, `.FullCommit`, `.CommitDate` and
    `mod_timestamp: {{ .CommitTimestamp }}` for the binaries, `nfpms.mtime` and a `file_info.mtime`
    per declared content for the packages — so the second build produces the same bytes as the one
    the dry run inspected. Verified by building the same commit twice and comparing SHA-256. The
    package half of that was not true for v0.5.0, whose deb and rpm carried the build clock and so
    differed between the two builds of the one release run; the archives always matched.
12. **Attest** the archives, packages and checksum file.
13. **`cloudsmith push deb|rpm ... --republish`** for each package.

### Recovering a failed publish

Dispatch the workflow again with the **same version, given explicitly**, or re-run the failed run
with **"Re-run all jobs"** — never "Re-run failed jobs". Only the current attempt's `audit` counts:
"Re-run failed jobs" does not re-run a job that passed, and `require-job-success` never trusts an
earlier attempt's pass, so the release job would wait for an audit that is not coming and fail.
Everything after the tag push is idempotent by construction:

- the tag check passes, because the tag points at `HEAD`, and the push step is skipped;
- `release.mode: replace` and `replace_existing_artifacts: true` refresh the GitHub release instead
  of appending a second copy of every asset;
- goreleaser skips a cask whose content has not changed;
- `--republish` overwrites the same package version on Cloudsmith.

There is no cleanup step, and the tag is never deleted. That is the stronger promise, not the weaker
one: a version that was ever published never points anywhere else.

If the release is wrong rather than incomplete — the wrong commit, a bad build — the answer is a new
version, not a moved tag.

## Versioning

Semantic versioning, derived from Conventional Commits by [`svu`](https://github.com/caarlos0/svu):
`feat` is a minor, `fix` is a patch, `!` or `BREAKING CHANGE:` is a major, and everything else
(`build`, `chore`, `ci`, `docs`, `perf`, `refactor`, `revert`, `style`, `test`) bumps nothing — so a
performance improvement, refactor or revert that users should receive is committed as a `fix` (or
`feat`). `svu` is a single binary that reads tags and commit messages and does nothing else —
goreleaser already renders the changelog, so a second changelog engine would be one more thing to
keep in step. gh-composite-actions' `setup-svu` installs it from its release archive, checked
against a SHA-256 pinned in the action, so its version moves with the action's pin, which dependabot
updates.

One project-specific rule sits on top: **any change to `internal/catalog/models.yaml` that adds or
moves a byte this binary can send to a monitor is at least a minor release, and is called out at the
top of the notes.** Somebody deciding whether to upgrade needs to know that the new version can send
something the old one could not.

"Can send" is wider than "is write-enabled", and deliberately so. An entry with
`write_enabled: false` is never matched and never written to by `monmux switch` — but
`catalog.Model.UnsafeOperation` ignores the flag, and that is exactly what `--unsafe-model` calls,
so recording a model or an input puts a value in the binary that the previous one had no way to send
at all. Recording one is therefore a minor too. The practical consequence: a release that adds
entries to the catalog is never a patch, even when every one of them arrives disabled.

What does *not* count: losing something (a model removed, an input dropped, a model or identity no
longer write-enabled) and record-keeping (evidence, per-model notes, sources). None of those adds a
byte.

Generated commit notes cannot know any of this, so a small tool does:
[`internal/catalog/internal/generate/cmd/relnotes`](../internal/catalog/internal/generate/cmd/relnotes).
It takes the previous release's `models.yaml` (from `git show <previous-tag>:...`) and the current
one, and prints the section that goes at the top of the release notes. The sections that count as
enabling come first — models newly write-enabled, inputs newly enabled, values changed on an enabled
input, identities added, models and inputs newly recorded, values changed on a recorded input — then
the ones that do not: identities removed, models no longer write-enabled, inputs no longer recorded,
models removed, and a last section naming any entry whose evidence, notes or sources moved. That
last one exists so that "no catalog change" can only mean the file is identical, rather than meaning
the bytes happen to match. It also prints `enabled=true|false`, which is what the workflow acts on:

- derived version, patch bump, `enabled=true` → the workflow raises it to the next minor after the
  previous release (what `svu minor` prints) and prints a notice;
- explicit version, patch bump, `enabled=true` → the workflow fails and names the minor to pass. An
  explicit input is a choice; the rule outranks it.

When nothing changed, the header still says so — *"No catalog change: this release writes exactly
what `<previous>` wrote."* — because silence would be indistinguishable from a step that did not
run.

The commit-message changelog is grouped underneath: `feat(catalog)` and `fix(catalog)` first under
**Catalog**, then **Features**, then **Bug fixes**, then everything else; `docs`, `chore`, `ci`,
`test`, `style` and merge commits are left out.

## What the workflows trust

**goreleaser is pinned to an exact version**, in `GORELEASER_VERSION`, rather than to `~> v2`. A
recovery re-run rebuilds a tagged commit and has to produce the same artifacts; a goreleaser
released in between could change them, which would defeat the reproducibility the rebuild depends
on. `ci.yaml` checks the configuration against the same pin, so `goreleaser check` validates against
the binary that will actually build the release. It is a human's job to bump.

**Every action is pinned to a full commit SHA**, with the version in a trailing comment, in every
workflow. A tag is a mutable pointer: `@v7` resolves to whatever that tag points at on the
day the job runs, and an action that moves — or whose repository is compromised — would execute
inside the release job, which holds `contents: write`, an OIDC identity good for Cloudsmith, and the
tap token. A SHA cannot be moved. Dependabot updates the pins and rewrites the comment, so this
costs nothing to maintain. `govulncheck` is pinned as a version, `GOVULNCHECK_VERSION` in
`.mk/audit-deps.mk` (the `make audit-deps` target both workflows and the pre-commit hook run),
because dependabot cannot see a `go run` argument; it is a human's job to bump. Pinning the
scanner does not pin what it knows about: govulncheck fetches the vulnerability database at run
time.

**The vulnerability scan runs on the toolchain the release builds with.** Both workflows call
`setup-go` with `go-version-file: go.mod` and then `govulncheck` directly, rather than using
`golang/govulncheck-action`. That action always forwards its own `go-version-input`, which defaults
to `stable`, and `setup-go` prefers it over `go-version-file` — so go.mod was ignored and the
standard-library half of the report described a binary nobody ships.

**Verification pins the signing identity, not just the repository.** The keyless cosign certificate
carries the workflow that produced it, so the documented command checks
`--certificate-identity 'https://github.com/leinardi/monmux/.github/workflows/release.yaml@refs/heads/main'`
rather than a regexp over the repository. An identity regexp matching any workflow in the repository
would accept a signature produced by any other workflow, which is a lower bar than it looks.
`gh attestation verify` gets the same treatment through `--signer-workflow` plus
`--source-ref refs/heads/main`: the workflow path alone would still accept a run of that file from
any branch, which is the very thing the `guard` job exists to prevent.

## CI

[`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml) runs on every pull request, on pushes to
`main` and on dispatch. A pull request's runs share one concurrency group, so a newer push cancels
the older run; every other run gets a group of its own and is never cancelled or replaced, because
the release requires CI to have passed on the push of the exact commit it releases.

| Job | Runner | What it does |
| --- | --- | --- |
| `pre-commit-all` | ubuntu | Every pre-commit hook on every file, minus `go-test-repo-mod` and `audit-deps-go`, which `test` and `govulncheck` run; fails if a hook changed or created a file. |
| `scripts` | ubuntu | The shell tests of the licence-header script, in scratch git repositories. |
| `test` | ubuntu + macos | `make go-test` (`go test -race ./...`) on both, so each backend's build-tagged tests run. Defined in [`go-test.yaml`](../.github/workflows/go-test.yaml), which the Go cache warm-up runs too. |
| `cross` | ubuntu | `make go-build-cross`, `make go-vet-cross`, `make go-lint-cross`. |
| `lint-macos` | macos | `golangci-lint run ./...` natively, the only look at `internal/backend/m1ddc` under its tag. |
| `govulncheck` | ubuntu | Vulnerability scan of the module, test files included. |
| `release-config` | ubuntu | `goreleaser check`. |
| `actionlint` | ubuntu, pull requests | Workflow linting, as inline review comments. |
| `pre-commit-hooks` | ubuntu, pull requests | The hook suite `make check` runs, minus the hooks with their own job and minus `go-test-repo-mod`. |
| `markdownlint` | ubuntu, pull requests | Markdown linting. |
| `shellcheck` | ubuntu, pull requests | Shell linting. |
| `yamllint` | ubuntu, pull requests | YAML linting. |
| `conventional-commits` | ubuntu, pull requests | Every commit message in the range, because the release version is derived from them. |
| `ci-result` | ubuntu | Needs every job above except the reviewdog annotation jobs, and fails unless each succeeded; `conventional-commits` may be skipped off pull requests. The one required check, and the job the release's `require-ci-green` reads. |

Four things there are deliberate:

- **`test` runs on both a Linux and a macOS runner.** Each backend sits behind a build tag, so a
  linter or a compiler running on one OS does not analyse the other's files at all, and three real
  findings in `internal/backend/m1ddc` sat unreported until the first `make check` was run on a Mac.
  The macOS runner needs no `m1ddc` installed, because the suite executes no external binary —
  [testing.md](testing.md).
- **The linter also runs natively on macOS**, rather than only through `make go-lint-cross`. The
  cross target narrows the gap but does not close it: it depends on `golangci-lint` being built with
  a Go no older than the toolchain, and aborts in `GOROOT` — reporting nothing about this repository
  — when it is not, which is a bad thing to have as the only line of defence. Both jobs read the
  linter version out of `.pre-commit-config.yaml`, so there is one source of truth and dependabot
  keeps it moving.
- **The pre-commit cache is warmed.**
  [`pre-commit-warmup.yaml`](../.github/workflows/pre-commit-warmup.yaml) refills
  `~/.cache/pre-commit` on every push to `main` that touches `.pre-commit-config.yaml`, under the
  key the reviewdog composite actions read. It matters more here than in the sibling repositories:
  the `golangci-lint` hook builds the linter from source on a cold cache.
- **Only `main` writes a cache.** A pull request can read the caches saved on `main`, and nothing a
  pull request saves is readable by `main` or by another pull request, so a save anywhere else
  would only spend the repository's cache quota. The Go module and build cache is saved by
  [`go-cache-warmup.yaml`](../.github/workflows/go-cache-warmup.yaml) alone: it runs
  `go-test.yaml` on both OSes on every push to `main` that touches Go code, `go.mod`, `go.sum`,
  `models.yaml`, test data or the build files (and weekly, so GitHub does not evict the entry), and
  saves one entry per OS keyed by commit, then prunes all but the newest two of each with
  gh-composite-actions' `prune-caches`. Every CI Go job sets `setup-go` up with `cache: false` and
  restores that entry, never saving; the keys fall back from the exact `go.sum` to the Go version to
  the OS, so a dependency bump still starts warm. The native macOS lint's own `golangci-lint`
  cache is saved on `main` only. The release restores and saves nothing: it builds from a clean
  cache.

The `pre-commit-hooks` job sets Go up before the composite action, because `go-generate-catalog` and
the `golangci-lint` hooks need a toolchain matching `go.mod` and the action installs only
`pre-commit`. `go-generate-catalog` is in that job on purpose: a pull request that edits
`models.yaml` without regenerating what is built from it gets the diff posted as a review comment.

## What CI proves for a release

The release used to repeat `test`, `cross` and `govulncheck` before building. It now requires CI
green on the exact commit instead (`require-ci-green`: the newest run of `ci.yaml` for the push of
that SHA to `main`, its newest attempt, exactly one `ci-result` job concluded success), and keeps
only the check whose answer changes with the calendar.

| Check | CI (`ci-result` needs it) | Release |
| --- | --- | --- |
| `make go-test` with `-race`, Linux and macOS | `test` | dropped: covered by `ci-green` |
| `make go-build-cross`, `go-vet-cross`, `go-lint-cross` | `cross` | dropped: covered by `ci-green` |
| Native macOS lint | `lint-macos` | never ran here; covered by `ci-green` |
| Every pre-commit hook on every file | `pre-commit-all` | never ran here; covered by `ci-green` |
| Hook script tests | `scripts` | never ran here; covered by `ci-green` |
| `goreleaser check` | `release-config` | still runs, against the same pinned goreleaser |
| govulncheck (`make audit-deps`) | `govulncheck` | still runs: `audit`, in parallel with the build, required before anything is published |
| Build, package, sign | not run | runs: the only build of the shipped artifacts |

What that relies on: the commit is the same (the gate compares the run's `head_sha` with the
release's), the Go toolchain is resolved the same way (`go-version-file: go.mod` with
`check-latest`, so a Go patch released between CI and the release would build with a newer patch
than CI tested), and goreleaser is the pinned one in both workflows. CI does not build the release
artifacts themselves: goreleaser's cross-compiled builds, archives, packages and signatures are
produced only by the release, and `goreleaser check` is the extent of CI's view of them.

## Before the first release

Setup that is done by hand, once, in this order.

1. **Cloudsmith.** Create the repository `monmux` under `leinardi` and mark it Open-Source (no
   pre-approval; 50 GB storage and 200 GB of monthly traffic; attribution in the README is the
   condition). Note the repository's GPG key fingerprint for the README's apt and rpm instructions.
   Create a service account with write access to `monmux` and note its slug. Under Settings →
   Authentication → OpenID Connect, add a provider with URL
   `https://token.actions.githubusercontent.com`, bound to that service account. "Required OpenID
   Token Claims" is a flat JSON object, and every claim in it must match for a token to be accepted:

   ```json
   {
     "repository": "leinardi/monmux",
     "repository_owner": "leinardi",
     "job_workflow_ref": "leinardi/monmux/.github/workflows/release.yaml@refs/heads/main"
   }
   ```

   At least one claim is effectively mandatory: the keys that verify GitHub's tokens are shared
   across every GitHub organization, so a provider with no claim would trust any workflow anywhere
   that knows the service slug. `job_workflow_ref` pins it to this one workflow file on `main`,
   which is why renaming the file, or dispatching a release from another branch, breaks
   authentication — deliberately. Store the slug as the repository **variable**
   `CLOUDSMITH_SERVICE_SLUG`. No API key is stored in GitHub.

   The workspace slug has to be `leinardi`: it is the namespace in `.goreleaser.yaml`'s Cloudsmith
   paths, in the workflow's `oidc-namespace`, and in every
   `dl.cloudsmith.io/public/leinardi/monmux/...` URL the README prints.
2. **The tap.** Create the public repository `leinardi/homebrew-tap` with a `Casks/` directory and a
   README; goreleaser pushes into it but will not create it. Users tap it as `leinardi/tap`.
3. **The one long-lived secret.** A fine-grained PAT, resource owner `leinardi`, repository access
   limited to `homebrew-tap`, permission Contents: read and write. Store it as the repository secret
   `HOMEBREW_TAP_TOKEN`. GitHub caps fine-grained PATs at one year, so set a reminder for its
   expiry. If rotating it becomes a nuisance, the alternative is a GitHub App installed on the tap
   and minted per run with `actions/create-github-app-token`.
4. **The `release` environment.** It is created on the first release run with no protection rules,
   which by itself changes nothing. Add a deployment branch rule limiting it to `main`, as a
   server-side second line of defence behind the `guard` job, and required reviewers if a release
   should wait for a human.
5. **Rulesets.** Leave "Immutable tags" exactly as it is; the release job is written for it. The
   rulesets are managed as code in
   [`leinardi/gh-leinardi-iac`](https://github.com/leinardi/gh-leinardi-iac), in the `repo_monmux`
   module of `stacks/github-repos/repos.tf`, never by hand: a change is a pull request there. The
   one required check is to be `ci-result`, set there as
   `default_branch_required_checks = ["ci-result"]`; it is not set yet. Until it is, nothing stops a
   merge on a red CI, though the release still refuses a commit whose CI did not pass. It needs
   every gating job of `ci.yaml`, so a new job becomes required by joining its `needs`, not the
   ruleset; GitHub counts a skipped required job as passing, which is why the jobs themselves are
   not listed. Once the ruleset is applied, open one pull request and confirm the check reports
   under exactly the name `ci-result`: a required check that never reports blocks every merge.
6. **Labels.** `catalog`, for monitor reports and catalog pull requests; confirm `dependencies` and
   `ci` exist for dependabot.
7. **The release itself.** Record the hardware checklist in [testing.md](testing.md) for every model
   the release enables, then dispatch with `dry_run: true` and version `0.1.0` and read the summary.
   Then dispatch for real with `0.1.0` given explicitly — there is no previous tag for `svu` to
   reason from.
