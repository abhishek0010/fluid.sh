package proxmox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestServer creates a mock Proxmox API server and returns a Client pointed at it.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	cfg := Config{
		Host:      server.URL,
		TokenID:   "test@pam!test",
		Secret:    "test-secret",
		Node:      "pve1",
		VMIDStart: 9000,
		VMIDEnd:   9999,
	}
	client := NewClient(cfg, nil)
	return client, server
}

// envelope wraps data in Proxmox API response format.
func envelope(data any) []byte {
	resp := struct {
		Data any `json:"data"`
	}{Data: data}
	b, _ := json.Marshal(resp)
	return b
}

// --- Auth & Headers ---

func TestAuthHeader(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "PVEAPIToken=test@pam!test=test-secret" {
			t.Errorf("unexpected auth header: %s", auth)
		}
		_, _ = w.Write(envelope([]VMListEntry{}))
	})
	defer server.Close()
	_, _ = client.ListVMs(context.Background())
}

func TestContentTypeSetForPOST(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if r.Method == http.MethodPost {
			if ct != "application/x-www-form-urlencoded" {
				t.Errorf("expected form content-type for POST with body, got %q", ct)
			}
		}
		_, _ = w.Write(envelope("UPID:test"))
	})
	defer server.Close()
	// CloneVM sends a POST with url.Values body, so Content-Type should be set
	_, _ = client.CloneVM(context.Background(), 100, 200, "clone", true)
}

func TestContentTypeNotSetForGET(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if r.Method == http.MethodGet && ct != "" {
			t.Errorf("expected no content-type for GET, got %q", ct)
		}
		_, _ = w.Write(envelope([]VMListEntry{}))
	})
	defer server.Close()
	_, _ = client.ListVMs(context.Background())
}

// --- ListVMs ---

func TestListVMs(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/qemu" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		vms := []VMListEntry{
			{VMID: 100, Name: "ubuntu-base", Status: "stopped", Template: 1, MaxMem: 4294967296},
			{VMID: 101, Name: "sandbox-1", Status: "running", CPU: 0.15, Mem: 1073741824},
		}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	vms, err := client.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("expected 2 VMs, got %d", len(vms))
	}
	if vms[0].Name != "ubuntu-base" {
		t.Errorf("expected ubuntu-base, got %s", vms[0].Name)
	}
	if vms[0].Template != 1 {
		t.Errorf("expected template=1")
	}
	if vms[0].MaxMem != 4294967296 {
		t.Errorf("expected maxmem 4294967296, got %d", vms[0].MaxMem)
	}
	if vms[1].VMID != 101 {
		t.Errorf("expected VMID 101, got %d", vms[1].VMID)
	}
	if vms[1].Status != "running" {
		t.Errorf("expected running, got %s", vms[1].Status)
	}
}

func TestListVMs_Empty(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{}))
	})
	defer server.Close()

	vms, err := client.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs: %v", err)
	}
	if len(vms) != 0 {
		t.Errorf("expected 0 VMs, got %d", len(vms))
	}
}

// --- GetVMStatus ---

func TestGetVMStatus(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/status/current" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		status := VMStatus{
			VMID:      100,
			Name:      "ubuntu-base",
			Status:    "running",
			QMPStatus: "running",
			CPU:       0.15,
			Mem:       1073741824,
			MaxMem:    4294967296,
			MaxDisk:   10737418240,
			Uptime:    3600,
			PID:       12345,
		}
		_, _ = w.Write(envelope(status))
	})
	defer server.Close()

	status, err := client.GetVMStatus(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetVMStatus: %v", err)
	}
	if status.Status != "running" {
		t.Errorf("expected running, got %s", status.Status)
	}
	if status.VMID != 100 {
		t.Errorf("expected VMID 100, got %d", status.VMID)
	}
	if status.QMPStatus != "running" {
		t.Errorf("expected qmpstatus running, got %s", status.QMPStatus)
	}
	if status.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", status.PID)
	}
	if status.MaxDisk != 10737418240 {
		t.Errorf("expected maxdisk, got %d", status.MaxDisk)
	}
}

