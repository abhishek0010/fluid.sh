# Virsh Sandbox Project Rules

This is a monorepo containing multiple projects. See project-specific rules below.

## Mandatory Testing Requirement

**CRITICAL**: After every code change, tests MUST be created or updated to verify the new behavior. Changes without corresponding tests are not considered complete.

## Project References

- @sdk/AGENTS.md - Python SDK for virsh-sandbox API
- @virsh-sandbox/AGENTS.md - Main virsh-sandbox Go service
- @tmux-client/AGENTS.md - Tmux client Go service
- @web/AGENTS.md - Web frontend
- @examples/agent-example/AGENTS.md - Example agent implementation
