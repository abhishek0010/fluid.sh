---
title: "How Fluid Reads Source VMs Without Breaking Anything"
pubDate: 2026-02-11
description: "A technical deep-dive into how Fluid's CLI lets AI agents inspect golden VM images with defense-in-depth read-only enforcement -- without cloning, without waiting, and without risk."
author: "Collin @ Fluid.sh"
authorImage: "../images/skeleton_smoking_cigarette.jpg"
authorEmail: "cpfeifer@madcactus.org"
authorPhone: "+3179955114"
authorDiscord: "https://discordapp.com/users/301068417685913600"
---

## The Problem

Fluid already lets AI agents spin up sandboxes by cloning golden VM images via QCOW2 copy-on-write overlays. That flow is fast and safe -- the base image is never modified because every write lands on the overlay.

But sometimes an agent doesn't need a whole clone. It needs to *look* at the source VM: check what packages are installed, inspect a config file, read logs, verify a service is running. Spinning up a full sandbox just to run `dpkg -l` is wasteful. It burns disk, consumes a DHCP lease, and adds latency that kills the feedback loop agents need to stay productive.

The question is simple: how do you let an untrusted AI agent SSH into a production golden image and run commands -- without letting it `rm -rf /` the thing?

## The Design: Three Independent Walls

The answer is defense-in-depth. No single mechanism is trusted to enforce read-only access. Instead, three independent layers each enforce the constraint, so a bypass in any one layer is still contained by the other two.

```
Agent sends command: "cat /etc/nginx/nginx.conf"
         |
         v
  Layer 1: Client-side allowlist (Go, in fluid CLI)
         |  Parses pipeline. Checks each segment against ~70 allowed commands.
         |  Blocks shell metacharacters: $(), backticks, >>, <(), newlines.
         v
  Layer 2: SSH principal separation
         |  Certificate issued with principal "fluid-readonly"
         |  VM's sshd maps this principal to the fluid-readonly user
         |  That user's login shell is the restricted shell
         v
  Layer 3: Server-side restricted shell (bash, on the VM)
         |  Parses pipeline segments again independently.
         |  Blocks ~90 destructive command patterns.
         |  Blocks metacharacters again.
         v
  Command executes (or exits 126 if blocked)
```

If an attacker somehow gets past the client-side validation, the SSH certificate still routes them to a user whose login shell is a restricted script. If they somehow forge a certificate with the wrong principal, the client-side validation already rejected the dangerous command. If both fail, the restricted shell on the VM still blocks it.

## Layer 1: Client-Side Command Validation

Before any SSH connection is established, the fluid CLI validates the command in Go. The validator lives in `fluid/internal/readonly/validate.go` and follows a strict allowlist approach -- not a blocklist.

### The Allowlist

About 70 commands are permitted, organized by category:

| Category | Commands |
|----------|----------|
| File inspection | `cat`, `ls`, `find`, `head`, `tail`, `stat`, `file`, `wc`, `du`, `tree`, `strings`, `md5sum`, `sha256sum` |
| Process/system | `ps`, `top`, `pgrep`, `systemctl` (status only), `journalctl`, `dmesg` |
| Network | `ss`, `netstat`, `ip`, `ifconfig`, `dig`, `nslookup`, `ping` |
| Disk | `df`, `lsblk`, `blkid` |
| Package query | `dpkg -l`, `rpm -qa`, `apt list`, `pip list` |
| System info | `uname`, `hostname`, `uptime`, `free`, `lscpu`, `lsmod`, `lspci`, `lsusb` |
| Pipe targets | `grep`, `awk`, `sed`, `sort`, `uniq`, `cut`, `tr`, `xargs` |

Any command not on the list is rejected before a network connection is even attempted.

### Subcommand Restrictions

Some commands are only partially safe. `systemctl status nginx` is fine; `systemctl restart nginx` is not. The validator enforces this with a subcommand restriction map:

```go
var subcommandRestrictions = map[string]map[string]bool{
    "systemctl": {
        "status": true, "show": true, "list-units": true,
        "is-active": true, "is-enabled": true,
    },
    "dpkg": {"-l": true, "--list": true},
    "rpm":  {"-qa": true, "-q": true},
    "apt":  {"list": true},
    "pip":  {"list": true},
}
```

### Metacharacter Blocking

An allowlist alone isn't enough. `cat /etc/passwd` is safe, but `cat /etc/passwd; rm -rf /` is not -- and `cat` is on the allowlist. The validator handles this by:

1. **Splitting pipelines**: The command is parsed on `|`, `;`, `&&`, and `||` boundaries, respecting quote context. Each segment is validated independently.
2. **Blocking injection primitives**: Command substitution (`$(...)`, backticks), process substitution (`<(...)`, `>(...)`), output redirection (`>`, `>>`), and newline characters are all rejected when found outside of quotes.

The quote-aware parser tracks single and double quote state character by character, ensuring that metacharacters inside quoted strings (like `grep ">"`) are not falsely flagged.