func TestGetVMStatus_Stopped(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 200, Status: "stopped", Template: 1}))
	})
	defer server.Close()

	status, err := client.GetVMStatus(context.Background(), 200)
	if err != nil {
		t.Fatalf("GetVMStatus: %v", err)
	}
	if status.Status != "stopped" {
		t.Errorf("expected stopped, got %s", status.Status)
	}
	if status.Template != 1 {
		t.Errorf("expected template=1")
	}
}

func TestGetVMStatus_Locked(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMStatus{VMID: 300, Status: "running", Lock: "clone"}))
	})
	defer server.Close()

	status, err := client.GetVMStatus(context.Background(), 300)
	if err != nil {
		t.Fatalf("GetVMStatus: %v", err)
	}
	if status.Lock != "clone" {
		t.Errorf("expected lock=clone, got %s", status.Lock)
	}
}

// --- GetVMConfig ---

func TestGetVMConfig(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		cfg := VMConfig{
			Name:      "ubuntu-base",
			Memory:    4096,
			Cores:     2,
			Sockets:   1,
			CPU:       "host",
			Net0:      "virtio=AA:BB:CC:DD:EE:FF,bridge=vmbr0",
			Agent:     "1",
			IPConfig0: "ip=dhcp",
			CIUser:    "ubuntu",
			Boot:      "order=scsi0",
		}
		_, _ = w.Write(envelope(cfg))
	})
	defer server.Close()

	cfg, err := client.GetVMConfig(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetVMConfig: %v", err)
	}
	if cfg.Cores != 2 {
		t.Errorf("expected 2 cores, got %d", cfg.Cores)
	}
	if cfg.Agent != "1" {
		t.Errorf("expected agent=1, got %s", cfg.Agent)
	}
	if cfg.Memory != 4096 {
		t.Errorf("expected memory 4096, got %d", cfg.Memory)
	}
	if cfg.CIUser != "ubuntu" {
		t.Errorf("expected ciuser=ubuntu, got %s", cfg.CIUser)
	}
	if cfg.IPConfig0 != "ip=dhcp" {
		t.Errorf("expected ipconfig0=ip=dhcp, got %s", cfg.IPConfig0)
	}
	if cfg.CPU != "host" {
		t.Errorf("expected cpu=host, got %s", cfg.CPU)
	}
}

func TestGetVMConfig_Minimal(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(VMConfig{Name: "minimal", Memory: 512, Cores: 1}))
	})
	defer server.Close()

	cfg, err := client.GetVMConfig(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetVMConfig: %v", err)
	}
	if cfg.Net0 != "" {
		t.Errorf("expected empty net0, got %s", cfg.Net0)
	}
	if cfg.Agent != "" {
		t.Errorf("expected empty agent, got %s", cfg.Agent)
	}
}

// --- CloneVM ---

func TestCloneVM_Full(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/clone" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, "newid=9001") {
			t.Errorf("expected newid=9001 in body, got: %s", bodyStr)
		}
		if !strings.Contains(bodyStr, "name=sandbox-1") {
			t.Errorf("expected name=sandbox-1 in body, got: %s", bodyStr)
		}
		if !strings.Contains(bodyStr, "full=1") {
			t.Errorf("expected full=1 in body, got: %s", bodyStr)
		}
		_, _ = w.Write(envelope("UPID:pve1:00001234:00000000:12345678:qmclone:100:root@pam:"))
	})
	defer server.Close()

	upid, err := client.CloneVM(context.Background(), 100, 9001, "sandbox-1", true)
	if err != nil {
		t.Fatalf("CloneVM: %v", err)
	}
	if !strings.Contains(upid, "qmclone") {
		t.Errorf("expected UPID with qmclone, got %s", upid)
	}
}

func TestCloneVM_Linked(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if strings.Contains(bodyStr, "full=") {
			t.Errorf("linked clone should not have full param, got: %s", bodyStr)
		}
		_, _ = w.Write(envelope("UPID:pve1:linked"))
	})
	defer server.Close()

	upid, err := client.CloneVM(context.Background(), 100, 9001, "sandbox-1", false)
	if err != nil {
		t.Fatalf("CloneVM linked: %v", err)
	}
	if upid == "" {
		t.Error("expected non-empty UPID")
	}
}

