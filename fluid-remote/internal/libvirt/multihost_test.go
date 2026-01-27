package libvirt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-remote/internal/config"
)

// mockSSHRunner implements SSHRunner for testing.
type mockSSHRunner struct {
	mu           sync.Mutex
	responses    map[string]string // command -> response
	errors       map[string]error  // command -> error
	defaultError error
	delay        time.Duration
	callCount    atomic.Int64
	callLog      []mockSSHCall
}

type mockSSHCall struct {
	Address string
	User    string
	Port    int
	Command string
}

func newMockSSHRunner() *mockSSHRunner {
	return &mockSSHRunner{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockSSHRunner) Run(ctx context.Context, address, user string, port int, command string) (string, error) {
	m.callCount.Add(1)

	m.mu.Lock()
	m.callLog = append(m.callLog, mockSSHCall{
		Address: address,
		User:    user,
		Port:    port,
		Command: command,
	})
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	// Check for address-specific errors first
	if err, ok := m.errors[address]; ok {
		return "", err
	}

	// Check for command-specific responses
	if resp, ok := m.responses[command]; ok {
		return resp, nil
	}

	if m.defaultError != nil {
		return "", m.defaultError
	}

	return "", nil
}

func (m *mockSSHRunner) setResponse(command, response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[command] = response
}

func (m *mockSSHRunner) setHostError(address string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[address] = err
}

func (m *mockSSHRunner) getCalls() []mockSSHCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]mockSSHCall, len(m.callLog))
	copy(result, m.callLog)
	return result
}

