# Fluid - Development Guide

Fluid is an embedded CLI tool that lets AI agents create and manage VM sandboxes directly - no HTTP server required. Local SQLite for state, direct libvirt access via local socket or SSH.

## Architecture

```
AI Agent (Claude Code, etc.)
    |
    v (subprocess/tool calls)
fluid CLI
    |
    +-- SQLite store (~/.fluid/state.db)
    +-- Libvirt manager
    +-- VM service
    |
    v
libvirt (qemu:///system or qemu+ssh://host/system)
```

## Quick Start

```bash
# Build the CLI
make build

# Initialize configuration (creates ~/.fluid/config.yaml)
./bin/fluid init

# List available VMs to clone from
./bin/fluid vms

# Create a sandbox from a source VM
./bin/fluid create --source-vm=ubuntu-base

# Run commands in the sandbox
./bin/fluid run <sandbox-id> "whoami"

# Destroy when done
./bin/fluid destroy <sandbox-id>
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `fluid init` | Initialize configuration |
| `fluid create` | Create a new sandbox |
| `fluid list` | List sandboxes |
| `fluid get <id>` | Get sandbox details |
| `fluid destroy <id>` | Destroy a sandbox |
| `fluid start <id>` | Start a sandbox |
| `fluid stop <id>` | Stop a sandbox |
| `fluid ip <id>` | Discover IP address |
| `fluid run <id> <cmd>` | Run a command |
| `fluid ssh-inject <id>` | Inject SSH public key |
| `fluid snapshot <id>` | Create a snapshot |
| `fluid diff <id>` | Compare snapshots |
| `fluid vms` | List available VMs |
| `fluid version` | Print version |
| `fluid tui` | Launch interactive TUI |
| `fluid mcp` | Start MCP server on stdio |

All commands output JSON by default for easy agent parsing.

## Interactive TUI

Fluid includes an interactive terminal UI for human operators, built with Bubble Tea, Bubbles, and Lipgloss.

```bash
# Launch the TUI
./bin/fluid tui
```

### TUI Features

- **Real-time feedback**: See tool calls and their results as they happen
- **Conversation view**: Scrollable history with markdown rendering
- **Thinking indicator**: Animated spinner while processing
- **Tool result display**: Success/failure indicators with result summaries

### TUI Commands

The TUI accepts natural commands:

| Command | Description |
|---------|-------------|
| `list` (ls) | List all sandboxes |
| `create <source-vm>` | Create a new sandbox |
| `destroy <id>` | Destroy a sandbox |
| `get <id>` | Get sandbox details |
| `start <id>` | Start a stopped sandbox |
| `stop <id>` | Stop a running sandbox |
| `run <id> <cmd>` | Run a command in a sandbox |
| `snapshot <id> [name]` | Create a snapshot |
| `vms` | List available VMs for cloning |
| `help` | Show help message |

### TUI Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `/settings` | Open settings editor |
| `PgUp/PgDn` | Scroll conversation history |
| `Ctrl+R` | Reset conversation |
| `Ctrl+C` | Quit |

### Settings Editor

Type `/settings` or `settings` to open the configuration editor. The settings screen allows you to edit:

**Host Configuration:**
- Host name and address

**Libvirt Configuration:**
- Libvirt URI (e.g., `qemu:///system` or `qemu+ssh://user@host/system`)
- Network name
- Base image directory
- Work directory
- SSH key injection method

**VM Defaults:**
- Default vCPUs
- Default memory (MB)
- Command timeout
- IP discovery timeout

**SSH Configuration:**
- Default SSH user
- SSH proxy jump (for isolated networks)

Settings editor shortcuts:
| Key | Action |
|-----|--------|
| `Tab/Down` | Next field |
| `Shift+Tab/Up` | Previous field |
| `Ctrl+S` | Save and exit |
| `Esc` | Cancel and exit |

### Example TUI Session

```
> list
  v list_sandboxes
    -> {"count":1,"sandboxes":[{"id":"SBX-abc123",...}]}

Found 1 sandbox(es):
- sbx-test (SBX-abc123)
  State: RUNNING | IP: 192.168.122.45

> run SBX-abc123 whoami
  v run_command
    -> {"exit_code":0,"stdout":"root\n",...}

Command completed (exit code: 0)
**stdout:**
root
```

## MCP Server

