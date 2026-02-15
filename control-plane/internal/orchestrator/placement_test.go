package orchestrator

import (
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/control-plane/internal/registry"
	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

type mockStream struct{}

func (m *mockStream) Send(msg *fluidv1.ControlMessage) error { return nil }

// registerHost is a helper that registers a host and sets its registration.
// It directly manipulates LastHeartbeat via the registry's SetRegistration (which
// updates it to time.Now()). For stale heartbeats, the caller must update the
// host after this call.
func registerHost(t *testing.T, reg *registry.Registry, hostID, hostname string, registration *fluidv1.HostRegistration) {
	t.Helper()
	if err := reg.Register(hostID, hostname, &mockStream{}); err != nil {
		t.Fatalf("Register(%q) failed: %v", hostID, err)
	}
	if registration != nil {
		reg.SetRegistration(hostID, registration)
	}
}

// makeStale sets a host's heartbeat to more than 90s ago by re-registering it
// and then overwriting the registration (which sets heartbeat to now), and then
// waiting. Instead, we use a direct approach: register, set registration, then
// manipulate the host through the registry's exported GetHost.
// Since ConnectedHost.LastHeartbeat is an exported field, we can set it directly.
func makeStale(t *testing.T, reg *registry.Registry, hostID string) {
	t.Helper()
	h, ok := reg.GetHost(hostID)
	if !ok {
		t.Fatalf("host %q not found for makeStale", hostID)
	}
	h.LastHeartbeat = time.Now().Add(-2 * time.Minute)
}

func TestSelectHost_NoHosts(t *testing.T) {
	reg := registry.New()
	_, err := SelectHost(reg, "ubuntu-22.04")
	if err == nil {
		t.Fatal("expected error when no hosts connected")
	}
}

func TestSelectHost_NoImage(t *testing.T) {
	reg := registry.New()
	registerHost(t, reg, "host-1", "node-a", &fluidv1.HostRegistration{
		BaseImages:        []string{"debian-12"},
		AvailableCpus:     4,
		AvailableMemoryMb: 4096,
	})

	_, err := SelectHost(reg, "ubuntu-22.04")
	if err == nil {
		t.Fatal("expected error when host does not have requested image")
	}
}

func TestSelectHost_InsufficientResources(t *testing.T) {
	reg := registry.New()

	// Host has the image but 0 CPUs.
	registerHost(t, reg, "host-nocpu", "node-a", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     0,
		AvailableMemoryMb: 4096,
	})

	// Host has the image but insufficient memory.
	registerHost(t, reg, "host-nomem", "node-b", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 256,
	})

	_, err := SelectHost(reg, "ubuntu-22.04")
	if err == nil {
		t.Fatal("expected error when hosts have insufficient resources")
	}
}

func TestSelectHost_UnhealthyHost(t *testing.T) {
	reg := registry.New()
	registerHost(t, reg, "host-1", "node-a", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 4096,
	})
	makeStale(t, reg, "host-1")

	_, err := SelectHost(reg, "ubuntu-22.04")
	if err == nil {
		t.Fatal("expected error when host heartbeat is stale")
	}
}

func TestSelectHost_LeastLoaded(t *testing.T) {
	reg := registry.New()

	registerHost(t, reg, "host-low", "node-a", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     2,
		AvailableMemoryMb: 1024,
	})

	registerHost(t, reg, "host-high", "node-b", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 8192,
	})

	h, err := SelectHost(reg, "ubuntu-22.04")
	if err != nil {
		t.Fatalf("SelectHost returned error: %v", err)
	}
	if h.HostID != "host-high" {
		t.Errorf("expected host-high (most memory), got %q", h.HostID)
	}
}

func TestSelectHost_MultipleQualifying(t *testing.T) {
	reg := registry.New()

	registerHost(t, reg, "host-a", "node-a", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 2048,
	})

	registerHost(t, reg, "host-b", "node-b", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 4096,
	})

	registerHost(t, reg, "host-c", "node-c", &fluidv1.HostRegistration{
		BaseImages:        []string{"ubuntu-22.04"},
		AvailableCpus:     4,
		AvailableMemoryMb: 3072,
	})

	h, err := SelectHost(reg, "ubuntu-22.04")
	if err != nil {
		t.Fatalf("SelectHost returned error: %v", err)
	}
	// Deterministic: host with most available memory wins.
	if h.HostID != "host-b" {
		t.Errorf("expected host-b (4096 MB), got %q", h.HostID)
	}
}

func TestSelectHostForSourceVM(t *testing.T) {
	reg := registry.New()
	registerHost(t, reg, "host-1", "node-a", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "my-source-vm", State: "running"},
			{Name: "other-vm", State: "shutoff"},
		},
	})

	h, err := SelectHostForSourceVM(reg, "my-source-vm")
	if err != nil {
		t.Fatalf("SelectHostForSourceVM returned error: %v", err)
	}
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
}

func TestSelectHostForSourceVM_NotFound(t *testing.T) {
	reg := registry.New()
	registerHost(t, reg, "host-1", "node-a", &fluidv1.HostRegistration{
		SourceVms: []*fluidv1.SourceVMInfo{
			{Name: "some-other-vm", State: "running"},
		},
	})

	_, err := SelectHostForSourceVM(reg, "nonexistent-vm")
	if err == nil {
		t.Fatal("expected error when source VM not found on any host")
	}
}
