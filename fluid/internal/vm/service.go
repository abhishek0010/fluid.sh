package vm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
	"github.com/aspectrr/fluid.sh/fluid/internal/provider" // VM manager interface
	"github.com/aspectrr/fluid.sh/fluid/internal/readonly"
	"github.com/aspectrr/fluid.sh/fluid/internal/sshkeys"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
	"github.com/aspectrr/fluid.sh/fluid/internal/telemetry"
)

// RemoteManagerFactory creates a provider.Manager for a remote host.
// This allows the service to create managers for sandboxes on different hosts
// without depending on a specific provider implementation.
type RemoteManagerFactory func(host config.HostConfig) provider.Manager

// Service orchestrates VM operations and data persistence.
// It represents the main application layer for sandbox lifecycle, command exec,
// snapshotting, diffing, and artifact generation orchestration.
type Service struct {
	mgr                  provider.Manager
	store                store.Store
	ssh                  SSHRunner
	keyMgr               sshkeys.KeyProvider // Optional: manages SSH keys for RunCommand
	telemetry            telemetry.Service
	cfg                  Config
	remoteManagerFactory RemoteManagerFactory // Creates managers for remote hosts
	timeNowFn            func() time.Time
	logger               *slog.Logger
}

// Config controls default VM parameters and timeouts used by the service.
type Config struct {
	// Default libvirt network name (e.g., "default") used when creating VMs.
	Network string

	// Default shape if not provided by callers.
	DefaultVCPUs    int
	DefaultMemoryMB int

	// CommandTimeout sets a default timeout for RunCommand when caller doesn't provide one.
	CommandTimeout time.Duration

	// IPDiscoveryTimeout controls how long StartSandbox waits for the VM IP (when requested).
	IPDiscoveryTimeout time.Duration

	// SSHReadinessTimeout controls how long to wait for SSH to become available after IP discovery.
	// If zero, SSH readiness check is skipped. Default: 60s
	SSHReadinessTimeout time.Duration

	// SSHProxyJump specifies a jump host for SSH connections to VMs.
	// Format: "user@host:port" or just "host" for default user/port.
	// Required when VMs are on an isolated network not directly reachable.
	SSHProxyJump string
}

// Option configures the Service during construction.
type Option func(*Service)

// WithSSHRunner overrides the default SSH runner implementation.
func WithSSHRunner(r SSHRunner) Option {
	return func(s *Service) { s.ssh = r }
}

// WithTelemetry sets the telemetry service.
func WithTelemetry(t telemetry.Service) Option {
	return func(s *Service) { s.telemetry = t }
}

// WithTimeNow overrides the clock (useful for tests).
func WithTimeNow(fn func() time.Time) Option {
	return func(s *Service) { s.timeNowFn = fn }
}

// WithLogger sets a custom logger for the service.
func WithLogger(l *slog.Logger) Option {
	return func(s *Service) { s.logger = l }
}

// WithKeyManager sets a key manager for managed SSH credentials.
// When set, RunCommand can be called without explicit privateKeyPath.
func WithKeyManager(km sshkeys.KeyProvider) Option {
	return func(s *Service) { s.keyMgr = km }
}

// WithRemoteManagerFactory sets the factory for creating remote managers.
func WithRemoteManagerFactory(f RemoteManagerFactory) Option {
	return func(s *Service) { s.remoteManagerFactory = f }
}

// NewService constructs a VM service with the provided manager, store and config.
func NewService(mgr provider.Manager, st store.Store, cfg Config, opts ...Option) *Service {
	if cfg.DefaultVCPUs <= 0 {
		cfg.DefaultVCPUs = 2
	}
	if cfg.DefaultMemoryMB <= 0 {
		cfg.DefaultMemoryMB = 2048
	}
	if cfg.CommandTimeout <= 0 {
		cfg.CommandTimeout = 10 * time.Minute
	}
	if cfg.IPDiscoveryTimeout <= 0 {
		cfg.IPDiscoveryTimeout = 2 * time.Minute
	}
	if cfg.SSHReadinessTimeout <= 0 {
		cfg.SSHReadinessTimeout = 60 * time.Second
	}
	s := &Service{
		mgr:       mgr,
		store:     st,
		cfg:       cfg,
		ssh:       &DefaultSSHRunner{DefaultProxyJump: cfg.SSHProxyJump, Logger: slog.Default()},
		timeNowFn: time.Now,
		logger:    slog.Default(),
	}
	for _, o := range opts {
		o(s)
	}
	// Ensure SSH runner uses the same logger
	if r, ok := s.ssh.(*DefaultSSHRunner); ok {
		r.Logger = s.logger
	}
	// Default to noop telemetry if not provided
	if s.telemetry == nil {
		s.telemetry = telemetry.NewNoopService()
	}
	return s
}

// getManagerForSandbox returns the appropriate manager for a sandbox.
// If the sandbox was created on a remote host and a remote factory is available,
// returns a remote manager. Otherwise, returns the local manager.
func (s *Service) getManagerForSandbox(sb *store.Sandbox) provider.Manager {
	if sb.HostAddress != nil && *sb.HostAddress != "" && s.remoteManagerFactory != nil {
		hostName := ""
		if sb.HostName != nil {
			hostName = *sb.HostName
		}
		host := config.HostConfig{
			Name:    hostName,
			Address: *sb.HostAddress,
			SSHUser: "root",
			SSHPort: 22,
		}
		return s.remoteManagerFactory(host)
	}
	return s.mgr
}

// ResourceValidationResult contains the results of validating resources for sandbox creation.
// If NeedsApproval is true, the caller should request human approval before proceeding.
type ResourceValidationResult struct {
	Valid         bool
	NeedsApproval bool
	SourceVMValid bool
	VMErrors      []string
	VMWarnings    []string
	ResourceCheck *provider.ResourceCheckResult
}

// CheckResourcesForSandbox validates resources without failing.
// Returns a ResourceValidationResult that indicates whether approval is needed.
func (s *Service) CheckResourcesForSandbox(ctx context.Context, mgr provider.Manager, sourceVMName string, cpu, memoryMB int) *ResourceValidationResult {
	result := &ResourceValidationResult{
		Valid:         true,
		NeedsApproval: false,
		SourceVMValid: true,
	}

	// Apply defaults
	if cpu <= 0 {
		cpu = s.cfg.DefaultVCPUs
	}
	if memoryMB <= 0 {
		memoryMB = s.cfg.DefaultMemoryMB
	}

	// 1. Validate source VM
	vmValidation, err := mgr.ValidateSourceVM(ctx, sourceVMName)
	if err != nil {
		s.logger.Warn("source VM validation failed", "source_vm", sourceVMName, "error", err)
		result.SourceVMValid = false
		result.VMErrors = append(result.VMErrors, fmt.Sprintf("validation error: %v", err))
		result.Valid = false
	} else if !vmValidation.Valid {
		result.SourceVMValid = false
		result.VMErrors = vmValidation.Errors
		result.VMWarnings = vmValidation.Warnings
		result.Valid = false
	} else {
		result.VMWarnings = vmValidation.Warnings
	}

	// 2. Check host resources
	resourceCheck, err := mgr.CheckHostResources(ctx, cpu, memoryMB)
	if err != nil {
		s.logger.Warn("host resource check failed", "error", err)
		// Resource check failed, but this shouldn't block - set needsApproval
		result.ResourceCheck = &provider.ResourceCheckResult{
			Valid:            false,
			RequiredMemoryMB: memoryMB,
			RequiredCPUs:     cpu,
			Errors:           []string{fmt.Sprintf("resource check error: %v", err)},
		}
		result.NeedsApproval = true
	} else {
		result.ResourceCheck = resourceCheck
		if !resourceCheck.Valid {
			// Resources insufficient - needs approval but can proceed
			result.NeedsApproval = true
			result.Valid = false
		}
	}

	return result
}

// GetManager returns the default manager.
func (s *Service) GetManager() provider.Manager {
	return s.mgr
}

// GetRemoteManager returns a manager for a specific remote host.
func (s *Service) GetRemoteManager(host *config.HostConfig) provider.Manager {
	if host == nil || s.remoteManagerFactory == nil {
		return s.mgr
	}
	return s.remoteManagerFactory(*host)
}

// GetDefaultMemory returns the default memory in MB
func (s *Service) GetDefaultMemory() int {
	return s.cfg.DefaultMemoryMB
}

