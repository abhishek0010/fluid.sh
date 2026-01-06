package vm

import (
	"context"
	"errors"
	"testing"
	"time"

	"virsh-sandbox/internal/store"
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
