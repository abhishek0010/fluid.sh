# VirtualBox Support Plan

This plan details the steps to add VirtualBox support to `virsh-sandbox` alongside the existing KVM/libvirt implementation.

## 1. Abstract Hypervisor Interface

The current `VirshManager` is tightly coupled to KVM/libvirt commands and XML generation. We need to extract a generic `HypervisorManager` interface that both KVM and VirtualBox implementations will satisfy.

**Goal:** Allow switching between `kvm` and `vbox` backends via configuration.

### Steps:
1.  **Refactor Interface:**
    - Review `virsh-sandbox/internal/libvirt/virsh.go`'s `Manager` interface. It's already an interface, but the implementation (`VirshManager`) is specific.
    - Move `Manager` interface to a shared package if necessary, or keep it in `libvirt` (renamed to `hypervisor` or `provider`) if it's generic enough. Currently it's in `internal/libvirt`, which implies KVM.
    - Create a new package `internal/hypervisor` to define the common interface.
    - Refactor `internal/libvirt` to implement this interface as `KVMManager` (or keep as `VirshManager`).

2.  **Define Interface Methods:**
    - `CloneVM(ctx, baseImage, name, cpu, mem, network)`
    - `StartVM(ctx, name)`
    - `StopVM(ctx, name, force)`
    - `DestroyVM(ctx, name)`
    - `GetIPAddress(ctx, name)`
    - `InjectSSHKey(ctx, name, user, key)`
    - `CreateSnapshot(ctx, name, snapName)`
    - `DiffSnapshot(ctx, name, from, to)` - *Note: might be tricky for VBox*

## 2. Implement VirtualBox Manager

Create a new implementation in `internal/virtualbox` that uses `VBoxManage` CLI commands.

### Key Mappings:
-   **CloneVM:** `VBoxManage clonevm <base> --name <new> --register` + `VBoxManage modifyvm` for specs.
-   **StartVM:** `VBoxManage startvm <name> --type headless`
-   **StopVM:** `VBoxManage controlvm <name> acpipowerbutton` (graceful) or `poweroff` (force).
-   **DestroyVM:** `VBoxManage unregistervm <name> --delete`
-   **GetIPAddress:**
    -   *NAT Mode:* VBox doesn't easily expose guest IP. We might need `VBoxManage guestproperty get` (requires Guest Additions) or a port forwarding strategy + SSH check.
    -   *Bridged Mode:* ARP table lookup (similar to `socket_vmnet` logic).
    -   *Host-Only:* Parse `vboxnet` DHCP leases (if available).
-   **InjectSSHKey:**
    -   *Cloud-Init:* Attach a config-drive ISO (similar to KVM implementation).
    -   *Guest Control:* `VBoxManage guestcontrol` (requires Guest Additions/credentials). *Cloud-init ISO is preferred for consistency.*

## 3. Disk Image Management

VirtualBox prefers VDI or VMDK. QCOW2 is supported but might be slower or read-only in some contexts.

-   **Base Images:** Users should provide VDI base images for VirtualBox.
-   **Overlay/Clones:** `VBoxManage snapshot` or "Linked Clones" use differential disks naturally.
-   **Conversion:** We might need `qemu-img convert` to create VDIs from QCOW2s if we want to share base images (complex, maybe out of scope for V1). *Decision: Assume native VDI base images for VBox mode.*

## 4. Configuration & Factory

Update `cmd/api/main.go` and `Config` to support selecting the hypervisor.

-   **Env Vars:**
    -   `HYPERVISOR`: `kvm` (default) or `vbox`.
    -   `VBOX_MANAGE_PATH`: Path to binary (default: lookup in PATH).
    -   `BASE_IMAGE_DIR`: Should point to a dir with VDIs for VBox.

-   **Factory Logic:**
    -   If `HYPERVISOR=vbox`, instantiate `VirtualBoxManager`.
    -   Else, instantiate `VirshManager`.

## 5. Networking

VirtualBox networking differs from Libvirt.

-   **Default:** NAT is easiest but isolates the VM.
-   **Host-Only:** Good for local comms, requires setting up a `vboxnet0` adapter.
-   **Bridged:** Good for LAN access, but requires specifying a physical interface.

*Strategy:* Default to **NAT** with **Port Forwarding** for SSH (host random port -> guest 22). Or use **Host-Only** networking if we want direct IP access like KVM.
*Recommendation:* **Host-Only** is closest to the `virsh` "default" network experience (VM gets an IP reachable by host).

## 6. Snapshot & Diff (Advanced)

-   **CreateSnapshot:** `VBoxManage snapshot <vm> take <name>`
-   **Diff:** `VBoxManage clonehd` or mounting VDIs.
    -   *Challenge:* Mounting VDI on host requires `nbd` + `qemu-nbd` (works with VDI!) or `vbox-img`.
    -   We can likely reuse the `qemu-nbd` logic if we compile QEMU with VDI support (standard).

## 7. Execution Plan

1.  **Refactor:** Extract `Manager` interface to `internal/hypervisor/manager.go`.
2.  **Scaffold:** Create `internal/virtualbox/manager.go` struct.
3.  **Implement Basic Lifecycle:** `Start`, `Stop`, `Destroy`.
4.  **Implement Cloning:** `CloneVM` using `VBoxManage`.
5.  **Implement Networking/IP:** Decide on Host-Only vs NAT. Implement IP retrieval.
6.  **Wire Up:** Update `main.go` to use the new flags/env vars.
7.  **Test:** Verify with a simple Alpine VDI.

