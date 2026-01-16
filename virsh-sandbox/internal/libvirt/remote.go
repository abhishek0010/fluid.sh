package libvirt

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"virsh-sandbox/internal/config"
)

// RemoteVirshManager implements Manager for remote libvirt hosts via SSH.
// It executes virsh and related commands on a remote host.
type RemoteVirshManager struct {
	host   config.HostConfig
	cfg    Config
	logger *slog.Logger
}

// NewRemoteVirshManager creates a new RemoteVirshManager for the given host.
func NewRemoteVirshManager(host config.HostConfig, cfg Config, logger *slog.Logger) *RemoteVirshManager {
	if cfg.DefaultVCPUs == 0 {
		cfg.DefaultVCPUs = 2
	}
	if cfg.DefaultMemoryMB == 0 {
		cfg.DefaultMemoryMB = 2048
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &RemoteVirshManager{
		host:   host,
		cfg:    cfg,
		logger: logger,
	}
}

// CloneVM creates a linked-clone VM on the remote host.
func (m *RemoteVirshManager) CloneVM(ctx context.Context, baseImage, newVMName string, cpu, memoryMB int, network string) (DomainRef, error) {
	return DomainRef{}, fmt.Errorf("CloneVM not implemented for remote hosts - use CloneFromVM instead")
}

// CloneFromVM creates a linked-clone VM from an existing VM on the remote host.
func (m *RemoteVirshManager) CloneFromVM(ctx context.Context, sourceVMName, newVMName string, cpu, memoryMB int, network string) (DomainRef, error) {
	if newVMName == "" {
		return DomainRef{}, fmt.Errorf("new VM name is required")
	}
	if sourceVMName == "" {
		return DomainRef{}, fmt.Errorf("source VM name is required")
	}
	if cpu <= 0 {
		cpu = m.cfg.DefaultVCPUs
	}
	if memoryMB <= 0 {
		memoryMB = m.cfg.DefaultMemoryMB
	}
	if network == "" {
		network = m.cfg.DefaultNetwork
	}

	m.logger.Info("cloning VM on remote host",
		"host", m.host.Name,
		"source_vm", sourceVMName,
		"new_vm", newVMName,
	)

	// Get source VM's disk path
	out, err := m.runSSH(ctx, fmt.Sprintf("virsh domblklist %s --details", shellEscape(sourceVMName)))
	if err != nil {
		return DomainRef{}, fmt.Errorf("lookup source VM %q: %w", sourceVMName, err)
	}

	basePath := ""
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == "file" && fields[1] == "disk" {
			basePath = fields[3]
			break
		}
	}
	if basePath == "" {
		return DomainRef{}, fmt.Errorf("could not find disk path for source VM %q", sourceVMName)
	}

	// Create job directory on remote host
	jobDir := fmt.Sprintf("%s/%s", m.cfg.WorkDir, newVMName)
	if _, err := m.runSSH(ctx, fmt.Sprintf("mkdir -p %s", shellEscape(jobDir))); err != nil {
		return DomainRef{}, fmt.Errorf("create job dir: %w", err)
	}

	// Create overlay disk
	overlayPath := fmt.Sprintf("%s/disk-overlay.qcow2", jobDir)
	if _, err := m.runSSH(ctx, fmt.Sprintf("qemu-img create -f qcow2 -F qcow2 -b %s %s",
		shellEscape(basePath), shellEscape(overlayPath))); err != nil {
		return DomainRef{}, fmt.Errorf("create overlay: %w", err)
	}

	// Dump source VM XML and modify it
	sourceXML, err := m.runSSH(ctx, fmt.Sprintf("virsh dumpxml %s", shellEscape(sourceVMName)))
	if err != nil {
		return DomainRef{}, fmt.Errorf("dumpxml source vm: %w", err)
	}

	newXML, err := modifyClonedXMLHelper(sourceXML, newVMName, overlayPath, cpu, memoryMB, network)
	if err != nil {
		return DomainRef{}, fmt.Errorf("modify cloned xml: %w", err)
	}

	// Write domain XML to remote host
	xmlPath := fmt.Sprintf("%s/domain.xml", jobDir)
	escapedXML := strings.ReplaceAll(newXML, "'", "'\\''")
	if _, err := m.runSSH(ctx, fmt.Sprintf("echo '%s' > %s", escapedXML, shellEscape(xmlPath))); err != nil {
		return DomainRef{}, fmt.Errorf("write domain xml: %w", err)
	}

	// Define the domain
	if _, err := m.runSSH(ctx, fmt.Sprintf("virsh define %s", shellEscape(xmlPath))); err != nil {
		return DomainRef{}, fmt.Errorf("virsh define: %w", err)
	}

	// Get UUID
	out, err = m.runSSH(ctx, fmt.Sprintf("virsh domuuid %s", shellEscape(newVMName)))
	if err != nil {
		return DomainRef{Name: newVMName}, nil
	}

	return DomainRef{Name: newVMName, UUID: strings.TrimSpace(out)}, nil
}

