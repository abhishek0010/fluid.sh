package proxmox

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/provider"
)

// ProxmoxManager implements provider.Manager for Proxmox VE.
type ProxmoxManager struct {
	client   *Client
	cfg      Config
	resolver *VMResolver
	logger   *slog.Logger
	vmidMu   sync.Mutex
}

// NewProxmoxManager creates a new Proxmox provider manager.
func NewProxmoxManager(cfg Config, logger *slog.Logger) (*ProxmoxManager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid proxmox config: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}

	client := NewClient(cfg, logger)
	return &ProxmoxManager{
		client:   client,
		cfg:      cfg,
		resolver: NewVMResolver(client),
		logger:   logger,
	}, nil
}

// CloneVM clones a VM from a base image. On Proxmox, base images are templates
// (which are themselves VMs), so CloneVM and CloneFromVM are semantically identical.
// This delegates to CloneFromVM, treating baseImage as a source VM/template name.
func (m *ProxmoxManager) CloneVM(ctx context.Context, baseImage, newVMName string, cpu, memoryMB int, network string) (provider.VMRef, error) {
	return m.CloneFromVM(ctx, baseImage, newVMName, cpu, memoryMB, network)
}

// CloneFromVM creates a clone of an existing VM.
// It resolves the source VM name to a VMID, allocates a new VMID, and clones.
// The network parameter sets the Proxmox bridge via the net0 config param
// (format: "virtio,bridge=<network>"). If network is empty, falls back to
// Config.Bridge. If both are empty, the template's network config is preserved.
func (m *ProxmoxManager) CloneFromVM(ctx context.Context, sourceVMName, newVMName string, cpu, memoryMB int, network string) (provider.VMRef, error) {
	sourceVMID, err := m.resolver.ResolveVMID(ctx, sourceVMName)
	if err != nil {
		return provider.VMRef{}, fmt.Errorf("resolve source VM %q: %w", sourceVMName, err)
	}

	// Lock to serialize VMID allocation + clone so concurrent callers
	// cannot receive the same VMID before Proxmox reserves it.
	m.vmidMu.Lock()
	newVMID, err := m.client.NextVMID(ctx, m.cfg.VMIDStart, m.cfg.VMIDEnd)
	if err != nil {
		m.vmidMu.Unlock()
		return provider.VMRef{}, fmt.Errorf("allocate VMID: %w", err)
	}

	full := m.cfg.CloneMode == "full"
	m.logger.Info("cloning VM",
		"source", sourceVMName,
		"source_vmid", sourceVMID,
		"new_name", newVMName,
		"new_vmid", newVMID,
		"full_clone", full,
	)

	upid, err := m.client.CloneVM(ctx, sourceVMID, newVMID, newVMName, full)
	m.vmidMu.Unlock()
	if err != nil {
		return provider.VMRef{}, fmt.Errorf("clone VM: %w", err)
	}

	if err := m.client.WaitForTask(ctx, upid); err != nil {
		return provider.VMRef{}, fmt.Errorf("wait for clone: %w", err)
	}

	// Configure the clone with requested CPU/memory/network
	if cpu > 0 || memoryMB > 0 || network != "" || m.cfg.Bridge != "" {
		params := url.Values{}
		if cpu > 0 {
			params.Set("cores", fmt.Sprintf("%d", cpu))
		}
		if memoryMB > 0 {
			params.Set("memory", fmt.Sprintf("%d", memoryMB))
		}
		if network != "" {
			params.Set("net0", fmt.Sprintf("virtio,bridge=%s", network))
		} else if m.cfg.Bridge != "" {
			params.Set("net0", fmt.Sprintf("virtio,bridge=%s", m.cfg.Bridge))
		}
		if err := m.client.SetVMConfig(ctx, newVMID, params); err != nil {
			return provider.VMRef{}, fmt.Errorf("set CPU/memory on clone %d: %w", newVMID, err)
		}
	}

	// Refresh resolver cache to include the new VM
	_ = m.resolver.Refresh(ctx)

	return provider.VMRef{
		Name: newVMName,
		UUID: fmt.Sprintf("%d", newVMID),
	}, nil
}

// InjectSSHKey sets SSH keys on a VM via cloud-init configuration.
func (m *ProxmoxManager) InjectSSHKey(ctx context.Context, sandboxName, username, publicKey string) error {
	vmid, err := m.resolver.ResolveVMID(ctx, sandboxName)
	if err != nil {
		return fmt.Errorf("resolve VM %q: %w", sandboxName, err)
	}

	params := url.Values{}
	if username != "" {
		params.Set("ciuser", username)
	}
	// Proxmox requires URL-encoded SSH keys
	params.Set("sshkeys", url.QueryEscape(strings.TrimSpace(publicKey)))

	return m.client.SetVMConfig(ctx, vmid, params)
}

