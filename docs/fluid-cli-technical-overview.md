# Fluid CLI: Technical Overview

How the fluid CLI builds sandboxes and provides secure, ephemeral access to AI agents.

---

## Table of Contents

1. [Architecture](#architecture)
2. [Sandbox Creation: Linked Clones from Source VMs](#sandbox-creation-linked-clones-from-source-vms)
3. [SSH Certificate Authority](#ssh-certificate-authority)
4. [Command Execution](#command-execution)
5. [Read-Only Source VM Access](#read-only-source-vm-access)
6. [File Operations](#file-operations)
7. [Snapshots and Diffs](#snapshots-and-diffs)
8. [Cleanup and Destruction](#cleanup-and-destruction)
9. [Data Persistence](#data-persistence)

---

## Architecture

The fluid CLI is an embedded tool that AI agents invoke as a subprocess. There is no HTTP server in the loop. The CLI talks directly to libvirt (locally or over SSH) and stores state in a local SQLite database.

```
AI Agent (Claude Code, etc.)
    |
    v  (subprocess / tool call)
fluid CLI
    |
    +-- SQLite store (~/.fluid/state.db)
    +-- SSH Certificate Authority (~/.fluid/ssh-ca/)
    +-- Key Manager (~/.fluid/sandbox-keys/)
    +-- Libvirt Manager
    |
    v
libvirt (qemu:///system or qemu+ssh://host/system)
    |
    v
KVM / QEMU virtual machines
```

All commands output JSON for easy agent parsing. State is persisted in SQLite at `~/.fluid/state.db`, and the database schema is auto-migrated on first run.

---

## Sandbox Creation: Linked Clones from Source VMs

Sandboxes are created by cloning an existing "source" (golden) VM. Fluid uses **QCOW2 linked clones** — copy-on-write overlays that reference the source VM's disk as a backing store. This means no disk is duplicated; sandbox creation is fast and storage-efficient.

### The Clone Process

When `fluid create --source-vm=ubuntu-base` is called, the following sequence executes in `CloneFromVM` (`fluid/internal/libvirt/virsh.go:323`):

**1. Look up the source VM's disk path**

```bash
virsh domblklist <source-vm> --details
```

The output is parsed to find the first `file disk` entry, which gives the path to the source VM's QCOW2 disk image (e.g., `/var/lib/libvirt/images/base/ubuntu-base.qcow2`).

**2. Create a workspace directory**

A per-sandbox working directory is created at `{WorkDir}/{sandbox-name}/`. This directory holds all sandbox-specific artifacts:

```
/var/lib/libvirt/images/sandboxes/sbx-abc123/
├── disk-overlay.qcow2    # Copy-on-write overlay
├── cloud-init.iso         # Unique cloud-init seed
└── domain.xml             # Libvirt domain definition
```

**3. Create the QCOW2 overlay**

```bash
qemu-img create -f qcow2 -F qcow2 -b <base-disk-path> <overlay-path>
```

This creates a thin overlay that starts empty and only grows as the sandbox writes data. All reads that don't hit the overlay fall through to the backing (source) disk.

**4. Generate a unique cloud-init ISO**

Each sandbox gets its own cloud-init NoCloud seed ISO. This is critical — without a unique `instance-id`, cloud-init inside the clone would detect it already ran (from the source VM's state) and skip network initialization, leaving the sandbox with no IP address.

The ISO contains:

- `meta-data`: A unique `instance-id` set to the sandbox name, plus `local-hostname`
- `user-data`: Network configuration enabling DHCP on virtio interfaces

```yaml
# meta-data
instance-id: sbx-abc123
local-hostname: sbx-abc123

# user-data
#cloud-config
network:
  version: 2
  ethernets:
    id0:
      match:
        driver: virtio*
      dhcp4: true
```

The ISO is built using `genisoimage` or `cloud-localds`, whichever is available on the host.

**5. Modify the source VM's XML**

The source VM's libvirt domain XML is dumped via `virsh dumpxml` and then modified (`modifyClonedXML` at `virsh.go:421`):

- **Name**: Set to the new sandbox name
- **UUID**: Removed (libvirt assigns a new one)
- **Disk path**: Updated to point at the overlay, not the source disk
- **MAC address**: Regenerated to ensure a unique network identity and separate DHCP lease
- **Cloud-init CDROM**: Attached (or existing CDROM updated) to point at the new ISO
- **PCI addresses**: Removed from network interfaces to avoid slot conflicts

**6. Define and start the VM**

```bash
virsh define <domain.xml>
virsh start <sandbox-name>
```

After starting, the CLI polls for an IP address via `virsh domifaddr` using DHCP lease data, then verifies SSH connectivity before returning the sandbox as ready.

### Orchestration Layer

The VM service (`fluid/internal/vm/service.go:394`) orchestrates the full create flow:

1. `libvirt.Manager.CloneFromVM()` — creates the overlay, ISO, and domain
2. `store.CreateSandbox()` — persists sandbox metadata to SQLite
3. `libvirt.Manager.StartVM()` — boots the VM
4. `libvirt.Manager.GetIPAddress()` — waits for DHCP lease
5. `vm.Service.waitForSSH()` — verifies SSH is reachable
6. `store.UpdateSandboxState()` — marks sandbox as `RUNNING`

---

## SSH Certificate Authority

Fluid runs its own SSH Certificate Authority (CA) to provide ephemeral, auditable access to sandboxes. No long-lived SSH keys are stored on VMs and no `authorized_keys` files are managed.

### CA Key Pair

On first run (`fluid init`), the CLI generates an Ed25519 CA key pair via `ssh-keygen`:

```
~/.fluid/ssh-ca/
├── ssh-ca          # CA private key (0600 permissions)
└── ssh-ca.pub      # CA public key (baked into VM images)
```

The CA public key is installed into source VM images so that `sshd` can verify certificates signed by this CA. This is done by adding `TrustedUserCAKeys /etc/ssh/fluid_ca.pub` to the VM's `sshd_config`.

### Certificate Issuance

When the CLI needs to execute a command in a sandbox, the `KeyManager` (`fluid/internal/sshkeys/manager.go:149`) handles credential lifecycle:

1. **Check cache**: If valid cached credentials exist for this sandbox (with a 30-second refresh margin), reuse them
2. **Generate ephemeral keypair**: Create a new Ed25519 keypair via `ssh-keygen`
3. **Request certificate**: The CA signs the public key with `ssh-keygen -s`:

```bash
ssh-keygen -s ~/.fluid/ssh-ca/ssh-ca \
    -I "user:<agent-id>-vm:<vm-id>-sbx:<sandbox-id>-cert:<cert-id>" \
    -n sandbox \
    -V +30m \
    -z <serial> \
    -O no-port-forwarding \
    -O no-agent-forwarding \
    -O no-X11-forwarding \
    <user_key.pub>
```

4. **Store credentials**: Private key and certificate are written to disk:

```
~/.fluid/sandbox-keys/
└── SBX-abc123/
    ├── key              # Private key (0600)
    └── key-cert.pub     # Signed certificate
```

### Certificate Properties

Each certificate (`fluid/internal/sshca/ca.go:260`) includes:

| Property | Value |
|----------|-------|
| Key type | Ed25519 |
| Identity | `user:{agent-id}-vm:{vm-id}-sbx:{sandbox-id}-cert:{cert-id}` |
| Principals | `["sandbox"]` (the SSH username allowed) |
| TTL | 30 minutes (default), max 60 minutes |
| Clock skew | 1-minute backdate on `ValidAfter` |
| Serial | Monotonically incrementing, random initial seed |
| Restrictions | No port forwarding, no agent forwarding, no X11 forwarding |
| Allowed | PTY access (for interactive sessions) |

### Security Properties

- **Short-lived**: Certificates expire after 30 minutes by default. Even if a key is leaked, the window of exposure is narrow.
- **Per-sandbox isolation**: Each sandbox gets its own keypair. Compromising one sandbox's credentials doesn't grant access to another.
- **Automatic renewal**: The `KeyManager` transparently regenerates credentials 30 seconds before expiry.
- **Audit trail**: Certificate identity strings embed the agent ID, VM ID, sandbox ID, and a unique cert ID, making every access traceable.
- **Permission enforcement**: CA private key must have `0600` or `0400` permissions; the CLI refuses to start if permissions are too open.

---

## Command Execution

Commands are executed inside sandboxes via SSH. The `RunCommand` method (`fluid/internal/vm/service.go:1124`) handles the full flow:

### Execution Flow

1. **Re-discover IP**: The sandbox's IP is always re-fetched from DHCP leases before execution (not cached), preventing stale routing
2. **Validate IP uniqueness**: Checks that no other running sandbox shares the same IP, guarding against DHCP conflicts
3. **Obtain credentials**: Calls `KeyManager.GetCredentials()` to get a valid cert (cached or freshly generated)
4. **Execute via SSH**:

```bash
ssh -i <private-key> \
    -o CertificateFile=<cert-path> \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=15 \
    -o ServerAliveInterval=30 \
    -o ServerAliveCountMax=1000 \
    sandbox@<ip> \
    -- <command>
```

5. **Persist result**: The command, exit code, stdout, and stderr are saved to SQLite as an audit record

### Retry Logic

SSH commands that fail with exit code 255 (connection error) are retried with exponential backoff: 2s, 4s, 8s, 16s, 30s (capped), up to 5 attempts. Only transient failures (connection refused, timeout, DNS errors) trigger retries — application-level errors (non-zero exit codes from the command itself) are returned immediately.

### Remote Host Support

For sandboxes running on remote KVM hosts, the CLI constructs an SSH ProxyJump chain:

```bash
ssh -i <private-key> \
    -o CertificateFile=<cert-path> \
    -J root@<host-address> \
    sandbox@<sandbox-ip> -- <command>
```

The `RemoteVirshManager` (`fluid/internal/libvirt/remote.go`) tunnels all `virsh` commands through SSH to the remote host's libvirt socket.

---

## Read-Only Source VM Access

The `fluid source prepare` command configures a source (golden) VM for read-only access, allowing agents to inspect the VM's state without modifying it.

### Preparation Steps

The `Prepare` function (`fluid/internal/readonly/prepare.go:33`) executes six idempotent steps over SSH:

1. **Install restricted shell** at `/usr/local/bin/fluid-readonly-shell`
2. **Create `fluid-readonly` user** with the restricted shell as its login shell
3. **Install CA public key** at `/etc/ssh/fluid_ca.pub`
4. **Configure sshd** to trust the CA (`TrustedUserCAKeys`) and use principal-based authorization (`AuthorizedPrincipalsFile`)
5. **Create authorized principals** file mapping the `fluid-readonly` user to the `fluid-readonly` principal
6. **Restart sshd** to apply changes

### The Restricted Shell

The restricted shell (`fluid/internal/readonly/shell.go`) is a bash script installed on the VM that enforces read-only access:

- **No interactive login**: Requires `SSH_ORIGINAL_COMMAND` — direct shell access is denied
- **Command blocklist**: Over 80 patterns are blocked, including:
  - Destructive operations: `rm`, `mv`, `dd`, `mkfs`, `chmod`, `chown`
  - Package management: `apt install`, `yum`, `pip install`
  - Process control: `kill`, `shutdown`, `reboot`, `systemctl start/stop`
  - Network tools: `wget`, `curl`, `scp`, `rsync`
  - Interpreters: `python`, `perl`, `ruby`, `node`, `bash`
  - Editors: `vim`, `nano`, `emacs`, `sed -i`
- **Subshell prevention**: Command substitution (`$(...)`, backticks), process substitution (`<(...)`, `>(...)`), and output redirection are blocked
- **Pipeline validation**: Each segment of piped/chained commands is validated independently — `cat /etc/passwd | rm -rf /` would block on the `rm` segment

Certificates issued for source VM access use the `fluid-readonly` principal instead of `sandbox`, ensuring they can only authenticate as the restricted user.

---

## File Operations

Fluid does not have dedicated file read/write CLI commands. Instead, file operations are performed through `fluid run`, executing standard shell commands over SSH:

```bash
# Read a file
fluid run SBX-abc123 "cat /etc/nginx/nginx.conf"

# Write a file
fluid run SBX-abc123 "cat > /tmp/config.yaml << 'EOF'
server:
  port: 8080
EOF"

# List directory contents
fluid run SBX-abc123 "ls -la /var/log/"

# Check file differences
fluid run SBX-abc123 "diff /etc/nginx/nginx.conf /etc/nginx/nginx.conf.bak"
```

This approach keeps the CLI surface small while giving agents full shell access to the sandbox. Since sandboxes are isolated VMs, arbitrary shell commands carry no risk to the host.

### Pre-boot File Injection

Before a sandbox boots, SSH keys can be injected into the disk image using one of two methods (configured via `ssh_key_inject_method`):

**virt-customize** (offline injection):
```bash
virt-customize -a <overlay-disk> \
    --run-command 'useradd -m -s /bin/bash sandbox' \
    --ssh-inject 'sandbox:string:<public-key>'
```

**cloud-init**: The SSH public key is embedded in the cloud-init ISO's `user-data`, and cloud-init configures the user on first boot.

---

## Snapshots and Diffs

### Creating Snapshots

Fluid supports two snapshot types (`fluid/internal/libvirt/virsh.go:972`):

**Internal snapshots** (managed by QEMU/libvirt):
```bash
virsh snapshot-create-as <vm-name> <snapshot-name>
```
Stored within the VM's QCOW2 file. Faster to create, but the VM must be in a consistent state.

**External snapshots** (separate QCOW2 files):
```bash
virsh snapshot-create-as <vm-name> <snapshot-name> \
    --disk-only --atomic --no-metadata \
    --diskspec vda,file=<work-dir>/<vm>/snap-<name>.qcow2
```
Creates a new QCOW2 overlay at each snapshot point. Can be taken while the VM is running.

### Diffing Snapshots

The `DiffSnapshot` method (`virsh.go:1002`) returns a comparison plan. For external snapshots, it provides instructions for mounting both QCOW2 files via `qemu-nbd` and diffing the filesystem trees. Command history between snapshot points is also available from the audit trail stored in SQLite.

---

## Cleanup and Destruction

When `fluid destroy <sandbox-id>` is called, the VM service orchestrates a multi-layer cleanup (`fluid/internal/vm/service.go:982`):

### Cleanup Sequence

**1. SSH credential cleanup** (`sshkeys/manager.go:198`)

The `KeyManager.CleanupSandbox()` method:
- Removes all cached credentials from memory (all usernames for this sandbox)
- Deletes the sandbox's key directory from disk: `~/.fluid/sandbox-keys/{sandbox-id}/`
- Removes the per-sandbox mutex from the lock map

**2. VM destruction** (`libvirt/virsh.go:922`)

The `DestroyVM` method executes in order:

- **Get MAC address**: `virsh domiflist <vm-name>` — captured before destruction for DHCP cleanup
- **Force stop**: `virsh destroy <vm-name>` — immediately kills the VM process (best-effort, non-fatal if already stopped)
- **Undefine domain**: `virsh undefine --remove-all-storage <vm-name>` — removes the domain definition and attempts to delete associated storage volumes. Falls back to `virsh undefine` without `--remove-all-storage` for older libvirt versions
- **Release DHCP lease**: Removes the VM's MAC address from the dnsmasq lease file (`/var/lib/libvirt/dnsmasq/{network}.leases`), preventing IP address conflicts when future VMs are created
- **Remove workspace**: Deletes the entire working directory `{WorkDir}/{vm-name}/`, which includes the disk overlay, cloud-init ISO, domain XML, and any external snapshot files

**3. Database cleanup** (`store/sqlite/sqlite.go`)

Sandbox records are **soft-deleted** — a `deleted_at` timestamp is set rather than physically removing the row. All queries filter on `WHERE deleted_at IS NULL`, so deleted sandboxes become invisible to the application. Related records (commands, snapshots, diffs) are retained for audit purposes.

### Session-Based Cleanup

When using the interactive TUI (`fluid tui`), the agent tracks every sandbox it creates during the session. On exit (including `Ctrl+C`), a deferred cleanup function iterates through all created sandboxes and destroys them, implementing a "leave no trace" policy.

### What Gets Cleaned Up

| Resource | Cleanup Method | Location |
|----------|---------------|----------|
| VM process | `virsh destroy` | libvirt |
| Domain definition | `virsh undefine` | libvirt |
| Disk overlay | `os.RemoveAll(jobDir)` | `{WorkDir}/{vm-name}/` |
| Cloud-init ISO | `os.RemoveAll(jobDir)` | `{WorkDir}/{vm-name}/` |
| Domain XML | `os.RemoveAll(jobDir)` | `{WorkDir}/{vm-name}/` |
| External snapshots | `os.RemoveAll(jobDir)` | `{WorkDir}/{vm-name}/` |
| DHCP lease | Lease file rewrite | `/var/lib/libvirt/dnsmasq/` |
| SSH private key | `os.RemoveAll(keyDir)` | `~/.fluid/sandbox-keys/{id}/` |
| SSH certificate | `os.RemoveAll(keyDir)` | `~/.fluid/sandbox-keys/{id}/` |
| In-memory credential cache | `delete(m.credentials, key)` | KeyManager |
| Database record | Soft delete (`deleted_at`) | `~/.fluid/state.db` |

---

## Data Persistence

All state is stored in SQLite at `~/.fluid/state.db`. The schema is auto-migrated on startup using GORM.

### Tables

| Table | Purpose |
|-------|---------|
| `sandboxes` | VM sandbox metadata: ID, name, state, IP, host address |
| `snapshots` | Snapshot records: ID, sandbox ID, name, kind (internal/external), ref |
| `commands` | Command audit trail: ID, sandbox ID, command, exit code, stdout, stderr |
| `diffs` | Snapshot comparison results |
| `changesets` | Generated Ansible/Puppet configurations |
| `source_vms` | Golden VM metadata: name, prepared status, CA fingerprint |

The command audit trail in particular is valuable for AI agent workflows — every command executed in every sandbox is persisted with its full output, enabling replay, debugging, and compliance review.