// GetDefaultCPUs returns the default number of CPUs
func (s *Service) GetDefaultCPUs() int {
	return s.cfg.DefaultVCPUs
}

// CreateSandbox clones a VM from an existing VM and persists a Sandbox record.
//
// sourceSandboxName is the name of the existing VM in libvirt to clone from.
// SandboxName is optional; if empty, a name will be generated.
// cpu and memoryMB are optional; if <=0 the service defaults are used.
// ttlSeconds is optional; if provided, sets the TTL for auto garbage collection.
// autoStart if true will start the VM immediately after creation.
// waitForIP if true (and autoStart is true), will wait for IP discovery.
// Returns the sandbox, the discovered IP (if autoStart and waitForIP), and any error.
// validateIPUniqueness checks if the given IP is already assigned to another running sandbox.
// Returns an error if the IP is assigned to a different sandbox that is still running.
func (s *Service) validateIPUniqueness(ctx context.Context, currentSandboxID, ip string) error {
	// Check both RUNNING and STARTING sandboxes to prevent race conditions
	// where two sandboxes might discover the same IP simultaneously
	statesToCheck := []store.SandboxState{
		store.SandboxStateRunning,
		store.SandboxStateStarting,
	}

	for _, state := range statesToCheck {
		stateFilter := state
		sandboxes, err := s.store.ListSandboxes(ctx, store.SandboxFilter{
			State: &stateFilter,
		}, nil)
		if err != nil {
			return fmt.Errorf("list sandboxes (state=%s) for IP validation: %w", state, err)
		}

		for _, sb := range sandboxes {
			if sb.ID == currentSandboxID {
				continue // Skip the current sandbox
			}
			if sb.IPAddress != nil && *sb.IPAddress == ip {
				s.logger.Error("IP address conflict detected",
					"conflict_ip", ip,
					"current_sandbox_id", currentSandboxID,
					"conflicting_sandbox_id", sb.ID,
					"conflicting_sandbox_name", sb.SandboxName,
					"conflicting_sandbox_state", sb.State,
				)
				return fmt.Errorf("IP %s is already assigned to sandbox %s (vm: %s, state: %s)", ip, sb.ID, sb.SandboxName, sb.State)
			}
		}
	}
	return nil
}

// waitForSSH waits until SSH is accepting connections on the given IP.
// It uses exponential backoff to probe SSH readiness.
// proxyJump is optional - used when the sandbox is on a remote host.
func (s *Service) waitForSSH(ctx context.Context, sandboxID, ip, proxyJump string, timeout time.Duration) error {
	if timeout <= 0 {
		return nil // SSH readiness check disabled
	}

	// Skip if no key manager configured
	if s.keyMgr == nil {
		s.logger.Debug("no key manager configured, skipping SSH readiness check")
		return nil
	}

	s.logger.Info("waiting for SSH to become ready",
		"sandbox_id", sandboxID,
		"ip", ip,
		"timeout", timeout,
	)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Get credentials for SSH probe - use default "sandbox" user
	creds, err := s.keyMgr.GetCredentials(ctx, sandboxID, "sandbox")
	if err != nil {
		s.logger.Warn("failed to get SSH credentials for readiness check, skipping",
			"sandbox_id", sandboxID,
			"error", err,
		)
		return nil // Don't fail sandbox creation if we can't get creds
	}

	// Use short command timeout for probes
	probeTimeout := 10 * time.Second

	// Exponential backoff: 1s, 2s, 4s, 8s, 16s (capped)
	initialDelay := 1 * time.Second
	maxDelay := 16 * time.Second
	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("SSH readiness timeout after %v: %w", timeout, ctx.Err())
		default:
		}

		// Try to run a simple command
		_, _, exitCode, runErr := s.ssh.RunWithCert(
			ctx,
			ip,
			creds.Username,
			creds.PrivateKeyPath,
			creds.CertificatePath,
			"true", // Simple command that succeeds if SSH works
			probeTimeout,
			nil,
			proxyJump,
		)

		if runErr == nil && exitCode == 0 {
			s.logger.Info("SSH is ready",
				"sandbox_id", sandboxID,
				"ip", ip,
				"attempts", attempt+1,
			)
			return nil
		}

		// Calculate backoff delay
		delay := initialDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}

		s.logger.Debug("SSH not ready, retrying",
			"sandbox_id", sandboxID,
			"ip", ip,
			"attempt", attempt+1,
			"delay", delay,
			"error", runErr,
		)

		select {
		case <-time.After(delay):
			attempt++
		case <-ctx.Done():
			return fmt.Errorf("SSH readiness timeout after %v: %w", timeout, ctx.Err())
		}
	}
}

func (s *Service) CreateSandbox(ctx context.Context, sourceSandboxName, agentID, sandboxName string, cpu, memoryMB int, ttlSeconds *int, autoStart, waitForIP bool) (*store.Sandbox, string, error) {
	if strings.TrimSpace(sourceSandboxName) == "" {
		return nil, "", fmt.Errorf("sourceSandboxName is required")
	}
	if strings.TrimSpace(agentID) == "" {
		return nil, "", fmt.Errorf("agentID is required")
	}
	if cpu <= 0 {
		cpu = s.cfg.DefaultVCPUs
	}
	if memoryMB <= 0 {
		memoryMB = s.cfg.DefaultMemoryMB
	}

	// Use provided sandbox name or generate one with sbx- prefix
	if sandboxName == "" {
		sandboxName = fmt.Sprintf("sbx-%s", shortID())
	}

	s.logger.Info("creating sandbox",
		"source_vm_name", sourceSandboxName,
		"agent_id", agentID,
		"sandbox_name", sandboxName,
		"cpu", cpu,
		"memory_mb", memoryMB,
		"auto_start", autoStart,
		"wait_for_ip", waitForIP,
	)

	jobID := fmt.Sprintf("JOB-%s", shortID())

	// Create the VM via libvirt manager by cloning from existing VM
	_, err := s.mgr.CloneFromVM(ctx, sourceSandboxName, sandboxName, cpu, memoryMB, s.cfg.Network)
	if err != nil {
		s.logger.Error("failed to clone VM",
			"source_vm_name", sourceSandboxName,
			"sandbox_name", sandboxName,
			"error", err,
		)
		return nil, "", fmt.Errorf("clone vm: %w", err)
	}

	sb := &store.Sandbox{
		ID:          fmt.Sprintf("SBX-%s", shortID()),
		JobID:       jobID,
		AgentID:     agentID,
		SandboxName: sandboxName,
		BaseImage:   sourceSandboxName, // Store the source VM name for reference
		Network:     s.cfg.Network,
		State:       store.SandboxStateCreated,
		TTLSeconds:  ttlSeconds,
		VCPUs:       cpu,
		MemoryMB:    memoryMB,
		CreatedAt:   s.timeNowFn().UTC(),
		UpdatedAt:   s.timeNowFn().UTC(),
	}
	if err := s.store.CreateSandbox(ctx, sb); err != nil {
		return nil, "", fmt.Errorf("persist sandbox: %w", err)
	}

	s.logger.Debug("sandbox cloned successfully",
		"sandbox_id", sb.ID,
		"sandbox_name", sandboxName,
	)

	// If autoStart is requested, start the VM immediately
	var ip string
	if autoStart {
		s.logger.Info("auto-starting sandbox",
			"sandbox_id", sb.ID,
			"sandbox_name", sb.SandboxName,
		)

		if err := s.mgr.StartVM(ctx, sb.SandboxName); err != nil {
			s.logger.Error("auto-start failed",
				"sandbox_id", sb.ID,
				"sandbox_name", sb.SandboxName,
				"error", err,
			)
			_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateError, nil)
			return sb, "", fmt.Errorf("auto-start vm: %w", err)
		}

		// Update state -> STARTING
		if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateStarting, nil); err != nil {
			return sb, "", err
		}
		sb.State = store.SandboxStateStarting

		if waitForIP {
			s.logger.Info("waiting for IP address",
				"sandbox_id", sb.ID,
				"timeout", s.cfg.IPDiscoveryTimeout,
			)

			var mac string
			ip, mac, err = s.mgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
			if err != nil {
				s.logger.Warn("IP discovery failed",
					"sandbox_id", sb.ID,
					"sandbox_name", sb.SandboxName,
					"error", err,
				)
				// Still mark as running even if we couldn't discover the IP
				_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil)
				sb.State = store.SandboxStateRunning
				return sb, "", fmt.Errorf("get ip: %w", err)
			}

			// Validate IP uniqueness before storing
			if err := s.validateIPUniqueness(ctx, sb.ID, ip); err != nil {
				s.logger.Error("IP conflict during sandbox creation",
					"sandbox_id", sb.ID,
					"sandbox_name", sb.SandboxName,
					"ip_address", ip,
					"mac_address", mac,
					"error", err,
				)
				_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil)
				sb.State = store.SandboxStateRunning
				return sb, "", fmt.Errorf("ip conflict: %w", err)
			}

			// Wait for SSH to become ready before marking as RUNNING
			// Local sandbox - no proxy jump needed
			if err := s.waitForSSH(ctx, sb.ID, ip, "", s.cfg.SSHReadinessTimeout); err != nil {
				s.logger.Warn("SSH readiness check failed",
					"sandbox_id", sb.ID,
					"ip_address", ip,
					"error", err,
				)
				// Don't fail - sandbox is still usable, just may need retries
			}

			if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, &ip); err != nil {
				return sb, ip, err
			}
			sb.State = store.SandboxStateRunning
			sb.IPAddress = &ip
		} else {
			if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil); err != nil {
				return sb, "", err
			}
			sb.State = store.SandboxStateRunning
		}
	}

	s.logger.Info("sandbox created",
		"sandbox_id", sb.ID,
		"state", sb.State,
		"ip_address", ip,
	)

	s.telemetry.Track("sandbox_create", map[string]any{
		"sandbox_id":  sb.ID,
		"base_image":  sb.BaseImage,
		"cpu":         cpu,
		"memory_mb":   memoryMB,
		"auto_start":  autoStart,
		"wait_for_ip": waitForIP,
		"agent_id":    agentID,
		"success":     true,
	})

	return sb, ip, nil
}