### Where It Runs

Validation happens at the top of `RunSourceVMCommand` in `fluid/internal/vm/service.go`, before IP discovery or credential generation:

```go
if err := readonly.ValidateCommand(command); err != nil {
    s.telemetry.Track("source_vm_command_blocked", map[string]any{
        "source_vm": sourceVMName,
        "reason":    err.Error(),
    })
    return nil, fmt.Errorf("command not allowed in read-only mode: %w", err)
}
```

Blocked commands are tracked via telemetry, so operators can see if an agent is repeatedly trying to run disallowed commands.

## Layer 2: SSH Certificate Principal Separation

Even if the client-side validation were completely bypassed -- say, by an attacker calling the SSH binary directly -- the server-side authentication model constrains what's possible.

### Two Principal Types

Fluid's SSH certificate authority issues certificates with different *principals* depending on the access type:

| Access Type | Principal | Username | Shell |
|-------------|-----------|----------|-------|
| Sandbox (full access) | `sandbox` | `sandbox` | `/bin/bash` |
| Source VM (read-only) | `fluid-readonly` | `fluid-readonly` | `/usr/local/bin/fluid-readonly-shell` |

When the fluid CLI requests credentials for a source VM, the key manager issues a certificate with the `fluid-readonly` principal:

```go
certReq := sshca.CertificateRequest{
    UserID:     fmt.Sprintf("source-readonly:%s", sourceVMName),
    Principals: []string{"fluid-readonly"},
    TTL:        m.cfg.CertificateTTL,  // 30 minutes default
}
```

### How the VM Enforces It

During `fluid source prepare`, the source VM's `sshd_config` is configured with:

```
TrustedUserCAKeys /etc/ssh/fluid_ca.pub
AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u
```

The file `/etc/ssh/authorized_principals/fluid-readonly` contains exactly one line: `fluid-readonly`. This means sshd will only accept certificates with the `fluid-readonly` principal when authenticating as the `fluid-readonly` user. A certificate with the `sandbox` principal cannot log in as `fluid-readonly`, and vice versa.

The `fluid-readonly` user is created as a system user with `/usr/local/bin/fluid-readonly-shell` as its login shell. There is no password. There is no home directory. The only way in is via a valid certificate from the CA, and the only thing that runs is the restricted shell.

### Short-Lived Credentials

Certificates have a 30-minute TTL by default, with a 30-second refresh margin. They're cached in memory and regenerated on demand:

```go
if ok && !creds.IsExpired(m.cfg.RefreshMargin) {
    return creds, nil  // Use cached credentials
}
// Otherwise, generate fresh certificate
```

Per-VM key material is stored under `{keyDir}/sourcevm-{sanitizedName}/`, where `sanitizedName` strips everything except `A-Za-z0-9_-` to prevent path traversal:

```go
var vmNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_-]`)
```

A VM name like `../../etc` becomes `______etc`, making directory escape impossible.

## Layer 3: Server-Side Restricted Shell

The final layer runs on the VM itself. Even if a command passes client-side validation and arrives via a valid certificate, the restricted shell at `/usr/local/bin/fluid-readonly-shell` performs its own independent validation.

### No Interactive Login

The shell immediately exits if `SSH_ORIGINAL_COMMAND` is not set, blocking interactive sessions:

```bash
if [ -z "${SSH_ORIGINAL_COMMAND:-}" ]; then
    echo "ERROR: Interactive login is not permitted." >&2
    exit 1
fi
```

### Destructive Command Blocklist

Unlike the client-side allowlist, the server-side shell uses a *blocklist* -- a complementary strategy. It matches approximately 90 patterns against each pipeline segment:

- **File operations**: `rm`, `mv`, `cp`, `dd`
- **Privilege escalation**: `sudo`, `su`
- **Process control**: `kill`, `killall`, `pkill`, `shutdown`, `reboot`
- **User management**: `useradd`, `userdel`, `usermod`, `passwd`
- **Disk operations**: `mkfs`, `mount`, `umount`, `fdisk`, `parted`
- **Package installation**: `apt install`, `dpkg -i`, `rpm -i`, `pip install`
- **Interpreters/shells**: `bash`, `python`, `perl`, `ruby`, `node`, `sh`
- **Editors**: `vi`, `vim`, `nano`, `emacs`
- **In-place editing**: `sed -i`, `tee`
- **Network tools**: `wget`, `curl`, `scp`, `rsync`
- **Firewall**: `iptables`, `nft`
- **Systemctl mutations**: `systemctl start/stop/restart/enable/disable/mask`

### Independent Pipeline Parsing

The restricted shell parses the command on `|`, `;`, `&&`, and `||` boundaries with its own quote-aware parser written in bash. Each segment is validated against the blocklist independently. This means `ls; rm -rf /` is caught even though `ls` is safe -- the `rm -rf /` segment triggers a block.

### Exit Code Convention

When the restricted shell blocks a command, it exits with code **126** -- the conventional Unix code for "command not executable." The fluid CLI recognizes this exit code and logs it as a server-side block, distinct from a client-side rejection.

## How It Comes Together: The Full Execution Path

Here's the complete path when an AI agent runs a read-only command on a source VM:

```
1. Agent calls fluid CLI:
   $ fluid run-source my-golden-vm "systemctl status nginx"

