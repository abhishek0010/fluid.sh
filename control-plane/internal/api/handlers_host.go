package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// HostInfo is the REST API representation of a connected host.
type HostInfo struct {
	HostID           string   `json:"host_id"`
	Hostname         string   `json:"hostname"`
	Status           string   `json:"status"`
	ActiveSandboxes  int      `json:"active_sandboxes"`
	AvailableCPUs    int32    `json:"available_cpus"`
	AvailableMemMB   int64    `json:"available_memory_mb"`
	AvailableDiskMB  int64    `json:"available_disk_mb"`
	BaseImages       []string `json:"base_images"`
	LastHeartbeat    string   `json:"last_heartbeat"`
}

// HostOrchestrator defines host operations.
type HostOrchestrator interface {
	ListHosts(ctx context.Context) ([]*HostInfo, error)
	GetHost(ctx context.Context, id string) (*HostInfo, error)
}

func (s *Server) handleListHosts(w http.ResponseWriter, r *http.Request) {
	hosts, err := s.orchestrator.ListHosts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"hosts": hosts,
		"count": len(hosts),
	})
}

func (s *Server) handleGetHost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host, err := s.orchestrator.GetHost(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, host)
}
