//go:build libvirt

package libvirt

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/beevik/etree"
)

// generateMACAddress generates a random MAC address with the locally administered bit set.
// Uses the 52:54:00 prefix which is commonly used by QEMU/KVM.
func generateMACAddress() string {
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", buf[0], buf[1], buf[2])
}

// Manager defines the VM orchestration operations we support against libvirt/KVM via virsh.
type Manager interface {
	// CloneVM creates a linked-clone VM from a golden base image and defines a libvirt domain for it.
	// cpu and memoryMB are the VM shape. network is the libvirt network name (e.g., "default").
	CloneVM(ctx context.Context, baseImage, newVMName string, cpu, memoryMB int, network string) (DomainRef, error)

	// CloneFromVM creates a linked-clone VM from an existing VM's disk.
	// It looks up the source VM by name in libvirt, retrieves its disk path,
	// and creates an overlay pointing to that disk as the backing file.
	CloneFromVM(ctx context.Context, sourceVMName, newVMName string, cpu, memoryMB int, network string) (DomainRef, error)

	// InjectSSHKey injects an SSH public key for a user into the VM disk before boot.
	// The mechanism is determined by configuration (e.g., virt-customize or cloud-init seed).
	InjectSSHKey(ctx context.Context, sandboxName, username, publicKey string) error

	// StartVM boots a defined domain.
	StartVM(ctx context.Context, vmName string) error

	// StopVM gracefully shuts down a domain, or forces if force is true.
	StopVM(ctx context.Context, vmName string, force bool) error

	// DestroyVM undefines the domain and removes its workspace (overlay files, domain XML, seeds).
	// If the domain is running, it will be destroyed first.
	DestroyVM(ctx context.Context, vmName string) error

	// CreateSnapshot creates a snapshot with the given name.
	// If external is true, attempts a disk-only external snapshot.
	CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (SnapshotRef, error)

	// DiffSnapshot prepares a plan to compare two snapshots' filesystems.
	// The returned plan includes advice or prepared mounts where possible.
	DiffSnapshot(ctx context.Context, vmName, fromSnapshot, toSnapshot string) (*FSComparePlan, error)

	// GetIPAddress attempts to fetch the VM's primary IP via libvirt leases.
	// Returns the IP address and MAC address of the VM's primary interface.
	GetIPAddress(ctx context.Context, vmName string, timeout time.Duration) (ip string, mac string, err error)

	// GetVMState returns the current state of a VM using virsh domstate.
	GetVMState(ctx context.Context, vmName string) (VMState, error)

	// ValidateSourceVM performs pre-flight checks on a source VM before cloning.
	// Returns a ValidationResult with warnings and errors about the VM's readiness.
	ValidateSourceVM(ctx context.Context, vmName string) (*VMValidationResult, error)

	// CheckHostResources validates that the host has sufficient resources for a new sandbox.
	// Returns a ResourceCheckResult with available resources and any warnings.
	CheckHostResources(ctx context.Context, requiredCPUs, requiredMemoryMB int) (*ResourceCheckResult, error)
}

// VMValidationResult contains the results of validating a source VM.
type VMValidationResult struct {
	VMName     string   `json:"vm_name"`
	Valid      bool     `json:"valid"`
	State      VMState  `json:"state"`
	MACAddress string   `json:"mac_address,omitempty"`
	IPAddress  string   `json:"ip_address,omitempty"`
	HasNetwork bool     `json:"has_network"`
	Warnings   []string `json:"warnings,omitempty"`
	Errors     []string `json:"errors,omitempty"`
}

// ResourceCheckResult contains the results of checking host resources.
type ResourceCheckResult struct {
	Valid             bool     `json:"valid"`
	AvailableMemoryMB int64    `json:"available_memory_mb"`
	TotalMemoryMB     int64    `json:"total_memory_mb"`
	AvailableCPUs     int      `json:"available_cpus"`
	AvailableDiskMB   int64    `json:"available_disk_mb"`
	RequiredMemoryMB  int      `json:"required_memory_mb"`
	RequiredCPUs      int      `json:"required_cpus"`
	Warnings          []string `json:"warnings,omitempty"`
	Errors            []string `json:"errors,omitempty"`
}

// VMState represents possible VM states from virsh domstate.
type VMState string

const (
	VMStateRunning   VMState = "running"
	VMStatePaused    VMState = "paused"
	VMStateShutOff   VMState = "shut off"
	VMStateCrashed   VMState = "crashed"
	VMStateSuspended VMState = "pmsuspended"
	VMStateUnknown   VMState = "unknown"
)

// Config controls how the virsh-based manager interacts with the host.
type Config struct {
	LibvirtURI            string // e.g., qemu:///system
	BaseImageDir          string // e.g., /var/lib/libvirt/images/base
	WorkDir               string // e.g., /var/lib/libvirt/images/jobs
	DefaultNetwork        string // e.g., default
	SSHKeyInjectMethod    string // "virt-customize" or "cloud-init"
	CloudInitMetaTemplate string // optional meta-data template for cloud-init seed

	// SSH CA public key for managed credentials.
	// If set, this will be injected into VMs via cloud-init so they trust
	// certificates signed by this CA.
	SSHCAPubKey string

	// SSH ProxyJump host for reaching VMs on an isolated network.
	// Format: "user@host:port" or just "host" for default user/port.
	// If set, SSH commands will use -J flag to proxy through this host.
	SSHProxyJump string

	// Optional explicit paths to binaries; if empty these are looked up in PATH.
	VirshPath         string
	QemuImgPath       string
	VirtCustomizePath string
	QemuNbdPath       string

	// Socket VMNet configuration (macOS only)
	// If DefaultNetwork is "socket_vmnet", this wrapper script is used as the emulator.
	// The wrapper should invoke qemu through socket_vmnet_client.
	SocketVMNetWrapper string // e.g., /path/to/qemu-socket-vmnet-wrapper.sh

	// Domain defaults
	DefaultVCPUs    int
	DefaultMemoryMB int
}

// DomainRef is a minimal reference to a libvirt domain (VM).
type DomainRef struct {
	Name string
	UUID string
}

// SnapshotRef references a snapshot created for a domain.
type SnapshotRef struct {
	Name string
	// Kind: "INTERNAL" or "EXTERNAL"
	Kind string
	// Ref is driver-specific; could be an internal UUID or a file path for external snapshots.
	Ref string
}

// FSComparePlan describes a plan for diffing two snapshots' filesystems.
type FSComparePlan struct {
	VMName       string
	FromSnapshot string
	ToSnapshot   string

	// Best-effort mount points (if prepared); may be empty strings when not mounted automatically.
	FromMount string
	ToMount   string

	// Devices or files used; informative.
	FromRef string
	ToRef   string

	// Free-form notes with instructions if the manager couldn't mount automatically.
	Notes []string
}

// VirshManager implements Manager using virsh/qemu-img/qemu-nbd/virt-customize and simple domain XML.
type VirshManager struct {
	cfg    Config
	logger *slog.Logger
}

