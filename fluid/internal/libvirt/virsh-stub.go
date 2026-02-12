//go:build !libvirt

package libvirt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aspectrr/fluid.sh/fluid/internal/provider"
)

// ErrLibvirtNotAvailable is returned by all stub methods when libvirt support is not compiled in.
var ErrLibvirtNotAvailable = errors.New("libvirt support not available: rebuild with -tags libvirt")

// Manager is the provider-neutral VM manager interface.
type Manager = provider.Manager

// Type aliases to provider types - keeps all existing imports working.
type (
	VMValidationResult  = provider.VMValidationResult
	ResourceCheckResult = provider.ResourceCheckResult
	VMState             = provider.VMState
	DomainRef           = provider.VMRef
	SnapshotRef         = provider.SnapshotRef
	FSComparePlan       = provider.FSComparePlan
)

// Forward VMState constants.
const (
	VMStateRunning   = provider.VMStateRunning
	VMStatePaused    = provider.VMStatePaused
	VMStateShutOff   = provider.VMStateShutOff
	VMStateCrashed   = provider.VMStateCrashed
	VMStateSuspended = provider.VMStateSuspended
	VMStateUnknown   = provider.VMStateUnknown
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
	SSHCAPubKey string

	// SSH ProxyJump host for reaching VMs on an isolated network.
	SSHProxyJump string

	// Optional explicit paths to binaries; if empty these are looked up in PATH.
	VirshPath         string
	QemuImgPath       string
	VirtCustomizePath string
	QemuNbdPath       string

	// Socket VMNet configuration (macOS only)
	SocketVMNetWrapper string // e.g., /path/to/qemu-socket-vmnet-wrapper.sh

	// Domain defaults
	DefaultVCPUs    int
	DefaultMemoryMB int
}

// VirshManager implements Manager using virsh/qemu-img/qemu-nbd/virt-customize and simple domain XML.
// This is a stub implementation that returns errors when libvirt is not available.
type VirshManager struct {
	cfg    Config
	logger *slog.Logger
}

// ConfigFromEnv returns a Config populated from environment variables.
func ConfigFromEnv() Config {
	return Config{
		LibvirtURI:         os.Getenv("LIBVIRT_URI"),
		BaseImageDir:       os.Getenv("BASE_IMAGE_DIR"),
		WorkDir:            os.Getenv("SANDBOX_WORKDIR"),
		DefaultNetwork:     os.Getenv("LIBVIRT_NETWORK"),
		SSHKeyInjectMethod: os.Getenv("SSH_KEY_INJECT_METHOD"),
	}
}

// NewVirshManager creates a new VirshManager with the provided config.
// Note: This stub implementation will return errors for all operations.
func NewVirshManager(cfg Config, logger *slog.Logger) *VirshManager {
	return &VirshManager{cfg: cfg, logger: logger}
}

// NewFromEnv builds a Config from environment variables and returns a manager.
// Note: This stub implementation will return errors for all operations.
func NewFromEnv() *VirshManager {
	cfg := Config{
		DefaultVCPUs:    2,
		DefaultMemoryMB: 2048,
	}
	return NewVirshManager(cfg, nil)
}

// CloneVM is a stub that returns an error when libvirt is not available.
func (m *VirshManager) CloneVM(ctx context.Context, baseImage, newVMName string, cpu, memoryMB int, network string) (DomainRef, error) {
	return DomainRef{}, ErrLibvirtNotAvailable
}

// CloneFromVM is a stub that returns an error when libvirt is not available.
func (m *VirshManager) CloneFromVM(ctx context.Context, sourceVMName, newVMName string, cpu, memoryMB int, network string) (DomainRef, error) {
	return DomainRef{}, ErrLibvirtNotAvailable
}

// InjectSSHKey is a stub that returns an error when libvirt is not available.
func (m *VirshManager) InjectSSHKey(ctx context.Context, sandboxName, username, publicKey string) error {
	return ErrLibvirtNotAvailable
}

// StartVM is a stub that returns an error when libvirt is not available.
func (m *VirshManager) StartVM(ctx context.Context, vmName string) error {
	return ErrLibvirtNotAvailable
}

// StopVM is a stub that returns an error when libvirt is not available.
func (m *VirshManager) StopVM(ctx context.Context, vmName string, force bool) error {
	return ErrLibvirtNotAvailable
}

// DestroyVM is a stub that returns an error when libvirt is not available.
func (m *VirshManager) DestroyVM(ctx context.Context, vmName string) error {
	return ErrLibvirtNotAvailable
}

// CreateSnapshot is a stub that returns an error when libvirt is not available.
func (m *VirshManager) CreateSnapshot(ctx context.Context, vmName, snapshotName string, external bool) (SnapshotRef, error) {
	return SnapshotRef{}, ErrLibvirtNotAvailable
}

// DiffSnapshot is a stub that returns an error when libvirt is not available.
func (m *VirshManager) DiffSnapshot(ctx context.Context, vmName, fromSnapshot, toSnapshot string) (*FSComparePlan, error) {
	return nil, ErrLibvirtNotAvailable
}

// GetIPAddress is a stub that returns an error when libvirt is not available.
func (m *VirshManager) GetIPAddress(ctx context.Context, vmName string, timeout time.Duration) (string, string, error) {
	return "", "", ErrLibvirtNotAvailable
}

// GetVMState is a stub that returns an error when libvirt is not available.
func (m *VirshManager) GetVMState(ctx context.Context, vmName string) (VMState, error) {
	return VMStateUnknown, ErrLibvirtNotAvailable
}

// GetVMMAC is a stub that returns an error when libvirt is not available.
func (m *VirshManager) GetVMMAC(ctx context.Context, vmName string) (string, error) {
	return "", ErrLibvirtNotAvailable
}

// ReleaseDHCPLease is a stub that returns an error when libvirt is not available.
func (m *VirshManager) ReleaseDHCPLease(ctx context.Context, network, mac string) error {
	return ErrLibvirtNotAvailable
}

// ValidateSourceVM is a stub that returns an error when libvirt is not available.
func (m *VirshManager) ValidateSourceVM(ctx context.Context, vmName string) (*VMValidationResult, error) {
	return nil, ErrLibvirtNotAvailable
}

// CheckHostResources validates that the host has sufficient resources for a new sandbox.
// Returns a ResourceCheckResult with available resources and any warnings.
func (m *VirshManager) CheckHostResources(ctx context.Context, requiredCPUs, requiredMemoryMB int) (*ResourceCheckResult, error) {
	return nil, fmt.Errorf("CheckHostResources not implemented in stub")
}
