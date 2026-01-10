# Hacker News - fluid.sh Launch

**Best posting time:** Tuesday-Thursday, 9-11am PT

---

## Title 

1. `Show HN: fluid.sh – Make Infrastructure Safe for AI

---

## Post Body

```
Hey HN,

I built fluid.sh because I wanted AI agents to actually DO infrastructure work—not just suggest commands for me to copy-paste.

The problem: We can give AI agents root access to codebases, but not to servers. One bad command on production and you're restoring from backups. So we limit agents to "suggestion mode" while humans do the actual execution.

The solution: Give the agent root access to an isolated VM instead. Let it work autonomously—provision servers, configure firewalls, set up services, deploy applications. Then diff the changes, review what it did, and approve an Ansible playbook to apply to production.

The agent does the work. The human just approves.

How it works:
1. Clone a golden VM image into an isolated KVM sandbox
2. Agent gets full root access and works autonomously
3. Snapshots checkpoint progress (rollback if needed)
4. Diff snapshots to see exactly what changed
5. Auto-generate Ansible playbook from the diff
6. Human reviews and approves (blocking workflow)
7. Playbook applies to production

Tech stack: Go + libvirt/KVM, React UI, Python SDK, PostgreSQL

Key insight: The bottleneck isn't AI capability—it's trust. This gives you a way to let agents work autonomously while maintaining human oversight at the critical moment.

Use cases I'm excited about:
- Autonomous server provisioning and configuration
- Self-healing infrastructure (agent diagnoses and fixes issues)
- Migration automation (agent figures out the steps)
- Compliance remediation
- On-call automation (agent triages and resolves alerts)

Why VMs instead of containers?
- Full OS isolation (not namespace isolation)
- Real networking stack for firewall/routing work
- Snapshot/restore is native to the hypervisor
- Agents can reboot, modify kernel params, install kernel modules
- Closer to production reality

GitHub: [link]
Demo video: [link]

I'd love feedback on:
1. The approval workflow—does this match how you'd want to deploy AI agents?
2. Use cases I'm missing
3. Security model concerns

Happy to answer questions about the architecture.
```

---

## Response Templates

**For "Why not containers?"**
```
Containers share the host kernel and have limited isolation. For 
infrastructure work where agents might:
- Configure iptables/nftables
- Modify sysctl params
- Install kernel modules
- Reboot the system
- Test systemd units

You need a full VM. Also, snapshot/restore is much cleaner at 
the hypervisor level than Docker commit.
```

**For "What about security?"**
```
A few layers:
1. KVM provides hardware-level isolation (not namespace isolation)
2. VMs run on isolated virtual networks
3. Ephemeral SSH certificates (auto-expire in 1-10 min)
4. Blocking human approval before any production changes
5. Full audit trail of every command
6. Snapshot rollback if something goes wrong

The threat model assumes the agent might do anything inside the 
sandbox—that's fine. The protection is the approval gate before 
production.
```

**For "How does the Ansible generation work?"**
```
Right now it's diff-based—comparing filesystem state between 
snapshots to identify:
- Packages installed (apt/yum history)
- Files created/modified
- Services enabled/started
- User/group changes

Then templating those into Ansible tasks. It's not perfect 
(complex state is hard), but it handles 80% of common 
infrastructure changes well.
```

**For "What LLMs work best?"**
```
Tested with GPT-4, Claude, and Llama 3. The SDK uses function 
calling, so anything that supports that works. For complex 
multi-step infrastructure tasks, GPT-4 and Claude have been 
most reliable in my testing.
```
