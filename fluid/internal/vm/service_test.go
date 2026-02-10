package vm

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/libvirt"
	"github.com/aspectrr/fluid.sh/fluid/internal/sshkeys"
	"github.com/aspectrr/fluid.sh/fluid/internal/store"
	"github.com/aspectrr/fluid.sh/fluid/internal/telemetry"
)

// mockStore implements store.Store for testing
type mockStore struct {
	getSandboxFn    func(ctx context.Context, id string) (*store.Sandbox, error)
	listCommandsFn  func(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error)
	listSandboxesFn func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error)
}

func (m *mockStore) Config() store.Config { return store.Config{} }
func (m *mockStore) Ping(ctx context.Context) error {
	return nil
}

func (m *mockStore) WithTx(ctx context.Context, fn func(tx store.DataStore) error) error {
	return fn(m)
}
func (m *mockStore) Close() error { return nil }

func (m *mockStore) CreateSandbox(ctx context.Context, sb *store.Sandbox) error {
	return nil
}

func (m *mockStore) GetSandbox(ctx context.Context, id string) (*store.Sandbox, error) {
	if m.getSandboxFn != nil {
		return m.getSandboxFn(ctx, id)
	}
	return nil, store.ErrNotFound
}

func (m *mockStore) GetSandboxByVMName(ctx context.Context, vmName string) (*store.Sandbox, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListSandboxes(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
	if m.listSandboxesFn != nil {
		return m.listSandboxesFn(ctx, filter, opt)
	}
	return nil, nil
}

func (m *mockStore) UpdateSandbox(ctx context.Context, sb *store.Sandbox) error {
	return nil
}

func (m *mockStore) UpdateSandboxState(ctx context.Context, id string, newState store.SandboxState, ipAddr *string) error {
	return nil
}

func (m *mockStore) DeleteSandbox(ctx context.Context, id string) error {
	return nil
}

func (m *mockStore) CreateSnapshot(ctx context.Context, sn *store.Snapshot) error {
	return nil
}

func (m *mockStore) GetSnapshot(ctx context.Context, id string) (*store.Snapshot, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetSnapshotByName(ctx context.Context, sandboxID, name string) (*store.Snapshot, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListSnapshots(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Snapshot, error) {
	return nil, nil
}

func (m *mockStore) SaveCommand(ctx context.Context, cmd *store.Command) error {
	return nil
}

func (m *mockStore) GetCommand(ctx context.Context, id string) (*store.Command, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListCommands(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error) {
	if m.listCommandsFn != nil {
		return m.listCommandsFn(ctx, sandboxID, opt)
	}
	return nil, nil
}

func (m *mockStore) SaveDiff(ctx context.Context, d *store.Diff) error {
	return nil
}

func (m *mockStore) GetDiff(ctx context.Context, id string) (*store.Diff, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetDiffBySnapshots(ctx context.Context, sandboxID, fromSnapshot, toSnapshot string) (*store.Diff, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) CreateChangeSet(ctx context.Context, cs *store.ChangeSet) error {
	return nil
}

func (m *mockStore) GetChangeSet(ctx context.Context, id string) (*store.ChangeSet, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetChangeSetByJob(ctx context.Context, jobID string) (*store.ChangeSet, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) CreatePublication(ctx context.Context, p *store.Publication) error {
	return nil
}

func (m *mockStore) UpdatePublicationStatus(ctx context.Context, id string, status store.PublicationStatus, commitSHA, prURL, errMsg *string) error {
	return nil
}

func (m *mockStore) GetPublication(ctx context.Context, id string) (*store.Publication, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) CreatePlaybook(ctx context.Context, pb *store.Playbook) error {
	return nil
}

func (m *mockStore) GetPlaybook(ctx context.Context, id string) (*store.Playbook, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) GetPlaybookByName(ctx context.Context, name string) (*store.Playbook, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListPlaybooks(ctx context.Context, opt *store.ListOptions) ([]*store.Playbook, error) {
	return nil, nil
}

func (m *mockStore) UpdatePlaybook(ctx context.Context, pb *store.Playbook) error {
	return nil
}

func (m *mockStore) DeletePlaybook(ctx context.Context, id string) error {
	return nil
}

func (m *mockStore) CreatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	return nil
}

func (m *mockStore) GetPlaybookTask(ctx context.Context, id string) (*store.PlaybookTask, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) ListPlaybookTasks(ctx context.Context, playbookID string, opt *store.ListOptions) ([]*store.PlaybookTask, error) {
	return nil, nil
}

func (m *mockStore) UpdatePlaybookTask(ctx context.Context, task *store.PlaybookTask) error {
	return nil
}

func (m *mockStore) DeletePlaybookTask(ctx context.Context, id string) error {
	return nil
}

func (m *mockStore) ReorderPlaybookTasks(ctx context.Context, playbookID string, taskIDs []string) error {
	return nil
}

func (m *mockStore) GetNextTaskPosition(ctx context.Context, playbookID string) (int, error) {
	return 0, nil
}

func (m *mockStore) GetSourceVM(ctx context.Context, name string) (*store.SourceVM, error) {
	return nil, store.ErrNotFound
}

func (m *mockStore) UpsertSourceVM(ctx context.Context, svm *store.SourceVM) error {
	return nil
}

func (m *mockStore) ListSourceVMs(ctx context.Context) ([]*store.SourceVM, error) {
	return nil, nil
}

func TestGetSandbox_Success(t *testing.T) {
	ip := "192.168.1.100"
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:          id,
				JobID:       "JOB-123",
				AgentID:     "agent-456",
				SandboxName: "test-sandbox",
				State:       store.SandboxStateRunning,
				IPAddress:   &ip,
			}, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
	}

	sb, err := svc.GetSandbox(context.Background(), "SBX-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sb == nil {
		t.Fatal("expected sandbox, got nil")
	}
	if sb.ID != "SBX-123" {
		t.Errorf("expected ID %q, got %q", "SBX-123", sb.ID)
	}
	if sb.State != store.SandboxStateRunning {
		t.Errorf("expected state %s, got %s", store.SandboxStateRunning, sb.State)
	}
}

func TestGetSandbox_NotFound(t *testing.T) {
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
	}

	_, err := svc.GetSandbox(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSandbox_EmptyID(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     &mockStore{},
		timeNowFn: time.Now,
	}

	_, err := svc.GetSandbox(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestGetSandbox_WhitespaceID(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     &mockStore{},
		timeNowFn: time.Now,
	}

	_, err := svc.GetSandbox(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected error for whitespace ID, got nil")
	}
}

func TestGetSandboxCommands_Success(t *testing.T) {
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id}, nil
		},
		listCommandsFn: func(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error) {
			return []*store.Command{
				{
					ID:        "CMD-001",
					SandboxID: sandboxID,
					Command:   "ls -la",
					Stdout:    "total 0\n",
					ExitCode:  0,
				},
				{
					ID:        "CMD-002",
					SandboxID: sandboxID,
					Command:   "pwd",
					Stdout:    "/home/user\n",
					ExitCode:  0,
				},
			}, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
	}

	cmds, err := svc.GetSandboxCommands(context.Background(), "SBX-123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cmds) != 2 {
		t.Errorf("expected 2 commands, got %d", len(cmds))
	}
	if cmds[0].Command != "ls -la" {
		t.Errorf("expected command %q, got %q", "ls -la", cmds[0].Command)
	}
}

func TestGetSandboxCommands_SandboxNotFound(t *testing.T) {
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
	}

	_, err := svc.GetSandboxCommands(context.Background(), "nonexistent-id", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetSandboxCommands_EmptyID(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     &mockStore{},
		timeNowFn: time.Now,
	}

	_, err := svc.GetSandboxCommands(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty ID, got nil")
	}
}

func TestGetSandboxCommands_WithListOptions(t *testing.T) {
	var capturedOpts *store.ListOptions
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{ID: id}, nil
		},
		listCommandsFn: func(ctx context.Context, sandboxID string, opt *store.ListOptions) ([]*store.Command, error) {
			capturedOpts = opt
			return []*store.Command{}, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
	}

	opts := &store.ListOptions{Limit: 10, Offset: 5}
	_, err := svc.GetSandboxCommands(context.Background(), "SBX-123", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOpts == nil {
		t.Fatal("expected list options to be passed")
	}
	if capturedOpts.Limit != 10 {
		t.Errorf("expected limit %d, got %d", 10, capturedOpts.Limit)
	}
	if capturedOpts.Offset != 5 {
		t.Errorf("expected offset %d, got %d", 5, capturedOpts.Offset)
	}
}

// mockSSHRunner is a mock implementation of SSHRunner for testing
type mockSSHRunner struct {
	runFn         func(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (stdout, stderr string, exitCode int, err error)
	runWithCertFn func(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (stdout, stderr string, exitCode int, err error)
}

func (m *mockSSHRunner) Run(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
	if m.runFn != nil {
		return m.runFn(ctx, addr, user, privateKeyPath, command, timeout, env, proxyJump)
	}
	return "", "", 0, nil
}

func (m *mockSSHRunner) RunWithCert(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
	if m.runWithCertFn != nil {
		return m.runWithCertFn(ctx, addr, user, privateKeyPath, certPath, command, timeout, env, proxyJump)
	}
	// Fall back to runFn if runWithCertFn is not set
	if m.runFn != nil {
		return m.runFn(ctx, addr, user, privateKeyPath, command, timeout, env, proxyJump)
	}
	return "", "", 0, nil
}

func (m *mockSSHRunner) RunStreaming(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string, outputChan chan<- OutputChunk) (string, string, int, error) {
	// For tests, just delegate to Run and ignore the output channel
	return m.Run(ctx, addr, user, privateKeyPath, command, timeout, env, proxyJump)
}

func (m *mockSSHRunner) RunWithCertStreaming(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string, outputChan chan<- OutputChunk) (string, string, int, error) {
	// For tests, just delegate to RunWithCert and ignore the output channel
	return m.RunWithCert(ctx, addr, user, privateKeyPath, certPath, command, timeout, env, proxyJump)
}

// mockManager is a mock implementation of libvirt.Manager for testing
type mockManager struct {
	getIPAddressFn func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error)
}

func (m *mockManager) CloneVM(ctx context.Context, baseImage, newVMName string, cpu, memoryMB int, network string) (libvirt.DomainRef, error) {
	return libvirt.DomainRef{}, nil
}

func (m *mockManager) CloneFromVM(ctx context.Context, sourceVMName, newVMName string, cpu, memoryMB int, network string) (libvirt.DomainRef, error) {
	return libvirt.DomainRef{}, nil
}

func (m *mockManager) InjectSSHKey(ctx context.Context, sandboxName, username, publicKey string) error {
	return nil
}

func (m *mockManager) StartVM(ctx context.Context, vmName string) error {
	return nil
}

func (m *mockManager) StopVM(ctx context.Context, vmName string, force bool) error {
	return nil
}

func (m *mockManager) DestroyVM(ctx context.Context, vmName string) error {
	return nil
}

func (m *mockManager) CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (libvirt.SnapshotRef, error) {
	return libvirt.SnapshotRef{}, nil
}

func (m *mockManager) DiffSnapshot(ctx context.Context, vmName, fromSnapshot, toSnapshot string) (*libvirt.FSComparePlan, error) {
	return nil, nil
}

func (m *mockManager) GetIPAddress(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
	if m.getIPAddressFn != nil {
		return m.getIPAddressFn(ctx, vmName, timeout)
	}
	return "192.168.1.100", "52:54:00:12:34:56", nil
}

func (m *mockManager) GetVMState(ctx context.Context, vmName string) (libvirt.VMState, error) {
	return libvirt.VMState("running"), nil
}

func (m *mockManager) ValidateSourceVM(ctx context.Context, vmName string) (*libvirt.VMValidationResult, error) {
	return &libvirt.VMValidationResult{
		VMName:   vmName,
		Valid:    true,
		State:    libvirt.VMStateRunning,
		Warnings: []string{},
		Errors:   []string{},
	}, nil
}

func (m *mockManager) CheckHostResources(ctx context.Context, requiredCPUs, requiredMemoryMB int) (*libvirt.ResourceCheckResult, error) {
	return &libvirt.ResourceCheckResult{
		Valid:             true,
		AvailableMemoryMB: 8192,
		TotalMemoryMB:     16384,
		AvailableCPUs:     8,
		AvailableDiskMB:   102400,
		RequiredMemoryMB:  requiredMemoryMB,
		RequiredCPUs:      requiredCPUs,
		Warnings:          []string{},
		Errors:            []string{},
	}, nil
}

func TestRunCommand_Success(t *testing.T) {
	ip := "192.168.1.100"
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:          id,
				SandboxName: "test-sandbox",
				State:       store.SandboxStateRunning,
				IPAddress:   &ip,
			}, nil
		},
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			// Return empty list - no other sandboxes with this IP
			return []*store.Sandbox{}, nil
		},
	}

	mockSSH := &mockSSHRunner{
		runFn: func(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
			return "file1.txt\nfile2.txt\n", "", 0, nil
		},
	}

	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.100", "52:54:00:12:34:56", nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		ssh:       mockSSH,
		mgr:       mockMgr,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	cmd, err := svc.RunCommand(context.Background(), "SBX-123", "ubuntu", "/path/to/key", "ls -l", 60*time.Second, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cmd == nil {
		t.Fatal("expected command result, got nil")
	}
	if cmd.Stdout != "file1.txt\nfile2.txt\n" {
		t.Errorf("expected stdout %q, got %q", "file1.txt\nfile2.txt\n", cmd.Stdout)
	}
	if cmd.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", cmd.ExitCode)
	}
}

