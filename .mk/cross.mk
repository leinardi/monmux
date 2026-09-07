# Repo-local cross-compilation targets.
#
# monmux is developed on Linux but ships a macOS backend (internal/backend/m1ddc,
# //go:build darwin). These targets keep that backend compile-checked from the
# Linux development host; they never run anything.

.PHONY: go-build-darwin
go-build-darwin: ## Compile-check the darwin build (no binary kept)
	GOOS=darwin $(GO) build -o /dev/null $(GO_PKG)

.PHONY: go-vet-darwin
go-vet-darwin: ## Static checks (go vet) for GOOS=darwin
	GOOS=darwin $(GO) vet $(GO_PKG)

# The supported-monitor catalog is written in internal/catalog/models.yaml and
# rendered into internal/catalog/models_gen.go, which is committed. Run this
# after editing the YAML and commit both files; a test fails the build if they
# disagree. Scoped to the catalog because that is the only generator here.
.PHONY: go-generate
go-generate: ## Regenerate internal/catalog/models_gen.go from models.yaml
	$(GO) generate ./internal/catalog/...