// CreateSandboxOnHost creates a sandbox on a specific remote host.
// This is used when multi-host support is enabled and the source VM is on a remote host.
func (s *Service) CreateSandboxOnHost(ctx context.Context, host *config.HostConfig, sourceSandboxName, agentID, sandboxName string, cpu, memoryMB int, ttlSeconds *int, autoStart, waitForIP bool) (*store.Sandbox, string, error) {
	if host == nil {
		return nil, "", fmt.Errorf("host is required for remote sandbox creation")
	}
	if strings.TrimSpace(sourceSandboxName) == "" {
		return nil, "", fmt.Errorf("sourceSandboxName is required")
	}
	if strings.TrimSpace(agentID) == "" {
		return nil, "", fmt.Errorf("agentID is required")
	}
	if cpu <= 0 {
		cpu = s.cfg.DefaultVCPUs
	}
	if memoryMB <= 0 {
		memoryMB = s.cfg.DefaultMemoryMB
	}

	// Use provided sandbox name or generate one with sbx- prefix
	if sandboxName == "" {
		sandboxName = fmt.Sprintf("sbx-%s", shortID())
	}

	// Create a remote manager for this host
	if s.remoteManagerFactory == nil {
		return nil, "", fmt.Errorf("remote manager factory not configured")
	}
	remoteMgr := s.remoteManagerFactory(*host)

	s.logger.Info("creating sandbox on remote host",
		"host_name", host.Name,
		"host_address", host.Address,
		"source_vm_name", sourceSandboxName,
		"agent_id", agentID,
		"sandbox_name", sandboxName,
		"cpu", cpu,
		"memory_mb", memoryMB,
		"auto_start", autoStart,
		"wait_for_ip", waitForIP,
	)

	jobID := fmt.Sprintf("JOB-%s", shortID())

	// Create the VM via remote libvirt manager
	_, err := remoteMgr.CloneFromVM(ctx, sourceSandboxName, sandboxName, cpu, memoryMB, s.cfg.Network)
	if err != nil {
		s.logger.Error("failed to clone VM on remote host",
			"host", host.Name,
			"source_vm_name", sourceSandboxName,
			"sandbox_name", sandboxName,
			"error", err,
		)
		return nil, "", fmt.Errorf("clone vm on host %s: %w", host.Name, err)
	}

	hostName := host.Name
	hostAddr := host.Address
	sb := &store.Sandbox{
		ID:          fmt.Sprintf("SBX-%s", shortID()),
		JobID:       jobID,
		AgentID:     agentID,
		SandboxName: sandboxName,
		BaseImage:   sourceSandboxName,
		Network:     s.cfg.Network,
		State:       store.SandboxStateCreated,
		TTLSeconds:  ttlSeconds,
		VCPUs:       cpu,
		MemoryMB:    memoryMB,
		HostName:    &hostName,
		HostAddress: &hostAddr,
		CreatedAt:   s.timeNowFn().UTC(),
		UpdatedAt:   s.timeNowFn().UTC(),
	}
	if err := s.store.CreateSandbox(ctx, sb); err != nil {
		return nil, "", fmt.Errorf("persist sandbox: %w", err)
	}

	s.logger.Debug("sandbox cloned on remote host",
		"sandbox_id", sb.ID,
		"sandbox_name", sandboxName,
		"host", host.Name,
	)

	// If autoStart is requested, start the VM immediately
	var ip string
	if autoStart {
		s.logger.Info("auto-starting sandbox on remote host",
			"sandbox_id", sb.ID,
			"sandbox_name", sb.SandboxName,
			"host", host.Name,
		)

		if err := remoteMgr.StartVM(ctx, sb.SandboxName); err != nil {
			s.logger.Error("auto-start failed on remote host",
				"sandbox_id", sb.ID,
				"sandbox_name", sb.SandboxName,
				"host", host.Name,
				"error", err,
			)
			_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateError, nil)
			return sb, "", fmt.Errorf("auto-start vm on host %s: %w", host.Name, err)
		}

		// Update state -> STARTING
		if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateStarting, nil); err != nil {
			return sb, "", err
		}
		sb.State = store.SandboxStateStarting

		if waitForIP {
			s.logger.Info("waiting for IP address on remote host",
				"sandbox_id", sb.ID,
				"host", host.Name,
				"timeout", s.cfg.IPDiscoveryTimeout,
			)

			var mac string
			ip, mac, err = remoteMgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
			if err != nil {
				s.logger.Warn("IP discovery failed on remote host",
					"sandbox_id", sb.ID,
					"sandbox_name", sb.SandboxName,
					"host", host.Name,
					"error", err,
				)
				_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil)
				sb.State = store.SandboxStateRunning
				return sb, "", fmt.Errorf("get ip on host %s: %w", host.Name, err)
			}

			// Validate IP uniqueness
			if err := s.validateIPUniqueness(ctx, sb.ID, ip); err != nil {
				s.logger.Error("IP conflict on remote host",
					"sandbox_id", sb.ID,
					"sandbox_name", sb.SandboxName,
					"ip_address", ip,
					"mac_address", mac,
					"host", host.Name,
					"error", err,
				)
				_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil)
				sb.State = store.SandboxStateRunning
				return sb, "", fmt.Errorf("ip conflict on host %s: %w", host.Name, err)
			}

			// Wait for SSH to become ready before marking as RUNNING
			// Remote sandbox - use host address as proxy jump
			proxyJump := fmt.Sprintf("root@%s", host.Address)
			if err := s.waitForSSH(ctx, sb.ID, ip, proxyJump, s.cfg.SSHReadinessTimeout); err != nil {
				s.logger.Warn("SSH readiness check failed on remote host",
					"sandbox_id", sb.ID,
					"ip_address", ip,
					"host", host.Name,
					"error", err,
				)
				// Don't fail - sandbox is still usable, just may need retries
			}

			if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, &ip); err != nil {
				return sb, ip, err
			}
			sb.State = store.SandboxStateRunning
			sb.IPAddress = &ip
		} else {
			if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil); err != nil {
				return sb, "", err
			}
			sb.State = store.SandboxStateRunning
		}
	}

	s.logger.Info("sandbox created on remote host",
		"sandbox_id", sb.ID,
		"host", host.Name,
		"state", sb.State,
		"ip_address", ip,
	)

	s.telemetry.Track("sandbox_create", map[string]any{
		"sandbox_id":   sb.ID,
		"base_image":   sb.BaseImage,
		"cpu":          cpu,
		"memory_mb":    memoryMB,
		"auto_start":   autoStart,
		"wait_for_ip":  waitForIP,
		"agent_id":     agentID,
		"host_name":    host.Name,
		"host_address": host.Address,
		"success":      true,
	})

	return sb, ip, nil
}