2. Client-side validation (validate.go):
   - Split "systemctl status nginx" into segments: ["systemctl status nginx"]
   - Check metacharacters: none found
   - Check base command: "systemctl" is in allowlist
   - Check subcommand: "status" is in systemctl's allowed subcommands
   ✓ Passes

3. IP discovery:
   - Query libvirt for my-golden-vm's IP via DHCP lease or agent

4. Credential generation (sshkeys/manager.go):
   - Check cache for "sourcevm:my-golden-vm:fluid-readonly"
   - If expired/missing: generate Ed25519 keypair, sign certificate
     with principal "fluid-readonly" and 30-min TTL
   - Write to ~/.fluid/keys/sourcevm-my-golden-vm/key{,-cert.pub}

5. SSH connection:
   - Connect to VM IP as user "fluid-readonly"
   - Authenticate with short-lived certificate
   - sshd verifies cert against /etc/ssh/fluid_ca.pub
   - sshd verifies principal matches /etc/ssh/authorized_principals/fluid-readonly
   - sshd invokes login shell: /usr/local/bin/fluid-readonly-shell

6. Server-side restricted shell:
   - Receives SSH_ORIGINAL_COMMAND="systemctl status nginx"
   - Check metacharacters: none found
   - Parse segments: ["systemctl status nginx"]
   - Check against blocklist: "systemctl status" not matched
     (only "systemctl start", "systemctl stop", etc. are blocked)
   ✓ Passes
   - Executes: exec /bin/bash -c "systemctl status nginx"

7. Return result to agent (not persisted to database)
```

## Why Source VM Preparation is One-Time

Running `fluid source prepare <vm-name>` configures the VM once. The preparation is idempotent -- running it again safely updates the restricted shell and CA key without breaking anything. Each step checks for existing state:

```go
// Create user only if it doesn't already exist
userCmd := `id fluid-readonly >/dev/null 2>&1 || useradd -r -s /usr/local/bin/fluid-readonly-shell -d /nonexistent -M fluid-readonly`

// Add sshd config only if not already present
`grep -q 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' /etc/ssh/sshd_config || echo 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' >> /etc/ssh/sshd_config`
```

The preparation state is tracked in SQLite so fluid knows which VMs are ready:

```go
type SourceVM struct {
    Name          string
    Prepared      bool
    PreparedAt    *time.Time
    CAFingerprint *string  // Detects CA key rotation
}
```

If the CA key changes, the fingerprint mismatch tells fluid the VM needs re-preparation.

## Productivity: No Clone Overhead

The key productivity win is avoiding the full clone cycle. Here's what running a diagnostic command on a source VM *doesn't* require:

- No QCOW2 overlay creation
- No XML domain definition
- No cloud-init ISO generation
- No MAC address generation
- No DHCP lease negotiation
- No boot wait
- No sandbox database record
- No cleanup/destroy afterward

The agent gets a response to `cat /etc/nginx/nginx.conf` or `dpkg -l | grep python` directly from the running golden image. The results aren't persisted to the store (unlike sandbox commands), keeping the audit trail clean -- source VM reads are tracked through telemetry instead.

This matters for the agent workflow. When an agent is deciding *which* source VM to clone, or *whether* a source VM has the right software stack, it can inspect first and clone only when it's ready to make changes. The read-inspect-decide loop stays fast.

## What Could Go Wrong (and Why It's Contained)

| Scenario | Mitigation |
|----------|------------|
| Agent sends `rm -rf /` | Client-side allowlist rejects `rm` before SSH connection |
| Agent sends `cat /etc/passwd; rm -rf /` | Pipeline parser splits on `;`, validates each segment, rejects `rm` |
| Agent sends `$(rm -rf /)` | Metacharacter detector blocks `$()` outside quotes |
| Attacker forges SSH certificate | Restricted shell still blocks destructive commands server-side |
| Attacker bypasses restricted shell | The `fluid-readonly` user has no sudo access, no home directory, no writable paths |
| VM name contains `../../etc` | `sanitizeVMName` strips all non-alphanumeric characters, prevents path traversal |
| Agent tries interactive SSH session | Restricted shell exits immediately when `SSH_ORIGINAL_COMMAND` is empty |
| Credential stolen | 30-minute TTL limits window; certificate only grants `fluid-readonly` principal |

## Summary

Reading a source VM safely requires solving a specific problem: letting untrusted code run *some* commands on a machine that must not be modified. Fluid's approach is to make read-only the default at every layer -- not just one check at the front door, but independent enforcement at the client, the authentication system, and the server. The result is an agent that can inspect golden images at full speed without the overhead of cloning, and without the risk of corruption.
