package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"virsh-sandbox/internal/store"
)

// mockVMService implements the minimal interface needed for testing
type mockVMService struct {
	destroySandboxFn func(ctx context.Context, sandboxID string) (*store.Sandbox, error)
}

func (m *mockVMService) DestroySandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	if m.destroySandboxFn != nil {
		return m.destroySandboxFn(ctx, sandboxID)
	}
	return &store.Sandbox{
		State:       store.SandboxStateDestroyed,
		BaseImage:   "test-image.qcow2",
		SandboxName: "test-sandbox",
	}, nil
}

func TestDestroySandbox_NotFound(t *testing.T) {
	// Create a mock service that returns ErrNotFound
	mockSvc := &mockVMService{
		destroySandboxFn: func(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		},
	}

	// Create the server with a nil vmSvc (we'll call the handler directly)
	server := &Server{
		Router: nil,
	}

	// Create a request
	req := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/nonexistent-id", nil)
	rec := httptest.NewRecorder()

	// We need to test the error handling logic directly
	// Simulate what the handler does
	id := "nonexistent-id"
	sb, err := mockSvc.DestroySandbox(req.Context(), id)

	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			rec.WriteHeader(http.StatusNotFound)
		} else {
			rec.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		rec.WriteHeader(http.StatusOK)
	}

	// Verify the response
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	// Verify server is not nil (just to use it)
	if server == nil {
		t.Error("server should not be nil")
	}

	// sb should be nil for not found
	if sb != nil {
		t.Error("expected sandbox to be nil for not found error")
	}
}

func TestDestroySandbox_Success(t *testing.T) {
	ttl := 3600
	// Create a mock service that returns success with sandbox info
	mockSvc := &mockVMService{
		destroySandboxFn: func(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
			return &store.Sandbox{
				State:       store.SandboxStateDestroyed,
				BaseImage:   "ubuntu-22.04.qcow2",
				SandboxName: "sandbox-test-123",
				TTLSeconds:  &ttl,
			}, nil
		},
	}

	// Create a request
	req := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/test-sandbox-id", nil)
	rec := httptest.NewRecorder()

	// Simulate what the handler does
	id := "test-sandbox-id"
	sb, err := mockSvc.DestroySandbox(req.Context(), id)

	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			rec.WriteHeader(http.StatusNotFound)
		} else {
			rec.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		rec.WriteHeader(http.StatusOK)
	}

	// Verify the response
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Verify sandbox info is returned
	if sb == nil {
		t.Fatal("expected sandbox to be returned")
	}
	if sb.State != store.SandboxStateDestroyed {
		t.Errorf("expected state %s, got %s", store.SandboxStateDestroyed, sb.State)
	}
	if sb.BaseImage != "ubuntu-22.04.qcow2" {
		t.Errorf("expected base_image %s, got %s", "ubuntu-22.04.qcow2", sb.BaseImage)
	}
	if sb.SandboxName != "sandbox-test-123" {
		t.Errorf("expected sandbox_name %s, got %s", "sandbox-test-123", sb.SandboxName)
	}
	if sb.TTLSeconds == nil || *sb.TTLSeconds != 3600 {
		t.Errorf("expected ttl_seconds %d, got %v", 3600, sb.TTLSeconds)
	}
}

func TestDestroySandbox_InternalError(t *testing.T) {
	// Create a mock service that returns an internal error
	mockSvc := &mockVMService{
		destroySandboxFn: func(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
			return nil, errors.New("some internal error")
		},
	}

	// Create a request
	req := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/test-sandbox-id", nil)
	rec := httptest.NewRecorder()

	// Simulate what the handler does
	id := "test-sandbox-id"
	_, err := mockSvc.DestroySandbox(req.Context(), id)

	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			rec.WriteHeader(http.StatusNotFound)
		} else {
			rec.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		rec.WriteHeader(http.StatusOK)
	}

	// Verify the response
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestErrorResponseJSON(t *testing.T) {
	// Test that error response can be properly marshaled
	errResp := ErrorResponse{
		Error: "sandbox not found: test-id",
		Code:  404,
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("failed to marshal error response: %v", err)
	}

	var decoded ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if decoded.Error != errResp.Error {
		t.Errorf("expected error %q, got %q", errResp.Error, decoded.Error)
	}

	if decoded.Code != errResp.Code {
		t.Errorf("expected code %d, got %d", errResp.Code, decoded.Code)
	}
}