func TestRunCommand_SSHConnectionFailed(t *testing.T) {
	ip := "192.168.1.100"
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:          id,
				SandboxName: "test-sandbox",
				State:       store.SandboxStateRunning,
				IPAddress:   &ip,
			}, nil
		},
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			return []*store.Sandbox{}, nil
		},
	}

	mockSSH := &mockSSHRunner{
		runFn: func(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
			return "", "ssh: connect to host 192.168.1.100 port 22: Connection refused", 255, errors.New("exit status 255: ssh: connect to host 192.168.1.100 port 22: Connection refused")
		},
	}

	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.100", "52:54:00:12:34:56", nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		ssh:       mockSSH,
		mgr:       mockMgr,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	cmd, err := svc.RunCommand(context.Background(), "SBX-123", "ubuntu", "/path/to/key", "ls -l", 60*time.Second, nil)

	// Should return error but also the command with stderr
	if err == nil {
		t.Fatal("expected error for SSH connection failure")
	}
	if cmd == nil {
		t.Fatal("expected command result with stderr even on SSH failure")
	}
	if cmd.ExitCode != 255 {
		t.Errorf("expected exit code 255, got %d", cmd.ExitCode)
	}
	if cmd.Stderr != "ssh: connect to host 192.168.1.100 port 22: Connection refused" {
		t.Errorf("expected stderr to contain SSH error, got %q", cmd.Stderr)
	}
}

