package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the sandbox host daemon.
type Config struct {
	// HostID is a persistent identifier for this host. Generated on first run.
	HostID string `yaml:"host_id"`

	// ControlPlane configures the connection to the control plane.
	ControlPlane ControlPlaneConfig `yaml:"control_plane"`

	// MicroVM configures QEMU microVM defaults.
	MicroVM MicroVMConfig `yaml:"microvm"`

	// Network configures bridge and TAP networking.
	Network NetworkConfig `yaml:"network"`

	// Image configures base image storage.
	Image ImageConfig `yaml:"image"`

	// SSH configures SSH CA and key management.
	SSH SSHConfig `yaml:"ssh"`

	// Libvirt configures libvirt access for source VM operations.
	Libvirt LibvirtConfig `yaml:"libvirt"`

	// State configures local state storage.
	State StateConfig `yaml:"state"`

	// Janitor configures TTL enforcement.
	Janitor JanitorConfig `yaml:"janitor"`
}

// ControlPlaneConfig configures the gRPC connection to the control plane.
type ControlPlaneConfig struct {
	// Address is the control plane gRPC endpoint (host:port).
	Address string `yaml:"address"`

	// TLS configures mTLS for the connection.
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`

	// Insecure disables TLS (for development).
	Insecure bool `yaml:"insecure"`
}

// MicroVMConfig configures QEMU microVM defaults.
type MicroVMConfig struct {
	// QEMUBinary is the path to qemu-system-x86_64.
	QEMUBinary string `yaml:"qemu_binary"`

	// WorkDir is the directory for sandbox runtime data (overlays, PID files).
	WorkDir string `yaml:"work_dir"`

	// DefaultVCPUs is the default number of vCPUs per sandbox.
	DefaultVCPUs int `yaml:"default_vcpus"`

	// DefaultMemoryMB is the default memory per sandbox in MB.
	DefaultMemoryMB int `yaml:"default_memory_mb"`

	// CommandTimeout is the default command execution timeout.
	CommandTimeout time.Duration `yaml:"command_timeout"`

	// IPDiscoveryTimeout is how long to wait for IP discovery.
	IPDiscoveryTimeout time.Duration `yaml:"ip_discovery_timeout"`
}

// NetworkConfig configures networking for sandboxes.
type NetworkConfig struct {
	// DefaultBridge is the default bridge for sandboxes.
	DefaultBridge string `yaml:"default_bridge"`

	// BridgeMap maps libvirt network names to local bridge names.
	BridgeMap map[string]string `yaml:"bridge_map"`

	// DHCPMode determines IP discovery strategy: "libvirt", "arp", or "dnsmasq".
	DHCPMode string `yaml:"dhcp_mode"`
}

// ImageConfig configures base image storage and management.
type ImageConfig struct {
	// BaseDir is the directory containing base QCOW2 images.
	BaseDir string `yaml:"base_dir"`
}

// SSHConfig configures SSH CA and key management.
type SSHConfig struct {
	// CAKeyPath is the path to the SSH CA private key.
	CAKeyPath string `yaml:"ca_key_path"`

	// CAPubKeyPath is the path to the SSH CA public key.
	CAPubKeyPath string `yaml:"ca_pub_key_path"`

	// KeyDir is the directory for ephemeral SSH keys.
	KeyDir string `yaml:"key_dir"`

	// CertTTL is the lifetime of issued SSH certificates.
	CertTTL time.Duration `yaml:"cert_ttl"`

	// DefaultUser is the default SSH user for sandbox access.
	DefaultUser string `yaml:"default_user"`

	// ProxyJump is an optional SSH proxy jump host.
	ProxyJump string `yaml:"proxy_jump"`
}

// LibvirtConfig configures libvirt access for source VM operations.
type LibvirtConfig struct {
	// URI is the libvirt connection URI (e.g., "qemu:///system").
	URI string `yaml:"uri"`

	// Network is the default libvirt network name.
	Network string `yaml:"network"`
}

// StateConfig configures local state storage.
type StateConfig struct {
	// DBPath is the path to the SQLite database file.
	DBPath string `yaml:"db_path"`
}

// JanitorConfig configures TTL enforcement.
type JanitorConfig struct {
	// Interval is how often the janitor runs.
	Interval time.Duration `yaml:"interval"`

	// DefaultTTL is the default sandbox TTL if none is specified.
	DefaultTTL time.Duration `yaml:"default_ttl"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	fluidDir := filepath.Join(home, ".fluid")

	return Config{
		ControlPlane: ControlPlaneConfig{
			Address:  "localhost:9090",
			Insecure: true,
		},
		MicroVM: MicroVMConfig{
			QEMUBinary:         "qemu-system-x86_64",
			WorkDir:            "/var/lib/fluid/sandboxes",
			DefaultVCPUs:       2,
			DefaultMemoryMB:    2048,
			CommandTimeout:     5 * time.Minute,
			IPDiscoveryTimeout: 2 * time.Minute,
		},
		Network: NetworkConfig{
			DefaultBridge: "virbr0",
			BridgeMap: map[string]string{
				"default": "virbr0",
			},
			DHCPMode: "arp",
		},
		Image: ImageConfig{
			BaseDir: "/var/lib/fluid/images",
		},
		SSH: SSHConfig{
			CAKeyPath:    filepath.Join(fluidDir, "ssh_ca"),
			CAPubKeyPath: filepath.Join(fluidDir, "ssh_ca.pub"),
			KeyDir:       filepath.Join(fluidDir, "keys"),
			CertTTL:      30 * time.Minute,
			DefaultUser:  "sandbox",
		},
		Libvirt: LibvirtConfig{
			URI:     "qemu:///system",
			Network: "default",
		},
		State: StateConfig{
			DBPath: filepath.Join(fluidDir, "sandbox-host.db"),
		},
		Janitor: JanitorConfig{
			Interval:   1 * time.Minute,
			DefaultTTL: 24 * time.Hour,
		},
	}
}

// Load reads configuration from a YAML file, falling back to defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to a YAML file.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0o644)
}
