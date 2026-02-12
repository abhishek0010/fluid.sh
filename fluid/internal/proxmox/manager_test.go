package proxmox

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/provider"
)

// mockProxmoxAPI creates a mock Proxmox API server that handles common endpoints.
func mockProxmoxAPI(t *testing.T) (*ProxmoxManager, *httptest.Server) {
	t.Helper()

	mux := http.NewServeMux()

	// List VMs
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 100, Name: "ubuntu-template", Status: "stopped", Template: 1},
			{VMID: 101, Name: "sandbox-1", Status: "running"},
			{VMID: 102, Name: "sandbox-paused", Status: "paused"},
			{VMID: 103, Name: "no-net-vm", Status: "stopped"},
			{VMID: 104, Name: "no-agent-vm", Status: "stopped"},
		}
		_, _ = w.Write(envelope(vms))
	})

	// VM status endpoints
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/status/current", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 100, Name: "ubuntu-template", Status: "stopped"}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101/status/current", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 101, Name: "sandbox-1", Status: "running"}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/102/status/current", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 102, Name: "sandbox-paused", Status: "paused"}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/103/status/current", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 103, Name: "no-net-vm", Status: "stopped"}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/104/status/current", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 104, Name: "no-agent-vm", Status: "stopped"}))
	})

	// VM config endpoints
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write(envelope(VMConfig{
				Name:    "ubuntu-template",
				Memory:  4096,
				Cores:   2,
				Sockets: 1,
				Net0:    "virtio=AA:BB:CC:DD:EE:FF,bridge=vmbr0",
				Agent:   "1",
			}))
		} else {
			_, _ = w.Write(envelope(nil))
		}
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write(envelope(VMConfig{
				Name:   "sandbox-1",
				Memory: 2048,
				Cores:  1,
				Net0:   "virtio=11:22:33:44:55:66,bridge=vmbr0",
				Agent:  "1",
			}))
		} else {
			_, _ = w.Write(envelope(nil))
		}
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/103/config", func(w http.ResponseWriter, r *http.Request) {
		// VM with no network
		_, _ = w.Write(envelope(VMConfig{Name: "no-net-vm", Memory: 1024, Cores: 1, Agent: "1"}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/104/config", func(w http.ResponseWriter, r *http.Request) {
		// VM with no guest agent
		_, _ = w.Write(envelope(VMConfig{Name: "no-agent-vm", Memory: 1024, Cores: 1, Net0: "virtio=FF:FF:FF:FF:FF:FF,bridge=vmbr0"}))
	})

	// Clone
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/clone", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:00001234:clone"))
	})

	// Start/Stop/Shutdown
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/status/start", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:start:100"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101/status/start", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:start:101"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101/status/stop", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:stop:101"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/status/stop", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:stop:100"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101/status/shutdown", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:shutdown:101"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/status/shutdown", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:shutdown:100"))
	})

	// Snapshot
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101/snapshot", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:snapshot:101"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/snapshot", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:pve1:snapshot:100"))
	})

	// Guest agent
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101/agent/network-get-interfaces", func(w http.ResponseWriter, r *http.Request) {
		ifaces := struct {
			Result []NetworkInterface `json:"result"`
		}{
			Result: []NetworkInterface{
				{
					Name:            "lo",
					HardwareAddress: "00:00:00:00:00:00",
					IPAddresses:     []GuestIPAddress{{IPAddressType: "ipv4", IPAddress: "127.0.0.1", Prefix: 8}},
				},
				{
					Name:            "eth0",
					HardwareAddress: "AA:BB:CC:DD:EE:FF",
					IPAddresses:     []GuestIPAddress{{IPAddressType: "ipv4", IPAddress: "10.0.0.50", Prefix: 24}},
				},
			},
		}
		_, _ = w.Write(envelope(ifaces))
	})

	// Node status
	mux.HandleFunc("/api2/json/nodes/pve1/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(NodeStatus{
			CPU:    0.15,
			MaxCPU: 16,
			Memory: MemoryStatus{
				Total: 64 * 1024 * 1024 * 1024,
				Used:  16 * 1024 * 1024 * 1024,
				Free:  48 * 1024 * 1024 * 1024,
			},
			RootFS: DiskStatus{
				Total:     500 * 1024 * 1024 * 1024,
				Used:      100 * 1024 * 1024 * 1024,
				Available: 400 * 1024 * 1024 * 1024,
			},
		}))
	})

	// Task status - always return completed
	mux.HandleFunc("/api2/json/nodes/pve1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "stopped", ExitStatus: "OK"}))
	})

	// Delete VMs
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/101", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			_, _ = w.Write(envelope("UPID:pve1:delete:101"))
		}
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			_, _ = w.Write(envelope("UPID:pve1:delete:100"))
		}
	})

	// Config for newly cloned VMs
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/config", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(nil))
	})

	server := httptest.NewServer(mux)

	cfg := Config{
		Host:      server.URL,
		TokenID:   "test@pam!test",
		Secret:    "test-secret",
		Node:      "pve1",
		VMIDStart: 9000,
		VMIDEnd:   9999,
		CloneMode: "full",
	}

	mgr, err := NewProxmoxManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewProxmoxManager: %v", err)
	}

	return mgr, server
}

