# Security Model

fluid.sh uses defense-in-depth to isolate AI agent workloads in VM sandboxes. This document describes the security architecture, covering SSH certificate management, read-only source VM enforcement, VM isolation, and credential lifecycle.

## Overview

Four layers enforce isolation:

1. **SSH Certificate Authority** - short-lived certificates replace persistent credentials
2. **Principal separation** - sandbox (`sandbox`) and read-only (`fluid-readonly`) access use distinct SSH principals
3. **Read-only enforcement** - client-side allowlist + server-side restricted shell block destructive commands on source VMs
4. **VM isolation** - KVM/libvirt hypervisor isolation with copy-on-write overlays

## SSH Certificate Authority

The SSH CA signs short-lived certificates for all sandbox and source VM access. No persistent SSH keys are stored on VMs.

**Key generation**: Ed25519 CA key pair generated via `ssh-keygen`. Private key stored at configurable path (default `/etc/virsh-sandbox/ssh_ca`) with 0600 permissions. Public key at the same path with `.pub` suffix.

**Certificate identity format**:
```
user:{UserID}-vm:{VMID}-sbx:{SandboxID}-cert:{CertID}
```

**Certificate properties**:
- Default TTL: 30 minutes
- Maximum TTL: 60 minutes
- Minimum TTL: 1 minute
- Clock skew buffer: 1 minute (validity starts 1 minute before issuance)
- Serial numbers: random 64-bit, incremented per issuance
- Extensions: `permit-pty` only
- Restrictions: `no-port-forwarding`, `no-agent-forwarding`, `no-X11-forwarding`

**Permission validation**: the CA enforces that private key files have mode 0600 or 0400 (no group/world access) before signing.

Source: `fluid/internal/sshca/ca.go`

## Sandbox Credentials

Each sandbox gets ephemeral Ed25519 key pairs, generated on demand and cached until expiry.

- **Principal**: `"sandbox"`
- **Key directory**: `{keyDir}/{sandboxID}/` with 0700 permissions
- **Private keys**: 0600 permissions
- **Certificates**: 0644 permissions
- **Auto-refresh**: credentials regenerate 30 seconds before certificate expiry
- **Thread safety**: per-sandbox mutexes prevent concurrent key generation
- **Cleanup**: key files and cache entries removed on sandbox destroy

Pre-flight permission checks run before every SSH connection: the runner verifies the private key file has no group/world permissions (`perm & 0077 == 0`) and rejects the connection otherwise.

Source: `fluid/internal/sshkeys/manager.go`

## Source VM Read-Only Mode

Source (golden) VMs are accessible only for inspection, never modification. Three enforcement layers ensure this.

### Layer 1: Client-side allowlist

`ValidateCommand()` parses the command into pipeline segments and checks each segment's base command against an allowlist of ~70 safe commands.

**Allowed categories**:
- File inspection: `cat`, `ls`, `find`, `head`, `tail`, `stat`, `file`, `wc`, `du`, `tree`, `strings`, `md5sum`, `sha256sum`, `readlink`, `realpath`, `basename`, `dirname`, `base64`
- Process/system info: `ps`, `top`, `pgrep`, `systemctl`, `journalctl`, `dmesg`
- Network info: `ss`, `netstat`, `ip`, `ifconfig`, `dig`, `nslookup`, `ping`
- Disk info: `df`, `lsblk`, `blkid`
- Package queries: `dpkg`, `rpm`, `apt`, `pip` (restricted subcommands only)
- System info: `uname`, `hostname`, `uptime`, `free`, `lscpu`, `lsmod`, `lspci`, `lsusb`, `arch`, `nproc`
- User info: `whoami`, `id`, `groups`, `who`, `w`, `last`
- Misc: `env`, `printenv`, `date`, `which`, `type`, `echo`, `test`
- Pipe targets: `grep`, `awk`, `sed`, `sort`, `uniq`, `cut`, `tr`, `xargs`

**Subcommand restrictions** (first argument must match allowlist):
- `systemctl`: `status`, `show`, `list-units`, `is-active`, `is-enabled`
- `dpkg`: `-l`, `--list`
- `rpm`: `-qa`, `-q`
- `apt`: `list`
- `pip`: `list`

**Metacharacter blocking**:
- Command substitution: `$(...)` and backticks
- Process substitution: `<(...)` and `>(...)`
- Output redirection: `>` and `>>`
- Newlines: `\n` and `\r`

Source: `fluid/internal/readonly/validate.go`

### Layer 2: Server-side restricted shell

A bash script installed at `/usr/local/bin/fluid-readonly-shell` on source VMs acts as the login shell for the `fluid-readonly` user. It:

1. Denies interactive login (requires `SSH_ORIGINAL_COMMAND`)
2. Blocks command substitution, subshells, output redirection, and newlines
3. Parses the command on pipe/semicolon/`&&`/`||` boundaries
4. Checks each segment against a blocklist of destructive command patterns