// InjectSSHKey injects an SSH public key on the remote host.
func (m *RemoteVirshManager) InjectSSHKey(ctx context.Context, sandboxName, username, publicKey string) error {
	if sandboxName == "" {
		return fmt.Errorf("sandboxName is required")
	}
	if username == "" {
		username = "sandbox"
	}
	if strings.TrimSpace(publicKey) == "" {
		return fmt.Errorf("publicKey is required")
	}

	jobDir := fmt.Sprintf("%s/%s", m.cfg.WorkDir, sandboxName)
	overlay := fmt.Sprintf("%s/disk-overlay.qcow2", jobDir)

	switch strings.ToLower(m.cfg.SSHKeyInjectMethod) {
	case "virt-customize":
		cmdArgs := fmt.Sprintf("virt-customize -a %s --run-command 'id -u %s >/dev/null 2>&1 || useradd -m -s /bin/bash %s' --ssh-inject '%s:string:%s'",
			shellEscape(overlay),
			shellEscape(username),
			shellEscape(username),
			shellEscape(username),
			shellEscape(publicKey),
		)
		if _, err := m.runSSH(ctx, cmdArgs); err != nil {
			return fmt.Errorf("virt-customize inject: %w", err)
		}
	default:
		return fmt.Errorf("unsupported SSHKeyInjectMethod for remote: %s", m.cfg.SSHKeyInjectMethod)
	}
	return nil
}

// StartVM starts a VM on the remote host.
func (m *RemoteVirshManager) StartVM(ctx context.Context, vmName string) error {
	if vmName == "" {
		return fmt.Errorf("vmName is required")
	}

	m.logger.Info("starting VM on remote host",
		"host", m.host.Name,
		"vm_name", vmName,
	)

	_, err := m.runSSH(ctx, fmt.Sprintf("virsh start %s", shellEscape(vmName)))
	if err != nil {
		return fmt.Errorf("virsh start: %w", err)
	}
	return nil
}

// StopVM stops a VM on the remote host.
func (m *RemoteVirshManager) StopVM(ctx context.Context, vmName string, force bool) error {
	if vmName == "" {
		return fmt.Errorf("vmName is required")
	}

	cmd := "shutdown"
	if force {
		cmd = "destroy"
	}

	_, err := m.runSSH(ctx, fmt.Sprintf("virsh %s %s", cmd, shellEscape(vmName)))
	return err
}

// DestroyVM destroys and undefines a VM on the remote host.
func (m *RemoteVirshManager) DestroyVM(ctx context.Context, vmName string) error {
	if vmName == "" {
		return fmt.Errorf("vmName is required")
	}

	// Best-effort destroy if running
	_, _ = m.runSSH(ctx, fmt.Sprintf("virsh destroy %s", shellEscape(vmName)))

	// Undefine
	if _, err := m.runSSH(ctx, fmt.Sprintf("virsh undefine %s", shellEscape(vmName))); err != nil {
		// Continue to remove files
		_ = err
	}

	// Remove workspace
	jobDir := fmt.Sprintf("%s/%s", m.cfg.WorkDir, vmName)
	_, _ = m.runSSH(ctx, fmt.Sprintf("rm -rf %s", shellEscape(jobDir)))

	return nil
}