// --- SetVMConfig ---

func TestSetVMConfig(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, "cores=4") {
			t.Errorf("expected cores=4 in body, got: %s", bodyStr)
		}
		if !strings.Contains(bodyStr, "memory=8192") {
			t.Errorf("expected memory=8192 in body, got: %s", bodyStr)
		}
		_, _ = w.Write(envelope(nil))
	})
	defer server.Close()

	params := make(map[string][]string)
	params["cores"] = []string{"4"}
	params["memory"] = []string{"8192"}
	err := client.SetVMConfig(context.Background(), 100, params)
	if err != nil {
		t.Fatalf("SetVMConfig: %v", err)
	}
}

// --- StartVM ---

func TestStartVM(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/status/start" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_, _ = w.Write(envelope("UPID:pve1:00001234:start"))
	})
	defer server.Close()

	upid, err := client.StartVM(context.Background(), 100)
	if err != nil {
		t.Fatalf("StartVM: %v", err)
	}
	if upid == "" {
		t.Error("expected non-empty UPID")
	}
}

// --- StopVM ---

func TestStopVM(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/status/stop" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write(envelope("UPID:pve1:00001234:stop"))
	})
	defer server.Close()

	upid, err := client.StopVM(context.Background(), 100)
	if err != nil {
		t.Fatalf("StopVM: %v", err)
	}
	if upid == "" {
		t.Error("expected non-empty UPID")
	}
}

// --- ShutdownVM ---

func TestShutdownVM(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/status/shutdown" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_, _ = w.Write(envelope("UPID:pve1:00001234:shutdown"))
	})
	defer server.Close()

	upid, err := client.ShutdownVM(context.Background(), 100)
	if err != nil {
		t.Fatalf("ShutdownVM: %v", err)
	}
	if upid == "" {
		t.Error("expected non-empty UPID")
	}
}

// --- DeleteVM ---

func TestDeleteVM(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// Verify purge and destroy-unreferenced-disks params in URL
		if !strings.Contains(r.URL.RawQuery, "purge=1") {
			t.Errorf("expected purge=1 in query, got %s", r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.RawQuery, "destroy-unreferenced-disks=1") {
			t.Errorf("expected destroy-unreferenced-disks=1, got %s", r.URL.RawQuery)
		}
		_, _ = w.Write(envelope("UPID:pve1:00001234:delete"))
	})
	defer server.Close()

	upid, err := client.DeleteVM(context.Background(), 100)
	if err != nil {
		t.Fatalf("DeleteVM: %v", err)
	}
	if upid == "" {
		t.Error("expected non-empty UPID")
	}
}

// --- GetTaskStatus ---

func TestGetTaskStatus(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		status := TaskStatus{
			Status:     "stopped",
			ExitStatus: "OK",
			Type:       "qmclone",
			Node:       "pve1",
			PID:        12345,
			StartTime:  1700000000,
			EndTime:    1700000060,
		}
		_, _ = w.Write(envelope(status))
	})
	defer server.Close()

	status, err := client.GetTaskStatus(context.Background(), "UPID:pve1:00001234:test")
	if err != nil {
		t.Fatalf("GetTaskStatus: %v", err)
	}
	if status.Status != "stopped" {
		t.Errorf("expected stopped, got %s", status.Status)
	}
	if status.ExitStatus != "OK" {
		t.Errorf("expected OK, got %s", status.ExitStatus)
	}
	if status.Type != "qmclone" {
		t.Errorf("expected qmclone, got %s", status.Type)
	}
	if status.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", status.PID)
	}
}

func TestGetTaskStatus_Running(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "running", Type: "qmstart"}))
	})
	defer server.Close()

	status, err := client.GetTaskStatus(context.Background(), "UPID:pve1:running")
	if err != nil {
		t.Fatalf("GetTaskStatus: %v", err)
	}
	if status.Status != "running" {
		t.Errorf("expected running, got %s", status.Status)
	}
}

// --- WaitForTask ---