// NewVirshManager creates a new VirshManager with the provided config and optional logger.
// If logger is nil, slog.Default() is used.
func NewVirshManager(cfg Config, logger *slog.Logger) *VirshManager {
	// Fill sensible defaults
	if cfg.DefaultVCPUs == 0 {
		cfg.DefaultVCPUs = 2
	}
	if cfg.DefaultMemoryMB == 0 {
		cfg.DefaultMemoryMB = 2048
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &VirshManager{cfg: cfg, logger: logger}
}

// NewFromEnv builds a Config from environment variables and returns a manager.
// LIBVIRT_URI, BASE_IMAGE_DIR, SANDBOX_WORKDIR, LIBVIRT_NETWORK, SSH_KEY_INJECT_METHOD
func NewFromEnv() *VirshManager {
	cfg := Config{
		LibvirtURI:         getenvDefault("LIBVIRT_URI", "qemu:///system"),
		BaseImageDir:       getenvDefault("BASE_IMAGE_DIR", "/var/lib/libvirt/images/base"),
		WorkDir:            getenvDefault("SANDBOX_WORKDIR", "/var/lib/libvirt/images/jobs"),
		DefaultNetwork:     getenvDefault("LIBVIRT_NETWORK", "default"),
		SSHKeyInjectMethod: getenvDefault("SSH_KEY_INJECT_METHOD", "virt-customize"),
		SSHCAPubKey:        readSSHCAPubKey(getenvDefault("SSH_CA_PUB_KEY_PATH", "")),
		SSHProxyJump:       getenvDefault("SSH_PROXY_JUMP", ""),
		DefaultVCPUs:       intFromEnv("DEFAULT_VCPUS", 2),
		DefaultMemoryMB:    intFromEnv("DEFAULT_MEMORY_MB", 2048),
	}
	return NewVirshManager(cfg, nil)
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	return Config{
		LibvirtURI:         getenvDefault("LIBVIRT_URI", "qemu:///system"),
		BaseImageDir:       getenvDefault("BASE_IMAGE_DIR", "/var/lib/libvirt/images/base"),
		WorkDir:            getenvDefault("SANDBOX_WORKDIR", "/var/lib/libvirt/images/jobs"),
		DefaultNetwork:     getenvDefault("LIBVIRT_NETWORK", "default"),
		SSHKeyInjectMethod: getenvDefault("SSH_KEY_INJECT_METHOD", "virt-customize"),
		SSHCAPubKey:        readSSHCAPubKey(getenvDefault("SSH_CA_PUB_KEY_PATH", "")),
		SSHProxyJump:       getenvDefault("SSH_PROXY_JUMP", ""),
		SocketVMNetWrapper: getenvDefault("SOCKET_VMNET_WRAPPER", ""),
		DefaultVCPUs:       intFromEnv("DEFAULT_VCPUS", 2),
		DefaultMemoryMB:    intFromEnv("DEFAULT_MEMORY_MB", 2048),
	}
}

// readSSHCAPubKey reads the SSH CA public key from a file path.
// Returns empty string if path is empty or file cannot be read.
func readSSHCAPubKey(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (m *VirshManager) CloneVM(ctx context.Context, baseImage, newVMName string, cpu, memoryMB int, network string) (DomainRef, error) {
	if newVMName == "" {
		return DomainRef{}, fmt.Errorf("new VM name is required")
	}
	if baseImage == "" {
		return DomainRef{}, fmt.Errorf("base image is required")
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

	basePath := filepath.Join(m.cfg.BaseImageDir, baseImage)
	if _, err := os.Stat(basePath); err != nil {
		return DomainRef{}, fmt.Errorf("base image not accessible: %s: %w", basePath, err)
	}

	jobDir := filepath.Join(m.cfg.WorkDir, newVMName)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		return DomainRef{}, fmt.Errorf("create job dir: %w", err)
	}

	overlayPath := filepath.Join(jobDir, "disk-overlay.qcow2")
	qemuImg := m.binPath("qemu-img", m.cfg.QemuImgPath)
	if _, err := m.run(ctx, qemuImg, "create", "-f", "qcow2", "-F", "qcow2", "-b", basePath, overlayPath); err != nil {
		return DomainRef{}, fmt.Errorf("create overlay: %w", err)
	}

	// Create minimal domain XML referencing overlay disk and network.
	xmlPath := filepath.Join(jobDir, "domain.xml")
	xml, err := renderDomainXML(domainXMLParams{
		Name:      newVMName,
		MemoryMB:  memoryMB,
		VCPUs:     cpu,
		DiskPath:  overlayPath,
		Network:   network,
		BootOrder: []string{"hd", "cdrom", "network"},
	})
	log.Println("Generated domain XML:", xml)
	if err != nil {
		return DomainRef{}, fmt.Errorf("render domain xml: %w", err)
	}
	if err := os.WriteFile(xmlPath, []byte(xml), 0o644); err != nil {
		return DomainRef{}, fmt.Errorf("write domain xml: %w", err)
	}

	// virsh define
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	if _, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "define", xmlPath); err != nil {
		return DomainRef{}, fmt.Errorf("virsh define: %w", err)
	}

	// Fetch UUID
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domuuid", newVMName)
	if err != nil {
		// Best-effort: If domuuid fails, we still return Name.
		return DomainRef{Name: newVMName}, nil
	}
	return DomainRef{Name: newVMName, UUID: strings.TrimSpace(out)}, nil
}

// CloneFromVM creates a linked-clone VM from an existing VM's disk.
// It looks up the source VM by name, retrieves its disk path, and creates an overlay.
func (m *VirshManager) CloneFromVM(ctx context.Context, sourceVMName, newVMName string, cpu, memoryMB int, network string) (DomainRef, error) {
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

	// Look up the source VM's disk path using virsh domblklist
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domblklist", sourceVMName, "--details")
	if err != nil {
		return DomainRef{}, fmt.Errorf("lookup source VM %q: %w", sourceVMName, err)
	}

	// Parse domblklist output to find the disk path
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

	// Verify the disk exists
	if _, err := os.Stat(basePath); err != nil {
		return DomainRef{}, fmt.Errorf("source VM disk not accessible: %s: %w", basePath, err)
	}

	jobDir := filepath.Join(m.cfg.WorkDir, newVMName)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		return DomainRef{}, fmt.Errorf("create job dir: %w", err)
	}

	overlayPath := filepath.Join(jobDir, "disk-overlay.qcow2")
	qemuImg := m.binPath("qemu-img", m.cfg.QemuImgPath)
	if _, err := m.run(ctx, qemuImg, "create", "-f", "qcow2", "-F", "qcow2", "-b", basePath, overlayPath); err != nil {
		return DomainRef{}, fmt.Errorf("create overlay: %w", err)
	}

	// Generate a unique cloud-init ISO for the cloned VM
	// This ensures the clone gets a new instance-id and DHCP network config,
	// regardless of how the source VM's cloud-init was configured (static IP, MAC, etc.)
	cloudInitISO := filepath.Join(jobDir, "cloud-init.iso")
	if err := m.buildCloudInitSeedForClone(ctx, newVMName, cloudInitISO); err != nil {
		// Log warning but don't fail - VM might still work if source didn't use cloud-init
		log.Printf("WARNING: failed to build cloud-init seed for clone %s: %v", newVMName, err)
		cloudInitISO = "" // Don't try to attach a non-existent ISO
	}

	// Dump the source VM's XML and modify it for the new VM
	sourceXML, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "dumpxml", sourceVMName)
	if err != nil {
		return DomainRef{}, fmt.Errorf("dumpxml source vm: %w", err)
	}

	newXML, err := modifyClonedXMLHelper(sourceXML, newVMName, overlayPath, cloudInitISO, cpu, memoryMB, network)
	if err != nil {
		return DomainRef{}, fmt.Errorf("modify cloned xml: %w", err)
	}

	xmlPath := filepath.Join(jobDir, "domain.xml")
	if err := os.WriteFile(xmlPath, []byte(newXML), 0o644); err != nil {
		return DomainRef{}, fmt.Errorf("write domain xml: %w", err)
	}

	// virsh define
	if _, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "define", xmlPath); err != nil {
		return DomainRef{}, fmt.Errorf("virsh define: %w", err)
	}

	// Fetch UUID
	out, err = m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domuuid", newVMName)
	if err != nil {
		return DomainRef{Name: newVMName}, nil
	}
	return DomainRef{Name: newVMName, UUID: strings.TrimSpace(out)}, nil
}