// CreateSnapshot creates a snapshot on the remote host.
func (m *RemoteVirshManager) CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (SnapshotRef, error) {
	if vmName == "" || snapshotName == "" {
		return SnapshotRef{}, fmt.Errorf("vmName and snapshotName are required")
	}

	if external {
		jobDir := fmt.Sprintf("%s/%s", m.cfg.WorkDir, vmName)
		snapPath := fmt.Sprintf("%s/snap-%s.qcow2", jobDir, snapshotName)
		args := fmt.Sprintf("virsh snapshot-create-as %s %s --disk-only --atomic --no-metadata --diskspec vda,file=%s",
			shellEscape(vmName), shellEscape(snapshotName), shellEscape(snapPath))
		if _, err := m.runSSH(ctx, args); err != nil {
			return SnapshotRef{}, fmt.Errorf("external snapshot create: %w", err)
		}
		return SnapshotRef{Name: snapshotName, Kind: "EXTERNAL", Ref: snapPath}, nil
	}

	if _, err := m.runSSH(ctx, fmt.Sprintf("virsh snapshot-create-as %s %s",
		shellEscape(vmName), shellEscape(snapshotName))); err != nil {
		return SnapshotRef{}, fmt.Errorf("internal snapshot create: %w", err)
	}
	return SnapshotRef{Name: snapshotName, Kind: "INTERNAL", Ref: snapshotName}, nil
}

// DiffSnapshot returns a diff plan for the remote host.
func (m *RemoteVirshManager) DiffSnapshot(ctx context.Context, vmName, fromSnapshot, toSnapshot string) (*FSComparePlan, error) {
	if vmName == "" || fromSnapshot == "" || toSnapshot == "" {
		return nil, fmt.Errorf("vmName, fromSnapshot and toSnapshot are required")
	}

	plan := &FSComparePlan{
		VMName:       vmName,
		FromSnapshot: fromSnapshot,
		ToSnapshot:   toSnapshot,
		Notes:        []string{"Remote host snapshot diffing - manual intervention required"},
	}
	return plan, nil
}

// GetIPAddress discovers the IP address of a VM on the remote host.
func (m *RemoteVirshManager) GetIPAddress(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
	if vmName == "" {
		return "", "", fmt.Errorf("vmName is required")
	}

	m.logger.Info("discovering IP on remote host",
		"host", m.host.Name,
		"vm_name", vmName,
		"timeout", timeout,
	)

	deadline := time.Now().Add(timeout)
	attempt := 0

	for {
		attempt++
		out, err := m.runSSH(ctx, fmt.Sprintf("virsh domifaddr %s --source lease", shellEscape(vmName)))
		if err == nil {
			ip, mac := parseDomIfAddrIPv4WithMACHelper(out)
			if ip != "" {
				m.logger.Info("IP discovered on remote host",
					"host", m.host.Name,
					"vm_name", vmName,
					"ip", ip,
					"mac", mac,
				)
				return ip, mac, nil
			}
		}

		if time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Second)
	}

	return "", "", fmt.Errorf("ip address not found within timeout on remote host %s", m.host.Name)
}

// GetVMState returns the state of a VM on the remote host.
func (m *RemoteVirshManager) GetVMState(ctx context.Context, vmName string) (VMState, error) {
	if vmName == "" {
		return VMStateUnknown, fmt.Errorf("vmName is required")
	}

	out, err := m.runSSH(ctx, fmt.Sprintf("virsh domstate %s", shellEscape(vmName)))
	if err != nil {
		return VMStateUnknown, fmt.Errorf("get vm state: %w", err)
	}
	return parseVMStateHelper(out), nil
}

// runSSH executes a command on the remote host via SSH.
func (m *RemoteVirshManager) runSSH(ctx context.Context, command string) (string, error) {
	sshUser := m.host.SSHUser
	if sshUser == "" {
		sshUser = "root"
	}
	sshPort := m.host.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	args := []string{
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		"-p", fmt.Sprintf("%d", sshPort),
		fmt.Sprintf("%s@%s", sshUser, m.host.Address),
		command,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return stdout.String(), fmt.Errorf("%w: %s", err, errStr)
		}
		return stdout.String(), err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// HostConfig returns the host configuration for this manager.
func (m *RemoteVirshManager) HostConfig() config.HostConfig {
	return m.host
}
