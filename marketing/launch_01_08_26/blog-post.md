# Blog Post - virsh-sandbox Launch

---

# The Future of Infrastructure: Autonomous AI Agents with Human Approval

## Stop Suggesting. Start Doing.

Today's AI assistants are incredible at writing code. But when it comes to infrastructure, we're stuck in "suggestion mode":

> "You should run `apt install nginx`"  
> "Try adding this to your nginx.conf"  
> "Maybe restart the service with `systemctl restart nginx`"

The human still has to execute every command. Copy-paste. Verify. Repeat.

**Why?** Because we can't give AI agents root access to production. One hallucinated `rm -rf /` and you're restoring from backups.

## What If the Agent Could Just Do It?

Imagine telling an agent:

> "Set up nginx with SSL for api.example.com, configure rate limiting, and set up log rotation"

And it just... does it. Autonomously. Figures out the steps, installs packages, edits configs, tests the setup.

Then you review what it did—a clean diff of every change—and approve. One click and an Ansible playbook applies the exact same changes to production.

**The agent does the work. You just approve.**

That's what Fluid.sh enables.

---

## The Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Autonomous Infrastructure Agent Workflow             │
│                                                                         │
│  ┌─────────┐     ┌─────────────────┐     ┌──────────┐     ┌──────────┐  │
│  │  Task   │────►│  Sandbox VM     │────►│  Human   │────►│Production│  │
│  │         │     │  (autonomous)   │     │ Approval │     │  Server  │  │
│  └─────────┘     └─────────────────┘     └──────────┘     └──────────┘  │
│                         │                      │                        │
│                    • Full root access     • Review diff                 │
│                    • Install packages     • Approve Ansible             │
│                    • Edit configs         • One-click apply             │
│                    • Configure network                                  │
│                    • Snapshot/restore                                   │
└─────────────────────────────────────────────────────────────────────────┘
```

The key insight: **The bottleneck isn't AI capability—it's trust.**

AI agents are already capable of doing infrastructure work. The problem is we can't let them near production. virsh-sandbox solves this by:

1. Giving agents an isolated environment where they can work freely
2. Capturing exactly what they did via snapshots
3. Letting humans review and approve before production

---

## How It Works

### 1. Create a Sandbox

The agent gets a fresh VM, cloned from a golden image. Full root access, isolated network.

```python
from virsh_sandbox import VirshSandbox

client = VirshSandbox("http://localhost:8080")

sandbox = client.sandbox.create_sandbox(
    source_vm_name="ubuntu-24.04-base",
    agent_id="nginx-setup-agent",
    auto_start=True,
    wait_for_ip=True
).sandbox
```

### 2. Agent Works Autonomously

The agent runs commands, installs packages, edits files—whatever it needs to do. No hand-holding required.

```python
# Agent installs and configures nginx
client.sandbox.run_sandbox_command(sandbox.id, "apt update")
client.sandbox.run_sandbox_command(sandbox.id, "apt install -y nginx certbot python3-certbot-nginx")
client.sandbox.run_sandbox_command(sandbox.id, "systemctl enable nginx")

# Agent configures SSL
client.sandbox.run_sandbox_command(sandbox.id, "certbot --nginx -d api.example.com --non-interactive --agree-tos -m admin@example.com")

# Agent sets up rate limiting
client.sandbox.run_sandbox_command(sandbox.id, """
cat > /etc/nginx/conf.d/rate-limit.conf << 'EOF'
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
EOF
""")