// StartVM boots a VM.
func (m *ProxmoxManager) StartVM(ctx context.Context, vmName string) error {
	vmid, err := m.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return fmt.Errorf("resolve VM %q: %w", vmName, err)
	}

	upid, err := m.client.StartVM(ctx, vmid)
	if err != nil {
		return fmt.Errorf("start VM: %w", err)
	}

	return m.client.WaitForTask(ctx, upid)
}

// StopVM gracefully shuts down or force-stops a VM.
func (m *ProxmoxManager) StopVM(ctx context.Context, vmName string, force bool) error {
	vmid, err := m.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return fmt.Errorf("resolve VM %q: %w", vmName, err)
	}

	var upid string
	if force {
		upid, err = m.client.StopVM(ctx, vmid)
	} else {
		upid, err = m.client.ShutdownVM(ctx, vmid)
	}
	if err != nil {
		return fmt.Errorf("stop VM: %w", err)
	}

	return m.client.WaitForTask(ctx, upid)
}

// DestroyVM stops (if running) and deletes a VM and all its resources.
func (m *ProxmoxManager) DestroyVM(ctx context.Context, vmName string) error {
	vmid, err := m.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return fmt.Errorf("resolve VM %q: %w", vmName, err)
	}

	// Check if running, stop first
	status, err := m.client.GetVMStatus(ctx, vmid)
	if err != nil {
		return fmt.Errorf("get VM status: %w", err)
	}
	if status.Status == "running" {
		m.logger.Info("stopping running VM before destroy", "vm", vmName, "vmid", vmid)
		upid, err := m.client.StopVM(ctx, vmid)
		if err != nil {
			return fmt.Errorf("stop VM before destroy: %w", err)
		}
		if err := m.client.WaitForTask(ctx, upid); err != nil {
			return fmt.Errorf("wait for stop: %w", err)
		}
	}

	upid, err := m.client.DeleteVM(ctx, vmid)
	if err != nil {
		return fmt.Errorf("delete VM: %w", err)
	}

	if err := m.client.WaitForTask(ctx, upid); err != nil {
		return fmt.Errorf("wait for delete: %w", err)
	}

	// Refresh resolver cache
	_ = m.resolver.Refresh(ctx)

	return nil
}

// CreateSnapshot creates a snapshot of a VM.
// The external parameter is ignored for Proxmox.
func (m *ProxmoxManager) CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (provider.SnapshotRef, error) {
	vmid, err := m.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return provider.SnapshotRef{}, fmt.Errorf("resolve VM %q: %w", vmName, err)
	}

	upid, err := m.client.CreateSnapshot(ctx, vmid, snapshotName, "")
	if err != nil {
		return provider.SnapshotRef{}, fmt.Errorf("create snapshot: %w", err)
	}

	if err := m.client.WaitForTask(ctx, upid); err != nil {
		return provider.SnapshotRef{}, fmt.Errorf("wait for snapshot: %w", err)
	}

	return provider.SnapshotRef{
		Name: snapshotName,
		Kind: "INTERNAL",
		Ref:  fmt.Sprintf("proxmox:%d:%s", vmid, snapshotName),
	}, nil
}

// DiffSnapshot returns a plan describing the snapshots.
// Proxmox does not support native filesystem diff, so this returns notes.
func (m *ProxmoxManager) DiffSnapshot(ctx context.Context, vmName, fromSnapshot, toSnapshot string) (*provider.FSComparePlan, error) {
	return &provider.FSComparePlan{
		VMName:       vmName,
		FromSnapshot: fromSnapshot,
		ToSnapshot:   toSnapshot,
		Notes: []string{
			"Proxmox does not support native snapshot filesystem diff.",
			"To compare changes, mount snapshots manually or use QEMU guest agent.",
		},
	}, nil
}