func TestRunCommand_CommandFailed(t *testing.T) {
	ip := "192.168.1.100"
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:          id,
				SandboxName: "test-sandbox",
				State:       store.SandboxStateRunning,
				IPAddress:   &ip,
			}, nil
		},
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			return []*store.Sandbox{}, nil
		},
	}

	mockSSH := &mockSSHRunner{
		runFn: func(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
			// Command ran but returned non-zero exit code (not an SSH error)
			return "", "ls: cannot access '/nonexistent': No such file or directory", 2, nil
		},
	}

	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.100", "52:54:00:12:34:56", nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		ssh:       mockSSH,
		mgr:       mockMgr,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	cmd, err := svc.RunCommand(context.Background(), "SBX-123", "ubuntu", "/path/to/key", "ls /nonexistent", 60*time.Second, nil)
	if err != nil {
		t.Fatalf("unexpected error for command with non-zero exit: %v", err)
	}

	if cmd == nil {
		t.Fatal("expected command result, got nil")
	}
	if cmd.ExitCode != 2 {
		t.Errorf("expected exit code 2, got %d", cmd.ExitCode)
	}
	if cmd.Stderr == "" {
		t.Error("expected stderr to contain error message")
	}
}

func TestRunCommand_EmptySandboxID(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     &mockStore{},
		timeNowFn: time.Now,
	}

	_, err := svc.RunCommand(context.Background(), "", "ubuntu", "/path/to/key", "ls", 60*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for empty sandbox ID")
	}
}

