package libvirt

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

const (
	// DefaultSSHUser is the default SSH user for remote hosts.
	DefaultSSHUser = "root"
	// DefaultSSHPort is the default SSH port.
	DefaultSSHPort = 22
	// DefaultHostQueryTimeout is the default per-host query timeout.
	DefaultHostQueryTimeout = 30 * time.Second
	// MaxShellInputLength is the maximum allowed length for shell input.
	MaxShellInputLength = 4096
)

// MultiHostDomainInfo extends DomainInfo with host identification.
type MultiHostDomainInfo struct {
	Name        string
	UUID        string
	State       DomainState
	Persistent  bool
	DiskPath    string
	HostName    string // Display name of the host
	HostAddress string // IP or hostname of the host
}

// HostError represents an error from querying a specific host.
type HostError struct {
	HostName    string `json:"host_name"`
	HostAddress string `json:"host_address"`
	Error       string `json:"error"`
}

// MultiHostListResult contains the aggregated result from querying all hosts.
type MultiHostListResult struct {
	Domains    []*MultiHostDomainInfo
	HostErrors []HostError
}

// SSHRunner executes commands on a remote host via SSH.
// This interface enables testing without actual SSH connections.
type SSHRunner interface {
	Run(ctx context.Context, address, user string, port int, command string) (string, error)
}

// defaultSSHRunner implements SSHRunner using actual SSH commands.
type defaultSSHRunner struct{}

func (r *defaultSSHRunner) Run(ctx context.Context, address, user string, port int, command string) (string, error) {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		"-p", fmt.Sprintf("%d", port),
		fmt.Sprintf("%s@%s", user, address),
		command,
	}

	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh command failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// MultiHostDomainManager queries multiple libvirt hosts via SSH.
type MultiHostDomainManager struct {
	hosts     []config.HostConfig
	logger    *slog.Logger
	sshRunner SSHRunner
}

// NewMultiHostDomainManager creates a new MultiHostDomainManager.
func NewMultiHostDomainManager(hosts []config.HostConfig, logger *slog.Logger) *MultiHostDomainManager {
	return &MultiHostDomainManager{
		hosts:     hosts,
		logger:    logger,
		sshRunner: &defaultSSHRunner{},
	}
}

// NewMultiHostDomainManagerWithRunner creates a MultiHostDomainManager with a custom SSH runner.
// This is primarily useful for testing.
func NewMultiHostDomainManagerWithRunner(hosts []config.HostConfig, logger *slog.Logger, runner SSHRunner) *MultiHostDomainManager {
	return &MultiHostDomainManager{
		hosts:     hosts,
		logger:    logger,
		sshRunner: runner,
	}
}

