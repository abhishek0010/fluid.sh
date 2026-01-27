# fluid-remote API Makefile

.PHONY: all build run clean fmt lint vet test check generate-openapi help install-tools

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod
BINARY_NAME=fluid-remote
CMD_PATH=./cmd/api
TAGS=libvirt

# PostHog key (empty by default for dev builds)
POSTHOG_KEY ?=

# Build flags
LDFLAGS=-s -w
ifneq ($(POSTHOG_KEY),)
  LDFLAGS += -X github.com/aspectrr/fluid.sh/fluid-remote/internal/telemetry.posthogAPIKey=$(POSTHOG_KEY)
endif
BUILD_FLAGS=-ldflags "$(LDFLAGS)" -tags "$(TAGS)"

# Default target
all: check build

## Build targets

build: ## Build the API binary
	$(GOBUILD) $(BUILD_FLAGS) -o bin/$(BINARY_NAME) $(CMD_PATH)

run: ## Run the API server
	$(GORUN) $(CMD_PATH)

clean: ## Clean build artifacts
	rm -rf bin/
	$(GOCMD) clean

## Code quality targets

fmt: ## Format code with gofumpt
	@echo "Formatting code..."
	gofumpt -w .

lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run --build-tags=$(TAGS) ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) -tags=$(TAGS) ./...

test: ## Run tests
	$(GOTEST) -v -race -tags=$(TAGS) ./...

test-coverage: ## Run tests with coverage
	$(GOTEST) -v -race -tags=$(TAGS) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

check: fmt vet lint ## Run all code quality checks (fmt, vet, lint)

## Dependency management

deps: ## Download dependencies
	$(GOMOD) download

tidy: ## Tidy and verify dependencies
	$(GOMOD) tidy
	$(GOMOD) verify

## Code generation

generate-openapi: ## Generate OpenAPI/Swagger documentation
	./scripts/generate-openapi.sh

generate: generate-openapi ## Run all code generation

## Development setup

install-tools: ## Install development tools (gofumpt, golangci-lint, swag)
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest

## Docker targets

docker-build: ## Build Docker image
	docker build -f Dockerfile -t fluid-remote-api .

## Help

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