// modifyClonedXML takes the XML from a source domain and adapts it for a new cloned domain.
// It sets a new name, UUID, disk path, MAC address, and cloud-init ISO path. It removes the
// <address> element from the network interface to prevent PCI slot conflicts.
// If cloudInitISO is provided, any existing CDROM device is updated to use it, ensuring the
// cloned VM gets a unique instance-id and fresh network configuration via cloud-init.
func modifyClonedXML(sourceXML, newName, newDiskPath, cloudInitISO string) (string, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromString(sourceXML); err != nil {
		return "", fmt.Errorf("parse source XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return "", fmt.Errorf("invalid XML: no root element")
	}

	// Update VM name
	nameElem := root.SelectElement("name")
	if nameElem == nil {
		return "", fmt.Errorf("invalid XML: missing <name> element")
	}
	nameElem.SetText(newName)

	// Remove UUID
	if uuidElem := root.SelectElement("uuid"); uuidElem != nil {
		root.RemoveChild(uuidElem)
	}

	// Update disk path for the main virtual disk (vda)
	// This finds the first disk with a virtio bus and assumes it's the one to replace.
	// This might need to be more robust if multiple virtio disks are present.
	var diskReplaced bool
	for _, disk := range root.FindElements("./devices/disk[@device='disk']") {
		if target := disk.SelectElement("target"); target != nil {
			if bus := target.SelectAttr("bus"); bus != nil && bus.Value == "virtio" {
				if source := disk.SelectElement("source"); source != nil {
					source.SelectAttr("file").Value = newDiskPath
					diskReplaced = true
					break
				}
			}
		}
	}
	if !diskReplaced {
		return "", fmt.Errorf("could not find a virtio disk in the source XML to replace")
	}

	// Handle cloud-init CDROM: update existing or add new one
	// This is critical for cloned VMs - they need a unique instance-id to trigger
	// cloud-init re-initialization, including DHCP network configuration
	if cloudInitISO != "" {
		devices := root.SelectElement("devices")
		if devices == nil {
			return "", fmt.Errorf("invalid XML: missing <devices> element")
		}

		// Look for existing CDROM device to update
		var cdromUpdated bool
		for _, disk := range root.FindElements("./devices/disk[@device='cdrom']") {
			if source := disk.SelectElement("source"); source != nil {
				if fileAttr := source.SelectAttr("file"); fileAttr != nil {
					fileAttr.Value = cloudInitISO
					cdromUpdated = true
					break
				}
			}
		}

		// If no existing CDROM, add one with SCSI controller
		if !cdromUpdated {
			// Add SCSI controller if not present
			hasScsiController := false
			for _, ctrl := range root.FindElements("./devices/controller[@type='scsi']") {
				if model := ctrl.SelectAttr("model"); model != nil && model.Value == "virtio-scsi" {
					hasScsiController = true
					break
				}
			}
			if !hasScsiController {
				scsiCtrl := devices.CreateElement("controller")
				scsiCtrl.CreateAttr("type", "scsi")
				scsiCtrl.CreateAttr("model", "virtio-scsi")
			}

			// Add CDROM device
			cdrom := devices.CreateElement("disk")
			cdrom.CreateAttr("type", "file")
			cdrom.CreateAttr("device", "cdrom")

			driver := cdrom.CreateElement("driver")
			driver.CreateAttr("name", "qemu")
			driver.CreateAttr("type", "raw")

			source := cdrom.CreateElement("source")
			source.CreateAttr("file", cloudInitISO)

			target := cdrom.CreateElement("target")
			target.CreateAttr("dev", "sda")
			target.CreateAttr("bus", "scsi")

			cdrom.CreateElement("readonly")
		}
	}

	// Update network interface: set new MAC and remove PCI address
	if iface := root.FindElement("./devices/interface"); iface != nil {
		// This handles standard libvirt network interfaces.
		// Set a new MAC address
		macElem := iface.SelectElement("mac")
		if macElem != nil {
			if addrAttr := macElem.SelectAttr("address"); addrAttr != nil {
				addrAttr.Value = generateMACAddress()
			}
		} else {
			// If no <mac> element, create one
			macElem = iface.CreateElement("mac")
			macElem.CreateAttr("address", generateMACAddress())
		}

		// Remove the address element to let libvirt assign a new one
		if addrElem := iface.SelectElement("address"); addrElem != nil {
			iface.RemoveChild(addrElem)
		}
	} else {
		// Handle socket_vmnet case (qemu:commandline)
		// The namespace makes selection tricky, so we iterate.
		var cmdline *etree.Element
		for _, child := range root.ChildElements() {
			if child.Tag == "commandline" && child.Space == "qemu" {
				cmdline = child
				break
			}
		}

		if cmdline != nil {
			for _, child := range cmdline.ChildElements() {
				if child.Tag == "arg" && child.Space == "qemu" {
					if valAttr := child.SelectAttr("value"); valAttr != nil {
						if strings.HasPrefix(valAttr.Value, "virtio-net-pci") && strings.Contains(valAttr.Value, "mac=") {
							parts := strings.Split(valAttr.Value, ",")
							newParts := make([]string, 0, len(parts))
							macUpdated := false
							for _, part := range parts {
								if strings.HasPrefix(part, "mac=") {
									newParts = append(newParts, "mac="+generateMACAddress())
									macUpdated = true
								} else {
									newParts = append(newParts, part)
								}
							}
							if macUpdated {
								valAttr.Value = strings.Join(newParts, ",")
								break // Assuming only one network device per command line
							}
						}
					}
				}
			}
		}
	}

	// Remove existing graphics password
	if graphics := root.FindElement("./devices/graphics"); graphics != nil {
		graphics.RemoveAttr("passwd")
	}

	// Remove existing sound devices
	for _, sound := range root.FindElements("./devices/sound") {
		root.SelectElement("devices").RemoveChild(sound)
	}

	doc.Indent(2)
	newXML, err := doc.WriteToString()
	if err != nil {
		return "", fmt.Errorf("failed to write modified XML: %w", err)
	}

	return newXML, nil
}

func (m *VirshManager) InjectSSHKey(ctx context.Context, sandboxName, username, publicKey string) error {
	if sandboxName == "" {
		return fmt.Errorf("sandboxName is required")
	}
	if username == "" {
		username = defaultGuestUser(sandboxName)
	}
	if strings.TrimSpace(publicKey) == "" {
		return fmt.Errorf("publicKey is required")
	}

	jobDir := filepath.Join(m.cfg.WorkDir, sandboxName)
	overlay := filepath.Join(jobDir, "disk-overlay.qcow2")
	if _, err := os.Stat(overlay); err != nil {
		return fmt.Errorf("overlay not found for VM %s: %w", sandboxName, err)
	}

	switch strings.ToLower(m.cfg.SSHKeyInjectMethod) {
	case "virt-customize":
		// Requires libguestfs tools on host.
		virtCustomize := m.binPath("virt-customize", m.cfg.VirtCustomizePath)
		// Ensure account exists and inject key. This is offline before first boot.
		cmdArgs := []string{
			"-a", overlay,
			"--run-command", fmt.Sprintf("id -u %s >/dev/null 2>&1 || useradd -m -s /bin/bash %s", shEscape(username), shEscape(username)),
			"--ssh-inject", fmt.Sprintf("%s:string:%s", username, publicKey),
		}
		if _, err := m.run(ctx, virtCustomize, cmdArgs...); err != nil {
			return fmt.Errorf("virt-customize inject: %w", err)
		}
	case "cloud-init":
		// Build a NoCloud seed with the provided key and attach as CD-ROM.
		seedISO := filepath.Join(jobDir, "seed.iso")
		if err := m.buildCloudInitSeed(ctx, sandboxName, username, publicKey, seedISO); err != nil {
			return fmt.Errorf("build cloud-init seed: %w", err)
		}
		// Attach seed ISO to domain XML (adds a CDROM) and redefine the domain.
		xmlPath := filepath.Join(jobDir, "domain.xml")
		if err := m.attachISOToDomainXML(xmlPath, seedISO); err != nil {
			return fmt.Errorf("attach seed iso to domain xml: %w", err)
		}
		virsh := m.binPath("virsh", m.cfg.VirshPath)
		if _, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "define", xmlPath); err != nil {
			return fmt.Errorf("re-define domain with seed: %w", err)
		}
	default:
		return fmt.Errorf("unsupported SSHKeyInjectMethod: %s", m.cfg.SSHKeyInjectMethod)
	}
	return nil
}

func (m *VirshManager) StartVM(ctx context.Context, vmName string) error {
	if vmName == "" {
		return fmt.Errorf("vmName is required")
	}

	m.logger.Info("starting VM",
		"vm_name", vmName,
		"libvirt_uri", m.cfg.LibvirtURI,
	)

	virsh := m.binPath("virsh", m.cfg.VirshPath)
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "start", vmName)
	if err != nil {
		m.logger.Error("failed to start VM",
			"vm_name", vmName,
			"error", err,
			"output", out,
		)
		return err
	}

	m.logger.Debug("virsh start command completed",
		"vm_name", vmName,
		"output", out,
	)

	// Verify VM actually started by checking state
	state, stateErr := m.GetVMState(ctx, vmName)
	if stateErr != nil {
		m.logger.Warn("unable to verify VM state after start",
			"vm_name", vmName,
			"error", stateErr,
		)
	} else {
		m.logger.Info("VM state after start command",
			"vm_name", vmName,
			"state", state,
		)
		if state != VMStateRunning {
			m.logger.Warn("VM not in running state after start command",
				"vm_name", vmName,
				"actual_state", state,
				"expected_state", VMStateRunning,
				"hint", "On ARM Macs with Lima, VMs may fail to start due to CPU mode limitations",
			)
		}
	}

	return nil
}

// GetVMState returns the current state of a VM using virsh domstate.
func (m *VirshManager) GetVMState(ctx context.Context, vmName string) (VMState, error) {
	if vmName == "" {
		return VMStateUnknown, fmt.Errorf("vmName is required")
	}
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domstate", vmName)
	if err != nil {
		return VMStateUnknown, fmt.Errorf("get vm state: %w", err)
	}
	return parseVMState(out), nil
}