func (s *Service) GetSandboxes(ctx context.Context, filter store.SandboxFilter, opts *store.ListOptions) ([]*store.Sandbox, error) {
	return s.store.ListSandboxes(ctx, filter, opts)
}

// GetSandbox retrieves a single sandbox by ID.
func (s *Service) GetSandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	if strings.TrimSpace(sandboxID) == "" {
		return nil, fmt.Errorf("sandboxID is required")
	}
	return s.store.GetSandbox(ctx, sandboxID)
}

// GetSandboxCommands retrieves all commands executed in a sandbox.
func (s *Service) GetSandboxCommands(ctx context.Context, sandboxID string, opts *store.ListOptions) ([]*store.Command, error) {
	if strings.TrimSpace(sandboxID) == "" {
		return nil, fmt.Errorf("sandboxID is required")
	}
	// Verify sandbox exists
	if _, err := s.store.GetSandbox(ctx, sandboxID); err != nil {
		return nil, err
	}
	return s.store.ListCommands(ctx, sandboxID, opts)
}

// InjectSSHKey injects a public key for a user into the VM disk prior to boot.
func (s *Service) InjectSSHKey(ctx context.Context, sandboxID, username, publicKey string) error {
	if strings.TrimSpace(sandboxID) == "" {
		return fmt.Errorf("sandboxID is required")
	}
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}
	if strings.TrimSpace(publicKey) == "" {
		return fmt.Errorf("publicKey is required")
	}
	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return err
	}
	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)
	if err := mgr.InjectSSHKey(ctx, sb.SandboxName, username, publicKey); err != nil {
		return fmt.Errorf("inject ssh key: %w", err)
	}
	sb.UpdatedAt = s.timeNowFn().UTC()
	return s.store.UpdateSandbox(ctx, sb)
}

// StartSandbox boots the VM and optionally waits for IP discovery.
// Returns the discovered IP if waitForIP is true and discovery succeeds (empty string otherwise).
func (s *Service) StartSandbox(ctx context.Context, sandboxID string, waitForIP bool) (string, error) {
	if strings.TrimSpace(sandboxID) == "" {
		return "", fmt.Errorf("sandboxID is required")
	}

	s.logger.Info("starting sandbox",
		"sandbox_id", sandboxID,
		"wait_for_ip", waitForIP,
	)

	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return "", err
	}

	s.logger.Debug("sandbox found",
		"sandbox_name", sb.SandboxName,
		"current_state", sb.State,
		"host_name", sb.HostName,
		"host_address", sb.HostAddress,
	)

	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)

	if err := mgr.StartVM(ctx, sb.SandboxName); err != nil {
		s.logger.Error("failed to start VM",
			"sandbox_id", sb.ID,
			"sandbox_name", sb.SandboxName,
			"error", err,
		)
		_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateError, nil)
		return "", fmt.Errorf("start vm: %w", err)
	}

	// Update state -> STARTING
	if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateStarting, nil); err != nil {
		return "", err
	}

	var ip string
	if waitForIP {
		s.logger.Info("waiting for IP address",
			"sandbox_id", sb.ID,
			"timeout", s.cfg.IPDiscoveryTimeout,
		)

		var mac string
		ip, mac, err = mgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
		if err != nil {
			s.logger.Warn("IP discovery failed",
				"sandbox_id", sb.ID,
				"sandbox_name", sb.SandboxName,
				"error", err,
			)
			// Still mark as running even if we couldn't discover the IP
			_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil)
			return "", fmt.Errorf("get ip: %w", err)
		}

		// Validate IP uniqueness before storing
		if err := s.validateIPUniqueness(ctx, sb.ID, ip); err != nil {
			s.logger.Error("IP conflict during sandbox start",
				"sandbox_id", sb.ID,
				"sandbox_name", sb.SandboxName,
				"ip_address", ip,
				"mac_address", mac,
				"error", err,
			)
			_ = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil)
			return "", fmt.Errorf("ip conflict: %w", err)
		}

		if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, &ip); err != nil {
			return "", err
		}
	} else {
		if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, nil); err != nil {
			return "", err
		}
	}

	s.logger.Info("sandbox started",
		"sandbox_id", sb.ID,
		"ip_address", ip,
	)

	s.telemetry.Track("sandbox_start", map[string]any{
		"sandbox_id":  sb.ID,
		"wait_for_ip": waitForIP,
		"success":     true,
	})

	return ip, nil
}

// DiscoverIP attempts to discover the IP address for a sandbox.
// This is useful for async workflows where wait_for_ip was false during start.
// Returns the discovered IP address, or an error if discovery fails.
func (s *Service) DiscoverIP(ctx context.Context, sandboxID string) (string, error) {
	if strings.TrimSpace(sandboxID) == "" {
		return "", fmt.Errorf("sandboxID is required")
	}

	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return "", err
	}

	// Check if VM is in a state where IP discovery makes sense
	if sb.State != store.SandboxStateRunning && sb.State != store.SandboxStateStarting {
		return "", fmt.Errorf("sandbox is in state %s, must be running or starting for IP discovery", sb.State)
	}

	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)

	s.logger.Info("discovering IP for sandbox",
		"sandbox_id", sandboxID,
		"sandbox_name", sb.SandboxName,
	)

	ip, mac, err := mgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
	if err != nil {
		return "", fmt.Errorf("ip discovery failed: %w", err)
	}

	// Validate IP uniqueness
	if err := s.validateIPUniqueness(ctx, sb.ID, ip); err != nil {
		s.logger.Warn("IP conflict during discovery",
			"sandbox_id", sb.ID,
			"ip_address", ip,
			"mac_address", mac,
			"error", err,
		)
		return "", fmt.Errorf("ip conflict: %w", err)
	}

	// Update the sandbox with the discovered IP
	if err := s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateRunning, &ip); err != nil {
		return "", fmt.Errorf("persist ip: %w", err)
	}

	s.logger.Info("IP discovered and stored",
		"sandbox_id", sandboxID,
		"ip_address", ip,
		"mac_address", mac,
	)

	return ip, nil
}

// StopSandbox gracefully shuts down the VM or forces if force is true.
func (s *Service) StopSandbox(ctx context.Context, sandboxID string, force bool) error {
	if strings.TrimSpace(sandboxID) == "" {
		return fmt.Errorf("sandboxID is required")
	}
	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return err
	}
	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)
	if err := mgr.StopVM(ctx, sb.SandboxName, force); err != nil {
		return fmt.Errorf("stop vm: %w", err)
	}
	err = s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateStopped, sb.IPAddress)
	if err == nil {
		s.telemetry.Track("sandbox_stop", map[string]any{
			"sandbox_id": sb.ID,
			"force":      force,
			"success":    true,
		})
	}
	return err
}

// DestroySandbox forcibly destroys and undefines the VM and removes its workspace.
// The sandbox is then soft-deleted from the store. Returns the sandbox info after destruction.
func (s *Service) DestroySandbox(ctx context.Context, sandboxID string) (*store.Sandbox, error) {
	if strings.TrimSpace(sandboxID) == "" {
		return nil, fmt.Errorf("sandboxID is required")
	}
	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	// Cleanup managed SSH keys for this sandbox (non-fatal if it fails)
	if s.keyMgr != nil {
		if err := s.keyMgr.CleanupSandbox(ctx, sandboxID); err != nil {
			s.logger.Warn("failed to cleanup SSH keys",
				"sandbox_id", sandboxID,
				"error", err,
			)
		}
	}

	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)
	if err := mgr.DestroyVM(ctx, sb.SandboxName); err != nil {
		return nil, fmt.Errorf("destroy vm: %w", err)
	}
	if err := s.store.DeleteSandbox(ctx, sandboxID); err != nil {
		return nil, err
	}
	// Update state to reflect destruction
	sb.State = store.SandboxStateDestroyed

	s.telemetry.Track("sandbox_destroy", map[string]any{
		"sandbox_id": sandboxID,
		"success":    true,
	})

	return sb, nil
}

