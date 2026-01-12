# Lima Mount Missing: cloud-init.iso Not Found Error

## Overview

This document describes an issue where sandbox VMs fail to start because libvirt inside the Lima VM cannot find the cloud-init.iso file created by the Docker container. This is caused by a missing mount configuration in Lima.

**Date**: January 2026
**Affected Components**: Lima VM configuration, libvirt image paths
**Symptom**: API returns 500 error with "Cannot access storage file ... cloud-init.iso: No such file or directory"

---

## Problem Description

### Error Message

```
HTTP response body: {"error":"create sandbox: auto-start vm: virsh --connect
qemu+ssh://user@host.docker.internal:52136/system start sbx-791acd48 failed:
exit status 1: error: Failed to start domain 'sbx-791acd48'\nerror: Cannot
access storage file '/var/lib/libvirt/images/jobs/sbx-791acd48/cloud-init.iso':
No such file or directory","code":500}
```

### Root Cause

The architecture involves three components that need to share the same filesystem path:

1. **Docker container** - Creates the cloud-init.iso file
2. **macOS host** - Hosts the filesystem via bind mount
3. **Lima VM** - Runs libvirt which needs to access the ISO

```
┌──────────────────────────────────────────────────────────────────────────┐
│  macOS Host                                                              │
│                                                                          │
│  /var/lib/libvirt/images/jobs/sbx-xxx/cloud-init.iso  <-- File exists   │
│                    │                                                     │
│         ┌─────────┴─────────┐                                           │
│         │                   │                                            │
│         ▼                   ▼                                            │
│  ┌─────────────────┐  ┌─────────────────────┐                           │
│  │ Docker Container │  │ Lima VM             │                           │
│  │                  │  │                     │                           │
│  │ bind mount:      │  │ /var/lib/libvirt/  │                           │
│  │ /var/lib/libvirt │  │ images/            │                           │
│  │ /images/jobs     │  │                     │                           │
│  │     │            │  │  ❌ NOT MOUNTED!    │                           │
│  │     │            │  │  (empty directory)  │                           │
│  │     ▼            │  │                     │                           │
│  │ Creates ISO here │  │ libvirt tries to   │                           │
│  │                  │  │ access ISO here    │                           │
│  └─────────────────┘  └─────────────────────┘                           │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

The problem:
1. Docker creates cloud-init.iso at `/var/lib/libvirt/images/jobs/sbx-xxx/` (bind-mounted from host)
2. The file exists on the macOS host at that path
3. Lima VM's `/var/lib/libvirt/images/` is a local directory (not mounted from host)
4. Libvirt inside Lima looks for the ISO but finds an empty directory

---

## Solution

### Fix for Existing Lima VMs

1. **Stop the Lima VM:**
   ```bash
   limactl stop virsh-sandbox-dev --force
   ```

2. **Edit the Lima configuration:**
   ```bash
   # Open the config file directly
   vim ~/.lima/virsh-sandbox-dev/lima.yaml

   # Or use limactl edit
   limactl edit virsh-sandbox-dev
   ```

3. **Add the mount configuration** (add after the `firmware:` section):
   ```yaml
   # Mount libvirt images directory to share between host, Docker, and Lima
   # This is required for Docker-created cloud-init ISOs to be accessible by libvirt
   mounts:
     - location: "/var/lib/libvirt/images"
       writable: true
   ```

4. **Start the Lima VM:**
   ```bash
   limactl start virsh-sandbox-dev
   ```

5. **Update .env with new SSH port** (Lima port changes on restart):
   ```bash
   # Check new port
   limactl list

   # Update .env
   # LIBVIRT_URI=qemu+ssh://user@host.docker.internal:<new-port>/system
   ```

6. **Restart Docker container:**
   ```bash
   docker-compose down && docker-compose up -d
   ```

### Verification

Verify the mount is working:

```bash
# Create a test file on the host
sudo touch /var/lib/libvirt/images/jobs/test-file

# Check if it's visible in Lima
limactl shell virsh-sandbox-dev ls /var/lib/libvirt/images/jobs/test-file

# Clean up
sudo rm /var/lib/libvirt/images/jobs/test-file
```

---

## Prevention

The setup script (`virsh-sandbox/scripts/setup-lima-libvirt.sh`) has been updated to include this mount by default. New Lima VMs created with the script will have the correct configuration.

If you created the Lima VM before this fix, you'll need to manually add the mount as described above.

### Complete Lima Mount Configuration

```yaml
# Mount libvirt images directory to share between host, Docker, and Lima
# This is required for Docker-created cloud-init ISOs to be accessible by libvirt
mounts:
  - location: "/var/lib/libvirt/images"
    writable: true
```

---

## Related Issues

This issue often occurs together with the [Lima SSH Port Mismatch](./lima-ssh-port-mismatch.md) issue, since restarting Lima to apply the mount configuration also changes the dynamic SSH port.

When fixing this issue, remember to:
1. Add the mount configuration
2. Restart Lima
3. Update the SSH port in `.env`
4. Restart Docker

---

## Architecture Notes

### Why Three Components Need the Same Path

1. **Docker container** runs the virsh-sandbox API which:
   - Creates sandbox directories at `/var/lib/libvirt/images/jobs/<sandbox-id>/`
   - Generates cloud-init.iso files for new sandboxes
   - Defines VM XML with paths like `/var/lib/libvirt/images/jobs/sbx-xxx/cloud-init.iso`

2. **macOS host** provides the actual filesystem:
   - Docker bind-mounts `${JOBS_DIR:-/var/lib/libvirt/images/jobs}` into the container
   - The files physically exist at `/var/lib/libvirt/images/jobs/` on macOS

3. **Lima VM** runs libvirt which:
   - Executes `virsh start` commands
   - Needs to access the cloud-init.iso to boot the VM
   - Must see the same path that Docker wrote to

### Mount Types

Lima supports several mount types:
- `9p` (default) - Plan 9 filesystem protocol
- `virtiofs` - Higher performance but requires specific kernel support
- `sshfs` - SSH-based mounting

The default `9p` mount works for this use case.
