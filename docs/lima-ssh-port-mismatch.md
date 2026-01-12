# Lima SSH Port Mismatch: Docker-to-Lima Connectivity Issue

## Overview

This document describes a common connectivity issue when running virsh-sandbox in Docker while using a Lima VM for libvirt/KVM on macOS. The issue occurs because Lima assigns dynamic SSH ports that change when the VM restarts.

**Date**: January 2026
**Affected Components**: Docker container networking, Lima VM SSH port
**Symptom**: API returns 500 error with "Network is unreachable" when creating sandboxes

---

## Problem Description

### Error Message

```
HTTP response body: {"error":"create sandbox: clone vm: lookup source VM \"test-vm\":
virsh --connect qemu+ssh://user@host.docker.internal:58411/system domblklist test-vm
--details failed: exit status 1: error: failed to connect to the hypervisor\nerror:
Cannot recv data: ssh: connect to host host.docker.internal port 58411: Network is
unreachable: Connection reset by peer","code":500}
```

### Root Cause

The virsh-sandbox Docker container connects to libvirt running inside a Lima VM via SSH. The connection uses:

```
LIBVIRT_URI=qemu+ssh://<user>@host.docker.internal:<port>/system
```

**The problem:**

1. Lima assigns a **dynamic SSH port** to VMs (e.g., 52136, 58411, 63255)
2. This port changes every time the Lima VM restarts
3. The `.env` file contains a **stale port number** from a previous Lima session
4. Docker cannot reach the old port, causing "Network is unreachable"

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│  macOS Host                                                         │
│                                                                     │
│  ┌─────────────────────────┐     ┌─────────────────────────────┐   │
│  │  Docker Container       │     │  Lima VM (virsh-sandbox-dev) │   │
│  │                         │     │                              │   │
│  │  virsh-sandbox-api      │     │  libvirtd                    │   │
│  │                         │     │  QEMU/KVM                    │   │
│  │  LIBVIRT_URI=           │────>│  SSH port: 52136 (dynamic)  │   │
│  │  ...@host.docker.       │     │                              │   │
│  │  internal:58411  <──────│─────│──WRONG PORT!                 │   │
│  │                         │     │                              │   │
│  └─────────────────────────┘     └─────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Solution

### Quick Fix

1. **Check the current Lima SSH port:**

   ```bash
   limactl list
   ```

   Output:
   ```
   NAME                 STATUS     SSH                CPUS    MEMORY    DISK
   virsh-sandbox-dev    Running    127.0.0.1:52136    4       8GiB      50GiB
   ```

2. **Update the `.env` file** with the correct port:

   ```bash
   # Edit .env in the repo root
   LIBVIRT_URI=qemu+ssh://<user>@host.docker.internal:52136/system
   ```

3. **Restart the Docker container** to pick up the new environment:

   ```bash
   docker-compose down && docker-compose up -d
   ```

### Verification

Test SSH connectivity from your host before restarting Docker:

```bash
# Replace <port> with the actual Lima SSH port
ssh -o ConnectTimeout=5 -i ~/.lima/_config/user <user>@127.0.0.1 -p <port> "virsh list --all"
```

If this works, the Docker container should be able to connect after restart.

---

## Prevention

### Option 1: Pin the Lima SSH Port (Recommended)

Configure Lima to use a fixed SSH port by editing the Lima VM configuration:

```bash
# Stop the VM first
limactl stop virsh-sandbox-dev

# Edit the configuration
limactl edit virsh-sandbox-dev
```

Add or modify the `ssh` section:

```yaml
ssh:
  localPort: 60022  # Fixed port
```

Then start the VM:

```bash
limactl start virsh-sandbox-dev
```

Update `.env` to use the fixed port:

```
LIBVIRT_URI=qemu+ssh://<user>@host.docker.internal:60022/system
```

### Option 2: Use a Startup Script

Create a script that automatically updates `.env` when starting development:

```bash
#!/bin/bash
# update-lima-port.sh

# Get current Lima SSH port
PORT=$(limactl list virsh-sandbox-dev --format '{{.SSHLocalPort}}' 2>/dev/null)

if [ -z "$PORT" ]; then
    echo "Error: Lima VM not running. Start it with: limactl start virsh-sandbox-dev"
    exit 1
fi

# Get current user
USER=$(whoami)

# Update .env file
sed -i.bak "s|LIBVIRT_URI=qemu+ssh://.*@host.docker.internal:[0-9]*/system|LIBVIRT_URI=qemu+ssh://${USER}@host.docker.internal:${PORT}/system|" .env

echo "Updated .env with Lima SSH port: $PORT"
echo "Restart Docker: docker-compose down && docker-compose up -d"
```

### Option 3: Use TCP Instead of SSH

Configure libvirt to listen on TCP (less secure, but port is stable):

1. Inside the Lima VM, enable TCP listening in `/etc/libvirt/libvirtd.conf`:

   ```
   listen_tls = 0
   listen_tcp = 1
   tcp_port = "16509"
   auth_tcp = "none"
   ```

2. Forward port 16509 from Lima to host (in Lima config):

   ```yaml
   portForwards:
     - guestPort: 16509
       hostPort: 16509
   ```

3. Update `.env`:

   ```
   LIBVIRT_URI=qemu+tcp://host.docker.internal:16509/system
   ```

**Warning:** TCP without authentication is insecure. Only use this for local development.

---

## Troubleshooting

### Check Lima VM Status

```bash
limactl list
```

If status is not "Running", start the VM:

```bash
limactl start virsh-sandbox-dev
```

### Test SSH Connectivity from Host

```bash
# Get the current port
PORT=$(limactl list virsh-sandbox-dev --format '{{.SSHLocalPort}}')

# Test connection
ssh -i ~/.lima/_config/user $(whoami)@127.0.0.1 -p $PORT "echo connected"
```

### Test SSH from Docker Container

```bash
# Start a shell in the virsh-sandbox container
docker exec -it virsh-sandbox-api /bin/sh

# Try connecting (this will fail if port is wrong)
ssh -i /root/.ssh/id_lima $(whoami)@host.docker.internal -p <port> "virsh list --all"
```

### Check Docker Network Resolution

```bash
docker exec virsh-sandbox-api ping -c 1 host.docker.internal
```

If this fails, ensure `extra_hosts` is set in `docker-compose.yml`:

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

### View API Logs

```bash
docker-compose logs virsh-sandbox | grep -i "failed to connect"
```

---

## Related Configuration Files

| File | Purpose |
|------|---------|
| `.env` | Environment variables including `LIBVIRT_URI` |
| `.env.example` | Template for `.env` file |
| `docker-compose.yml` | Docker service configuration |
| `~/.lima/virsh-sandbox-dev/lima.yaml` | Lima VM configuration |
| `~/.lima/_config/user` | Lima SSH private key |

---

## References

- [Lima Documentation](https://lima-vm.io/)
- [Libvirt Connection URIs](https://libvirt.org/uri.html)
- [Docker host.docker.internal](https://docs.docker.com/desktop/networking/#i-want-to-connect-from-a-container-to-a-service-on-the-host)