func TestRunCommand_EmptyUsername(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     &mockStore{},
		timeNowFn: time.Now,
	}

	_, err := svc.RunCommand(context.Background(), "SBX-123", "", "/path/to/key", "ls", 60*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for empty username")
	}
}

func TestRunCommand_EmptyPrivateKeyPath(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     &mockStore{},
		timeNowFn: time.Now,
	}

	_, err := svc.RunCommand(context.Background(), "SBX-123", "ubuntu", "", "ls", 60*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for empty private key path")
	}
}

func TestRunCommand_EmptyCommand(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     &mockStore{},
		timeNowFn: time.Now,
	}

	_, err := svc.RunCommand(context.Background(), "SBX-123", "ubuntu", "/path/to/key", "", 60*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestRunCommand_SandboxNotFound(t *testing.T) {
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return nil, store.ErrNotFound
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
	}

	_, err := svc.RunCommand(context.Background(), "nonexistent", "ubuntu", "/path/to/key", "ls", 60*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for sandbox not found")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRunCommand_WithEnvironmentVariables(t *testing.T) {
	ip := "192.168.1.100"
	var capturedEnv map[string]string
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:          id,
				SandboxName: "test-sandbox",
				State:       store.SandboxStateRunning,
				IPAddress:   &ip,
			}, nil
		},
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			return []*store.Sandbox{}, nil
		},
	}

	mockSSH := &mockSSHRunner{
		runFn: func(ctx context.Context, addr, user, privateKeyPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
			capturedEnv = env
			return "test\n", "", 0, nil
		},
	}

	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.100", "52:54:00:12:34:56", nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		ssh:       mockSSH,
		mgr:       mockMgr,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	env := map[string]string{"MY_VAR": "test_value", "OTHER_VAR": "other"}
	_, err := svc.RunCommand(context.Background(), "SBX-123", "ubuntu", "/path/to/key", "echo $MY_VAR", 60*time.Second, env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedEnv == nil {
		t.Fatal("expected environment variables to be passed")
	}
	if capturedEnv["MY_VAR"] != "test_value" {
		t.Errorf("expected MY_VAR=%q, got %q", "test_value", capturedEnv["MY_VAR"])
	}
}

