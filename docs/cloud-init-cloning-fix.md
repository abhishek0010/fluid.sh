# Cloud-Init Cloning Fix for Sandbox VMs

## Overview

This document describes a critical bug fix for sandbox VM networking that was causing `run_command` API calls to timeout. The fix ensures that cloned VMs properly initialize their network interfaces through cloud-init.

**Date**: January 2026  
**Affected Components**: `virsh-sandbox/internal/libvirt/virsh.go`  
**Symptom**: `run_command` times out with "IP discovery timeout" error

---

## Problem Description

### Symptoms

When creating a sandbox from a base VM and calling `run_command`, the API would:
1. Wait for approximately 2 minutes (the IP discovery timeout)
2. Return a 500 error with message: "IP discovery timeout"
3. Leave the sandbox VM in a running state but with no network connectivity

### Root Cause Analysis

The issue had two contributing factors:

#### 1. Cloud-Init Instance-ID Collision (Primary Cause)

When cloning a VM using qcow2 overlay disks, the clone inherits the base VM's disk state, which includes cloud-init's record of having already run for a specific `instance-id`.

**The problem flow:**
1. Base VM `test-vm-arm64` boots with cloud-init ISO containing `instance-id: test-vm-arm64`
2. Cloud-init runs, configures networking for MAC address `52:54:00:14:74:62`, and records completion
3. Sandbox `sbx-abc123` is created as a linked clone (qcow2 overlay on base disk)
4. Sandbox boots with the **same** cloud-init ISO (`instance-id: test-vm-arm64`)
5. Cloud-init checks instance-id → matches stored value → **skips initialization**
6. Sandbox has a **different** MAC address (`52:54:00:xx:xx:xx`) but no network config for it
7. No DHCP request is sent → No IP address is obtained

**Evidence observed:**
```bash
# VM interface statistics showing zero TX packets after 2+ minutes
$ virsh domifstat sbx-abc123 vnet15
vnet15 rx_bytes 180
vnet15 rx_packets 2
vnet15 tx_bytes 0        # No outgoing traffic!
vnet15 tx_packets 0
```

#### 2. Slow ARM64 Emulation (Secondary Factor)

VMs running under QEMU TCG emulation (when KVM is unavailable, e.g., ARM64 on x86 host) take significantly longer to boot:
- Expected boot time: 15-30 seconds
- Actual boot time under TCG: 150+ seconds
- Default IP discovery timeout: 120 seconds

This meant that even if networking was properly configured, the timeout would expire before the VM could boot and obtain an IP.

---

## Solution

### Fix Implementation

The fix generates a **unique cloud-init ISO for each sandbox** with a new `instance-id`, forcing cloud-init to re-run network initialization.

#### Changes Made

**1. Added `CloudInitISO` field to domain XML parameters:**
```go
type domainXMLParams struct {
    Name         string
    MemoryMB     int
    VCPUs        int
    DiskPath     string
    CloudInitISO string // NEW: Optional path to cloud-init ISO
    Network      string
    // ...
}
```

**2. Updated domain XML template to include CDROM device:**
```xml
{{- if .CloudInitISO }}
<disk type="file" device="cdrom">
  <driver name="qemu" type="raw"/>
  <source file="{{ .CloudInitISO }}"/>
  <target dev="sda" bus="scsi"/>
  <readonly/>
</disk>
<controller type="scsi" model="virtio-scsi"/>
{{- end }}
```

**3. Added `buildCloudInitSeedForClone()` function:**

This function creates a minimal cloud-init seed that:
- Uses the sandbox name as a unique `instance-id`
- Includes a netplan configuration that enables DHCP on virtio interfaces
- Preserves existing user accounts from the base image

```go
func (m *VirshManager) buildCloudInitSeedForClone(ctx context.Context, vmName, outISO string) error {
    userData := `#cloud-config
network:
  version: 2
  ethernets:
    id0:
      match:
        driver: virtio*
      dhcp4: true
`
    metaData := fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, vmName, vmName)
    
    // Create ISO using genisoimage or cloud-localds
    // ...
}
```

**4. Modified `CloneFromVM()` to generate unique cloud-init ISO:**

```go
func (m *VirshManager) CloneFromVM(...) (DomainRef, error) {
    // ... existing code ...
    
    // Detect if source VM has cloud-init
    if sourceCloudInitISO != "" {
        cloudInitISO = filepath.Join(jobDir, "cloud-init.iso")
        if err := m.buildCloudInitSeedForClone(ctx, newVMName, cloudInitISO); err != nil {
            log.Printf("WARNING: failed to build cloud-init seed: %v", err)
            cloudInitISO = sourceCloudInitISO // Fallback
        }
    }
    
    // Include in domain XML
    xml, err := renderDomainXML(domainXMLParams{
        CloudInitISO: cloudInitISO,
        // ...
    })
}
```

### How the Fix Works

1. When `CloneFromVM()` is called, it detects if the source VM has a cloud-init CDROM
2. If yes, it generates a new cloud-init ISO at `/var/lib/libvirt/images/jobs/<sandbox-name>/cloud-init.iso`
3. The ISO contains:
   - `meta-data` with `instance-id: <sandbox-name>` (unique per sandbox)
   - `user-data` with network configuration for DHCP
4. When the sandbox boots, cloud-init sees a **different** instance-id
5. Cloud-init re-runs initialization, including network configuration
6. The netplan config enables DHCP on the virtio network interface
7. The VM obtains an IP address via DHCP

---

## Verification

### Test Results

After the fix, sandbox VMs successfully obtain IP addresses:

```bash
# Create sandbox with auto_start
$ curl -X POST http://localhost:8080/v1/sandboxes \
  -H "Content-Type: application/json" \
  -d '{"source_vm_name": "test-vm-arm64", "agent_id": "test", "auto_start": true}'

# After ~150 seconds (ARM64 boot time), check for IP
$ virsh domifaddr sbx-28a48bc8 --source lease
 Name       MAC address          Protocol     Address
-------------------------------------------------------------------------------
 vnet18     52:54:00:b8:09:c3    ipv4         192.168.122.228/24

# Verify connectivity
$ ping -c 3 192.168.122.228
64 bytes from 192.168.122.228: icmp_seq=1 ttl=64 time=12.2 ms
```

### Unit Tests Added

New tests in `virsh-sandbox/internal/libvirt/virsh_test.go`:

- `TestRenderDomainXML_WithCloudInitISO` - Verifies CDROM is included in XML
- `TestRenderDomainXML_WithoutCloudInitISO` - Verifies no CDROM when ISO is empty
- `TestCloudInitSeedForClone_UniqueInstanceID` - Verifies unique ISO paths per sandbox

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `IP_DISCOVERY_TIMEOUT` | `2m` | Maximum time to wait for VM to obtain IP address |

For environments with slow VM boot times (e.g., ARM64 under TCG emulation), consider increasing this value:

```bash
export IP_DISCOVERY_TIMEOUT=5m
```

### Base VM Requirements

For optimal cloning behavior, base VMs should:

1. **Have a cloud-init CDROM attached** with a valid NoCloud seed
2. **Use virtio network interfaces** (for the netplan match rule to work)
3. **Have cloud-init installed** in the guest OS

---

## Next Steps & Recommendations

### Short-Term Improvements

1. **Increase default IP discovery timeout for known slow environments**
   - Detect ARM64 under TCG and automatically extend timeout
   - Add configuration option per base VM for expected boot time

2. **Add IP polling endpoint**
   - Allow clients to create sandbox with `wait_for_ip: false`
   - Provide endpoint to check/poll for IP address separately
   - Reduces API timeout issues for slow-booting VMs

3. **Improve error messages**
   - When IP discovery times out, include diagnostic info:
     - VM state (running/paused/etc)
     - Network interface statistics (TX/RX packets)
     - Suggestion to check cloud-init logs

### Medium-Term Improvements

1. **Cloud-init status detection**
   - Use qemu-guest-agent to query cloud-init status inside VM
   - Detect if cloud-init is still running vs. failed vs. completed
   - Provide more accurate progress feedback to clients

2. **Network configuration options**
   - Allow specifying static IP for sandboxes
   - Support custom netplan configurations
   - Enable IPv6 DHCP option

3. **Base VM validation**
   - Add pre-flight check when registering base VMs
   - Verify cloud-init is installed and configured
   - Warn if expected boot time exceeds IP discovery timeout

### Long-Term Improvements

1. **Alternative network initialization methods**
   - Support cloud-init "ConfigDrive" in addition to NoCloud
   - Consider QEMU guest agent for network config injection
   - Explore using cloud-init's "clean" command instead of new ISO

2. **VM boot optimization**
   - Profile boot process to identify slow components
   - Consider using pre-booted VM snapshots for faster startup
   - Evaluate alternative emulation options (e.g., Rosetta on macOS)

3. **Monitoring & Observability**
   - Add metrics for VM boot time, IP discovery time
   - Track cloud-init success/failure rates
   - Alert on sandboxes that fail to obtain IP

---

## Troubleshooting Guide

### Sandbox Has No IP After Expected Boot Time

1. **Check VM is running:**
   ```bash
   virsh list --all | grep <sandbox-name>
   ```

2. **Check network interface statistics:**
   ```bash
   virsh domifstat <sandbox-name> <vnet-interface>
   ```
   - If `tx_packets` is 0, VM isn't sending any traffic
   - Likely cloud-init issue or VM not fully booted

3. **Check cloud-init ISO is attached:**
   ```bash
   virsh dumpxml <sandbox-name> | grep -A5 cdrom
   ```
   - Should show path to sandbox-specific ISO
   - Path should be `/var/lib/libvirt/images/jobs/<sandbox-name>/cloud-init.iso`

4. **Verify cloud-init seed content:**
   ```bash
   cat /var/lib/libvirt/images/jobs/<sandbox-name>/meta-data
   cat /var/lib/libvirt/images/jobs/<sandbox-name>/user-data
   ```
   - `instance-id` should match sandbox name
   - `user-data` should contain network configuration

5. **Check DHCP server (dnsmasq) is running:**
   ```bash
   virsh net-dhcp-leases default
   ```

6. **Access VM console for debugging:**
   ```bash
   # Try serial console
   virsh console <sandbox-name>
   
   # Or get VNC display
   virsh vncdisplay <sandbox-name>
   ```

### Cloud-Init ISO Not Being Created

Check service logs for errors:
```bash
docker compose logs virsh-sandbox | grep -i cloud-init
```

Common issues:
- Missing `genisoimage` or `cloud-localds` tools in container
- Permission issues writing to job directory
- Source VM doesn't have cloud-init CDROM (check `virsh domblklist`)

---

## References

- [Cloud-Init NoCloud Data Source](https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html)
- [Netplan Configuration](https://netplan.readthedocs.io/)
- [Libvirt Domain XML Format](https://libvirt.org/formatdomain.html)
- [QEMU Disk Images](https://qemu.readthedocs.io/en/latest/system/images.html)