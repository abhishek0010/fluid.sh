// Package grpc provides the gRPC server that accepts bidirectional streams
// from sandbox hosts.
package grpc

import (
	"fmt"
	"log/slog"
	"net"

	fluidv1 "github.com/aspectrr/fluid.sh/proto/gen/go/fluid/v1"

	"github.com/aspectrr/fluid.sh/control-plane/internal/registry"
	"github.com/aspectrr/fluid.sh/control-plane/internal/store/postgres"

	"google.golang.org/grpc"
)

// Server wraps a gRPC server that sandbox hosts connect to.
type Server struct {
	listener   net.Listener
	grpcServer *grpc.Server
	handler    *StreamHandler
	registry   *registry.Registry
	store      *postgres.Store
	logger     *slog.Logger
}

// NewServer creates a gRPC server listening on addr and registers the
// HostService stream handler. The returned Server exposes the StreamHandler
// so the orchestrator can call SendAndWait to dispatch commands to hosts.
func NewServer(
	addr string,
	reg *registry.Registry,
	st *postgres.Store,
	logger *slog.Logger,
) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}

	gs := grpc.NewServer()

	handler := NewStreamHandler(reg, st, logger)
	fluidv1.RegisterHostServiceServer(gs, handler)

	s := &Server{
		listener:   lis,
		grpcServer: gs,
		handler:    handler,
		registry:   reg,
		store:      st,
		logger:     logger.With("component", "grpc"),
	}

	return s, nil
}

// Handler returns the stream handler, allowing the orchestrator to call
// SendAndWait for dispatching commands to connected hosts.
func (s *Server) Handler() *StreamHandler {
	return s.handler
}

// Start begins accepting connections. It blocks until the server is stopped
// or an unrecoverable error occurs.
func (s *Server) Start() error {
	s.logger.Info("gRPC server starting", "addr", s.listener.Addr().String())
	return s.grpcServer.Serve(s.listener)
}

// Stop performs a graceful shutdown of the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.grpcServer.GracefulStop()
}