func TestParseVirshState(t *testing.T) {
	tests := []struct {
		input    string
		expected DomainState
	}{
		{"running", DomainStateRunning},
		{"Running", DomainStateRunning},
		{"RUNNING", DomainStateRunning},
		{"paused", DomainStatePaused},
		{"shut off", DomainStateStopped},
		{"shutdown", DomainStateShutdown},
		{"crashed", DomainStateCrashed},
		{"pmsuspended", DomainStateSuspended},
		{"unknown", DomainStateUnknown},
		{"", DomainStateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseVirshState(tt.input)
			if result != tt.expected {
				t.Errorf("parseVirshState(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDiskPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "standard output",
			input: `Type   Device   Target   Source
------------------------------------------------
file   disk     vda      /var/lib/libvirt/images/test.qcow2
file   cdrom    sda      -`,
			expected: "/var/lib/libvirt/images/test.qcow2",
		},
		{
			name: "multiple disks",
			input: `Type   Device   Target   Source
------------------------------------------------
file   disk     vda      /var/lib/libvirt/images/root.qcow2
file   disk     vdb      /var/lib/libvirt/images/data.qcow2`,
			expected: "/var/lib/libvirt/images/root.qcow2",
		},
		{
			name:     "empty output",
			input:    "",
			expected: "",
		},
		{
			name: "no disks",
			input: `Type   Device   Target   Source
------------------------------------------------`,
			expected: "",
		},
		{
			name: "cdrom only",
			input: `Type   Device   Target   Source
------------------------------------------------
file   cdrom    sda      /path/to/iso.iso`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDiskPath(tt.input)
			if result != tt.expected {
				t.Errorf("parseDiskPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"simple", "'simple'", false},
		{"with spaces", "'with spaces'", false},
		{"with'quote", "'with'\"'\"'quote'", false},
		{"", "''", false},
		{"test-vm-01", "'test-vm-01'", false},
		{"with\ttab", "'with\ttab'", false},
		{"with\nnewline", "'with\nnewline'", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := shellEscape(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("shellEscape(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestShellEscapeValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "null byte",
			input:   "test\x00value",
			wantErr: ErrShellInputNullByte,
		},
		{
			name:    "control character bell",
			input:   "test\x07value",
			wantErr: ErrShellInputControlChar,
		},
		{
			name:    "control character escape",
			input:   "test\x1bvalue",
			wantErr: ErrShellInputControlChar,
		},
		{
			name:    "control character carriage return",
			input:   "test\rvalue",
			wantErr: ErrShellInputControlChar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := shellEscape(tt.input)
			if err == nil {
				t.Errorf("shellEscape(%q) expected error, got nil", tt.input)
				return
			}
			if err != tt.wantErr {
				t.Errorf("shellEscape(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestShellEscapeMaxLength(t *testing.T) {
	// Test input at max length (should succeed)
	atMax := make([]byte, MaxShellInputLength)
	for i := range atMax {
		atMax[i] = 'a'
	}
	_, err := shellEscape(string(atMax))
	if err != nil {
		t.Errorf("shellEscape at max length should succeed, got error: %v", err)
	}

	// Test input over max length (should fail)
	overMax := make([]byte, MaxShellInputLength+1)
	for i := range overMax {
		overMax[i] = 'a'
	}
	_, err = shellEscape(string(overMax))
	if err != ErrShellInputTooLong {
		t.Errorf("shellEscape over max length should return ErrShellInputTooLong, got: %v", err)
	}
}

func TestValidateShellInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"valid simple", "simple", nil},
		{"valid with spaces", "with spaces", nil},
		{"valid with tab", "with\ttab", nil},
		{"valid with newline", "with\nnewline", nil},
		{"invalid null byte", "test\x00value", ErrShellInputNullByte},
		{"invalid bell", "test\x07value", ErrShellInputControlChar},
		{"invalid escape", "test\x1bvalue", ErrShellInputControlChar},
		{"invalid backspace", "test\x08value", ErrShellInputControlChar},
		{"invalid form feed", "test\x0cvalue", ErrShellInputControlChar},
		{"invalid carriage return", "test\rvalue", ErrShellInputControlChar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateShellInput(tt.input)
			if err != tt.wantErr {
				t.Errorf("validateShellInput(%q) = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNewMultiHostDomainManager(t *testing.T) {
	manager := NewMultiHostDomainManager(nil, nil)
	if manager == nil {
		t.Fatal("NewMultiHostDomainManager returned nil")
	}
	if manager.hosts != nil {
		t.Error("Expected nil hosts slice")
	}
}

func TestGetHosts(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
	}
	manager := NewMultiHostDomainManager(hosts, nil)

	result := manager.GetHosts()
	if len(result) != 2 {
		t.Errorf("Expected 2 hosts, got %d", len(result))
	}
	if result[0].Name != "host1" {
		t.Errorf("Expected first host name to be 'host1', got %s", result[0].Name)
	}
}

// TestListDomainsAllHostsUnreachable tests the case when all configured hosts fail.
func TestListDomainsAllHostsUnreachable(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
		{Name: "host3", Address: "192.168.1.3"},
	}

	mock := newMockSSHRunner()
	mock.defaultError = errors.New("connection refused")

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	result, err := manager.ListDomains(ctx)
	// ListDomains returns an error aggregation, not a top-level error
	if err != nil {
		t.Fatalf("ListDomains should not return top-level error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// All hosts should have failed
	if len(result.HostErrors) != 3 {
		t.Errorf("Expected 3 host errors, got %d", len(result.HostErrors))
	}

	// No domains should be returned
	if len(result.Domains) != 0 {
		t.Errorf("Expected 0 domains, got %d", len(result.Domains))
	}

	// Verify each host error is recorded
	hostErrorMap := make(map[string]bool)
	for _, he := range result.HostErrors {
		hostErrorMap[he.HostName] = true
		if he.Error == "" {
			t.Errorf("Host error for %s should have error message", he.HostName)
		}
	}

	for _, h := range hosts {
		if !hostErrorMap[h.Name] {
			t.Errorf("Expected error for host %s", h.Name)
		}
	}
}

// TestListDomainsPartialHostFailure tests when some hosts succeed and some fail.
func TestListDomainsPartialHostFailure(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
	}

	mock := newMockSSHRunner()
	// host1 fails
	mock.setHostError("192.168.1.1", errors.New("connection refused"))
	// host2 succeeds with VMs
	mock.setResponse("virsh list --all --name", "vm1\nvm2\n")
	mock.setResponse("virsh dominfo 'vm1'", "UUID: 1234\nState: running\nPersistent: yes\n")
	mock.setResponse("virsh dominfo 'vm2'", "UUID: 5678\nState: shut off\nPersistent: yes\n")
	mock.setResponse("virsh domblklist 'vm1' --details", "Type   Device   Target   Source\n------------------------------------------------\nfile   disk     vda      /var/lib/libvirt/images/vm1.qcow2\n")
	mock.setResponse("virsh domblklist 'vm2' --details", "Type   Device   Target   Source\n------------------------------------------------\nfile   disk     vda      /var/lib/libvirt/images/vm2.qcow2\n")

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	result, err := manager.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains should not return top-level error: %v", err)
	}

	// One host should have failed
	if len(result.HostErrors) != 1 {
		t.Errorf("Expected 1 host error, got %d", len(result.HostErrors))
	}

	if result.HostErrors[0].HostName != "host1" {
		t.Errorf("Expected host1 to fail, got %s", result.HostErrors[0].HostName)
	}

	// VMs from host2 should be returned
	if len(result.Domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(result.Domains))
	}
}

// TestSSHConnectionTimeout tests that SSH timeouts are handled correctly.
func TestSSHConnectionTimeout(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "slow-host", Address: "192.168.1.100"},
	}

	mock := newMockSSHRunner()
	mock.delay = 5 * time.Second // Simulate slow response

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	// Use a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := manager.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains should not return top-level error: %v", err)
	}

	// The host should have timed out
	if len(result.HostErrors) != 1 {
		t.Errorf("Expected 1 host error due to timeout, got %d", len(result.HostErrors))
	}

	if len(result.HostErrors) > 0 && result.HostErrors[0].HostName != "slow-host" {
		t.Errorf("Expected slow-host to fail, got %s", result.HostErrors[0].HostName)
	}
}

// TestFindHostForVMAllHostsUnreachable tests FindHostForVM when all hosts fail.
func TestFindHostForVMAllHostsUnreachable(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
	}

	mock := newMockSSHRunner()
	mock.defaultError = errors.New("connection timed out")

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	_, err := manager.FindHostForVM(ctx, "test-vm")

	if err == nil {
		t.Fatal("FindHostForVM should return error when all hosts are unreachable")
	}

	// Error should mention the VM name
	if !errors.Is(err, nil) && err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestFindHostForVMNotFoundOnAnyHost tests when VM doesn't exist on any host.
func TestFindHostForVMNotFoundOnAnyHost(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
	}

	mock := newMockSSHRunner()
	// All hosts respond but VM not found (dominfo fails)
	mock.defaultError = errors.New("domain not found")

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	_, err := manager.FindHostForVM(ctx, "nonexistent-vm")

	if err == nil {
		t.Fatal("FindHostForVM should return error when VM not found")
	}
}

// TestConcurrentVMOperationsOnSameHost tests thread safety during concurrent queries.
func TestConcurrentVMOperationsOnSameHost(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
	}

	mock := newMockSSHRunner()
	mock.setResponse("virsh list --all --name", "vm1\nvm2\nvm3\n")
	mock.setResponse("virsh dominfo 'vm1'", "UUID: 1111\nState: running\nPersistent: yes\n")
	mock.setResponse("virsh dominfo 'vm2'", "UUID: 2222\nState: running\nPersistent: yes\n")
	mock.setResponse("virsh dominfo 'vm3'", "UUID: 3333\nState: running\nPersistent: yes\n")
	mock.setResponse("virsh domblklist 'vm1' --details", "Type   Device   Target   Source\n------------------------------------------------\nfile   disk     vda      /var/lib/libvirt/images/vm1.qcow2\n")
	mock.setResponse("virsh domblklist 'vm2' --details", "Type   Device   Target   Source\n------------------------------------------------\nfile   disk     vda      /var/lib/libvirt/images/vm2.qcow2\n")
	mock.setResponse("virsh domblklist 'vm3' --details", "Type   Device   Target   Source\n------------------------------------------------\nfile   disk     vda      /var/lib/libvirt/images/vm3.qcow2\n")
	mock.delay = 10 * time.Millisecond // Small delay to create overlap

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	// Run multiple concurrent ListDomains operations
	const concurrency = 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)
	results := make(chan *MultiHostListResult, concurrency)

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := manager.ListDomains(ctx)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}()
	}

	wg.Wait()
	close(errors)
	close(results)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify all results are consistent
	var resultCount int
	for result := range results {
		resultCount++
		if len(result.Domains) != 3 {
			t.Errorf("Expected 3 domains, got %d", len(result.Domains))
		}
	}

	if resultCount != concurrency {
		t.Errorf("Expected %d results, got %d", concurrency, resultCount)
	}
}

// TestConcurrentFindHostForVM tests concurrent FindHostForVM operations.
func TestConcurrentFindHostForVM(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
		{Name: "host2", Address: "192.168.1.2"},
		{Name: "host3", Address: "192.168.1.3"},
	}

	mock := newMockSSHRunner()
	// VM exists on host2 only
	mock.setHostError("192.168.1.1", errors.New("domain not found"))
	mock.setHostError("192.168.1.3", errors.New("domain not found"))
	mock.setResponse("virsh dominfo 'target-vm'", "UUID: abc123\nState: running\n")

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	// Run multiple concurrent FindHostForVM operations
	const concurrency = 5
	var wg sync.WaitGroup
	foundHosts := make(chan *config.HostConfig, concurrency)
	foundErrors := make(chan error, concurrency)

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			host, err := manager.FindHostForVM(ctx, "target-vm")
			if err != nil {
				foundErrors <- err
				return
			}
			foundHosts <- host
		}()
	}

	wg.Wait()
	close(foundHosts)
	close(foundErrors)

	// All should find host2
	for host := range foundHosts {
		if host.Name != "host2" {
			t.Errorf("Expected host2, got %s", host.Name)
		}
	}

	// Should have no errors
	for err := range foundErrors {
		t.Errorf("Unexpected error: %v", err)
	}
}