func TestWaitForTask_ImmediateSuccess(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "stopped", ExitStatus: "OK"}))
	})
	defer server.Close()

	err := client.WaitForTask(context.Background(), "UPID:pve1:test")
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}
}

func TestWaitForTask_EmptyUPID(t *testing.T) {
	// Should return nil immediately without making any API calls
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make API call for empty UPID")
	})
	defer server.Close()

	err := client.WaitForTask(context.Background(), "")
	if err != nil {
		t.Fatalf("WaitForTask empty UPID: %v", err)
	}
}

func TestWaitForTask_TaskFailure(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "stopped", ExitStatus: "command 'qm clone' failed: storage error"}))
	})
	defer server.Close()

	err := client.WaitForTask(context.Background(), "UPID:pve1:fail")
	if err == nil {
		t.Fatal("expected error for failed task")
	}
	if !strings.Contains(err.Error(), "task failed") {
		t.Errorf("expected 'task failed' in error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "storage error") {
		t.Errorf("expected exit status in error, got: %s", err.Error())
	}
}

func TestWaitForTask_ContextCancelled(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(TaskStatus{Status: "running"}))
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.WaitForTask(ctx, "UPID:pve1:slow")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestWaitForTask_PollsUntilDone(t *testing.T) {
	var callCount int32
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n < 3 {
			_, _ = w.Write(envelope(TaskStatus{Status: "running"}))
		} else {
			_, _ = w.Write(envelope(TaskStatus{Status: "stopped", ExitStatus: "OK"}))
		}
	})
	defer server.Close()

	err := client.WaitForTask(context.Background(), "UPID:pve1:poll")
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}
	if atomic.LoadInt32(&callCount) < 3 {
		t.Errorf("expected at least 3 polls, got %d", callCount)
	}
}

// --- CreateSnapshot ---

func TestCreateSnapshot_WithDescription(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, "snapname=snap1") {
			t.Errorf("expected snapname=snap1, got: %s", bodyStr)
		}
		if !strings.Contains(bodyStr, "description=test+snapshot") {
			t.Errorf("expected description in body, got: %s", bodyStr)
		}
		_, _ = w.Write(envelope("UPID:pve1:snapshot"))
	})
	defer server.Close()

	upid, err := client.CreateSnapshot(context.Background(), 100, "snap1", "test snapshot")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if upid == "" {
		t.Error("expected non-empty UPID")
	}
}

func TestCreateSnapshot_WithoutDescription(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if strings.Contains(bodyStr, "description") {
			t.Errorf("expected no description, got: %s", bodyStr)
		}
		_, _ = w.Write(envelope("UPID:pve1:snapshot"))
	})
	defer server.Close()

	_, err := client.CreateSnapshot(context.Background(), 100, "snap1", "")
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
}

func TestCreateSnapshot_SyncReturn(t *testing.T) {
	// Proxmox sometimes returns null data for synchronous snapshot completion
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(nil))
	})
	defer server.Close()

	upid, err := client.CreateSnapshot(context.Background(), 100, "snap1", "")
	if err != nil {
		t.Fatalf("CreateSnapshot sync: %v", err)
	}
	// Should return empty string without error for sync completion
	if upid != "" {
		t.Errorf("expected empty UPID for sync snapshot, got %s", upid)
	}
}

// --- GetGuestAgentInterfaces ---

func TestGetGuestAgentInterfaces_WrappedResult(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/qemu/100/agent/network-get-interfaces" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		ifaces := struct {
			Result []NetworkInterface `json:"result"`
		}{
			Result: []NetworkInterface{
				{
					Name:            "eth0",
					HardwareAddress: "AA:BB:CC:DD:EE:FF",
					IPAddresses: []GuestIPAddress{
						{IPAddressType: "ipv4", IPAddress: "192.168.1.100", Prefix: 24},
						{IPAddressType: "ipv6", IPAddress: "fe80::1", Prefix: 64},
					},
				},
				{
					Name:            "lo",
					HardwareAddress: "00:00:00:00:00:00",
					IPAddresses: []GuestIPAddress{
						{IPAddressType: "ipv4", IPAddress: "127.0.0.1", Prefix: 8},
					},
				},
			},
		}
		_, _ = w.Write(envelope(ifaces))
	})
	defer server.Close()

	ifaces, err := client.GetGuestAgentInterfaces(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetGuestAgentInterfaces: %v", err)
	}
	if len(ifaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(ifaces))
	}
	if ifaces[0].Name != "eth0" {
		t.Errorf("expected eth0, got %s", ifaces[0].Name)
	}
	if len(ifaces[0].IPAddresses) != 2 {
		t.Errorf("expected 2 IPs on eth0, got %d", len(ifaces[0].IPAddresses))
	}
	if ifaces[0].IPAddresses[1].IPAddressType != "ipv6" {
		t.Errorf("expected ipv6, got %s", ifaces[0].IPAddresses[1].IPAddressType)
	}
}

