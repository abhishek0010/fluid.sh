package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration for virsh-sandbox API.
type Config struct {
	Libvirt            LibvirtConfig   `yaml:"libvirt"`
	VM                 VMConfig        `yaml:"vm"`
	SSH                SSHConfig       `yaml:"ssh"`
	Ansible            AnsibleConfig   `yaml:"ansible"`
	Logging            LoggingConfig   `yaml:"logging"`
	Telemetry          TelemetryConfig `yaml:"telemetry"`
	AIAgent            AIAgentConfig   `yaml:"ai_agent"`
	Hosts              []HostConfig    `yaml:"hosts"`               // Remote libvirt hosts for multi-host VM management
	OnboardingComplete bool            `yaml:"onboarding_complete"` // Whether onboarding wizard has been completed
}

// AIAgentConfig holds settings for LLM integration.
type AIAgentConfig struct {
	Provider      string `yaml:"provider"` // e.g., "openrouter"
	APIKey        string `yaml:"api_key"`
	Model         string `yaml:"model"`
	Endpoint      string `yaml:"endpoint"`
	SiteURL       string `yaml:"site_url"`
	SiteName      string `yaml:"site_name"`
	DefaultSystem string `yaml:"default_system"`
	// Context window management
	TotalContextTokens int     `yaml:"max_context_tokens"` // Max tokens for context window (default: 200000)
	CompactModel       string  `yaml:"compact_model"`      // Smaller model for compaction (default: Claude 4.5 Haiku)
	CompactThreshold   float64 `yaml:"compact_threshold"`  // Auto-compact at this % of context (default: 0.9)
	TokensPerChar      float64 `yaml:"tokens_per_char"`    // Estimated tokens per character (default: 0.25)
}