// CreateSnapshot creates a snapshot and persists a Snapshot record.
func (s *Service) CreateSnapshot(ctx context.Context, sandboxID, name string, external bool) (*store.Snapshot, error) {
	if strings.TrimSpace(sandboxID) == "" || strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("sandboxID and name are required")
	}
	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return nil, err
	}
	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)
	ref, err := mgr.CreateSnapshot(ctx, sb.SandboxName, name, external)
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	sn := &store.Snapshot{
		ID:        fmt.Sprintf("SNP-%s", shortID()),
		SandboxID: sb.ID,
		Name:      ref.Name,
		Kind:      snapshotKindFromString(ref.Kind),
		Ref:       ref.Ref,
		CreatedAt: s.timeNowFn().UTC(),
	}
	if err := s.store.CreateSnapshot(ctx, sn); err != nil {
		return nil, err
	}

	s.telemetry.Track("snapshot_create", map[string]any{
		"sandbox_id":    sandboxID,
		"snapshot_name": name,
		"snapshot_kind": ref.Kind,
		"external":      external,
		"success":       true,
	})

	return sn, nil
}

// DiffSnapshots computes a normalized change set between two snapshots and persists a Diff.
// Note: This implementation currently aggregates command history into CommandsRun and
// leaves file/package/service diffs empty. A dedicated diff engine should populate these fields
// by mounting snapshots and computing differences.
func (s *Service) DiffSnapshots(ctx context.Context, sandboxID, from, to string) (*store.Diff, error) {
	if strings.TrimSpace(sandboxID) == "" || strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return nil, fmt.Errorf("sandboxID, from, to are required")
	}
	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)

	// Best-effort: get a plan (notes/instructions) from manager; ignore failure.
	_, _ = mgr.DiffSnapshot(ctx, sb.SandboxName, from, to)

	// For now, compose CommandsRun from command history as partial diff signal.
	cmds, err := s.store.ListCommands(ctx, sandboxID, &store.ListOptions{OrderBy: "started_at", Asc: true})
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("list commands: %w", err)
	}
	var cr []store.CommandSummary
	for _, c := range cmds {
		cr = append(cr, store.CommandSummary{
			Cmd:      c.Command,
			ExitCode: c.ExitCode,
			At:       c.EndedAt,
		})
	}

	diff := &store.Diff{
		ID:           fmt.Sprintf("DIF-%s", shortID()),
		SandboxID:    sandboxID,
		FromSnapshot: from,
		ToSnapshot:   to,
		DiffJSON: store.ChangeDiff{
			FilesModified:   []string{},
			FilesAdded:      []string{},
			FilesRemoved:    []string{},
			PackagesAdded:   []store.PackageInfo{},
			PackagesRemoved: []store.PackageInfo{},
			ServicesChanged: []store.ServiceChange{},
			CommandsRun:     cr,
		},
		CreatedAt: s.timeNowFn().UTC(),
	}
	if err := s.store.SaveDiff(ctx, diff); err != nil {
		return nil, err
	}

	s.telemetry.Track("snapshot_diff", map[string]any{
		"sandbox_id":    sandboxID,
		"from_snapshot": from,
		"to_snapshot":   to,
		"success":       true,
	})

	return diff, nil
}

// RunCommand executes a command inside the sandbox via SSH.
// If privateKeyPath is empty and a key manager is configured, managed credentials will be used.
// Otherwise, username and privateKeyPath are required for SSH auth.
func (s *Service) RunCommand(ctx context.Context, sandboxID, username, privateKeyPath, command string, timeout time.Duration, env map[string]string) (*store.Command, error) {
	return s.RunCommandWithCallback(ctx, sandboxID, username, privateKeyPath, command, timeout, env, nil)
}

// RunCommandWithCallback executes a command inside the sandbox via SSH with optional streaming output.
// If outputCallback is non-nil, it will be called for each chunk of output as it arrives.
// The full output is still returned in the Command result after the command completes.
func (s *Service) RunCommandWithCallback(ctx context.Context, sandboxID, username, privateKeyPath, command string, timeout time.Duration, env map[string]string, outputCallback OutputCallback) (*store.Command, error) {
	if strings.TrimSpace(sandboxID) == "" {
		return nil, fmt.Errorf("sandboxID is required")
	}
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command is required")
	}
	if timeout <= 0 {
		timeout = s.cfg.CommandTimeout
	}

	// Determine if we're using managed credentials
	var useManagedCreds bool
	var certPath string
	if strings.TrimSpace(privateKeyPath) == "" {
		if s.keyMgr == nil {
			return nil, fmt.Errorf("privateKeyPath is required (no key manager configured)")
		}
		useManagedCreds = true
		// Default username for managed credentials
		if strings.TrimSpace(username) == "" {
			username = "sandbox"
		}
	} else {
		// Traditional mode: username is required
		if strings.TrimSpace(username) == "" {
			return nil, fmt.Errorf("username is required")
		}
	}

	sb, err := s.store.GetSandbox(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	// Get the appropriate manager (local or remote) for this sandbox
	mgr := s.getManagerForSandbox(sb)

	// Always re-discover IP to ensure we have the correct one for THIS sandbox.
	// This is important because:
	// 1. Cached IPs might be stale if the VM was restarted
	// 2. Another sandbox might have been assigned the same IP erroneously
	// 3. DHCP leases can change
	ip, mac, err := mgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
	if err != nil {
		return nil, fmt.Errorf("discover ip for sandbox %s (vm: %s): %w", sb.ID, sb.SandboxName, err)
	}

	// Check if this IP is already assigned to a DIFFERENT running sandbox
	if err := s.validateIPUniqueness(ctx, sb.ID, ip); err != nil {
		s.logger.Warn("IP conflict detected",
			"sandbox_id", sb.ID,
			"sandbox_name", sb.SandboxName,
			"ip_address", ip,
			"mac_address", mac,
			"error", err,
		)
		return nil, fmt.Errorf("ip conflict: %w", err)
	}

	// Update IP if it changed or wasn't set
	if sb.IPAddress == nil || *sb.IPAddress != ip {
		if err := s.store.UpdateSandboxState(ctx, sb.ID, sb.State, &ip); err != nil {
			return nil, fmt.Errorf("persist ip: %w", err)
		}
	}

	// Get managed credentials if needed
	if useManagedCreds {
		creds, err := s.keyMgr.GetCredentials(ctx, sandboxID, username)
		if err != nil {
			return nil, fmt.Errorf("get managed credentials: %w", err)
		}
		privateKeyPath = creds.PrivateKeyPath
		certPath = creds.CertificatePath
		username = creds.Username
	}

	cmdID := fmt.Sprintf("CMD-%s", shortID())
	now := s.timeNowFn().UTC()

	// Encode environment for persistence.
	var envJSON *string
	if len(env) > 0 {
		b, _ := json.Marshal(env)
		tmp := string(b)
		envJSON = &tmp
	}

	// Determine proxy jump - if sandbox is on remote host, use that host as jump
	proxyJump := ""
	if sb.HostAddress != nil && *sb.HostAddress != "" {
		// Format: user@host for SSH ProxyJump
		proxyJump = fmt.Sprintf("root@%s", *sb.HostAddress)
	}

	// Execute SSH command
	var stdout, stderr string
	var code int
	var runErr error

	if outputCallback != nil {
		// Use streaming variant
		outputChan := make(chan OutputChunk, 100)

		// Goroutine to forward chunks to callback
		go func() {
			for chunk := range outputChan {
				outputCallback(chunk)
			}
		}()

		if useManagedCreds {
			stdout, stderr, code, runErr = s.ssh.RunWithCertStreaming(ctx, ip, username, privateKeyPath, certPath, commandWithEnv(command, env), timeout, env, proxyJump, outputChan)
		} else {
			stdout, stderr, code, runErr = s.ssh.RunStreaming(ctx, ip, username, privateKeyPath, commandWithEnv(command, env), timeout, env, proxyJump, outputChan)
		}
		close(outputChan)
	} else {
		// Use existing non-streaming variant
		if useManagedCreds {
			stdout, stderr, code, runErr = s.ssh.RunWithCert(ctx, ip, username, privateKeyPath, certPath, commandWithEnv(command, env), timeout, env, proxyJump)
		} else {
			stdout, stderr, code, runErr = s.ssh.Run(ctx, ip, username, privateKeyPath, commandWithEnv(command, env), timeout, env, proxyJump)
		}
	}

	cmd := &store.Command{
		ID:        cmdID,
		SandboxID: sandboxID,
		Command:   command,
		EnvJSON:   envJSON,
		Stdout:    stdout,
		Stderr:    stderr,
		ExitCode:  code,
		StartedAt: now,
		EndedAt:   s.timeNowFn().UTC(),
	}
	if err := s.store.SaveCommand(ctx, cmd); err != nil {
		return nil, fmt.Errorf("save command: %w", err)
	}

	s.telemetry.Track("sandbox_command", map[string]any{
		"sandbox_id":  sandboxID,
		"command_id":  cmdID,
		"exit_code":   code,
		"duration_ms": cmd.EndedAt.Sub(cmd.StartedAt).Milliseconds(),
		"success":     true,
	})

	if runErr != nil {
		return cmd, fmt.Errorf("ssh run: %w", runErr)
	}
	return cmd, nil
}

