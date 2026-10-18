PROJECT_NAME := UltimaVRedux
BIN_DIR := bin
BINARY := $(BIN_DIR)/ultimav
MAIN_PACKAGE := ./cmd/ultimav

GO ?= go
PYTHON ?= python3
TOOLS_BIN := $(CURDIR)/$(BIN_DIR)/tools
GOIMPORTS ?= $(TOOLS_BIN)/goimports
GOLANGCI_LINT ?= $(TOOLS_BIN)/golangci-lint
LINT_PACKAGES ?= ./...
TEST_PROFILE ?= host
TEST_GATE_FLAGS = -profile=$(TEST_PROFILE) $(if $(filter 1,$(ULTIMA5_REQUIRE_DATA)),-strict,)
TEST_PROFILE_CPUS = $(if $(filter throughput,$(TEST_PROFILE)),4,2)
TEST_PROFILE_MEMORY = $(if $(filter throughput,$(TEST_PROFILE)),2GiB,256MiB)
TEST_PROFILE_TARGETS := test test-render test-all test-race test-provisioned test-race-provisioned lint lint-all check validate quality-gate

# Capture original Windows environment names before target exports can mask them.
ifeq ($(OS),Windows_NT)
TEST_PROFILE_CASE_KEYS := $(shell env | awk -F= 'toupper($$1) ~ /^(GOMAXPROCS|GOMEMLIMIT)$$/ && $$1 != toupper($$1) { print $$1 }')
endif

CONVERSATION_DATA_DIR ?= ./OLD
CONVERSATION_WORKSPACE ?= $(HOME)/.ultima_v_redux/conversations
CONVERSATION_PATCH_DIR ?= data/conversation-patches

.DEFAULT_GOAL := help

# Serialize Make stages even with inherited -j, without limiting host package
# compilation or the non-graphics lane. Separate Make processes need one owner.
.NOTPARALLEL:

$(TEST_PROFILE_TARGETS): export TEST_PROFILE := $(TEST_PROFILE)
ifneq ($(TEST_PROFILE),host)
$(TEST_PROFILE_TARGETS): export GOMAXPROCS ?= $(TEST_PROFILE_CPUS)
$(TEST_PROFILE_TARGETS): export GOMEMLIMIT ?= $(TEST_PROFILE_MEMORY)
$(TEST_PROFILE_TARGETS): export GOGC ?= 50
$(TEST_PROFILE_TARGETS): export TEST_GO_BUILD_FLAGS := -p=1
endif
# Host leaves Go resource defaults and explicit caller settings intact. Keep
# GOFLAGS untouched, including settings persisted in the user's GOENV file.
$(TEST_PROFILE_TARGETS): test-profile-check

