# Lima 9p Mount chmod Error

## Overview

This document describes an issue where the `libvirt-daemon-system` package fails to install because it cannot change permissions on a 9p-mounted directory.

**Date**: January 2026
**Affected Components**: Lima VM, libvirt-daemon-system package, 9p filesystem
**Symptom**: Package installation fails with "chmod: changing permissions of '/var/lib/libvirt/images/': Operation not permitted"

---

## Problem Description

### Error Message

```
chmod: changing permissions of '/var/lib/libvirt/images/': Operation not permitted
dpkg: error processing package libvirt-daemon-system (--configure):
 installed libvirt-daemon-system package post-installation script subprocess returned error exit status 1
```

### Root Cause

When a host directory is mounted directly at `/var/lib/libvirt/images` via Lima's 9p filesystem:
1. The `libvirt-daemon-system` package's post-install script tries to `chmod` this directory
2. 9p mounts don't support permission changes - permissions are inherited from the host
3. The package installation fails, leaving libvirtd unconfigured

### Verification

Check if the mount is causing the issue:

```bash
# Inside Lima VM
mount | grep libvirt
df -h /var/lib/libvirt/images
```

---

## Solution

### Mount to an Alternate Location

Instead of mounting directly at `/var/lib/libvirt/images`, mount to `/mnt/host-libvirt-images` and symlink the subdirectories after libvirt is installed.

#### Lima Configuration

```yaml
mounts:
  - location: "/var/lib/libvirt/images"
    mountPoint: "/mnt/host-libvirt-images"
    writable: true
```

#### Provision Script

After libvirt is installed, create symlinks:

```bash
# Create directories in the mounted location
mkdir -p /mnt/host-libvirt-images/base
mkdir -p /mnt/host-libvirt-images/jobs

# Remove libvirt's default directories and replace with symlinks
rm -rf /var/lib/libvirt/images/base
rm -rf /var/lib/libvirt/images/jobs
ln -sf /mnt/host-libvirt-images/base /var/lib/libvirt/images/base
ln -sf /mnt/host-libvirt-images/jobs /var/lib/libvirt/images/jobs
```

### Verification

```bash
# Check symlinks are in place
ls -la /var/lib/libvirt/images/
# Should show:
# base -> /mnt/host-libvirt-images/base
# jobs -> /mnt/host-libvirt-images/jobs

# Verify libvirt can access the directories
virsh -c qemu:///system list --all
```

---

## Why This Works

1. The libvirt package installs without errors because `/var/lib/libvirt/images` is a regular directory during installation
2. After installation, we symlink the subdirectories (`base` and `jobs`) to the 9p-mounted location
3. QEMU can access disk images through the symlinks because we've disabled AppArmor restrictions (see `lima-apparmor-permission-denied.md`)

---

## Related Issues

This issue typically occurs in combination with other Lima/libvirt issues:

1. [Lima SSH Port Mismatch](./lima-ssh-port-mismatch.md) - Dynamic SSH port changes
2. [Lima Mount Missing](./lima-mount-missing.md) - Missing 9p mounts
3. [Lima AppArmor Permission Denied](./lima-apparmor-permission-denied.md) - AppArmor blocks 9p mount access
4. **This document** - Package install fails due to chmod on 9p mount

---

## Implementation

The fix has been implemented in `virsh-sandbox/scripts/setup-lima-libvirt.sh`:
- Mount configuration uses `/mnt/host-libvirt-images` as the mount point
- Provision script creates symlinks after libvirt package installation
