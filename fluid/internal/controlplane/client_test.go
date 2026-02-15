package controlplane

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestServer creates an httptest server that validates method and path,
// then returns the given status code and JSON-encoded response.
func newTestServer(t *testing.T, method, path string, status int, response any) (*httptest.Server, *Client) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			t.Errorf("expected method %s, got %s", method, r.Method)
		}
		if r.URL.Path != path {
			t.Errorf("expected path %s, got %s", path, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if response != nil {
			_ = json.NewEncoder(w).Encode(response)
		}
	}))
	client := NewClient(ts.URL)
	return ts, client
}

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:9090")
	if c.baseURL != "http://localhost:9090" {
		t.Errorf("expected baseURL http://localhost:9090, got %s", c.baseURL)
	}
	if c.httpClient == nil {
		t.Fatal("expected httpClient to be non-nil")
	}
	if c.httpClient.Timeout != 10*time.Minute {
		t.Errorf("expected timeout 10m, got %v", c.httpClient.Timeout)
	}
}

func TestCreateSandbox(t *testing.T) {
	want := SandboxResponse{
		ID:       "SBX-abc123",
		Name:     "sbx-test",
		AgentID:  "agent-1",
		SourceVM: "ubuntu-base",
		State:    "RUNNING",
		VCPUs:    2,
		MemoryMB: 2048,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sandboxes" {
			t.Errorf("expected path /v1/sandboxes, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var req CreateSandboxRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if req.AgentID != "agent-1" {
			t.Errorf("expected agent_id agent-1, got %s", req.AgentID)
		}
		if req.SourceVM != "ubuntu-base" {
			t.Errorf("expected source_vm ubuntu-base, got %s", req.SourceVM)
		}
		if req.VCPUs != 2 {
			t.Errorf("expected vcpus 2, got %d", req.VCPUs)
		}
		if req.MemoryMB != 2048 {
			t.Errorf("expected memory_mb 2048, got %d", req.MemoryMB)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	got, err := client.CreateSandbox(context.Background(), CreateSandboxRequest{
		AgentID:  "agent-1",
		SourceVM: "ubuntu-base",
		VCPUs:    2,
		MemoryMB: 2048,
	})
	if err != nil {
		t.Fatalf("CreateSandbox returned error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected ID %s, got %s", want.ID, got.ID)
	}
	if got.Name != want.Name {
		t.Errorf("expected Name %s, got %s", want.Name, got.Name)
	}
	if got.State != want.State {
		t.Errorf("expected State %s, got %s", want.State, got.State)
	}
	if got.SourceVM != want.SourceVM {
		t.Errorf("expected SourceVM %s, got %s", want.SourceVM, got.SourceVM)
	}
}

func TestGetSandbox(t *testing.T) {
	want := SandboxResponse{
		ID:        "SBX-abc123",
		Name:      "sbx-test",
		State:     "RUNNING",
		IPAddress: "192.168.122.45",
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/sandboxes/SBX-abc123", http.StatusOK, want)
	defer ts.Close()

	got, err := client.GetSandbox(context.Background(), "SBX-abc123")
	if err != nil {
		t.Fatalf("GetSandbox returned error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected ID %s, got %s", want.ID, got.ID)
	}
	if got.IPAddress != want.IPAddress {
		t.Errorf("expected IPAddress %s, got %s", want.IPAddress, got.IPAddress)
	}
	if got.State != want.State {
		t.Errorf("expected State %s, got %s", want.State, got.State)
	}
}

func TestListSandboxes(t *testing.T) {
	resp := struct {
		Sandboxes []*SandboxResponse `json:"sandboxes"`
		Count     int                `json:"count"`
	}{
		Sandboxes: []*SandboxResponse{
			{ID: "SBX-1", Name: "sbx-one", State: "RUNNING"},
			{ID: "SBX-2", Name: "sbx-two", State: "STOPPED"},
		},
		Count: 2,
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/sandboxes", http.StatusOK, resp)
	defer ts.Close()

	got, err := client.ListSandboxes(context.Background())
	if err != nil {
		t.Fatalf("ListSandboxes returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sandboxes, got %d", len(got))
	}
	if got[0].ID != "SBX-1" {
		t.Errorf("expected first sandbox ID SBX-1, got %s", got[0].ID)
	}
	if got[1].ID != "SBX-2" {
		t.Errorf("expected second sandbox ID SBX-2, got %s", got[1].ID)
	}
	if got[1].State != "STOPPED" {
		t.Errorf("expected second sandbox state STOPPED, got %s", got[1].State)
	}
}

func TestListSandboxes_Empty(t *testing.T) {
	resp := struct {
		Sandboxes []*SandboxResponse `json:"sandboxes"`
		Count     int                `json:"count"`
	}{
		Sandboxes: []*SandboxResponse{},
		Count:     0,
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/sandboxes", http.StatusOK, resp)
	defer ts.Close()

	got, err := client.ListSandboxes(context.Background())
	if err != nil {
		t.Fatalf("ListSandboxes returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 sandboxes, got %d", len(got))
	}
}

func TestDestroySandbox(t *testing.T) {
	ts, client := newTestServer(t, http.MethodDelete, "/v1/sandboxes/SBX-abc123", http.StatusOK, nil)
	defer ts.Close()

	err := client.DestroySandbox(context.Background(), "SBX-abc123")
	if err != nil {
		t.Fatalf("DestroySandbox returned error: %v", err)
	}
}

func TestStartSandbox(t *testing.T) {
	ts, client := newTestServer(t, http.MethodPost, "/v1/sandboxes/SBX-abc123/start", http.StatusOK, nil)
	defer ts.Close()

	err := client.StartSandbox(context.Background(), "SBX-abc123")
	if err != nil {
		t.Fatalf("StartSandbox returned error: %v", err)
	}
}

func TestStopSandbox(t *testing.T) {
	ts, client := newTestServer(t, http.MethodPost, "/v1/sandboxes/SBX-abc123/stop", http.StatusOK, nil)
	defer ts.Close()

	err := client.StopSandbox(context.Background(), "SBX-abc123")
	if err != nil {
		t.Fatalf("StopSandbox returned error: %v", err)
	}
}

func TestRunCommand(t *testing.T) {
	want := CommandResponse{
		ID:         "CMD-xyz",
		SandboxID:  "SBX-abc123",
		Command:    "whoami",
		Stdout:     "root\n",
		Stderr:     "",
		ExitCode:   0,
		DurationMS: 42,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sandboxes/SBX-abc123/run" {
			t.Errorf("expected path /v1/sandboxes/SBX-abc123/run, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var req RunCommandRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if req.Command != "whoami" {
			t.Errorf("expected command whoami, got %s", req.Command)
		}
		if req.TimeoutSec != 30 {
			t.Errorf("expected timeout_seconds 30, got %d", req.TimeoutSec)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	got, err := client.RunCommand(context.Background(), "SBX-abc123", "whoami", 30)
	if err != nil {
		t.Fatalf("RunCommand returned error: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("expected ID %s, got %s", want.ID, got.ID)
	}
	if got.Stdout != want.Stdout {
		t.Errorf("expected Stdout %q, got %q", want.Stdout, got.Stdout)
	}
	if got.ExitCode != want.ExitCode {
		t.Errorf("expected ExitCode %d, got %d", want.ExitCode, got.ExitCode)
	}
	if got.Command != want.Command {
		t.Errorf("expected Command %s, got %s", want.Command, got.Command)
	}
}

func TestGetSandboxIP(t *testing.T) {
	resp := struct {
		SandboxID string `json:"sandbox_id"`
		IPAddress string `json:"ip_address"`
	}{
		SandboxID: "SBX-abc123",
		IPAddress: "192.168.122.45",
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/sandboxes/SBX-abc123/ip", http.StatusOK, resp)
	defer ts.Close()

	got, err := client.GetSandboxIP(context.Background(), "SBX-abc123")
	if err != nil {
		t.Fatalf("GetSandboxIP returned error: %v", err)
	}
	if got != "192.168.122.45" {
		t.Errorf("expected IP 192.168.122.45, got %s", got)
	}
}

func TestCreateSnapshot(t *testing.T) {
	want := SnapshotResponse{
		SnapshotID:   "SNP-xyz",
		SandboxID:    "SBX-abc123",
		SnapshotName: "after-nginx",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sandboxes/SBX-abc123/snapshot" {
			t.Errorf("expected path /v1/sandboxes/SBX-abc123/snapshot, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if req.Name != "after-nginx" {
			t.Errorf("expected snapshot name after-nginx, got %s", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	got, err := client.CreateSnapshot(context.Background(), "SBX-abc123", "after-nginx")
	if err != nil {
		t.Fatalf("CreateSnapshot returned error: %v", err)
	}
	if got.SnapshotID != want.SnapshotID {
		t.Errorf("expected SnapshotID %s, got %s", want.SnapshotID, got.SnapshotID)
	}
	if got.SandboxID != want.SandboxID {
		t.Errorf("expected SandboxID %s, got %s", want.SandboxID, got.SandboxID)
	}
	if got.SnapshotName != want.SnapshotName {
		t.Errorf("expected SnapshotName %s, got %s", want.SnapshotName, got.SnapshotName)
	}
}

func TestListCommands(t *testing.T) {
	resp := struct {
		Commands []*CommandResponse `json:"commands"`
		Count    int                `json:"count"`
	}{
		Commands: []*CommandResponse{
			{ID: "CMD-1", SandboxID: "SBX-abc123", Command: "whoami", ExitCode: 0},
			{ID: "CMD-2", SandboxID: "SBX-abc123", Command: "ls -la", ExitCode: 0},
		},
		Count: 2,
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/sandboxes/SBX-abc123/commands", http.StatusOK, resp)
	defer ts.Close()

	got, err := client.ListCommands(context.Background(), "SBX-abc123")
	if err != nil {
		t.Fatalf("ListCommands returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(got))
	}
	if got[0].ID != "CMD-1" {
		t.Errorf("expected first command ID CMD-1, got %s", got[0].ID)
	}
	if got[1].Command != "ls -la" {
		t.Errorf("expected second command 'ls -la', got %s", got[1].Command)
	}
}

func TestListHosts(t *testing.T) {
	resp := struct {
		Hosts []*HostInfo `json:"hosts"`
		Count int         `json:"count"`
	}{
		Hosts: []*HostInfo{
			{
				HostID:          "host-1",
				Hostname:        "kvm-node-1",
				Status:          "online",
				ActiveSandboxes: 3,
				AvailableCPUs:   16,
				AvailableMemMB:  32768,
				AvailableDiskMB: 500000,
				BaseImages:      []string{"ubuntu-22.04", "debian-12"},
			},
		},
		Count: 1,
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/hosts", http.StatusOK, resp)
	defer ts.Close()

	got, err := client.ListHosts(context.Background())
	if err != nil {
		t.Fatalf("ListHosts returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 host, got %d", len(got))
	}
	if got[0].HostID != "host-1" {
		t.Errorf("expected HostID host-1, got %s", got[0].HostID)
	}
	if got[0].Hostname != "kvm-node-1" {
		t.Errorf("expected Hostname kvm-node-1, got %s", got[0].Hostname)
	}
	if got[0].ActiveSandboxes != 3 {
		t.Errorf("expected ActiveSandboxes 3, got %d", got[0].ActiveSandboxes)
	}
	if len(got[0].BaseImages) != 2 {
		t.Errorf("expected 2 base images, got %d", len(got[0].BaseImages))
	}
}

func TestGetHost(t *testing.T) {
	want := HostInfo{
		HostID:          "host-1",
		Hostname:        "kvm-node-1",
		Status:          "online",
		ActiveSandboxes: 5,
		AvailableCPUs:   8,
		AvailableMemMB:  16384,
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/hosts/host-1", http.StatusOK, want)
	defer ts.Close()

	got, err := client.GetHost(context.Background(), "host-1")
	if err != nil {
		t.Fatalf("GetHost returned error: %v", err)
	}
	if got.HostID != want.HostID {
		t.Errorf("expected HostID %s, got %s", want.HostID, got.HostID)
	}
	if got.Hostname != want.Hostname {
		t.Errorf("expected Hostname %s, got %s", want.Hostname, got.Hostname)
	}
	if got.AvailableCPUs != want.AvailableCPUs {
		t.Errorf("expected AvailableCPUs %d, got %d", want.AvailableCPUs, got.AvailableCPUs)
	}
}

func TestListVMs(t *testing.T) {
	resp := struct {
		VMs   []*VMInfo `json:"vms"`
		Count int       `json:"count"`
	}{
		VMs: []*VMInfo{
			{Name: "ubuntu-base", State: "running", IPAddress: "192.168.122.10", Prepared: true, HostID: "host-1"},
			{Name: "debian-base", State: "shutoff", Prepared: false, HostID: "host-1"},
		},
		Count: 2,
	}
	ts, client := newTestServer(t, http.MethodGet, "/v1/vms", http.StatusOK, resp)
	defer ts.Close()

	got, err := client.ListVMs(context.Background())
	if err != nil {
		t.Fatalf("ListVMs returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 VMs, got %d", len(got))
	}
	if got[0].Name != "ubuntu-base" {
		t.Errorf("expected first VM name ubuntu-base, got %s", got[0].Name)
	}
	if !got[0].Prepared {
		t.Error("expected first VM to be prepared")
	}
	if got[1].IPAddress != "" {
		t.Errorf("expected second VM IPAddress to be empty, got %s", got[1].IPAddress)
	}
}

func TestRunSourceCommand(t *testing.T) {
	want := SourceCommandResult{
		SourceVM: "ubuntu-base",
		ExitCode: 0,
		Stdout:   "root\n",
		Stderr:   "",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sources/ubuntu-base/run" {
			t.Errorf("expected path /v1/sources/ubuntu-base/run, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var req struct {
			Command    string `json:"command"`
			TimeoutSec int    `json:"timeout_seconds"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if req.Command != "whoami" {
			t.Errorf("expected command whoami, got %s", req.Command)
		}
		if req.TimeoutSec != 10 {
			t.Errorf("expected timeout_seconds 10, got %d", req.TimeoutSec)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	got, err := client.RunSourceCommand(context.Background(), "ubuntu-base", "whoami", 10)
	if err != nil {
		t.Fatalf("RunSourceCommand returned error: %v", err)
	}
	if got.SourceVM != want.SourceVM {
		t.Errorf("expected SourceVM %s, got %s", want.SourceVM, got.SourceVM)
	}
	if got.Stdout != want.Stdout {
		t.Errorf("expected Stdout %q, got %q", want.Stdout, got.Stdout)
	}
	if got.ExitCode != want.ExitCode {
		t.Errorf("expected ExitCode %d, got %d", want.ExitCode, got.ExitCode)
	}
}

func TestReadSourceFile(t *testing.T) {
	want := SourceFileResult{
		SourceVM: "ubuntu-base",
		Path:     "/etc/hostname",
		Content:  "ubuntu-base\n",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sources/ubuntu-base/read" {
			t.Errorf("expected path /v1/sources/ubuntu-base/read, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if req.Path != "/etc/hostname" {
			t.Errorf("expected path /etc/hostname, got %s", req.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	got, err := client.ReadSourceFile(context.Background(), "ubuntu-base", "/etc/hostname")
	if err != nil {
		t.Fatalf("ReadSourceFile returned error: %v", err)
	}
	if got.SourceVM != want.SourceVM {
		t.Errorf("expected SourceVM %s, got %s", want.SourceVM, got.SourceVM)
	}
	if got.Path != want.Path {
		t.Errorf("expected Path %s, got %s", want.Path, got.Path)
	}
	if got.Content != want.Content {
		t.Errorf("expected Content %q, got %q", want.Content, got.Content)
	}
}

func TestPrepareSourceVM(t *testing.T) {
	wantResp := map[string]any{
		"status":  "prepared",
		"vm_name": "ubuntu-base",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sources/ubuntu-base/prepare" {
			t.Errorf("expected path /v1/sources/ubuntu-base/prepare, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		var req struct {
			SSHUser    string `json:"ssh_user"`
			SSHKeyPath string `json:"ssh_key_path"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if req.SSHUser != "sandbox" {
			t.Errorf("expected ssh_user sandbox, got %s", req.SSHUser)
		}
		if req.SSHKeyPath != "/home/user/.ssh/id_rsa" {
			t.Errorf("expected ssh_key_path /home/user/.ssh/id_rsa, got %s", req.SSHKeyPath)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(wantResp)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	got, err := client.PrepareSourceVM(context.Background(), "ubuntu-base", "sandbox", "/home/user/.ssh/id_rsa")
	if err != nil {
		t.Fatalf("PrepareSourceVM returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	// Result is decoded into any (map[string]any for JSON objects)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["status"] != "prepared" {
		t.Errorf("expected status prepared, got %v", m["status"])
	}
}

func TestValidateSourceVM(t *testing.T) {
	wantResp := map[string]any{
		"valid":   true,
		"vm_name": "ubuntu-base",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sources/ubuntu-base/validate" {
			t.Errorf("expected path /v1/sources/ubuntu-base/validate, got %s", r.URL.Path)
		}
		// ValidateSourceVM sends no body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(wantResp)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	got, err := client.ValidateSourceVM(context.Background(), "ubuntu-base")
	if err != nil {
		t.Fatalf("ValidateSourceVM returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", got)
	}
	if m["valid"] != true {
		t.Errorf("expected valid true, got %v", m["valid"])
	}
}

// ---------------------------------------------------------------------------
// Error handling tests
// ---------------------------------------------------------------------------

func TestErrorHandling_404(t *testing.T) {
	errResp := struct {
		Error string `json:"error"`
	}{Error: "sandbox not found"}

	ts, client := newTestServer(t, http.MethodGet, "/v1/sandboxes/SBX-nonexistent", http.StatusNotFound, errResp)
	defer ts.Close()

	_, err := client.GetSandbox(context.Background(), "SBX-nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	want := "control plane error (404): sandbox not found"
	if err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestErrorHandling_500(t *testing.T) {
	errResp := struct {
		Error string `json:"error"`
	}{Error: "internal server error"}

	ts, client := newTestServer(t, http.MethodGet, "/v1/sandboxes", http.StatusInternalServerError, errResp)
	defer ts.Close()

	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	want := "control plane error (500): internal server error"
	if err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestErrorHandling_400(t *testing.T) {
	errResp := struct {
		Error string `json:"error"`
	}{Error: "invalid request: source_vm is required"}

	ts, client := newTestServer(t, http.MethodPost, "/v1/sandboxes", http.StatusBadRequest, errResp)
	defer ts.Close()

	_, err := client.CreateSandbox(context.Background(), CreateSandboxRequest{})
	if err == nil {
		t.Fatal("expected error for 400 response, got nil")
	}
	want := "control plane error (400): invalid request: source_vm is required"
	if err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestErrorHandling_409(t *testing.T) {
	errResp := struct {
		Error string `json:"error"`
	}{Error: "sandbox already running"}

	ts, client := newTestServer(t, http.MethodPost, "/v1/sandboxes/SBX-abc123/start", http.StatusConflict, errResp)
	defer ts.Close()

	err := client.StartSandbox(context.Background(), "SBX-abc123")
	if err == nil {
		t.Fatal("expected error for 409 response, got nil")
	}
	want := "control plane error (409): sandbox already running"
	if err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestErrorHandling_DeleteError(t *testing.T) {
	errResp := struct {
		Error string `json:"error"`
	}{Error: "sandbox not found"}

	ts, client := newTestServer(t, http.MethodDelete, "/v1/sandboxes/SBX-gone", http.StatusNotFound, errResp)
	defer ts.Close()

	err := client.DestroySandbox(context.Background(), "SBX-gone")
	if err == nil {
		t.Fatal("expected error for 404 DELETE response, got nil")
	}
	want := "control plane error (404): sandbox not found"
	if err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestErrorHandling_NonJSONError(t *testing.T) {
	// Server returns a non-JSON error body
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("Bad Gateway"))
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.ListSandboxes(context.Background())
	if err == nil {
		t.Fatal("expected error for 502 response, got nil")
	}
	want := "control plane error (502): Bad Gateway"
	if err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}
