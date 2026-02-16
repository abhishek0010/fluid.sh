package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the control plane service.
type Config struct {
	// GRPC configures the gRPC server for sandbox host connections.
	GRPC GRPCConfig `yaml:"grpc"`

	// API configures the REST API server.
	API APIConfig `yaml:"api"`

	// Database configures PostgreSQL.
	Database DatabaseConfig `yaml:"database"`

	// Orchestrator configures sandbox lifecycle management.
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
}

// GRPCConfig configures the gRPC server.
type GRPCConfig struct {
	// Address is the listen address for the gRPC server (host:port).
	Address string `yaml:"address"`

	// TLS configures mTLS for host connections.
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`

	// Insecure disables TLS (for development).
	Insecure bool `yaml:"insecure"`
}

// APIConfig configures the REST API server.
type APIConfig struct {
	// Address is the listen address for the HTTP server (host:port).
	Address string `yaml:"address"`
}

// DatabaseConfig configures PostgreSQL.
type DatabaseConfig struct {
	// URL is the PostgreSQL connection string.
	URL string `yaml:"url"`

	// MaxOpenConns sets the maximum number of open connections.
	MaxOpenConns int `yaml:"max_open_conns"`

	// MaxIdleConns sets the maximum number of idle connections.
	MaxIdleConns int `yaml:"max_idle_conns"`

	// ConnMaxLifetime sets the maximum connection lifetime.
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`

	// AutoMigrate enables automatic schema migration.
	AutoMigrate bool `yaml:"auto_migrate"`
}

// OrchestratorConfig configures sandbox lifecycle management.
type OrchestratorConfig struct {
	// HeartbeatTimeout is how long before a host is considered unhealthy.
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout"`

	// DefaultTTL is the default sandbox TTL if none is specified.
	DefaultTTL time.Duration `yaml:"default_ttl"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		GRPC: GRPCConfig{
			Address:  ":9090",
			Insecure: true,
		},
		API: APIConfig{
			Address: ":8080",
		},
		Database: DatabaseConfig{
			URL:             "postgresql://fluid:fluid@localhost:5432/fluid?sslmode=disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			AutoMigrate:     true,
		},
		Orchestrator: OrchestratorConfig{
			HeartbeatTimeout: 90 * time.Second,
			DefaultTTL:       24 * time.Hour,
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
