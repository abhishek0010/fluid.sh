# virsh-sandbox

Autonomous AI agents for infrastructure—with human approval.

## What This Is

virsh-sandbox lets AI agents do infrastructure work in isolated VM sandboxes. Agent works autonomously. Human approves before production.

## Project Structure

```
virsh-sandbox/    # Go API - VM management via libvirt
tmux-client/      # Go API - Terminal, files, commands
web/              # React - UI for monitoring/approval
sdk/              # Python SDK - Build agents
examples/         # Working agent examples
landing-page/     # Astro - Marketing site (fluid.sh)
```

## Testing Required

Every code change needs tests. See project-specific AGENTS.md files for details.

## Quick Reference

```bash
docker-compose up --build              # Start everything
cd virsh-sandbox && make test          # Test API
cd tmux-client && make test            # Test terminal service
cd sdk/virsh-sandbox-py && pytest      # Test SDK
```

## Project Docs

- @virsh-sandbox/AGENTS.md
- @tmux-client/AGENTS.md
- @sdk/AGENTS.md
- @web/AGENTS.md
- @examples/agent-example/AGENTS.md
- @landing-page/AGENTS.md