// parseVMState converts virsh domstate output to VMState.
func parseVMState(output string) VMState {
	state := strings.TrimSpace(output)
	switch state {
	case "running":
		return VMStateRunning
	case "paused":
		return VMStatePaused
	case "shut off":
		return VMStateShutOff
	case "crashed":
		return VMStateCrashed
	case "pmsuspended":
		return VMStateSuspended
	default:
		return VMStateUnknown
	}
}

// GetVMMAC returns the MAC address of the VM's primary network interface.
// This is useful for DHCP lease management.
func (m *VirshManager) GetVMMAC(ctx context.Context, vmName string) (string, error) {
	if vmName == "" {
		return "", fmt.Errorf("vmName is required")
	}
	virsh := m.binPath("virsh", m.cfg.VirshPath)

	// Use domiflist to get interface info (works even if VM is not running)
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domiflist", vmName)
	if err != nil {
		return "", fmt.Errorf("get vm interfaces: %w", err)
	}

	// Parse domiflist output:
	// Interface  Type       Source     Model       MAC
	// -------------------------------------------------------
	// -          network    default    virtio      52:54:00:6b:3c:86
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Interface") || strings.HasPrefix(line, "-") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			mac := fields[4]
			// Validate MAC format
			if strings.Count(mac, ":") == 5 {
				return mac, nil
			}
		}
	}
	return "", fmt.Errorf("no MAC address found for VM %s", vmName)
}

// ReleaseDHCPLease attempts to release the DHCP lease for a given MAC address.
// This helps prevent IP conflicts when VMs are rapidly created and destroyed.
// It tries multiple methods:
// 1. Remove static DHCP host entry (if any)
// 2. Use dhcp_release utility to release dynamic lease
// 3. Remove from lease file directly as fallback
func (m *VirshManager) ReleaseDHCPLease(ctx context.Context, network, mac string) error {
	if network == "" {
		network = m.cfg.DefaultNetwork
	}
	if mac == "" {
		return fmt.Errorf("MAC address is required")
	}

	virsh := m.binPath("virsh", m.cfg.VirshPath)

	// Try to remove any static DHCP host entry (if exists)
	// This is a best-effort operation - it may fail if no static entry exists
	hostXML := fmt.Sprintf("<host mac='%s'/>", mac)
	_, _ = m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI,
		"net-update", network, "delete", "ip-dhcp-host", hostXML, "--live", "--config")

	// Get the bridge interface name for the network (e.g., virbr0)
	bridgeName, ip := m.getNetworkBridgeAndLeaseIP(ctx, network, mac)

	if bridgeName != "" && ip != "" {
		// Try dhcp_release utility first (cleanest method)
		// dhcp_release <interface> <ip> <mac>
		if _, err := m.run(ctx, "dhcp_release", bridgeName, ip, mac); err == nil {
			m.logger.Info("released DHCP lease via dhcp_release",
				"network", network,
				"bridge", bridgeName,
				"ip", ip,
				"mac", mac,
			)
			return nil
		}

		// Fallback: try to remove from lease file directly
		if err := m.removeLeaseFromFile(network, mac); err == nil {
			m.logger.Info("removed DHCP lease from lease file",
				"network", network,
				"mac", mac,
			)
			return nil
		}
	}

	m.logger.Debug("DHCP lease release attempted (may not have fully succeeded)",
		"network", network,
		"mac", mac,
	)

	return nil
}

// getNetworkBridgeAndLeaseIP returns the bridge interface name and leased IP for a MAC address.
func (m *VirshManager) getNetworkBridgeAndLeaseIP(ctx context.Context, network, mac string) (bridge, ip string) {
	virsh := m.binPath("virsh", m.cfg.VirshPath)

	// Get bridge name from network XML
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "net-info", network)
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "Bridge:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					bridge = parts[1]
				}
			}
		}
	}

	// Get IP from DHCP leases
	out, err = m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "net-dhcp-leases", network)
	if err == nil {
		// Parse output:
		// Expiry Time           MAC address         Protocol   IP address          Hostname   Client ID
		// 2024-01-08 12:00:00   52:54:00:6b:3c:86   ipv4       192.168.122.63/24   vm-name    -
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, mac) {
				fields := strings.Fields(line)
				// Fields: [date, time, mac, protocol, ip/cidr, hostname, clientid]
				if len(fields) >= 5 {
					ipCIDR := fields[4]
					if idx := strings.Index(ipCIDR, "/"); idx > 0 {
						ip = ipCIDR[:idx]
					} else {
						ip = ipCIDR
					}
				}
			}
		}
	}

	return bridge, ip
}

// removeLeaseFromFile removes a DHCP lease entry from the dnsmasq lease file.
func (m *VirshManager) removeLeaseFromFile(network, mac string) error {
	// Lease file is typically at /var/lib/libvirt/dnsmasq/<network>.leases
	leaseFile := fmt.Sprintf("/var/lib/libvirt/dnsmasq/%s.leases", network)

	data, err := os.ReadFile(leaseFile)
	if err != nil {
		return fmt.Errorf("read lease file: %w", err)
	}

	// Lease file format: <expiry> <mac> <ip> <hostname> <client-id>
	// Example: 1704672000 52:54:00:6b:3c:86 192.168.122.63 vm-name *
	var newLines []string
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, mac) {
			found = true
			continue // Skip this line (remove the lease)
		}
		newLines = append(newLines, line)
	}

	if !found {
		return fmt.Errorf("lease not found for MAC %s", mac)
	}

	// Write back the modified lease file
	newData := strings.Join(newLines, "\n")
	if len(newLines) > 0 {
		newData += "\n"
	}
	if err := os.WriteFile(leaseFile, []byte(newData), 0o644); err != nil {
		return fmt.Errorf("write lease file: %w", err)
	}

	return nil
}

func (m *VirshManager) StopVM(ctx context.Context, vmName string, force bool) error {
	if vmName == "" {
		return fmt.Errorf("vmName is required")
	}
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	if force {
		_, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "destroy", vmName)
		return err
	}
	_, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "shutdown", vmName)
	return err
}

func (m *VirshManager) DestroyVM(ctx context.Context, vmName string) error {
	if vmName == "" {
		return fmt.Errorf("vmName is required")
	}

	// Get MAC address before destroying (for DHCP lease cleanup)
	mac, macErr := m.GetVMMAC(ctx, vmName)
	if macErr != nil {
		m.logger.Debug("could not get MAC address for DHCP cleanup",
			"vm_name", vmName,
			"error", macErr,
		)
	}

	virsh := m.binPath("virsh", m.cfg.VirshPath)
	// Best-effort destroy if running
	_, _ = m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "destroy", vmName)
	// Undefine
	if _, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "undefine", vmName); err != nil {
		// continue to remove files even if undefine fails
		_ = err
	}

	// Release DHCP lease to prevent IP conflicts with future VMs
	if mac != "" {
		if err := m.ReleaseDHCPLease(ctx, m.cfg.DefaultNetwork, mac); err != nil {
			m.logger.Debug("failed to release DHCP lease",
				"vm_name", vmName,
				"mac", mac,
				"error", err,
			)
		} else {
			m.logger.Info("released DHCP lease",
				"vm_name", vmName,
				"mac", mac,
			)
		}
	}

	// Remove workspace
	jobDir := filepath.Join(m.cfg.WorkDir, vmName)
	if err := os.RemoveAll(jobDir); err != nil {
		return fmt.Errorf("cleanup job dir: %w", err)
	}
	return nil
}

func (m *VirshManager) CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (SnapshotRef, error) {
	if vmName == "" || snapshotName == "" {
		return SnapshotRef{}, fmt.Errorf("vmName and snapshotName are required")
	}
	virsh := m.binPath("virsh", m.cfg.VirshPath)

	if external {
		// External disk-only snapshot.
		jobDir := filepath.Join(m.cfg.WorkDir, vmName)
		snapPath := filepath.Join(jobDir, fmt.Sprintf("snap-%s.qcow2", snapshotName))
		// NOTE: This is a simplified attempt; real-world disk-only snapshots may need
		// additional options and disk target identification.
		args := []string{
			"--connect", m.cfg.LibvirtURI, "snapshot-create-as", vmName, snapshotName,
			"--disk-only", "--atomic", "--no-metadata",
			"--diskspec", fmt.Sprintf("vda,file=%s", snapPath),
		}
		if _, err := m.run(ctx, virsh, args...); err != nil {
			return SnapshotRef{}, fmt.Errorf("external snapshot create: %w", err)
		}
		return SnapshotRef{Name: snapshotName, Kind: "EXTERNAL", Ref: snapPath}, nil
	}

	// Internal snapshot (managed by libvirt/qemu).
	if _, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "snapshot-create-as", vmName, snapshotName); err != nil {
		return SnapshotRef{}, fmt.Errorf("internal snapshot create: %w", err)
	}
	return SnapshotRef{Name: snapshotName, Kind: "INTERNAL", Ref: snapshotName}, nil
}