// --- Interface compliance ---

var (
	_ provider.Manager         = (*ProxmoxManager)(nil)
	_ provider.MultiHostLister = (*MultiNodeManager)(nil)
)

// --- NewProxmoxManager ---

func TestNewProxmoxManager_Valid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(nil))
	}))
	defer server.Close()

	cfg := Config{
		Host:    server.URL,
		TokenID: "root@pam!test",
		Secret:  "secret",
		Node:    "pve1",
	}
	mgr, err := NewProxmoxManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewProxmoxManager: %v", err)
	}
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
}

func TestNewProxmoxManager_InvalidConfig(t *testing.T) {
	_, err := NewProxmoxManager(Config{}, nil)
	if err == nil {
		t.Fatal("expected error for empty config")
	}
	if !strings.Contains(err.Error(), "invalid proxmox config") {
		t.Errorf("expected 'invalid proxmox config', got: %s", err.Error())
	}
}

func TestNewProxmoxManager_MissingHost(t *testing.T) {
	_, err := NewProxmoxManager(Config{
		TokenID: "root@pam!test",
		Secret:  "secret",
		Node:    "pve1",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing host")
	}
}

func TestNewProxmoxManager_MissingSecret(t *testing.T) {
	_, err := NewProxmoxManager(Config{
		Host:    "https://pve:8006",
		TokenID: "root@pam!test",
		Node:    "pve1",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestNewProxmoxManager_MissingNode(t *testing.T) {
	_, err := NewProxmoxManager(Config{
		Host:    "https://pve:8006",
		TokenID: "root@pam!test",
		Secret:  "secret",
	}, nil)
	if err == nil {
		t.Fatal("expected error for missing node")
	}
}

// --- CloneVM ---

func TestManagerCloneVM_Unsupported(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	_, err := mgr.CloneVM(context.Background(), "base.img", "new-vm", 2, 4096, "vmbr0")
	if err == nil {
		t.Fatal("expected error for CloneVM on Proxmox")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected 'not supported', got: %s", err.Error())
	}
}

// --- CloneFromVM ---

func TestManagerCloneFromVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	ref, err := mgr.CloneFromVM(context.Background(), "ubuntu-template", "new-sandbox", 2, 4096, "vmbr0")
	if err != nil {
		t.Fatalf("CloneFromVM: %v", err)
	}
	if ref.Name != "new-sandbox" {
		t.Errorf("expected new-sandbox, got %s", ref.Name)
	}
	if ref.UUID == "" {
		t.Error("expected non-empty UUID (VMID)")
	}
	if ref.UUID != "9000" {
		t.Errorf("expected UUID 9000, got %s", ref.UUID)
	}
}

func TestManagerCloneFromVM_ZeroCPUMemory(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	// With 0 cpu and 0 memory, should not call SetVMConfig
	ref, err := mgr.CloneFromVM(context.Background(), "ubuntu-template", "minimal-clone", 0, 0, "")
	if err != nil {
		t.Fatalf("CloneFromVM zero: %v", err)
	}
	if ref.Name != "minimal-clone" {
		t.Errorf("expected minimal-clone, got %s", ref.Name)
	}
}

func TestManagerCloneFromVM_SourceNotFound(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	_, err := mgr.CloneFromVM(context.Background(), "nonexistent-vm", "new-sandbox", 2, 4096, "vmbr0")
	if err == nil {
		t.Fatal("expected error for nonexistent source VM")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got: %s", err.Error())
	}
}

// --- StartVM ---

func TestManagerStartVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.StartVM(context.Background(), "ubuntu-template")
	if err != nil {
		t.Fatalf("StartVM: %v", err)
	}
}

func TestManagerStartVM_NonexistentVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.StartVM(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("expected error for nonexistent VM")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got: %s", err.Error())
	}
}

// --- StopVM ---

func TestManagerStopVM_Graceful(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.StopVM(context.Background(), "sandbox-1", false)
	if err != nil {
		t.Fatalf("StopVM graceful: %v", err)
	}
}

func TestManagerStopVM_Force(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.StopVM(context.Background(), "sandbox-1", true)
	if err != nil {
		t.Fatalf("StopVM force: %v", err)
	}
}

func TestManagerStopVM_NonexistentVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.StopVM(context.Background(), "ghost-vm", false)
	if err == nil {
		t.Fatal("expected error for nonexistent VM")
	}
}

// --- DestroyVM ---

