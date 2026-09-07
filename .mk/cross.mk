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
