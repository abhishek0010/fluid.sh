# Lima AppArmor: QEMU Permission Denied on 9p Mounts

## Overview

This document describes a permissions issue where QEMU inside Lima cannot access disk images on 9p-mounted host directories due to AppArmor restrictions.

**Date**: January 2026
**Affected Components**: Lima VM, libvirt/QEMU, AppArmor
**Symptom**: API returns 500 error with "Could not open ... Permission denied"

---

## Problem Description

### Error Message

```
HTTP response body: {"error":"create sandbox: auto-start vm: ... Could not open
'/var/lib/libvirt/images/jobs/sbx-xxx/disk-overlay.qcow2': Permission denied"}
```

### Root Cause

Ubuntu's AppArmor profile for libvirt restricts which paths QEMU can access. The default profile allows access to:
- `/var/lib/libvirt/images/**`
- `/var/lib/libvirt/boot/**`
- Various system paths

However, when `/var/lib/libvirt/images` is a 9p mount from the host (rather than a local directory), AppArmor blocks access because:
1. The 9p mount appears as a different filesystem type
2. AppArmor profiles may not account for virtualized filesystem paths
3. The security context of files on the mount doesn't match expected patterns

### Verification

You can verify this is an AppArmor issue by checking:

```bash
# Check AppArmor status
limactl shell virsh-sandbox-dev sudo aa-status | grep libvirt

# Check for denials in audit log
limactl shell virsh-sandbox-dev sudo dmesg | grep -i "apparmor.*DENIED"
```

---

## Solution

### Quick Fix for Existing Lima VMs

Disable the security driver and configure QEMU to run as root in libvirt's QEMU configuration:

```bash
limactl shell virsh-sandbox-dev sudo bash -c 'cat >> /etc/libvirt/qemu.conf << EOF

# Disable security driver to allow access to 9p-mounted host paths
security_driver = "none"

# Run QEMU as root to access 9p-mounted files owned by host user
user = "root"
group = "root"
EOF
systemctl restart libvirtd'
```

**Note:** Both settings are required:
- `security_driver = "none"` disables AppArmor restrictions
- `user = "root"` and `group = "root"` allow QEMU to access files owned by the host user on 9p mounts

### Verification

Verify the change took effect:

```bash
limactl shell virsh-sandbox-dev sudo grep security_driver /etc/libvirt/qemu.conf
# Should output: security_driver = "none"
```

### For New Lima VMs

The setup script (`virsh-sandbox/scripts/setup-lima-libvirt.sh`) has been updated to include this configuration by default.

---

## Security Considerations

**Warning:** Disabling the security driver removes an important security layer. This is acceptable for local development but should NOT be used in production.

The security driver (AppArmor on Ubuntu, SELinux on RHEL) provides:
- Isolation between VMs
- Protection against VM escape attacks
- Restriction of QEMU's access to host filesystem

### Alternative Solutions (More Secure)

If you need to maintain security:

1. **Add AppArmor exception for 9p mount:**
   ```bash
   # Create a local AppArmor override
   sudo tee /etc/apparmor.d/local/abstractions/libvirt-qemu << 'EOF'
   /var/lib/libvirt/images/** rwk,
   EOF
   sudo apparmor_parser -r /etc/apparmor.d/libvirt/TEMPLATE.qemu
   ```

2. **Use virtiofs instead of 9p:**
   ```yaml
   # In lima.yaml
   mountType: virtiofs
   ```
   Note: virtiofs may have better AppArmor integration but requires specific kernel support.

3. **Store disk images locally in Lima:**
   Instead of sharing via 9p, copy disk images into the Lima VM's local filesystem.

---

## Related Files

| File | Purpose |
|------|---------|
| `/etc/libvirt/qemu.conf` | QEMU configuration including security driver |
| `/etc/apparmor.d/libvirt/TEMPLATE.qemu` | AppArmor template for QEMU VMs |
| `/etc/apparmor.d/abstractions/libvirt-qemu` | Shared AppArmor rules for libvirt |

---

## Related Issues

This issue typically occurs after fixing the [Lima Mount Missing](./lima-mount-missing.md) issue. Once the mount is added, AppArmor blocks access to the mounted files.

The full sequence of fixes for a working Lima setup:
1. [Lima SSH Port Mismatch](./lima-ssh-port-mismatch.md) - Fix dynamic SSH port
2. [Lima Mount Missing](./lima-mount-missing.md) - Add 9p mount for images directory
3. **This document** - Disable AppArmor security driver