func (m *VirshManager) DiffSnapshot(ctx context.Context, vmName, fromSnapshot, toSnapshot string) (*FSComparePlan, error) {
	if vmName == "" || fromSnapshot == "" || toSnapshot == "" {
		return nil, fmt.Errorf("vmName, fromSnapshot and toSnapshot are required")
	}

	// Implementation shell:
	// Strategy options:
	// 1) For internal snapshots: use qemu-nbd with snapshot selection to mount and diff trees.
	// 2) For external snapshots: mount the two qcow2 snapshot files via qemu-nbd.
	//
	// Because snapshot storage varies, we return advisory plan data and notes.
	plan := &FSComparePlan{
		VMName:       vmName,
		FromSnapshot: fromSnapshot,
		ToSnapshot:   toSnapshot,
		Notes:        []string{},
	}

	// Attempt to detect external snapshot files in job dir.
	jobDir := filepath.Join(m.cfg.WorkDir, vmName)
	fromPath := filepath.Join(jobDir, fmt.Sprintf("snap-%s.qcow2", fromSnapshot))
	toPath := filepath.Join(jobDir, fmt.Sprintf("snap-%s.qcow2", toSnapshot))
	if fileExists(fromPath) && fileExists(toPath) {
		plan.FromRef = fromPath
		plan.ToRef = toPath
		plan.Notes = append(plan.Notes,
			"External snapshots detected. You can mount them with qemu-nbd and diff the trees.",
			fmt.Sprintf("sudo modprobe nbd max_part=16 && sudo qemu-nbd --connect=/dev/nbd0 %s", shEscape(fromPath)),
			fmt.Sprintf("sudo qemu-nbd --connect=/dev/nbd1 %s", shEscape(toPath)),
			"sudo mount /dev/nbd0p1 /mnt/from && sudo mount /dev/nbd1p1 /mnt/to",
			"Then run: sudo diff -ruN /mnt/from /mnt/to or use rsync --dry-run to list changes.",
			"Be sure to umount and disconnect nbd after.",
		)
		return plan, nil
	}

	// Fallback: internal snapshots guidance.
	plan.Notes = append(plan.Notes,
		"Internal snapshots assumed. Use qemu-nbd with -s to select snapshot, then mount and diff.",
		"For example: qemu-nbd may support --snapshot=<name> (varies by version) or use qemu-img to create temporary exports.",
		"Alternatively, boot the VM into each snapshot separately and export filesystem states.",
	)
	return plan, nil
}

func (m *VirshManager) GetIPAddress(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
	if vmName == "" {
		return "", "", fmt.Errorf("vmName is required")
	}

	m.logger.Info("discovering IP address",
		"vm_name", vmName,
		"timeout", timeout,
		"network", m.cfg.DefaultNetwork,
	)

	// First check VM state - if not running, IP discovery will definitely fail
	state, stateErr := m.GetVMState(ctx, vmName)
	if stateErr == nil && state != VMStateRunning {
		m.logger.Warn("attempting IP discovery on non-running VM",
			"vm_name", vmName,
			"state", state,
			"hint", "VM must be in 'running' state to have an IP address",
		)
	}

	// For socket_vmnet, use ARP-based discovery
	if m.cfg.DefaultNetwork == "socket_vmnet" {
		return m.getIPAddressViaARP(ctx, vmName, timeout)
	}

	// For regular libvirt networks, use lease-based discovery
	return m.getIPAddressViaLease(ctx, vmName, timeout)
}

// getIPAddressViaLease discovers IP using libvirt DHCP lease information.
// This works for libvirt-managed networks (default, NAT, etc.)
func (m *VirshManager) getIPAddressViaLease(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	deadline := time.Now().Add(timeout)
	startTime := time.Now()
	attempt := 0
	for {
		attempt++
		out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domifaddr", vmName, "--source", "lease")
		if err == nil {
			ip, mac := parseDomIfAddrIPv4WithMAC(out)
			if ip != "" {
				m.logger.Info("IP address discovered via lease",
					"vm_name", vmName,
					"ip_address", ip,
					"mac_address", mac,
					"attempts", attempt,
					"elapsed", time.Since(startTime),
				)
				return ip, mac, nil
			}
		}

		// Log progress every 10 attempts (20 seconds)
		if attempt%10 == 0 {
			m.logger.Debug("IP discovery in progress (lease)",
				"vm_name", vmName,
				"attempts", attempt,
				"elapsed", time.Since(startTime),
				"domifaddr_output", out,
			)
		}

		if time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Final state check for better error message
	finalState, _ := m.GetVMState(ctx, vmName)
	m.logger.Error("IP address discovery failed (lease)",
		"vm_name", vmName,
		"timeout", timeout,
		"attempts", attempt,
		"final_vm_state", finalState,
	)

	return "", "", fmt.Errorf("ip address not found within timeout (VM state: %s)", finalState)
}

// getIPAddressViaARP discovers IP using ARP table lookup.
// This is used for socket_vmnet on macOS where libvirt doesn't manage DHCP.
func (m *VirshManager) getIPAddressViaARP(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
	// First, get the VM's MAC address from the domain XML
	mac, err := m.getVMMAC(ctx, vmName)
	if err != nil {
		m.logger.Error("failed to get VM MAC address for ARP lookup",
			"vm_name", vmName,
			"error", err,
		)
		return "", "", fmt.Errorf("failed to get VM MAC address: %w", err)
	}

	m.logger.Info("starting ARP-based IP discovery",
		"vm_name", vmName,
		"mac_address", mac,
		"timeout", timeout,
	)

	deadline := time.Now().Add(timeout)
	startTime := time.Now()
	attempt := 0
	for {
		attempt++
		ip, err := lookupIPByMAC(mac)
		if err == nil && ip != "" {
			m.logger.Info("IP address discovered via ARP",
				"vm_name", vmName,
				"ip_address", ip,
				"mac_address", mac,
				"attempts", attempt,
				"elapsed", time.Since(startTime),
			)
			return ip, mac, nil
		}

		// Log progress every 10 attempts (20 seconds)
		if attempt%10 == 0 {
			m.logger.Debug("IP discovery in progress (ARP)",
				"vm_name", vmName,
				"mac_address", mac,
				"attempts", attempt,
				"elapsed", time.Since(startTime),
			)
		}

		if time.Now().After(deadline) {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Final state check for better error message
	finalState, _ := m.GetVMState(ctx, vmName)
	m.logger.Error("IP address discovery failed (ARP)",
		"vm_name", vmName,
		"mac_address", mac,
		"timeout", timeout,
		"attempts", attempt,
		"final_vm_state", finalState,
	)

	return "", "", fmt.Errorf("ip address not found in ARP table within timeout (VM state: %s, MAC: %s)", finalState, mac)
}

// --- Helpers ---

func (m *VirshManager) binPath(defaultName, override string) string {
	if override != "" {
		return override
	}
	return defaultName
}

func (m *VirshManager) run(ctx context.Context, bin string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	// Provide a default timeout if the context has none.
	if _, ok := ctx.Deadline(); !ok {
		ctx2, cancel := context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
		ctx = ctx2
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Pass LIBVIRT_DEFAULT_URI for convenience when set.
	env := os.Environ()
	if m.cfg.LibvirtURI != "" {
		env = append(env, "LIBVIRT_DEFAULT_URI="+m.cfg.LibvirtURI)
	}
	cmd.Env = env

	err := cmd.Run()
	outStr := strings.TrimSpace(stdout.String())
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return outStr, fmt.Errorf("%s %s failed: %w: %s", bin, strings.Join(args, " "), err, errStr)
		}
		return outStr, fmt.Errorf("%s %s failed: %w", bin, strings.Join(args, " "), err)
	}
	return outStr, nil
}

func getenvDefault(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func intFromEnv(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	var parsed int
	_, err := fmt.Sscanf(v, "%d", &parsed)
	if err != nil {
		return def
	}
	return parsed
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func shEscape(s string) string {
	// naive escape for use inside run-command; rely on controlled inputs.
	s = strings.ReplaceAll(s, `'`, `'\'\'`)
	return s
}

func defaultGuestUser(vmName string) string {
	// Heuristic default depending on distro naming conventions.
	// Adjust as needed by calling code.
	if strings.Contains(strings.ToLower(vmName), "ubuntu") {
		return "ubuntu"
	}
	if strings.Contains(strings.ToLower(vmName), "centos") || strings.Contains(strings.ToLower(vmName), "rhel") {
		return "centos"
	}
	return "cloud-user"
}

// parseDomIfAddrIPv4WithMAC parses virsh domifaddr output and returns both IP and MAC address.
// This allows callers to verify the IP belongs to the expected VM by checking the MAC.
func parseDomIfAddrIPv4WithMAC(s string) (ip string, mac string) {
	// virsh domifaddr output example:
	// Name       MAC address          Protocol     Address
	// ----------------------------------------------------------------------------
	// vnet0      52:54:00:6b:3c:86    ipv4         192.168.122.63/24
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "Name") || strings.HasPrefix(l, "-") {
			continue
		}
		parts := strings.Fields(l)
		if len(parts) >= 4 && parts[2] == "ipv4" {
			mac = parts[1]
			addr := parts[3]
			if i := strings.IndexByte(addr, '/'); i > 0 {
				ip = addr[:i]
			} else {
				ip = addr
			}
			return ip, mac
		}
	}
	return "", ""
}

// getVMMAC extracts the MAC address from a VM's domain XML.
// For socket_vmnet VMs, the MAC is in the qemu:commandline section.
// For regular VMs, it's in the interface element.
func (m *VirshManager) getVMMAC(ctx context.Context, vmName string) (string, error) {
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "dumpxml", vmName)
	if err != nil {
		return "", fmt.Errorf("failed to get domain XML: %w", err)
	}

	// Try to find MAC in qemu:commandline (socket_vmnet)
	// Look for: <qemu:arg value="virtio-net-pci,netdev=vnet,mac=52:54:00:xx:xx:xx"/>
	if strings.Contains(out, "qemu:commandline") {
		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if strings.Contains(line, "virtio-net-pci") && strings.Contains(line, "mac=") {
				// Extract MAC from value="...mac=52:54:00:xx:xx:xx..."
				start := strings.Index(line, "mac=")
				if start != -1 {
					start += 4        // skip "mac="
					end := start + 17 // MAC address is 17 chars (xx:xx:xx:xx:xx:xx)
					if end <= len(line) {
						mac := line[start:end]
						// Validate it looks like a MAC
						if strings.Count(mac, ":") == 5 {
							return mac, nil
						}
					}
				}
			}
		}
	}

	// Try to find MAC in regular interface element
	// Look for: <mac address='52:54:00:xx:xx:xx'/>
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<mac address=") {
			// Extract MAC from <mac address='52:54:00:xx:xx:xx'/>
			start := strings.Index(line, "'")
			if start != -1 {
				end := strings.Index(line[start+1:], "'")
				if end != -1 {
					return line[start+1 : start+1+end], nil
				}
			}
			// Try double quotes
			start = strings.Index(line, `"`)
			if start != -1 {
				end := strings.Index(line[start+1:], `"`)
				if end != -1 {
					return line[start+1 : start+1+end], nil
				}
			}
		}
	}

	return "", fmt.Errorf("MAC address not found in domain XML")
}

