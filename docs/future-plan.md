# Future Development Plan

This document outlines proposed features and architectural improvements for the `virsh-sandbox` project.

## Support for `cloud-init` User-Data on Sandbox Creation

### Problem

When cloning customer-provided VMs (e.g., CentOS, RHEL), the resulting sandbox often fails to acquire a network connection. This is because the base image's network configuration scripts (e.g., `/etc/sysconfig/network-scripts/ifcfg-eth0`) contain a hardcoded MAC address (`HWADDR`). Since a cloned VM gets a new, unique MAC address, the network configuration is not applied, and the interface is not brought up.

As we cannot modify the customer's base image, we need a way to fix the network configuration of the sandbox *after* it has been cloned but *before* it is used.

### Proposed Solution

Enhance the `virsh-sandbox` API to accept `cloud-init` `user-data` during sandbox creation. This allows the agent creating the sandbox to provide a generic network configuration that will override the faulty one in the base image.

This is the standard, "cloud native" way to handle per-instance customization and is more robust and portable than attempting to mount and modify the disk image manually.

### Implementation Details

1.  **Modify API Endpoint:** The `POST /v1/sandboxes` endpoint will be updated to accept an optional `user_data` field in its JSON request body.
    ```json
    {
      "source_vm_name": "centos-base-image",
      "agent_id": "my-agent",
      "user_data": "#cloud-config\nnetwork:\n  version: 1\n..."
    }
    ```

2.  **Update Service Layer:** The `vm.Service.CreateSandbox` function will be updated to accept the `user_data` string.

3.  **Enhance Libvirt Manager:** The `libvirt.Manager.CloneFromVM` function will receive the `user_data` and perform the following steps:
    *   If `user_data` is provided, create a temporary `cloud-init.iso` file containing the user-data and default meta-data.
    *   When defining the new cloned VM, add a CD-ROM device to the libvirt XML that points to this temporary ISO.
    *   Ensure the temporary ISO file is deleted after the VM has been successfully defined.

### Example: Fixing a CentOS Clone

An agent wanting to create a sandbox from a CentOS base image would provide the following `user_data` to correctly configure the primary network interface for DHCP:

```yaml
#cloud-config

# This user-data configures the network for a RHEL/CentOS based system.
# It will generate a new ifcfg-eth0 file without a HWADDR, ensuring
# the network is configured correctly on the cloned VM.

network:
  version: 1
  config:
    - type: physical
      name: eth0
      subnets:
        - type: dhcp
```
