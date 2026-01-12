# VM Architecture Mismatch on Apple Silicon

## Overview

This document describes an issue where VMs fail to boot or run extremely slowly because the base disk image architecture doesn't match the host architecture.

**Date**: January 2026
**Affected Components**: Base VM disk images, libvirt domain configuration
**Symptom**: VMs fail to start, boot extremely slowly, or crash during boot

---

## Problem Description

### Symptoms

- VMs take several minutes to show any boot activity
- QEMU uses 100% CPU but VM makes no progress
- VMs crash with CPU-related errors
- Boot timeout errors from the API

### Root Cause

The `create-test-vm.sh` script was hardcoded to download an **amd64** (x86_64) cloud image regardless of the host architecture:

```bash
# Old (incorrect) code
CLOUD_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/ubuntu-22.04-minimal-cloudimg-amd64.img"
```

On Apple Silicon Macs (M1/M2/M3), the Lima VM runs as `aarch64`. When creating a VM with an x86_64 disk image:

1. QEMU must emulate the entire x86_64 instruction set
2. This is approximately 10-100x slower than native execution
3. Boot times increase from seconds to minutes (or hours)
4. Some x86_64 instructions may not emulate correctly

### Verification

Check if you have a mismatch:

```bash
# Check host architecture
uname -m
# Should show: arm64 (macOS) or aarch64 (Linux)

# Check base image backing file
qemu-img info /var/lib/libvirt/images/base/test-vm.qcow2 | grep backing
# If it shows "amd64" on an ARM host, you have a mismatch
```

---

## Solution

### For Existing VMs

1. **Check current backing file:**
   ```bash
   qemu-img info /var/lib/libvirt/images/base/test-vm.qcow2 | grep backing
   ```

2. **If it shows amd64 on ARM, recreate the disk:**
   ```bash
   # Remove old VM definition
   limactl shell virsh-sandbox-dev virsh undefine test-vm --nvram

   # Remove old disk
   rm /var/lib/libvirt/images/base/test-vm.qcow2

   # Download ARM64 image if not present
   wget -O /var/lib/libvirt/images/base/ubuntu-22.04-server-cloudimg-arm64.img \
     https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-arm64.img

   # Create new disk with correct backing
   qemu-img create -f qcow2 -F qcow2 \
     -b /var/lib/libvirt/images/base/ubuntu-22.04-server-cloudimg-arm64.img \
     /var/lib/libvirt/images/base/test-vm.qcow2 10G
   ```

3. **Recreate the VM definition** (see example XML below)

### Script Fix

The `create-test-vm.sh` and `setup-lima-libvirt.sh` scripts have been updated to auto-detect architecture and use appropriate virt-install options:

```bash
# Detect architecture and select appropriate cloud image and virt-install options
ARCH=$(uname -m)
case "${ARCH}" in
    aarch64|arm64)
        CLOUD_IMAGE_ARCH="arm64"
        CLOUD_IMAGE_URL="https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-arm64.img"
        # ARM64 requires specific virt-install options
        VIRT_INSTALL_ARCH_OPTS="--arch aarch64 --machine virt --boot uefi"
        ;;
    x86_64|amd64)
        CLOUD_IMAGE_ARCH="amd64"
        CLOUD_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/ubuntu-22.04-minimal-cloudimg-amd64.img"
        # x86_64 uses default options
        VIRT_INSTALL_ARCH_OPTS=""
        ;;
    *)
        echo "Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

# virt-install with architecture options
sudo virt-install \
    --name "${VM_NAME}" \
    --disk "path=${VM_DISK},format=qcow2,bus=virtio" \
    --disk "path=${CLOUD_INIT_ISO},device=cdrom,bus=scsi" \
    --controller scsi,model=virtio-scsi \
    --network network=default,model=virtio \
    ${VIRT_INSTALL_ARCH_OPTS}  # Adds --arch, --machine, --boot for ARM64
```

---

## Example ARM64 VM Definition

For Apple Silicon Macs, use this VM configuration:

```xml
<domain type='qemu'>
  <name>test-vm</name>
  <memory unit='MiB'>2048</memory>
  <vcpu>2</vcpu>
  <os>
    <type arch='aarch64' machine='virt'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/AAVMF/AAVMF_CODE.fd</loader>
    <nvram template='/usr/share/AAVMF/AAVMF_VARS.fd'/>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <gic version='3'/>
  </features>
  <cpu mode='custom'>
    <model>cortex-a57</model>
  </cpu>
  <devices>
    <emulator>/usr/bin/qemu-system-aarch64</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='writeback'/>
      <source file='/var/lib/libvirt/images/base/test-vm.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='/var/lib/libvirt/images/base/test-vm-cloud-init.iso'/>
      <target dev='sda' bus='scsi'/>
      <readonly/>
    </disk>
    <controller type='scsi' model='virtio-scsi'/>
    <interface type='network'>
      <source network='default'/>
      <model type='virtio'/>
    </interface>
    <console type='pty'>
      <target type='serial'/>
    </console>
  </devices>
</domain>
```

Key differences for ARM64:
- `type='qemu'` instead of `type='kvm'` (no nested KVM on Apple Silicon)
- `arch='aarch64'` and `machine='virt'`
- AAVMF firmware instead of OVMF
- `gic version='3'` for ARM interrupt controller
- `cpu mode='custom'` with `cortex-a57` model

---

## KVM vs QEMU Emulation

On Apple Silicon with Lima:
- **KVM is not available** for nested virtualization
- VMs run under **QEMU TCG** (Tiny Code Generator) emulation
- This is slower than native KVM but still usable
- Using the correct architecture minimizes emulation overhead

Performance comparison:
| Configuration | Boot Time | Performance |
|--------------|-----------|-------------|
| ARM64 on ARM64 (QEMU) | ~2-3 min | Acceptable |
| x86_64 on ARM64 (QEMU) | 10+ min | Very slow |
| ARM64 on ARM64 (KVM) | ~30 sec | Fast (not available on Lima) |

---

## Related Issues

This issue is often discovered after fixing:
1. [Lima SSH Port Mismatch](./lima-ssh-port-mismatch.md)
2. [Lima Mount Missing](./lima-mount-missing.md)
3. [Lima AppArmor Permission Denied](./lima-apparmor-permission-denied.md)

If VMs still fail to start after fixing those issues, check the architecture mismatch.