**Blocked command categories** (regex patterns on each pipeline segment):
- Privilege escalation: `sudo`, `su`
- File mutation: `rm`, `mv`, `cp`, `dd`, `chmod`, `chown`, `chgrp`
- Process control: `kill`, `killall`, `pkill`, `shutdown`, `reboot`, `halt`, `poweroff`
- User management: `useradd`, `userdel`, `usermod`, `groupadd`, `groupdel`, `passwd`
- Disk operations: `mkfs`, `mount`, `umount`, `fdisk`, `parted`
- Network tools: `wget`, `curl`, `scp`, `rsync`, `ftp`, `sftp`
- Interpreters/shells: `python`, `perl`, `ruby`, `node`, `bash`, `sh`, `zsh`, `dash`, `csh`
- Editors: `vi`, `vim`, `nano`, `emacs`
- Build tools: `make`, `gcc`, `g++`, `cc`
- Package installation: `apt install/remove/purge`, `apt-get`, `dpkg -i/--install/--remove/--purge`, `rpm -i/--install/-e/--erase`, `yum`, `dnf`, `pip install/uninstall`
- Service mutation: `systemctl start/stop/restart/reload/enable/disable/daemon/mask/unmask/edit/set`
- Firewall: `iptables`, `ip6tables`, `nft`
- Write tools: `sed -i`, `tee`, `install`

Source: `fluid/internal/readonly/shell.go`

### Layer 3: SSH principal separation

Source VM credentials use the `"fluid-readonly"` principal. The `sshd` on source VMs is configured with:
- `TrustedUserCAKeys /etc/ssh/fluid_ca.pub`
- `AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u`

Only certificates with the `fluid-readonly` principal are accepted for the `fluid-readonly` user. Sandbox certificates (principal `"sandbox"`) cannot authenticate to source VMs.

Source VM preparation (`fluid source prepare`) is idempotent and performs:
1. Install restricted shell at `/usr/local/bin/fluid-readonly-shell`
2. Create `fluid-readonly` system user with the restricted shell as login shell
3. Copy CA public key to `/etc/ssh/fluid_ca.pub`
4. Configure `sshd` to trust the CA key and use per-user authorized principals
5. Create `/etc/ssh/authorized_principals/fluid-readonly` containing `fluid-readonly`
6. Restart `sshd`

Source: `fluid/internal/readonly/prepare.go`, `fluid/internal/sshkeys/manager.go`

## VM Isolation

- **Hypervisor**: libvirt/KVM provides hardware-level isolation between VMs
- **Copy-on-write overlays**: sandboxes are linked clones from golden images via qcow2 overlay files, so the source disk is never modified
- **Random MAC addresses**: each clone gets a random MAC in the `52:54:00` QEMU prefix via `generateMACAddress()`
- **Cloud-init re-initialization**: each clone gets a unique `instance-id` in a fresh cloud-init ISO, forcing cloud-init to re-run and acquire a new DHCP lease
- **Network isolation**: VMs connect to configurable libvirt networks; optional SSH `ProxyJump` for isolated networks not directly reachable from the host

Source: `fluid/internal/libvirt/virsh.go`

## Command Execution Security

- **Shell escaping**: environment variable values are single-quote escaped via `shellQuote()` (replaces `'` with `'\''`)
- **Environment variable name sanitization**: `safeShellIdent()` strips all characters except `[A-Za-z0-9_]`, replacing them with underscores
- **SSH retry with backoff**: transient connection failures retry up to 5 times with exponential backoff (2s initial, 30s max delay)
- **IP conflict detection**: before every command execution, the service re-discovers the VM IP and validates it is not assigned to another running or starting sandbox
- **StrictHostKeyChecking disabled**: ephemeral VMs have no stable host keys; trust is established via the CA certificate chain instead

Source: `fluid/internal/vm/service.go`

## Path Traversal Prevention

VM names used in filesystem paths are sanitized via `sanitizeVMName()`:

```
regex: [^A-Za-z0-9_-]  ->  replaced with underscore
```

This prevents `../` sequences and absolute path injection in source VM names when constructing key directories.

Source: `fluid/internal/sshkeys/manager.go`

## File Permissions Summary

| Asset | Permission | Notes |
|-------|-----------|-------|
| CA private key | 0600 | Enforced at initialization; 0400 also accepted |
| CA public key | 0644 | Readable by sshd on VMs |
| Key directories | 0700 | Per-sandbox and per-source-VM |
| Private keys | 0600 | Validated before every SSH connection |
| Certificates | 0644 | Standard SSH certificate permissions |
| CA work directory | 0700 | Temp directory for certificate operations |
| Restricted shell | 0755 | Executable on source VMs |

## Timeouts

| Operation | Default | Notes |
|-----------|---------|-------|
| Command execution | 10 minutes | Configurable per-call |
| IP discovery | 2 minutes | Polls libvirt leases or ARP table |
| SSH readiness | 60 seconds | Exponential backoff probes after IP discovery |
| SSH connect | 15 seconds | Per-connection `ConnectTimeout` |
| Certificate TTL | 30 minutes | Max 60 minutes, min 1 minute |
| Credential refresh | 30 seconds before expiry | Auto-regenerates keys and certificates |
