package libvirt

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/aspectrr/fluid.sh/fluid-remote/internal/config"
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

	// Validate inputs for shell escaping
	escapedSourceVM, err := shellEscape(sourceVMName)
	if err != nil {
		return DomainRef{}, fmt.Errorf("invalid source VM name: %w", err)
	}
	escapedNewVM, err := shellEscape(newVMName)
	if err != nil {
		return DomainRef{}, fmt.Errorf("invalid new VM name: %w", err)
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
	out, err := m.runSSH(ctx, fmt.Sprintf("virsh domblklist %s --details", escapedSourceVM))
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

	// Validate and escape paths
	jobDir := fmt.Sprintf("%s/%s", m.cfg.WorkDir, newVMName)
	escapedJobDir, err := shellEscape(jobDir)
	if err != nil {
		return DomainRef{}, fmt.Errorf("invalid job directory path: %w", err)
	}

	// Create job directory on remote host
	if _, err := m.runSSH(ctx, fmt.Sprintf("mkdir -p %s", escapedJobDir)); err != nil {
		return DomainRef{}, fmt.Errorf("create job dir: %w", err)
	}

	// Create overlay disk
	overlayPath := fmt.Sprintf("%s/disk-overlay.qcow2", jobDir)
	escapedBasePath, err := shellEscape(basePath)
	if err != nil {
		return DomainRef{}, fmt.Errorf("invalid base path: %w", err)
	}
	escapedOverlayPath, err := shellEscape(overlayPath)
	if err != nil {
		return DomainRef{}, fmt.Errorf("invalid overlay path: %w", err)
	}
	if _, err := m.runSSH(ctx, fmt.Sprintf("qemu-img create -f qcow2 -F qcow2 -b %s %s",
		escapedBasePath, escapedOverlayPath)); err != nil {
		return DomainRef{}, fmt.Errorf("create overlay: %w", err)
	}

	// Generate a unique cloud-init ISO for the cloned VM on the remote host
	// This ensures the clone gets a new instance-id and DHCP network config
	cloudInitISO := fmt.Sprintf("%s/cloud-init.iso", jobDir)
	if err := m.buildCloudInitSeedOnRemote(ctx, newVMName, jobDir, cloudInitISO); err != nil {
		// Log warning but don't fail - VM might still work if source didn't use cloud-init
		m.logger.Warn("failed to build cloud-init seed for clone, continuing without it",
			"vm", newVMName,
			"error", err,
		)
		cloudInitISO = "" // Don't try to attach a non-existent ISO
	}

	// Dump source VM XML and modify it
	sourceXML, err := m.runSSH(ctx, fmt.Sprintf("virsh dumpxml %s", escapedSourceVM))
	if err != nil {
		return DomainRef{}, fmt.Errorf("dumpxml source vm: %w", err)
	}

	newXML, err := modifyClonedXMLHelper(sourceXML, newVMName, overlayPath, cloudInitISO, cpu, memoryMB, network)
	if err != nil {
		return DomainRef{}, fmt.Errorf("modify cloned xml: %w", err)
	}

	// Write domain XML to remote host using base64 to avoid shell escaping issues
	xmlPath := fmt.Sprintf("%s/domain.xml", jobDir)
	escapedXMLPath, err := shellEscape(xmlPath)
	if err != nil {
		return DomainRef{}, fmt.Errorf("invalid XML path: %w", err)
	}
	encodedXML := base64.StdEncoding.EncodeToString([]byte(newXML))
	if _, err := m.runSSH(ctx, fmt.Sprintf("echo %s | base64 -d > %s", encodedXML, escapedXMLPath)); err != nil {
		return DomainRef{}, fmt.Errorf("write domain xml: %w", err)
	}

	// Define the domain
	if _, err := m.runSSH(ctx, fmt.Sprintf("virsh define %s", escapedXMLPath)); err != nil {
		return DomainRef{}, fmt.Errorf("virsh define: %w", err)
	}

	// Get UUID
	out, err = m.runSSH(ctx, fmt.Sprintf("virsh domuuid %s", escapedNewVM))
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

	// Validate inputs for shell escaping
	escapedOverlay, err := shellEscape(overlay)
	if err != nil {
		return fmt.Errorf("invalid overlay path: %w", err)
	}
	escapedUsername, err := shellEscape(username)
	if err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}
	escapedPublicKey, err := shellEscape(publicKey)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	switch strings.ToLower(m.cfg.SSHKeyInjectMethod) {
	case "virt-customize":
		cmdArgs := fmt.Sprintf("virt-customize -a %s --run-command 'id -u %s >/dev/null 2>&1 || useradd -m -s /bin/bash %s' --ssh-inject '%s:string:%s'",
			escapedOverlay,
			escapedUsername,
			escapedUsername,
			escapedUsername,
			escapedPublicKey,
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

	escapedName, err := shellEscape(vmName)
	if err != nil {
		return fmt.Errorf("invalid VM name: %w", err)
	}

	m.logger.Info("starting VM on remote host",
		"host", m.host.Name,
		"vm_name", vmName,
	)

	_, err = m.runSSH(ctx, fmt.Sprintf("virsh start %s", escapedName))
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

	escapedName, err := shellEscape(vmName)
	if err != nil {
		return fmt.Errorf("invalid VM name: %w", err)
	}

	cmd := "shutdown"
	if force {
		cmd = "destroy"
	}

	_, err = m.runSSH(ctx, fmt.Sprintf("virsh %s %s", cmd, escapedName))
	return err
}

// DestroyVM destroys and undefines a VM on the remote host.
func (m *RemoteVirshManager) DestroyVM(ctx context.Context, vmName string) error {
	if vmName == "" {
		return fmt.Errorf("vmName is required")
	}

	escapedName, err := shellEscape(vmName)
	if err != nil {
		return fmt.Errorf("invalid VM name: %w", err)
	}

	// Best-effort destroy if running
	_, _ = m.runSSH(ctx, fmt.Sprintf("virsh destroy %s", escapedName))

	// Undefine
	if _, err := m.runSSH(ctx, fmt.Sprintf("virsh undefine %s", escapedName)); err != nil {
		// Continue to remove files
		_ = err
	}

	// Remove workspace
	jobDir := fmt.Sprintf("%s/%s", m.cfg.WorkDir, vmName)
	escapedJobDir, err := shellEscape(jobDir)
	if err != nil {
		return fmt.Errorf("invalid job directory path: %w", err)
	}
	_, _ = m.runSSH(ctx, fmt.Sprintf("rm -rf %s", escapedJobDir))

	return nil
}

// CreateSnapshot creates a snapshot on the remote host.
func (m *RemoteVirshManager) CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (SnapshotRef, error) {
	if vmName == "" || snapshotName == "" {
		return SnapshotRef{}, fmt.Errorf("vmName and snapshotName are required")
	}

	escapedVMName, err := shellEscape(vmName)
	if err != nil {
		return SnapshotRef{}, fmt.Errorf("invalid VM name: %w", err)
	}
	escapedSnapshotName, err := shellEscape(snapshotName)
	if err != nil {
		return SnapshotRef{}, fmt.Errorf("invalid snapshot name: %w", err)
	}

	if external {
		jobDir := fmt.Sprintf("%s/%s", m.cfg.WorkDir, vmName)
		snapPath := fmt.Sprintf("%s/snap-%s.qcow2", jobDir, snapshotName)
		escapedSnapPath, err := shellEscape(snapPath)
		if err != nil {
			return SnapshotRef{}, fmt.Errorf("invalid snapshot path: %w", err)
		}
		args := fmt.Sprintf("virsh snapshot-create-as %s %s --disk-only --atomic --no-metadata --diskspec vda,file=%s",
			escapedVMName, escapedSnapshotName, escapedSnapPath)
		if _, err := m.runSSH(ctx, args); err != nil {
			return SnapshotRef{}, fmt.Errorf("external snapshot create: %w", err)
		}
		return SnapshotRef{Name: snapshotName, Kind: "EXTERNAL", Ref: snapPath}, nil
	}

	if _, err := m.runSSH(ctx, fmt.Sprintf("virsh snapshot-create-as %s %s",
		escapedVMName, escapedSnapshotName)); err != nil {
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

	escapedName, err := shellEscape(vmName)
	if err != nil {
		return "", "", fmt.Errorf("invalid VM name: %w", err)
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
		out, err := m.runSSH(ctx, fmt.Sprintf("virsh domifaddr %s --source lease", escapedName))
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

	escapedName, err := shellEscape(vmName)
	if err != nil {
		return VMStateUnknown, fmt.Errorf("invalid VM name: %w", err)
	}

	out, err := m.runSSH(ctx, fmt.Sprintf("virsh domstate %s", escapedName))
	if err != nil {
		return VMStateUnknown, fmt.Errorf("get vm state: %w", err)
	}
	return parseVMStateHelper(out), nil
}

// ValidateSourceVM performs pre-flight checks on a source VM on the remote host.
func (m *RemoteVirshManager) ValidateSourceVM(ctx context.Context, vmName string) (*VMValidationResult, error) {
	if vmName == "" {
		return nil, fmt.Errorf("vmName is required")
	}

	escapedName, err := shellEscape(vmName)
	if err != nil {
		return nil, fmt.Errorf("invalid VM name: %w", err)
	}

	result := &VMValidationResult{
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
	}

	// Check VM state
	state, err := m.GetVMState(ctx, vmName)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get VM state: %v", err))
		return result, nil
	}
	result.State = state

	// Check MAC address using domiflist
	out, err := m.runSSH(ctx, fmt.Sprintf("virsh domiflist %s", escapedName))
	if err != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Could not get network interfaces: %v", err))
	} else {
		lines := strings.Split(out, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Interface") || strings.HasPrefix(line, "-") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				mac := fields[4]
				if strings.Count(mac, ":") == 5 {
					result.MACAddress = mac
					result.HasNetwork = true
					break
				}
			}
		}
		if result.MACAddress == "" {
			result.Warnings = append(result.Warnings,
				"Could not find MAC address - source VM may not have a network interface")
		}
	}

	// Check IP address if running
	switch state {
	case VMStateRunning:
		out, err = m.runSSH(ctx, fmt.Sprintf("virsh domifaddr %s --source lease", escapedName))
		if err == nil {
			ip, mac := parseDomIfAddrIPv4WithMACHelper(out)
			if ip != "" {
				result.IPAddress = ip
				if mac != "" && result.MACAddress == "" {
					result.MACAddress = mac
					result.HasNetwork = true
				}
			} else {
				result.Warnings = append(result.Warnings,
					"Source VM is running but has no IP address assigned")
				result.Warnings = append(result.Warnings,
					"This may indicate cloud-init or DHCP issues - cloned sandboxes may also fail to get IPs")
			}
		}
	case VMStateShutOff:
		result.Warnings = append(result.Warnings,
			"Source VM is shut off - cannot verify network configuration (IP/DHCP)")
	}

	return result, nil
}