func TestRunCommand_IPConflictDetected(t *testing.T) {
	ip := "192.168.1.100"
	otherIP := "192.168.1.100" // Same IP as another sandbox
	mockSt := &mockStore{
		getSandboxFn: func(ctx context.Context, id string) (*store.Sandbox, error) {
			return &store.Sandbox{
				ID:          id,
				SandboxName: "test-sandbox",
				State:       store.SandboxStateRunning,
				IPAddress:   &ip,
			}, nil
		},
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			// Return another sandbox with the same IP - simulating a conflict
			return []*store.Sandbox{
				{
					ID:          "SBX-OTHER",
					SandboxName: "other-sandbox",
					State:       store.SandboxStateRunning,
					IPAddress:   &otherIP,
				},
			}, nil
		},
	}

	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.100", "52:54:00:12:34:56", nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		mgr:       mockMgr,
		timeNowFn: time.Now,
		logger:    slog.Default(),
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	_, err := svc.RunCommand(context.Background(), "SBX-123", "ubuntu", "/path/to/key", "ls -l", 60*time.Second, nil)
	if err == nil {
		t.Fatal("expected error for IP conflict, got nil")
	}
	if !strings.Contains(err.Error(), "ip conflict") {
		t.Errorf("expected error to contain 'ip conflict', got: %v", err)
	}
}

