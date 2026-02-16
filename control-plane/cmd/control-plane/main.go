package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/aspectrr/fluid.sh/control-plane/internal/api"
	"github.com/aspectrr/fluid.sh/control-plane/internal/config"
	cpgrpc "github.com/aspectrr/fluid.sh/control-plane/internal/grpc"
	"github.com/aspectrr/fluid.sh/control-plane/internal/orchestrator"
	"github.com/aspectrr/fluid.sh/control-plane/internal/registry"
	"github.com/aspectrr/fluid.sh/control-plane/internal/store/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load config
	cfgPath := *configPath
	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = filepath.Join(home, ".fluid", "control-plane.yaml")
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	logger.Info("control-plane starting",
		"grpc_addr", cfg.GRPC.Address,
		"api_addr", cfg.API.Address,
		"database", cfg.Database.URL,
	)

	// Initialize PostgreSQL store
	st, err := postgres.New(ctx, cfg.Database.URL, cfg.Database.AutoMigrate)
	if err != nil {
		return err
	}
	defer st.Close()
	logger.Info("postgres store initialized")

	// Initialize host registry
	reg := registry.New()

	// Initialize gRPC server
	grpcServer, err := cpgrpc.NewServer(cfg.GRPC.Address, reg, st, logger)
	if err != nil {
		return err
	}

	// Initialize orchestrator (uses gRPC handler as the HostSender)
	orch := orchestrator.New(reg, st, grpcServer.Handler(), logger, cfg.Orchestrator.DefaultTTL)

	// Initialize REST API server
	apiServer := api.NewServer(orch, logger)

	// Start gRPC server in background
	grpcErrCh := make(chan error, 1)
	go func() {
		grpcErrCh <- grpcServer.Start()
	}()

	// Start REST API server in background
	apiErrCh := make(chan error, 1)
	go func() {
		apiErrCh <- apiServer.StartHTTP(cfg.API.Address)
	}()

	logger.Info("control-plane ready",
		"grpc_addr", cfg.GRPC.Address,
		"api_addr", cfg.API.Address,
	)

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		logger.Info("control-plane shutting down")
		grpcServer.Stop()
	case err := <-grpcErrCh:
		logger.Error("gRPC server error", "error", err)
		return err
	case err := <-apiErrCh:
		logger.Error("API server error", "error", err)
		return err
	}

	return nil
}
