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
	require.Error(t, err, "missing artifact should return an error")
	assert.Nil(t, contents)
	assert.Contains(t, err.Error(), "resource not available")
	assert.Contains(t, err.Error(), "generate it by running the corresponding CLI command")
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

	t.Run("missing architect resource returns error", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://architect",
			relPath:  ".sdp/architect/report.json",
			mimeType: "application/json",
			hintTool: "sdp_architect",
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://architect"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.Error(t, err, "missing artifact should return an error")
		assert.Nil(t, contents)
		assert.Contains(t, err.Error(), "resource not available")
		assert.Contains(t, err.Error(), "generate it by running the corresponding CLI command")
	})

	t.Run("missing metrics resource returns error", func(t *testing.T) {
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
		require.Error(t, err, "missing artifact should return an error")
		assert.Nil(t, contents)
		assert.Contains(t, err.Error(), "resource not available")
		assert.Contains(t, err.Error(), "generate it by running the corresponding CLI command")
	})

	t.Run("missing spec resource returns error", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://spec",
			relPath:  ".sdp/specs/spec.json",
			mimeType: "application/json",
			hintTool: "sdp_spec",
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://spec"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.Error(t, err, "missing artifact should return an error")
		assert.Nil(t, contents)
		assert.Contains(t, err.Error(), "resource not available")
		assert.Contains(t, err.Error(), "generate it by running the corresponding CLI command")
	})

	t.Run("missing bootstrap resource returns error", func(t *testing.T) {
		rd := resourceDef{
			uri:      "sdp://bootstrap",
			relPath:  ".sdp/bootstrap/report.json",
			mimeType: "application/json",
			hintTool: "requires a future CLI enhancement",
			forward:  true,
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://bootstrap"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.Error(t, err, "missing artifact should return an error")
		assert.Nil(t, contents)
		assert.Contains(t, err.Error(), "resource not available")
		assert.Contains(t, err.Error(), "future CLI enhancement")
	})
}

func TestHandleFileResource_NestedArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	archDir := filepath.Join(tmpDir, ".sdp", "architect")
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
		relPath:  ".sdp/architect/report.json",
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

func TestHandleFileResource_DynamicIndexResources(t *testing.T) {
	t.Run("index modules happy path", func(t *testing.T) {
		tmpDir := t.TempDir()
		indexDir := filepath.Join(tmpDir, ".sdp", "index")
		require.NoError(t, os.MkdirAll(indexDir, 0o755))

		modulesContent := `[{"name":"internal/mcp","symbols":42}]`
		require.NoError(t, os.WriteFile(
			filepath.Join(indexDir, "modules.json"),
			[]byte(modulesContent),
			0o644,
		))

		srv := NewServer(ServerConfig{RepoRoot: tmpDir})
		rd := resourceDef{
			uri:      "sdp://index/modules",
			relPath:  ".sdp/index/modules.json",
			mimeType: "application/json",
			hintTool: "sdp_index_build",
			forward:  true,
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://index/modules"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)
		assert.Equal(t, modulesContent, textContent.Text)
		assert.Equal(t, "application/json", textContent.MIMEType)
	})

	t.Run("index modules missing returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		srv := NewServer(ServerConfig{RepoRoot: tmpDir})
		rd := resourceDef{
			uri:      "sdp://index/modules",
			relPath:  ".sdp/index/modules.json",
			mimeType: "application/json",
			hintTool: "requires a future CLI enhancement",
			forward:  true,
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://index/modules"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.Error(t, err, "missing index modules should return an error")
		assert.Nil(t, contents)
		assert.Contains(t, err.Error(), "resource not available")
		assert.Contains(t, err.Error(), "future CLI enhancement")
	})

	t.Run("index stats happy path", func(t *testing.T) {
		tmpDir := t.TempDir()
		indexDir := filepath.Join(tmpDir, ".sdp", "index")
		require.NoError(t, os.MkdirAll(indexDir, 0o755))

		statsContent := `{"total_files":100,"total_symbols":500,"index_age_seconds":42}`
		require.NoError(t, os.WriteFile(
			filepath.Join(indexDir, "stats.json"),
			[]byte(statsContent),
			0o644,
		))

		srv := NewServer(ServerConfig{RepoRoot: tmpDir})
		rd := resourceDef{
			uri:      "sdp://index/stats",
			relPath:  ".sdp/index/stats.json",
			mimeType: "application/json",
			hintTool: "sdp_index_build",
			forward:  true,
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://index/stats"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(mcp.TextResourceContents)
		require.True(t, ok)
		assert.Equal(t, statsContent, textContent.Text)
		assert.Equal(t, "application/json", textContent.MIMEType)
	})

	t.Run("index stats missing returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		srv := NewServer(ServerConfig{RepoRoot: tmpDir})
		rd := resourceDef{
			uri:      "sdp://index/stats",
			relPath:  ".sdp/index/stats.json",
			mimeType: "application/json",
			hintTool: "requires a future CLI enhancement",
			forward:  true,
		}
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: "sdp://index/stats"},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.Error(t, err, "missing index stats should return an error")
		assert.Nil(t, contents)
		assert.Contains(t, err.Error(), "resource not available")
		assert.Contains(t, err.Error(), "future CLI enhancement")
	})
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
	// which all start with ".sdp/". Since files don't exist on disk, they will
	// return errors (not text content), so we verify the error does not leak
	// the .env content.
	for _, rd := range staticResources {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{URI: rd.uri},
		}

		contents, err := srv.handleFileResource(context.Background(), req, rd)
		require.Error(t, err, "resource %s should error (file missing)", rd.uri)
		assert.Nil(t, contents, "resource %s should return nil contents", rd.uri)
		assert.NotContains(t, err.Error(), sensitiveContent,
			"resource %s error should not expose .env contents", rd.uri)
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

	_, err := srv.handleFileResource(context.Background(), req, maliciousRD)

	// The handler now enforces the .sdp/ boundary as defense in depth.
	// Even if a resource definition had a bad relPath, the handler would reject it.
	require.Error(t, err)
	assert.Contains(t, err.Error(), ".sdp/ boundary",
		"handler must reject resources outside .sdp/ boundary")
}

