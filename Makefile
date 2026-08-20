# Makefile for ling-base — a multi-module Go library.
#
# Usage:
#   make build           Build all modules in the workspace.
#   make build-pkg PKG=scheduler   Build a single module (by directory name).
#   make test            Run tests for all modules.
#   make test-pkg PKG=scheduler    Test a single module.
#   make test-cover      Run tests with coverage for all modules.
#   make vet             Run go vet for all modules.
#   make fmt             Format all Go source files.
#   make fmt-check       Check formatting without modifying files.
#   make lint            Run golangci-lint (if installed).
#   make check           Run fmt-check + vet + build + test (CI equivalent).
#   make cover-html      Generate HTML coverage report (aggregate).
#   make clean           Remove generated coverage files and demo binaries.
#   make tags            List all git tags grouped by module.
#   make release-patch   Bump patch version for a module and create a tag.
#   make release-minor   Bump minor version for a module and create a tag.
#   make release-major   Bump major version for a module and create a tag.
#   make release-all     Auto-detect changed modules and bump patch versions.
#   make push-tags       Push all tags to the remote.
#
# Release examples:
#   make release-patch PKG=scheduler              # scheduler/v0.1.0 → scheduler/v0.1.1
#   make release-minor PKG=common/jwtutil         # common/jwtutil/v0.1.0 → common/jwtutil/v0.2.0
#   make release-major PKG=common                 # common/v0.3.1 → common/v1.0.0
#   make release-all                              # Auto-bump all modules with changes since last tag.
#   make release-all LEVEL=minor                  # Auto-bump all changed modules by minor version.
#
# Root module release:
#   make release-patch PKG=.                      # v0.2.1 → v0.2.2

SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

# ──────────────────────────────────────────────
# Variables
# ──────────────────────────────────────────────

GO        := go
ROOT      := $(shell pwd)
GOFLAGS   := -trimpath
TAG_PREFIX := github.com/LingByte/ling-base
RELEASE_LEVEL ?= patch

# Find all go.mod directories (excluding vendor).
GO_MOD_DIRS := $(shell find . -name go.mod -not -path './vendor/*' -print0 | xargs -0 -n1 dirname | sed 's|^\./||' | sort)

# ──────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@echo "ling-base Makefile — multi-module Go library"
	@echo ""
	@echo "Build:"
	@echo "  make build              Build all modules"
	@echo "  make build-pkg PKG=dir  Build a single module"
	@echo ""
	@echo "Test:"
	@echo "  make test               Test all modules"
	@echo "  make test-pkg PKG=dir   Test a single module"
	@echo "  make test-cover         Test with coverage"
	@echo "  make vet                Run go vet"
	@echo "  make check              fmt-check + vet + build + test"
	@echo ""
	@echo "Code quality:"
	@echo "  make fmt                Format all files"
	@echo "  make fmt-check          Check formatting"
	@echo "  make lint               Run golangci-lint (if installed)"
	@echo "  make vuln               Run govulncheck (if installed)"
	@echo "  make check              fmt-check + vet + build + test"
	@echo "  make check-all          check + lint + vuln"
	@echo ""
	@echo "Release:"
	@echo "  make release-patch PKG=dir   Bump patch version (v0.1.0 → v0.1.1)"
	@echo "  make release-minor PKG=dir   Bump minor version (v0.1.0 → v0.2.0)"
	@echo "  make release-major PKG=dir   Bump major version (v0.1.0 → v1.0.0)"
	@echo "  make release-all             Auto-bump all changed modules (patch)"
	@echo "  make release-all LEVEL=minor Auto-bump all changed modules (minor)"
	@echo "  make tags                    List all tags"
	@echo "  make push-tags               Push all tags to remote"
	@echo ""
	@echo "Misc:"
	@echo "  make clean              Remove coverage files and demo binaries"
	@echo "  make modules            List all Go modules in the workspace"

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

.PHONY: build build-pkg
build: ## Build all modules
	@echo "==> Building all modules..."
	@$(GO) build $(GOFLAGS) ./...

