// Package controlplane provides a REST client for the fluid control plane.
// All sandbox and source VM operations route through the control plane API
// instead of directly accessing libvirt or SSH.
package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a REST client for the fluid control plane API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a control plane client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// ---------------------------------------------------------------------------
// Response types - match control-plane/internal/api/ types
// ---------------------------------------------------------------------------

// SandboxResponse is the response from sandbox operations.
type SandboxResponse struct {
	ID         string     `json:"id"`
	HostID     string     `json:"host_id"`
	Name       string     `json:"name"`
	AgentID    string     `json:"agent_id"`
	BaseImage  string     `json:"base_image"`
	Bridge     string     `json:"bridge"`
	TAPDevice  string     `json:"tap_device"`
	MACAddress string     `json:"mac_address"`
	IPAddress  string     `json:"ip_address"`
	State      string     `json:"state"`
	VCPUs      int32      `json:"vcpus"`
	MemoryMB   int32      `json:"memory_mb"`
	TTLSeconds int32      `json:"ttl_seconds"`
	SourceVM   string     `json:"source_vm"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// CommandResponse is the response from a command execution.
type CommandResponse struct {
	ID         string    `json:"id"`
	SandboxID  string    `json:"sandbox_id"`
	Command    string    `json:"command"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	ExitCode   int32     `json:"exit_code"`
	DurationMS int64     `json:"duration_ms"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at"`
}

// VMInfo describes a source VM.
type VMInfo struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	IPAddress string `json:"ip_address,omitempty"`
	Prepared  bool   `json:"prepared"`
	HostID    string `json:"host_id,omitempty"`
}

// HostInfo describes a connected host.
type HostInfo struct {
	HostID          string   `json:"host_id"`
	Hostname        string   `json:"hostname"`
	Status          string   `json:"status"`
	ActiveSandboxes int      `json:"active_sandboxes"`
	AvailableCPUs   int32    `json:"available_cpus"`
	AvailableMemMB  int64    `json:"available_memory_mb"`
	AvailableDiskMB int64    `json:"available_disk_mb"`
	BaseImages      []string `json:"base_images"`
	LastHeartbeat   string   `json:"last_heartbeat"`
}

