# virsh-sandbox API - Development Guide

This is the main virsh-sandbox Go API service that orchestrates KVM/libvirt virtual machines.

## Important Development Notes

### Mandatory Testing

After every code change, tests MUST be created or updated to verify the new behavior:
- Add unit tests in `internal/<package>/*_test.go` files
- Run `make test` to verify all tests pass before considering work complete

### Strict JSON Decoding

The API uses strict JSON decoding (`DisallowUnknownFields()`). This means:
- Adding new fields to request structs requires rebuilding the API
- The running API will reject requests with fields it doesn't recognize
- Always rebuild and restart after modifying request/response DTOs

### Rebuilding After Changes

When modifying the API, you must rebuild for changes to take effect:

```bash
# From repo root - rebuild and restart via docker-compose
docker-compose down && docker-compose up --build -d

# Or rebuild locally
cd virsh-sandbox && make build
```

### ARM Mac (Apple Silicon) Limitations

On ARM Macs using Lima for libvirt:
- VMs may fail to start with "CPU mode 'host-passthrough' not supported" errors
- This is a hypervisor limitation, not a code issue
- The VM will be created but remain in "shut off" state

## Prerequisites

- Go 1.21+
- libvirt/KVM installed and running
- PostgreSQL database
- Development tools (gofumpt, golangci-lint, swag)

## Quick Start

```bash
# Install development tools
make install-tools

# Download dependencies
make deps

# Run all checks and build
make all

# Run the API server
make run
```

## Build Scripts

```bash
# Build the API binary
make build
# Output: bin/virsh-sandbox-api

# Clean build artifacts
make clean

# Build Docker image
make docker-build
```

## Test Scripts

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage
# Generates: coverage.out, coverage.html
```

## Code Quality Scripts

```bash
# Run all checks (format, vet, lint)
make check

# Format code with gofumpt
make fmt
# Or: ./scripts/fmt.sh

# Run go vet
make vet
# Or: ./scripts/vet.sh

# Run golangci-lint
make lint
# Or: ./scripts/lint.sh
```

## Dependency Management

```bash
# Download dependencies
make deps

# Tidy and verify go.mod
make tidy
```

## Code Generation

```bash
# Generate OpenAPI/Swagger documentation
make generate-openapi
# Or: ./scripts/generate-openapi.sh

# Run all code generation
make generate
```

## Development Setup

```bash
# Install development tools (gofumpt, golangci-lint, swag)
make install-tools

# Setup Lima libvirt environment (macOS)
make setup-lima
# Or: ./scripts/setup-lima-libvirt.sh

# Create a test VM for development
make create-test-vm
# Or: ./scripts/create-test-vm.sh
```

## Running Locally

### Environment Variables

```bash
export LOG_FORMAT=text
export LOG_LEVEL=debug
export API_HTTP_ADDR=:8080
export LIBVIRT_URI=qemu:///system
export LIBVIRT_NETWORK=default
export BASE_IMAGE_DIR=/var/lib/libvirt/images/base
export SANDBOX_WORKDIR=/var/lib/libvirt/images/jobs
export DATABASE_URL=postgresql://virsh_sandbox:virsh_sandbox@localhost:5432/virsh_sandbox
export DEFAULT_VCPUS=2
export DEFAULT_MEMORY_MB=2048
export COMMAND_TIMEOUT_SEC=600
export IP_DISCOVERY_TIMEOUT_SEC=120
```

### Run with make

```bash
make run
```

### Run with Docker Compose (from repo root)

```bash
docker-compose up virsh-sandbox --build
```

## All Makefile Targets

Run `make help` to see all available targets:

| Target | Description |
|--------|-------------|
| `all` | Run checks and build (default) |
| `build` | Build the API binary |
| `run` | Run the API server |
| `clean` | Clean build artifacts |
| `fmt` | Format code with gofumpt |
| `lint` | Run golangci-lint |
| `vet` | Run go vet |
| `test` | Run tests |
| `test-coverage` | Run tests with coverage |
| `check` | Run all code quality checks |
| `deps` | Download dependencies |
| `tidy` | Tidy and verify dependencies |
| `generate-openapi` | Generate OpenAPI documentation |
| `generate` | Run all code generation |
| `install-tools` | Install development tools |
| `setup-lima` | Setup Lima libvirt environment |
| `create-test-vm` | Create a test VM |
| `docker-build` | Build Docker image |
| `help` | Show help message |