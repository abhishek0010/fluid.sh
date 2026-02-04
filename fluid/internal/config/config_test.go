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
	assert.Equal(t, 4096, cfg.VM.DefaultMemoryMB)
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
logging:
  level: "info"
`
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	// Set env vars to override (only LOG_LEVEL is supported)
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := LoadWithEnvOverride(configPath)
	require.NoError(t, err)

	// Env vars should override YAML
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestApplyEnvOverrides_AllFields(t *testing.T) {
	cfg := DefaultConfig()

	// Only these env vars are currently supported by applyEnvOverrides
	t.Setenv("ENABLE_ANONYMOUS_USAGE", "false")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("OPENROUTER_API_KEY", "test-api-key")

	applyEnvOverrides(cfg)

	assert.Equal(t, false, cfg.Telemetry.EnableAnonymousUsage)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "test-api-key", cfg.AIAgent.APIKey)
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
