package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupPromptTestServer creates a server with a temp directory containing
// some .sdp/ artifacts for prompt testing.
func setupPromptTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	// Write a scout.json so prompts have some data.
	scoutContent := `{"name":"test-project","languages":["Go"]}`
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "scout.json"),
		[]byte(scoutContent),
		0o644,
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})
	return srv, tmpDir
}

func TestRegisterPrompts(t *testing.T) {
	srv := NewServer(ServerConfig{RepoRoot: "."})
	require.NotNil(t, srv)
	assert.NotNil(t, srv.inner)
}

func TestUnderstandPrompt_DefaultArgs(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      "understand",
			Arguments: map[string]string{},
		},
	}

	result, err := srv.handleUnderstandPrompt(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Understand this codebase", result.Description)
	require.Len(t, result.Messages, 1)
	assert.Equal(t, mcp.RoleUser, result.Messages[0].Role)
	assert.Contains(t, result.Messages[0].Content.(mcp.TextContent).Text, "test-project")
	assert.Contains(t, result.Messages[0].Content.(mcp.TextContent).Text, "Depth: standard")
}

func TestUnderstandPrompt_DeepWithFocus(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "understand",
			Arguments: map[string]string{
				"depth": "deep",
				"focus": "security",
			},
		},
	}

	result, err := srv.handleUnderstandPrompt(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Depth: deep")
	assert.Contains(t, text, "Focus area: security")
}

func TestUnderstandPrompt_QuickDepth(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "understand",
			Arguments: map[string]string{
				"depth": "quick",
			},
		},
	}

	result, err := srv.handleUnderstandPrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Depth: quick")
}

func TestUnderstandPrompt_NilArgs(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      "understand",
			Arguments: nil,
		},
	}

	result, err := srv.handleUnderstandPrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Depth: standard")
}

func TestBuildPrompt_WithDescription(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "build",
			Arguments: map[string]string{
				"description": "Add user authentication",
				"scope":       "feature",
			},
		},
	}

	result, err := srv.handleBuildPrompt(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Build a new feature", result.Description)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Add user authentication")
	assert.Contains(t, text, "Scope: feature")
	assert.Contains(t, text, "test-project")
}

func TestBuildPrompt_MissingDescription(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      "build",
			Arguments: map[string]string{},
		},
	}

	_, err := srv.handleBuildPrompt(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "description")
	assert.Contains(t, err.Error(), "required")
}

func TestBuildPrompt_DefaultScope(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "build",
			Arguments: map[string]string{
				"description": "New module",
			},
		},
	}

	result, err := srv.handleBuildPrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Scope: feature")
}

func TestFixPrompt_WithAllArgs(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "fix",
			Arguments: map[string]string{
				"description": "Null pointer in handler",
				"severity":    "critical",
				"issue":       "BUG-123",
			},
		},
	}

	result, err := srv.handleFixPrompt(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Diagnose and fix a problem", result.Description)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Null pointer in handler")
	assert.Contains(t, text, "Severity: critical")
	assert.Contains(t, text, "BUG-123")
}

func TestFixPrompt_MissingDescription(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      "fix",
			Arguments: map[string]string{},
		},
	}

	_, err := srv.handleFixPrompt(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestFixPrompt_DefaultSeverity(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "fix",
			Arguments: map[string]string{
				"description": "Something broke",
			},
		},
	}

	result, err := srv.handleFixPrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Severity: normal")
}

func TestReviewPrompt_DefaultScope(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      "review",
			Arguments: map[string]string{},
		},
	}

	result, err := srv.handleReviewPrompt(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Review changes for quality", result.Description)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Scope: code")
	assert.Contains(t, text, "test-project")
}

func TestReviewPrompt_SecurityScope(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "review",
			Arguments: map[string]string{
				"scope": "security",
			},
		},
	}

	result, err := srv.handleReviewPrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Scope: security")
}

func TestOperatePrompt_DefaultMode(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name:      "operate",
			Arguments: map[string]string{},
		},
	}

	result, err := srv.handleOperatePrompt(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Operational task", result.Description)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Mode: triage")
}

func TestOperatePrompt_DeployMode(t *testing.T) {
	srv, _ := setupPromptTestServer(t)

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "operate",
			Arguments: map[string]string{
				"mode": "deploy",
			},
		},
	}

	result, err := srv.handleOperatePrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text
	assert.Contains(t, text, "Mode: deploy")
}

func TestPromptTemplate_MissingArtifacts(t *testing.T) {
	// Server with no .sdp/ directory at all.
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "understand",
			Arguments: map[string]string{
				"depth": "standard",
			},
		},
	}

	result, err := srv.handleUnderstandPrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text

	// Should contain a hint to run sdp_scout.
	assert.Contains(t, text, "Run sdp_scout first")
	// Should NOT contain the scout data section header without data.
	assert.NotContains(t, text, "### Project Card (from scout)")
}

func TestPromptTemplate_WithAllArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(filepath.Join(sdpDir, "architecture"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sdpDir, "metrics"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sdpDir, "specs"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(sdpDir, "scout.json"), []byte(`{"name":"full-project"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sdpDir, "architecture", "report.json"), []byte(`{"components":10}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sdpDir, "metrics", "report.json"), []byte(`{"commits":500}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(sdpDir, "specs", "report.json"), []byte(`{"apis":25}`), 0o644))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "review",
			Arguments: map[string]string{
				"scope": "arch",
			},
		},
	}

	result, err := srv.handleReviewPrompt(context.Background(), req)
	require.NoError(t, err)
	text := result.Messages[0].Content.(mcp.TextContent).Text

	// All data sections should be present.
	assert.Contains(t, text, "full-project")
	assert.Contains(t, text, "### Architecture (from architect)")
	assert.Contains(t, text, "### Process Health (from metrics)")
	assert.Contains(t, text, "### Specifications (from spec)")
	assert.Contains(t, text, "Scope: arch")
}

func TestGetArg(t *testing.T) {
	t.Run("nil map returns default", func(t *testing.T) {
		assert.Equal(t, "default", getArg(nil, "key", "default"))
	})

	t.Run("missing key returns default", func(t *testing.T) {
		assert.Equal(t, "default", getArg(map[string]string{}, "key", "default"))
	})

	t.Run("empty value returns default", func(t *testing.T) {
		assert.Equal(t, "default", getArg(map[string]string{"key": ""}, "key", "default"))
	})

	t.Run("present value overrides default", func(t *testing.T) {
		assert.Equal(t, "value", getArg(map[string]string{"key": "value"}, "key", "default"))
	})
}

func TestCollectAvailableData(t *testing.T) {
	t.Run("no artifacts returns empty data with repo name", func(t *testing.T) {
		tmpDir := t.TempDir()
		srv := NewServer(ServerConfig{RepoRoot: tmpDir})
		data := srv.collectAvailableData()
		assert.Equal(t, filepath.Base(tmpDir), data.RepoName)
		assert.Empty(t, data.ScoutJSON)
		assert.Empty(t, data.ArchitectSummary)
		assert.Empty(t, data.MetricsSummary)
	})

	t.Run("with artifacts populates fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		sdpDir := filepath.Join(tmpDir, ".sdp")
		require.NoError(t, os.MkdirAll(sdpDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(sdpDir, "scout.json"), []byte(`{"name":"x"}`), 0o644))

		srv := NewServer(ServerConfig{RepoRoot: tmpDir})
		data := srv.collectAvailableData()
		assert.Equal(t, `{"name":"x"}`, data.ScoutJSON)
		assert.Empty(t, data.ArchitectSummary) // no architect file
	})
}