// SourceCommandResult is the result of a source VM command.
type SourceCommandResult struct {
	SourceVM string `json:"source_vm"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// SourceFileResult is the result of reading a source VM file.
type SourceFileResult struct {
	SourceVM string `json:"source_vm"`
	Path     string `json:"path"`
	Content  string `json:"content"`
}

// SnapshotResponse is returned after creating a snapshot.
type SnapshotResponse struct {
	SnapshotID   string    `json:"snapshot_id"`
	SandboxID    string    `json:"sandbox_id"`
	SnapshotName string    `json:"snapshot_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

// CreateSandboxRequest creates a sandbox.
type CreateSandboxRequest struct {
	AgentID    string `json:"agent_id"`
	SourceVM   string `json:"source_vm"`
	BaseImage  string `json:"base_image"`
	Name       string `json:"name"`
	VCPUs      int    `json:"vcpus,omitempty"`
	MemoryMB   int    `json:"memory_mb,omitempty"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
	Network    string `json:"network,omitempty"`
}

// RunCommandRequest runs a command in a sandbox.
type RunCommandRequest struct {
	Command    string            `json:"command"`
	TimeoutSec int               `json:"timeout_seconds,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// ---------------------------------------------------------------------------
// Sandbox operations
// ---------------------------------------------------------------------------

// CreateSandbox creates a new sandbox via the control plane.
func (c *Client) CreateSandbox(ctx context.Context, req CreateSandboxRequest) (*SandboxResponse, error) {
	var result SandboxResponse
	if err := c.post(ctx, "/v1/sandboxes", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSandbox retrieves a sandbox by ID.
func (c *Client) GetSandbox(ctx context.Context, id string) (*SandboxResponse, error) {
	var result SandboxResponse
	if err := c.get(ctx, fmt.Sprintf("/v1/sandboxes/%s", id), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListSandboxes returns all sandboxes.
func (c *Client) ListSandboxes(ctx context.Context) ([]*SandboxResponse, error) {
	var result struct {
		Sandboxes []*SandboxResponse `json:"sandboxes"`
		Count     int                `json:"count"`
	}
	if err := c.get(ctx, "/v1/sandboxes", &result); err != nil {
		return nil, err
	}
	return result.Sandboxes, nil
}

// DestroySandbox destroys a sandbox by ID.
func (c *Client) DestroySandbox(ctx context.Context, id string) error {
	return c.delete(ctx, fmt.Sprintf("/v1/sandboxes/%s", id))
}

// StartSandbox starts a stopped sandbox.
func (c *Client) StartSandbox(ctx context.Context, id string) error {
	return c.post(ctx, fmt.Sprintf("/v1/sandboxes/%s/start", id), nil, nil)
}

// StopSandbox stops a running sandbox.
func (c *Client) StopSandbox(ctx context.Context, id string) error {
	return c.post(ctx, fmt.Sprintf("/v1/sandboxes/%s/stop", id), nil, nil)
}

// RunCommand executes a command in a sandbox.
func (c *Client) RunCommand(ctx context.Context, sandboxID, command string, timeoutSec int) (*CommandResponse, error) {
	req := RunCommandRequest{
		Command:    command,
		TimeoutSec: timeoutSec,
	}
	var result CommandResponse
	if err := c.post(ctx, fmt.Sprintf("/v1/sandboxes/%s/run", sandboxID), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSandboxIP returns the IP address of a sandbox.
func (c *Client) GetSandboxIP(ctx context.Context, id string) (string, error) {
	var result struct {
		SandboxID string `json:"sandbox_id"`
		IPAddress string `json:"ip_address"`
	}
	if err := c.get(ctx, fmt.Sprintf("/v1/sandboxes/%s/ip", id), &result); err != nil {
		return "", err
	}
	return result.IPAddress, nil
}

// CreateSnapshot creates a snapshot of a sandbox.
func (c *Client) CreateSnapshot(ctx context.Context, sandboxID, name string) (*SnapshotResponse, error) {
	req := struct {
		Name string `json:"name"`
	}{Name: name}
	var result SnapshotResponse
	if err := c.post(ctx, fmt.Sprintf("/v1/sandboxes/%s/snapshot", sandboxID), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListCommands returns commands for a sandbox.
func (c *Client) ListCommands(ctx context.Context, sandboxID string) ([]*CommandResponse, error) {
	var result struct {
		Commands []*CommandResponse `json:"commands"`
		Count    int                `json:"count"`
	}
	if err := c.get(ctx, fmt.Sprintf("/v1/sandboxes/%s/commands", sandboxID), &result); err != nil {
		return nil, err
	}
	return result.Commands, nil
}

// ---------------------------------------------------------------------------
// Host operations
// ---------------------------------------------------------------------------

// ListHosts returns all connected hosts.
func (c *Client) ListHosts(ctx context.Context) ([]*HostInfo, error) {
	var result struct {
		Hosts []*HostInfo `json:"hosts"`
		Count int         `json:"count"`
	}
	if err := c.get(ctx, "/v1/hosts", &result); err != nil {
		return nil, err
	}
	return result.Hosts, nil
}

// GetHost returns a specific host.
func (c *Client) GetHost(ctx context.Context, id string) (*HostInfo, error) {
	var result HostInfo
	if err := c.get(ctx, fmt.Sprintf("/v1/hosts/%s", id), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// Source VM operations
// ---------------------------------------------------------------------------

// ListVMs returns all source VMs across all hosts.
func (c *Client) ListVMs(ctx context.Context) ([]*VMInfo, error) {
	var result struct {
		VMs   []*VMInfo `json:"vms"`
		Count int       `json:"count"`
	}
	if err := c.get(ctx, "/v1/vms", &result); err != nil {
		return nil, err
	}
	return result.VMs, nil
}

// PrepareSourceVM prepares a source VM for read-only access.
func (c *Client) PrepareSourceVM(ctx context.Context, vmName, sshUser, keyPath string) (any, error) {
	req := struct {
		SSHUser    string `json:"ssh_user"`
		SSHKeyPath string `json:"ssh_key_path"`
	}{SSHUser: sshUser, SSHKeyPath: keyPath}
	var result any
	if err := c.post(ctx, fmt.Sprintf("/v1/sources/%s/prepare", vmName), req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ValidateSourceVM validates a source VM's readiness.
func (c *Client) ValidateSourceVM(ctx context.Context, vmName string) (any, error) {
	var result any
	if err := c.post(ctx, fmt.Sprintf("/v1/sources/%s/validate", vmName), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RunSourceCommand runs a read-only command on a source VM.
func (c *Client) RunSourceCommand(ctx context.Context, vmName, command string, timeoutSec int) (*SourceCommandResult, error) {
	req := struct {
		Command    string `json:"command"`
		TimeoutSec int    `json:"timeout_seconds,omitempty"`
	}{Command: command, TimeoutSec: timeoutSec}
	var result SourceCommandResult
	if err := c.post(ctx, fmt.Sprintf("/v1/sources/%s/run", vmName), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReadSourceFile reads a file from a source VM.
func (c *Client) ReadSourceFile(ctx context.Context, vmName, path string) (*SourceFileResult, error) {
	req := struct {
		Path string `json:"path"`
	}{Path: path}
	var result SourceFileResult
	if err := c.post(ctx, fmt.Sprintf("/v1/sources/%s/read", vmName), req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

func (c *Client) get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, result)
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.doRequest(req, result)
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, nil)
}

func (c *Client) doRequest(req *http.Request, result any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(data, &errResp) == nil && errResp.Error != "" {
			return fmt.Errorf("control plane error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return fmt.Errorf("control plane error (%d): %s", resp.StatusCode, string(data))
	}

	if result != nil && len(data) > 0 {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