func TestGetGuestAgentInterfaces_DirectArray(t *testing.T) {
	// Some Proxmox versions may return the array directly without wrapping
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		ifaces := []NetworkInterface{
			{
				Name:            "ens18",
				HardwareAddress: "11:22:33:44:55:66",
				IPAddresses: []GuestIPAddress{
					{IPAddressType: "ipv4", IPAddress: "10.0.0.5", Prefix: 24},
				},
			},
		}
		_, _ = w.Write(envelope(ifaces))
	})
	defer server.Close()

	ifaces, err := client.GetGuestAgentInterfaces(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetGuestAgentInterfaces direct: %v", err)
	}
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}
	if ifaces[0].Name != "ens18" {
		t.Errorf("expected ens18, got %s", ifaces[0].Name)
	}
}

func TestGetGuestAgentInterfaces_Empty(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope(struct {
			Result []NetworkInterface `json:"result"`
		}{}))
	})
	defer server.Close()

	ifaces, err := client.GetGuestAgentInterfaces(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetGuestAgentInterfaces: %v", err)
	}
	if len(ifaces) != 0 {
		t.Errorf("expected 0 interfaces, got %d", len(ifaces))
	}
}

// --- GetNodeStatus ---

func TestGetNodeStatus(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/nodes/pve1/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		status := NodeStatus{
			CPU:    0.25,
			MaxCPU: 8,
			Memory: MemoryStatus{
				Total: 16 * 1024 * 1024 * 1024,
				Used:  8 * 1024 * 1024 * 1024,
				Free:  8 * 1024 * 1024 * 1024,
			},
			RootFS: DiskStatus{
				Total:     100 * 1024 * 1024 * 1024,
				Used:      30 * 1024 * 1024 * 1024,
				Available: 70 * 1024 * 1024 * 1024,
			},
			Uptime:   86400,
			KVersion: "6.1.0-amd64",
		}
		_, _ = w.Write(envelope(status))
	})
	defer server.Close()

	status, err := client.GetNodeStatus(context.Background())
	if err != nil {
		t.Fatalf("GetNodeStatus: %v", err)
	}
	if status.MaxCPU != 8 {
		t.Errorf("expected 8 CPUs, got %d", status.MaxCPU)
	}
	if status.Uptime != 86400 {
		t.Errorf("expected uptime 86400, got %d", status.Uptime)
	}
	if status.KVersion != "6.1.0-amd64" {
		t.Errorf("expected kversion, got %s", status.KVersion)
	}
	if status.Memory.Free != 8*1024*1024*1024 {
		t.Errorf("unexpected free memory: %d", status.Memory.Free)
	}
	if status.RootFS.Available != 70*1024*1024*1024 {
		t.Errorf("unexpected available disk: %d", status.RootFS.Available)
	}
}

// --- NextVMID ---

func TestNextVMID_Gap(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 9000, Name: "vm1"},
			{VMID: 9001, Name: "vm2"},
			{VMID: 9003, Name: "vm3"},
		}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	id, err := client.NextVMID(context.Background(), 9000, 9999)
	if err != nil {
		t.Fatalf("NextVMID: %v", err)
	}
	if id != 9002 {
		t.Errorf("expected 9002, got %d", id)
	}
}

