package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"

	"github.com/aspectrr/fluid.sh/sandbox-host/internal/agent"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/config"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/image"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/janitor"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/microvm"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/network"
	"github.com/aspectrr/fluid.sh/sandbox-host/internal/state"
)

const version = "0.1.0"

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
		cfgPath = filepath.Join(home, ".fluid", "sandbox-host.yaml")
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	// Ensure host ID
	if cfg.HostID == "" {
		cfg.HostID = uuid.NewString()[:8]
		_ = config.Save(cfgPath, cfg)
		logger.Info("generated host ID", "host_id", cfg.HostID)
	}

	logger.Info("sandbox-host starting",
		"host_id", cfg.HostID,
		"config", cfgPath,
	)

	// Initialize SQLite state store
	st, err := state.NewStore(cfg.State.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()
	logger.Info("state store initialized", "db_path", cfg.State.DBPath)

	// Initialize microVM manager
	vmMgr, err := microvm.NewManager(cfg.MicroVM.QEMUBinary, cfg.MicroVM.WorkDir, logger)
	if err != nil {
		logger.Warn("microVM manager initialization failed (qemu not available)", "error", err)
		// Continue without VM manager - useful for development
		vmMgr = nil
	} else {
		// Recover state from any running VMs
		if err := vmMgr.RecoverState(ctx); err != nil {
			logger.Warn("state recovery failed", "error", err)
		}
		logger.Info("microVM manager initialized", "work_dir", cfg.MicroVM.WorkDir)
	}

	// Initialize network manager
	netMgr := network.NewNetworkManager(
		cfg.Network.DefaultBridge,
		cfg.Network.BridgeMap,
		cfg.Network.DHCPMode,
		logger,
	)
	logger.Info("network manager initialized",
		"default_bridge", cfg.Network.DefaultBridge,
		"dhcp_mode", cfg.Network.DHCPMode,
	)

	// Initialize image store
	imgStore, err := image.NewStore(cfg.Image.BaseDir, logger)
	if err != nil {
		return err
	}
	images, _ := imgStore.ListNames()
	logger.Info("image store initialized",
		"base_dir", cfg.Image.BaseDir,
		"images", len(images),
	)

	// Initialize janitor
	destroyFn := func(ctx context.Context, sandboxID string) error {
		if vmMgr != nil {
			info, err := vmMgr.Get(sandboxID)
			if err == nil {
				_ = network.DestroyTAP(ctx, info.TAPDevice)
			}
			if err := vmMgr.Destroy(ctx, sandboxID); err != nil {
				return err
			}
		}
		microvm.RemoveOverlay(cfg.MicroVM.WorkDir, sandboxID)
		return st.DeleteSandbox(ctx, sandboxID)
	}

	jan := janitor.New(st, destroyFn, cfg.Janitor.DefaultTTL, logger)
	go jan.Start(ctx, cfg.Janitor.Interval)

	// Initialize gRPC agent client
	agentClient := agent.NewClient(
		agent.Config{
			HostID:   cfg.HostID,
			Version:  version,
			Address:  cfg.ControlPlane.Address,
			Insecure: cfg.ControlPlane.Insecure,
			CertFile: cfg.ControlPlane.CertFile,
			KeyFile:  cfg.ControlPlane.KeyFile,
			CAFile:   cfg.ControlPlane.CAFile,
		},
		vmMgr,
		netMgr,
		imgStore,
		nil, // sourcevm.Manager - initialized separately if libvirt is available
		st,
		logger,
	)

	logger.Info("sandbox-host ready",
		"host_id", cfg.HostID,
		"control_plane", cfg.ControlPlane.Address,
	)

	// Start gRPC agent in background (reconnects automatically)
	agentErrCh := make(chan error, 1)
	go func() {
		agentErrCh <- agentClient.Run(ctx)
	}()

	// Wait for shutdown signal or agent fatal error
	select {
	case <-ctx.Done():
		logger.Info("sandbox-host shutting down")
	case err := <-agentErrCh:
		if err != nil && ctx.Err() == nil {
			logger.Error("agent error", "error", err)
			return err
		}
	}

	return nil
}