// ListDomains queries all configured hosts in parallel and aggregates VM listings.
// Returns all VMs found along with any host errors encountered.
func (m *MultiHostDomainManager) ListDomains(ctx context.Context) (*MultiHostListResult, error) {
	if len(m.hosts) == 0 {
		return &MultiHostListResult{}, nil
	}

	type hostResult struct {
		domains []*MultiHostDomainInfo
		err     *HostError
	}

	results := make(chan hostResult, len(m.hosts))
	var wg sync.WaitGroup

	for _, host := range m.hosts {
		wg.Add(1)
		go func(h config.HostConfig) {
			defer wg.Done()

			domains, err := m.queryHost(ctx, h)
			if err != nil {
				m.logger.Warn("failed to query host",
					"host_name", h.Name,
					"host_address", h.Address,
					"error", err,
				)
				results <- hostResult{
					err: &HostError{
						HostName:    h.Name,
						HostAddress: h.Address,
						Error:       err.Error(),
					},
				}
				return
			}
			results <- hostResult{domains: domains}
		}(host)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate results
	var allDomains []*MultiHostDomainInfo
	var hostErrors []HostError

	for result := range results {
		if result.err != nil {
			hostErrors = append(hostErrors, *result.err)
		} else {
			allDomains = append(allDomains, result.domains...)
		}
	}

	return &MultiHostListResult{
		Domains:    allDomains,
		HostErrors: hostErrors,
	}, nil
}

// queryHost queries a single host for its VM list via SSH.
func (m *MultiHostDomainManager) queryHost(ctx context.Context, host config.HostConfig) ([]*MultiHostDomainInfo, error) {
	// Apply defaults
	sshUser := host.SSHUser
	if sshUser == "" {
		sshUser = DefaultSSHUser
	}
	sshPort := host.SSHPort
	if sshPort == 0 {
		sshPort = DefaultSSHPort
	}
	queryTimeout := host.QueryTimeout
	if queryTimeout == 0 {
		queryTimeout = DefaultHostQueryTimeout
	}

	// Create context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Get list of VM names
	vmNames, err := m.runSSHCommand(queryCtx, host.Address, sshUser, sshPort,
		"virsh list --all --name")
	if err != nil {
		return nil, fmt.Errorf("list VMs: %w", err)
	}

	// Parse VM names (one per line, skip empty lines)
	var names []string
	scanner := bufio.NewScanner(strings.NewReader(vmNames))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			names = append(names, name)
		}
	}

	if len(names) == 0 {
		return nil, nil
	}

	// Get details for each VM
	var domains []*MultiHostDomainInfo
	for _, name := range names {
		domain, err := m.getDomainInfo(queryCtx, host, sshUser, sshPort, name)
		if err != nil {
			m.logger.Debug("failed to get domain info",
				"host", host.Name,
				"domain", name,
				"error", err,
			)
			// Continue with other VMs even if one fails
			continue
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// getDomainInfo gets detailed information for a single domain.
func (m *MultiHostDomainManager) getDomainInfo(ctx context.Context, host config.HostConfig, sshUser string, sshPort int, name string) (*MultiHostDomainInfo, error) {
	escapedName, err := shellEscape(name)
	if err != nil {
		return nil, fmt.Errorf("invalid domain name: %w", err)
	}

	// Get domain info using virsh dominfo
	output, err := m.runSSHCommand(ctx, host.Address, sshUser, sshPort,
		fmt.Sprintf("virsh dominfo %s", escapedName))
	if err != nil {
		return nil, fmt.Errorf("dominfo: %w", err)
	}

	domain := &MultiHostDomainInfo{
		Name:        name,
		HostName:    host.Name,
		HostAddress: host.Address,
	}

	// Parse dominfo output
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "UUID":
			domain.UUID = value
		case "State":
			domain.State = parseVirshState(value)
		case "Persistent":
			domain.Persistent = value == "yes"
		}
	}

	// Get disk path using virsh domblklist (reuse escapedName from above)
	diskOutput, err := m.runSSHCommand(ctx, host.Address, sshUser, sshPort,
		fmt.Sprintf("virsh domblklist %s --details", escapedName))
	if err == nil {
		domain.DiskPath = parseDiskPath(diskOutput)
	}

	return domain, nil
}

// runSSHCommand executes a command on a remote host via SSH.
func (m *MultiHostDomainManager) runSSHCommand(ctx context.Context, address, user string, port int, command string) (string, error) {
	return m.sshRunner.Run(ctx, address, user, port, command)
}

// parseVirshState converts virsh state string to DomainState.
func parseVirshState(state string) DomainState {
	switch strings.ToLower(state) {
	case "running":
		return DomainStateRunning
	case "paused":
		return DomainStatePaused
	case "shut off":
		return DomainStateStopped
	case "shutdown":
		return DomainStateShutdown
	case "crashed":
		return DomainStateCrashed
	case "pmsuspended":
		return DomainStateSuspended
	default:
		return DomainStateUnknown
	}
}

// parseDiskPath extracts the primary disk path from virsh domblklist output.
func parseDiskPath(output string) string {
	// Output format:
	// Type   Device   Target   Source
	// ------------------------------------------------
	// file   disk     vda      /var/lib/libvirt/images/vm.qcow2
	scanner := bufio.NewScanner(strings.NewReader(output))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		// Skip header lines
		if lineNum <= 2 {
			continue
		}
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[1] == "disk" {
			return fields[3]
		}
	}
	return ""
}