// lookupIPByMAC looks up an IP address in the system ARP table by MAC address.
// This is used for socket_vmnet where libvirt doesn't track DHCP leases.
// On macOS, this parses the output of `arp -an`.
func lookupIPByMAC(mac string) (string, error) {
	// Normalize MAC to lowercase for comparison
	mac = strings.ToLower(mac)

	// Run arp -an to get the ARP table
	cmd := exec.Command("arp", "-an")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run arp -an: %w", err)
	}

	// Parse ARP output
	// macOS format: ? (192.168.105.2) at 52:54:0:ab:cd:ef on bridge100 ifscope [ethernet]
	// Note: macOS may omit leading zeros in MAC (52:54:0:ab:cd:ef instead of 52:54:00:ab:cd:ef)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract MAC from the line
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue
		}

		// Find the MAC address in the line (after "at")
		var arpMAC string
		for i, p := range parts {
			if p == "at" && i+1 < len(parts) {
				arpMAC = strings.ToLower(parts[i+1])
				break
			}
		}
		if arpMAC == "" {
			continue
		}

		// Normalize the ARP MAC (expand shortened octets like 0 -> 00)
		normalizedArpMAC := normalizeMAC(arpMAC)

		if normalizedArpMAC == mac {
			// Extract IP from (x.x.x.x)
			for _, p := range parts {
				if strings.HasPrefix(p, "(") && strings.HasSuffix(p, ")") {
					ip := p[1 : len(p)-1]
					// Validate it looks like an IP
					if strings.Count(ip, ".") == 3 {
						return ip, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("MAC %s not found in ARP table", mac)
}

// normalizeMAC normalizes a MAC address by ensuring each octet has two digits.
// e.g., "52:54:0:ab:cd:ef" -> "52:54:00:ab:cd:ef"
func normalizeMAC(mac string) string {
	parts := strings.Split(mac, ":")
	if len(parts) != 6 {
		return mac
	}
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		}
	}
	return strings.Join(parts, ":")
}

// --- Domain XML rendering ---

type domainXMLParams struct {
	Name            string
	MemoryMB        int
	VCPUs           int
	DiskPath        string
	CloudInitISO    string // Optional path to cloud-init ISO for networking config
	Network         string // "default", "user", "socket_vmnet", or custom network name
	SocketVMNetPath string // Path to socket_vmnet socket (used when Network="socket_vmnet")
	Emulator        string // Optional custom emulator path (e.g., wrapper script for socket_vmnet)
	BootOrder       []string
	Arch            string // e.g., "x86_64" or "aarch64"
	Machine         string // e.g., "pc-q35-6.2" or "virt"
	DomainType      string // e.g., "kvm" or "qemu"
	MACAddress      string // Optional MAC address for the network interface
}

func renderDomainXML(p domainXMLParams) (string, error) {
	// Set defaults if not provided
	if p.Arch == "" {
		p.Arch = "x86_64"
	}
	if p.Machine == "" {
		if p.Arch == "aarch64" {
			p.Machine = "virt"
		} else {
			p.Machine = "pc-q35-6.2"
		}
	}
	if p.DomainType == "" {
		p.DomainType = "kvm"
	}
	// Generate MAC address if not provided and using socket_vmnet
	if p.MACAddress == "" {
		p.MACAddress = generateMACAddress()
	}
	// Default socket_vmnet path
	if p.Network == "socket_vmnet" && p.SocketVMNetPath == "" {
		p.SocketVMNetPath = "/opt/homebrew/var/run/socket_vmnet"
	}

	// A minimal domain XML; adjust virtio model as needed by your environment.
	// Use conditional sections for architecture-specific elements.
	// For socket_vmnet, we need the qemu namespace for commandline passthrough.
	const tpl = `<?xml version="1.0" encoding="utf-8"?>
<domain type="{{ .DomainType }}"{{ if eq .Network "socket_vmnet" }} xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0"{{ end }}>
  <name>{{ .Name }}</name>
  <memory unit="MiB">{{ .MemoryMB }}</memory>
  <vcpu placement="static">{{ .VCPUs }}</vcpu>
{{- if eq .Arch "aarch64" }}
  <os firmware="efi">
    <type arch="{{ .Arch }}" machine="{{ .Machine }}">hvm</type>
    <boot dev="hd"/>
    <boot dev="cdrom"/>
  </os>
{{- else }}
  <os>
    <type arch="{{ .Arch }}" machine="{{ .Machine }}">hvm</type>
    <boot dev="hd"/>
    <boot dev="cdrom"/>
  </os>
{{- end }}
  <features>
    <acpi/>
{{- if eq .Arch "aarch64" }}
    <gic version="2"/>
{{- else }}
    <apic/>
    <pae/>
{{- end }}
  </features>
{{- if and (eq .Arch "aarch64") (eq .DomainType "qemu") }}
  <cpu mode="custom" match="exact">
    <model fallback="allow">cortex-a72</model>
  </cpu>
{{- else }}
  <cpu mode="host-passthrough"/>
{{- end }}
  <devices>
{{- if .Emulator }}
    <emulator>{{ .Emulator }}</emulator>
{{- end }}
    <disk type="file" device="disk">
      <driver name="qemu" type="qcow2" cache="none"/>
      <source file="{{ .DiskPath }}"/>
      <target dev="vda" bus="virtio"/>
    </disk>
{{- if .CloudInitISO }}
    <disk type="file" device="cdrom">
      <driver name="qemu" type="raw"/>
      <source file="{{ .CloudInitISO }}"/>
      <target dev="sda" bus="scsi"/>
      <readonly/>
    </disk>
    <controller type="scsi" model="virtio-scsi"/>
{{- end }}
    <controller type="pci" model="pcie-root"/>
{{- if eq .Arch "aarch64" }}
    <controller type="usb" model="qemu-xhci"/>
{{- end }}
{{- if eq .Network "socket_vmnet" }}
    <!-- Network configured via qemu:commandline for socket_vmnet -->
{{- else if or (eq .Network "user") (eq .Network "") }}
    <interface type="user">
      <model type="virtio"/>
    </interface>
{{- else }}
    <interface type="network">
      <source network="{{ .Network }}"/>
      <model type="virtio"/>
    </interface>
{{- end }}
    <graphics type="vnc" autoport="yes" listen="0.0.0.0"/>
    <console type="pty"/>
{{- if ne .Arch "aarch64" }}
    <input type="tablet" bus="usb"/>
{{- end }}
    <rng model="virtio">
      <backend model="random">/dev/urandom</backend>
    </rng>
  </devices>
{{- if eq .Network "socket_vmnet" }}
  <qemu:commandline>
    <qemu:arg value="-netdev"/>
    <qemu:arg value="socket,id=vnet,fd=3"/>
    <qemu:arg value="-device"/>
    <qemu:arg value="virtio-net-pci,netdev=vnet,mac={{ .MACAddress }}"/>
  </qemu:commandline>
{{- end }}
</domain>
`
	var b bytes.Buffer
	t := template.Must(template.New("domain").Parse(tpl))
	if err := t.Execute(&b, p); err != nil {
		return "", err
	}
	return b.String(), nil
}

// attachISOToDomainXML is a simple XML string replacement to add a CD-ROM pointing to seed ISO.
// For a production system, consider parsing XML and building a proper DOM.
func (m *VirshManager) attachISOToDomainXML(xmlPath, isoPath string) error {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return err
	}
	xml := string(data)
	needle := "</devices>"
	cdrom := fmt.Sprintf(`
    <disk type="file" device="cdrom">
      <driver name="qemu" type="raw"/>
      <source file="%s"/>
      <target dev="sda" bus="sata"/>
      <readonly/>
    </disk>`, isoPath)
	if strings.Contains(xml, cdrom) {
		// already attached
		return nil
	}
	xml = strings.Replace(xml, needle, cdrom+"\n  "+needle, 1)
	return os.WriteFile(xmlPath, []byte(xml), 0o644)
}

