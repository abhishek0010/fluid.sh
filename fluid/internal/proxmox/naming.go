package proxmox

import (
	"context"
	"fmt"
	"sync"
)

// VMResolver resolves VM names to VMIDs and vice versa.
// It caches the VM list and can be refreshed on demand.
type VMResolver struct {
	client *Client
	mu     sync.RWMutex
	byName map[string]int
	byID   map[int]string
}

// NewVMResolver creates a new VMResolver backed by the given client.
func NewVMResolver(client *Client) *VMResolver {
	return &VMResolver{
		client: client,
		byName: make(map[string]int),
		byID:   make(map[int]string),
	}
}

// Refresh reloads the VM list from Proxmox and rebuilds the cache.
func (r *VMResolver) Refresh(ctx context.Context) error {
	vms, err := r.client.ListVMs(ctx)
	if err != nil {
		return fmt.Errorf("refresh VM list: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byName = make(map[string]int, len(vms))
	r.byID = make(map[int]string, len(vms))
	for _, vm := range vms {
		r.byName[vm.Name] = vm.VMID
		r.byID[vm.VMID] = vm.Name
	}
	return nil
}

// ResolveVMID returns the VMID for a given VM name.
// If the name is not in the cache, it refreshes first.
func (r *VMResolver) ResolveVMID(ctx context.Context, name string) (int, error) {
	r.mu.RLock()
	vmid, ok := r.byName[name]
	r.mu.RUnlock()
	if ok {
		return vmid, nil
	}

	// Cache miss - refresh and retry
	if err := r.Refresh(ctx); err != nil {
		return 0, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	vmid, ok = r.byName[name]
	if !ok {
		return 0, fmt.Errorf("VM %q not found", name)
	}
	return vmid, nil
}

// ResolveName returns the name for a given VMID.
func (r *VMResolver) ResolveName(ctx context.Context, vmid int) (string, error) {
	r.mu.RLock()
	name, ok := r.byID[vmid]
	r.mu.RUnlock()
	if ok {
		return name, nil
	}

	if err := r.Refresh(ctx); err != nil {
		return "", err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok = r.byID[vmid]
	if !ok {
		return "", fmt.Errorf("VMID %d not found", vmid)
	}
	return name, nil
}

// ListAll returns all cached VM entries. Refreshes if cache is empty.
func (r *VMResolver) ListAll(ctx context.Context) ([]VMListEntry, error) {
	r.mu.RLock()
	empty := len(r.byName) == 0
	r.mu.RUnlock()

	if empty {
		if err := r.Refresh(ctx); err != nil {
			return nil, err
		}
	}

	return r.client.ListVMs(ctx)
}
