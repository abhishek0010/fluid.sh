package registry

import (
	"sync"
	"testing"
	"time"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"
)

type mockStream struct{}

func (m *mockStream) Send(msg *fluidv1.ControlMessage) error { return nil }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := New()
	stream := &mockStream{}

	err := r.Register("host-1", "node-a", stream)
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	h, ok := r.GetHost("host-1")
	if !ok {
		t.Fatal("expected host to be found")
	}
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
	if h.Hostname != "node-a" {
		t.Errorf("Hostname = %q, want %q", h.Hostname, "node-a")
	}
	if h.Stream != stream {
		t.Error("Stream does not match registered stream")
	}
	if time.Since(h.LastHeartbeat) > 2*time.Second {
		t.Error("LastHeartbeat should be recent")
	}
}

func TestRegistry_Register_EmptyID(t *testing.T) {
	r := New()
	err := r.Register("", "node-a", &mockStream{})
	if err == nil {
		t.Fatal("expected error for empty host ID")
	}
}

func TestRegistry_Register_NilStream(t *testing.T) {
	r := New()
	err := r.Register("host-1", "node-a", nil)
	if err == nil {
		t.Fatal("expected error for nil stream")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := New()
	_ = r.Register("host-1", "node-a", &mockStream{})

	r.Unregister("host-1")

	_, ok := r.GetHost("host-1")
	if ok {
		t.Fatal("expected host to be removed after Unregister")
	}
}

func TestRegistry_ListConnected(t *testing.T) {
	r := New()
	_ = r.Register("host-1", "node-a", &mockStream{})
	_ = r.Register("host-2", "node-b", &mockStream{})

	hosts := r.ListConnected()
	if len(hosts) != 2 {
		t.Fatalf("ListConnected returned %d hosts, want 2", len(hosts))
	}

	ids := map[string]bool{}
	for _, h := range hosts {
		ids[h.HostID] = true
	}
	if !ids["host-1"] || !ids["host-2"] {
		t.Errorf("expected host-1 and host-2, got %v", ids)
	}
}

func TestRegistry_SetRegistration(t *testing.T) {
	r := New()
	_ = r.Register("host-1", "node-a", &mockStream{})

	// Sleep briefly so we can detect heartbeat update.
	time.Sleep(10 * time.Millisecond)

	reg := &fluidv1.HostRegistration{
		BaseImages: []string{"ubuntu-22.04"},
		TotalCpus:  8,
	}
	r.SetRegistration("host-1", reg)

	h, ok := r.GetHost("host-1")
	if !ok {
		t.Fatal("host not found")
	}
	if h.Registration == nil {
		t.Fatal("Registration should not be nil after SetRegistration")
	}
	if len(h.Registration.GetBaseImages()) != 1 || h.Registration.GetBaseImages()[0] != "ubuntu-22.04" {
		t.Errorf("unexpected BaseImages: %v", h.Registration.GetBaseImages())
	}
	// Heartbeat should have been refreshed.
	if time.Since(h.LastHeartbeat) > 2*time.Second {
		t.Error("LastHeartbeat should have been updated by SetRegistration")
	}
}

func TestRegistry_UpdateHeartbeat(t *testing.T) {
	r := New()
	_ = r.Register("host-1", "node-a", &mockStream{})

	h, _ := r.GetHost("host-1")
	firstBeat := h.LastHeartbeat

	time.Sleep(10 * time.Millisecond)
	r.UpdateHeartbeat("host-1")

	h, _ = r.GetHost("host-1")
	if !h.LastHeartbeat.After(firstBeat) {
		t.Error("LastHeartbeat should be later after UpdateHeartbeat")
	}
}

func TestRegistry_SelectHostForImage(t *testing.T) {
	r := New()
	_ = r.Register("host-1", "node-a", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages: []string{"ubuntu-22.04", "debian-12"},
	})

	h, err := r.SelectHostForImage("debian-12")
	if err != nil {
		t.Fatalf("SelectHostForImage returned error: %v", err)
	}
	if h.HostID != "host-1" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-1")
	}
}

func TestRegistry_SelectHostForImage_NotFound(t *testing.T) {
	r := New()
	_ = r.Register("host-1", "node-a", &mockStream{})
	r.SetRegistration("host-1", &fluidv1.HostRegistration{
		BaseImages: []string{"ubuntu-22.04"},
	})

	_, err := r.SelectHostForImage("centos-9")
	if err == nil {
		t.Fatal("expected error when image not found")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	const goroutines = 50

	// Half the goroutines register, half read.
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			if n%3 == 0 {
				_ = r.Register("host-concurrent", "node", &mockStream{})
			} else if n%3 == 1 {
				r.GetHost("host-concurrent")
			} else {
				r.ListConnected()
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or data-race. Just verify registry is still functional.
	_ = r.Register("host-final", "node-final", &mockStream{})
	h, ok := r.GetHost("host-final")
	if !ok {
		t.Fatal("expected host-final after concurrent access")
	}
	if h.HostID != "host-final" {
		t.Errorf("HostID = %q, want %q", h.HostID, "host-final")
	}
}
