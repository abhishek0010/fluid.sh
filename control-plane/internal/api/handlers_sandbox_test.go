package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aspectrr/fluid.sh/control-plane/internal/store"
)

// ---------------------------------------------------------------------------
// mockOrchestrator - shared across all test files in the api package
// ---------------------------------------------------------------------------

type mockOrchestrator struct {
	createSandboxFn    func(ctx context.Context, req CreateSandboxRequest) (*store.Sandbox, error)
	getSandboxFn       func(ctx context.Context, id string) (*store.Sandbox, error)
	listSandboxesFn    func(ctx context.Context) ([]*store.Sandbox, error)
	destroySandboxFn   func(ctx context.Context, id string) error
	startSandboxFn     func(ctx context.Context, id string) error
	stopSandboxFn      func(ctx context.Context, id string) error
	runCommandFn       func(ctx context.Context, sandboxID, command string, timeoutSec int) (*store.Command, error)
	createSnapshotFn   func(ctx context.Context, sandboxID, name string) (*SnapshotResponse, error)
	listCommandsFn     func(ctx context.Context, sandboxID string) ([]*store.Command, error)
	listHostsFn        func(ctx context.Context) ([]*HostInfo, error)
	getHostFn          func(ctx context.Context, id string) (*HostInfo, error)
	listVMsFn          func(ctx context.Context) ([]*VMInfo, error)
	prepareSourceVMFn  func(ctx context.Context, vmName, sshUser, keyPath string) (any, error)
	validateSourceVMFn func(ctx context.Context, vmName string) (any, error)
	runSourceCommandFn func(ctx context.Context, vmName, command string, timeoutSec int) (*SourceCommandResult, error)
	readSourceFileFn   func(ctx context.Context, vmName, path string) (*SourceFileResult, error)
}