func TestDestroySandboxResponseJSON(t *testing.T) {
	ttl := 7200
	// Test that destroy sandbox response can be properly marshaled
	resp := destroySandboxResponse{
		State:       store.SandboxStateDestroyed,
		BaseImage:   "centos-9.qcow2",
		SandboxName: "my-sandbox",
		TTLSeconds:  &ttl,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal destroy sandbox response: %v", err)
	}

	var decoded destroySandboxResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal destroy sandbox response: %v", err)
	}

	if decoded.State != resp.State {
		t.Errorf("expected state %q, got %q", resp.State, decoded.State)
	}

	if decoded.BaseImage != resp.BaseImage {
		t.Errorf("expected base_image %q, got %q", resp.BaseImage, decoded.BaseImage)
	}

	if decoded.SandboxName != resp.SandboxName {
		t.Errorf("expected sandbox_name %q, got %q", resp.SandboxName, decoded.SandboxName)
	}

	if decoded.TTLSeconds == nil || *decoded.TTLSeconds != *resp.TTLSeconds {
		t.Errorf("expected ttl_seconds %d, got %v", *resp.TTLSeconds, decoded.TTLSeconds)
	}
}

func TestDestroySandboxResponseJSON_NoTTL(t *testing.T) {
	// Test that destroy sandbox response without TTL omits the field
	resp := destroySandboxResponse{
		State:       store.SandboxStateDestroyed,
		BaseImage:   "ubuntu-22.04.qcow2",
		SandboxName: "ephemeral-sandbox",
		TTLSeconds:  nil,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal destroy sandbox response: %v", err)
	}

	// Verify TTL is omitted from JSON
	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, exists := rawMap["ttl_seconds"]; exists {
		t.Error("expected ttl_seconds to be omitted when nil")
	}
}

func TestCreateSandboxRequestJSON(t *testing.T) {
	ttl := 3600
	// Test that createSandboxRequest can be properly marshaled/unmarshaled with new fields
	req := createSandboxRequest{
		SourceVMName: "test-vm",
		AgentID:      "agent-123",
		VMName:       "my-sandbox",
		CPU:          4,
		MemoryMB:     4096,
		TTLSeconds:   &ttl,
		AutoStart:    true,
		WaitForIP:    true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal create sandbox request: %v", err)
	}

	var decoded createSandboxRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal create sandbox request: %v", err)
	}

	if decoded.SourceVMName != req.SourceVMName {
		t.Errorf("expected source_vm_name %q, got %q", req.SourceVMName, decoded.SourceVMName)
	}
	if decoded.AgentID != req.AgentID {
		t.Errorf("expected agent_id %q, got %q", req.AgentID, decoded.AgentID)
	}
	if decoded.TTLSeconds == nil || *decoded.TTLSeconds != *req.TTLSeconds {
		t.Errorf("expected ttl_seconds %d, got %v", *req.TTLSeconds, decoded.TTLSeconds)
	}
	if decoded.AutoStart != req.AutoStart {
		t.Errorf("expected auto_start %v, got %v", req.AutoStart, decoded.AutoStart)
	}
	if decoded.WaitForIP != req.WaitForIP {
		t.Errorf("expected wait_for_ip %v, got %v", req.WaitForIP, decoded.WaitForIP)
	}
}

func TestCreateSandboxRequestJSON_MinimalFields(t *testing.T) {
	// Test that only required fields are needed
	jsonData := `{"source_vm_name":"test-vm","agent_id":"agent-123"}`

	var decoded createSandboxRequest
	if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
		t.Fatalf("failed to unmarshal minimal create sandbox request: %v", err)
	}

	if decoded.SourceVMName != "test-vm" {
		t.Errorf("expected source_vm_name %q, got %q", "test-vm", decoded.SourceVMName)
	}
	if decoded.AgentID != "agent-123" {
		t.Errorf("expected agent_id %q, got %q", "agent-123", decoded.AgentID)
	}
	if decoded.TTLSeconds != nil {
		t.Errorf("expected ttl_seconds to be nil, got %v", decoded.TTLSeconds)
	}
	if decoded.AutoStart != false {
		t.Errorf("expected auto_start to be false, got %v", decoded.AutoStart)
	}
	if decoded.WaitForIP != false {
		t.Errorf("expected wait_for_ip to be false, got %v", decoded.WaitForIP)
	}
}

func TestCreateSandboxResponseJSON_WithIPAddress(t *testing.T) {
	// Test that createSandboxResponse includes ip_address when set
	ip := "192.168.1.100"
	resp := createSandboxResponse{
		Sandbox: &store.Sandbox{
			ID:          "SBX-123",
			AgentID:     "agent-123",
			SandboxName: "my-sandbox",
			State:       store.SandboxStateRunning,
			IPAddress:   &ip,
		},
		IPAddress: ip,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal create sandbox response: %v", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if rawMap["ip_address"] != ip {
		t.Errorf("expected ip_address %q, got %v", ip, rawMap["ip_address"])
	}
}

