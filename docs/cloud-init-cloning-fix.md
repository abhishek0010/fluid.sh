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

This section provides comprehensive debugging steps for diagnosing VM networking issues.

---

### Quick Diagnostic Commands

Run these first to get an overview of the system state:

```bash
# 1. Check DHCP leases
cat /var/lib/libvirt/dnsmasq/default.leases

# 2. Check network config
virsh net-dumpxml default | grep -A5 dhcp

# 3. List all VM MACs and IPs
for vm in $(virsh list --name); do
  echo "=== $vm ==="
  virsh domifaddr "$vm" --source lease
done

# 4. Check for duplicate MACs (causes IP conflicts)
virsh list --name | xargs -I{} virsh domiflist {} 2>/dev/null | grep -E "^[a-z]" | awk '{print $5}' | sort | uniq -d
```

---

### Pre-Flight Validation (fluid CLI)

Before creating sandboxes, use the built-in validation:

```bash
# Validate source VM and host resources
fluid validate <source-vm-name>

# Example output showing warnings
{
  "source_vm": "test-vm-1",
  "valid": true,
  "vm_state": "running",
  "has_network": true,
  "mac_address": "52:54:00:12:34:56",
  "warnings": [
    "Source VM is running but has no IP address assigned",
    "This may indicate cloud-init or DHCP issues - cloned sandboxes may also fail to get IPs"
  ]
}
```

---

### Source VM Has No IP Address

If `virsh domifaddr <source-vm> --source lease` returns empty results, investigate:

#### 1. Check if libvirt network is running

```bash
# List networks
virsh net-list --all

# Expected output:
#  Name      State    Autostart   Persistent
# --------------------------------------------
#  default   active   yes         yes

# If not active, start it:
virsh net-start default
virsh net-autostart default
```

#### 2. Verify DHCP is enabled on the network

```bash
virsh net-dumpxml default | grep -A10 '<dhcp>'

# Expected output should include:
# <dhcp>
#   <range start='192.168.122.2' end='192.168.122.254'/>
# </dhcp>
```

If DHCP is missing, edit the network:
```bash
virsh net-edit default
# Add inside <ip> block:
# <dhcp>
#   <range start='192.168.122.2' end='192.168.122.254'/>
# </dhcp>

# Restart network
virsh net-destroy default
virsh net-start default
```

#### 3. Check VM has a network interface

```bash
virsh domiflist <vm-name>

# Expected output:
#  Interface   Type     Source    Model    MAC
# -------------------------------------------------------------
#  vnet0       network  default   virtio   52:54:00:xx:xx:xx
```

If empty, the VM XML is missing network configuration.

#### 4. Check dnsmasq is running (provides DHCP)

```bash
# Check process
ps aux | grep dnsmasq

# Check dnsmasq logs
journalctl -u libvirtd | grep dnsmasq

# Or check syslog
grep dnsmasq /var/log/syslog | tail -20
```

#### 5. Verify cloud-init is installed in the VM

Access VM console and check:
```bash
virsh console <vm-name>

# Inside VM:
cloud-init --version
systemctl status cloud-init

# Check cloud-init data directory exists
ls -la /var/lib/cloud/
```

#### 6. Check cloud-init status inside VM

```bash
# Inside VM:
cloud-init status
# Should show: status: done

# If status shows error, check logs:
cat /var/log/cloud-init.log | grep -i error
cat /var/log/cloud-init-output.log
```

#### 7. Check network configuration inside VM

```bash
# Inside VM:
ip addr show
ip route show

# Check if interface is up but has no IP
# This indicates DHCP client issue

# Check DHCP client logs
journalctl -u systemd-networkd | tail -50
# or
journalctl -u NetworkManager | tail -50
```

---

### Sandbox Has No IP After Expected Boot Time

#### 1. Check VM is running

```bash
virsh list --all | grep <sandbox-name>

# State should be "running"
```

#### 2. Check network interface statistics

```bash
# Get interface name first
virsh domiflist <sandbox-name>

# Then check stats
virsh domifstat <sandbox-name> <vnet-interface>

# Example output:
# vnet15 rx_bytes 180
# vnet15 rx_packets 2
# vnet15 tx_bytes 0        # Zero = no outgoing traffic!
# vnet15 tx_packets 0
```