// TestListDomainsEmptyHosts tests behavior with no hosts configured.
func TestListDomainsEmptyHosts(t *testing.T) {
	manager := NewMultiHostDomainManager(nil, nil)

	ctx := context.Background()
	result, err := manager.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains with empty hosts should not error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(result.Domains) != 0 {
		t.Errorf("Expected 0 domains, got %d", len(result.Domains))
	}

	if len(result.HostErrors) != 0 {
		t.Errorf("Expected 0 host errors, got %d", len(result.HostErrors))
	}
}

// TestFindHostForVMNoHostsConfigured tests FindHostForVM with no hosts.
func TestFindHostForVMNoHostsConfigured(t *testing.T) {
	manager := NewMultiHostDomainManager(nil, nil)

	ctx := context.Background()
	_, err := manager.FindHostForVM(ctx, "any-vm")

	if err == nil {
		t.Fatal("Expected error when no hosts configured")
	}

	expectedMsg := "no hosts configured"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

// TestSSHDefaultsApplied tests that SSH defaults are correctly applied.
func TestSSHDefaultsApplied(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"}, // No SSHUser or SSHPort set
	}

	mock := newMockSSHRunner()
	mock.setResponse("virsh list --all --name", "")

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	_, _ = manager.ListDomains(ctx)

	calls := mock.getCalls()
	if len(calls) == 0 {
		t.Fatal("Expected at least one SSH call")
	}

	// Verify defaults were applied
	if calls[0].User != DefaultSSHUser {
		t.Errorf("Expected default SSH user %q, got %q", DefaultSSHUser, calls[0].User)
	}

	if calls[0].Port != DefaultSSHPort {
		t.Errorf("Expected default SSH port %d, got %d", DefaultSSHPort, calls[0].Port)
	}
}