// buildCloudInitSeed creates a NoCloud seed ISO with a single user and SSH key.
// Requires cloud-localds (cloud-image-utils) on the host if implemented via external tool.
// This implementation writes user-data/meta-data and attempts to use genisoimage or mkisofs.
func (m *VirshManager) buildCloudInitSeed(ctx context.Context, vmName, username, publicKey, outISO string) error {
	jobDir := filepath.Dir(outISO)
	userData := fmt.Sprintf(`#cloud-config
users:
  - name: %s
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: users, admin, sudo
    shell: /bin/bash
    ssh_authorized_keys:
      - %s
`, username, publicKey)

	metaData := fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, vmName, vmName)

	userDataPath := filepath.Join(jobDir, "user-data")
	metaDataPath := filepath.Join(jobDir, "meta-data")
	if err := os.WriteFile(userDataPath, []byte(userData), 0o644); err != nil {
		return fmt.Errorf("write user-data: %w", err)
	}
	if err := os.WriteFile(metaDataPath, []byte(metaData), 0o644); err != nil {
		return fmt.Errorf("write meta-data: %w", err)
	}

	// Try cloud-localds if available
	if hasBin("cloud-localds") {
		if _, err := m.run(ctx, "cloud-localds", outISO, userDataPath, metaDataPath); err == nil {
			return nil
		}
	}

	// Fallback to genisoimage/mkisofs
	if hasBin("genisoimage") {
		// genisoimage -output seed.iso -volid cidata -joliet -rock user-data meta-data
		_, err := m.run(ctx, "genisoimage", "-output", outISO, "-volid", "cidata", "-joliet", "-rock", userDataPath, metaDataPath)
		return err
	}
	if hasBin("mkisofs") {
		_, err := m.run(ctx, "mkisofs", "-output", outISO, "-V", "cidata", "-J", "-R", userDataPath, metaDataPath)
		return err
	}

	return fmt.Errorf("cloud-init seed build tools not found: need cloud-localds or genisoimage/mkisofs")
}

// buildCloudInitSeedForClone creates a cloud-init ISO for a cloned VM.
// The key purpose is to provide a NEW instance-id that differs from what's stored
// on the cloned disk. This forces cloud-init to re-run its initialization,
// including network configuration for the clone's new MAC address.
//
// If SSHCAPubKey is configured, this function also:
// - Creates a 'sandbox' user with sudo access for managed SSH credentials
// - Injects the SSH CA public key and configures sshd to trust it
func (m *VirshManager) buildCloudInitSeedForClone(ctx context.Context, vmName, outISO string) error {
	jobDir := filepath.Dir(outISO)

	// Build cloud-init user-data
	var userDataBuilder strings.Builder
	userDataBuilder.WriteString(`#cloud-config
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
`)

	// If SSH CA is configured, add sandbox user and SSH CA trust
	if m.cfg.SSHCAPubKey != "" {
		userDataBuilder.WriteString(`
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
      `)
		userDataBuilder.WriteString(m.cfg.SSHCAPubKey)
		userDataBuilder.WriteString(`
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
`)
	}

	userData := userDataBuilder.String()

	// Use a unique instance-id based on the VM name
	// This is the critical part: cloud-init checks if instance-id has changed
	// If it has, cloud-init re-runs initialization including network setup
	metaData := fmt.Sprintf(`instance-id: %s
local-hostname: %s
`, vmName, vmName)

	userDataPath := filepath.Join(jobDir, "user-data")
	metaDataPath := filepath.Join(jobDir, "meta-data")
	if err := os.WriteFile(userDataPath, []byte(userData), 0o644); err != nil {
		return fmt.Errorf("write user-data: %w", err)
	}
	if err := os.WriteFile(metaDataPath, []byte(metaData), 0o644); err != nil {
		return fmt.Errorf("write meta-data: %w", err)
	}

	// Try cloud-localds if available
	if hasBin("cloud-localds") {
		if _, err := m.run(ctx, "cloud-localds", outISO, userDataPath, metaDataPath); err == nil {
			// Verify the ISO was actually created
			if _, statErr := os.Stat(outISO); statErr == nil {
				return nil
			} else {
				log.Printf("WARNING: cloud-localds succeeded but ISO not found at %s: %v", outISO, statErr)
			}
		} else {
			log.Printf("WARNING: cloud-localds failed: %v, trying fallback tools", err)
		}
	}

	// Fallback to genisoimage/mkisofs
	if hasBin("genisoimage") {
		_, err := m.run(ctx, "genisoimage", "-output", outISO, "-volid", "cidata", "-joliet", "-rock", userDataPath, metaDataPath)
		if err != nil {
			log.Printf("WARNING: genisoimage failed: %v", err)
			return err
		}
		// Verify the ISO was actually created
		if _, statErr := os.Stat(outISO); statErr != nil {
			return fmt.Errorf("genisoimage succeeded but ISO not found at %s: %w", outISO, statErr)
		}
		return nil
	}
	if hasBin("mkisofs") {
		_, err := m.run(ctx, "mkisofs", "-output", outISO, "-V", "cidata", "-J", "-R", userDataPath, metaDataPath)
		if err != nil {
			log.Printf("WARNING: mkisofs failed: %v", err)
			return err
		}
		// Verify the ISO was actually created
		if _, statErr := os.Stat(outISO); statErr != nil {
			return fmt.Errorf("mkisofs succeeded but ISO not found at %s: %w", outISO, statErr)
		}
		return nil
	}

	return fmt.Errorf("cloud-init seed build tools not found: need cloud-localds or genisoimage/mkisofs")
}

