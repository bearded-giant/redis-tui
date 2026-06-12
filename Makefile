# Redis TUI Manager Makefile
SHELL := /bin/bash
APP_NAME := redis-tui
PREFIX  ?= $(HOME)/.local
INSTALL_DIR := $(PREFIX)/bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build install uninstall dev-install clean test test-cover lint fmt run dev release snapshot decode-blob help \
	docker-up docker-down docker-seed \
	docker-up-standalone docker-up-standalone-stack docker-up-cluster docker-up-cluster-stack \
	docker-down-standalone docker-down-standalone-stack docker-down-cluster docker-down-cluster-stack \
	docker-seed-standalone docker-seed-standalone-stack docker-seed-cluster docker-seed-cluster-stack

all: build

build: ## Build the application to bin/redis-tui
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./

decode-blob: ## Build cmd/decode-blob CLI for one-off blob inspection
	go build -o bin/decode-blob ./cmd/decode-blob

install: build ## Build + install to $(INSTALL_DIR) (defaults to ~/.local/bin)
	@mkdir -p $(INSTALL_DIR)
	install -m 0755 bin/$(APP_NAME) $(INSTALL_DIR)/$(APP_NAME)
	@echo "installed -> $(INSTALL_DIR)/$(APP_NAME)  (version=$(VERSION))"

dev-install: install ## Alias for install — fast iteration loop: edit, make dev-install, redis-tui
	@true

uninstall: ## Remove $(INSTALL_DIR)/$(APP_NAME)
	rm -f $(INSTALL_DIR)/$(APP_NAME)
	@echo "removed -> $(INSTALL_DIR)/$(APP_NAME)"

clean: ## Remove bin/ and dist/
	rm -rf bin/
	rm -rf dist/

test: ## Run tests with race detector
	go test -v -race ./...

test-cover: ## Run tests with HTML coverage report (coverage.html)
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run go vet
	go vet ./...

fmt: ## Format code with go fmt
	go fmt ./...

run: ## Run the application
	go run ./

dev: docker-up-standalone docker-seed-standalone run ## Boot standalone Redis (:6379), seed, run app

release: ## Tag-driven release via goreleaser (requires VERSION tag pushed)
	goreleaser release --clean

snapshot: ## Local cross-platform build via goreleaser (no publish, output in dist/)
	goreleaser release --snapshot --clean

## --- Docker Examples ---

docker-up: docker-up-standalone docker-up-standalone-stack docker-up-cluster docker-up-cluster-stack ## Start all example Redis instances

docker-down: docker-down-standalone docker-down-standalone-stack docker-down-cluster docker-down-cluster-stack ## Stop all example Redis instances

docker-seed: docker-seed-standalone docker-seed-standalone-stack docker-seed-cluster docker-seed-cluster-stack ## Seed all running example instances

docker-up-standalone: ## Start standalone Redis (redis:7-alpine on :6379)
	docker compose -f examples/standalone/docker-compose.yml up -d

docker-down-standalone: ## Stop standalone Redis
	docker compose -f examples/standalone/docker-compose.yml down

docker-seed-standalone: ## Seed standalone Redis with example data
	go run ./examples/seed -flush

docker-up-standalone-stack: ## Start standalone Redis Stack (redis-stack on :6390)
	docker compose -f examples/standalone-redis-stack/docker-compose.yml up -d

docker-down-standalone-stack: ## Stop standalone Redis Stack
	docker compose -f examples/standalone-redis-stack/docker-compose.yml down

docker-seed-standalone-stack: ## Seed standalone Redis Stack with example data
	go run ./examples/seed -addr localhost:6390 -flush

docker-up-cluster: ## Start cluster (redis:7-alpine on :6380-6385)
	docker compose -f examples/cluster/docker-compose.yml up -d

docker-down-cluster: ## Stop cluster
	docker compose -f examples/cluster/docker-compose.yml down

docker-seed-cluster: ## Seed cluster with example data
	go run ./examples/seed -addr localhost:6380 -cluster -flush

docker-up-cluster-stack: ## Start cluster Redis Stack (redis-stack on :6386-6392)
	docker compose -f examples/cluster-redis-stack/docker-compose.yml up -d

docker-down-cluster-stack: ## Stop cluster Redis Stack
	docker compose -f examples/cluster-redis-stack/docker-compose.yml down

docker-seed-cluster-stack: ## Seed cluster Redis Stack with example data
	go run ./examples/seed -addr localhost:6386 -cluster -flush

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "; printf "Targets:\n\n"} \
		/^## --- / {gsub(/^## --- | ---$$/, "", $$0); printf "\n  %s\n", $$0; next} \
		/^[a-zA-Z_-]+:.*?## / {printf "  %-30s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
