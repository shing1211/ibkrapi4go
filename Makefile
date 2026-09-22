# Copyright 2026 shing1211
# SPDX-License-Identifier: Apache-2.0

SHELL := /bin/bash
GO ?= go
LICENSE_HOLDER ?= shing1211
LICENSE_YEAR ?= 2026
OAPI_CODEGEN_VERSION ?= v2.8.0

.DEFAULT_GOAL := help

.PHONY: help tools fmt vet test test-race test-integration coverage check \
        codegen codegen-verify docs-spec docs-check \
        license license-check mock-gateway clean

help: ## List targets
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

tools: ## Install build/test tools
	$(GO) install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)
	$(GO) install github.com/google/addlicense@latest

fmt: ## Format Go sources
	@if [ -f go.mod ]; then gofmt -s -w .; else echo "no go.mod yet; skipping fmt"; fi

vet: ## Run go vet
	@if [ -f go.mod ]; then $(GO) vet ./...; else echo "no go.mod yet; skipping vet"; fi

test: ## Run unit tests
	@if [ -f go.mod ]; then $(GO) test ./... -count=1; else echo "no go.mod yet; skipping tests"; fi

test-race: ## Run tests with the race detector
	@if [ -f go.mod ]; then $(GO) test ./... -race -count=1; else echo "no go.mod yet; skipping"; fi

test-integration: ## Run integration tests against a real gateway (requires IBKR_GATEWAY, IBKR_USERNAME, IBKR_PASSWORD)
	$(GO) test --tags=integration ./test/...

coverage: ## Write coverage.out and coverage.html
	@if [ -f go.mod ]; then \
		$(GO) test ./... -coverprofile=coverage.out -count=1 && \
		$(GO) tool cover -html=coverage.out -o coverage.html && \
		echo "wrote coverage.out and coverage.html"; \
	else echo "no go.mod yet; skipping"; fi

check: fmt vet money-check test ## Format, vet, check money types, and test

money-check: ## Fail if pkg/ibkr exposes float money fields (ADR 0008)
	python3 scripts/check_money.py

codegen: ## Regenerate client/ from the OpenAPI spec
	./scripts/codegen.sh

codegen-verify: ## Fail if generated code drifts from committed output
	./scripts/validate_codegen.sh

docs-spec: ## Regenerate docs/SPEC.md from the spec
	python3 scripts/gen_spec_index.py specs/ibkr_spec.json > docs/SPEC.md

docs-check: ## Check markdown links, README translations, and design doc accuracy
	python3 scripts/check_links.py
	python3 scripts/check_i18n.py
	go run ./scripts/check_design

mock-gateway: ## Run the standalone mock IBKR gateway
	$(GO) run ./cmd/ibkr-mock-gateway

license: ## Apply SPDX headers to sources
	@if [ -f go.mod ]; then \
		dirs="$$(for d in scripts pkg internal cmd; do [ -d "$$d" ] && echo "$$d"; done)"; \
		$(GO) run github.com/google/addlicense@latest -c "$(LICENSE_HOLDER)" -y "$(LICENSE_YEAR)" \
			-l apache -s=only $$dirs ; \
	else echo "no go.mod yet; run scripts manually"; fi

license-check: ## Verify SPDX headers are present
	@if [ -f go.mod ]; then \
		dirs="$$(for d in scripts pkg internal cmd; do [ -d "$$d" ] && echo "$$d"; done)"; \
		$(GO) run github.com/google/addlicense@latest -check $$dirs ; \
	else echo "no go.mod yet; skipping"; fi

clean: ## Remove build artifacts
	rm -f coverage.out coverage.html
	rm -rf "$(CURDIR)/client/client.gen.go.tmp"
