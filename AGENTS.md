# Virsh Sandbox Project Rules

This is a monorepo containing multiple projects. See project-specific rules below.

## Project References

- @sdk/AGENTS.md - Python SDK for virsh-sandbox API
- @virsh-sandbox/AGENTS.md - Main virsh-sandbox Go service
- @tmux-client/AGENTS.md - Tmux client Go service
- @web/AGENTS.md - Web frontend
- @examples/agent-example/AGENTS.md - Example agent implementation

## Mandatory Testing Requirement

**IMPORTANT**: After every code change, tests MUST be created or updated to verify the new behavior. This applies to all projects in the monorepo.

- For Go services: Add unit tests in `*_test.go` files
- For Python SDK: Add tests in `test/test_client.py`
- For Web frontend: Add component/integration tests as appropriate

Changes without corresponding tests should not be considered complete.

### Building

Please use docker-compose to build and test the services by specifiying the container needed like `docker-compose up virsh-sandbox` or `docker-compose up tmux-client`

### Quick Start with Docker Compose

```bash
# Start all services (recommended for quick setup)
docker-compose up --build

# Start specific services
docker-compose up postgres          # Database only
docker-compose up virsh-sandbox     # API server
docker-compose up tmux-client       # Tmux client
docker-compose up web               # Frontend

# Start in detached mode
docker-compose up -d

# View logs
docker-compose logs -f [service-name]

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### Git Hooks with Lefthook

```bash
# Install lefthook
brew install lefthook  # macOS
# or: go install github.com/evilmartians/lefthook@latest

# Install git hooks
lefthook install

# Run hooks manually
lefthook run pre-commit
```

Pre-commit hooks run:
- `make vet` and `make fmt` for Go services (virsh-sandbox, tmux-client)

## Service Ports

| Service | Port | Description |
|---------|------|-------------|
| virsh-sandbox API | 8080 | Main REST API |
| tmux-client | 8081 | Terminal/file API |
| web | 5173 | React frontend |
| PostgreSQL | 5432 | Database |

## Environment Configuration

Create a `.env` file in the project root to customize settings:

```bash
# Logging
LOG_FORMAT=text
LOG_LEVEL=debug

# API
API_HTTP_ADDR=:8080

# Libvirt/KVM
LIBVIRT_URI=qemu:///system
LIBVIRT_NETWORK=default

# Database
DATABASE_URL=postgresql://virsh_sandbox:virsh_sandbox@localhost:5432/virsh_sandbox

# VM Defaults
DEFAULT_VCPUS=2
DEFAULT_MEMORY_MB=2048
COMMAND_TIMEOUT_SEC=600
IP_DISCOVERY_TIMEOUT_SEC=120
```

## Individual Project Commands

### virsh-sandbox API (Go)

```bash
cd virsh-sandbox
make build          # Build binary
make run            # Run server
make test           # Run tests
make check          # Run all linters
make help           # Show all targets
```

### tmux-client (Go)

```bash
cd tmux-client
make build          # Build binary
make run            # Run server
make test           # Run tests
make check          # Run all linters
make help           # Show all targets
```

### Web Frontend (React/TypeScript)

```bash
cd web
bun install         # Install dependencies
bun run dev         # Start dev server
bun run build       # Build for production
bun run lint        # Run ESLint
bun run generate-api  # Regenerate API client
```

### Python SDK

```bash
cd sdk/virsh-sandbox-py
pip install -r requirements.txt       # Install deps
pytest test/test_client.py -v         # Run tests
mypy virsh_sandbox                    # Type check

# Generate from OpenAPI (from sdk/ directory)
cd sdk && ./scripts/generate.sh
```

### Agent Example (Python)

```bash
cd examples/agent-example
uv sync                    # Install dependencies
uv run python main.py      # Run the agent
```

## macOS Development Setup (Lima)

For macOS, use Lima to run libvirt in a Linux VM:

```bash
# Install Lima and libvirt
brew install lima libvirt

# Set up Lima VM with libvirt
cd virsh-sandbox
./scripts/setup-lima-libvirt.sh --create-test-vm

# Source environment variables
source .env.lima
```

## Full Test Suite

Run tests across all projects:

```bash
# Go services
(cd virsh-sandbox && make test)
(cd tmux-client && make test)

# Python SDK
(cd sdk/virsh-sandbox-py && pytest)

# Frontend type check
(cd web && bunx tsc --noEmit)
```

## Code Quality Across Projects

```bash
# Go services
(cd virsh-sandbox && make check)
(cd tmux-client && make check)

# Python SDK
(cd sdk/virsh-sandbox-py && mypy virsh_sandbox && black --check . && isort --check .)

# Frontend
(cd web && bun run lint)
```
