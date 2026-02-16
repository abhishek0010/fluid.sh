package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// VMInfo is the REST representation of a source VM.
type VMInfo struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IPAddress string `json:"ip_address,omitempty"`
	Prepared  bool   `json:"prepared"`
	HostID    string `json:"host_id,omitempty"`
}

// PrepareRequest is the REST API request for preparing a source VM.
type PrepareRequest struct {
	SSHUser    string `json:"ssh_user"`
	SSHKeyPath string `json:"ssh_key_path"`
}

// RunSourceRequest is the REST API request for running a command on a source VM.
type RunSourceRequest struct {
	Command    string `json:"command"`
	TimeoutSec int    `json:"timeout_seconds,omitempty"`
}

// ReadSourceRequest is the REST API request for reading a file from a source VM.
type ReadSourceRequest struct {
	Path string `json:"path"`
}

// SourceCommandResult is the REST response for a source VM command.
type SourceCommandResult struct {
	SourceVM string `json:"source_vm"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// SourceFileResult is the REST response for reading a source VM file.
type SourceFileResult struct {
	SourceVM string `json:"source_vm"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}

// SourceVMOrchestrator defines source VM operations.
type SourceVMOrchestrator interface {
	ListVMs(ctx context.Context) ([]*VMInfo, error)
	PrepareSourceVM(ctx context.Context, vmName, sshUser, keyPath string) (any, error)
	ValidateSourceVM(ctx context.Context, vmName string) (any, error)
	RunSourceCommand(ctx context.Context, vmName, command string, timeoutSec int) (*SourceCommandResult, error)
	ReadSourceFile(ctx context.Context, vmName, path string) (*SourceFileResult, error)
}

func (s *Server) handleListVMs(w http.ResponseWriter, r *http.Request) {
	vms, err := s.orchestrator.ListVMs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"vms":   vms,
		"count": len(vms),
	})
}

func (s *Server) handlePrepareSourceVM(w http.ResponseWriter, r *http.Request) {
	vm := chi.URLParam(r, "vm")

	var req PrepareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := s.orchestrator.PrepareSourceVM(r.Context(), vm, req.SSHUser, req.SSHKeyPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleValidateSourceVM(w http.ResponseWriter, r *http.Request) {
	vm := chi.URLParam(r, "vm")

	result, err := s.orchestrator.ValidateSourceVM(r.Context(), vm)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleRunSourceCommand(w http.ResponseWriter, r *http.Request) {
	vm := chi.URLParam(r, "vm")

	var req RunSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Command == "" {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}

	result, err := s.orchestrator.RunSourceCommand(r.Context(), vm, req.Command, req.TimeoutSec)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleReadSourceFile(w http.ResponseWriter, r *http.Request) {
	vm := chi.URLParam(r, "vm")

	var req ReadSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	result, err := s.orchestrator.ReadSourceFile(r.Context(), vm, req.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
