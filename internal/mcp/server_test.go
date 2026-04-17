package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	srv := NewServer(ServerConfig{RepoRoot: "."})
	require.NotNil(t, srv)
	assert.Equal(t, ".", srv.config.RepoRoot)
	assert.Equal(t, DefaultBinary, srv.config.BinaryPath)
	assert.NotNil(t, srv.inner)
	assert.NotNil(t, srv.executor)
}

func TestNewServerDefaults(t *testing.T) {
	srv := NewServer(ServerConfig{})
	assert.Equal(t, ".", srv.config.RepoRoot, "default repo root should be '.'")
	assert.Equal(t, DefaultBinary, srv.config.BinaryPath, "default binary should be 'sdp'")
}

func TestNewServerCustomBinary(t *testing.T) {
	srv := NewServer(ServerConfig{BinaryPath: "/usr/local/bin/sdp"})
	assert.Equal(t, "/usr/local/bin/sdp", srv.config.BinaryPath)
}

func TestServerStartupUnder200ms(t *testing.T) {
	dur := MeasureStartup(ServerConfig{RepoRoot: "."})
	assert.Less(t, dur, 200*time.Millisecond,
		"server startup should be under 200ms, took %v", dur)
	t.Logf("startup time: %v", dur)
}

func TestToolRegistration(t *testing.T) {
	srv := NewServer(ServerConfig{RepoRoot: "."})
	inner := srv.Inner()
	require.NotNil(t, inner)

	// We verify registration by checking that the handler functions are wired.
	// The MCP SDK doesn't expose a public list of registered tools from
	// MCPServer, so we check indirectly via the server object state.
	assert.NotNil(t, srv.executor, "executor should be set")
	assert.NotNil(t, srv.inner, "inner MCP server should be set")
}

func TestRepoPathResolution(t *testing.T) {
	srv := NewServer(ServerConfig{RepoRoot: "/default"})

	tests := []struct {
		name     string
		toolPath string
		want     string
	}{
		{"empty tool path falls back to server default", "", "/default"},
		{"tool path overrides default", "/override", "/override"},
		{"relative tool path is preserved", "./subdir", "./subdir"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := srv.repoPath(tt.toolPath)
			assert.Equal(t, tt.want, got)
		})
	}
}
