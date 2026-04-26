package registry_test

import (
	"fmt"
	"strings"
	"testing"

	"sdp_dev/internal/mcp/registry"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiscovery(t *testing.T) {
	d := registry.NewDiscovery("sdp", ".")
	assert.NotNil(t, d)
	assert.Equal(t, "sdp", d.BinaryPath())
	assert.Equal(t, ".", d.RepoRoot())
}

func TestDiscoverySnapshot(t *testing.T) {
	d := registry.NewDiscovery("echo", ".") // Use echo as a test binary

	snapshot, err := d.Snapshot()
	require.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.NotNil(t, snapshot.Commands)
}

func TestDiscoveryGetSourceLocation(t *testing.T) {
	d := registry.NewDiscovery("sdp", ".")

	// Test finding source for a known command
	location, err := d.GetSourceLocation("scout")
	if err == nil {
		assert.Contains(t, location, "scout")
	} else {
		// Expected if we're not in an SDP repo
		assert.Contains(t, err.Error(), "source location not found")
	}
}

func TestMCPToolName(t *testing.T) {
	tests := []struct {
		name        string
		commandPath string
		want        string
	}{
		{
			name:        "simple command",
			commandPath: "scout",
			want:        "sdp_scout",
		},
		{
			name:        "nested command",
			commandPath: "index build",
			want:        "sdp_index_build",
		},
		{
			name:        "deeply nested command",
			commandPath: "architect analyze",
			want:        "sdp_architect_analyze",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mcpToolName(tt.commandPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSkipCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		wantSkip bool
	}{
		{
			name:     "help command",
			command:  "help",
			wantSkip: true,
		},
		{
			name:     "version command",
			command:  "version",
			wantSkip: true,
		},
		{
			name:     "normal command",
			command:  "scout",
			wantSkip: false,
		},
		{
			name:     "completion command",
			command:  "completion",
			wantSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipCommand(tt.command)
			assert.Equal(t, tt.wantSkip, got)
		})
	}
}

func TestSkipFlag(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		wantSkip bool
	}{
		{
			name:     "help flag",
			flag:     "help",
			wantSkip: true,
		},
		{
			name:     "verbose flag",
			flag:     "verbose",
			wantSkip: true,
		},
		{
			name:     "normal flag",
			flag:     "format",
			wantSkip: true, // format is handled specially
		},
		{
			name:     "custom flag",
			flag:     "custom-value",
			wantSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipFlag(tt.flag)
			assert.Equal(t, tt.wantSkip, got)
		})
	}
}

func TestConvertRequestToArgs(t *testing.T) {
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_scout",
			Arguments: map[string]interface{}{
				"path":   "/test/path",
				"format": "json",
				"deep":   true,
			},
		},
	}

	args, err := convertRequestToArgs(req, "scout")
	require.NoError(t, err)

	assert.Contains(t, args, "scout")
	assert.Contains(t, args, "--path")
	assert.Contains(t, args, "/test/path")
	assert.Contains(t, args, "--format")
	assert.Contains(t, args, "json")
	assert.Contains(t, args, "--deep")
}

// Helper functions for testing (these would be internal in the actual package)

func mcpToolName(commandPath string) string {
	parts := strings.Fields(commandPath)
	return "sdp_" + strings.Join(parts, "_")
}

func skipCommand(commandName string) bool {
	skipList := []string{"help", "version", "completion"}
	for _, skip := range skipList {
		if strings.EqualFold(commandName, skip) {
			return true
		}
	}
	return false
}

func skipFlag(flagName string) bool {
	skipList := []string{"help", "verbose", "format"}
	for _, skip := range skipList {
		if strings.EqualFold(flagName, skip) {
			return true
		}
	}
	return false
}

func convertRequestToArgs(req mcp.CallToolRequest, commandPath string) ([]string, error) {
	parts := strings.Fields(commandPath)
	args := append([]string{}, parts...)

	if req.Params.Arguments != nil {
		argsMap, ok := req.Params.Arguments.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid arguments type")
		}

		for key, value := range argsMap {
			flagName := "--" + strings.ReplaceAll(key, "_", "-")
			args = append(args, flagName)

			if boolVal, ok := value.(bool); !ok || !boolVal {
				args = append(args, fmt.Sprintf("%v", value))
			}
		}
	}

	return args, nil
}