func TestManagerDestroyVM_RunningVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	// sandbox-1 is running - should stop first, then delete
	err := mgr.DestroyVM(context.Background(), "sandbox-1")
	if err != nil {
		t.Fatalf("DestroyVM running: %v", err)
	}
}

func TestManagerDestroyVM_StoppedVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	// ubuntu-template is stopped - should delete directly
	err := mgr.DestroyVM(context.Background(), "ubuntu-template")
	if err != nil {
		t.Fatalf("DestroyVM stopped: %v", err)
	}
}

func TestManagerDestroyVM_NonexistentVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.DestroyVM(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent VM")
	}
}

// --- GetVMState ---

func TestManagerGetVMState_Running(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	state, err := mgr.GetVMState(context.Background(), "sandbox-1")
	if err != nil {
		t.Fatalf("GetVMState: %v", err)
	}
	if state != provider.VMStateRunning {
		t.Errorf("expected running, got %s", state)
	}
}

func TestManagerGetVMState_Stopped(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	state, err := mgr.GetVMState(context.Background(), "ubuntu-template")
	if err != nil {
		t.Fatalf("GetVMState: %v", err)
	}
	if state != provider.VMStateShutOff {
		t.Errorf("expected shut off, got %s", state)
	}
}

func TestManagerGetVMState_Paused(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	state, err := mgr.GetVMState(context.Background(), "sandbox-paused")
	if err != nil {
		t.Fatalf("GetVMState: %v", err)
	}
	if state != provider.VMStatePaused {
		t.Errorf("expected paused, got %s", state)
	}
}

func TestManagerGetVMState_Nonexistent(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	_, err := mgr.GetVMState(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error for nonexistent VM")
	}
}

// --- GetIPAddress ---

func TestManagerGetIPAddress(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	ip, mac, err := mgr.GetIPAddress(context.Background(), "sandbox-1", 5*time.Second)
	if err != nil {
		t.Fatalf("GetIPAddress: %v", err)
	}
	if ip != "10.0.0.50" {
		t.Errorf("expected 10.0.0.50, got %s", ip)
	}
	if mac != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("expected AA:BB:CC:DD:EE:FF, got %s", mac)
	}
}

func TestManagerGetIPAddress_SkipsLoopback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 200, Name: "lo-only"}}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/200/agent/network-get-interfaces", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(struct {
			Result []NetworkInterface `json:"result"`
		}{
			Result: []NetworkInterface{
				{
					Name:            "lo",
					HardwareAddress: "00:00:00:00:00:00",
					IPAddresses:     []GuestIPAddress{{IPAddressType: "ipv4", IPAddress: "127.0.0.1", Prefix: 8}},
				},
			},
		}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, _ := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)

	_, _, err := mgr.GetIPAddress(context.Background(), "lo-only", 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout for loopback-only VM")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got: %s", err.Error())
	}
}

func TestManagerGetIPAddress_IPv6Only(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 201, Name: "v6-only"}}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/201/agent/network-get-interfaces", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(struct {
			Result []NetworkInterface `json:"result"`
		}{
			Result: []NetworkInterface{
				{
					Name:            "eth0",
					HardwareAddress: "AA:BB:CC:DD:EE:FF",
					IPAddresses:     []GuestIPAddress{{IPAddressType: "ipv6", IPAddress: "fe80::1", Prefix: 64}},
				},
			},
		}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, _ := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)

	_, _, err := mgr.GetIPAddress(context.Background(), "v6-only", 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout for IPv6-only VM")
	}
}

func TestManagerGetIPAddress_ContextCancelled(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 202, Name: "slow-vm"}}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/202/agent/network-get-interfaces", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("guest agent not running"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, _ := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err := mgr.GetIPAddress(ctx, "slow-vm", 30*time.Second)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestManagerGetIPAddress_NonexistentVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	_, _, err := mgr.GetIPAddress(context.Background(), "ghost", 1*time.Second)
	if err == nil {
		t.Fatal("expected error for nonexistent VM")
	}
}

// --- ValidateSourceVM ---

func TestManagerValidateSourceVM_Valid(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	result, err := mgr.ValidateSourceVM(context.Background(), "ubuntu-template")
	if err != nil {
		t.Fatalf("ValidateSourceVM: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid VM")
	}
	if !result.HasNetwork {
		t.Error("expected has_network=true")
	}
	if result.State != provider.VMStateShutOff {
		t.Errorf("expected shut off, got %s", result.State)
	}
	if result.VMName != "ubuntu-template" {
		t.Errorf("expected ubuntu-template, got %s", result.VMName)
	}
}

func TestManagerValidateSourceVM_RunningVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	result, err := mgr.ValidateSourceVM(context.Background(), "sandbox-1")
	if err != nil {
		t.Fatalf("ValidateSourceVM: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid VM (running VMs can be cloned)")
	}
	if result.State != provider.VMStateRunning {
		t.Errorf("expected running, got %s", result.State)
	}
}