func (m *mockOrchestrator) CreateSandbox(ctx context.Context, req CreateSandboxRequest) (*store.Sandbox, error) {
	if m.createSandboxFn != nil {
		return m.createSandboxFn(ctx, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) GetSandbox(ctx context.Context, id string) (*store.Sandbox, error) {
	if m.getSandboxFn != nil {
		return m.getSandboxFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) ListSandboxes(ctx context.Context) ([]*store.Sandbox, error) {
	if m.listSandboxesFn != nil {
		return m.listSandboxesFn(ctx)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) DestroySandbox(ctx context.Context, id string) error {
	if m.destroySandboxFn != nil {
		return m.destroySandboxFn(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) StartSandbox(ctx context.Context, id string) error {
	if m.startSandboxFn != nil {
		return m.startSandboxFn(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) StopSandbox(ctx context.Context, id string) error {
	if m.stopSandboxFn != nil {
		return m.stopSandboxFn(ctx, id)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) RunCommand(ctx context.Context, sandboxID, command string, timeoutSec int) (*store.Command, error) {
	if m.runCommandFn != nil {
		return m.runCommandFn(ctx, sandboxID, command, timeoutSec)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) CreateSnapshot(ctx context.Context, sandboxID, name string) (*SnapshotResponse, error) {
	if m.createSnapshotFn != nil {
		return m.createSnapshotFn(ctx, sandboxID, name)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) ListCommands(ctx context.Context, sandboxID string) ([]*store.Command, error) {
	if m.listCommandsFn != nil {
		return m.listCommandsFn(ctx, sandboxID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) ListHosts(ctx context.Context) ([]*HostInfo, error) {
	if m.listHostsFn != nil {
		return m.listHostsFn(ctx)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) GetHost(ctx context.Context, id string) (*HostInfo, error) {
	if m.getHostFn != nil {
		return m.getHostFn(ctx, id)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) ListVMs(ctx context.Context) ([]*VMInfo, error) {
	if m.listVMsFn != nil {
		return m.listVMsFn(ctx)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) PrepareSourceVM(ctx context.Context, vmName, sshUser, keyPath string) (any, error) {
	if m.prepareSourceVMFn != nil {
		return m.prepareSourceVMFn(ctx, vmName, sshUser, keyPath)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) ValidateSourceVM(ctx context.Context, vmName string) (any, error) {
	if m.validateSourceVMFn != nil {
		return m.validateSourceVMFn(ctx, vmName)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) RunSourceCommand(ctx context.Context, vmName, command string, timeoutSec int) (*SourceCommandResult, error) {
	if m.runSourceCommandFn != nil {
		return m.runSourceCommandFn(ctx, vmName, command, timeoutSec)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockOrchestrator) ReadSourceFile(ctx context.Context, vmName, path string) (*SourceFileResult, error) {
	if m.readSourceFileFn != nil {
		return m.readSourceFileFn(ctx, vmName, path)
	}
	return nil, fmt.Errorf("not implemented")
}

// ---------------------------------------------------------------------------
// Sandbox handler tests
// ---------------------------------------------------------------------------

func TestHandleCreateSandbox_Success(t *testing.T) {
	mock := &mockOrchestrator{
		createSandboxFn: func(_ context.Context, req CreateSandboxRequest) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:        "SBX-123",
				Name:      "test-sandbox",
				SourceVM:  req.SourceVM,
				State:     store.SandboxStateRunning,
				IPAddress: "192.168.1.10",
			}, nil
		},
	}

	s := NewServer(mock, nil)
	body := `{"source_vm":"ubuntu-base","agent_id":"agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp store.Sandbox
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != "SBX-123" {
		t.Fatalf("expected sandbox ID SBX-123, got %s", resp.ID)
	}
	if resp.SourceVM != "ubuntu-base" {
		t.Fatalf("expected source_vm ubuntu-base, got %s", resp.SourceVM)
	}
}

func TestHandleCreateSandbox_MissingFields(t *testing.T) {
	mock := &mockOrchestrator{}
	s := NewServer(mock, nil)

	body := `{"agent_id":"agent-1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatal("expected error message in response")
	}
}

func TestHandleCreateSandbox_InvalidJSON(t *testing.T) {
	mock := &mockOrchestrator{}
	s := NewServer(mock, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewBufferString(`{invalid`))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleListSandboxes(t *testing.T) {
	mock := &mockOrchestrator{
		listSandboxesFn: func(_ context.Context) ([]*store.Sandbox, error) {
			return []*store.Sandbox{
				{ID: "SBX-1", Name: "sb-1", State: store.SandboxStateRunning},
				{ID: "SBX-2", Name: "sb-2", State: store.SandboxStateStopped},
			}, nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes", nil)
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var count float64
	if err := json.Unmarshal(resp["count"], &count); err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if int(count) != 2 {
		t.Fatalf("expected count 2, got %d", int(count))
	}
}

func TestHandleGetSandbox(t *testing.T) {
	mock := &mockOrchestrator{
		getSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			if id == "SBX-123" {
				return &store.Sandbox{
					ID:        "SBX-123",
					Name:      "test-sb",
					State:     store.SandboxStateRunning,
					IPAddress: "10.0.0.5",
				}, nil
			}
			return nil, fmt.Errorf("sandbox not found: %s", id)
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/SBX-123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp store.Sandbox
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ID != "SBX-123" {
		t.Fatalf("expected ID SBX-123, got %s", resp.ID)
	}
}

func TestHandleGetSandbox_NotFound(t *testing.T) {
	mock := &mockOrchestrator{
		getSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return nil, fmt.Errorf("sandbox not found: %s", id)
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/SBX-NOPE", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-NOPE")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleDestroySandbox(t *testing.T) {
	destroyed := false
	mock := &mockOrchestrator{
		destroySandboxFn: func(_ context.Context, id string) error {
			if id == "SBX-123" {
				destroyed = true
				return nil
			}
			return fmt.Errorf("not found")
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/SBX-123", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !destroyed {
		t.Fatal("expected DestroySandbox to have been called")
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["destroyed"] != true {
		t.Fatalf("expected destroyed=true, got %v", resp["destroyed"])
	}
	if resp["sandbox_id"] != "SBX-123" {
		t.Fatalf("expected sandbox_id=SBX-123, got %v", resp["sandbox_id"])
	}
}

func TestHandleRunCommand_Success(t *testing.T) {
	mock := &mockOrchestrator{
		runCommandFn: func(_ context.Context, sandboxID, command string, _ int) (*store.Command, error) {
			return &store.Command{
				ID:        "CMD-1",
				SandboxID: sandboxID,
				Command:   command,
				Stdout:    "hello\n",
				ExitCode:  0,
			}, nil
		},
	}

	s := NewServer(mock, nil)
	body := `{"command":"echo hello"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/SBX-123/run", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp store.Command
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Stdout != "hello\n" {
		t.Fatalf("expected stdout 'hello\\n', got %q", resp.Stdout)
	}
}

func TestHandleRunCommand_EmptyCommand(t *testing.T) {
	mock := &mockOrchestrator{}
	s := NewServer(mock, nil)

	body := `{"command":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/SBX-123/run", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleStartSandbox(t *testing.T) {
	mock := &mockOrchestrator{
		startSandboxFn: func(_ context.Context, id string) error {
			return nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/SBX-123/start", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["started"] != true {
		t.Fatalf("expected started=true, got %v", resp["started"])
	}
}

func TestHandleStopSandbox(t *testing.T) {
	mock := &mockOrchestrator{
		stopSandboxFn: func(_ context.Context, id string) error {
			return nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/SBX-123/stop", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["stopped"] != true {
		t.Fatalf("expected stopped=true, got %v", resp["stopped"])
	}
}

func TestHandleGetSandboxIP(t *testing.T) {
	mock := &mockOrchestrator{
		getSandboxFn: func(_ context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:        id,
				IPAddress: "10.0.0.42",
			}, nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/SBX-123/ip", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["ip_address"] != "10.0.0.42" {
		t.Fatalf("expected ip_address=10.0.0.42, got %v", resp["ip_address"])
	}
	if resp["sandbox_id"] != "SBX-123" {
		t.Fatalf("expected sandbox_id=SBX-123, got %v", resp["sandbox_id"])
	}
}

func TestHandleCreateSnapshot(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	mock := &mockOrchestrator{
		createSnapshotFn: func(_ context.Context, sandboxID, name string) (*SnapshotResponse, error) {
			return &SnapshotResponse{
				SnapshotID:   "SNP-1",
				SandboxID:    sandboxID,
				SnapshotName: name,
				CreatedAt:    now,
			}, nil
		},
	}

	s := NewServer(mock, nil)
	body := `{"name":"before-deploy"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/SBX-123/snapshot", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp SnapshotResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.SnapshotID != "SNP-1" {
		t.Fatalf("expected snapshot_id SNP-1, got %s", resp.SnapshotID)
	}
	if resp.SandboxID != "SBX-123" {
		t.Fatalf("expected sandbox_id SBX-123, got %s", resp.SandboxID)
	}
	if resp.SnapshotName != "before-deploy" {
		t.Fatalf("expected snapshot_name before-deploy, got %s", resp.SnapshotName)
	}
}

func TestHandleListCommands(t *testing.T) {
	mock := &mockOrchestrator{
		listCommandsFn: func(_ context.Context, sandboxID string) ([]*store.Command, error) {
			return []*store.Command{
				{ID: "CMD-1", SandboxID: sandboxID, Command: "whoami", ExitCode: 0},
				{ID: "CMD-2", SandboxID: sandboxID, Command: "ls", ExitCode: 0},
			}, nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/SBX-123/commands", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "SBX-123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	var count float64
	if err := json.Unmarshal(resp["count"], &count); err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if int(count) != 2 {
		t.Fatalf("expected count 2, got %d", int(count))
	}
}