func TestValidateIPUniqueness_NoConflict(t *testing.T) {
	ip := "192.168.1.100"
	mockSt := &mockStore{
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			// Return a sandbox with a different IP
			otherIP := "192.168.1.200"
			return []*store.Sandbox{
				{
					ID:          "SBX-OTHER",
					SandboxName: "other-sandbox",
					State:       store.SandboxStateRunning,
					IPAddress:   &otherIP,
				},
			}, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
		logger:    slog.Default(),
	}

	err := svc.validateIPUniqueness(context.Background(), "SBX-123", ip)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIPUniqueness_Conflict(t *testing.T) {
	ip := "192.168.1.100"
	mockSt := &mockStore{
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			// Return another sandbox with the same IP
			return []*store.Sandbox{
				{
					ID:          "SBX-OTHER",
					SandboxName: "other-sandbox",
					State:       store.SandboxStateRunning,
					IPAddress:   &ip,
				},
			}, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
		logger:    slog.Default(),
	}

	err := svc.validateIPUniqueness(context.Background(), "SBX-123", ip)
	if err == nil {
		t.Fatal("expected error for IP conflict, got nil")
	}
	if !strings.Contains(err.Error(), "already assigned") {
		t.Errorf("expected error to contain 'already assigned', got: %v", err)
	}
}

func TestValidateIPUniqueness_SameSandboxIgnored(t *testing.T) {
	ip := "192.168.1.100"
	mockSt := &mockStore{
		listSandboxesFn: func(ctx context.Context, filter store.SandboxFilter, opt *store.ListOptions) ([]*store.Sandbox, error) {
			// Return the same sandbox - should be ignored
			return []*store.Sandbox{
				{
					ID:          "SBX-123", // Same ID as the one being validated
					SandboxName: "test-sandbox",
					State:       store.SandboxStateRunning,
					IPAddress:   &ip,
				},
			}, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		store:     mockSt,
		timeNowFn: time.Now,
		logger:    slog.Default(),
	}

	err := svc.validateIPUniqueness(context.Background(), "SBX-123", ip)
	if err != nil {
		t.Fatalf("unexpected error (same sandbox should be ignored): %v", err)
	}
}

// mockKeyManager is a mock implementation of sshkeys.KeyProvider for testing
type mockKeyManager struct {
	getSourceVMCredentialsFn func(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error)
}

func (m *mockKeyManager) GetCredentials(ctx context.Context, sandboxID, username string) (*sshkeys.Credentials, error) {
	return nil, nil
}

func (m *mockKeyManager) GetSourceVMCredentials(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error) {
	if m.getSourceVMCredentialsFn != nil {
		return m.getSourceVMCredentialsFn(ctx, sourceVMName)
	}
	return nil, nil
}

func (m *mockKeyManager) CleanupSandbox(ctx context.Context, sandboxID string) error {
	return nil
}

func (m *mockKeyManager) Close() error {
	return nil
}

func TestRunSourceVMCommand_AllowlistRejection(t *testing.T) {
	mockMgr := &mockManager{}
	mockKeyMgr := &mockKeyManager{}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		mgr:       mockMgr,
		keyMgr:    mockKeyMgr,
		timeNowFn: time.Now,
		logger:    slog.Default(),
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	// Test command not in the readonly allowlist (e.g., "rm" is not allowed)
	_, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "rm -rf /tmp/test", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for disallowed command, got nil")
	}
	if !strings.Contains(err.Error(), "command not allowed in read-only mode") {
		t.Errorf("expected 'command not allowed in read-only mode' error, got: %v", err)
	}
}

func TestRunSourceVMCommand_IPDiscoveryFailure(t *testing.T) {
	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "", "", errors.New("VM not found")
		},
	}
	mockKeyMgr := &mockKeyManager{}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		mgr:       mockMgr,
		keyMgr:    mockKeyMgr,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	// Use a command that would pass the allowlist check
	_, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "ls -l", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for IP discovery failure, got nil")
	}
	if !strings.Contains(err.Error(), "discover IP for source VM") {
		t.Errorf("expected 'discover IP for source VM' error, got: %v", err)
	}
}

func TestRunSourceVMCommand_MissingKeyManager(t *testing.T) {
	mockMgr := &mockManager{}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		mgr:       mockMgr,
		keyMgr:    nil, // No key manager set
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	// Use a command that would pass the allowlist check
	_, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "ls -l", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for missing key manager, got nil")
	}
	if !strings.Contains(err.Error(), "key manager is required for source VM access") {
		t.Errorf("expected 'key manager is required for source VM access' error, got: %v", err)
	}
}

