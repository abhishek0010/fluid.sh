package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandleListHosts(t *testing.T) {
	mock := &mockOrchestrator{
		listHostsFn: func(_ context.Context) ([]*HostInfo, error) {
			return []*HostInfo{
				{HostID: "host-1", Hostname: "node-1", Status: "ONLINE", ActiveSandboxes: 3},
				{HostID: "host-2", Hostname: "node-2", Status: "OFFLINE", ActiveSandboxes: 0},
			}, nil
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/hosts", nil)
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

	var hosts []HostInfo
	if err := json.Unmarshal(resp["hosts"], &hosts); err != nil {
		t.Fatalf("decode hosts: %v", err)
	}
	if hosts[0].HostID != "host-1" {
		t.Fatalf("expected first host_id host-1, got %s", hosts[0].HostID)
	}
}

func TestHandleGetHost(t *testing.T) {
	mock := &mockOrchestrator{
		getHostFn: func(_ context.Context, id string) (*HostInfo, error) {
			if id == "host-1" {
				return &HostInfo{
					HostID:          "host-1",
					Hostname:        "node-1",
					Status:          "ONLINE",
					ActiveSandboxes: 5,
					AvailableCPUs:   8,
					AvailableMemMB:  16384,
				}, nil
			}
			return nil, fmt.Errorf("host not found: %s", id)
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/hosts/host-1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "host-1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp HostInfo
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.HostID != "host-1" {
		t.Fatalf("expected host_id host-1, got %s", resp.HostID)
	}
	if resp.Hostname != "node-1" {
		t.Fatalf("expected hostname node-1, got %s", resp.Hostname)
	}
	if resp.AvailableCPUs != 8 {
		t.Fatalf("expected available_cpus 8, got %d", resp.AvailableCPUs)
	}
}

func TestHandleGetHost_NotFound(t *testing.T) {
	mock := &mockOrchestrator{
		getHostFn: func(_ context.Context, id string) (*HostInfo, error) {
			return nil, fmt.Errorf("host not found: %s", id)
		},
	}

	s := NewServer(mock, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/hosts/host-nope", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "host-nope")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	s.Router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["error"] == "" {
		t.Fatal("expected error message in response")
	}
}