// TelemetryConfig holds telemetry settings.
type TelemetryConfig struct {
	EnableAnonymousUsage bool `yaml:"enable_anonymous_usage"`
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

// HostConfig represents a remote libvirt host for multi-host VM management.
// Authentication uses system SSH config (~/.ssh/config and ssh-agent).
type HostConfig struct {
	Name         string        `yaml:"name"`          // Display name (e.g., "kvm-01")
	Address      string        `yaml:"address"`       // IP or hostname
	SSHUser      string        `yaml:"ssh_user"`      // SSH user for host (default: root)
	SSHPort      int           `yaml:"ssh_port"`      // SSH port (default: 22)
	SSHVMUser    string        `yaml:"ssh_vm_user"`   // SSH user for VMs on this host (default: root)
	QueryTimeout time.Duration `yaml:"query_timeout"` // Per-host query timeout (default: 30s)
}

// DefaultConfig returns config with sensible defaults.
func DefaultConfig() *Config {
	configDir, _ := GetConfigDir()

	return &Config{
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
			DefaultMemoryMB:    4096,
			CommandTimeout:     30 * time.Minute,
			IPDiscoveryTimeout: 2 * time.Minute,
		},
		SSH: SSHConfig{
			CAKeyPath:   filepath.Join(configDir, "ssh-ca", "ssh-ca"),
			CAPubPath:   filepath.Join(configDir, "ssh-ca", "ssh-ca.pub"),
			KeyDir:      filepath.Join(configDir, "sandbox-keys"),
			CertTTL:     30 * time.Minute,
			MaxTTL:      60 * time.Minute,
			WorkDir:     filepath.Join(configDir, "ssh-ca", "workdir"),
			DefaultUser: "sandbox",
		},
		Ansible: AnsibleConfig{
			InventoryPath: filepath.Join(configDir, "ansible", "inventory"),
			PlaybooksDir:  filepath.Join(configDir, "ansible", "playbooks"),
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		AIAgent: AIAgentConfig{
			Provider: "openrouter",
			Model:    "anthropic/claude-opus-4.6",
			Endpoint: "https://openrouter.ai/api/v1",
			DefaultSystem: "You are Fluid, an infrastructure automation agent." +
				"- Your goal is to complete the user's task by generating an Ansible playbook that recreates the task on a production machine." +
				"- Test your updates by running relevant commands on the sandbox and then building out the playbook. Do not make assumptions on outputs." +
				"- You MUST use the Ansible tools to create and manage the playbook." +
				"- Do not add an extension to the playbook name like .yml or .yaml" +
				"- Add any steps to the playbook that are necessary to fully recreate the intended output on the production system." +
				"- You CANNOT UNDER ANY CIRCUMSTANCES make a sandbox from a VM if asked to work on a different VM. For example if asked to make a sandbox of VM-1, you CANNOT make a sandbox from VM-2 if the sandbox does not work. If that happens, please stop at once.",
			TotalContextTokens: 1000000,
			CompactModel:       "anthropic/claude-haiku-4.5",
			CompactThreshold:   0.90,
			TokensPerChar:      0.33,
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

	// Apply defaults for any empty values that should have defaults
	applyDefaults(cfg)

	return cfg, nil
}

// applyDefaults fills in default values for any empty config fields.
// This handles cases where a config file exists but doesn't specify all fields.
func applyDefaults(cfg *Config) {
	defaults := DefaultConfig()

	// SSH defaults - these are critical for the tool to work
	if cfg.SSH.CAKeyPath == "" {
		cfg.SSH.CAKeyPath = defaults.SSH.CAKeyPath
	}
	if cfg.SSH.CAPubPath == "" {
		cfg.SSH.CAPubPath = defaults.SSH.CAPubPath
	}
	if cfg.SSH.KeyDir == "" {
		cfg.SSH.KeyDir = defaults.SSH.KeyDir
	}
	if cfg.SSH.WorkDir == "" {
		cfg.SSH.WorkDir = defaults.SSH.WorkDir
	}
	if cfg.SSH.DefaultUser == "" {
		cfg.SSH.DefaultUser = defaults.SSH.DefaultUser
	}
	if cfg.SSH.CertTTL == 0 {
		cfg.SSH.CertTTL = defaults.SSH.CertTTL
	}
	if cfg.SSH.MaxTTL == 0 {
		cfg.SSH.MaxTTL = defaults.SSH.MaxTTL
	}

	// Libvirt defaults
	if cfg.Libvirt.URI == "" {
		cfg.Libvirt.URI = defaults.Libvirt.URI
	}
	if cfg.Libvirt.Network == "" {
		cfg.Libvirt.Network = defaults.Libvirt.Network
	}
	if cfg.Libvirt.SSHKeyInjectMethod == "" {
		cfg.Libvirt.SSHKeyInjectMethod = defaults.Libvirt.SSHKeyInjectMethod
	}

	// VM defaults
	if cfg.VM.DefaultVCPUs == 0 {
		cfg.VM.DefaultVCPUs = defaults.VM.DefaultVCPUs
	}
	if cfg.VM.DefaultMemoryMB == 0 {
		cfg.VM.DefaultMemoryMB = defaults.VM.DefaultMemoryMB
	}
	if cfg.VM.CommandTimeout == 0 {
		cfg.VM.CommandTimeout = defaults.VM.CommandTimeout
	}
	if cfg.VM.IPDiscoveryTimeout == 0 {
		cfg.VM.IPDiscoveryTimeout = defaults.VM.IPDiscoveryTimeout
	}

	// AIAgent defaults
	if cfg.AIAgent.Provider == "" {
		cfg.AIAgent.Provider = defaults.AIAgent.Provider
	}
	if cfg.AIAgent.Model == "" {
		cfg.AIAgent.Model = defaults.AIAgent.Model
	}
	if cfg.AIAgent.Endpoint == "" {
		cfg.AIAgent.Endpoint = defaults.AIAgent.Endpoint
	}
	// if cfg.AIAgent.DefaultSystem == "" {
	// 	cfg.AIAgent.DefaultSystem = defaults.AIAgent.DefaultSystem
	// }
	if cfg.AIAgent.TotalContextTokens == 0 {
		cfg.AIAgent.TotalContextTokens = defaults.AIAgent.TotalContextTokens
	}
	if cfg.AIAgent.CompactModel == "" {
		cfg.AIAgent.CompactModel = defaults.AIAgent.CompactModel
	}
	if cfg.AIAgent.CompactThreshold == 0 {
		cfg.AIAgent.CompactThreshold = defaults.AIAgent.CompactThreshold
	}
	if cfg.AIAgent.TokensPerChar == 0 {
		cfg.AIAgent.TokensPerChar = defaults.AIAgent.TokensPerChar
	}
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
	// Telemetry
	if v := os.Getenv("ENABLE_ANONYMOUS_USAGE"); v != "" {
		cfg.Telemetry.EnableAnonymousUsage = v == "true"
	}

	// Logging
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}

	// Prioritize environment variables for API Key
	if v := os.Getenv("OPENROUTER_API_KEY"); v != "" {
		cfg.AIAgent.APIKey = v
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

// Save writes the current config back to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