func TestRunSourceVMCommand_Success(t *testing.T) {
	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.200", "52:54:00:12:34:57", nil
		},
	}
	mockKeyMgr := &mockKeyManager{
		getSourceVMCredentialsFn: func(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error) {
			return &sshkeys.Credentials{
				Username:        "fluid-readonly",
				PrivateKeyPath:  "/tmp/source-vm-key",
				CertificatePath: "/tmp/source-vm-key-cert.pub",
			}, nil
		},
	}
	mockSSH := &mockSSHRunner{
		runWithCertFn: func(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
			return "file1.txt\nfile2.txt\n", "", 0, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		mgr:       mockMgr,
		keyMgr:    mockKeyMgr,
		ssh:       mockSSH,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	result, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "ls -l", 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.SourceVM != "source-vm-1" {
		t.Errorf("expected SourceVM 'source-vm-1', got %q", result.SourceVM)
	}
	if result.Stdout != "file1.txt\nfile2.txt\n" {
		t.Errorf("expected stdout %q, got %q", "file1.txt\nfile2.txt\n", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRunSourceVMCommandWithCallback_StreamingSuccess(t *testing.T) {
	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.200", "52:54:00:12:34:57", nil
		},
	}
	mockKeyMgr := &mockKeyManager{
		getSourceVMCredentialsFn: func(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error) {
			return &sshkeys.Credentials{
				Username:        "fluid-readonly",
				PrivateKeyPath:  "/tmp/source-vm-key",
				CertificatePath: "/tmp/source-vm-key-cert.pub",
			}, nil
		},
	}

	mockSSH := &mockSSHRunner{
		runWithCertFn: func(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
			return "file1.txt\nfile2.txt\n", "", 0, nil
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		mgr:       mockMgr,
		keyMgr:    mockKeyMgr,
		ssh:       mockSSH,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	// Provide a simple callback - it won't be invoked with the current mock setup,
	// but this test verifies the code path compiles and executes without error.
	callback := func(chunk OutputChunk) {
		// Callback provided but not invoked by mock
	}

	result, err := svc.RunSourceVMCommandWithCallback(context.Background(), "source-vm-1", "ls -l", 60*time.Second, callback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.SourceVM != "source-vm-1" {
		t.Errorf("expected SourceVM 'source-vm-1', got %q", result.SourceVM)
	}
	// Note: The mock SSH runner delegates streaming to non-streaming, so callback won't be invoked in this test.
	// This test verifies the code path compiles and executes without error when a callback is provided.
}

func TestRunSourceVMCommand_EmptySourceVMName(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	_, err := svc.RunSourceVMCommand(context.Background(), "", "ls -l", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for empty sourceVMName, got nil")
	}
	if !strings.Contains(err.Error(), "sourceVMName is required") {
		t.Errorf("expected 'sourceVMName is required' error, got: %v", err)
	}
}

func TestRunSourceVMCommand_EmptyCommand(t *testing.T) {
	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	_, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for empty command, got nil")
	}
	if !strings.Contains(err.Error(), "command is required") {
		t.Errorf("expected 'command is required' error, got: %v", err)
	}
}

func TestRunSourceVMCommand_GetCredentialsFailure(t *testing.T) {
	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.200", "52:54:00:12:34:57", nil
		},
	}
	mockKeyMgr := &mockKeyManager{
		getSourceVMCredentialsFn: func(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error) {
			return nil, errors.New("credentials not found")
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		mgr:       mockMgr,
		keyMgr:    mockKeyMgr,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	_, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "ls -l", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for credentials failure, got nil")
	}
	if !strings.Contains(err.Error(), "get source VM credentials") {
		t.Errorf("expected 'get source VM credentials' error, got: %v", err)
	}
}

func TestRunSourceVMCommand_SSHRunFailure(t *testing.T) {
	mockMgr := &mockManager{
		getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
			return "192.168.1.200", "52:54:00:12:34:57", nil
		},
	}
	mockKeyMgr := &mockKeyManager{
		getSourceVMCredentialsFn: func(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error) {
			return &sshkeys.Credentials{
				Username:        "fluid-readonly",
				PrivateKeyPath:  "/tmp/source-vm-key",
				CertificatePath: "/tmp/source-vm-key-cert.pub",
			}, nil
		},
	}
	mockSSH := &mockSSHRunner{
		runWithCertFn: func(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
			return "", "connection timeout", 1, errors.New("SSH connection failed")
		},
	}

	svc := &Service{
		telemetry: telemetry.NewNoopService(),
		mgr:       mockMgr,
		keyMgr:    mockKeyMgr,
		ssh:       mockSSH,
		timeNowFn: time.Now,
		cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
	}

	result, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "ls -l", 60*time.Second)
	if err == nil {
		t.Fatal("expected error for SSH failure, got nil")
	}
	if !strings.Contains(err.Error(), "ssh run on source VM") {
		t.Errorf("expected 'ssh run on source VM' error, got: %v", err)
	}
	// Result should still be returned with exit code and stderr
	if result == nil {
		t.Fatal("expected result even with error, got nil")
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
	if result.Stderr != "connection timeout" {
		t.Errorf("expected stderr %q, got %q", "connection timeout", result.Stderr)
	}
}

// mockTelemetryService records Track calls for test assertions.
type mockTelemetryService struct {
	mu     sync.Mutex
	events []trackedEvent
}

type trackedEvent struct {
	name       string
	properties map[string]any
}

func (m *mockTelemetryService) Track(event string, properties map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, trackedEvent{name: event, properties: properties})
}

func (m *mockTelemetryService) Close() {}

func (m *mockTelemetryService) getEvents() []trackedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]trackedEvent, len(m.events))
	copy(out, m.events)
	return out
}