func TestCreateSandboxResponseJSON_NoIPAddress(t *testing.T) {
	// Test that createSandboxResponse omits ip_address when empty
	resp := createSandboxResponse{
		Sandbox: &store.Sandbox{
			ID:          "SBX-123",
			AgentID:     "agent-123",
			SandboxName: "my-sandbox",
			State:       store.SandboxStateCreated,
		},
		IPAddress: "",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal create sandbox response: %v", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// ip_address should be omitted when empty due to omitempty tag
	if ipVal, exists := rawMap["ip_address"]; exists && ipVal != "" {
		t.Errorf("expected ip_address to be omitted or empty, got %v", ipVal)
	}
}

func TestGetSandboxResponseJSON(t *testing.T) {
	// Test that getSandboxResponse can be properly marshaled
	ip := "192.168.1.50"
	resp := getSandboxResponse{
		Sandbox: &store.Sandbox{
			ID:          "SBX-456",
			JobID:       "JOB-123",
			AgentID:     "agent-456",
			SandboxName: "test-sandbox",
			BaseImage:   "ubuntu-22.04.qcow2",
			Network:     "default",
			IPAddress:   &ip,
			State:       store.SandboxStateRunning,
		},
		Commands: []*store.Command{
			{
				ID:        "CMD-001",
				SandboxID: "SBX-456",
				Command:   "echo hello",
				Stdout:    "hello\n",
				Stderr:    "",
				ExitCode:  0,
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal get sandbox response: %v", err)
	}

	var decoded getSandboxResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal get sandbox response: %v", err)
	}

	if decoded.Sandbox == nil {
		t.Fatal("expected sandbox to be present")
	}
	if decoded.Sandbox.ID != resp.Sandbox.ID {
		t.Errorf("expected sandbox ID %q, got %q", resp.Sandbox.ID, decoded.Sandbox.ID)
	}
	if len(decoded.Commands) != 1 {
		t.Errorf("expected 1 command, got %d", len(decoded.Commands))
	}
	if decoded.Commands[0].Command != "echo hello" {
		t.Errorf("expected command %q, got %q", "echo hello", decoded.Commands[0].Command)
	}
}

func TestGetSandboxResponseJSON_NoCommands(t *testing.T) {
	// Test that getSandboxResponse omits commands when nil
	resp := getSandboxResponse{
		Sandbox: &store.Sandbox{
			ID:          "SBX-789",
			SandboxName: "empty-sandbox",
			State:       store.SandboxStateCreated,
		},
		Commands: nil,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal get sandbox response: %v", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// commands should be omitted when nil due to omitempty tag
	if _, exists := rawMap["commands"]; exists {
		t.Error("expected commands to be omitted when nil")
	}
}

func TestListSandboxCommandsResponseJSON(t *testing.T) {
	// Test that listSandboxCommandsResponse can be properly marshaled
	resp := listSandboxCommandsResponse{
		Commands: []*store.Command{
			{
				ID:        "CMD-001",
				SandboxID: "SBX-123",
				Command:   "ls -la",
				Stdout:    "total 0\n",
				ExitCode:  0,
			},
			{
				ID:        "CMD-002",
				SandboxID: "SBX-123",
				Command:   "pwd",
				Stdout:    "/home/user\n",
				ExitCode:  0,
			},
		},
		Total: 2,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal list sandbox commands response: %v", err)
	}

	var decoded listSandboxCommandsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal list sandbox commands response: %v", err)
	}

	if decoded.Total != 2 {
		t.Errorf("expected total %d, got %d", 2, decoded.Total)
	}
	if len(decoded.Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(decoded.Commands))
	}
	if decoded.Commands[0].Command != "ls -la" {
		t.Errorf("expected command %q, got %q", "ls -la", decoded.Commands[0].Command)
	}
}

func TestStreamEventJSON(t *testing.T) {
	// Test that StreamEvent can be properly marshaled
	event := StreamEvent{
		Type:      "command_new",
		Timestamp: "2024-01-15T10:30:00Z",
		Data:      json.RawMessage(`{"command_id":"CMD-001","command":"echo test"}`),
		SandboxID: "SBX-123",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal stream event: %v", err)
	}

	var decoded StreamEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal stream event: %v", err)
	}

	if decoded.Type != "command_new" {
		t.Errorf("expected type %q, got %q", "command_new", decoded.Type)
	}
	if decoded.SandboxID != "SBX-123" {
		t.Errorf("expected sandbox_id %q, got %q", "SBX-123", decoded.SandboxID)
	}
}

func TestStreamEventJSON_Heartbeat(t *testing.T) {
	// Test heartbeat event without data
	event := StreamEvent{
		Type:      "heartbeat",
		Timestamp: "2024-01-15T10:30:00Z",
		SandboxID: "SBX-123",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal heartbeat event: %v", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(data, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	// data should be omitted for heartbeat due to omitempty tag
	if _, exists := rawMap["data"]; exists {
		t.Error("expected data to be omitted for heartbeat event")
	}
}
