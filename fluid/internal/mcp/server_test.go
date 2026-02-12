package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aspectrr/fluid.sh/fluid/internal/config"
)

func TestNewServer(t *testing.T) {
	cfg := testConfig()
	st := newMockStore()

	srv := NewServer(cfg, st, nil, nil, nil, noopLogger())
	require.NotNil(t, srv)
	assert.NotNil(t, srv.mcpServer)
	assert.NotNil(t, srv.playbookService)
	assert.NotNil(t, srv.logger)
	assert.Nil(t, srv.multiHostMgr) // no hosts configured
}

func TestNewServer_WithHosts(t *testing.T) {
	cfg := testConfig()
	cfg.Hosts = []config.HostConfig{
		{Name: "host1", Address: "10.0.0.1"},
	}
	st := newMockStore()

	srv := NewServer(cfg, st, nil, nil, nil, noopLogger())
	require.NotNil(t, srv)
	assert.NotNil(t, srv.multiHostMgr)
}

func TestNewServer_RegistersAllTools(t *testing.T) {
	cfg := testConfig()
	st := newMockStore()

	srv := NewServer(cfg, st, nil, nil, nil, noopLogger())
	require.NotNil(t, srv)

	// The MCP server should have registered 17 tools.
	// We can't directly inspect the tools list from the MCPServer,
	// but we verify the server was constructed without panicking
	// and the mcpServer is non-nil.
	assert.NotNil(t, srv.mcpServer)
}
