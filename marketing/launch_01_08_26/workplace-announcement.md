# Workplace Announcement - virsh-sandbox

Use this for Slack, Teams, or internal email announcements.

---

## Slack/Teams Post

```
🚀 Just open-sourced a side project: virsh-sandbox

TL;DR: A platform for autonomous AI agents to do infrastructure work—with human approval before production.

**The problem:** AI agents can write code, but we can't let them configure servers. Too risky.

**The solution:** Give agents isolated VM sandboxes with full root access. They work autonomously, then a human reviews the diff and approves before anything touches production.

**Why it matters for us:**
- Could automate routine infrastructure tasks
- AI agent does the work, engineer just approves
- Safe way to experiment with AI for ops

**Demo:** [video link]
**GitHub:** [link]

Happy to do a lunch & learn demo if there's interest! 🙋
```

---

## Email Version

**Subject:** Side project launch: AI agents for infrastructure automation

```
Hi team,

I wanted to share a side project I've been working on that I think could be interesting for [company/team].

**virsh-sandbox** is a platform for running AI agents that do infrastructure work autonomously—with human approval before any production changes.

**The idea:**
Instead of AI assistants that suggest commands for humans to execute, what if agents could actually do the work? Install packages, configure services, set up networking—all autonomously in an isolated VM. Then a human reviews what changed and approves before production.

**How it works:**
1. Agent gets a fresh VM sandbox (cloned from a golden image)
2. Agent works autonomously with full root access
3. Snapshots capture checkpoints
4. Human reviews the diff of what changed
5. Approve → auto-generated Ansible applies to production

**Potential use cases for [company]:**
- Automating routine server setup tasks
- Self-healing infrastructure (agent diagnoses and fixes issues)
- Onboarding new services (agent provisions, configures, tests)
- Compliance automation

**Links:**
- GitHub: [link]
- Demo video: [link]
- Blog post: [link]

I'd be happy to do a demo if anyone's interested. Just let me know!

[Your name]
```

---

## Lunch & Learn Outline (30 min)

### Intro (5 min)
- The problem: AI can code but can't do ops
- The vision: Autonomous agents with human approval

### Demo (15 min)
- Create a sandbox
- Show agent working autonomously
- Create snapshots
- Review diff in Web UI
- Show approval workflow
- (Optional) Apply to a test server

### Architecture (5 min)
- Why VMs over containers
- Security model
- How Ansible generation works

### Discussion (5 min)
- Use cases for our team
- Questions and feedback