func TestRunSourceVMCommand_TelemetryTracking(t *testing.T) {
	t.Run("blocked by allowlist", func(t *testing.T) {
		mt := &mockTelemetryService{}
		svc := &Service{
			telemetry: mt,
			mgr:       &mockManager{},
			keyMgr:    &mockKeyManager{},
			timeNowFn: time.Now,
			logger:    slog.Default(),
			cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
		}

		_, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "rm -rf /", 60*time.Second)
		if err == nil {
			t.Fatal("expected error for disallowed command")
		}

		events := mt.getEvents()
		if len(events) != 1 {
			t.Fatalf("expected 1 telemetry event, got %d", len(events))
		}
		if events[0].name != "source_vm_command_blocked" {
			t.Errorf("expected event 'source_vm_command_blocked', got %q", events[0].name)
		}
		if events[0].properties["source_vm"] != "source-vm-1" {
			t.Errorf("expected source_vm 'source-vm-1', got %v", events[0].properties["source_vm"])
		}
		if _, ok := events[0].properties["reason"]; !ok {
			t.Error("expected 'reason' property in telemetry event")
		}
	})

	t.Run("successful command", func(t *testing.T) {
		mt := &mockTelemetryService{}
		svc := &Service{
			telemetry: mt,
			mgr: &mockManager{
				getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
					return "192.168.1.200", "52:54:00:12:34:57", nil
				},
			},
			keyMgr: &mockKeyManager{
				getSourceVMCredentialsFn: func(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error) {
					return &sshkeys.Credentials{
						Username:        "fluid-readonly",
						PrivateKeyPath:  "/tmp/key",
						CertificatePath: "/tmp/key-cert.pub",
					}, nil
				},
			},
			ssh: &mockSSHRunner{
				runWithCertFn: func(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
					return "output", "", 0, nil
				},
			},
			timeNowFn: time.Now,
			logger:    slog.Default(),
			cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
		}

		_, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "ls -l", 60*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		events := mt.getEvents()
		if len(events) != 1 {
			t.Fatalf("expected 1 telemetry event, got %d", len(events))
		}
		if events[0].name != "source_vm_command" {
			t.Errorf("expected event 'source_vm_command', got %q", events[0].name)
		}
		if events[0].properties["exit_code"] != 0 {
			t.Errorf("expected exit_code 0, got %v", events[0].properties["exit_code"])
		}
	})

	t.Run("blocked by restricted shell", func(t *testing.T) {
		mt := &mockTelemetryService{}
		svc := &Service{
			telemetry: mt,
			mgr: &mockManager{
				getIPAddressFn: func(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
					return "192.168.1.200", "52:54:00:12:34:57", nil
				},
			},
			keyMgr: &mockKeyManager{
				getSourceVMCredentialsFn: func(ctx context.Context, sourceVMName string) (*sshkeys.Credentials, error) {
					return &sshkeys.Credentials{
						Username:        "fluid-readonly",
						PrivateKeyPath:  "/tmp/key",
						CertificatePath: "/tmp/key-cert.pub",
					}, nil
				},
			},
			ssh: &mockSSHRunner{
				runWithCertFn: func(ctx context.Context, addr, user, privateKeyPath, certPath, command string, timeout time.Duration, env map[string]string, proxyJump string) (string, string, int, error) {
					return "", "command not permitted", 126, nil
				},
			},
			timeNowFn: time.Now,
			logger:    slog.Default(),
			cfg:       Config{CommandTimeout: 30 * time.Second, IPDiscoveryTimeout: 30 * time.Second},
		}

		result, err := svc.RunSourceVMCommand(context.Background(), "source-vm-1", "ls -l", 60*time.Second)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 126 {
			t.Errorf("expected exit code 126, got %d", result.ExitCode)
		}

		events := mt.getEvents()
		// Should have both: source_vm_command_blocked (restricted_shell) and source_vm_command
		if len(events) != 2 {
			t.Fatalf("expected 2 telemetry events, got %d", len(events))
		}
		if events[0].name != "source_vm_command_blocked" {
			t.Errorf("expected first event 'source_vm_command_blocked', got %q", events[0].name)
		}
		if events[0].properties["reason"] != "restricted_shell" {
			t.Errorf("expected reason 'restricted_shell', got %v", events[0].properties["reason"])
		}
		if events[1].name != "source_vm_command" {
			t.Errorf("expected second event 'source_vm_command', got %q", events[1].name)
		}
	})
}