func TestManagerValidateSourceVM_Nonexistent(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	result, err := mgr.ValidateSourceVM(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ValidateSourceVM: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid for nonexistent VM")
	}
	if len(result.Errors) == 0 {
		t.Error("expected errors for nonexistent VM")
	}
}

func TestManagerValidateSourceVM_NoNetwork(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	result, err := mgr.ValidateSourceVM(context.Background(), "no-net-vm")
	if err != nil {
		t.Fatalf("ValidateSourceVM: %v", err)
	}
	if result.HasNetwork {
		t.Error("expected has_network=false")
	}
	hasNetWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "no network interface") {
			hasNetWarning = true
		}
	}
	if !hasNetWarning {
		t.Error("expected warning about no network interface")
	}
}

func TestManagerValidateSourceVM_NoGuestAgent(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	result, err := mgr.ValidateSourceVM(context.Background(), "no-agent-vm")
	if err != nil {
		t.Fatalf("ValidateSourceVM: %v", err)
	}
	hasAgentWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "guest agent") {
			hasAgentWarning = true
		}
	}
	if !hasAgentWarning {
		t.Error("expected warning about guest agent")
	}
}

// --- CheckHostResources ---

func TestManagerCheckHostResources_Sufficient(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	result, err := mgr.CheckHostResources(context.Background(), 2, 4096)
	if err != nil {
		t.Fatalf("CheckHostResources: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid resources")
	}
	if result.TotalCPUs != 16 {
		t.Errorf("expected 16 CPUs, got %d", result.TotalCPUs)
	}
	if result.RequiredCPUs != 2 {
		t.Errorf("expected required 2 CPUs, got %d", result.RequiredCPUs)
	}
	if result.RequiredMemoryMB != 4096 {
		t.Errorf("expected required 4096 MB, got %d", result.RequiredMemoryMB)
	}
	if result.NeedsMemoryApproval {
		t.Error("should not need memory approval (48GB free)")
	}
	if result.NeedsCPUApproval {
		t.Error("should not need CPU approval (15% usage)")
	}
	// 48GB free = 49152 MB
	if result.AvailableMemoryMB != 49152 {
		t.Errorf("expected 49152 MB free, got %d", result.AvailableMemoryMB)
	}
}

func TestManagerCheckHostResources_InsufficientMemory(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(NodeStatus{
			CPU:    0.10,
			MaxCPU: 4,
			Memory: MemoryStatus{
				Total: 8 * 1024 * 1024 * 1024,
				Used:  7 * 1024 * 1024 * 1024,
				Free:  1 * 1024 * 1024 * 1024, // 1GB free
			},
			RootFS: DiskStatus{Available: 50 * 1024 * 1024 * 1024},
		}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, _ := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)

	result, err := mgr.CheckHostResources(context.Background(), 1, 4096)
	if err != nil {
		t.Fatalf("CheckHostResources: %v", err)
	}
	if !result.NeedsMemoryApproval {
		t.Error("expected memory approval needed (4GB requested, 1GB free)")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning about insufficient memory")
	}
}

func TestManagerCheckHostResources_HighCPU(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(NodeStatus{
			CPU:    0.92, // 92% usage
			MaxCPU: 4,
			Memory: MemoryStatus{
				Total: 8 * 1024 * 1024 * 1024,
				Free:  4 * 1024 * 1024 * 1024,
			},
			RootFS: DiskStatus{Available: 50 * 1024 * 1024 * 1024},
		}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, _ := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)

	result, err := mgr.CheckHostResources(context.Background(), 2, 2048)
	if err != nil {
		t.Fatalf("CheckHostResources: %v", err)
	}
	if !result.NeedsCPUApproval {
		t.Error("expected CPU approval needed (92% usage)")
	}
}

// --- InjectSSHKey ---

func TestManagerInjectSSHKey(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.InjectSSHKey(context.Background(), "sandbox-1", "ubuntu", "ssh-ed25519 AAAA... user@host")
	if err != nil {
		t.Fatalf("InjectSSHKey: %v", err)
	}
}

func TestManagerInjectSSHKey_EmptyUsername(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	// Should not set ciuser when username is empty
	err := mgr.InjectSSHKey(context.Background(), "sandbox-1", "", "ssh-rsa AAAA...")
	if err != nil {
		t.Fatalf("InjectSSHKey empty user: %v", err)
	}
}

func TestManagerInjectSSHKey_NonexistentVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	err := mgr.InjectSSHKey(context.Background(), "ghost-vm", "user", "ssh-rsa AAAA...")
	if err == nil {
		t.Fatal("expected error for nonexistent VM")
	}
}

// --- CreateSnapshot ---

