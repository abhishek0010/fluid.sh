package provider

import (
	"context"
	"time"
)

// Manager defines the VM orchestration operations supported by all providers.
// Implementations exist for libvirt/KVM (via virsh) and Proxmox VE (via REST API).
type Manager interface {
	// CloneVM creates a VM from a golden base image and defines it.
	// For libvirt: baseImage is a filename in base_image_dir; the image is copied and a new domain defined.
	// For Proxmox: templates are VMs, so this delegates to CloneFromVM (baseImage = template name).
	// Remote libvirt hosts do not support CloneVM; use CloneFromVM for provider-agnostic cloning.
	// cpu and memoryMB are the VM shape. network is provider-specific (e.g., libvirt network name,
	// Proxmox bridge name).
	CloneVM(ctx context.Context, baseImage, newVMName string, cpu, memoryMB int, network string) (VMRef, error)

	// CloneFromVM creates a clone from an existing VM by name.
	// Works uniformly across all providers: resolves the source VM, copies/clones its disk,
	// and creates a new VM with the specified shape. Prefer this over CloneVM for
	// provider-agnostic code.
	CloneFromVM(ctx context.Context, sourceVMName, newVMName string, cpu, memoryMB int, network string) (VMRef, error)

	// InjectSSHKey injects an SSH public key for a user into the VM before boot.
	// The mechanism is provider-specific (virt-customize, cloud-init, Proxmox sshkeys param, etc.).
	InjectSSHKey(ctx context.Context, sandboxName, username, publicKey string) error

	// StartVM boots a defined VM.
	StartVM(ctx context.Context, vmName string) error

	// StopVM gracefully shuts down a VM, or forces if force is true.
	StopVM(ctx context.Context, vmName string, force bool) error

	// DestroyVM removes the VM and its storage.
	// If the VM is running, it will be stopped first.
	DestroyVM(ctx context.Context, vmName string) error

	// CreateSnapshot creates a snapshot with the given name.
	// If external is true, attempts a disk-only external snapshot (libvirt-specific; Proxmox ignores this).
	CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (SnapshotRef, error)

	// DiffSnapshot prepares a plan to compare two snapshots' filesystems.
	// The returned plan includes advice or prepared mounts where possible.
	DiffSnapshot(ctx context.Context, vmName, fromSnapshot, toSnapshot string) (*FSComparePlan, error)

	// GetIPAddress attempts to fetch the VM's primary IP.
	// Returns the IP address and MAC address of the VM's primary interface.
	GetIPAddress(ctx context.Context, vmName string, timeout time.Duration) (ip string, mac string, err error)

	// GetVMState returns the current state of a VM.
	GetVMState(ctx context.Context, vmName string) (VMState, error)

	// ValidateSourceVM performs pre-flight checks on a source VM before cloning.
	// Returns a ValidationResult with warnings and errors about the VM's readiness.
	ValidateSourceVM(ctx context.Context, vmName string) (*VMValidationResult, error)

	// CheckHostResources validates that the host has sufficient resources for a new sandbox.
	// Returns a ResourceCheckResult with available resources and any warnings.
	CheckHostResources(ctx context.Context, requiredCPUs, requiredMemoryMB int) (*ResourceCheckResult, error)
}
