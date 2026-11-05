SHELL := /bin/bash
SERVICES := core-api identity consumer scheduler
IMAGES   := $(SERVICES) migrate web
REGISTRY ?= nocu
TAG      ?= latest
BIN      := bin

# Load .env when present so `make run-*` and `make migrate` work locally.
ifneq (,$(wildcard .env))
  include .env
  export
endif

.PHONY: help build build-migrate run-% dev dev-% migrate migrate-down migrate-status \
        test test-race cover lint fmt tidy docker-build up down logs clean \
        web-install dev-web web-build web-lint web-test web-typecheck

help: ## Show targets
	@grep -hE '^[a-zA-Z_%-]+:.*?## .*$$' $(firstword $(MAKEFILE_LIST)) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: $(addprefix build-,$(SERVICES)) build-migrate ## Build every binary into ./bin

build-migrate:
	@mkdir -p $(BIN)
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN)/migrate ./cmd/migrate

build-%:
	@mkdir -p $(BIN)
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN)/$* ./services/$*/cmd

run-%: ## Run a service locally, e.g. make run-core-api
	go run ./services/$*/cmd

dev-%: ## Hot reload one service with Air, e.g. make dev-core-api
	@command -v air >/dev/null 2>&1 || { echo "air not found: go install github.com/air-verse/air@latest"; exit 1; }
	@mkdir -p tmp/air/$*
	air -c .air/$*.toml

dev: ## Hot reload every service in one terminal (Ctrl+C stops all)
	@command -v air >/dev/null 2>&1 || { echo "air not found: go install github.com/air-verse/air@latest"; exit 1; }
	@mkdir -p $(addprefix tmp/air/,$(SERVICES))
	@trap 'kill 0' INT TERM; \
	for s in $(SERVICES); do air -c .air/$$s.toml & done; \
	wait

migrate: ## Apply pending migrations
	go run ./cmd/migrate up

migrate-down: ## Roll back the last migration
	go run ./cmd/migrate down

migrate-status: ## Show migration state
	go run ./cmd/migrate status

test: web-test ## Run unit tests (Go + web)
	go test ./...

test-race: ## Run unit tests with the race detector
	go test -race ./...

cover: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

lint: web-lint ## gofmt check + go vet + web lint/typecheck
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)
	go vet ./...

fmt: ## Format code
	gofmt -w .

tidy: ## Tidy go.mod
	go mod tidy

docker-build: $(addprefix docker-build-,$(IMAGES)) ## Build every image

docker-build-%: ## Build one image, e.g. make docker-build-core-api
	docker build -f build/Dockerfile.$* -t $(REGISTRY)/$*:$(TAG) .

up: ## Start the full stack with docker compose
	docker compose up --build -d

down: ## Stop the stack
	docker compose down

logs: ## Tail service logs
	docker compose logs -f core-api identity consumer scheduler

clean: ## Remove build artefacts
	rm -rf $(BIN) tmp coverage.out web/.next

# ---- web (Next.js) -------------------------------------------------------
# pnpm workspace at the repo root; the app lives in web/.

web-install: ## Install web dependencies (pnpm)
	@command -v pnpm >/dev/null 2>&1 || { echo "pnpm not found: corepack enable && corepack prepare pnpm@latest --activate"; exit 1; }
	pnpm install --frozen-lockfile

dev-web: ## Run the web app with hot reload on :3000
	pnpm --filter web dev

web-build: ## Production build of the web app
	pnpm --filter web build

web-lint: ## ESLint + tsc for the web app
	pnpm --filter web lint
	pnpm --filter web typecheck

web-test: ## Vitest unit tests for the web app
	pnpm --filter web test