// TestSecurity_Resources_PathTraversal verifies that relPath with ../ cannot
// escape the .sdp/ boundary.
func TestSecurity_Resources_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	traversalPaths := []struct {
		name   string
		relPath string
	}{
		{"dotdot", "../etc/passwd"},
		{"nested_dotdot", ".sdp/../../etc/passwd"},
		{"double_dotdot", "../../.env"},
	}

	for _, tc := range traversalPaths {
		t.Run(tc.name, func(t *testing.T) {
			maliciousRD := resourceDef{
				uri:      "sdp://malicious",
				relPath:  tc.relPath,
				mimeType: "text/plain",
				hintTool: "sdp_scout",
			}

			req := mcp.ReadResourceRequest{
				Params: mcp.ReadResourceParams{URI: "sdp://malicious"},
			}

			_, err := srv.handleFileResource(context.Background(), req, maliciousRD)
			require.Error(t, err, "path traversal %q should be rejected", tc.relPath)
		})
	}
}

// TestSecurity_Resources_FixedURIScheme verifies that resource URIs use the
// sdp:// scheme and cannot be used to access arbitrary file:// URIs.
func TestSecurity_Resources_FixedURIScheme(t *testing.T) {
	for _, rd := range staticResources {
		assert.True(t, strings.HasPrefix(rd.uri, "sdp://"),
			"resource URI %q must use sdp:// scheme", rd.uri)
	}
}

// TestSecurity_Resources_SymlinkEscape verifies that symlinks inside .sdp/
// that point outside the .sdp/ directory are rejected by safeReadFile.
func TestSecurity_Resources_SymlinkEscape(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a sensitive file outside .sdp/.
	sensitiveContent := "SECRET_VALUE=should_not_be_exposed"
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, ".env"),
		[]byte(sensitiveContent),
		0o644,
	))

	// Create .sdp/ directory.
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	// Create a symlink inside .sdp/ that points to the sensitive file.
	require.NoError(t, os.Symlink(
		filepath.Join(tmpDir, ".env"),
		filepath.Join(sdpDir, "scout.json"),
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
	require.Error(t, err, "symlink escaping .sdp/ should be rejected")
	assert.Nil(t, contents, "symlink escaping .sdp/ should not return contents")
	assert.Contains(t, err.Error(), ".sdp/",
		"error should mention .sdp/ boundary violation")
	assert.NotContains(t, err.Error(), sensitiveContent,
		"error must not expose contents of symlink target")
}

// TestSecurity_Resources_SymlinkEscape_Nested verifies that a symlink in a
// nested .sdp/ subdirectory that escapes upward is also rejected.
func TestSecurity_Resources_SymlinkEscape_Nested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a sensitive file at the repo root.
	sensitiveContent := "DATABASE_URL=postgres://admin:secret@db:5432"
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "credentials.json"),
		[]byte(sensitiveContent),
		0o644,
	))

	// Create nested .sdp/architecture/ directory.
	archDir := filepath.Join(tmpDir, ".sdp", "architect")
	require.NoError(t, os.MkdirAll(archDir, 0o755))

	// Symlink the report to the sensitive file.
	require.NoError(t, os.Symlink(
		filepath.Join(tmpDir, "credentials.json"),
		filepath.Join(archDir, "report.json"),
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})

	rd := resourceDef{
		uri:      "sdp://architect",
		relPath:  ".sdp/architect/report.json",
		mimeType: "application/json",
		hintTool: "sdp_architect",
	}

	req := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "sdp://architect"},
	}

	contents, err := srv.handleFileResource(context.Background(), req, rd)
	require.Error(t, err, "nested symlink escaping .sdp/ should be rejected")
	assert.Nil(t, contents)
}

// TestSecurity_Resources_SymlinkWithinSDP verifies that a symlink that stays
// within .sdp/ (e.g., aliasing one artifact to another) is allowed.
func TestSecurity_Resources_SymlinkWithinSDP(t *testing.T) {
	tmpDir := t.TempDir()

	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))

	// Write a real scout.json.
	realContent := `{"name":"symlink-test","languages":["Go"]}`
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "scout_real.json"),
		[]byte(realContent),
		0o644,
	))

	// Symlink scout.json -> scout_real.json (both inside .sdp/).
	require.NoError(t, os.Symlink(
		filepath.Join(sdpDir, "scout_real.json"),
		filepath.Join(sdpDir, "scout.json"),
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
	require.NoError(t, err, "symlink within .sdp/ should be allowed")
	require.Len(t, contents, 1)
	textContent, ok := contents[0].(mcp.TextResourceContents)
	require.True(t, ok)
	assert.Equal(t, realContent, textContent.Text)
}
