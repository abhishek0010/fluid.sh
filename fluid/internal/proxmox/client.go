package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a pure Go HTTP client for the Proxmox VE API.
// Authentication uses API tokens (no session/CSRF needed).
type Client struct {
	baseURL    string
	tokenID    string
	secret     string
	node       string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient creates a new Proxmox API client.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !cfg.VerifySSL,
		},
	}
	return &Client{
		baseURL: strings.TrimRight(cfg.Host, "/"),
		tokenID: cfg.TokenID,
		secret:  cfg.Secret,
		node:    cfg.Node,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		logger: logger,
	}
}

// do executes an HTTP request against the Proxmox API.
func (c *Client) do(ctx context.Context, method, path string, body url.Values) (json.RawMessage, error) {
	apiURL := fmt.Sprintf("%s/api2/json%s", c.baseURL, path)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(body.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.tokenID, c.secret))
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	// Parse the outer response envelope
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors json.RawMessage `json:"errors,omitempty"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return envelope.Data, nil
}

// ListVMs returns all QEMU VMs on the configured node.
func (c *Client) ListVMs(ctx context.Context) ([]VMListEntry, error) {
	path := fmt.Sprintf("/nodes/%s/qemu", c.node)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var vms []VMListEntry
	if err := json.Unmarshal(data, &vms); err != nil {
		return nil, fmt.Errorf("unmarshal VM list: %w", err)
	}
	return vms, nil
}

// GetVMStatus returns the status of a VM by VMID.
func (c *Client) GetVMStatus(ctx context.Context, vmid int) (*VMStatus, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/current", c.node, vmid)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status VMStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal VM status: %w", err)
	}
	return &status, nil
}

// GetVMConfig returns the configuration of a VM by VMID.
func (c *Client) GetVMConfig(ctx context.Context, vmid int) (*VMConfig, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config", c.node, vmid)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var cfg VMConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal VM config: %w", err)
	}
	return &cfg, nil
}

// CloneVM clones a VM. Returns the UPID of the clone task.
func (c *Client) CloneVM(ctx context.Context, sourceVMID, newVMID int, name string, full bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/clone", c.node, sourceVMID)
	params := url.Values{
		"newid": {fmt.Sprintf("%d", newVMID)},
		"name":  {name},
	}
	if full {
		params.Set("full", "1")
	}

	data, err := c.do(ctx, http.MethodPost, path, params)
	if err != nil {
		return "", err
	}

	// Response is a UPID string
	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// SetVMConfig updates VM configuration parameters.
func (c *Client) SetVMConfig(ctx context.Context, vmid int, params url.Values) error {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config", c.node, vmid)
	_, err := c.do(ctx, http.MethodPut, path, params)
	return err
}

// StartVM starts a VM. Returns the UPID of the start task.
func (c *Client) StartVM(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/start", c.node, vmid)
	data, err := c.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// StopVM stops a VM (hard stop). Returns the UPID.
func (c *Client) StopVM(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", c.node, vmid)
	data, err := c.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// ShutdownVM gracefully shuts down a VM. Returns the UPID.
func (c *Client) ShutdownVM(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/shutdown", c.node, vmid)
	data, err := c.do(ctx, http.MethodPost, path, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// DeleteVM deletes a VM and all its resources. Returns the UPID.
func (c *Client) DeleteVM(ctx context.Context, vmid int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d", c.node, vmid)
	params := url.Values{
		"purge":                      {"1"},
		"destroy-unreferenced-disks": {"1"},
	}
	data, err := c.do(ctx, http.MethodDelete, path+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		return "", fmt.Errorf("unmarshal UPID: %w", err)
	}
	return upid, nil
}

// CreateSnapshot creates a snapshot of a VM. Returns the UPID (or nil for sync).
func (c *Client) CreateSnapshot(ctx context.Context, vmid int, name, description string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", c.node, vmid)
	params := url.Values{
		"snapname": {name},
	}
	if description != "" {
		params.Set("description", description)
	}

	data, err := c.do(ctx, http.MethodPost, path, params)
	if err != nil {
		return "", err
	}

	var upid string
	if err := json.Unmarshal(data, &upid); err != nil {
		// Snapshot may return null data on sync completion
		return "", nil
	}
	return upid, nil
}

// GetGuestAgentInterfaces returns network interfaces via the QEMU guest agent.
func (c *Client) GetGuestAgentInterfaces(ctx context.Context, vmid int) ([]NetworkInterface, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", c.node, vmid)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	// Proxmox wraps the result in a "result" field
	var result struct {
		Result []NetworkInterface `json:"result"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		// Try direct unmarshal
		var ifaces []NetworkInterface
		if err2 := json.Unmarshal(data, &ifaces); err2 != nil {
			return nil, fmt.Errorf("unmarshal interfaces: %w", err)
		}
		return ifaces, nil
	}
	return result.Result, nil
}

// GetNodeStatus returns the resource status of the configured node.
func (c *Client) GetNodeStatus(ctx context.Context) (*NodeStatus, error) {
	path := fmt.Sprintf("/nodes/%s/status", c.node)
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status NodeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal node status: %w", err)
	}
	return &status, nil
}

// GetTaskStatus returns the status of a task by UPID.
func (c *Client) GetTaskStatus(ctx context.Context, upid string) (*TaskStatus, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", c.node, url.PathEscape(upid))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("unmarshal task status: %w", err)
	}
	return &status, nil
}

// WaitForTask polls a task until it completes or the context is cancelled.
// Returns an error if the task fails.
func (c *Client) WaitForTask(ctx context.Context, upid string) error {
	if upid == "" {
		return nil
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, err := c.GetTaskStatus(ctx, upid)
			if err != nil {
				return fmt.Errorf("check task status: %w", err)
			}
			if status.Status == "stopped" {
				if status.ExitStatus != "OK" {
					return fmt.Errorf("task failed with status: %s", status.ExitStatus)
				}
				return nil
			}
		}
	}
}

// NextVMID finds the next available VMID in the configured range.
func (c *Client) NextVMID(ctx context.Context, start, end int) (int, error) {
	vms, err := c.ListVMs(ctx)
	if err != nil {
		return 0, fmt.Errorf("list VMs for VMID allocation: %w", err)
	}

	used := make(map[int]bool, len(vms))
	for _, vm := range vms {
		used[vm.VMID] = true
	}

	for id := start; id <= end; id++ {
		if !used[id] {
			return id, nil
		}
	}
	return 0, fmt.Errorf("no available VMID in range %d-%d", start, end)
}

// ResizeVM changes the VM's CPU and memory configuration.
func (c *Client) ResizeVM(ctx context.Context, vmid, cores, memoryMB int) error {
	params := url.Values{
		"cores":  {fmt.Sprintf("%d", cores)},
		"memory": {fmt.Sprintf("%d", memoryMB)},
	}
	return c.SetVMConfig(ctx, vmid, params)
}