// TestSSHCustomPortAndUser tests that custom SSH settings are used.
func TestSSHCustomPortAndUser(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1", SSHUser: "admin", SSHPort: 2222},
	}

	mock := newMockSSHRunner()
	mock.setResponse("virsh list --all --name", "")

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	_, _ = manager.ListDomains(ctx)

	calls := mock.getCalls()
	if len(calls) == 0 {
		t.Fatal("Expected at least one SSH call")
	}

	if calls[0].User != "admin" {
		t.Errorf("Expected SSH user 'admin', got %q", calls[0].User)
	}

	if calls[0].Port != 2222 {
		t.Errorf("Expected SSH port 2222, got %d", calls[0].Port)
	}
}

// TestListDomainsWithDomainInfoFailure tests graceful handling when dominfo fails for one VM.
func TestListDomainsWithDomainInfoFailure(t *testing.T) {
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"},
	}

	mock := &selectiveMockSSHRunner{
		responses: map[string]mockResponse{
			"virsh list --all --name":          {output: "vm1\nvm2\nvm3\n"},
			"virsh dominfo 'vm1'":              {output: "UUID: 1111\nState: running\nPersistent: yes\n"},
			"virsh dominfo 'vm2'":              {err: errors.New("domain info failed")},
			"virsh dominfo 'vm3'":              {output: "UUID: 3333\nState: running\nPersistent: yes\n"},
			"virsh domblklist 'vm1' --details": {output: "Type   Device   Target   Source\n------------------------------------------------\nfile   disk     vda      /var/lib/libvirt/images/vm1.qcow2\n"},
			"virsh domblklist 'vm3' --details": {output: "Type   Device   Target   Source\n------------------------------------------------\nfile   disk     vda      /var/lib/libvirt/images/vm3.qcow2\n"},
		},
	}

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	result, err := manager.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains should not return top-level error: %v", err)
	}

	// Should get 2 VMs (vm1 and vm3), vm2 failed
	if len(result.Domains) != 2 {
		t.Errorf("Expected 2 domains (vm2 should be skipped), got %d", len(result.Domains))
	}

	// No host errors since the host itself is reachable
	if len(result.HostErrors) != 0 {
		t.Errorf("Expected 0 host errors, got %d", len(result.HostErrors))
	}
}

