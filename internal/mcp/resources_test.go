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

func TestRegisterResources(t *testing.T) {
	srv := NewServer(ServerConfig{RepoRoot: "."})
	require.NotNil(t, srv)
	// Resources are registered during NewServer; verify the inner server exists.
	assert.NotNil(t, srv.inner)
}

func TestHandleFileResource_HappyPath(t *testing.T) {
	// Create a temp directory with .sdp/ artifacts.
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	// Write a manifest file.
	manifestContent := "# Test Manifest\n\nThis is a test manifest."
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "manifest.md"),
		[]byte(manifestContent),
		0o644,
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	// Find the manifest resource def.
	rd := resourceDef{
		uri:      "sdp://manifest",
		relPath:  ".sdp/manifest.md",
		mimeType: "text/markdown",
		hintTool: "sdp_index_build",
	}

	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "sdp://manifest",
		},
	}

	contents, err := srv.handleFileResource(context.Background(), req, rd)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok, "expected TextResourceContents")
	assert.Equal(t, manifestContent, textContent.Text)
	assert.Equal(t, "text/markdown", textContent.MIMEType)
	assert.Equal(t, "sdp://manifest", textContent.URI)
}

func TestHandleFileResource_MissingArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	rd := resourceDef{
		uri:      "sdp://scout",
		relPath:  ".sdp/scout.json",
		mimeType: "application/json",
		hintTool: "sdp_scout",
	}

	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "sdp://scout",
		},
	}

	contents, err := srv.handleFileResource(context.Background(), req, rd)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok, "expected TextResourceContents")
	assert.Contains(t, textContent.Text, "Artifact not found")
	assert.Contains(t, textContent.Text, "sdp_scout")
}

func TestHandleFileResource_PartialArtifacts(t *testing.T) {
	// Set up a repo with only some artifacts.
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	// Only write scout.json, not the others.
	scoutContent := `{"languages": ["Go"], "name": "test-project"}`
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "scout.json"),
		[]byte(scoutContent),
		0o644,
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	t.Run("existing scout resource", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://scout",
			relPath:  ".sdp/scout.json",
			mimeType: "application/json",
			hintTool: "sdp_scout",
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://scout"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)
		assert.Equal(t, scoutContent, textContent.Text)
		assert.Equal(t, "application/json", textContent.MIMEType)
	})

	t.Run("missing architect resource", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://architect",
			relPath:  ".sdp/architecture/report.json",
			mimeType: "application/json",
			hintTool: "sdp_architect",
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://architect"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Artifact not found")
		assert.Contains(t, textContent.Text, "sdp_architect")
	})

	t.Run("missing metrics resource", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://metrics",
			relPath:  ".sdp/metrics/report.json",
			mimeType: "application/json",
			hintTool: "sdp_metrics",
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://metrics"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Artifact not found")
		assert.Contains(t, textContent.Text, "sdp_metrics")
	})

	t.Run("missing spec resource", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://spec",
			relPath:  ".sdp/specs/report.json",
			mimeType: "application/json",
			hintTool: "sdp_spec",
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://spec"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Artifact not found")
		assert.Contains(t, textContent.Text, "sdp_spec")
	})

	t.Run("missing bootstrap resource", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://bootstrap",
			relPath:  ".sdp/bootstrap-report.json",
			mimeType: "application/json",
			hintTool: "sdp_bootstrap",
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://bootstrap"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)
		assert.Contains(t, textContent.Text, "Artifact not found")
		assert.Contains(t, textContent.Text, "sdp_bootstrap")
	})
}

func TestHandleFileResource_NestedArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	archDir := filepath.Join(tmpDir, ".sdp", "architecture")
	require.NoError(t, os.MkdirAll(archDir, 0o755))

	archContent := `{"components": 5, "quality": "good"}`
	require.NoError(t, os.WriteFile(
		filepath.Join(archDir, "report.json"),
		[]byte(archContent),
		0o644,
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	rd := resourceDef{
		uri:      "sdp://architect",
		relPath:  ".sdp/architecture/report.json",
		mimeType: "application/json",
		hintTool: "sdp_architect",
	}

	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "sdp://architect"},
	}

	contents, err := srv.handleFileResource(context.Background(), req, rd)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok)
	assert.Equal(t, archContent, textContent.Text)
}

func TestAllResourcesHaveValidDefs(t *testing.T) {
	uriSet := make(map[string]bool, len(staticResources))
	for _, rd := range staticResources {
		assert.NotEmpty(t, rd.uri, "resource URI must not be empty")
		assert.NotEmpty(t, rd.name, "resource name must not be empty")
		assert.NotEmpty(t, rd.relPath, "resource relPath must not be empty")
		assert.NotEmpty(t, rd.mimeType, "resource mimeType must not be empty")
		assert.NotEmpty(t, rd.hintTool, "resource hintTool must not be empty")
		assert.False(t, uriSet[rd.uri], "duplicate resource URI: %s", rd.uri)
		uriSet[rd.uri] = true
	}
}