build-pkg: ## Build a single module (PKG=dir)
	@test -n "$(PKG)" || (echo "Usage: make build-pkg PKG=<directory>" && exit 1)
	@echo "==> Building $(PKG)..."
	@cd $(PKG) && $(GO) build $(GOFLAGS) ./...

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

.PHONY: test test-pkg test-cover
test: ## Run tests for all modules
	@echo "==> Testing all modules..."
	@$(GO) test -count=1 ./...

test-pkg: ## Test a single module (PKG=dir)
	@test -n "$(PKG)" || (echo "Usage: make test-pkg PKG=<directory>" && exit 1)
	@echo "==> Testing $(PKG)..."
	@cd $(PKG) && $(GO) test -count=1 -v ./...

test-cover: ## Run tests with coverage for all modules
	@echo "==> Testing with coverage..."
	@for dir in $(GO_MOD_DIRS); do \
		echo "  → $$dir"; \
		(cd "$$dir" && $(GO) test -count=1 -cover ./... 2>&1) || true; \
	done

# ──────────────────────────────────────────────
# Code quality
# ──────────────────────────────────────────────

.PHONY: vet fmt fmt-check lint check
vet: ## Run go vet for all modules
	@echo "==> Running go vet..."
	@$(GO) vet ./...

fmt: ## Format all Go source files
	@echo "==> Formatting..."
	@gofmt -s -w $(shell find . -name '*.go' -not -path './vendor/*')

fmt-check: ## Check formatting without modifying files
	@echo "==> Checking formatting..."
	@unformatted=$$(gofmt -s -l $(shell find . -name '*.go' -not -path './vendor/*')); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi; \
	echo "All files are properly formatted."

lint: ## Run golangci-lint (if installed)
	@echo "==> Running golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is not installed"; exit 1; }
	@golangci-lint run --timeout 10m ./...

vuln: ## Run govulncheck on all modules (if installed)
	@echo "==> Running govulncheck..."
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "govulncheck is not installed. Install with:"; \
		echo "  go install golang.org/x/vuln/cmd/govulncheck@latest"; \
		exit 1; \
	}
	@for dir in $(GO_MOD_DIRS); do \
		echo "  → $$dir"; \
		(cd "$$dir" && govulncheck ./... 2>&1) || true; \
	done

check: fmt-check vet build test ## Run fmt-check + vet + build + test (CI equivalent)
	@echo "==> All checks passed!"

check-all: fmt-check vet build test lint vuln ## Run all checks including lint + vuln
	@echo "==> All checks (including lint + vuln) passed!"

# ──────────────────────────────────────────────
# Coverage (HTML)
# ──────────────────────────────────────────────

.PHONY: cover-html
cover-html: ## Generate HTML coverage report for a single module (PKG=dir)
	@test -n "$(PKG)" || (echo "Usage: make cover-html PKG=<directory>" && exit 1)
	@echo "==> Generating HTML coverage for $(PKG)..."
	@cd $(PKG) && $(GO) test -count=1 -coverprofile=coverage.out ./... && \
		$(GO) tool cover -html=coverage.out -o coverage.html && \
		echo "Coverage report: $(PKG)/coverage.html"

# ──────────────────────────────────────────────
# Module listing
# ──────────────────────────────────────────────

.PHONY: modules
modules: ## List all Go modules in the workspace
	@echo "Go modules in workspace:"
	@for dir in $(GO_MOD_DIRS); do \
		mod=$$(head -1 "$$dir/go.mod" | awk '{print $$2}'); \
		echo "  $$dir  ($$mod)"; \
	done

# ──────────────────────────────────────────────
# Tag management
# ──────────────────────────────────────────────

.PHONY: tags tags-pkg
tags: ## List all git tags grouped by module
	@echo "All git tags:"
	@git tag --sort=v:refname

tags-pkg: ## List tags for a specific module (PKG=dir)
	@test -n "$(PKG)" || (echo "Usage: make tags-pkg PKG=<directory>" && exit 1)
	@if [ "$(PKG)" = "." ]; then \
		prefix=""; \
	else \
		prefix="$(PKG)/"; \
	fi; \
	git tag --list "$${prefix}v*" --sort=v:refname