// TestCustomQueryTimeout tests that per-host QueryTimeout is respected.
func TestCustomQueryTimeout(t *testing.T) {
	// Host with short custom timeout
	hosts := []config.HostConfig{
		{Name: "fast-host", Address: "192.168.1.1", QueryTimeout: 50 * time.Millisecond},
	}

	mock := newMockSSHRunner()
	mock.delay = 200 * time.Millisecond // Exceeds the custom timeout

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	result, err := manager.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains should not return top-level error: %v", err)
	}

	// The host should have timed out due to custom timeout
	if len(result.HostErrors) != 1 {
		t.Errorf("Expected 1 host error due to custom timeout, got %d", len(result.HostErrors))
	}
}

// TestDefaultQueryTimeoutUsedWhenNotSet tests that default timeout is used when QueryTimeout is 0.
func TestDefaultQueryTimeoutUsedWhenNotSet(t *testing.T) {
	// Host without custom timeout (uses default)
	hosts := []config.HostConfig{
		{Name: "host1", Address: "192.168.1.1"}, // QueryTimeout = 0, should use default
	}

	mock := newMockSSHRunner()
	mock.setResponse("virsh list --all --name", "")
	// No delay - should complete within default timeout

	logger := slog.Default()
	manager := NewMultiHostDomainManagerWithRunner(hosts, logger, mock)

	ctx := context.Background()
	result, err := manager.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains should not return error: %v", err)
	}

	// Should succeed with no errors
	if len(result.HostErrors) != 0 {
		t.Errorf("Expected 0 host errors, got %d", len(result.HostErrors))
	}
}

// selectiveMockSSHRunner allows command-specific responses.
type selectiveMockSSHRunner struct {
	mu        sync.Mutex
	responses map[string]mockResponse
}

type mockResponse struct {
	output string
	err    error
}

func (m *selectiveMockSSHRunner) Run(ctx context.Context, address, user string, port int, command string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if resp, ok := m.responses[command]; ok {
		return resp.output, resp.err
	}
	return "", fmt.Errorf("no mock response for command: %s", command)
}
