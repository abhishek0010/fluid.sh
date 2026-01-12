package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"

	"virsh-sandbox/internal/libvirt"
	"virsh-sandbox/internal/sshkeys"
	"virsh-sandbox/internal/store"
)

// Service orchestrates libvirt operations and data persistence.
// It represents the main application layer for sandbox lifecycle, command exec,
// snapshotting, diffing, and artifact generation orchestration.
type Service struct {
	mgr       libvirt.Manager
	store     store.Store
	ssh       SSHRunner
	keyMgr    sshkeys.KeyProvider // Optional: manages SSH keys for RunCommand
	cfg       Config
	timeNowFn func() time.Time
	logger    *slog.Logger
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

// NewService constructs a VM service with the provided libvirt manager, store and config.
func NewService(mgr libvirt.Manager, st store.Store, cfg Config, opts ...Option) *Service {
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
	s := &Service{
		mgr:       mgr,
		store:     st,
		cfg:       cfg,
		ssh:       &DefaultSSHRunner{ProxyJump: cfg.SSHProxyJump},
		timeNowFn: time.Now,
		logger:    slog.Default(),
	}
	for _, o := range opts {
		o(s)
	}
	return s
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
	if err := s.mgr.InjectSSHKey(ctx, sb.SandboxName, username, publicKey); err != nil {
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
	)

	if err := s.mgr.StartVM(ctx, sb.SandboxName); err != nil {
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
		ip, mac, err = s.mgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
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

	s.logger.Info("discovering IP for sandbox",
		"sandbox_id", sandboxID,
		"sandbox_name", sb.SandboxName,
	)

	ip, mac, err := s.mgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
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
	if err := s.mgr.StopVM(ctx, sb.SandboxName, force); err != nil {
		return fmt.Errorf("stop vm: %w", err)
	}
	return s.store.UpdateSandboxState(ctx, sb.ID, store.SandboxStateStopped, sb.IPAddress)
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

	if err := s.mgr.DestroyVM(ctx, sb.SandboxName); err != nil {
		return nil, fmt.Errorf("destroy vm: %w", err)
	}
	if err := s.store.DeleteSandbox(ctx, sandboxID); err != nil {
		return nil, err
	}
	// Update state to reflect destruction
	sb.State = store.SandboxStateDestroyed
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
	ref, err := s.mgr.CreateSnapshot(ctx, sb.SandboxName, name, external)
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

	// Best-effort: get a plan (notes/instructions) from manager; ignore failure.
	_, _ = s.mgr.DiffSnapshot(ctx, sb.SandboxName, from, to)

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
	return diff, nil
}

// RunCommand executes a command inside the sandbox via SSH.
// If privateKeyPath is empty and a key manager is configured, managed credentials will be used.
// Otherwise, username and privateKeyPath are required for SSH auth.
func (s *Service) RunCommand(ctx context.Context, sandboxID, username, privateKeyPath, command string, timeout time.Duration, env map[string]string) (*store.Command, error) {
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

	// Always re-discover IP to ensure we have the correct one for THIS sandbox.
	// This is important because:
	// 1. Cached IPs might be stale if the VM was restarted
	// 2. Another sandbox might have been assigned the same IP erroneously
	// 3. DHCP leases can change
	ip, mac, err := s.mgr.GetIPAddress(ctx, sb.SandboxName, s.cfg.IPDiscoveryTimeout)
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

	// Execute SSH command
	var stdout, stderr string
	var code int
	var runErr error
	if useManagedCreds {
		stdout, stderr, code, runErr = s.ssh.RunWithCert(ctx, ip, username, privateKeyPath, certPath, commandWithEnv(command, env), timeout, env)
	} else {
		stdout, stderr, code, runErr = s.ssh.Run(ctx, ip, username, privateKeyPath, commandWithEnv(command, env), timeout, env)
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

	if runErr != nil {
		return cmd, fmt.Errorf("ssh run: %w", runErr)
	}
	return cmd, nil
}

// SSHRunner executes commands on a remote host via SSH.
type SSHRunner interface {
	// Run executes command on user@addr using the provided private key file.
	// Returns stdout, stderr, and exit code. Implementations should use StrictHostKeyChecking=no
	// or a known_hosts strategy appropriate for ephemeral sandboxes.
	Run(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string) (stdout, stderr string, exitCode int, err error)

	// RunWithCert executes command using certificate-based authentication.
	// The certPath should point to the SSH certificate file (key-cert.pub).
	RunWithCert(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string) (stdout, stderr string, exitCode int, err error)
}

// DefaultSSHRunner is a simple implementation backed by the system's ssh binary.
type DefaultSSHRunner struct {
	// ProxyJump specifies a jump host for SSH connections.
	// Format: "user@host:port" or just "host" for default user/port.
	// If empty, direct connections are made.
	ProxyJump string
}

// Run implements SSHRunner.Run using the local ssh client.
// It disables strict host key checking and sets a connect timeout.
// It assumes the VM is reachable on the default SSH port (22).
func (r *DefaultSSHRunner) Run(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, _ map[string]string) (string, string, int, error) {
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
	}
	// Add ProxyJump if configured
	if r.ProxyJump != "" {
		args = append(args, "-J", r.ProxyJump)
	}
	args = append(args,
		fmt.Sprintf("%s@%s", user, addr),
		"--",
		command,
	)

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
		// Include stderr in error message for better debugging
		stderrStr := stderr.String()
		if stderrStr != "" {
			err = fmt.Errorf("%w: %s", err, stderrStr)
		}
		return stdout.String(), stderrStr, exitCode, err
	}
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return stdout.String(), stderr.String(), exitCode, nil
}

// RunWithCert implements SSHRunner.RunWithCert using the local ssh client with certificate auth.
func (r *DefaultSSHRunner) RunWithCert(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, _ map[string]string) (string, string, int, error) {
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
	}
	// Add ProxyJump if configured
	if r.ProxyJump != "" {
		args = append(args, "-J", r.ProxyJump)
	}
	args = append(args,
		fmt.Sprintf("%s@%s", user, addr),
		"--",
		command,
	)

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
		if stderrStr != "" {
			err = fmt.Errorf("%w: %s", err, stderrStr)
		}
		return stdout.String(), stderrStr, exitCode, err
	}
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return stdout.String(), stderr.String(), exitCode, nil
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
