# Tmux Client Development Guidelines

## Overview

The tmux-client is a Go service that provides a REST API for interacting with tmux sessions, files, and commands. It enables interactive terminal access with audit logging.

## Prerequisites

- Go 1.21+
- tmux installed
- Docker (optional, for containerized deployment)

## Development Scripts

### Build

```bash
# Build the binary
make build

# Output: bin/tmux-client
```

### Run

```bash
# Run the server directly (uses config.yaml)
make run

# Or run manually
go run ./cmd/server

# Copy and configure before first run
cp config.example.yaml config.yaml
```

### Test

```bash
# Run all tests
make test

# Run tests with race detection
go test -v -race ./...

# Run tests with coverage
make test-coverage

# View coverage report (opens in browser)
go tool cover -html=coverage.out
```

### Code Quality

```bash
# Run all checks (fmt, vet, lint)
make check

# Format code with gofumpt
make fmt

# Run go vet
make vet

# Run golangci-lint
make lint
```

### Dependencies

```bash
# Download dependencies
make deps

# Tidy and verify go.mod
make tidy
```

### Code Generation

```bash
# Generate OpenAPI/Swagger documentation
make generate-openapi

# Run all code generation
make generate
```

### Docker

```bash
# Build Docker image
make docker-build

# Run with docker-compose (from repo root)
docker-compose up tmux-client --build
```

### Install Development Tools

```bash
# Install gofumpt, golangci-lint, and swag
make install-tools
```

## Configuration

The service is configured via `config.yaml`. See `config.example.yaml` for all available options:

- Server settings (host, port, TLS)
- Tmux tool configuration (allowed keys, max lines)
- File tool configuration (root directory, allowed/denied paths)
- Command tool configuration (allowed/denied commands)
- Human approval settings
- Audit logging configuration

## Project Structure

```
tmux-client/
├── cmd/server/          # Server entry point
├── internal/            # Internal packages
├── scripts/             # Utility scripts (fmt.sh, lint.sh, vet.sh)
├── docs/                # Generated OpenAPI docs
├── config.example.yaml  # Example configuration
├── config.yaml          # Local configuration (git-ignored)
├── Makefile             # Build commands
└── Dockerfile           # Container build
```

## API Port

The tmux-client runs on port **8081** by default.

## Makefile Reference

Run `make help` to see all available targets:

| Target | Description |
|--------|-------------|
| `make build` | Build the tmux-client binary |
| `make run` | Run the tmux-client server |
| `make clean` | Clean build artifacts |
| `make fmt` | Format code with gofumpt |
| `make lint` | Run golangci-lint |
| `make vet` | Run go vet |
| `make test` | Run tests |
| `make test-coverage` | Run tests with coverage |
| `make check` | Run all code quality checks |
| `make deps` | Download dependencies |
| `make tidy` | Tidy and verify dependencies |
| `make generate-openapi` | Generate OpenAPI docs |
| `make install-tools` | Install dev tools |
| `make docker-build` | Build Docker image |