package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandleListVMs(t *testing.T) {
	mock := &mockOrchestrator{
		listVMsFn: func(_ context.Context) ([]*VMInfo, error) {
			return []*VMInfo{
				{Name: "ubuntu-base", State: "running", Prepared: true},
				{Name: "debian-12", State: "shutoff", Prepared: false},
			}, nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/vms", nil)
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

	var vms []VMInfo
	if err := json.Unmarshal(resp["vms"], &vms); err != nil {
		t.Fatalf("decode vms: %v", err)
	}
	if vms[0].Name != "ubuntu-base" {
		t.Fatalf("expected first vm name ubuntu-base, got %s", vms[0].Name)
	}
}

func TestHandlePrepareSourceVM(t *testing.T) {
	mock := &mockOrchestrator{
		prepareSourceVMFn: func(_ context.Context, vmName, sshUser, keyPath string) (any, error) {
			return map[string]any{
				"status":  "prepared",
				"vm_name": vmName,
			}, nil
		},
	}

	s := NewServer(mock, nil)
	body := `{"ssh_user":"root","ssh_key_path":"/root/.ssh/id_rsa"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sources/ubuntu-base/prepare", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("vm", "ubuntu-base")
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
	if resp["status"] != "prepared" {
		t.Fatalf("expected status=prepared, got %v", resp["status"])
	}
	if resp["vm_name"] != "ubuntu-base" {
		t.Fatalf("expected vm_name=ubuntu-base, got %v", resp["vm_name"])
	}
}

func TestHandleValidateSourceVM(t *testing.T) {
	mock := &mockOrchestrator{
		validateSourceVMFn: func(_ context.Context, vmName string) (any, error) {
			return map[string]any{
				"valid":   true,
				"vm_name": vmName,
			}, nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/sources/ubuntu-base/validate", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("vm", "ubuntu-base")
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
	if resp["valid"] != true {
		t.Fatalf("expected valid=true, got %v", resp["valid"])
	}
}

func TestHandleRunSourceCommand(t *testing.T) {
	mock := &mockOrchestrator{
		runSourceCommandFn: func(_ context.Context, vmName, command string, _ int) (*SourceCommandResult, error) {
			return &SourceCommandResult{
				SourceVM: vmName,
				ExitCode: 0,
				Stdout:   "root\n",
				Stderr:   "",
			}, nil
		},
	}

	s := NewServer(mock, nil)
	body := `{"command":"whoami"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sources/ubuntu-base/run", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("vm", "ubuntu-base")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp SourceCommandResult
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.SourceVM != "ubuntu-base" {
		t.Fatalf("expected source_vm=ubuntu-base, got %s", resp.SourceVM)
	}
	if resp.Stdout != "root\n" {
		t.Fatalf("expected stdout 'root\\n', got %q", resp.Stdout)
	}
}

func TestHandleRunSourceCommand_EmptyCommand(t *testing.T) {
	mock := &mockOrchestrator{}
	s := NewServer(mock, nil)

	body := `{"command":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sources/ubuntu-base/run", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("vm", "ubuntu-base")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleReadSourceFile(t *testing.T) {
	mock := &mockOrchestrator{
		readSourceFileFn: func(_ context.Context, vmName, path string) (*SourceFileResult, error) {
			return &SourceFileResult{
				SourceVM: vmName,
				Path:     path,
				Content:  "file contents here",
			}, nil
		},
	}

	s := NewServer(mock, nil)
	body := `{"path":"/etc/hostname"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sources/ubuntu-base/read", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("vm", "ubuntu-base")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp SourceFileResult
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Path != "/etc/hostname" {
		t.Fatalf("expected path=/etc/hostname, got %s", resp.Path)
	}
	if resp.Content != "file contents here" {
		t.Fatalf("expected content 'file contents here', got %q", resp.Content)
	}
}

func TestHandleReadSourceFile_EmptyPath(t *testing.T) {
	mock := &mockOrchestrator{}
	s := NewServer(mock, nil)

	body := `{"path":""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/sources/ubuntu-base/read", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("vm", "ubuntu-base")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleHealth(t *testing.T) {
	mock := &mockOrchestrator{}
	s := NewServer(mock, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %s", resp["status"])
	}
}
