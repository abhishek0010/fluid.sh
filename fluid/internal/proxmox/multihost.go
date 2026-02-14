package proxmox

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aspectrr/fluid.sh/fluid/internal/provider"
)

// MultiNodeManager implements provider.MultiHostLister for Proxmox.
// Currently operates on a single node; can be extended for multi-node clusters.
type MultiNodeManager struct {
	client *Client
	cfg    Config
	logger *slog.Logger
}

// NewMultiNodeManager creates a new multi-node Proxmox manager.
func NewMultiNodeManager(cfg Config, logger *slog.Logger) *MultiNodeManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &MultiNodeManager{
		client: NewClient(cfg, logger),
		cfg:    cfg,
		logger: logger,
	}
}

// ListVMs returns all VMs on the configured Proxmox node.
func (m *MultiNodeManager) ListVMs(ctx context.Context) (*provider.MultiHostListResult, error) {
	vms, err := m.client.ListVMs(ctx)
	if err != nil {
		return &provider.MultiHostListResult{
			HostErrors: []provider.HostError{
				{
					HostName:    m.cfg.Node,
					HostAddress: m.cfg.Host,
					Error:       err.Error(),
				},
			},
		}, nil
	}

	result := &provider.MultiHostListResult{}
	for _, vm := range vms {
		state := vm.Status
		result.VMs = append(result.VMs, &provider.MultiHostVMInfo{
			Name:        vm.Name,
			UUID:        fmt.Sprintf("%d", vm.VMID),
			State:       state,
			Persistent:  true,
			HostName:    m.cfg.Node,
			HostAddress: m.cfg.Host,
		})
	}

	return result, nil
}

// FindHostForVM searches for a VM by name and returns its host info.
func (m *MultiNodeManager) FindHostForVM(ctx context.Context, vmName string) (string, string, error) {
	vms, err := m.client.ListVMs(ctx)
	if err != nil {
		return "", "", fmt.Errorf("list VMs: %w", err)
	}

	for _, vm := range vms {
		if vm.Name == vmName {
			return m.cfg.Node, m.cfg.Host, nil
		}
	}

	return "", "", fmt.Errorf("VM %q not found on node %s", vmName, m.cfg.Node)
}
