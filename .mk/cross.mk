# Repo-local cross-compilation targets.
#
# monmux ships two OS backends behind build tags - internal/backend/ddcutil is
# //go:build linux, internal/backend/m1ddc is //go:build darwin - so the host
# toolchain only ever compiles one of them, and the other is invisible to every
# check that runs natively. These targets compile-check and vet both, on either
# host. They never run anything.
#
# Both are named explicitly rather than derived as "the OS the host is not".
# `go env GOOS` reports the target, not the host, so an exported GOOS moves it -
# and go-build and go-vet in .mk/go.mk set no GOOS of their own, so they move
# with the environment too. Deriving from either value therefore leaves a case
# where both halves aim at the same OS and one backend is checked by nothing at
# all, silently. Naming both costs one extra, already-cached build and has no
# such case. `go env GOHOSTOS` does not fix this: it pins these targets to the
# host while the ones they complement still follow GOOS.
#
# go-lint-cross does the same for the linter, and it is the one that matters:
# lint is where the two backends diverge most, and three findings in
# internal/backend/m1ddc went unreported for as long as the linter only ever ran
# on Linux.
#
# It is also the one that can break for reasons that have nothing to do with
# this repository. golangci-lint typechecks the standard library from source
# when cross-targeting, using the go/types it was built with; when that is older
# than the toolchain it aborts in GOROOT and then reports nothing at all about
# this repository. golangci-lint v2.12.2 (built with go1.26.5) does exactly that
# against a Go 1.27 toolchain, while v2.13.2 (built with go1.27.0) is clean.
#
# If this target fails inside GOROOT rather than inside this repository, your
# golangci-lint is older than your Go toolchain: update .pre-commit-config.yaml,
# or skip the target. Never silence that typecheck error with a path exclusion.
# It does not restore the analysis, it only hides the abort - the run goes green
# having checked nothing, which is worse than not running it. Verified by
# planting a violation in a build-tagged file: the aborting version reports it
# nowhere, the working version reports it.

# The linter is not part of make-common's Go snippet, and the copy the hooks use
# lives inside pre-commit's cache, so this expects one on PATH. Override it to
# point at another: make go-lint-cross GOLANGCI_LINT=/path/to/golangci-lint
GOLANGCI_LINT ?= golangci-lint

.PHONY: go-build-cross
go-build-cross: ## Compile-check both OS backends, whatever the host (no binary kept)
	GOOS=linux $(GO) build -o /dev/null $(GO_PKG)
	GOOS=darwin $(GO) build -o /dev/null $(GO_PKG)

.PHONY: go-vet-cross
go-vet-cross: ## Static checks (go vet) for both OS backends, build-tagged tests included
	GOOS=linux $(GO) vet $(GO_PKG)
	GOOS=darwin $(GO) vet $(GO_PKG)

.PHONY: go-lint-cross
go-lint-cross: ## Lint both OS backends; see the note above if it fails inside GOROOT
	GOOS=linux $(GOLANGCI_LINT) run $(GO_PKG)
	GOOS=darwin $(GOLANGCI_LINT) run $(GO_PKG)

# The supported-monitor catalog is written in internal/catalog/models.yaml and
# rendered into two committed files: internal/catalog/models_gen.go, and the
# region between the markers in docs/compatibility.md. Run this after editing the
# YAML and commit everything it rewrites; a test per rendered file fails the
# build if any of them disagree. Scoped to the catalog because that is the only
# generator here.
.PHONY: go-generate
go-generate: ## Regenerate models_gen.go and the compatibility document from models.yaml
	$(GO) generate ./internal/catalog/...
