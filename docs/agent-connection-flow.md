# Agent Connection Flows

There are two ways for agents to execute commands in sandboxes:

1. **Tool Call Flow** (Recommended) - Direct API calls to virsh-sandbox
2. **Interactive Session Flow** - tmux-client for interactive terminal access

---

## Option 1: Tool Call Flow (Recommended)

For agents using tool/function calls, virsh-sandbox handles SSH credentials internally.
No key management required by the caller.

```
┌────────────────────────────────────────────────────────────────────────────┐
│                      Tool Call Flow (Simplified)                           │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  1. Agent calls virsh-sandbox API                                          │
│     POST /v1/sandboxes/{id}/run                                            │
│     { "command": "apt update && apt install -y nginx" }                    │
│              │                                                             │
│              ▼                                                             │
│  2. virsh-sandbox checks for cached credentials                            │
│     ┌─────────────────────────────────────────────────────────┐            │
│     │ /tmp/sandbox-keys/{sandbox_id}/                         │            │
│     │   - key (private key, 0600)                             │            │
│     │   - key-cert.pub (certificate)                          │            │
│     └─────────────────────────────────────────────────────────┘            │
│              │                                                             │
│              ├─── Cached & valid? ──> Skip to step 5                       │
│              │                                                             │
│              ▼                                                             │
│  3. virsh-sandbox generates ephemeral key pair (if needed)                 │
│     - ed25519 keypair                                                      │
│     - Stored in /tmp/sandbox-keys/{sandbox_id}/                            │
│              │                                                             │
│              ▼                                                             │
│  4. virsh-sandbox issues certificate internally (5 min TTL)                │
│     - Signs public key with SSH CA                                         │
│     - Saves certificate to key-cert.pub                                    │
│              │                                                             │
│              ▼                                                             │
│  5. virsh-sandbox executes SSH command                                     │
│     ssh -i key -o CertificateFile=key-cert.pub \                           │
│         sandbox@{vm_ip} -- "apt update && apt install -y nginx"            │
│              │                                                             │
│              ▼                                                             │
│  6. Returns command result to agent                                        │
│     {                                                                      │
│       "command": {                                                         │
│         "stdout": "...",                                                   │
│         "stderr": "...",                                                   │
│         "exit_code": 0                                                     │
│       }                                                                    │
│     }                                                                      │
│                                                                            │
│  Cleanup: Keys deleted when sandbox is destroyed                           │
│           DELETE /v1/sandboxes/{id}                                        │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### API Example

```bash
# Create sandbox
curl -X POST http://localhost:8080/v1/sandboxes \
  -H "Content-Type: application/json" \
  -d '{"source_vm_name": "ubuntu-base", "agent_id": "my-agent"}'

# Start sandbox
curl -X POST http://localhost:8080/v1/sandboxes/SBX-abc123/start \
  -H "Content-Type: application/json" \
  -d '{"wait_for_ip": true}'

# Run commands (no credentials needed!)
curl -X POST http://localhost:8080/v1/sandboxes/SBX-abc123/run \
  -H "Content-Type: application/json" \
  -d '{"command": "whoami"}'

# Response:
# {"command": {"stdout": "sandbox\n", "exit_code": 0, ...}}

# Destroy when done (cleans up keys automatically)
curl -X DELETE http://localhost:8080/v1/sandboxes/SBX-abc123
```

### Configuration

Set these environment variables on the virsh-sandbox API server:

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_CA_KEY_PATH` | `/etc/virsh-sandbox/ssh_ca` | SSH CA private key |
| `SSH_CA_PUB_KEY_PATH` | `/etc/virsh-sandbox/ssh_ca.pub` | SSH CA public key |
| `SSH_KEY_DIR` | `/tmp/sandbox-keys` | Ephemeral key storage |
| `SSH_CERT_TTL_SEC` | `300` | Certificate TTL (5 min) |

### VM Setup

VMs must trust the SSH CA:

```bash
# Copy CA public key to VM image
cp /etc/virsh-sandbox/ssh_ca.pub /etc/ssh/ssh_ca.pub

# Add to /etc/ssh/sshd_config
echo "TrustedUserCAKeys /etc/ssh/ssh_ca.pub" >> /etc/ssh/sshd_config

# Ensure 'sandbox' user exists
useradd -m -s /bin/bash sandbox
```

---

## Option 2: Interactive Session Flow (tmux-client)

For interactive terminal access, use the tmux-client which manages its own
SSH sessions and provides a persistent terminal interface.

```
┌────────────────────────────────────────────────────────────────────────────┐
│                    Interactive Session Flow (tmux-client)                  │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  1. Agent calls tmux-client API                                            │
│     POST /v1/sandbox/sessions/create                                       │
│     { "sandbox_id": "SBX-abc123" }                                         │
│              │                                                             │
│              ▼                                                             │
│  2. tmux-client generates ephemeral key pair                               │
│     ┌─────────────────────────────────────────────┐                        │
│     │ /tmp/sandbox-keys/sandbox_SBX-abc123_...    │                        │
│     │   - Private key (ed25519)                   │                        │
│     │   - Public key                              │                        │
│     └─────────────────────────────────────────────┘                        │
│              │                                                             │
│              ▼                                                             │
│  3. tmux-client calls virsh-sandbox API                                    │
│     POST /v1/access/request                                                │
│     { "sandbox_id": "...", "public_key": "ssh-ed25519 ..." }               │
│              │                                                             │
│              ▼                                                             │
│  4. virsh-sandbox issues certificate (5 min TTL)                           │
│     Returns: { "certificate": "ssh-ed25519-cert-v01...",                   │
│                "vm_ip_address": "192.168.122.10" }                         │
│              │                                                             │
│              ▼                                                             │
│  5. tmux-client saves certificate                                          │
│     /tmp/sandbox-keys/sandbox_SBX-abc123_...-cert.pub                      │
│              │                                                             │
│              ▼                                                             │
│  6. tmux-client creates tmux session with SSH command                      │
│     tmux new-session -d -s sandbox_SBX-abc123 \                            │
│       "ssh -i /tmp/.../key -o CertificateFile=/tmp/.../key-cert.pub \      │
│        sandbox@192.168.122.10"                                             │
│              │                                                             │
│              ▼                                                             │
│  7. Agent is now in a tmux session connected to the sandbox VM             │
│     ┌────────────────────────────────────────────┐                         │
│     │ sandbox@vm:~$                              │                         │
│     │ (tmux session - no shell escape)           │                         │
│     └────────────────────────────────────────────┘                         │
│                                                                            │
│  8. When done: DELETE /v1/sandbox/sessions/sandbox_SBX-abc123              │
│     - Kills tmux session                                                   │
│     - Deletes ephemeral keys                                               │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## Comparison

| Feature | Tool Call Flow | Interactive Session Flow |
|---------|---------------|-------------------------|
| Use case | Automated tool/function calls | Interactive terminal |
| Credential management | Automatic (virsh-sandbox) | Automatic (tmux-client) |
| Session type | Stateless per-command | Persistent tmux session |
| Best for | AI agents with tool calling | Human debugging, complex workflows |
| API | virsh-sandbox `/run` endpoint | tmux-client sessions API |

---

## Security Properties

Both flows share these security characteristics:

1. **Short-lived certificates** - 5 minute TTL by default
2. **Ephemeral keys** - Generated per-sandbox, deleted on destroy
3. **Certificate-based auth** - No passwords, no authorized_keys management
4. **Audit trail** - All certificates logged by the SSH CA
5. **Isolation** - Each sandbox has its own keypair