func TestManagerCreateSnapshot(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	ref, err := mgr.CreateSnapshot(context.Background(), "sandbox-1", "snap1", false)
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if ref.Name != "snap1" {
		t.Errorf("expected snap1, got %s", ref.Name)
	}
	if ref.Kind != "INTERNAL" {
		t.Errorf("expected INTERNAL, got %s", ref.Kind)
	}
	if !strings.HasPrefix(ref.Ref, "proxmox:") {
		t.Errorf("expected proxmox: prefix in ref, got %s", ref.Ref)
	}
	if !strings.Contains(ref.Ref, "101") {
		t.Errorf("expected VMID 101 in ref, got %s", ref.Ref)
	}
}

func TestManagerCreateSnapshot_ExternalIgnored(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	// external=true should work the same as false for Proxmox
	ref, err := mgr.CreateSnapshot(context.Background(), "sandbox-1", "snap-ext", true)
	if err != nil {
		t.Fatalf("CreateSnapshot external: %v", err)
	}
	if ref.Name != "snap-ext" {
		t.Errorf("expected snap-ext, got %s", ref.Name)
	}
	if ref.Kind != "INTERNAL" {
		t.Errorf("expected INTERNAL even with external=true, got %s", ref.Kind)
	}
}

func TestManagerCreateSnapshot_NonexistentVM(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	_, err := mgr.CreateSnapshot(context.Background(), "nonexistent", "snap1", false)
	if err == nil {
		t.Fatal("expected error for nonexistent VM")
	}
}

// --- DiffSnapshot ---

func TestManagerDiffSnapshot(t *testing.T) {
	mgr, server := mockProxmoxAPI(t)
	defer server.Close()

	plan, err := mgr.DiffSnapshot(context.Background(), "sandbox-1", "snap1", "snap2")
	if err != nil {
		t.Fatalf("DiffSnapshot: %v", err)
	}
	if plan.VMName != "sandbox-1" {
		t.Errorf("expected sandbox-1, got %s", plan.VMName)
	}
	if plan.FromSnapshot != "snap1" {
		t.Errorf("expected snap1, got %s", plan.FromSnapshot)
	}
	if plan.ToSnapshot != "snap2" {
		t.Errorf("expected snap2, got %s", plan.ToSnapshot)
	}
	if len(plan.Notes) == 0 {
		t.Error("expected notes about Proxmox limitations")
	}
	if plan.FromMount != "" || plan.ToMount != "" {
		t.Error("expected empty mounts for Proxmox")
	}
}

// --- Resolver ---

func TestVMResolverCaching(t *testing.T) {
	var callCount int32
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		vms := []VMListEntry{{VMID: 100, Name: "cached-vm"}}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	resolver := NewVMResolver(client)

	// First call: cache miss, triggers refresh
	vmid, err := resolver.ResolveVMID(context.Background(), "cached-vm")
	if err != nil {
		t.Fatalf("ResolveVMID: %v", err)
	}
	if vmid != 100 {
		t.Errorf("expected 100, got %d", vmid)
	}
	firstCount := atomic.LoadInt32(&callCount)

	// Second call: cache hit
	_, err = resolver.ResolveVMID(context.Background(), "cached-vm")
	if err != nil {
		t.Fatalf("ResolveVMID: %v", err)
	}
	if atomic.LoadInt32(&callCount) != firstCount {
		t.Error("expected cached result, but API was called again")
	}
}

func TestVMResolverRefresh(t *testing.T) {
	var callCount int32
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n <= 1 {
			_, _ = w.Write(envelope([]VMListEntry{{VMID: 100, Name: "vm1"}}))
		} else {
			// After refresh, new VM appears
			_, _ = w.Write(envelope([]VMListEntry{
				{VMID: 100, Name: "vm1"},
				{VMID: 101, Name: "vm2"},
			}))
		}
	})
	defer server.Close()

	resolver := NewVMResolver(client)

	// Pre-populate cache with first API call (only "vm1")
	if err := resolver.Refresh(context.Background()); err != nil {
		t.Fatalf("initial refresh: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatal("expected exactly 1 API call after initial refresh")
	}

	// Now resolve vm2 - cache miss triggers second API call which includes vm2
	vmid, err := resolver.ResolveVMID(context.Background(), "vm2")
	if err != nil {
		t.Fatalf("ResolveVMID after refresh: %v", err)
	}
	if vmid != 101 {
		t.Errorf("expected VMID 101, got %d", vmid)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Error("expected 2 API calls total")
	}
}

func TestVMResolverResolveName(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 100, Name: "test-vm"},
			{VMID: 200, Name: "other-vm"},
		}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	resolver := NewVMResolver(client)

	name, err := resolver.ResolveName(context.Background(), 200)
	if err != nil {
		t.Fatalf("ResolveName: %v", err)
	}
	if name != "other-vm" {
		t.Errorf("expected other-vm, got %s", name)
	}
}