# ──────────────────────────────────────────────
# Release — version bump + tag
# ──────────────────────────────────────────────

.PHONY: release-patch release-minor release-major release-all
release-patch: ## Bump patch version for a module (PKG=dir)
	@$(MAKE) _release PKG=$(PKG) LEVEL=patch

release-minor: ## Bump minor version for a module (PKG=dir)
	@$(MAKE) _release PKG=$(PKG) LEVEL=minor

release-major: ## Bump major version for a module (PKG=dir)
	@$(MAKE) _release PKG=$(PKG) LEVEL=major

release-all: ## Auto-bump all changed modules (LEVEL=patch|minor|major)
	@echo "==> Detecting changed modules since their last tag..."
	@bash $(ROOT)/scripts/release.sh --all --level $(RELEASE_LEVEL)

# Internal release target.
.PHONY: _release
_release:
	@test -n "$(PKG)" || (echo "Usage: make release-{patch,minor,major} PKG=<directory>" && exit 1)
	@test -d "$(PKG)" || (echo "Directory not found: $(PKG)" && exit 1)
	@bash $(ROOT)/scripts/release.sh --pkg "$(PKG)" --level $(LEVEL)

# ──────────────────────────────────────────────
# Push tags
# ──────────────────────────────────────────────

.PHONY: push-tags push-tag
push-tags: ## Push all tags to remote
	@echo "==> Pushing all tags to remote..."
	@git push origin --tags

push-tag: ## Push a specific tag (TAG=name)
	@test -n "$(TAG)" || (echo "Usage: make push-tag TAG=<tag-name>" && exit 1)
	@echo "==> Pushing tag $(TAG)..."
	@git push origin "$(TAG)"

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

.PHONY: clean
clean: ## Remove generated coverage files and demo binaries
	@echo "==> Cleaning..."
	@find . -name 'coverage.out' -delete 2>/dev/null || true
	@find . -name 'coverage.html' -delete 2>/dev/null || true
	@find ./example/cmd -type f -perm +111 ! -name '*.go' -delete 2>/dev/null || true
	@echo "Done."

# ── lingcli 构建 ──

# full 模式需要嵌入 ling-base 源码。
# 构建前运行此 target 将源码同步到 lingcli/embed_source/。
# CI/发布构建自动调用此 target。
EMBED_SOURCE_DIRS := middleware common/response common/response/gin \
	common/jwtutil common/jwtutil/gin common/limiter common/limiter/count \
	common/limiter/tokenbucket common/limiter/keycount common/circuitbreaker \
	common/crypto common/logger common/logger/gin common/constants bootstrap common/eventbus version \
	apidocs apidocs/humax apidocs/assets i18n i18n/gin common/metrics common/tracing \
	common/validate

.PHONY: prepare-cli-embed clean-cli-embed build-cli
prepare-cli-embed: ## Sync ling-base source to lingcli/embed_source/ for full mode
	@echo "==> Preparing lingcli embedded source..."
	@rm -rf lingcli/embed_source/*
	@touch lingcli/embed_source/.gitkeep
	@for dir in $(EMBED_SOURCE_DIRS); do \
		if [ -d "$$dir" ]; then \
			mkdir -p "lingcli/embed_source/$$dir"; \
			find "$$dir" -maxdepth 1 -type f \( -name '*.go' ! -name '*_test.go' -o -name '*.css' -o -name '*.svg' -o -name '*.png' -o -name '*.font' \) -exec cp {} "lingcli/embed_source/$$dir/" \; 2>/dev/null; \
		fi; \
	done
	@echo "  Embedded source: $$(find lingcli/embed_source -type f ! -name '.gitkeep' | wc -l | tr -d ' ') files"
	@echo "Done."

clean-cli-embed: ## Clean lingcli embedded source
	@rm -rf lingcli/embed_source/*
	@touch lingcli/embed_source/.gitkeep
	@echo "==> Cleaned lingcli embedded source."

build-cli: prepare-cli-embed ## Build lingcli binary with embedded source
	@echo "==> Building lingcli..."
	@cd lingcli && go build -o ../bin/lingcli .
	@echo "  Binary: bin/lingcli"
	@echo "Done."
