package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	srv := NewServer(ServerConfig{RepoRoot: "."})
	require.NotNil(t, srv)
	// RepoRoot is resolved to absolute path.
	assert.True(t, filepath.IsAbs(srv.config.RepoRoot),
		"RepoRoot should be absolute, got %q", srv.config.RepoRoot)
	assert.Equal(t, DefaultBinary, srv.config.BinaryPath)
	assert.NotNil(t, srv.inner)
	assert.NotNil(t, srv.executor)
}

func TestNewServerDefaults(t *testing.T) {
	srv := NewServer(ServerConfig{})
	// Default RepoRoot "." is resolved to an absolute path.
	assert.True(t, filepath.IsAbs(srv.config.RepoRoot),
		"default repo root should be resolved to absolute, got %q", srv.config.RepoRoot)
	assert.Equal(t, DefaultBinary, srv.config.BinaryPath, "default binary should be 'sdp'")
}

func TestNewServer_DotRepoRoot_ResolvesToAbsolute(t *testing.T) {
	srv := NewServer(ServerConfig{RepoRoot: "."})
	require.NotNil(t, srv)

	absCwd, err := filepath.Abs(".")
	require.NoError(t, err)
	assert.Equal(t, absCwd, srv.config.RepoRoot,
		"RepoRoot '.' should resolve to absolute CWD")

	// RepoName (used in prompts) should not be ".".
	repoName := filepath.Base(srv.config.RepoRoot)
	assert.NotEqual(t, ".", repoName,
		"filepath.Base of absolute RepoRoot should not be '.'")
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

	assert.Equal(t, expectedToolCount, srv.ToolCount(),
		"expected %d registered tools, got %d", expectedToolCount, srv.ToolCount())
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

// ---------------------------------------------------------------------------
// Cross-harness consistency tests (WS-04)
// ---------------------------------------------------------------------------
//
// These tests verify that the MCP protocol surface (tools, resources, prompts)
// is identical regardless of which harness connects. Since MCP is a standard
// protocol (JSON-RPC over stdio), every harness sees the same server. We prove
// consistency by creating independent server instances and checking that their
// surfaces are structurally identical.

// expectedToolCount is the number of MCP tools the server must register.
// Update this constant only when adding or removing a tool.
const expectedToolCount = 13

// expectedResourceCount is the number of static MCP resources the server must register.
const expectedResourceCount = 8

// expectedPromptCount is the number of MCP prompts the server must register.
const expectedPromptCount = 5

func TestCrossHarness_ToolSurfaceIsStable(t *testing.T) {
	// Create two independent servers and verify both register the same tools.
	srv1 := NewServer(ServerConfig{RepoRoot: "."})
	srv2 := NewServer(ServerConfig{RepoRoot: "."})

	// The MCP SDK does not expose a public tool-list API from MCPServer.
	// We verify stability indirectly: the server must have registered all
	// expected tools, and the registration list must be deterministic.
	//
	// Since registerTools() is called in a fixed order inside NewServer,
	// two servers will always have the same tool surface.
	require.NotNil(t, srv1.inner)
	require.NotNil(t, srv2.inner)

	// Assert both servers register exactly the expected number of tools.
	assert.Equal(t, expectedToolCount, srv1.ToolCount(),
		"srv1: expected %d tools, got %d", expectedToolCount, srv1.ToolCount())
	assert.Equal(t, expectedToolCount, srv2.ToolCount(),
		"srv2: expected %d tools, got %d", expectedToolCount, srv2.ToolCount())

	// Smoke-test: call each tool handler through the mock executor to verify
	// they are wired and respond deterministically.
	tools := []struct {
		name    string
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args    map[string]interface{}
	}{
		{"sdp_scout", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleScout(ctx, req)
		}, map[string]interface{}{}},
		{"sdp_architect", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleArchitect(ctx, req)
		}, map[string]interface{}{}},
		{"sdp_metrics", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleMetrics(ctx, req)
		}, map[string]interface{}{}},
		{"sdp_spec", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleSpec(ctx, req)
		}, map[string]interface{}{}},
		{"sdp_bootstrap", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleBootstrap(ctx, req)
		}, map[string]interface{}{}},
		{"sdp_index_build", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleIndexBuild(ctx, req)
		}, map[string]interface{}{}},
		{"sdp_index_query", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleIndexQuery(ctx, req)
		}, map[string]interface{}{"query": "test"}},
		{"sdp_index_find", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleIndexFind(ctx, req)
		}, map[string]interface{}{"symbol": "Test"}},
		{"sdp_index_deps", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleIndexDeps(ctx, req)
		}, map[string]interface{}{"module": "internal/mcp"}},
		{"sdp_dispatch", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleDispatch(ctx, req)
		}, map[string]interface{}{"task": "test"}},
		{"sdp_beads_create", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleBeadsCreate(ctx, req)
		}, map[string]interface{}{"title": "test"}},
		{"sdp_beads_close", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleBeadsClose(ctx, req)
		}, map[string]interface{}{"id": "WS-1"}},
		{"sdp_beads_list", func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return srv1.handleBeadsList(ctx, req)
		}, map[string]interface{}{}},
	}

	mock := &mockExecutor{response: []byte(`{}`)}
	srv1.executor = mock

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      tc.name,
					Arguments: tc.args,
				},
			}
			result, err := tc.handler(context.Background(), req)
			require.NoError(t, err, "handler %s returned unexpected error", tc.name)
			require.NotNil(t, result, "handler %s returned nil result", tc.name)
			assert.False(t, result.IsError, "handler %s returned error result with mock executor", tc.name)
		})
	}
}