// OutputChunk represents a piece of streaming output from a command
type OutputChunk struct {
	Data     []byte
	IsStderr bool
	IsRetry  bool
	Retry    *RetryInfo
}

// RetryInfo contains details about a retry attempt
type RetryInfo struct {
	Attempt int
	Max     int
	Delay   time.Duration
	Error   string
}

// OutputCallback is called for each chunk of streaming output
type OutputCallback func(chunk OutputChunk)

// SSHRunner executes commands on a remote host via SSH.
type SSHRunner interface {
	// Run executes command on user@addr using the provided private key file.
	// Returns stdout, stderr, and exit code. Implementations should use StrictHostKeyChecking=no
	// or a known_hosts strategy appropriate for ephemeral sandboxes.
	// proxyJump is optional - if non-empty, SSH will jump through that host.
	Run(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (stdout, stderr string, exitCode int, err error)

	// RunWithCert executes command using certificate-based authentication.
	// The certPath should point to the SSH certificate file (key-cert.pub).
	// proxyJump is optional - if non-empty, SSH will jump through that host.
	RunWithCert(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (stdout, stderr string, exitCode int, err error)

	// RunStreaming executes command with streaming output sent to outputChan.
	// proxyJump is optional - if non-empty, SSH will jump through that host.
	RunStreaming(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string, outputChan chan<- OutputChunk) (stdout, stderr string, exitCode int, err error)

	// RunWithCertStreaming executes command using certificate-based authentication with streaming output.
	// proxyJump is optional - if non-empty, SSH will jump through that host.
	RunWithCertStreaming(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string, outputChan chan<- OutputChunk) (stdout, stderr string, exitCode int, err error)
}

// DefaultSSHRunner is a simple implementation backed by the system's ssh binary.
type DefaultSSHRunner struct {
	// Logger for retry and connection status
	Logger *slog.Logger

	// DefaultProxyJump specifies a default jump host for SSH connections.
	// Can be overridden per-call via the proxyJump parameter.
	// Format: "user@host:port" or just "host" for default user/port.
	DefaultProxyJump string

	// MaxRetries is the maximum number of retry attempts for transient SSH failures.
	// Default: 5
	MaxRetries int

	// InitialRetryDelay is the initial delay before the first retry.
	// Default: 2s
	InitialRetryDelay time.Duration

	// MaxRetryDelay is the maximum delay between retries.
	// Default: 30s
	MaxRetryDelay time.Duration
}

// sshRetryConfig returns the retry configuration with defaults applied.
func (r *DefaultSSHRunner) sshRetryConfig() (maxRetries int, initialDelay, maxDelay time.Duration) {
	maxRetries = r.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 5
	}
	initialDelay = r.InitialRetryDelay
	if initialDelay <= 0 {
		initialDelay = 2 * time.Second
	}
	maxDelay = r.MaxRetryDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	return
}

// isRetryableSSHError checks if the error indicates a transient SSH failure
// that should be retried (e.g., connection refused, sshd not ready).
func isRetryableSSHError(stderr string, exitCode int) bool {
	// Exit code 255 indicates SSH connection failure
	if exitCode != 255 {
		return false
	}
	// Check for common transient connection errors
	retryablePatterns := []string{
		"Connection refused",
		"Connection closed",
		"Connection reset",
		"Connection timed out",
		"No route to host",
		"Network is unreachable",
		"Host is down",
		"port 22: Connection refused",
		"port 65535", // Malformed connection error
		"UNKNOWN",    // SSH parsing error during connection failure
	}
	stderrLower := strings.ToLower(stderr)
	for _, pattern := range retryablePatterns {
		if strings.Contains(stderrLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// Run implements SSHRunner.Run using the local ssh client.
// It disables strict host key checking and sets a connect timeout.
// It assumes the VM is reachable on the default SSH port (22).
// Includes retry logic with exponential backoff for transient connection failures.
func (r *DefaultSSHRunner) Run(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, _ map[string]string, proxyJump string) (string, string, int, error) {
	// Pre-flight check: verify the private key file exists and has correct permissions
	keyInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", 255, fmt.Errorf("ssh key file not found: %s", privateKeyPath)
		}
		return "", "", 255, fmt.Errorf("ssh key file error: %w", err)
	}
	// Check permissions - SSH keys should not be world-readable
	if keyInfo.Mode().Perm()&0o077 != 0 {
		return "", "", 255, fmt.Errorf("ssh key file %s has insecure permissions %o (should be 0600 or stricter)", privateKeyPath, keyInfo.Mode().Perm())
	}

	if _, ok := ctx.Deadline(); !ok && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	args := []string{
		"-i", privateKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=15",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=1000",
	}
	// Add ProxyJump if provided, otherwise use default
	effectiveProxyJump := proxyJump
	if effectiveProxyJump == "" {
		effectiveProxyJump = r.DefaultProxyJump
	}
	if effectiveProxyJump != "" {
		args = append(args, "-J", effectiveProxyJump)
	}
	args = append(args,
		fmt.Sprintf("%s@%s", user, addr),
		"--",
		command,
	)

	maxRetries, initialDelay, maxDelay := r.sshRetryConfig()
	var lastStdout, lastStderr string
	var lastExitCode int
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return lastStdout, lastStderr, lastExitCode, fmt.Errorf("context cancelled after %d attempts: %w", attempt, ctx.Err())
		}

		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, "ssh", args...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()
		exitCode := 0
		if err != nil {
			// Best-effort extract exit code
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 255
			}
			stderrStr := stderr.String()

			// Check if this is a retryable error
			if attempt < maxRetries && isRetryableSSHError(stderrStr, exitCode) {
				// Calculate backoff delay: 2s, 4s, 8s, 16s, 30s (capped)
				delay := initialDelay * time.Duration(1<<uint(attempt))
				if delay > maxDelay {
					delay = maxDelay
				}
				if r.Logger != nil {
					r.Logger.Warn("SSH connection failed, retrying",
						"attempt", attempt+1,
						"max_retries", maxRetries,
						"delay", delay,
						"addr", addr,
						"stderr", stderrStr,
					)
				}
				select {
				case <-time.After(delay):
					// Continue to next attempt
				case <-ctx.Done():
					return stdout.String(), stderrStr, exitCode, fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
				}
				lastStdout, lastStderr, lastExitCode, lastErr = stdout.String(), stderrStr, exitCode, err
				continue
			}

			// Not retryable or max retries exceeded
			if stderrStr != "" {
				err = fmt.Errorf("%w: %s", err, stderrStr)
			}
			return stdout.String(), stderrStr, exitCode, err
		}

		// Success
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}

	// Should not reach here, but return last error if we do
	return lastStdout, lastStderr, lastExitCode, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// RunWithCert implements SSHRunner.RunWithCert using the local ssh client with certificate auth.
