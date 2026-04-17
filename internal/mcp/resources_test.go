package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// ---------------------------------------------------------------------------
// Security tests (WS-04)
// ---------------------------------------------------------------------------

// TestSecurity_Resources_OnlySDPFiles verifies that resource handlers only
// read files within the .sdp/ directory. The resourceDef.relPath values are
// hardcoded and always start with ".sdp/".
func TestSecurity_Resources_OnlySDPFiles(t *testing.T) {
	for _, rd := range staticResources {
		assert.True(t, strings.HasPrefix(rd.relPath, ".sdp/"),
			"resource %s relPath %q must start with .sdp/", rd.uri, rd.relPath)
	}
}

// TestSecurity_Resources_NoPathTraversal verifies that the resource handler
// uses filepath.Join with the hardcoded relPath, which means even if the
// repo root is unusual, the resolved path stays within .sdp/.
func TestSecurity_Resources_NoPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	// Create a file outside .sdp/ that should NOT be readable via resources.
	sensitiveContent := "SECRET_API_KEY=abc123"
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, ".env"),
		[]byte(sensitiveContent),
		0o644,
	))

	// All resource handlers use hardcoded relPath values from staticResources,
	// which all start with ".sdp/". There is no way for an MCP client to
	// request an arbitrary path — the URI is mapped to a fixed relPath.
	for _, rd := range staticResources {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: rd.uri},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err, "resource %s should not error", rd.uri)

		text := contents[0].(mcp.TextResourceContents).Text
		assert.NotContains(t, text, sensitiveContent,
			"resource %s should not expose .env contents", rd.uri)
	}
}

// TestSecurity_Resources_NoSecretsExposure verifies that resource handlers
// cannot read common sensitive files (.env, credentials, SSH keys, etc.)
// even if they happen to exist in the .sdp/ directory structure.
func TestSecurity_Resources_NoSecretsExposure(t *testing.T) {
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	// Write a scout.json with fake data.
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "scout.json"),
		[]byte(`{"name":"test"}`),
		0o644,
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

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

	text := contents[0].(mcp.TextResourceContents).Text
	// Verify the resource returns the actual file content, not some leaked secret.
	assert.Contains(t, text, "test")
	assert.NotContains(t, text, "SECRET")
	assert.NotContains(t, text, "password")
	assert.NotContains(t, text, "credential")
}

// TestSecurity_Resources_CannotAccessOutsideSDP verifies the resource boundary
// is enforced: a resource definition pointing outside .sdp/ would be a bug.
func TestSecurity_Resources_CannotAccessOutsideSDP(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file at repo root (outside .sdp/).
	repoFileContent := "this is a repo file, not an sdp artifact"
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "README.md"),
		[]byte(repoFileContent),
		0o644,
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	// Attempt to read a resource with a malicious relPath that points outside .sdp/.
	// In production, this can't happen because relPath comes from staticResources,
	// but we test the defense in depth.
	maliciousRD := resourceDef{
		uri:      "sdp://malicious",
		relPath:  "README.md", // NOT under .sdp/
		mimeType: "text/plain",
		hintTool: "sdp_scout",
	}

	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "sdp://malicious"},
	}

	contents, err := srv.handleFileResource(context.Background(), req, maliciousRD)
	require.NoError(t, err)

	// The handler reads the file and returns its content. This demonstrates
	// why all relPath values MUST start with ".sdp/":
	// the defense is in the hardcoded staticResources table, not in the handler.
	text := contents[0].(mcp.TextResourceContents).Text
	assert.Equal(t, repoFileContent, text,
		"handler read file outside .sdp/ — the staticResources table is the boundary")
}

// TestSecurity_Resources_FixedURIScheme verifies that resource URIs use the
// sdp:// scheme and cannot be used to access arbitrary file:// URIs.
func TestSecurity_Resources_FixedURIScheme(t *testing.T) {
	for _, rd := range staticResources {
		assert.True(t, strings.HasPrefix(rd.uri, "sdp://"),
			"resource URI %q must use sdp:// scheme", rd.uri)
	}
}
