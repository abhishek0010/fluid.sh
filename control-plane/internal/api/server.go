// Package api provides the REST API server for the control plane.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Orchestrator defines the interface the API uses to manage sandboxes.
type Orchestrator interface {
	SandboxOrchestrator
	SourceVMOrchestrator
	HostOrchestrator
}

// Server is the REST API server.
type Server struct {
	Router       chi.Router
	orchestrator Orchestrator
	logger       *slog.Logger
}

// NewServer creates a REST API server with all routes registered.
func NewServer(orch Orchestrator, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.SetHeader("Content-Type", "application/json"))

	s := &Server{
		Router:       router,
		orchestrator: orch,
		logger:       logger.With("component", "api"),
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.Router.Get("/v1/health", s.handleHealth)

	// Sandbox routes
	s.Router.Post("/v1/sandboxes", s.handleCreateSandbox)
	s.Router.Get("/v1/sandboxes", s.handleListSandboxes)
	s.Router.Get("/v1/sandboxes/{id}", s.handleGetSandbox)
	s.Router.Delete("/v1/sandboxes/{id}", s.handleDestroySandbox)
	s.Router.Post("/v1/sandboxes/{id}/run", s.handleRunCommand)
	s.Router.Post("/v1/sandboxes/{id}/start", s.handleStartSandbox)
	s.Router.Post("/v1/sandboxes/{id}/stop", s.handleStopSandbox)
	s.Router.Get("/v1/sandboxes/{id}/ip", s.handleGetSandboxIP)
	s.Router.Post("/v1/sandboxes/{id}/snapshot", s.handleCreateSnapshot)
	s.Router.Get("/v1/sandboxes/{id}/commands", s.handleListCommands)

	// Host routes
	s.Router.Get("/v1/hosts", s.handleListHosts)
	s.Router.Get("/v1/hosts/{id}", s.handleGetHost)

	// Source VM routes
	s.Router.Get("/v1/vms", s.handleListVMs)
	s.Router.Post("/v1/sources/{vm}/prepare", s.handlePrepareSourceVM)
	s.Router.Post("/v1/sources/{vm}/validate", s.handleValidateSourceVM)
	s.Router.Post("/v1/sources/{vm}/run", s.handleRunSourceCommand)
	s.Router.Post("/v1/sources/{vm}/read", s.handleReadSourceFile)
}

// StartHTTP runs the HTTP server on the given address.
func (s *Server) StartHTTP(addr string) error {
	s.logger.Info("starting HTTP server", "addr", addr)
	return http.ListenAndServe(addr, s.Router)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
