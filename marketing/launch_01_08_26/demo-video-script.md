# Demo Video Script - virsh-sandbox Launch

**Target Length:** 2-3 minutes  
**Tone:** Technical but accessible, show don't tell  
**Tools:** Terminal recording (asciinema or OBS), Web UI, code editor

---

## Script

### HOOK (0:00 - 0:15)

**[Screen: Terminal with blinking cursor]**

**Voiceover:**
> "What if you could tell an AI agent: 'Set up nginx with SSL and rate limiting'... and it just does it?"

**[Screen: Fast montage of commands executing]**

> "Not suggestions. Not copy-paste. The agent actually does the work."

**[Screen: Approval UI appears]**

> "Then you review and approve. This is virsh-sandbox."

---

### THE PROBLEM (0:15 - 0:35)

**[Screen: Split - left shows AI chat suggesting commands, right shows human typing them]**

**Voiceover:**
> "Today's AI assistants are stuck in suggestion mode. They tell you what to run, but YOU still have to execute every command."

**[Screen: Red X over 'production server']**

> "Why? Because we can't give AI agents root on production. One mistake and you're restoring from backups."

---

### THE SOLUTION (0:35 - 1:00)

**[Screen: Architecture diagram animating]**

**Voiceover:**
> "virsh-sandbox gives agents an isolated VM to work in. Full root access. Real OS. They do the work autonomously."

**[Screen: Diff view showing changes]**

> "Then you see exactly what changed..."

**[Screen: Approval button]**

> "...and approve. Ansible applies to production."

**[Text overlay: "Agent does the work. Human approves."]**

---

### DEMO PART 1: Create Sandbox (1:00 - 1:30)

**[Screen: Terminal with Python code]**

**Voiceover:**
> "Let me show you. First, we create a sandbox from a base Ubuntu image."

```python
from virsh_sandbox import VirshSandbox

client = VirshSandbox("http://localhost:8080")

sandbox = client.sandbox.create_sandbox(
    source_vm_name="ubuntu-base",
    agent_id="demo-agent",
    auto_start=True,
    wait_for_ip=True
).sandbox

print(f"Sandbox ready: {sandbox.id}")
```

**[Screen: Output showing sandbox ID and IP]**

> "The agent now has a fresh VM with root access. Let's put it to work."

---

### DEMO PART 2: Agent Works Autonomously (1:30 - 2:15)

**[Screen: Terminal showing commands executing]**

**Voiceover:**
> "The agent installs nginx..."

```python
client.sandbox.run_sandbox_command(sandbox.id, "apt update && apt install -y nginx")
```

**[Screen: Output scrolling]**

> "Configures SSL with certbot..."

```python
client.sandbox.run_sandbox_command(sandbox.id, "apt install -y certbot python3-certbot-nginx")
client.sandbox.run_sandbox_command(sandbox.id, "certbot --nginx -d demo.example.com --non-interactive")
```

> "Sets up rate limiting..."

```python
client.sandbox.run_sandbox_command(sandbox.id, """
cat > /etc/nginx/conf.d/rate-limit.conf << 'EOF'
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
EOF
""")
```

> "And creates a checkpoint."

```python
client.sandbox.create_snapshot(sandbox.id, name="nginx-configured")
```

**[Text overlay: "All autonomous. No human in the loop yet."]**

---

### DEMO PART 3: Human Approval (2:15 - 2:45)

**[Screen: Web UI showing diff view]**

**Voiceover:**
> "Now let's see what the agent did."

**[Screen: Highlight packages installed, files changed, services enabled]**

> "We can see every package installed, every file changed, every service configured."

**[Screen: Generated Ansible playbook]**

> "And here's the auto-generated Ansible playbook, ready to apply to production."

**[Screen: Click approve button]**

> "One click to approve..."

**[Screen: Ansible running on production]**

> "...and the same changes apply to production."

---

### CLOSING (2:45 - 3:00)

**[Screen: GitHub repo]**

**Voiceover:**
> "virsh-sandbox. Autonomous AI agents for infrastructure, with human approval."

**[Screen: Text overlays]**
- GitHub: github.com/[your-org]/virsh-sandbox
- Python SDK included
- Open source

> "Star the repo and start building."

**[End card with logo and links]**

---

## B-Roll Suggestions

- Terminal scrolling with apt install output
- Web UI showing sandbox list
- Diff view with syntax highlighting
- Tmux session showing live agent work
- Architecture diagram animation

## Recording Tips

1. **Pre-record terminal sessions** - Don't do live typing, use asciinema or script playback
2. **Speed up waiting** - 2x speed for apt install, etc.
3. **Add subtle background music** - Lo-fi or ambient tech
4. **Use text overlays** - Reinforce key points
5. **Keep terminal font large** - Readable on mobile
6. **Web UI should be clean** - Hide browser chrome, use a clean theme

## Thumbnail Ideas

- Split screen: Robot arm + server rack + human with approval button
- Terminal with "Agent: Done. Awaiting approval."
- Before/after: "Suggestion mode" vs "Autonomous execution"