func TestNextVMID_FirstAvailable(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(envelope([]VMListEntry{}))
	})
	defer server.Close()

	id, err := client.NextVMID(context.Background(), 9000, 9999)
	if err != nil {
		t.Fatalf("NextVMID: %v", err)
	}
	if id != 9000 {
		t.Errorf("expected 9000, got %d", id)
	}
}

func TestNextVMID_AllUsed(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 9000}, {VMID: 9001}, {VMID: 9002},
		}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	_, err := client.NextVMID(context.Background(), 9000, 9002)
	if err == nil {
		t.Error("expected error when all VMIDs used")
	}
	if !strings.Contains(err.Error(), "no available VMID") {
		t.Errorf("expected 'no available VMID', got: %s", err.Error())
	}
}

func TestNextVMID_OutOfRangeVMsIgnored(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		vms := []VMListEntry{
			{VMID: 100, Name: "outside-range"},
			{VMID: 200, Name: "also-outside"},
		}
		_, _ = w.Write(envelope(vms))
	})
	defer server.Close()

	id, err := client.NextVMID(context.Background(), 9000, 9999)
	if err != nil {
		t.Fatalf("NextVMID: %v", err)
	}
	if id != 9000 {
		t.Errorf("expected 9000 (out-of-range VMs should not affect range), got %d", id)
	}
}

// --- ResizeVM ---

func TestResizeVM(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		bodyStr := string(body)
		if !strings.Contains(bodyStr, "cores=4") {
			t.Errorf("expected cores=4, got: %s", bodyStr)
		}
		if !strings.Contains(bodyStr, "memory=8192") {
			t.Errorf("expected memory=8192, got: %s", bodyStr)
		}
		_, _ = w.Write(envelope(nil))
	})
	defer server.Close()

	err := client.ResizeVM(context.Background(), 100, 4, 8192)
	if err != nil {
		t.Fatalf("ResizeVM: %v", err)
	}
}

// --- HTTP Error handling ---

func TestAPIError_Forbidden(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":{"token":"invalid"}}`))
	})
	defer server.Close()

	_, err := client.ListVMs(context.Background())
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected 403 in error, got: %s", err.Error())
	}
}

func TestAPIError_NotFound(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":{"vmid":"does not exist"}}`))
	})
	defer server.Close()

	_, err := client.GetVMStatus(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %s", err.Error())
	}
}

func TestAPIError_InternalServerError(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	})
	defer server.Close()

	_, err := client.GetNodeStatus(context.Background())
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got: %s", err.Error())
	}
}

func TestAPIError_Unauthorized(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"data":null}`))
	})
	defer server.Close()

	_, err := client.ListVMs(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %s", err.Error())
	}
}

func TestAPIError_InvalidJSON(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json at all"))
	})
	defer server.Close()

	_, err := client.ListVMs(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("expected unmarshal error, got: %s", err.Error())
	}
}

func TestAPIError_ConnectionRefused(t *testing.T) {
	cfg := Config{
		Host:      "http://127.0.0.1:1", // port 1 should refuse connections
		TokenID:   "test@pam!test",
		Secret:    "secret",
		Node:      "pve1",
		VMIDStart: 9000,
		VMIDEnd:   9999,
	}
	client := NewClient(cfg, nil)

	_, err := client.ListVMs(context.Background())
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestAPIError_ContextCancelled(t *testing.T) {
	client, server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		_, _ = w.Write(envelope([]VMListEntry{}))
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.ListVMs(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- NewClient ---

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	cfg := Config{
		Host:    "https://pve.example.com:8006/",
		TokenID: "test",
		Secret:  "secret",
		Node:    "pve1",
	}
	client := NewClient(cfg, nil)
	if strings.HasSuffix(client.baseURL, "/") {
		t.Errorf("expected trailing slash trimmed, got %s", client.baseURL)
	}
}

func TestNewClient_NilLogger(t *testing.T) {
	cfg := Config{
		Host:    "https://pve.example.com:8006",
		TokenID: "test",
		Secret:  "secret",
		Node:    "pve1",
	}
	client := NewClient(cfg, nil)
	if client.logger == nil {
		t.Error("expected default logger, got nil")
	}
}