// ErrShellInputTooLong is returned when input exceeds MaxShellInputLength.
var ErrShellInputTooLong = errors.New("shell input exceeds maximum length")

// ErrShellInputNullByte is returned when input contains null bytes.
var ErrShellInputNullByte = errors.New("shell input contains null byte")

// ErrShellInputControlChar is returned when input contains control characters.
var ErrShellInputControlChar = errors.New("shell input contains control character")

// validateShellInput checks input for dangerous characters before shell escaping.
func validateShellInput(s string) error {
	if len(s) > MaxShellInputLength {
		return ErrShellInputTooLong
	}
	for _, r := range s {
		if r == 0 {
			return ErrShellInputNullByte
		}
		// Reject control characters (0x00-0x1F) except tab (0x09) and newline (0x0A)
		if unicode.IsControl(r) && r != '\t' && r != '\n' {
			return ErrShellInputControlChar
		}
	}
	return nil
}

// shellEscape escapes a string for safe use in shell commands.
// Returns an error if the input contains dangerous characters.
func shellEscape(s string) (string, error) {
	if err := validateShellInput(s); err != nil {
		return "", err
	}
	// Wrap in single quotes and escape existing single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'", nil
}

// FindHostForVM searches all configured hosts to find which one has the given VM.
// Returns the host config if found, or an error if the VM is not found on any host.
func (m *MultiHostDomainManager) FindHostForVM(ctx context.Context, vmName string) (*config.HostConfig, error) {
	if len(m.hosts) == 0 {
		return nil, fmt.Errorf("no hosts configured")
	}

	type findResult struct {
		host  *config.HostConfig
		found bool
		err   error
	}

	results := make(chan findResult, len(m.hosts))
	var wg sync.WaitGroup

	for i := range m.hosts {
		wg.Add(1)
		go func(h *config.HostConfig) {
			defer wg.Done()

			found, err := m.hostHasVM(ctx, *h, vmName)
			if err != nil {
				m.logger.Debug("error checking host for VM",
					"host", h.Name,
					"vm_name", vmName,
					"error", err,
				)
				results <- findResult{err: err}
				return
			}
			if found {
				results <- findResult{host: h, found: true}
			} else {
				results <- findResult{found: false}
			}
		}(&m.hosts[i])
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results - return first host that has the VM
	var lastErr error
	for result := range results {
		if result.found {
			return result.host, nil
		}
		if result.err != nil {
			lastErr = result.err
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("VM %q not found on any host (last error: %w)", vmName, lastErr)
	}
	return nil, fmt.Errorf("VM %q not found on any configured host", vmName)
}

// hostHasVM checks if a specific host has the given VM.
func (m *MultiHostDomainManager) hostHasVM(ctx context.Context, host config.HostConfig, vmName string) (bool, error) {
	escapedName, err := shellEscape(vmName)
	if err != nil {
		return false, fmt.Errorf("invalid VM name: %w", err)
	}

	sshUser := host.SSHUser
	if sshUser == "" {
		sshUser = DefaultSSHUser
	}
	sshPort := host.SSHPort
	if sshPort == 0 {
		sshPort = DefaultSSHPort
	}
	queryTimeout := host.QueryTimeout
	if queryTimeout == 0 {
		queryTimeout = DefaultHostQueryTimeout
	}

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Check if VM exists using virsh dominfo
	_, err = m.runSSHCommand(queryCtx, host.Address, sshUser, sshPort,
		fmt.Sprintf("virsh dominfo %s", escapedName))
	if err != nil {
		// If virsh dominfo fails, the VM doesn't exist on this host
		return false, nil
	}
	return true, nil
}

// GetHosts returns the configured hosts.
func (m *MultiHostDomainManager) GetHosts() []config.HostConfig {
	return m.hosts
}