**Interpretation:**
- `tx_packets = 0`: VM isn't sending any traffic (cloud-init issue or VM not booted)
- `rx_packets > 0, tx_packets = 0`: VM receives broadcasts but doesn't respond
- Both non-zero: Network is working, check DHCP server

#### 3. Check cloud-init ISO is attached

```bash
virsh dumpxml <sandbox-name> | grep -A5 cdrom

# Should show:
# <disk type='file' device='cdrom'>
#   <source file='/var/lib/libvirt/images/sandboxes/<sandbox-name>/cloud-init.iso'/>
#   ...
# </disk>
```

#### 4. Verify cloud-init seed content

```bash
# Check meta-data (instance-id must be unique per sandbox)
cat /var/lib/libvirt/images/sandboxes/<sandbox-name>/meta-data

# Should show:
# instance-id: <sandbox-name>
# local-hostname: <sandbox-name>

# Check user-data (should have network config)
cat /var/lib/libvirt/images/sandboxes/<sandbox-name>/user-data

# Should contain:
# network:
#   version: 2
#   ethernets:
#     id0:
#       match:
#         driver: virtio*
#       dhcp4: true
```

#### 5. Check DHCP server has leases available

```bash
virsh net-dhcp-leases default

# Check lease file directly
cat /var/lib/libvirt/dnsmasq/default.leases
```

#### 6. Check for MAC address collision

```bash
# Get sandbox MAC
virsh domiflist <sandbox-name>

# Compare with source VM MAC
virsh domiflist <source-vm-name>

# They MUST be different! If same, the clone process failed to generate new MAC.
```

#### 7. Access VM console for debugging

```bash
# Serial console (if configured)
virsh console <sandbox-name>
# Press Enter, login with cloud-init credentials

# VNC display
virsh vncdisplay <sandbox-name>
# Connect with VNC viewer to localhost:<port>

# If neither works, check VM has console configured:
virsh dumpxml <sandbox-name> | grep -A3 '<console'
```

---

### Cloud-Init ISO Not Being Created

#### 1. Check for ISO creation tools

```bash
# At least one of these must be available:
which cloud-localds
which genisoimage
which mkisofs
```

Install if missing:
```bash
# Ubuntu/Debian
apt-get install cloud-image-utils  # provides cloud-localds
apt-get install genisoimage        # alternative

# RHEL/CentOS
yum install cloud-utils            # provides cloud-localds
yum install genisoimage            # alternative
```

#### 2. Check service logs

```bash
# For fluid CLI
# Check terminal output for warnings about cloud-init

# For docker deployment
docker compose logs virsh-sandbox | grep -i cloud-init

# For systemd service
journalctl -u fluid | grep -i cloud-init
```

#### 3. Check permissions on work directory

```bash
ls -la /var/lib/libvirt/images/sandboxes/

# Directory should be writable by the service user
# If permission denied, fix with:
chown -R libvirt-qemu:libvirt /var/lib/libvirt/images/sandboxes/
# or
chmod 775 /var/lib/libvirt/images/sandboxes/
```

#### 4. Check source VM has cloud-init CDROM

```bash
virsh domblklist <source-vm-name> --details

# Look for cdrom device - if missing, source VM doesn't use cloud-init
```

---

### Cloud-Init Runs But Network Fails

If cloud-init runs but network still fails, check inside the VM:

#### 1. Check cloud-init network config was applied

```bash
# Inside VM:
cat /etc/netplan/*.yaml
# or
cat /etc/network/interfaces
# or
nmcli device status
```

#### 2. Check for conflicting network configs

```bash
# Inside VM:
ls -la /etc/netplan/

# Multiple files can conflict - cloud-init creates 50-cloud-init.yaml
# Other files (00-installer-config.yaml) may override it
```

#### 3. Force cloud-init to re-run (for debugging)

```bash
# Inside VM:
sudo cloud-init clean --logs
sudo cloud-init init --local
sudo cloud-init init
sudo cloud-init modules --mode=config
sudo cloud-init modules --mode=final

# Check status
cloud-init status --long
```

#### 4. Check instance-id matches expectation

```bash
# Inside VM:
cat /var/lib/cloud/data/instance-id

# This should match the sandbox name
# If it matches the source VM name, cloud-init didn't re-run
```