func TestVMResolverResolveName_NotFound(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 100, Name: "vm1"}}))
	})
	defer server.Close()

	resolver := NewVMResolver(client)

	_, err := resolver.ResolveName(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for unknown VMID")
	}
	if !strings.Contains(err.Error(), "999") {
		t.Errorf("expected VMID in error, got: %s", err.Error())
	}
}

func TestVMResolverListAll(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 100, Name: "vm1"},
			{VMID: 101, Name: "vm2"},
			{VMID: 102, Name: "vm3"},
		}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	resolver := NewVMResolver(client)

	vms, err := resolver.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(vms) != 3 {
		t.Errorf("expected 3 VMs, got %d", len(vms))
	}
}

func TestVMResolverListAll_Cached(t *testing.T) {
	var callCount int32
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 100, Name: "vm1"}}))
	})
	defer server.Close()

	resolver := NewVMResolver(client)

	// Pre-populate cache
	_ = resolver.Refresh(context.Background())
	beforeCount := atomic.LoadInt32(&callCount)

	// ListAll should still call the API (to get fresh data) but not refresh
	_, _ = resolver.ListAll(context.Background())
	// With populated cache, ListAll calls ListVMs once (not Refresh + ListVMs)
	afterCount := atomic.LoadInt32(&callCount)
	if afterCount-beforeCount != 1 {
		t.Errorf("expected 1 additional API call, got %d", afterCount-beforeCount)
	}
}

func TestVMResolverRefresh_APIError(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	})
	defer server.Close()

	resolver := NewVMResolver(client)
	err := resolver.Refresh(context.Background())
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

// --- Config Validation ---

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid full config",
			cfg: Config{
				Host: "https://pve.example.com:8006", TokenID: "root@pam!fluid",
				Secret: "secret", Node: "pve1", VMIDStart: 9000, VMIDEnd: 9999,
			},
		},
		{
			name: "valid linked clone",
			cfg: Config{
				Host: "https://pve:8006", TokenID: "root@pam!fluid",
				Secret: "secret", Node: "pve1", CloneMode: "linked",
			},
		},
		{
			name:    "missing host",
			cfg:     Config{TokenID: "root@pam!fluid", Secret: "secret", Node: "pve1"},
			wantErr: true, errMsg: "host is required",
		},
		{
			name:    "missing token_id",
			cfg:     Config{Host: "https://pve:8006", Secret: "secret", Node: "pve1"},
			wantErr: true, errMsg: "token_id is required",
		},
		{
			name:    "missing secret",
			cfg:     Config{Host: "https://pve:8006", TokenID: "root@pam!test", Node: "pve1"},
			wantErr: true, errMsg: "secret is required",
		},
		{
			name:    "missing node",
			cfg:     Config{Host: "https://pve:8006", TokenID: "root@pam!test", Secret: "s"},
			wantErr: true, errMsg: "node is required",
		},
		{
			name: "bad vmid range",
			cfg: Config{
				Host: "https://pve:8006", TokenID: "t", Secret: "s", Node: "n",
				VMIDStart: 9999, VMIDEnd: 9000,
			},
			wantErr: true, errMsg: "vmid_end",
		},
		{
			name: "equal vmid range",
			cfg: Config{
				Host: "https://pve:8006", TokenID: "t", Secret: "s", Node: "n",
				VMIDStart: 9000, VMIDEnd: 9000,
			},
			wantErr: true, errMsg: "vmid_end",
		},
		{
			name: "bad clone mode",
			cfg: Config{
				Host: "https://pve:8006", TokenID: "t", Secret: "s", Node: "n",
				CloneMode: "snapshot",
			},
			wantErr: true, errMsg: "clone_mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got: %s", tt.errMsg, err.Error())
			}
		})
	}
}

func TestConfigValidation_DefaultVMIDRange(t *testing.T) {
	cfg := Config{
		Host: "https://pve:8006", TokenID: "t", Secret: "s", Node: "n",
	}
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if cfg.VMIDStart != 9000 {
		t.Errorf("expected default VMIDStart 9000, got %d", cfg.VMIDStart)
	}
	if cfg.VMIDEnd != 9999 {
		t.Errorf("expected default VMIDEnd 9999, got %d", cfg.VMIDEnd)
	}
}

func TestConfigValidation_DefaultCloneMode(t *testing.T) {
	cfg := Config{
		Host: "https://pve:8006", TokenID: "t", Secret: "s", Node: "n",
	}
	_ = cfg.Validate()
	if cfg.CloneMode != "full" {
		t.Errorf("expected default CloneMode full, got %s", cfg.CloneMode)
	}
}

// --- MultiNodeManager ---

