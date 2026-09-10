# Release

How a monmux release is built, what it produces, and what to do when one fails halfway. The pipeline is
[`.github/workflows/release.yaml`](../.github/workflows/release.yaml) and [`.goreleaser.yaml`](../.goreleaser.yaml); this page is
why it is shaped that way.

One rule frames all of it: **no job runs monmux against hardware.** There is no monitor in CI, and the rule that no automation
writes to a monitor is not suspended because the automation is called a workflow. Nothing in either workflow executes `ddcutil`,
`m1ddc` or `monmux switch`; the test suite refuses to start a process by construction, so a runner needs neither tool installed.
Hardware validation is a human checklist — [testing.md](testing.md).

## What a release produces

- Static binaries for `linux/amd64`, `linux/arm64` and `darwin/arm64`, each with `-trimpath`, `CGO_ENABLED=0` and the version,
  commit and build date compiled in, so `monmux version` reports the tag rather than `dev`.
    - `darwin/amd64` is deliberately absent: the macOS backend wraps `m1ddc`, which requires Apple Silicon. Building an Intel
    binary that can never work would be a promise monmux cannot keep.
- A `tar.gz` per target, carrying the binary, `LICENSE` and `README.md`.
- `checksums.txt` covering every artifact, the packages included, and `checksums.txt.sigstore.json`: a keyless
  [cosign](https://docs.sigstore.dev/) signature made with the release workflow's own OIDC identity. There is no private key
  anywhere. `monmux doctor` already prints the SHA-256 of the tool it runs; the same idea applies to monmux itself.
- A [build provenance attestation](https://docs.github.com/actions/security-guides/using-artifact-attestations) over the
  archives, the packages and the checksum file, verifiable with `gh attestation verify`.
- `.deb` and `.rpm` packages, built by [nfpm](https://nfpm.goreleaser.com/), published to
  [Cloudsmith](https://cloudsmith.io/~leinardi/repos/monmux/) as well as attached to the GitHub release.
- A [Homebrew **cask**](https://docs.brew.sh/Cask-Cookbook) in `leinardi/homebrew-tap`, depending on the `m1ddc` formula.
- Release notes: goreleaser's grouped changelog, under a header generated from `internal/catalog/models.yaml` itself, which
  names every catalog change the release makes.

### A cask, not a formula

goreleaser's `brews:` was soft-deprecated in v2.10 and hard-deprecated in v2.16; `homebrew_casks:` replaced it. That suits
monmux: a cask is macOS-only by nature, which is the only platform the tap needs to serve, and it takes
`dependencies: [{formula: m1ddc}]` with no OS guard.

It has one consequence worth stating plainly, because it is a security decision rather than a packaging detail: Homebrew
quarantines everything a cask downloads, and monmux's binaries carry no Apple Developer ID signature, so without intervention
Gatekeeper refuses to run them at all. The cask therefore strips `com.apple.quarantine` in a `postflight` hook, which is
goreleaser's documented answer. What that bypasses, and how to install without it, is in
[security.md](security.md#the-install-path).

### The package dependency is unversioned

The packages declare `Depends: ddcutil` and `Requires: ddcutil` with **no version**, and the package description names the 2.2
floor. This is deliberate, and it is a trade, so both sides are written down.

monmux needs `ddcutil` 2.2. Distributions today ship (as of September 2026): Debian 12 `1.4.1`, Debian 13 `2.2.0`,
Ubuntu 24.04 `1.4.1`, Ubuntu 26.04 `2.2.5`, Fedora 42 `2.1.4`, Fedora 43 `2.2.1`, Arch and Tumbleweed `2.2.7`,
openSUSE Leap 15.6 `1.4.1`.

- A hard floor would be honest at install time, but `apt` would refuse the package outright on Debian 12 and Ubuntu 24.04 — the
  two most common targets — including for somebody who built `ddcutil` 2.2 into `/usr/local`, which `dpkg` cannot see. There is
  no way past that short of `--force-depends`.
- The unversioned dependency installs everywhere. `monmux info` still works, because enumeration reads sysfs and needs no tool
  at all, and anything that needs the tool refuses with `backend-not-ready` naming the version it found and the version it
  needs. `monmux doctor` says the same thing before you try. A user is never left with a mystery, only with a version to
  upgrade, and the README says up front that Debian 12 and Ubuntu 24.04 ship one that is too old.

The floor is enforced where it can be checked properly — in preflight, against the binary that is actually installed — rather
than by a package manager guessing from a version string.

## How a release runs

One `workflow_dispatch` on `main`, with two inputs: `version` (optional, `X.Y.Z` or `vX.Y.Z`) and `dry_run` (boolean).

**One release runs at a time.** The workflow takes a `release` concurrency group with `cancel-in-progress: false`, so a second
dispatch queues behind the first rather than racing it. Two runs in flight together would each check their version against the
same latest tag, both pass, and then publish in whatever order they finished — a lower version landing last would take `latest`
with it. Nothing is ever cancelled either: a run killed between the tag push and the Cloudsmith upload is the one state this
pipeline has to be recovered from by hand.

The job order exists because **tags in this repository are immutable**. The "Immutable tags" ruleset covers `~ALL` tags except
`v[0-9]` and `v[1-9][0-9]`, blocks update, deletion and non-fast-forward, and has no bypass actors, so the familiar "delete the
tag on failure" cleanup step cannot work here — it would fail every time. Everything that can fail without a tag therefore
happens before the tag exists, and everything after it is idempotent.

1. **Refuse anything but `main`.** `workflow_dispatch` lets the caller pick any branch, and the release job holds
   `contents: write`, an OIDC identity and the tap token — dispatched from a side branch it would tag that branch's tree and
   publish it. The `guard` job fails, rather than skips, when `github.ref` is not `refs/heads/main`, and every other job waits on
   it, so such a run is red and nothing runs at all. The ruleset cannot express this: tag protection stops a tag from *moving*,
   not from being created on the wrong commit. The second line of defence is the `release` environment — add a deployment branch
   rule limiting it to `main`, and required reviewers if a release should need a second pair of eyes.
2. **Verify.** `test` on Linux and macOS, `cross`, `govulncheck`. These are copies of the CI jobs rather than a call into
   `ci.yaml`, so that a release never depends on which pull request last ran CI.
3. **Resolve the version.** An explicit input is validated against `^v[0-9]+\.[0-9]+\.[0-9]+$`. Otherwise `svu next` is compared
   with `svu current`: `svu` prints the current version unchanged, and exits 0, when nothing since the last tag is a `feat`, a
   `fix` or a breaking change, so equality is the "nothing to bump" case and it fails with
   *"No feat/fix or breaking-change commit since `<tag>`, so there is nothing to bump. Re-run with an explicit version to force
   one."*

   The version must then be **higher than every version already released**, whether it was derived or typed. Tags here are
   immutable and the release is marked latest, so publishing a `v1.1.1` after `v1.2.0` would create a permanent lower tag and
   hand everyone following the "latest" link an older binary. The only version a run may reuse is the one it is recovering.

   **A recovery is exactly this version already sitting on `HEAD`** — not "`HEAD` carries some tag". The wider test would misread
   a release cut from a commit that is already tagged with the *previous* version, and would then read the catalog baseline from
   one release further back. Every tag lookup in the step also matches `v[0-9]*.[0-9]*.[0-9]*` rather than taking the nearest
   tag, because `latest` is a separate pointer that the ruleset deliberately leaves mutable.

   The baseline for the notes is the previous release tag — read from `HEAD^` on a recovery, since the tag being released is on
   `HEAD` already and describing from there would compare the catalog against itself.
4. **Write the catalog notes**, and apply the catalog version rule below.
5. **Check the tag.** If `refs/tags/<version>` already exists on the remote, the run continues only when it points at `HEAD` — a
   re-run recovering a failed publish. Pointing anywhere else is a hard failure.
6. **Tag locally.** `git tag -a`, and nothing is pushed. goreleaser only checks that the tag exists locally on `HEAD`
   (`git describe --exact-match`), never on the remote, so building against a local-only tag is supported. That is also why the
   changelog uses `use: git`: the GitHub compare API would 404 on a tag it cannot see.
7. **`goreleaser check`**, then **`goreleaser release --clean --skip=publish`**: builds, archives, packages, checksums and signs,
   with nothing pushed anywhere. A configuration error, a build error or a cosign error stops the run here, with no tag on the
   remote and nothing to clean up.
8. **A dry run stops here**, having written the release notes and the artifact list to the job summary and uploaded `dist/` as a
   workflow artifact. This is the review step: the version is resolved, the notes are rendered and everything is built, without
   a tag, a release, a cask commit or a Cloudsmith upload.
9. **`git push origin <version>`** — the point of no return. Immediately before pushing, the highest released version is read
   again, from the **remote** this time: concurrency serializes this workflow, but a tag can arrive from outside it, and the
   check in step 3 ran several minutes of building ago. A version that is no longer the highest fails here, with nothing pushed.
10. **`goreleaser release --clean`** again: it rebuilds and publishes — creating the GitHub release with
    `target_commitish: {{ .FullCommit }}`, uploading every artifact and pushing the cask to the tap. Rebuilding is safe because
    the build is reproducible: everything stamped into a binary comes from the commit (`.Tag`, `.FullCommit`, `.CommitDate`,
    `mod_timestamp: {{ .CommitTimestamp }}`) rather than from the clock, so the second build produces the same bytes as the one
    the dry run inspected. Verified by building the same commit twice and comparing SHA-256.
11. **Attest** the archives, packages and checksum file.
12. **`cloudsmith push deb|rpm ... --republish`** for each package.

### Recovering a failed publish

Re-run the workflow with the **same version, given explicitly**. Everything after the tag push is idempotent by construction:

- the tag check passes, because the tag points at `HEAD`, and the push step is skipped;
- `release.mode: replace` and `replace_existing_artifacts: true` refresh the GitHub release instead of appending a second copy of
  every asset;
- goreleaser skips a cask whose content has not changed;
- `--republish` overwrites the same package version on Cloudsmith.

There is no cleanup step, and the tag is never deleted. That is the stronger promise, not the weaker one: a version that was ever
published never points anywhere else.

If the release is wrong rather than incomplete — the wrong commit, a bad build — the answer is a new version, not a moved tag.

## Versioning

Semantic versioning, derived from Conventional Commits by [`svu`](https://github.com/caarlos0/svu): `feat` is a minor, `fix` is a
patch, `!` or `BREAKING CHANGE:` is a major. `svu` is a single Go binary installed with the toolchain already on the runner, it
reads tags and commit messages, and it does nothing else — goreleaser already renders the changelog, so a second changelog
engine would be one more thing to keep in step. Its version is pinned in the workflow by hand, because dependabot does not track
a `go install` argument.

One project-specific rule sits on top: **any change to `internal/catalog/models.yaml` that adds or moves a byte this binary can
send to a monitor is at least a minor release, and is called out at the top of the notes.** Somebody deciding whether to upgrade
needs to know that the new version can send something the old one could not.

"Can send" is wider than "is write-enabled", and deliberately so. An entry with `write_enabled: false` is never matched and never
written to by `monmux switch` — but `catalog.Model.UnsafeOperation` ignores the flag, and that is exactly what `--unsafe-model`
calls, so recording a model or an input puts a value in the binary that the previous one had no way to send at all. Recording one
is therefore a minor too. The practical consequence: a release that adds entries to the catalog is never a patch, even when every
one of them arrives disabled.

What does *not* count: losing something (a model removed, an input dropped, a model or identity no longer write-enabled) and
record-keeping (evidence, per-model notes, sources). None of those adds a byte.

Generated commit notes cannot know any of this, so a small tool does:
[`internal/catalog/internal/generate/cmd/relnotes`](../internal/catalog/internal/generate/cmd/relnotes). It takes the previous
release's `models.yaml` (from `git show <previous-tag>:...`) and the current one, and prints the section that goes at the top of
the release notes. The sections that count as enabling come first — models newly write-enabled, inputs newly enabled, values
changed on an enabled input, identities added, models and inputs newly recorded, values changed on a recorded input — then the
ones that do not: identities removed, models no longer write-enabled, inputs no longer recorded, models removed, and a last
section naming any entry whose evidence, notes or sources moved. That last one exists so that "no catalog change" can only mean
the file is identical, rather than meaning the bytes happen to match. It also prints `enabled=true|false`, which is what the
workflow acts on:

- derived version, patch bump, `enabled=true` → the workflow uses `svu minor` instead and prints a notice;
- explicit version, patch bump, `enabled=true` → the workflow fails and names the minor to pass. An explicit input is a choice;
  the rule outranks it.

When nothing changed, the header still says so — *"No catalog change: this release writes exactly what `<previous>` wrote."* —
because silence would be indistinguishable from a step that did not run.

The commit-message changelog is grouped underneath: `feat(catalog)` and `fix(catalog)` first under **Catalog**, then
**Features**, then **Bug fixes**, then everything else; `docs`, `chore`, `ci`, `test`, `style` and merge commits are left out.

## What the workflows trust

**goreleaser is pinned to an exact version**, in `GORELEASER_VERSION`, rather than to `~> v2`. A recovery re-run rebuilds a
tagged commit and has to produce the same artifacts; a goreleaser released in between could change them, which would defeat the
reproducibility the rebuild depends on. `ci.yaml` checks the configuration against the same pin, so `goreleaser check` validates
against the binary that will actually build the release. Like `SVU_VERSION`, it is a human's job to bump.

**Every action is pinned to a full commit SHA**, with the version in a trailing comment, in all three workflows. A tag is a
mutable pointer: `@v7` resolves to whatever that tag points at on the day the job runs, and an action that moves — or whose
repository is compromised — would execute inside the release job, which holds `contents: write`, an OIDC identity good for
Cloudsmith, and the tap token. A SHA cannot be moved. Dependabot updates the pins and rewrites the comment, so this costs nothing
to maintain. `svu` is pinned the same way, as a version in `SVU_VERSION`, because dependabot cannot see a `go install` argument;
that one is a human's job to bump.

**Verification pins the signing identity, not just the repository.** The keyless cosign certificate carries the workflow that
produced it, so the documented command checks
`--certificate-identity 'https://github.com/leinardi/monmux/.github/workflows/release.yaml@refs/heads/main'` rather than a regexp
over the repository. An identity regexp matching any workflow in the repository would accept a signature produced by any other
workflow, which is a lower bar than it looks. `gh attestation verify` gets the same treatment through `--signer-workflow` plus
`--source-ref refs/heads/main`: the workflow path alone would still accept a run of that file from any branch, which is the very
thing the `guard` job exists to prevent.

## CI

[`.github/workflows/ci.yaml`](../.github/workflows/ci.yaml) runs on every pull request, on pushes to `main` and on dispatch.

| Job                    | Runner                | What it does                                                                                       |
| ---------------------- | --------------------- | -------------------------------------------------------------------------------------------------- |
| `test`                 | ubuntu + macos        | `make go-test` (`go test -race ./...`) on both, so each backend's build-tagged tests run.          |
| `cross`                | ubuntu                | `make go-build-cross`, `make go-vet-cross`, `make go-lint-cross`.                                  |
| `lint-macos`           | macos                 | `golangci-lint run ./...` natively, the only look at `internal/backend/m1ddc` under its tag.       |
| `govulncheck`          | ubuntu                | Vulnerability scan of the module.                                                                  |
| `release-config`       | ubuntu                | `goreleaser check`.                                                                                |
| `actionlint`           | ubuntu, pull requests | Workflow linting, as inline review comments.                                                       |
| `pre-commit-hooks`     | ubuntu, pull requests | The hook suite `make check` runs, minus the hooks with their own job and minus `go-test-repo-mod`. |
| `markdownlint`         | ubuntu, pull requests | Markdown linting.                                                                                  |
| `shellcheck`           | ubuntu, pull requests | Shell linting.                                                                                     |
| `yamllint`             | ubuntu, pull requests | YAML linting.                                                                                      |
| `conventional-commits` | ubuntu, pull requests | Every commit message in the range, because the release version is derived from them.               |

Three things there are deliberate:

- **`test` runs on both a Linux and a macOS runner.** Each backend sits behind a build tag, so a linter or a compiler running on
  one OS does not analyse the other's files at all, and three real findings in `internal/backend/m1ddc` sat unreported until the
  first `make check` was run on a Mac. The macOS runner needs no `m1ddc` installed, because the suite executes no external
  binary — [testing.md](testing.md).
- **The linter also runs natively on macOS**, rather than only through `make go-lint-cross`. The cross target narrows the gap but
  does not close it: it depends on `golangci-lint` being built with a Go no older than the toolchain, and aborts in `GOROOT` —
  reporting nothing about this repository — when it is not, which is a bad thing to have as the only line of defence. Both jobs
  read the linter version out of `.pre-commit-config.yaml`, so there is one source of truth and dependabot keeps it moving.
- **The pre-commit cache is warmed.** [`pre-commit-warmup.yaml`](../.github/workflows/pre-commit-warmup.yaml) refills
  `~/.cache/pre-commit` on every push to `main` that touches `.pre-commit-config.yaml`, under the key the reviewdog composite
  actions read. It matters more here than in the sibling repositories: the `golangci-lint` hook builds the linter from source on
  a cold cache.

The `pre-commit-hooks` job sets Go up before the composite action, because `go-generate-catalog` and the `golangci-lint` hooks
need a toolchain matching `go.mod` and the action installs only `pre-commit`. `go-generate-catalog` is in that job on purpose: a
pull request that edits `models.yaml` without regenerating what is built from it gets the diff posted as a review comment.

## Before the first release

Setup that is done by hand, once, in this order.

1. **Cloudsmith.** Create the repository `monmux` under `leinardi` and mark it Open-Source (no pre-approval; 50 GB storage and
   200 GB of monthly traffic; attribution in the README is the condition). Note the repository's GPG key fingerprint for the
   README's apt and rpm instructions. Create a service account with write access to `monmux` and note its slug. Under
   Settings → Authentication → OpenID Connect, add a provider with URL `https://token.actions.githubusercontent.com`, bound to
   that service account. "Required OpenID Token Claims" is a flat JSON object, and every claim in it must match for a token to
   be accepted:

   ```json
   {
     "repository": "leinardi/monmux",
     "repository_owner": "leinardi",
     "job_workflow_ref": "leinardi/monmux/.github/workflows/release.yaml@refs/heads/main"
   }
   ```

   At least one claim is effectively mandatory: the keys that verify GitHub's tokens are shared across every GitHub
   organization, so a provider with no claim would trust any workflow anywhere that knows the service slug. `job_workflow_ref`
   pins it to this one workflow file on `main`, which is why renaming the file, or dispatching a release from another branch,
   breaks authentication — deliberately. Store the slug as the repository **variable** `CLOUDSMITH_SERVICE_SLUG`. No API key is
   stored in GitHub.

   The workspace slug has to be `leinardi`: it is the namespace in `.goreleaser.yaml`'s Cloudsmith paths, in the workflow's
   `oidc-namespace`, and in every `dl.cloudsmith.io/public/leinardi/monmux/...` URL the README prints.
2. **The tap.** Create the public repository `leinardi/homebrew-tap` with a `Casks/` directory and a README; goreleaser pushes
   into it but will not create it. Users tap it as `leinardi/tap`.
3. **The one long-lived secret.** A fine-grained PAT, resource owner `leinardi`, repository access limited to `homebrew-tap`,
   permission Contents: read and write. Store it as the repository secret `HOMEBREW_TAP_TOKEN`. GitHub caps fine-grained PATs at
   one year, so set a reminder for its expiry. If rotating it becomes a nuisance, the alternative is a GitHub App installed on
   the tap and minted per run with `actions/create-github-app-token`.
4. **The `release` environment.** It is created on the first release run with no protection rules, which by itself changes
   nothing. Add a deployment branch rule limiting it to `main`, as a server-side second line of defence behind the `guard` job,
   and required reviewers if a release should wait for a human.
5. **Rulesets.** Leave "Immutable tags" exactly as it is; the release job is written for it. Once the workflows have run once,
   add the required status checks to the default-branch ruleset: `test (ubuntu-latest)`, `test (macos-latest)`, `cross`,
   `lint-macos`, `govulncheck`, `release-config`, `conventional-commits`, `pre-commit-hooks`, `markdownlint`, `yamllint`,
   `shellcheck`, `actionlint`.
6. **Labels.** `catalog`, for monitor reports and catalog pull requests; confirm `dependencies` and `ci` exist for dependabot.
7. **The release itself.** Record the hardware checklist in [testing.md](testing.md) for every model the release enables, then
   dispatch with `dry_run: true` and version `0.1.0` and read the summary. Then dispatch for real with `0.1.0` given
   explicitly — there is no previous tag for `svu` to reason from.
