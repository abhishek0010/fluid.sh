package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for fluid-remote API.
type Config struct {
	API       APIConfig       `yaml:"api"`
	Database  DatabaseConfig  `yaml:"database"`
	Libvirt   LibvirtConfig   `yaml:"libvirt"`
	VM        VMConfig        `yaml:"vm"`
	SSH       SSHConfig       `yaml:"ssh"`
	Ansible   AnsibleConfig   `yaml:"ansible"`
	Logging   LoggingConfig   `yaml:"logging"`
	Telemetry TelemetryConfig `yaml:"telemetry"`
	Janitor   JanitorConfig   `yaml:"janitor"` // Background cleanup of expired sandboxes
	Hosts     []HostConfig    `yaml:"hosts"`   // Remote libvirt hosts for multi-host VM listing
}

// APIConfig holds HTTP server settings.
type APIConfig struct {
	Addr            string        `yaml:"addr"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	URL             string        `yaml:"url"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	AutoMigrate     bool          `yaml:"auto_migrate"`
}

// TelemetryConfig holds telemetry settings.
type TelemetryConfig struct {
	EnableAnonymousUsage bool   `yaml:"enable_anonymous_usage"`
	APIKey               string `yaml:"api_key"`
	Endpoint             string `yaml:"endpoint"`
}

// LibvirtConfig holds libvirt/KVM settings.
type LibvirtConfig struct {
	URI                string `yaml:"uri"`
	Network            string `yaml:"network"`
	BaseImageDir       string `yaml:"base_image_dir"`
	WorkDir            string `yaml:"work_dir"`
	SSHKeyInjectMethod string `yaml:"ssh_key_inject_method"`
	SocketVMNetWrapper string `yaml:"socket_vmnet_wrapper"`
}

// VMConfig holds VM default settings.
type VMConfig struct {
	DefaultVCPUs       int           `yaml:"default_vcpus"`
	DefaultMemoryMB    int           `yaml:"default_memory_mb"`
	CommandTimeout     time.Duration `yaml:"command_timeout"`
	IPDiscoveryTimeout time.Duration `yaml:"ip_discovery_timeout"`
}

// SSHConfig holds SSH CA and key management settings.
type SSHConfig struct {
	ProxyJump   string        `yaml:"proxy_jump"`
	CAKeyPath   string        `yaml:"ca_key_path"`
	CAPubPath   string        `yaml:"ca_pub_path"`
	KeyDir      string        `yaml:"key_dir"`
	CertTTL     time.Duration `yaml:"cert_ttl"`
	MaxTTL      time.Duration `yaml:"max_ttl"`
	WorkDir     string        `yaml:"work_dir"`
	DefaultUser string        `yaml:"default_user"`
}