// CheckHostResources validates that the remote host has sufficient resources.
func (m *RemoteVirshManager) CheckHostResources(ctx context.Context, requiredCPUs, requiredMemoryMB int) (*ResourceCheckResult, error) {
	result := &ResourceCheckResult{
		Valid:            true,
		RequiredCPUs:     requiredCPUs,
		RequiredMemoryMB: requiredMemoryMB,
		Warnings:         []string{},
		Errors:           []string{},
	}

	// Check CPUs using virsh nodeinfo
	out, err := m.runSSH(ctx, "virsh nodeinfo")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "CPU(s):") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					_, _ = fmt.Sscanf(fields[1], "%d", &result.AvailableCPUs)
				}
			}
		}
		if requiredCPUs > result.AvailableCPUs {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Insufficient CPUs on %s: need %d but only %d available",
					m.host.Name, requiredCPUs, result.AvailableCPUs))
		}
	} else {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Could not check CPUs on %s: %v", m.host.Name, err))
	}

	// Check memory using virsh nodememstats
	out, err = m.runSSH(ctx, "virsh nodememstats")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				var val int64
				_, _ = fmt.Sscanf(fields[len(fields)-2], "%d", &val)
				switch {
				case strings.Contains(fields[0], "total"):
					result.TotalMemoryMB = val / 1024
				case strings.Contains(fields[0], "free"):
					result.AvailableMemoryMB = val / 1024
				}
			}
		}

		if result.TotalMemoryMB > 0 {
			if int64(requiredMemoryMB) > result.AvailableMemoryMB {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Insufficient memory on %s: need %d MB but only %d MB available",
						m.host.Name, requiredMemoryMB, result.AvailableMemoryMB))
			} else if float64(requiredMemoryMB) > float64(result.AvailableMemoryMB)*0.8 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Low memory warning on %s: requesting %d MB of %d MB available",
						m.host.Name, requiredMemoryMB, result.AvailableMemoryMB))
			}
		}
	} else {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Could not check memory on %s: %v", m.host.Name, err))
	}

	// Check disk space
	workDir := m.cfg.WorkDir
	if workDir == "" {
		workDir = "/var/lib/libvirt/images/sandboxes"
	}
	escapedWorkDir, err := shellEscape(workDir)
	if err == nil {
		out, err = m.runSSH(ctx, fmt.Sprintf("df -m %s | tail -1 | awk '{print $4}'", escapedWorkDir))
		if err == nil {
			var available int64
			_, _ = fmt.Sscanf(strings.TrimSpace(out), "%d", &available)
			result.AvailableDiskMB = available

			if available < 1024 {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Insufficient disk space on %s: only %d MB available in %s",
						m.host.Name, available, workDir))
			} else if available < 10*1024 {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Low disk space warning on %s: only %d MB available in %s",
						m.host.Name, available, workDir))
			}
		} else {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Could not check disk space on %s: %v", m.host.Name, err))
		}
	}

	return result, nil
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

