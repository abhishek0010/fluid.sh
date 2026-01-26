package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 2, cfg.VM.DefaultVCPUs)
	assert.Equal(t, 2048, cfg.VM.DefaultMemoryMB)
	assert.Equal(t, "qemu:///system", cfg.Libvirt.URI)
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.yaml")
	require.NoError(t, err)
	assert.Equal(t, DefaultConfig(), cfg)
}

func TestLoad_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
api:
  addr: ":9090"
  read_timeout: 30s

vm:
  default_vcpus: 4
  default_memory_mb: 4096
  command_timeout: 5m

logging:
  level: "debug"
  format: "json"
`
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	assert.Equal(t, 4, cfg.VM.DefaultVCPUs)
	assert.Equal(t, 4096, cfg.VM.DefaultMemoryMB)
	assert.Equal(t, 5*time.Minute, cfg.VM.CommandTimeout)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoad_PartialYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Only override some values - defaults should fill the rest
	yaml := `
api:
  addr: ":3000"
logging:
  level: "warn"
`
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)

	// Overridden values
	assert.Equal(t, "warn", cfg.Logging.Level)

	// Default values preserved
	assert.Equal(t, 2, cfg.VM.DefaultVCPUs)
	assert.Equal(t, "qemu:///system", cfg.Libvirt.URI)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0o644)
	require.NoError(t, err)

	_, err = Load(configPath)
	assert.Error(t, err)
}

func TestLoadWithEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
api:
  addr: ":8080"
`
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	// Set env vars to override
	t.Setenv("API_HTTP_ADDR", ":9999")
	t.Setenv("DEFAULT_VCPUS", "8")

	cfg, err := LoadWithEnvOverride(configPath)
	require.NoError(t, err)

	// Env vars should override YAML
	assert.Equal(t, 8, cfg.VM.DefaultVCPUs)
}

func TestApplyEnvOverrides_AllFields(t *testing.T) {
	cfg := DefaultConfig()

	t.Setenv("API_HTTP_ADDR", ":7777")
	t.Setenv("API_SHUTDOWN_TIMEOUT_SEC", "30")
	t.Setenv("LIBVIRT_URI", "qemu:///session")
	t.Setenv("LIBVIRT_NETWORK", "custom-net")
	t.Setenv("BASE_IMAGE_DIR", "/custom/base")
	t.Setenv("SANDBOX_WORKDIR", "/custom/work")
	t.Setenv("DEFAULT_VCPUS", "16")
	t.Setenv("DEFAULT_MEMORY_MB", "8192")
	t.Setenv("COMMAND_TIMEOUT_SEC", "300")
	t.Setenv("IP_DISCOVERY_TIMEOUT_SEC", "60")
	t.Setenv("SSH_PROXY_JUMP", "jump@host")
	t.Setenv("SSH_CA_KEY_PATH", "/custom/ca")
	t.Setenv("SSH_KEY_DIR", "/custom/keys")
	t.Setenv("SSH_CERT_TTL_SEC", "600")
	t.Setenv("ANSIBLE_INVENTORY_PATH", "/custom/inventory")
	t.Setenv("ANSIBLE_PLAYBOOKS_DIR", "/custom/playbooks")
	t.Setenv("ANSIBLE_IMAGE", "custom-ansible")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")

	applyEnvOverrides(cfg)

	assert.Equal(t, "qemu:///session", cfg.Libvirt.URI)
	assert.Equal(t, "custom-net", cfg.Libvirt.Network)
	assert.Equal(t, "/custom/base", cfg.Libvirt.BaseImageDir)
	assert.Equal(t, "/custom/work", cfg.Libvirt.WorkDir)
	assert.Equal(t, 16, cfg.VM.DefaultVCPUs)
	assert.Equal(t, 8192, cfg.VM.DefaultMemoryMB)
	assert.Equal(t, 5*time.Minute, cfg.VM.CommandTimeout)
	assert.Equal(t, time.Minute, cfg.VM.IPDiscoveryTimeout)
	assert.Equal(t, "jump@host", cfg.SSH.ProxyJump)
	assert.Equal(t, "/custom/ca", cfg.SSH.CAKeyPath)
	assert.Equal(t, "/custom/keys", cfg.SSH.KeyDir)
	assert.Equal(t, 10*time.Minute, cfg.SSH.CertTTL)
	assert.Equal(t, "/custom/inventory", cfg.Ansible.InventoryPath)
	assert.Equal(t, "/custom/playbooks", cfg.Ansible.PlaybooksDir)
	assert.Equal(t, "custom-ansible", cfg.Ansible.Image)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"60", 60 * time.Second},
		{"300", 5 * time.Minute},
		{"5m", 5 * time.Minute},
		{"1h", time.Hour},
		{"30s", 30 * time.Second},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
