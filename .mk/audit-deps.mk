ifndef MK_LOCAL_AUDIT_DEPS_INCLUDED
MK_LOCAL_AUDIT_DEPS_INCLUDED := 1

# Local snippet (NOT part of make-common): check the dependencies this repository ships against
# known vulnerabilities, with the tool the Go ecosystem publishes.
#
# What this is and is not. govulncheck is source-aware: it reports a vulnerable module only when
# the build actually reaches the affected symbol, so it answers "is this repository exposed", not
# "does a vulnerable version appear in the graph".
#
# It needs the network, and govulncheck over every package is too slow to run on every Go change.
# The pre-commit hook runs this same target, but only when go.mod or go.sum change; CI runs it on
# every pull request (the govulncheck job in .github/workflows/ci.yaml) and the release workflow
# before anything is built.
#
# -test scans the test files as well. The tests ship in no binary, but they run on every developer
# machine and in CI, so a vulnerable path reached only from a test is still a path this repository
# executes.
#
# The version is pinned so two runs mean the same thing. Bump it deliberately.

GOVULNCHECK_VERSION ?= v1.8.0
GOVULNCHECK        ?= golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: audit-deps
audit-deps: audit-deps-go ## Scan Go dependencies for known vulnerabilities (network required)

.PHONY: audit-deps-go
audit-deps-go: ## Run the pinned govulncheck over every Go package, test files included
	$(GO) run $(GOVULNCHECK) -test ./...

endif  # MK_LOCAL_AUDIT_DEPS_INCLUDED