// AnsibleConfig holds Ansible runner settings.
type AnsibleConfig struct {
	InventoryPath    string   `yaml:"inventory_path"`
	PlaybooksDir     string   `yaml:"playbooks_dir"`
	Image            string   `yaml:"image"`
	AllowedPlaybooks []string `yaml:"allowed_playbooks"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// JanitorConfig holds settings for the background sandbox cleanup service.
type JanitorConfig struct {
	// Enabled controls whether the janitor runs. Default: true
	Enabled bool `yaml:"enabled"`
	// Interval is how often the janitor checks for expired sandboxes. Default: 1m
	Interval time.Duration `yaml:"interval"`
	// DefaultTTL is the default TTL for sandboxes that don't specify one. Default: 0 (no auto-cleanup)
	// Sandboxes with TTLSeconds set will use their own TTL instead.
	DefaultTTL time.Duration `yaml:"default_ttl"`
}

// HostConfig represents a remote libvirt host for multi-host VM management.
// Authentication uses system SSH config (~/.ssh/config and ssh-agent).
type HostConfig struct {
	Name         string        `yaml:"name"`          // Display name (e.g., "kvm-01")
	Address      string        `yaml:"address"`       // IP or hostname
	SSHUser      string        `yaml:"ssh_user"`      // SSH user (default: root)
	SSHPort      int           `yaml:"ssh_port"`      // SSH port (default: 22)
	QueryTimeout time.Duration `yaml:"query_timeout"` // Per-host query timeout (default: 30s)
}

// DefaultConfig returns config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		API: APIConfig{
			Addr:            ":8080",
			ReadTimeout:     60 * time.Second,
			WriteTimeout:    120 * time.Second,
			IdleTimeout:     120 * time.Second,
			ShutdownTimeout: 20 * time.Second,
		},
		Database: DatabaseConfig{
			URL:             "postgresql://virsh_sandbox:virsh_sandbox@postgres:5432/virsh_sandbox",
			MaxOpenConns:    16,
			MaxIdleConns:    8,
			ConnMaxLifetime: time.Hour,
			AutoMigrate:     true,
		},
		Telemetry: TelemetryConfig{
			EnableAnonymousUsage: true,
		},
		Libvirt: LibvirtConfig{
			URI:                "qemu:///system",
			Network:            "default",
			BaseImageDir:       "/var/lib/libvirt/images/base",
			WorkDir:            "/var/lib/libvirt/images/jobs",
			SSHKeyInjectMethod: "virt-customize",
		},
		VM: VMConfig{
			DefaultVCPUs:       2,
			DefaultMemoryMB:    2048,
			CommandTimeout:     10 * time.Minute,
			IPDiscoveryTimeout: 2 * time.Minute,
		},
		SSH: SSHConfig{
			CAKeyPath:   "/etc/fluid-remote/ssh_ca",
			CAPubPath:   "/etc/fluid-remote/ssh_ca.pub",
			KeyDir:      "/tmp/sandbox-keys",
			CertTTL:     5 * time.Minute,
			MaxTTL:      10 * time.Minute,
			WorkDir:     "/tmp/sshca",
			DefaultUser: "sandbox",
		},
		Ansible: AnsibleConfig{
			InventoryPath:    "./.ansible/inventory",
			PlaybooksDir:     "./.ansible/playbooks",
			Image:            "ansible-sandbox",
			AllowedPlaybooks: []string{"ping.yml"},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Janitor: JanitorConfig{
			Enabled:    true,
			Interval:   1 * time.Minute,
			DefaultTTL: 0, // No auto-cleanup by default; sandboxes must set TTLSeconds
		},
	}
}

// Load reads config from a YAML file. If the file doesn't exist, returns default config.
// Environment variables can override config values - they take precedence.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file - use defaults
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// LoadWithEnvOverride loads config from YAML and allows env vars to override.
// Env vars use the pattern: VIRSH_SANDBOX_<SECTION>_<KEY> (uppercase, underscores).
func LoadWithEnvOverride(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to config.
// This allows backward compatibility with existing env var usage.
func applyEnvOverrides(cfg *Config) {
	// API
	if v := os.Getenv("API_HTTP_ADDR"); v != "" {
		cfg.API.Addr = v
	}
	if v := os.Getenv("API_SHUTDOWN_TIMEOUT_SEC"); v != "" {
		if d := parseDuration(v); d > 0 {
			cfg.API.ShutdownTimeout = d
		}
	}

	// Database
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}

	// Telemetry
	if v := os.Getenv("ENABLE_ANONYMOUS_USAGE"); v != "" {
		cfg.Telemetry.EnableAnonymousUsage = v == "true"
	}
	if v := os.Getenv("TELEMETRY_API_KEY"); v != "" {
		cfg.Telemetry.APIKey = v
	}
	if v := os.Getenv("TELEMETRY_ENDPOINT"); v != "" {
		cfg.Telemetry.Endpoint = v
	}

	// Libvirt
	if v := os.Getenv("LIBVIRT_URI"); v != "" {
		cfg.Libvirt.URI = v
	}
	if v := os.Getenv("LIBVIRT_NETWORK"); v != "" {
		cfg.Libvirt.Network = v
	}
	if v := os.Getenv("BASE_IMAGE_DIR"); v != "" {
		cfg.Libvirt.BaseImageDir = v
	}
	if v := os.Getenv("SANDBOX_WORKDIR"); v != "" {
		cfg.Libvirt.WorkDir = v
	}
	if v := os.Getenv("SSH_KEY_INJECT_METHOD"); v != "" {
		cfg.Libvirt.SSHKeyInjectMethod = v
	}
	if v := os.Getenv("SOCKET_VMNET_WRAPPER"); v != "" {
		cfg.Libvirt.SocketVMNetWrapper = v
	}

	// VM
	if v := os.Getenv("DEFAULT_VCPUS"); v != "" {
		if i := atoi(v); i > 0 {
			cfg.VM.DefaultVCPUs = i
		}
	}
	if v := os.Getenv("DEFAULT_MEMORY_MB"); v != "" {
		if i := atoi(v); i > 0 {
			cfg.VM.DefaultMemoryMB = i
		}
	}
	if v := os.Getenv("COMMAND_TIMEOUT_SEC"); v != "" {
		if d := parseDuration(v); d > 0 {
			cfg.VM.CommandTimeout = d
		}
	}
	if v := os.Getenv("IP_DISCOVERY_TIMEOUT_SEC"); v != "" {
		if d := parseDuration(v); d > 0 {
			cfg.VM.IPDiscoveryTimeout = d
		}
	}

	// SSH
	if v := os.Getenv("SSH_PROXY_JUMP"); v != "" {
		cfg.SSH.ProxyJump = v
	}
	if v := os.Getenv("SSH_CA_KEY_PATH"); v != "" {
		cfg.SSH.CAKeyPath = v
	}
	if v := os.Getenv("SSH_CA_PUB_KEY_PATH"); v != "" {
		cfg.SSH.CAPubPath = v
	}
	if v := os.Getenv("SSH_KEY_DIR"); v != "" {
		cfg.SSH.KeyDir = v
	}
	if v := os.Getenv("SSH_CERT_TTL_SEC"); v != "" {
		if d := parseDuration(v); d > 0 {
			cfg.SSH.CertTTL = d
		}
	}

	// Ansible
	if v := os.Getenv("ANSIBLE_INVENTORY_PATH"); v != "" {
		cfg.Ansible.InventoryPath = v
	}
	if v := os.Getenv("ANSIBLE_PLAYBOOKS_DIR"); v != "" {
		cfg.Ansible.PlaybooksDir = v
	}
	if v := os.Getenv("ANSIBLE_IMAGE"); v != "" {
		cfg.Ansible.Image = v
	}

	// Logging
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}

	// Janitor
	if v := os.Getenv("JANITOR_ENABLED"); v != "" {
		cfg.Janitor.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("JANITOR_INTERVAL"); v != "" {
		if d := parseDuration(v); d > 0 {
			cfg.Janitor.Interval = d
		}
	}
	if v := os.Getenv("SANDBOX_DEFAULT_TTL"); v != "" {
		if d := parseDuration(v); d > 0 {
			cfg.Janitor.DefaultTTL = d
		}
	}
}

func atoi(s string) int {
	var i int
	_, _ = fmt.Sscanf(s, "%d", &i)
	return i
}

func parseDuration(s string) time.Duration {
	// Try Go duration format first
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	// Fall back to seconds
	if sec := atoi(s); sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return 0
}
