package orchestrator

import (
	"fmt"
	"time"

	"github.com/aspectrr/fluid.sh/control-plane/internal/registry"
)

// SelectHost picks the best connected host for a sandbox that needs the given
// base image. It filters by:
//  1. Host advertises the requested base image.
//  2. Host has sufficient available resources (at least 1 CPU and 512 MB memory).
//  3. Host is healthy (heartbeat received within 90s).
//
// Among qualifying hosts, the least-loaded host (most available memory) is selected.
func SelectHost(reg *registry.Registry, baseImage string) (*registry.ConnectedHost, error) {
	hosts := reg.ListConnected()
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no connected hosts")
	}

	now := time.Now()
	var best *registry.ConnectedHost

	for _, h := range hosts {
		if h.Registration == nil {
			continue
		}

		// Filter: host must have the requested base image.
		if !hostHasImage(h, baseImage) {
			continue
		}

		// Filter: host must have sufficient resources.
		if h.Registration.GetAvailableCpus() < 1 {
			continue
		}
		if h.Registration.GetAvailableMemoryMb() < 512 {
			continue
		}

		// Filter: host must be healthy (heartbeat within 90s).
		if now.Sub(h.LastHeartbeat) > 90*time.Second {
			continue
		}

		// Select least-loaded: prefer the host with the most available memory.
		if best == nil || h.Registration.GetAvailableMemoryMb() > best.Registration.GetAvailableMemoryMb() {
			best = h
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no healthy host with image %q and sufficient resources", baseImage)
	}

	return best, nil
}

// SelectHostForSourceVM picks a connected host that has the given source VM.
// Returns an error if no connected host advertises the VM.
func SelectHostForSourceVM(reg *registry.Registry, vmName string) (*registry.ConnectedHost, error) {
	hosts := reg.ListConnected()
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no connected hosts")
	}

	now := time.Now()

	for _, h := range hosts {
		if h.Registration == nil {
			continue
		}

		// Skip unhealthy hosts.
		if now.Sub(h.LastHeartbeat) > 90*time.Second {
			continue
		}

		for _, vm := range h.Registration.GetSourceVms() {
			if vm.GetName() == vmName {
				return h, nil
			}
		}
	}

	return nil, fmt.Errorf("no connected host has source VM %q", vmName)
}

// hostHasImage checks whether a host's registration includes the given base image.
func hostHasImage(h *registry.ConnectedHost, baseImage string) bool {
	for _, img := range h.Registration.GetBaseImages() {
		if img == baseImage {
			return true
		}
	}
	return false
}