func TestMultiNodeManager_ListVMs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 100, Name: "vm1", Status: "running"},
			{VMID: 101, Name: "vm2", Status: "stopped"},
			{VMID: 102, Name: "vm3", Status: "paused"},
		}
		_, _ = w.Write(envelope(vms))
	}))
	defer server.Close()

	mnm := NewMultiNodeManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
	}, nil)

	result, err := mnm.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs: %v", err)
	}
	if len(result.VMs) != 3 {
		t.Fatalf("expected 3 VMs, got %d", len(result.VMs))
	}
	if len(result.HostErrors) != 0 {
		t.Errorf("expected no host errors, got %d", len(result.HostErrors))
	}

	// Check fields
	vm := result.VMs[0]
	if vm.Name != "vm1" {
		t.Errorf("expected vm1, got %s", vm.Name)
	}
	if vm.UUID != "100" {
		t.Errorf("expected UUID 100, got %s", vm.UUID)
	}
	if vm.State != "running" {
		t.Errorf("expected running, got %s", vm.State)
	}
	if vm.HostName != "pve1" {
		t.Errorf("expected pve1, got %s", vm.HostName)
	}
	if vm.HostAddress != server.URL {
		t.Errorf("expected %s, got %s", server.URL, vm.HostAddress)
	}
	if !vm.Persistent {
		t.Error("expected persistent=true")
	}
}

func TestMultiNodeManager_ListVMs_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	}))
	defer server.Close()

	mnm := NewMultiNodeManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
	}, nil)

	result, err := mnm.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("expected nil error (errors in result), got: %v", err)
	}
	if len(result.VMs) != 0 {
		t.Errorf("expected 0 VMs on error, got %d", len(result.VMs))
	}
	if len(result.HostErrors) != 1 {
		t.Fatalf("expected 1 host error, got %d", len(result.HostErrors))
	}
	if result.HostErrors[0].HostName != "pve1" {
		t.Errorf("expected pve1 in error, got %s", result.HostErrors[0].HostName)
	}
}

func TestMultiNodeManager_ListVMs_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{}))
	}))
	defer server.Close()

	mnm := NewMultiNodeManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
	}, nil)

	result, err := mnm.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs: %v", err)
	}
	if len(result.VMs) != 0 {
		t.Errorf("expected 0 VMs, got %d", len(result.VMs))
	}
}

func TestMultiNodeManager_FindHostForVM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 100, Name: "vm-alpha"},
			{VMID: 101, Name: "vm-beta"},
		}
		_, _ = w.Write(envelope(vms))
	}))
	defer server.Close()

	mnm := NewMultiNodeManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
	}, nil)

	// Found
	host, addr, err := mnm.FindHostForVM(context.Background(), "vm-beta")
	if err != nil {
		t.Fatalf("FindHostForVM: %v", err)
	}
	if host != "pve1" {
		t.Errorf("expected pve1, got %s", host)
	}
	if addr != server.URL {
		t.Errorf("expected %s, got %s", server.URL, addr)
	}

	// Not found
	_, _, err = mnm.FindHostForVM(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found', got: %s", err.Error())
	}
}

func TestMultiNodeManager_FindHostForVM_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	mnm := NewMultiNodeManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
	}, nil)

	_, _, err := mnm.FindHostForVM(context.Background(), "vm1")
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}

// --- Linked Clone Manager ---

func TestManagerCloneFromVM_LinkedClone(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 100, Name: "template"}}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/clone", func(w http.ResponseWriter, r *http.Request) {
		// Verify linked clone does NOT send full=1
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("full") != "" {
			t.Error("linked clone should not have full param")
		}
		_, _ = w.Write(envelope("UPID:pve1:linked-clone"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/config", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(nil))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "stopped", ExitStatus: "OK"}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, err := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
		CloneMode: "linked",
	}, nil)
	if err != nil {
		t.Fatalf("NewProxmoxManager: %v", err)
	}

	ref, err := mgr.CloneFromVM(context.Background(), "template", "linked-sandbox", 0, 0, "")
	if err != nil {
		t.Fatalf("CloneFromVM linked: %v", err)
	}
	if ref.Name != "linked-sandbox" {
		t.Errorf("expected linked-sandbox, got %s", ref.Name)
	}
}

// --- Full lifecycle test ---

