# devtool — build, test, and lint this repository.

BIN_DIR     := bin
BINARY      := $(BIN_DIR)/devtool
CMD         := ./cmd/devtool

GOPRIVATE   ?= github.com/gophertribe
export GOPRIVATE

GOTESTSUM_PKG        := gotest.tools/gotestsum
GOTESTSUM_VERSION    := v1.13.0
GOLANGCI_LINT_PKG    := github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GOLANGCI_LINT_VERSION := v2.7.2

# Prefer a tool on PATH when present; otherwise pin via go run (same as test/).
GOTESTSUM = $(shell command -v gotestsum 2>/dev/null || echo "go run $(GOTESTSUM_PKG)@$(GOTESTSUM_VERSION)")
GOLANGCI_LINT = $(shell command -v golangci-lint 2>/dev/null || echo "go run $(GOLANGCI_LINT_PKG)@$(GOLANGCI_LINT_VERSION)")

GOTESTSUM_FLAGS := --no-summary=skipped --format short
# JUnit output for CI; filename matches test/test.go convention.
GOTESTSUM_FLAGS += --junitfile ./coverage.xml

.PHONY: all build test lint check clean help patch-deps

PATCH_GO_GIT := ./scripts/patch-go-git.sh
GO_GIT_PATCHED := third_party/go-git/.patched

patch-deps: $(GO_GIT_PATCHED)

$(GO_GIT_PATCHED): go.mod go.sum patches/go-git-worktreeconfig.patch $(PATCH_GO_GIT)
	$(PATCH_GO_GIT)

all: build check

build: patch-deps
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) $(CMD)

test: patch-deps
	$(GOTESTSUM) $(GOTESTSUM_FLAGS) ./...

lint: patch-deps
	$(GOLANGCI_LINT) run --timeout 5m ./...

check: lint test

clean:
	rm -rf $(BIN_DIR) coverage.xml

help:
	@echo "Targets:"
	@echo "  build  - compile $(CMD) -> $(BINARY)"
	@echo "  test   - run unit tests (gotestsum)"
	@echo "  lint   - run golangci-lint"
	@echo "  check  - lint then test"
	@echo "  all    - build and check"
	@echo "  clean  - remove $(BIN_DIR) and coverage.xml"
	@echo "  help   - show this message"
