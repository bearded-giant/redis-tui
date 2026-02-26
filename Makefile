# Redis TUI Manager Makefile

APP_NAME := redis-tui
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build install clean test lint run release snapshot

all: build

## Build the application
build:
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./

## Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./

## Clean build artifacts
clean:
	rm -rf bin/
	rm -rf dist/

## Run tests
test:
	go test -v ./...

## Run tests with coverage
test-cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## Run linter
lint:
	go vet ./...

## Format code
fmt:
	go fmt ./...

## Run the application
run:
	go run ./

## Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 ./
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 ./
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 ./
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-arm64 ./
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe ./

## Create a release with goreleaser
release:
	goreleaser release --clean

## Create a snapshot release (no publish)
snapshot:
	goreleaser release --snapshot --clean

## Install development dependencies
dev-deps:
	go install github.com/goreleaser/goreleaser/v2@v2.13.1

## Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  install     - Install to GOPATH/bin"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  test-cover  - Run tests with coverage"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo "  run         - Run the application"
	@echo "  build-all   - Build for multiple platforms"
	@echo "  release     - Create a release with goreleaser"
	@echo "  snapshot    - Create a snapshot release"
	@echo "  dev-deps    - Install development dependencies"
	@echo "  help        - Show this help"