// Includes retry logic with exponential backoff for transient connection failures.
func (r *DefaultSSHRunner) RunWithCert(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, _ map[string]string, proxyJump string) (string, string, int, error) {
	// Pre-flight check: verify the private key file exists and has correct permissions
	keyInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", 255, fmt.Errorf("ssh key file not found: %s", privateKeyPath)
		}
		return "", "", 255, fmt.Errorf("ssh key file error: %w", err)
	}
	if keyInfo.Mode().Perm()&0o077 != 0 {
		return "", "", 255, fmt.Errorf("ssh key file %s has insecure permissions %o (should be 0600 or stricter)", privateKeyPath, keyInfo.Mode().Perm())
	}

	// Check certificate file exists
	if _, err := os.Stat(certPath); err != nil {
		if os.IsNotExist(err) {
			return "", "", 255, fmt.Errorf("ssh certificate file not found: %s", certPath)
		}
		return "", "", 255, fmt.Errorf("ssh certificate file error: %w", err)
	}

	if _, ok := ctx.Deadline(); !ok && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	args := []string{
		"-i", privateKeyPath,
		"-o", fmt.Sprintf("CertificateFile=%s", certPath),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=15",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=1000",
	}
	// Add ProxyJump if provided, otherwise use default
	effectiveProxyJump := proxyJump
	if effectiveProxyJump == "" {
		effectiveProxyJump = r.DefaultProxyJump
	}
	if effectiveProxyJump != "" {
		args = append(args, "-J", effectiveProxyJump)
	}
	args = append(args,
		fmt.Sprintf("%s@%s", user, addr),
		"--",
		command,
	)

	maxRetries, initialDelay, maxDelay := r.sshRetryConfig()
	var lastStdout, lastStderr string
	var lastExitCode int
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return lastStdout, lastStderr, lastExitCode, fmt.Errorf("context cancelled after %d attempts: %w", attempt, ctx.Err())
		}

		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, "ssh", args...)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()
		exitCode := 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				exitCode = ee.ExitCode()
			} else {
				exitCode = 255
			}
			stderrStr := stderr.String()

			// Check if this is a retryable error
			if attempt < maxRetries && isRetryableSSHError(stderrStr, exitCode) {
				// Calculate backoff delay: 2s, 4s, 8s, 16s, 30s (capped)
				delay := initialDelay * time.Duration(1<<uint(attempt))
				if delay > maxDelay {
					delay = maxDelay
				}
				if r.Logger != nil {
					r.Logger.Warn("SSH connection failed (cert auth), retrying",
						"attempt", attempt+1,
						"max_retries", maxRetries,
						"delay", delay,
						"addr", addr,
						"stderr", stderrStr,
					)
				}
				select {
				case <-time.After(delay):
					// Continue to next attempt
				case <-ctx.Done():
					return stdout.String(), stderrStr, exitCode, fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
				}
				lastStdout, lastStderr, lastExitCode, lastErr = stdout.String(), stderrStr, exitCode, err
				continue
			}

			// Not retryable or max retries exceeded
			if stderrStr != "" {
				err = fmt.Errorf("%w: %s", err, stderrStr)
			}
			return stdout.String(), stderrStr, exitCode, err
		}

		// Success
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}

	// Should not reach here, but return last error if we do
	return lastStdout, lastStderr, lastExitCode, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// SourceCommandResult holds the output from a read-only command on a source VM.
type SourceCommandResult struct {
	SourceVM string
	ExitCode int
	Stdout   string
	Stderr   string
}

// RunSourceVMCommand executes a validated read-only command on a golden/source VM.
// The command is validated against the readonly allowlist before execution.
// Results are NOT persisted to the store.
func (s *Service) RunSourceVMCommand(ctx context.Context, sourceVMName, command string, timeout time.Duration) (*SourceCommandResult, error) {
	return s.RunSourceVMCommandWithCallback(ctx, sourceVMName, command, timeout, nil)
}

// RunSourceVMCommandWithCallback executes a validated read-only command on a golden/source VM with streaming.
func (s *Service) RunSourceVMCommandWithCallback(ctx context.Context, sourceVMName, command string, timeout time.Duration, outputCallback OutputCallback) (*SourceCommandResult, error) {
	if strings.TrimSpace(sourceVMName) == "" {
		return nil, fmt.Errorf("sourceVMName is required")
	}
	if strings.TrimSpace(command) == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Validate command against the read-only allowlist.
	if err := readonly.ValidateCommand(command); err != nil {
		s.logger.Warn("source VM command blocked by allowlist",
			"source_vm", sourceVMName,
			"command", command,
			"reason", err.Error(),
		)
		s.telemetry.Track("source_vm_command_blocked", map[string]any{
			"source_vm": sourceVMName,
			"reason":    err.Error(),
		})
		return nil, fmt.Errorf("command not allowed in read-only mode: %w", err)
	}

	if timeout <= 0 {
		timeout = s.cfg.CommandTimeout
	}

	// Require key manager for source VM access.
	if s.keyMgr == nil {
		return nil, fmt.Errorf("key manager is required for source VM access")
	}

	// Look up source VM host info from store for remote IP discovery and proxy jump.
	var remoteHost *config.HostConfig
	if s.store != nil {
		if svm, err := s.store.GetSourceVM(ctx, sourceVMName); err == nil && svm.HostAddress != nil && *svm.HostAddress != "" {
			remoteHost = &config.HostConfig{
				Name:    derefStr(svm.HostName),
				Address: *svm.HostAddress,
			}
		}
	}

	// Discover VM IP using remote manager if VM is on a remote host.
	ipMgr := s.mgr
	if remoteHost != nil && s.remoteManagerFactory != nil {
		ipMgr = s.remoteManagerFactory(*remoteHost)
	}
	ip, _, err := ipMgr.GetIPAddress(ctx, sourceVMName, s.cfg.IPDiscoveryTimeout)
	if err != nil {
		return nil, fmt.Errorf("discover IP for source VM %s: %w", sourceVMName, err)
	}

	// Get read-only credentials.
	creds, err := s.keyMgr.GetSourceVMCredentials(ctx, sourceVMName)
	if err != nil {
		return nil, fmt.Errorf("get source VM credentials: %w", err)
	}

	// Determine proxy jump. Use remote host as jump host if VM is on a remote network.
	proxyJump := s.cfg.SSHProxyJump
	if remoteHost != nil && proxyJump == "" {
		sshUser := remoteHost.SSHUser
		if sshUser == "" {
			sshUser = "root"
		}
		proxyJump = fmt.Sprintf("%s@%s", sshUser, remoteHost.Address)
	}

	// Pass command directly - the restricted shell on source VMs handles execution.
	var stdout, stderr string
	var code int
	var runErr error

	if outputCallback != nil {
		outputChan := make(chan OutputChunk, 100)
		go func() {
			for chunk := range outputChan {
				outputCallback(chunk)
			}
		}()
		stdout, stderr, code, runErr = s.ssh.RunWithCertStreaming(ctx, ip, creds.Username, creds.PrivateKeyPath, creds.CertificatePath, command, timeout, nil, proxyJump, outputChan)
		close(outputChan)
	} else {
		stdout, stderr, code, runErr = s.ssh.RunWithCert(ctx, ip, creds.Username, creds.PrivateKeyPath, creds.CertificatePath, command, timeout, nil, proxyJump)
	}

	result := &SourceCommandResult{
		SourceVM: sourceVMName,
		ExitCode: code,
		Stdout:   stdout,
		Stderr:   stderr,
	}

	if code == 126 {
		s.logger.Warn("source VM command blocked by restricted shell",
			"source_vm", sourceVMName,
			"command", command,
			"stderr", stderr,
		)
		s.telemetry.Track("source_vm_command_blocked", map[string]any{
			"source_vm": sourceVMName,
			"reason":    "restricted_shell",
		})
	}

	if runErr != nil {
		return result, fmt.Errorf("ssh run on source VM: %w", runErr)
	}

	s.telemetry.Track("source_vm_command", map[string]any{
		"source_vm": sourceVMName,
		"exit_code": code,
	})

	return result, nil
}

// Helpers