func TestManagerFullLifecycle(t *testing.T) {
	// Simulate: clone -> start -> get IP -> snapshot -> stop -> destroy
	vmState := "stopped"

	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{
			{VMID: 100, Name: "template", Status: "stopped"},
			{VMID: 9000, Name: "lifecycle-vm", Status: vmState},
		}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/clone", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:clone"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/status/current", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 9000, Name: "lifecycle-vm", Status: vmState}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/status/start", func(w http.ResponseWriter, r *http.Request) {
		vmState = "running"
		_, _ = w.Write(envelope("UPID:start"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/status/stop", func(w http.ResponseWriter, r *http.Request) {
		vmState = "stopped"
		_, _ = w.Write(envelope("UPID:stop"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/status/shutdown", func(w http.ResponseWriter, r *http.Request) {
		vmState = "stopped"
		_, _ = w.Write(envelope("UPID:shutdown"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/config", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(nil))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/snapshot", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope("UPID:snapshot"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000/agent/network-get-interfaces", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(struct {
			Result []NetworkInterface `json:"result"`
		}{
			Result: []NetworkInterface{
				{
					Name: "eth0", HardwareAddress: "DE:AD:BE:EF:00:01",
					IPAddresses: []GuestIPAddress{{IPAddressType: "ipv4", IPAddress: "10.1.1.100", Prefix: 24}},
				},
			},
		}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/9000", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			_, _ = w.Write(envelope("UPID:delete"))
		}
	})
	mux.HandleFunc("/api2/json/nodes/pve1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "stopped", ExitStatus: "OK"}))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, err := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// 1. Start
	err = mgr.StartVM(context.Background(), "lifecycle-vm")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	// 2. Get state
	state, err := mgr.GetVMState(context.Background(), "lifecycle-vm")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if state != provider.VMStateRunning {
		t.Errorf("expected running after start, got %s", state)
	}

	// 3. Get IP
	ip, mac, err := mgr.GetIPAddress(context.Background(), "lifecycle-vm", 5*time.Second)
	if err != nil {
		t.Fatalf("get IP: %v", err)
	}
	if ip != "10.1.1.100" {
		t.Errorf("expected 10.1.1.100, got %s", ip)
	}
	if mac != "DE:AD:BE:EF:00:01" {
		t.Errorf("expected MAC, got %s", mac)
	}

	// 4. Snapshot
	snap, err := mgr.CreateSnapshot(context.Background(), "lifecycle-vm", "checkpoint", false)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap.Name != "checkpoint" {
		t.Errorf("expected checkpoint, got %s", snap.Name)
	}

	// 5. Stop
	err = mgr.StopVM(context.Background(), "lifecycle-vm", false)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	// 6. Destroy
	err = mgr.DestroyVM(context.Background(), "lifecycle-vm")
	if err != nil {
		t.Fatalf("destroy: %v", err)
	}
}

// --- Edge: VM names with special characters ---

func TestManagerVMNameWithSpaces(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 100, Name: "my special vm"}}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/status/current", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 100, Status: "stopped"}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, _ := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)

	state, err := mgr.GetVMState(context.Background(), "my special vm")
	if err != nil {
		t.Fatalf("GetVMState with spaces: %v", err)
	}
	if state != provider.VMStateShutOff {
		t.Errorf("expected shut off, got %s", state)
	}
}

// --- Concurrent resolver access ---

func TestVMResolverConcurrent(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 100, Name: "vm1"},
			{VMID: 101, Name: "vm2"},
			{VMID: 102, Name: "vm3"},
		}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	resolver := NewVMResolver(client)

	// Warm cache
	_ = resolver.Refresh(context.Background())

	// Launch concurrent lookups
	done := make(chan error, 30)
	for range 10 {
		go func() {
			_, err := resolver.ResolveVMID(context.Background(), "vm1")
			done <- err
		}()
		go func() {
			_, err := resolver.ResolveName(context.Background(), 101)
			done <- err
		}()
		go func() {
			err := resolver.Refresh(context.Background())
			done <- err
		}()
	}

	for range 30 {
		if err := <-done; err != nil {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}
}

// --- Verify StopVM routes ---

func TestManagerStopVM_VerifyGracefulRoute(t *testing.T) {
	var shutdownCalled, stopCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/api2/json/nodes/pve1/qemu", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{{VMID: 100, Name: "vm1"}}))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/status/shutdown", func(w http.ResponseWriter, r *http.Request) {
		shutdownCalled = true
		_, _ = w.Write(envelope("UPID:shutdown"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/qemu/100/status/stop", func(w http.ResponseWriter, r *http.Request) {
		stopCalled = true
		_, _ = w.Write(envelope("UPID:stop"))
	})
	mux.HandleFunc("/api2/json/nodes/pve1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "stopped", ExitStatus: "OK"}))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mgr, _ := NewProxmoxManager(Config{
		Host: server.URL, TokenID: "t", Secret: "s", Node: "pve1",
		VMIDStart: 9000, VMIDEnd: 9999,
	}, nil)

	// Graceful: should call shutdown, not stop
	_ = mgr.StopVM(context.Background(), "vm1", false)
	if !shutdownCalled {
		t.Error("graceful stop should call /shutdown")
	}
	if stopCalled {
		t.Error("graceful stop should not call /stop")
	}

	shutdownCalled = false
	stopCalled = false

	// Force: should call stop, not shutdown
	_ = mgr.StopVM(context.Background(), "vm1", true)
	if !stopCalled {
		t.Error("force stop should call /stop")
	}
	if shutdownCalled {
		t.Error("force stop should not call /shutdown")
	}
}

// suppress unused import
var _ = fmt.Sprint