// buildCloudInitSeedOnRemote creates a cloud-init ISO on the remote host.
// The key purpose is to provide a NEW instance-id that differs from what's stored
// on the cloned disk. This forces cloud-init to re-run its initialization,
// including network configuration for the clone's new MAC address.
func (m *RemoteVirshManager) buildCloudInitSeedOnRemote(ctx context.Context, vmName, jobDir, outISO string) error {
	// Build cloud-init user-data with DHCP networking
	userData := `#cloud-config
# Cloud-init config for cloned VMs
# This triggers cloud-init to re-run network configuration

# Ensure networking is configured via DHCP
network:
  version: 2
  ethernets:
    id0:
      match:
        driver: virtio*
      dhcp4: true
`

	// If SSH CA is configured, add sandbox user and SSH CA trust
	if m.cfg.SSHCAPubKey != "" {
		userData += fmt.Sprintf(`
# Create sandbox user for managed SSH credentials
users:
  - default
  - name: sandbox
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    lock_passwd: true

# Write SSH CA public key
write_files:
  - path: /etc/ssh/ssh_ca.pub
    content: |
      %s
    permissions: '0644'
    owner: root:root

# Configure sshd to trust the CA
runcmd:
  - |
    if [ -s /etc/ssh/ssh_ca.pub ]; then
      if ! grep -q "TrustedUserCAKeys" /etc/ssh/sshd_config; then
        echo "TrustedUserCAKeys /etc/ssh/ssh_ca.pub" >> /etc/ssh/sshd_config
        systemctl restart sshd || systemctl restart ssh || true
      fi
    fi
`, m.cfg.SSHCAPubKey)
	}

	// Use a unique instance-id based on the VM name
	metaData := fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, vmName, vmName)

	// Escape paths for shell
	escapedOutISO, err := shellEscape(outISO)
	if err != nil {
		return fmt.Errorf("invalid ISO path: %w", err)
	}

	// Write user-data and meta-data to remote host using base64
	userDataB64 := base64.StdEncoding.EncodeToString([]byte(userData))
	metaDataB64 := base64.StdEncoding.EncodeToString([]byte(metaData))

	userDataPath := fmt.Sprintf("%s/user-data", jobDir)
	metaDataPath := fmt.Sprintf("%s/meta-data", jobDir)
	escapedUserDataPath, err := shellEscape(userDataPath)
	if err != nil {
		return fmt.Errorf("invalid user-data path: %w", err)
	}
	escapedMetaDataPath, err := shellEscape(metaDataPath)
	if err != nil {
		return fmt.Errorf("invalid meta-data path: %w", err)
	}

	if _, err := m.runSSH(ctx, fmt.Sprintf("echo %s | base64 -d > %s", userDataB64, escapedUserDataPath)); err != nil {
		return fmt.Errorf("write user-data: %w", err)
	}
	if _, err := m.runSSH(ctx, fmt.Sprintf("echo %s | base64 -d > %s", metaDataB64, escapedMetaDataPath)); err != nil {
		return fmt.Errorf("write meta-data: %w", err)
	}

	// Try cloud-localds first, then genisoimage, then mkisofs
	isoCmd := fmt.Sprintf(`
if command -v cloud-localds >/dev/null 2>&1; then
  cloud-localds %s %s %s
elif command -v genisoimage >/dev/null 2>&1; then
  genisoimage -output %s -volid cidata -joliet -rock %s %s
elif command -v mkisofs >/dev/null 2>&1; then
  mkisofs -output %s -V cidata -J -R %s %s
else
  echo "No ISO creation tool found" >&2
  exit 1
fi
`, escapedOutISO, escapedUserDataPath, escapedMetaDataPath,
		escapedOutISO, escapedUserDataPath, escapedMetaDataPath,
		escapedOutISO, escapedUserDataPath, escapedMetaDataPath)

	if _, err := m.runSSH(ctx, isoCmd); err != nil {
		return fmt.Errorf("create cloud-init ISO: %w", err)
	}

	// Verify ISO was created
	if _, err := m.runSSH(ctx, fmt.Sprintf("test -f %s", escapedOutISO)); err != nil {
		return fmt.Errorf("cloud-init ISO not created at %s", outISO)
	}

	m.logger.Info("created cloud-init ISO on remote host",
		"host", m.host.Name,
		"vm", vmName,
		"iso", outISO,
	)

	return nil
}