---

### MAC Address Issues

#### 1. Check MAC was generated correctly

```bash
# Sandbox MAC should start with 52:54:00 (QEMU prefix)
virsh domiflist <sandbox-name>

# Verify it's different from source VM
virsh domiflist <source-vm-name>
```

#### 2. Check for MAC collision across all VMs

```bash
# List all MACs
for vm in $(virsh list --all --name); do
  echo -n "$vm: "
  virsh domiflist "$vm" 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1
done | sort -t: -k2

# Check for duplicates
virsh list --all --name | xargs -I{} virsh domiflist {} 2>/dev/null | \
  grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | sort | uniq -d
```

#### 3. Manually fix MAC if needed

```bash
# Stop VM
virsh destroy <sandbox-name>

# Edit XML
virsh edit <sandbox-name>
# Find <mac address='...'> and change to unique value

# Start VM
virsh start <sandbox-name>
```

---

### Host Resource Issues

#### 1. Check available memory

```bash
# System memory
free -h

# Libvirt view
virsh nodememstats

# Memory used by VMs
virsh list --all --name | xargs -I{} virsh dominfo {} 2>/dev/null | grep -E "^(Name|Max memory|Used memory)"
```

#### 2. Check disk space

```bash
df -h /var/lib/libvirt/images/

# Check individual sandbox sizes
du -sh /var/lib/libvirt/images/sandboxes/*
```

#### 3. Check for resource exhaustion

```bash
# Too many VMs?
virsh list --all | wc -l

# CPU overcommit?
virsh nodeinfo | grep "CPU(s)"
virsh list --all --name | xargs -I{} virsh vcpucount {} 2>/dev/null | grep current | awk '{sum+=$2} END {print "Total vCPUs: " sum}'
```

---

### Performance Issues (Slow Boot)

#### 1. Check if using KVM acceleration

```bash
# Inside VM or from host:
virsh dumpxml <vm-name> | grep -i kvm

# Check host supports KVM
ls -la /dev/kvm
# If missing, VMs run in slow TCG emulation mode
```

#### 2. Check VM architecture matches host

```bash
# Host architecture
uname -m

# VM architecture
virsh dumpxml <vm-name> | grep -i arch

# ARM64 VMs on x86 hosts use TCG emulation (very slow)
```

#### 3. Increase IP discovery timeout for slow VMs

```bash
# Set environment variable
export IP_DISCOVERY_TIMEOUT=5m

# Or in config file
# vm:
#   ip_discovery_timeout: 5m
```

---

### Debugging Checklist

Use this checklist when sandboxes fail to get IPs:

- [ ] Source VM exists and is defined in libvirt
- [ ] Source VM has network interface with MAC address
- [ ] Source VM (if running) has IP address
- [ ] Libvirt network is active (`virsh net-list`)
- [ ] DHCP is enabled on network (`virsh net-dumpxml default`)
- [ ] dnsmasq process is running
- [ ] Sandbox was created successfully
- [ ] Sandbox has unique MAC (different from source)
- [ ] Cloud-init ISO was created in sandbox directory
- [ ] Cloud-init ISO has unique instance-id
- [ ] Sandbox is in "running" state
- [ ] Sandbox network interface shows TX packets > 0
- [ ] No duplicate MACs across VMs
- [ ] Sufficient host memory available
- [ ] Sufficient disk space in work directory

---

### Getting Help

If issues persist after following this guide:

1. Collect diagnostic info:
   ```bash
   fluid validate <source-vm> > validation.json
   virsh dumpxml <sandbox-name> > sandbox.xml
   virsh net-dumpxml default > network.xml
   cat /var/lib/libvirt/images/sandboxes/<sandbox>/meta-data > meta-data.txt
   cat /var/lib/libvirt/images/sandboxes/<sandbox>/user-data > user-data.txt
   ```

2. Check cloud-init logs from inside VM if accessible

3. File an issue with the collected diagnostic files

---

## References

- [Cloud-Init NoCloud Data Source](https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html)
- [Netplan Configuration](https://netplan.readthedocs.io/)
- [Libvirt Domain XML Format](https://libvirt.org/formatdomain.html)
- [QEMU Disk Images](https://qemu.readthedocs.io/en/latest/system/images.html)
