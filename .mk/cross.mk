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
# There is deliberately no cross-lint target. golangci-lint with GOOS set to the
# other OS aborts inside the standard library's own typecheck, because the
# go/types it was built with is older than the toolchain's, and it then reports
# nothing whatsoever about this repository - a planted violation in a
# build-tagged file goes unreported. Excluding that typecheck error by path, the
# obvious next move, turns the run into a green result that checked nothing.
# Linting the other backend needs a runner of that OS; see docs/release.md.

.PHONY: go-build-cross
go-build-cross: ## Compile-check both OS backends, whatever the host (no binary kept)
	GOOS=linux $(GO) build -o /dev/null $(GO_PKG)
	GOOS=darwin $(GO) build -o /dev/null $(GO_PKG)

.PHONY: go-vet-cross
go-vet-cross: ## Static checks (go vet) for both OS backends, build-tagged tests included
	GOOS=linux $(GO) vet $(GO_PKG)
	GOOS=darwin $(GO) vet $(GO_PKG)

# The supported-monitor catalog is written in internal/catalog/models.yaml and
# rendered into internal/catalog/models_gen.go, which is committed. Run this
# after editing the YAML and commit both files; a test fails the build if they
# disagree. Scoped to the catalog because that is the only generator here.
.PHONY: go-generate
go-generate: ## Regenerate internal/catalog/models_gen.go from models.yaml
	$(GO) generate ./internal/catalog/...