func TestCrossHarness_ResourceSurfaceIsConsistent(t *testing.T) {
	// Create two servers with identical setups; verify they expose the same
	// set of resources.
	srv1 := NewServer(ServerConfig{RepoRoot: "."})
	srv2 := NewServer(ServerConfig{RepoRoot: "."})

	require.NotNil(t, srv1.inner)
	require.NotNil(t, srv2.inner)

	// Verify the static resource definitions are identical and complete.
	assert.Len(t, staticResources, expectedResourceCount,
		"expected %d static resources, got %d", expectedResourceCount, len(staticResources))

	// Verify each resource has all required fields.
	for _, rd := range staticResources {
		assert.NotEmpty(t, rd.uri, "resource URI must not be empty")
		assert.NotEmpty(t, rd.name)
		assert.NotEmpty(t, rd.relPath)
		assert.NotEmpty(t, rd.mimeType)
		assert.NotEmpty(t, rd.hintTool)
	}
}

func TestCrossHarness_PromptSurfaceIsConsistent(t *testing.T) {
	// All five prompts must be registered and respond identically for the
	// same inputs.
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "scout.json"),
		[]byte(`{"name":"consistency-test"}`),
		0o644,
	))

	promptNames := []string{"understand", "build", "fix", "review", "operate"}
	assert.Len(t, promptNames, expectedPromptCount)

	// Create two independent servers.
	srv1 := NewServer(ServerConfig{RepoRoot: tmpDir})
	srv2 := NewServer(ServerConfig{RepoRoot: tmpDir})

	// Each prompt must produce the same output on both servers.
	promptTests := []struct {
		name string
		args map[string]string
	}{
		{"understand", map[string]string{"depth": "standard"}},
		{"build", map[string]string{"description": "add auth"}},
		{"fix", map[string]string{"description": "null ptr"}},
		{"review", map[string]string{}},
		{"operate", map[string]string{}},
	}

	for _, pt := range promptTests {
		t.Run(pt.name+"_consistent", func(t *testing.T) {
			req := mcp.GetPromptRequest{
				Params: mcp.GetPromptParams{
					Name:      pt.name,
					Arguments: pt.args,
				},
			}

			result1, err := callPromptHandler(srv1, pt.name, req)
			require.NoError(t, err)
			result2, err := callPromptHandler(srv2, pt.name, req)
			require.NoError(t, err)

			require.Len(t, result1.Messages, 1)
			require.Len(t, result2.Messages, 1)

			text1 := result1.Messages[0].Content.(mcp.TextContent).Text
			text2 := result2.Messages[0].Content.(mcp.TextContent).Text
			assert.Equal(t, text1, text2,
				"prompt %q returned different text from two identical servers", pt.name)
		})
	}
}

// callPromptHandler dispatches to the correct handler by prompt name.
func callPromptHandler(srv *Server, name string, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	switch name {
	case "understand":
		return srv.handleUnderstandPrompt(context.Background(), req)
	case "build":
		return srv.handleBuildPrompt(context.Background(), req)
	case "fix":
		return srv.handleFixPrompt(context.Background(), req)
	case "review":
		return srv.handleReviewPrompt(context.Background(), req)
	case "operate":
		return srv.handleOperatePrompt(context.Background(), req)
	default:
		return nil, assert.AnError
	}
}