func hasBin(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ValidateSourceVM performs pre-flight checks on a source VM before cloning.
// It checks: VM state, network interface, MAC address, and IP address.
func (m *VirshManager) ValidateSourceVM(ctx context.Context, vmName string) (*VMValidationResult, error) {
	if vmName == "" {
		return nil, fmt.Errorf("vmName is required")
	}

	result := &VMValidationResult{
		VMName:   vmName,
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

	// Check if VM has a network interface with MAC address
	mac, macErr := m.GetVMMAC(ctx, vmName)
	if macErr != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Could not get MAC address from VM definition: %v", macErr))
		result.Warnings = append(result.Warnings,
			"The source VM may not have a network interface configured properly")
	} else {
		result.MACAddress = mac
		result.HasNetwork = true
	}

	// Check if VM has an IP address (only if running)
	switch state {
	case VMStateRunning:
		virsh := m.binPath("virsh", m.cfg.VirshPath)
		out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domifaddr", vmName, "--source", "lease")
		if err == nil {
			ip, mac := parseDomIfAddrIPv4WithMAC(out)
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

		// Check network interface statistics if we have a MAC
		if result.MACAddress != "" {
			stats, statsErr := m.getVMNetworkStats(ctx, vmName)
			if statsErr == nil {
				if stats.txPackets == 0 && stats.rxPackets == 0 {
					result.Warnings = append(result.Warnings,
						"Source VM network interface shows zero TX/RX packets - network may not be functioning")
				} else if stats.txPackets == 0 {
					result.Warnings = append(result.Warnings,
						"Source VM network interface shows zero TX packets - VM may not be sending network traffic")
				}
			}
		}
	case VMStateShutOff:
		// VM is shut off - this is fine for cloning, but warn that we can't verify network
		result.Warnings = append(result.Warnings,
			"Source VM is shut off - cannot verify network configuration (IP/DHCP)")
		result.Warnings = append(result.Warnings,
			"Consider starting the source VM to verify it can obtain an IP before cloning")
	default:
		// VM is in an unexpected state
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Source VM is in state '%s' - expected 'running' or 'shut off'", state))
	}

	// Check if VM has cloud-init CDROM (helpful for diagnostics)
	hasCloudInit := m.vmHasCloudInitCDROM(ctx, vmName)
	if !hasCloudInit {
		result.Warnings = append(result.Warnings,
			"Source VM does not appear to have a cloud-init CDROM attached")
		result.Warnings = append(result.Warnings,
			"Cloned sandboxes will still get their own cloud-init ISO for network config")
	}

	return result, nil
}

// vmNetworkStats holds network interface statistics
type vmNetworkStats struct {
	rxBytes   int64
	rxPackets int64
	txBytes   int64
	txPackets int64
}

// getVMNetworkStats returns network interface statistics for a VM
func (m *VirshManager) getVMNetworkStats(ctx context.Context, vmName string) (*vmNetworkStats, error) {
	virsh := m.binPath("virsh", m.cfg.VirshPath)

	// Get interface name first
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domiflist", vmName)
	if err != nil {
		return nil, fmt.Errorf("get interface list: %w", err)
	}

	// Parse domiflist to get interface name
	var ifaceName string
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Interface") || strings.HasPrefix(line, "-") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 && fields[0] != "-" {
			ifaceName = fields[0]
			break
		}
	}

	if ifaceName == "" || ifaceName == "-" {
		// Interface name is "-" for some network types, try to get stats anyway
		return nil, fmt.Errorf("no named interface found")
	}

	// Get interface stats
	out, err = m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domifstat", vmName, ifaceName)
	if err != nil {
		return nil, fmt.Errorf("get interface stats: %w", err)
	}

	stats := &vmNetworkStats{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			var val int64
			_, _ = fmt.Sscanf(fields[1], "%d", &val)
			switch {
			case strings.Contains(fields[0], "rx_bytes"):
				stats.rxBytes = val
			case strings.Contains(fields[0], "rx_packets"):
				stats.rxPackets = val
			case strings.Contains(fields[0], "tx_bytes"):
				stats.txBytes = val
			case strings.Contains(fields[0], "tx_packets"):
				stats.txPackets = val
			}
		}
	}

	return stats, nil
}

// vmHasCloudInitCDROM checks if a VM has a cloud-init CDROM attached
func (m *VirshManager) vmHasCloudInitCDROM(ctx context.Context, vmName string) bool {
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "domblklist", vmName, "--details")
	if err != nil {
		return false
	}

	// Look for cdrom device with cloud-init related path
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[1] == "cdrom" {
			path := strings.ToLower(fields[3])
			if strings.Contains(path, "cloud") || strings.Contains(path, "seed") ||
				strings.Contains(path, "cidata") || strings.Contains(path, "init") {
				return true
			}
		}
	}
	return false
}

// CheckHostResources validates that the host has sufficient resources for a new sandbox.
func (m *VirshManager) CheckHostResources(ctx context.Context, requiredCPUs, requiredMemoryMB int) (*ResourceCheckResult, error) {
	result := &ResourceCheckResult{
		Valid:            true,
		RequiredCPUs:     requiredCPUs,
		RequiredMemoryMB: requiredMemoryMB,
		Warnings:         []string{},
		Errors:           []string{},
	}

	// Check available resources using virsh nodeinfo
	info, err := m.getHostInfo(ctx)
	if err != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Could not check host resources: %v", err))
	} else {
		result.TotalMemoryMB = info.totalMB
		result.AvailableMemoryMB = info.availableMB
		result.AvailableCPUs = info.cpus

		// Check if we have enough memory
		if int64(requiredMemoryMB) > info.availableMB {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Insufficient memory: need %d MB but only %d MB available",
					requiredMemoryMB, info.availableMB))
		} else if float64(requiredMemoryMB) > float64(info.availableMB)*0.8 {
			// Warn if using more than 80% of available memory
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Low memory warning: requesting %d MB of %d MB available (%.1f%%)",
					requiredMemoryMB, info.availableMB,
					float64(requiredMemoryMB)/float64(info.availableMB)*100))
		}

		// Check if we have enough CPUs
		if requiredCPUs > info.cpus {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Insufficient CPUs: need %d but only %d available",
					requiredCPUs, info.cpus))
		}
	}

	// Check available disk space in work directory
	diskInfo, err := m.getWorkDirDiskSpace(ctx)
	if err != nil {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Could not check disk space: %v", err))
	} else {
		result.AvailableDiskMB = diskInfo.availableMB

		// Warn if disk space is low (less than 10GB)
		if diskInfo.availableMB < 10*1024 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Low disk space warning: only %d MB available in work directory (%s)",
					diskInfo.availableMB, m.cfg.WorkDir))
		}

		// Error if disk space is critically low (less than 1GB)
		if diskInfo.availableMB < 1024 {
			result.Valid = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Insufficient disk space: only %d MB available in work directory (%s)",
					diskInfo.availableMB, m.cfg.WorkDir))
		}
	}

	return result, nil
}

// hostInfo holds host resource information
type hostInfo struct {
	cpus        int
	totalMB     int64
	availableMB int64
}

// getHostInfo returns available and total resources on the host
func (m *VirshManager) getHostInfo(ctx context.Context) (*hostInfo, error) {
	virsh := m.binPath("virsh", m.cfg.VirshPath)
	info := &hostInfo{}

	// Get CPU info from nodeinfo
	out, err := m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "nodeinfo")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, "CPU(s):") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					_, _ = fmt.Sscanf(fields[1], "%d", &info.cpus)
				}
			}
			if strings.HasPrefix(line, "Memory size:") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					var val int64
					_, _ = fmt.Sscanf(fields[2], "%d", &val)
					info.totalMB = val / 1024
					info.availableMB = info.totalMB / 2 // Fallback estimate
				}
			}
		}
	}

	// Try virsh nodememstats for more accurate available memory
	out, err = m.run(ctx, virsh, "--connect", m.cfg.LibvirtURI, "nodememstats")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				var val int64
				_, _ = fmt.Sscanf(fields[len(fields)-2], "%d", &val)
				if strings.Contains(fields[0], "free") {
					info.availableMB = val / 1024
				}
			}
		}
	}

	if info.totalMB > 0 {
		return info, nil
	}

	// Fallback: use free command (Linux)
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, "free", "-m")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		for _, line := range strings.Split(stdout.String(), "\n") {
			if strings.HasPrefix(line, "Mem:") {
				fields := strings.Fields(line)
				if len(fields) >= 7 {
					var total, available int64
					_, _ = fmt.Sscanf(fields[1], "%d", &total)
					_, _ = fmt.Sscanf(fields[6], "%d", &available)
					info.totalMB = total
					info.availableMB = available
					return info, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("could not determine host memory")
}

// diskSpaceInfo holds disk space information
type diskSpaceInfo struct {
	availableMB int64
}

// getWorkDirDiskSpace returns available disk space in the work directory
func (m *VirshManager) getWorkDirDiskSpace(ctx context.Context) (*diskSpaceInfo, error) {
	workDir := m.cfg.WorkDir
	if workDir == "" {
		workDir = "/var/lib/libvirt/images/sandboxes"
	}

	// Use df command to get disk space
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, "df", "-m", workDir)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("df command failed: %w", err)
	}

	// Parse df output (second line, 4th column for available)
	lines := strings.Split(stdout.String(), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			var available int64
			_, _ = fmt.Sscanf(fields[3], "%d", &available)
			return &diskSpaceInfo{
				availableMB: available,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not parse df output")
}