// GetIPAddress retrieves the VM's primary IPv4 address via the QEMU guest agent.
// Polls until an IP is found or the timeout expires.
// Filtering rules: skips loopback interfaces by name ("lo"), loopback IPs
// (127.0.0.0/8 via net.IP.IsLoopback), link-local IPs (169.254.0.0/16 via
// net.IP.IsLinkLocalUnicast), unparseable IPs, and IPv6 addresses.
// Returns the first valid IPv4 address found.
func (m *ProxmoxManager) GetIPAddress(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
	vmid, err := m.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return "", "", fmt.Errorf("resolve VM %q: %w", vmName, err)
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		ifaces, err := m.client.GetGuestAgentInterfaces(ctx, vmid)
		if err == nil {
			for _, iface := range ifaces {
				// Skip loopback
				if iface.Name == "lo" {
					continue
				}
				for _, addr := range iface.IPAddresses {
					if addr.IPAddressType == "ipv4" {
						ip := net.ParseIP(addr.IPAddress)
						if ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
							return addr.IPAddress, iface.HardwareAddress, nil
						}
					}
				}
			}
		}

		if time.Now().After(deadline) {
			return "", "", fmt.Errorf("timeout waiting for IP address of VM %q", vmName)
		}

		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-ticker.C:
		}
	}
}

// GetVMState returns the current state of a VM as a provider.VMState.
func (m *ProxmoxManager) GetVMState(ctx context.Context, vmName string) (provider.VMState, error) {
	vmid, err := m.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		return provider.VMStateUnknown, fmt.Errorf("resolve VM %q: %w", vmName, err)
	}

	status, err := m.client.GetVMStatus(ctx, vmid)
	if err != nil {
		return provider.VMStateUnknown, fmt.Errorf("get VM status: %w", err)
	}

	switch status.Status {
	case "running":
		return provider.VMStateRunning, nil
	case "stopped":
		return provider.VMStateShutOff, nil
	case "paused":
		return provider.VMStatePaused, nil
	default:
		return provider.VMStateUnknown, nil
	}
}

// ValidateSourceVM checks that a source VM exists and is suitable for cloning.
func (m *ProxmoxManager) ValidateSourceVM(ctx context.Context, vmName string) (*provider.VMValidationResult, error) {
	result := &provider.VMValidationResult{
		VMName: vmName,
		Valid:  true,
	}

	vmid, err := m.resolver.ResolveVMID(ctx, vmName)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("VM %q not found: %v", vmName, err))
		return result, nil
	}

	status, err := m.client.GetVMStatus(ctx, vmid)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("failed to get VM status: %v", err))
		return result, nil
	}

	switch status.Status {
	case "running":
		result.State = provider.VMStateRunning
	case "stopped":
		result.State = provider.VMStateShutOff
	default:
		result.State = provider.VMStateUnknown
	}

	// Check VM config for network and guest agent
	vmCfg, err := m.client.GetVMConfig(ctx, vmid)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("could not read VM config: %v", err))
	} else {
		if vmCfg.Net0 == "" {
			result.HasNetwork = false
			result.Warnings = append(result.Warnings, "VM has no network interface (net0)")
		} else {
			result.HasNetwork = true
		}

		if vmCfg.Agent == "" || vmCfg.Agent == "0" {
			result.Warnings = append(result.Warnings, "QEMU guest agent not enabled; IP discovery may not work")
		}
	}

	return result, nil
}

// CheckHostResources checks that the Proxmox node has sufficient resources.
func (m *ProxmoxManager) CheckHostResources(ctx context.Context, requiredCPUs, requiredMemoryMB int) (*provider.ResourceCheckResult, error) {
	nodeStatus, err := m.client.GetNodeStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get node status: %w", err)
	}

	totalMemMB := nodeStatus.Memory.Total / (1024 * 1024)
	freeMemMB := nodeStatus.Memory.Free / (1024 * 1024)
	freeDiskMB := nodeStatus.RootFS.Available / (1024 * 1024)

	result := &provider.ResourceCheckResult{
		Valid:             true,
		AvailableMemoryMB: freeMemMB,
		TotalMemoryMB:     totalMemMB,
		AvailableCPUs:     nodeStatus.MaxCPU,
		TotalCPUs:         nodeStatus.MaxCPU,
		AvailableDiskMB:   freeDiskMB,
		RequiredMemoryMB:  requiredMemoryMB,
		RequiredCPUs:      requiredCPUs,
	}

	// Memory check
	if int64(requiredMemoryMB) > freeMemMB {
		result.NeedsMemoryApproval = true
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("requested %d MB memory but only %d MB free", requiredMemoryMB, freeMemMB))
	}

	// CPU check - Proxmox allows overcommit but warn if high
	cpuUsagePct := nodeStatus.CPU * 100
	if cpuUsagePct > 80 {
		result.NeedsCPUApproval = true
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("node CPU usage is %.0f%%", cpuUsagePct))
	}

	return result, nil
}