func snapshotKindFromString(k string) store.SnapshotKind {
	switch strings.ToUpper(k) {
	case "EXTERNAL":
		return store.SnapshotKindExternal
	default:
		return store.SnapshotKindInternal
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func shortID() string {
	id := uuid.NewString()
	if i := strings.IndexByte(id, '-'); i > 0 {
		return id[:i]
	}
	return id
}

func commandWithEnv(cmd string, env map[string]string) string {
	if len(env) == 0 {
		// Execute in login shell to emulate typical interactive environment
		return fmt.Sprintf("bash -lc %q", cmd)
	}
	var exports []string
	for k, v := range env {
		exports = append(exports, fmt.Sprintf(`export %s=%s`, safeShellIdent(k), shellQuote(v)))
	}
	preamble := strings.Join(exports, "; ") + "; "
	return fmt.Sprintf("bash -lc %q", preamble+cmd)
}

func shellQuote(s string) string {
	// Basic single-quote shell escaping
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func safeShellIdent(s string) string {
	// Allow alnum and underscore, replace others with underscore
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "VAR"
	}
	return out
}

// RunStreaming implements SSHRunner.RunStreaming using the local ssh client with streaming output.
// Only streams output on the final retry attempt for cleaner UX.
func (r *DefaultSSHRunner) RunStreaming(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, _ map[string]string, proxyJump string, outputChan chan<- OutputChunk) (string, string, int, error) {
	// Pre-flight check: verify the private key file exists and has correct permissions
	keyInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", 255, fmt.Errorf("ssh key file not found: %s", privateKeyPath)
		}
		return "", "", 255, fmt.Errorf("ssh key file error: %w", err)
	}
	if keyInfo.Mode().Perm()&0o077 != 0 {
		return "", "", 255, fmt.Errorf("ssh key file %s has insecure permissions %o (should be 0600 or stricter)", privateKeyPath, keyInfo.Mode().Perm())
	}

	if _, ok := ctx.Deadline(); !ok && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	args := []string{
		"-i", privateKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=15",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=1000",
	}
	effectiveProxyJump := proxyJump
	if effectiveProxyJump == "" {
		effectiveProxyJump = r.DefaultProxyJump
	}
	if effectiveProxyJump != "" {
		args = append(args, "-J", effectiveProxyJump)
	}
	args = append(args,
		fmt.Sprintf("%s@%s", user, addr),
		"--",
		command,
	)

	maxRetries, initialDelay, maxDelay := r.sshRetryConfig()
	var lastStdout, lastStderr string
	var lastExitCode int
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return lastStdout, lastStderr, lastExitCode, fmt.Errorf("context cancelled after %d attempts: %w", attempt, ctx.Err())
		}

		// Signal retry to clear previous output (attempt > 0 means this is a retry)
		if attempt > 0 && outputChan != nil {
			select {
			case outputChan <- OutputChunk{Data: nil, IsStderr: false}: // nil data signals reset
			default:
			}
		}

		// Always stream output - each attempt streams, retries override previous
		stdout, stderr, exitCode, err := r.runSingleAttemptStreaming(ctx, args, outputChan)

		if err == nil {
			return stdout, stderr, exitCode, nil
		}

		// Check if this is a retryable error
		isLastAttempt := attempt == maxRetries
		if !isLastAttempt && isRetryableSSHError(stderr, exitCode) {
			delay := initialDelay * time.Duration(1<<uint(attempt))
			if delay > maxDelay {
				delay = maxDelay
			}
			if r.Logger != nil {
				r.Logger.Warn("SSH connection failed, retrying",
					"attempt", attempt+1,
					"max_retries", maxRetries,
					"delay", delay,
					"addr", addr,
					"stderr", stderr,
				)
			}

			// Send retry notification to TUI
			if outputChan != nil {
				select {
				case outputChan <- OutputChunk{
					IsRetry: true,
					Retry: &RetryInfo{
						Attempt: attempt + 1,
						Max:     maxRetries,
						Delay:   delay,
						Error:   stderr,
					},
				}:
				default:
				}
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return stdout, stderr, exitCode, fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
			}
			lastStdout, lastStderr, lastExitCode, lastErr = stdout, stderr, exitCode, err
			continue
		}

		if stderr != "" {
			err = fmt.Errorf("%w: %s", err, stderr)
		}
		return stdout, stderr, exitCode, err
	}

	return lastStdout, lastStderr, lastExitCode, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// RunWithCertStreaming implements SSHRunner.RunWithCertStreaming using certificate-based auth with streaming.
func (r *DefaultSSHRunner) RunWithCertStreaming(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, _ map[string]string, proxyJump string, outputChan chan<- OutputChunk) (string, string, int, error) {
	// Pre-flight check: verify the private key file exists and has correct permissions
	keyInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", 255, fmt.Errorf("ssh key file not found: %s", privateKeyPath)
		}
		return "", "", 255, fmt.Errorf("ssh key file error: %w", err)
	}
	if keyInfo.Mode().Perm()&0o077 != 0 {
		return "", "", 255, fmt.Errorf("ssh key file %s has insecure permissions %o (should be 0600 or stricter)", privateKeyPath, keyInfo.Mode().Perm())
	}

	if _, err := os.Stat(certPath); err != nil {
		if os.IsNotExist(err) {
			return "", "", 255, fmt.Errorf("ssh certificate file not found: %s", certPath)
		}
		return "", "", 255, fmt.Errorf("ssh certificate file error: %w", err)
	}

	if _, ok := ctx.Deadline(); !ok && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	args := []string{
		"-i", privateKeyPath,
		"-o", fmt.Sprintf("CertificateFile=%s", certPath),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=15",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=1000",
	}
	effectiveProxyJump := proxyJump
	if effectiveProxyJump == "" {
		effectiveProxyJump = r.DefaultProxyJump
	}
	if effectiveProxyJump != "" {
		args = append(args, "-J", effectiveProxyJump)
	}
	args = append(args,
		fmt.Sprintf("%s@%s", user, addr),
		"--",
		command,
	)

	maxRetries, initialDelay, maxDelay := r.sshRetryConfig()
	var lastStdout, lastStderr string
	var lastExitCode int
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return lastStdout, lastStderr, lastExitCode, fmt.Errorf("context cancelled after %d attempts: %w", attempt, ctx.Err())
		}

		// Signal retry to clear previous output (attempt > 0 means this is a retry)
		if attempt > 0 && outputChan != nil {
			select {
			case outputChan <- OutputChunk{Data: nil, IsStderr: false}: // nil data signals reset
			default:
			}
		}

		// Always stream output - each attempt streams, retries override previous
		stdout, stderr, exitCode, err := r.runSingleAttemptStreaming(ctx, args, outputChan)

		if err == nil {
			return stdout, stderr, exitCode, nil
		}

		// Check if this is a retryable error
		isLastAttempt := attempt == maxRetries
		if !isLastAttempt && isRetryableSSHError(stderr, exitCode) {
			delay := initialDelay * time.Duration(1<<uint(attempt))
			if delay > maxDelay {
				delay = maxDelay
			}
			if r.Logger != nil {
				r.Logger.Warn("SSH connection failed (cert auth), retrying",
					"attempt", attempt+1,
					"max_retries", maxRetries,
					"delay", delay,
					"addr", addr,
					"stderr", stderr,
				)
			}

			// Send retry notification to TUI
			if outputChan != nil {
				select {
				case outputChan <- OutputChunk{
					IsRetry: true,
					Retry: &RetryInfo{
						Attempt: attempt + 1,
						Max:     maxRetries,
						Delay:   delay,
						Error:   stderr,
					},
				}:
				default:
				}
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return stdout, stderr, exitCode, fmt.Errorf("context cancelled during retry backoff: %w", ctx.Err())
			}
			lastStdout, lastStderr, lastExitCode, lastErr = stdout, stderr, exitCode, err
			continue
		}

		if stderr != "" {
			err = fmt.Errorf("%w: %s", err, stderr)
		}
		return stdout, stderr, exitCode, err
	}

	return lastStdout, lastStderr, lastExitCode, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, lastErr)
}

// runSingleAttemptStreaming executes a single SSH attempt with optional streaming output.
func (r *DefaultSSHRunner) runSingleAttemptStreaming(ctx context.Context, args []string, outputChan chan<- OutputChunk) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, "ssh", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", 255, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", "", 255, fmt.Errorf("stderr pipe: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	if err := cmd.Start(); err != nil {
		return "", "", 255, fmt.Errorf("start command: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		streamPipe(stdoutPipe, &stdoutBuf, outputChan, false)
	}()

	go func() {
		defer wg.Done()
		streamPipe(stderrPipe, &stderrBuf, outputChan, true)
	}()

	wg.Wait()

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			exitCode = 255
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode, err
}

// streamPipe reads from a pipe and sends chunks to the output channel while accumulating in buffer.
func streamPipe(pipe io.Reader, buf *bytes.Buffer, outputChan chan<- OutputChunk, isStderr bool) {
	reader := bufio.NewReader(pipe)
	chunk := make([]byte, 1024)

	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			data := make([]byte, n)
			copy(data, chunk[:n])
			buf.Write(data)

			if outputChan != nil {
				select {
				case outputChan <- OutputChunk{Data: data, IsStderr: isStderr}:
				default:
					// Don't block if channel is full
				}
			}
		}
		if err != nil {
			break
		}
	}
}