# Agent configures log rotation
client.sandbox.run_sandbox_command(sandbox.id, """
cat > /etc/logrotate.d/nginx-custom << 'EOF'
/var/log/nginx/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 www-data adm
    sharedscripts
    postrotate
        [ -f /var/run/nginx.pid ] && kill -USR1 `cat /var/run/nginx.pid`
    endscript
}
EOF
""")
```

### 3. Checkpoint Progress

Snapshots let the agent (or human) roll back if something goes wrong.

```python
client.sandbox.create_snapshot(sandbox.id, name="nginx-configured")
```

### 4. Human Reviews and Approves

The diff shows exactly what changed. The human reviews and decides:

- **Approve**: Auto-generated Ansible playbook applies to production
- **Reject**: Nothing happens, agent can try again
- **Modify**: Human tweaks the playbook before applying

### 5. Apply to Production

One click. The same changes that worked in the sandbox now apply to production via Ansible.

---

## Why VMs Instead of Containers?

| Aspect | Containers | VMs |
|--------|------------|-----|
| Isolation | Namespace (shared kernel) | Hardware-level (separate kernel) |
| Networking | Limited (usually bridged) | Full stack (iptables, routing, etc.) |
| Snapshots | Docker commit (awkward) | Native hypervisor support |
| Reboot | Not really possible | Full reboot support |
| Kernel params | Shared with host | Independent |
| Failure blast radius | Could affect host | Contained to VM |

For infrastructure work where agents might configure firewalls, modify kernel parameters, install kernel modules, or test reboots—you need real VMs.

---

## Use Cases

### Autonomous Server Provisioning

> "Set up a new web server with nginx, PostgreSQL, and Redis. Configure SSL, set up backups, and harden SSH."

Agent does it all in the sandbox. You review and approve.

### Self-Healing Infrastructure

> Alert: "Disk usage at 95% on prod-web-1"

Agent clones prod-web-1 into a sandbox, diagnoses the issue, cleans up old logs, sets up better rotation. You approve the fix.

### Migration Automation

> "Migrate this Django app from Ubuntu 20.04 to 24.04"

Agent figures out the dependency changes, tests the migration in a sandbox, creates the upgrade playbook. You approve.

### Compliance Remediation

> "Audit this server against CIS benchmarks and fix any issues"

Agent runs the audit, remediates issues in the sandbox, documents changes. You review and approve.

### On-Call Automation

> PagerDuty alert: "Service 'api' is not responding"

Agent diagnoses (OOM? Crashed? Config issue?), tests fixes in sandbox, proposes remediation. You approve from your phone.

---

## Security Model

### What the Agent Can Do (in sandbox)
- Full root access
- Install any packages
- Modify any files
- Configure networking
- Reboot the VM
- Anything you could do with root

### What the Agent Cannot Do
- Touch production (blocked by approval workflow)
- Escape the VM (KVM hardware isolation)
- Access other sandboxes (isolated networks)
- Persist beyond sandbox lifetime

### Safety Features
- **VM Isolation**: Hardware-level, not namespace
- **Ephemeral Credentials**: SSH certificates expire in 1-10 minutes
- **Blocking Approval**: Nothing happens until human approves
- **Audit Trail**: Every command logged
- **Snapshot Rollback**: Undo any mistake

---

## Getting Started

### Quick Start (Docker Compose)

```bash
git clone https://github.com/[your-org]/virsh-sandbox.git
cd virsh-sandbox
docker-compose up --build

# API:      http://localhost:8080
# Web UI:   http://localhost:5173
```

### Build Your First Agent

```python
from virsh_sandbox import VirshSandbox
from openai import OpenAI

client = VirshSandbox("http://localhost:8080")
openai = OpenAI()

# Create sandbox
sandbox = client.sandbox.create_sandbox(
    source_vm_name="ubuntu-base",
    agent_id="my-agent",
    auto_start=True
    wait_for_ip=True
).sandbox

# Your agent logic here...
# Use OpenAI function calling to map to sandbox API

# Clean up
client.sandbox.destroy_sandbox(sandbox.id)
```

See `examples/agent-example/` for a complete implementation.

---

## The Vision

Today: AI agents that suggest commands.

Tomorrow: **AI agents that manage infrastructure autonomously, with humans providing oversight at key decision points.**

virsh-sandbox is the infrastructure layer that makes this possible.

---

## Links

- **GitHub**: [github.com/your-org/virsh-sandbox](https://github.com/your-org/virsh-sandbox)
- **Documentation**: [docs link]
- **Demo Video**: [youtube link]
- **Python SDK**: [pypi link]

---

*Built for the future of infrastructure automation.*