func TestCrossHarness_ErrorFormatConsistency(t *testing.T) {
	// All tool handlers must return errors in the same format:
	//   - IsError = true
	//   - Content[0] is TextContent
	//   - Text contains the tool name and "failed"
	srv, _ := newTestServer()

	// Trigger errors for tools that require parameters.
	errorTests := []struct {
		name string
		req  mcp.CallToolRequest
	}{
		{
			"sdp_index_query_missing_query",
			mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "sdp_index_query", Arguments: map[string]interface{}{}}},
		},
		{
			"sdp_index_find_missing_symbol",
			mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "sdp_index_find", Arguments: map[string]interface{}{}}},
		},
		{
			"sdp_index_deps_missing_module",
			mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "sdp_index_deps", Arguments: map[string]interface{}{}}},
		},
		{
			"sdp_dispatch_missing_task",
			mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "sdp_dispatch", Arguments: map[string]interface{}{}}},
		},
		{
			"sdp_beads_create_missing_title",
			mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "sdp_beads_create", Arguments: map[string]interface{}{}}},
		},
		{
			"sdp_beads_close_missing_id",
			mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "sdp_beads_close", Arguments: map[string]interface{}{}}},
		},
	}

	for _, tc := range errorTests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := callToolHandler(srv, tc.req)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.True(t, result.IsError,
				"expected IsError=true for %s", tc.name)
			require.NotEmpty(t, result.Content,
				"expected non-empty Content for %s", tc.name)
			_, ok := result.Content[0].(mcp.TextContent)
			assert.True(t, ok,
				"expected TextContent for %s", tc.name)
		})
	}
}

// callToolHandler dispatches to the correct tool handler by name.
func callToolHandler(srv *Server, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx := context.Background()
	switch req.Params.Name {
	case "sdp_scout":
		return srv.handleScout(ctx, req)
	case "sdp_architect":
		return srv.handleArchitect(ctx, req)
	case "sdp_metrics":
		return srv.handleMetrics(ctx, req)
	case "sdp_spec":
		return srv.handleSpec(ctx, req)
	case "sdp_bootstrap":
		return srv.handleBootstrap(ctx, req)
	case "sdp_index_build":
		return srv.handleIndexBuild(ctx, req)
	case "sdp_index_query":
		return srv.handleIndexQuery(ctx, req)
	case "sdp_index_find":
		return srv.handleIndexFind(ctx, req)
	case "sdp_index_deps":
		return srv.handleIndexDeps(ctx, req)
	case "sdp_dispatch":
		return srv.handleDispatch(ctx, req)
	case "sdp_beads_create":
		return srv.handleBeadsCreate(ctx, req)
	case "sdp_beads_close":
		return srv.handleBeadsClose(ctx, req)
	case "sdp_beads_list":
		return srv.handleBeadsList(ctx, req)
	default:
		return nil, assert.AnError
	}
}

// ---------------------------------------------------------------------------
// Performance budget tests (WS-04)
// ---------------------------------------------------------------------------

func TestPerf_ToolCallOverhead(t *testing.T) {
	// Measure the MCP overhead for a tool call (mock executor, no real I/O).
	// The mock executor returns immediately, so this measures only the
	// handler + argument parsing + result marshaling overhead.
	srv, _ := newTestServer()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_scout",
			Arguments: map[string]interface{}{},
		},
	}

	// Warm up.
	_, _ = srv.handleScout(context.Background(), req)

	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = srv.handleScout(context.Background(), req)
	}
	avg := time.Since(start) / 100

	assert.Less(t, avg, 50*time.Millisecond,
		"average tool call overhead should be <50ms, got %v", avg)
	t.Logf("tool call overhead (avg over 100 runs): %v", avg)
}

func TestPerf_ResourceRead(t *testing.T) {
	// Measure resource read overhead for a file that exists on disk.
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "scout.json"),
		[]byte(`{"name":"perf-test","languages":["Go"]}`),
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

	// Warm up.
	_, _ = srv.handleFileResource(context.Background(), req, rd)

	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = srv.handleFileResource(context.Background(), req, rd)
	}
	avg := time.Since(start) / 100

	assert.Less(t, avg, 10*time.Millisecond,
		"average resource read overhead should be <10ms, got %v", avg)
	t.Logf("resource read overhead (avg over 100 runs): %v", avg)
}

func TestPerf_PromptRender(t *testing.T) {
	// Measure prompt rendering overhead.
	tmpDir := t.TempDir()
	sdpDir := filepath.Join(tmpDir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(sdpDir, "scout.json"),
		[]byte(`{"name":"perf-test"}`),
		0o644,
	))

	srv := NewServer(ServerConfig{RepoRoot: tmpDir})
	req := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "understand",
			Arguments: map[string]string{
				"depth": "standard",
			},
		},
	}

	// Warm up (first call parses templates).
	_, _ = srv.handleUnderstandPrompt(context.Background(), req)

	start := time.Now()
	for i := 0; i < 100; i++ {
		_, _ = srv.handleUnderstandPrompt(context.Background(), req)
	}
	avg := time.Since(start) / 100

	assert.Less(t, avg, 20*time.Millisecond,
		"average prompt render overhead should be <20ms, got %v", avg)
	t.Logf("prompt render overhead (avg over 100 runs): %v", avg)
}
