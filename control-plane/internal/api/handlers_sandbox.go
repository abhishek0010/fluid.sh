package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aspectrr/fluid.sh/control-plane/internal/store"
)

// SandboxOrchestrator defines sandbox operations.
type SandboxOrchestrator interface {
	CreateSandbox(ctx context.Context, req CreateSandboxRequest) (*store.Sandbox, error)
	GetSandbox(ctx context.Context, id string) (*store.Sandbox, error)
	ListSandboxes(ctx context.Context) ([]*store.Sandbox, error)
	DestroySandbox(ctx context.Context, id string) error
	StartSandbox(ctx context.Context, id string) error
	StopSandbox(ctx context.Context, id string) error
	RunCommand(ctx context.Context, sandboxID, command string, timeoutSec int) (*store.Command, error)
	CreateSnapshot(ctx context.Context, sandboxID, name string) (*SnapshotResponse, error)
	ListCommands(ctx context.Context, sandboxID string) ([]*store.Command, error)
}

// CreateSandboxRequest is the REST API request for creating a sandbox.
type CreateSandboxRequest struct {
	AgentID    string `json:"agent_id"`
	SourceVM   string `json:"source_vm"`
	BaseImage  string `json:"base_image"`
	Name       string `json:"name"`
	VCPUs      int    `json:"vcpus,omitempty"`
	MemoryMB   int    `json:"memory_mb,omitempty"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
	Network    string `json:"network,omitempty"`
}

// RunCommandRequest is the REST API request for running a command.
type RunCommandRequest struct {
	Command    string            `json:"command"`
	TimeoutSec int              `json:"timeout_seconds,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// SnapshotRequest is the REST API request for creating a snapshot.
type SnapshotRequest struct {
	Name string `json:"name"`
}

// SnapshotResponse is returned after creating a snapshot.
type SnapshotResponse struct {
	SnapshotID   string    `json:"snapshot_id"`
	SandboxID    string    `json:"sandbox_id"`
	SnapshotName string    `json:"snapshot_name"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Server) handleCreateSandbox(w http.ResponseWriter, r *http.Request) {
	var req CreateSandboxRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.SourceVM == "" && req.BaseImage == "" {
		writeError(w, http.StatusBadRequest, "source_vm or base_image is required")
		return
	}

	sandbox, err := s.orchestrator.CreateSandbox(r.Context(), req)
	if err != nil {
		s.logger.Error("create sandbox failed", "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, sandbox)
}

func (s *Server) handleListSandboxes(w http.ResponseWriter, r *http.Request) {
	sandboxes, err := s.orchestrator.ListSandboxes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sandboxes": sandboxes,
		"count":     len(sandboxes),
	})
}

func (s *Server) handleGetSandbox(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sandbox, err := s.orchestrator.GetSandbox(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, sandbox)
}

func (s *Server) handleDestroySandbox(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orchestrator.DestroySandbox(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"destroyed":  true,
		"sandbox_id": id,
	})
}

func (s *Server) handleRunCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req RunCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}

	result, err := s.orchestrator.RunCommand(r.Context(), id, req.Command, req.TimeoutSec)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleStartSandbox(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orchestrator.StartSandbox(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"started":    true,
		"sandbox_id": id,
	})
}

func (s *Server) handleStopSandbox(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.orchestrator.StopSandbox(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"stopped":    true,
		"sandbox_id": id,
	})
}

func (s *Server) handleGetSandboxIP(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sandbox, err := s.orchestrator.GetSandbox(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sandbox_id": id,
		"ip_address": sandbox.IPAddress,
	})
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req SnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := s.orchestrator.CreateSnapshot(r.Context(), id, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleListCommands(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	commands, err := s.orchestrator.ListCommands(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"commands": commands,
		"count":    len(commands),
	})
}