.PHONY: test-profile-check
test-profile-check:
	@case "$(TEST_PROFILE)" in host|constrained|throughput) ;; *) \
		printf 'Unknown TEST_PROFILE: %s; use host, constrained or throughput\n' "$(TEST_PROFILE)" >&2; exit 1;; esac; \
	if [ "$(TEST_PROFILE)" = host ]; then \
		printf 'TEST_PROFILE=host GOMAXPROCS=%s GOMEMLIMIT=%s GOGC=%s (caller settings or Go defaults; logic uses available Go CPUs, native -p=1 -parallel=1)\n' \
			"$${GOMAXPROCS:-auto}" "$${GOMEMLIMIT:-off}" "$${GOGC:-100}"; exit 0; \
	fi; \
	if [ -n "$(TEST_PROFILE_CASE_KEYS)" ]; then \
		printf 'Windows resource variables must use uppercase names before Make: %s\n' "$(TEST_PROFILE_CASE_KEYS)" >&2; exit 1; \
	fi; \
	for setting in "GOMAXPROCS=$$GOMAXPROCS:$(TEST_PROFILE_CPUS)" "GOMEMLIMIT=$$GOMEMLIMIT:$(TEST_PROFILE_MEMORY)"; do \
		name=$${setting%%=*}; values=$${setting#*=}; actual=$${values%:*}; expected=$${values##*:}; \
		if [ "$$actual" != "$$expected" ]; then \
			printf 'TEST_PROFILE=%s requires %s=%s, inherited %s; unset it or select matching settings\n' \
				"$(TEST_PROFILE)" "$$name" "$$expected" "$$actual" >&2; exit 1; \
		fi; \
	done

.PHONY: help all dev run tools format format-check lint lint-all lint-config-check core-dependencies-check vet test test-short test-race test-provisioned test-race-provisioned test-integration generate web-install web-build web-test-browser build audio-editor build-check check pre-commit policy-check docs-check gamefacts-check tooling-test validate quality-gate conversations-init conversations-diff conversations-check conversations-patches conversations-verify clean

help: ## Show available targets
	@printf "$(PROJECT_NAME) development targets:\n\n"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: check build ## Run checks and build the game

dev: build ## Build and play the game (pass options with ARGS="...")
	@./$(BINARY) $(ARGS)

run: ## Run the existing game binary (pass options with ARGS="...")
	@./$(BINARY) $(ARGS)

tools: ## Provision pinned Go validation tools and module dependencies (networked setup)
	@mkdir -p "$(TOOLS_BIN)"
	@$(GO) mod download
	@GOBIN="$(TOOLS_BIN)" $(GO) install golang.org/x/tools/cmd/goimports@v0.37.0
	@GOBIN="$(TOOLS_BIN)" $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2

format: ## Format Go source and organize imports
	@$(GOIMPORTS) -w .

format-check: ## Report Go files requiring formatting
	@files="$$($(GOIMPORTS) -l .)" || exit $$?; \
	test -z "$$files" || { \
		printf "Files require formatting; run 'make format':\n"; \
		printf '%s\n' "$$files"; \
		exit 1; \
	}

lint-config-check: ## Verify the pinned linter schema and executable errcheck settings
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/check-lint-config "$(GOLANGCI_LINT)"

lint: lint-config-check ## Run golangci-lint using the repository configuration
	@$(PYTHON) scripts/lint_gate.py --profile "$(TEST_PROFILE)" --linter "$(GOLANGCI_LINT)" --go "$(GO)" $(LINT_PACKAGES)
	@$(PYTHON) scripts/audio_module_gate.py --lint

lint-all: lint-config-check ## Report all configured lint findings without revision filtering or output caps
	@$(PYTHON) scripts/lint_gate.py --profile "$(TEST_PROFILE)" --linter "$(GOLANGCI_LINT)" --go "$(GO)" --all $(LINT_PACKAGES)
	@$(PYTHON) scripts/audio_module_gate.py --lint

core-dependencies-check: ## Reject transitive graphics/input dependencies in core code and tests
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/check-core-dependencies

vet: ## Run Go static analysis
	@$(GO) vet $(TEST_GO_BUILD_FLAGS) ./...

.PHONY: test-render test-all
test: ## Run uncached logic tests without a display
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/test-gate $(TEST_GATE_FLAGS) -suite=logic

test-render: ## Run required native/Ebitengine-linked tests (use Xvfb on headless Linux)
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/test-gate $(TEST_GATE_FLAGS) -suite=render

test-all: ## Run aggregate logic and native tests
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/test-gate $(TEST_GATE_FLAGS) -suite=all

test-short: ## Run unit tests without integration tests
	@$(GO) test -short ./... -timeout 90s

test-race: ## Run all tests with the race detector
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/test-gate $(TEST_GATE_FLAGS) -suite=all -race

test-provisioned: ## Require explicit original data and zero skipped tests
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/test-gate $(TEST_GATE_FLAGS) -suite=all -strict

test-race-provisioned: ## Require explicit original data and zero skips with the race detector
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/test-gate $(TEST_GATE_FLAGS) -suite=all -strict -race

test-integration: ## Run integration tests
	@$(GO) test -run Integration ./...

generate: ## Run Go code generation
	@$(GO) generate ./...

web-install: ## Install reproducible debug workbench frontend dependencies
	@npm --prefix internal/debugging/webui/client ci

web-build: web-install ## Type-check and build embedded debug workbench assets
	@npm --prefix internal/debugging/webui/client test
	@npm --prefix internal/debugging/webui/client run build

web-test-browser: ## Run Workbench Chromium regressions (browser must be preinstalled)
	@npm --prefix internal/debugging/webui/client run test:browser

build: ## Build the game into bin/ultimav using shipped assets
	@mkdir -p $(BIN_DIR)
	@$(GO) build -o $(BINARY) $(MAIN_PACKAGE)

audio-editor: ## Build the standalone Bubble Tea audio editor into bin/audio-editor
	@mkdir -p $(BIN_DIR)
	@$(GO) build -o $(BIN_DIR)/audio-editor ./cmd/audio-editor

.PHONY: audio-module-check
audio-module-check: ## Check the game-owned Berad Audio bridge against the pinned SDK revision
	@$(GO) test $(TEST_GO_BUILD_FLAGS) -count=1 -run '^TestBeradAudioModule$$' ./scripts/check-audio-module
	@$(PYTHON) scripts/audio_module_gate.py --lint

.PHONY: vm-settings-proof
vm-settings-proof: ## Run the opt-in Ebitengine VM keyboard/settings rollback proof (requires original data and graphics)
	@mkdir -p $(BIN_DIR)
	@$(GO) test -p=1 -c -tags=ebitenginevmguest,settingsproof -o $(BIN_DIR)/settings-vm-guest ./cmd/ultimav
	@$(GO) build -p=1 -o $(BIN_DIR)/vm-settings-proof ./scripts/vm-settings-proof
	@./$(BIN_DIR)/vm-settings-proof

check: format-check lint vet test ## Run formatting, lint, vet, and logic tests

pre-commit: ## Run the repository pre-commit validation script
	@./scripts/pre-commit-checks.sh

policy-check: ## Run policy checks without repeating formatting, vet, build, or tests
	@./scripts/pre-commit-checks.sh --policy-only

build-check: ## Compile every package without generating application assets
	@$(GO) build $(TEST_GO_BUILD_FLAGS) ./...

docs-check: ## Validate retained documentation links, catalog, ownership, and authority
	@$(GO) run $(TEST_GO_BUILD_FLAGS) ./scripts/check-documentation

gamefacts-check: ## Validate public stable references, graph, catalog, and code locators
	@$(PYTHON) scripts/check-gamefacts.py

tooling-test: ## Run pure Python developer-tool regressions
	@$(PYTHON) -m unittest discover -s scripts -p 'test_*.py'

validate: check policy-check build-check docs-check gamefacts-check tooling-test ## Run logic tests and canonical static checks once

# Provision with make tools first. Validation must not download missing tools/modules.
validate: export GOPROXY = off
validate: export GOSUMDB = off
validate: export GOTOOLCHAIN = local

quality-gate: ## Run the local checks required before creating or merging a PR
	@./scripts/quality-gate.sh

conversations-init: ## Create the private original-conversation editing workspace
	@$(GO) run ./cmd/conversation-tool workspace init --data-dir "$(CONVERSATION_DATA_DIR)" --workspace "$(CONVERSATION_WORKSPACE)" --patch-dir "$(CONVERSATION_PATCH_DIR)"

conversations-diff: ## Print private base-to-edited conversation diffs
	@$(GO) run ./cmd/conversation-tool workspace diff --workspace "$(CONVERSATION_WORKSPACE)"

conversations-check: ## Validate the private conversation workspace
	@$(GO) run ./cmd/conversation-tool workspace check --workspace "$(CONVERSATION_WORKSPACE)"

conversations-patches: ## Regenerate cumulative tracked conversation patches
	@$(GO) run ./cmd/conversation-tool workspace patches --workspace "$(CONVERSATION_WORKSPACE)" --patch-dir "$(CONVERSATION_PATCH_DIR)"

conversations-verify: ## Verify tracked patches reproduce all private edits
	@$(GO) run ./cmd/conversation-tool workspace verify --workspace "$(CONVERSATION_WORKSPACE)" --patch-dir "$(CONVERSATION_PATCH_DIR)"

clean: ## Remove generated build artifacts
	@rm -f $(BINARY)