Fluid exposes all sandbox management tools via the [Model Context Protocol](https://modelcontextprotocol.io/) for use with Claude Code, Cursor, and other MCP clients.

### Starting the MCP Server

```bash
./bin/fluid mcp
```

This starts a JSON-RPC server on stdio. All logging goes to `~/.fluid/fluid-mcp.log` since stdout is the MCP transport.

### Client Configuration

**Claude Code** (`~/.claude.json`):

```json
{
  "mcpServers": {
    "fluid": {
      "command": "/path/to/fluid",
      "args": ["mcp"]
    }
  }
}
```

**Cursor** (`.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "fluid": {
      "command": "/path/to/fluid",
      "args": ["mcp"]
    }
  }
}
```

### Available Tools

| Tool | Parameters | Description |
|------|-----------|-------------|
| `list_sandboxes` | (none) | List all sandboxes with state and IPs |
| `create_sandbox` | `source_vm` (required), `host`, `cpu`, `memory_mb` | Create a sandbox by cloning a source VM |
| `destroy_sandbox` | `sandbox_id` (required) | Destroy a sandbox and remove storage |
| `run_command` | `sandbox_id` (required), `command` (required) | Execute a shell command via SSH |
| `start_sandbox` | `sandbox_id` (required) | Start a stopped sandbox |
| `stop_sandbox` | `sandbox_id` (required) | Stop a running sandbox |
| `get_sandbox` | `sandbox_id` (required) | Get detailed sandbox info |
| `list_vms` | (none) | List available VMs for cloning |
| `create_snapshot` | `sandbox_id` (required), `name` | Snapshot current sandbox state |
| `create_playbook` | `name` (required), `hosts`, `become` | Create an Ansible playbook |
| `add_playbook_task` | `playbook_id` (required), `name` (required), `module` (required), `params` | Add a task to a playbook |
| `edit_file` | `sandbox_id` (required), `path` (required), `new_str` (required), `old_str` | Edit or create a file in a sandbox |
| `read_file` | `sandbox_id` (required), `path` (required) | Read a file from a sandbox |
| `list_playbooks` | (none) | List all created playbooks |
| `get_playbook` | `playbook_id` (required) | Get playbook definition and YAML |
| `run_source_command` | `source_vm` (required), `command` (required) | Run read-only command on a source VM |
| `read_source_file` | `source_vm` (required), `path` (required) | Read a file from a source VM |

### Differences from TUI Agent

- **No approval flows**: MCP clients handle their own approval. Sandbox creation and command execution proceed without interactive confirmation.
- **No streaming**: Commands return complete results. Use `run_command` not the streaming variant.
- **No source VM auto-preparation**: If a source VM isn't prepared for read-only access, the error propagates. Run `fluid source prepare <vm>` separately.
- **Agent ID**: MCP-created sandboxes use `agent_id: "mcp-agent"` to distinguish from TUI-created ones.

## Configuration

Default config location: `~/.fluid/config.yaml`

```yaml
libvirt:
  uri: qemu:///system  # or qemu+ssh://user@host/system
  network: default
  base_image_dir: /var/lib/libvirt/images/base
  work_dir: /var/lib/libvirt/images/sandboxes
  ssh_key_inject_method: virt-customize

vm:
  default_vcpus: 2
  default_memory_mb: 2048
  command_timeout: 5m
  ip_discovery_timeout: 2m

ssh:
  proxy_jump: ""  # Optional: user@jumphost for isolated networks
  default_user: sandbox
```

## Development

### Prerequisites

- Go 1.22+
- libvirt/KVM installed and running
- virsh command available

### Build

```bash
# Build the fluid CLI
make build
# Output: bin/fluid

# Clean build artifacts
make clean
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
# Generates: coverage.out, coverage.html
```

### Code Quality

```bash
# Format code
make fmt

# Run go vet
make vet

# Run all checks
make check
```

### Dependencies

```bash
# Download dependencies
make deps

# Tidy and verify dependencies
make tidy
```

## Makefile Targets

Run `make help` to see all available targets:

| Target | Description |
|--------|-------------|
| `all` | Run fmt, vet, test, and build (default) |
| `build` | Build the fluid CLI binary |
| `run` | Build and run the CLI |
| `clean` | Clean build artifacts |
| `fmt` | Format code |
| `vet` | Run go vet |
| `test` | Run tests |
| `test-coverage` | Run tests with coverage |
| `check` | Run all code quality checks |
| `deps` | Download dependencies |
| `tidy` | Tidy and verify dependencies |
| `install` | Install fluid to GOPATH/bin |
| `help` | Show help message |

## Example Agent Usage

```bash
# Agent creates sandbox
$ fluid create --source-vm=ubuntu-base
{"sandbox_id": "SBX-abc123", "name": "sbx-xyz", "state": "RUNNING", "ip": "192.168.122.45"}

# Agent runs commands
$ fluid run SBX-abc123 "apt update && apt install -y nginx"
{"sandbox_id": "SBX-abc123", "exit_code": 0, "stdout": "...", "stderr": ""}

# Agent takes snapshot
$ fluid snapshot SBX-abc123 --name=after-nginx
{"snapshot_id": "SNP-xyz", "sandbox_id": "SBX-abc123", "name": "after-nginx"}

# Agent checks diff
$ fluid diff SBX-abc123 --from=initial --to=after-nginx
{"diff_id": "DIF-xyz", "files_added": ["/etc/nginx/..."], "files_modified": [...]}

# Agent destroys sandbox
$ fluid destroy SBX-abc123
{"destroyed": true, "sandbox_id": "SBX-abc123"}
```

## Data Storage

State is stored in SQLite at `~/.fluid/state.db`:
- Sandboxes
- Snapshots
- Commands
- Diffs

The database is auto-migrated on first run.

If you remove a parameter from a function, don't just pass in nil/null/empty string in a different layer, make sure to remove the extra parameter from every place.
