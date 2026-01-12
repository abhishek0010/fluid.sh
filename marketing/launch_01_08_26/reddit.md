# Reddit Posts - virsh-sandbox Launch

**Best posting time:** Weekday mornings, stagger posts 1-2 hours apart

---

## r/devops

**Title:** `I built a platform for autonomous AI infrastructure agents (with human approval before production) [Open Source]`

**Body:**
```
The vision: AI agents that actually DO infrastructure work. Not "here's what you should run." Actually execute.

The problem: You can't give an AI agent root on production. So we limit agents to suggestions while humans copy-paste commands.

The solution: Give the agent root on an isolated VM. Let it work autonomously—provision servers, configure firewalls, deploy services. Then review the diff and approve before production.

How it works:
- Agent gets a fresh KVM sandbox with full root access
- Agent works autonomously (apt install, systemctl, iptables, etc.)
- Snapshots capture checkpoints
- Diff shows exactly what changed
- Human reviews and approves
- Ansible playbook auto-applies to production

**The agent does the work. You just approve.**

Tech: Go API, Python SDK, React UI, libvirt/KVM

Use cases I'm thinking about:
- "Provision a web server with nginx, SSL, and rate limiting"
- "Diagnose why the database is slow and fix it"
- "Migrate this service from Ubuntu 20.04 to 24.04"
- "Set up monitoring with Prometheus and Grafana"

The agent figures out the steps, executes them in the sandbox, and you review before production.

GitHub: [link]

Would love feedback from folks doing AI + infrastructure:
1. Does this match how you'd want autonomous agents to work?
2. What use cases are most interesting?
3. What's missing?
```

**Flair:** Tools / Open Source

---

## r/selfhosted

**Title:** `virsh-sandbox: Let AI agents configure your servers (safely) - open source`

**Body:**
```
For self-hosters who want AI to help manage infrastructure without risking their setup.

The idea: Give an AI agent root access to an isolated VM. Let it do the work—install packages, configure services, set up networking. Then review what it did and approve before applying to your real servers.

Why I built this:
- AI assistants are great at suggesting commands
- But I still have to execute everything manually
- And I can't let them touch production directly

The workflow:
1. Spin up a sandbox VM (cloned from your base image)
2. AI agent works autonomously inside
3. Create snapshots as checkpoints
4. Review the diff of what changed
5. Approve → auto-generated Ansible runs on your real server

Features:
- KVM/libvirt isolation (full VMs, not containers)
- Snapshot and restore
- Web UI for monitoring and approval
- Python SDK for building agents
- Works with GPT-4, Claude, local LLMs

Self-hostable, runs on any Linux box with KVM. Also works on Mac with Lima.

GitHub: [link]

Anyone else using AI for homelab/self-hosted infrastructure management?
```

**Flair:** Automation / AI

---

## r/LocalLLaMA

**Title:** `Built infrastructure for autonomous AI agents to do server ops (with human approval)`

**Body:**
```
For those building agents that need to do real infrastructure work—not just code.

The problem: You can run an LLM agent that writes code and commits to git. But you can't run one that configures servers because production is too risky.

The solution: Isolated VM sandboxes where agents can work autonomously, with human approval before anything hits production.

**What the agent gets:**
- Full root access to a KVM virtual machine
- Ability to install packages, configure services, set up networking
- Snapshot/restore for checkpointing
- No restrictions inside the sandbox

**What the human gets:**
- Diff of exactly what changed
- Auto-generated Ansible playbook
- Blocking approval workflow
- Full audit trail

**Works with any LLM** (tested with GPT-4, Claude, Llama 3, Mixtral). Python SDK makes it easy to build function-calling agents.

Example agent loop:
```python
from virsh_sandbox import VirshSandbox

client = VirshSandbox("http://localhost:8080")

# Agent gets a sandbox
sandbox = client.sandbox.create_sandbox(
    source_vm_name="ubuntu-base",
    agent_id="infra-agent"
).sandbox

# Agent works autonomously
client.sandbox.run_sandbox_command(sandbox.id, "apt update")
client.sandbox.run_sandbox_command(sandbox.id, "apt install -y nginx")
client.sandbox.run_sandbox_command(sandbox.id, "systemctl enable nginx")

# Checkpoint
client.sandbox.create_snapshot(sandbox.id, name="nginx-ready")

# Human reviews diff, approves, Ansible applies to prod
```

GitHub: [link]

Anyone else working on agents for infrastructure automation? Would love to compare approaches.
```

---

## r/MachineLearning

**Title:** `[P] virsh-sandbox: Sandboxed execution environment for autonomous infrastructure agents`

**Body:**
```
Sharing a project for running AI agents that do infrastructure/ops work safely.

**Problem:** LLM agents can write code, but we can't let them configure production servers. The failure modes are too catastrophic.

**Solution:** Isolated VM sandboxes where agents can work autonomously, with human approval before production deployment.

**Architecture:**
- Each agent gets a dedicated KVM virtual machine (full isolation)
- Agent has root access and works autonomously
- VM snapshots provide checkpointing
- Diff between snapshots shows exactly what changed
- Human reviews and approves before production

**Key design decisions:**
1. VMs over containers for true isolation
2. Snapshot-based checkpointing (not logging)
3. Blocking approval workflow (not async notifications)
4. Ansible generation for reproducible production deployment

**Tested with:** GPT-4, Claude 3, Llama 3 70B, Mixtral 8x22B

The SDK provides function-calling tools for:
- Command execution
- File read/write
- Snapshot creation
- Human approval requests

GitHub: [link]
Paper/writeup: [if you have one]

Interested in feedback on the agent-environment interface design. What primitives are missing for infrastructure work?
```

---

## r/homelab

**Title:** `Using AI agents to manage my homelab (safely) - open sourced the tooling`

**Body:**
```
I've been experimenting with AI agents for homelab management and built some tooling that others might find useful.

The setup:
- AI agent gets a sandbox VM (clone of my base image)
- Agent configures things autonomously
- I review what it did via a diff
- Approve → changes apply to my real servers via Ansible

Example tasks I've automated:
- Setting up new services (nginx, postgres, etc.)
- Debugging issues ("why is this container not starting?")
- Security hardening ("audit this server and fix issues")

The nice thing is the agent can try things, make mistakes, and iterate—all in the sandbox. I only see the final result and decide whether to apply it.

Tech: KVM/libvirt, Go API, Python SDK, React UI

Works on Linux with native KVM, or Mac with Lima.

GitHub: [link]

Anyone else using AI for homelab automation? What tasks have you found it useful for?
```
