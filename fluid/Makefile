# Fluid CLI Makefile

.PHONY: all build build-cli build-api run clean fmt vet test test-coverage check deps tidy help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

# Binary names and paths
CLI_BINARY=fluid
CLI_PATH=./cmd/fluid

# Build flags
LDFLAGS=-s -w
TAGS=libvirt
BUILD_FLAGS=-ldflags "$(LDFLAGS)" -tags "$(TAGS)"


# Default target: build the CLI
all: fmt vet test build

## Build targets

build:
	@echo "Building fluid CLI..."
	@rm -f bin/$(CLI_BINARY)
	@mkdir -p bin
	$(GOBUILD) $(BUILD_FLAGS) -o bin/$(CLI_BINARY) $(CLI_PATH)


run: build-cli ## Build and run the CLI
	./bin/$(CLI_BINARY)

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GOCMD) clean

## Code quality targets

fmt: ## Format code with gofmt
	@echo "Formatting code..."
	gofmt -w .

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

check: fmt vet ## Run all code quality checks (fmt, vet)

## Dependency management

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

tidy: ## Tidy and verify dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	$(GOMOD) verify

## Installation

install: build-cli ## Install fluid CLI to GOPATH/bin
	@echo "Installing fluid to GOPATH/bin..."
	@cp bin/$(CLI_BINARY) $(GOPATH)/bin/$(CLI_BINARY)

## Help

help: ## Show this help message
	@echo "Fluid CLI Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
