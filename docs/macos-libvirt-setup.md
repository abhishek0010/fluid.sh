# macOS Libvirt Setup Guide

This guide explains how to set up libvirt on macOS for use with virsh-sandbox.

## Overview

The virsh-sandbox container connects to libvirt via SSH (`qemu+ssh://`). This approach:
- Works the same for local Mac development and production bare metal servers
- Requires no special libvirt configuration
- Is secure (SSH encrypted)

## Prerequisites

- macOS 11+ (Big Sur or later)
- Homebrew installed
- Docker Desktop installed

## Installation

### 1. Install libvirt and QEMU

```bash
brew install libvirt qemu cdrtools
brew services start libvirt
```

### 2. Enable Remote Login (SSH)

The container connects to your Mac via SSH. Enable it:

1. Open **System Settings**
2. Go to **General > Sharing**
3. Enable **Remote Login**
4. Note your username (shown in the Remote Login panel)

### 3. Create image directories

```bash
sudo mkdir -p /var/lib/libvirt/images/base
sudo mkdir -p /var/lib/libvirt/images/jobs
sudo chmod 777 /var/lib/libvirt/images/base
sudo chmod 777 /var/lib/libvirt/images/jobs
```

### 4. Verify SSH works

```bash
# Test SSH to localhost
ssh $(whoami)@localhost

# Test libvirt over SSH
virsh -c qemu+ssh://$(whoami)@localhost/session list --all
```

## Configuration

### Environment Setup

Copy and edit the example environment file:

```bash
cp .env.example .env
```

Edit `.env` and set your username:

```env
LIBVIRT_URI=qemu+ssh://yourusername@host.docker.internal/session
```

### Docker Compose

The `docker-compose.yml` mounts your SSH key for authentication:

```yaml
volumes:
  - ~/.ssh/id_ed25519:/root/.ssh/id_rsa:ro
```

If you use a different SSH key, update this path.

## Creating a Test VM

Use the provided script:

```bash
./scripts/reset-libvirt-macos.sh
```

This script:
1. Deletes all existing VMs
2. Downloads Ubuntu cloud image (if needed)
3. Creates a test VM with cloud-init

## Running the API

```bash
# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f virsh-sandbox

# Verify connection
curl http://localhost:8080/v1/health
curl http://localhost:8080/v1/sandboxes
```

## Troubleshooting

### SSH connection refused

1. Verify Remote Login is enabled in System Settings
2. Test SSH manually: `ssh yourusername@localhost`

### Permission denied (publickey)

The container needs your SSH key. Verify the volume mount in `docker-compose.yml`:

```yaml
- ~/.ssh/id_ed25519:/root/.ssh/id_rsa:ro
```

### "Host key verification failed"

The container needs to trust the host. Add to the container's known_hosts or use:

```bash
# In the container
ssh-keyscan host.docker.internal >> /root/.ssh/known_hosts
```

### VM won't start

Check libvirt is running:

```bash
brew services list | grep libvirt
virsh -c qemu:///session list --all
```

## Quick Reference

```bash
# Connection URI
export LIBVIRT_URI="qemu+ssh://$(whoami)@localhost/session"

# List VMs
virsh -c $LIBVIRT_URI list --all

# Start VM
virsh -c $LIBVIRT_URI start test-vm

# Stop VM
virsh -c $LIBVIRT_URI destroy test-vm

# VM console
virsh -c $LIBVIRT_URI console test-vm

# Delete VM
virsh -c $LIBVIRT_URI undefine test-vm --nvram

# Reset and recreate test VM
./scripts/reset-libvirt-macos.sh
```

## Production Setup

For production with Foreman-managed bare metal servers:

```env
# Point to your bare metal server
LIBVIRT_URI=qemu+ssh://virsh-user@baremetal-host.example.com/system
```

The same SSH-based approach works - just change the host in the URI.
