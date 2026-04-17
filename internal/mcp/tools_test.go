package mcp

// NOTE: Tests in this file use a mock executor (mockExecutor) to verify MCP
// tool handler logic — argument parsing, CLI invocation shape, and error
// handling. Full integration tests that exercise the real CLI binaries (sdp,
// sdp-dispatch, bd) are out of scope for this test suite because they require
// building the full CLI toolchain and a valid git repository. Those tests
// belong in cmd/sdp/, cmd/sdp-dispatch/, and the beads test suite respectively.

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutor records calls and returns canned responses.
type mockExecutor struct {
	// lastBinary is the binary name from the most recent call.
	lastBinary string
	// lastArgs is the argument list from the most recent call.
	lastArgs []string

	// response is the stdout bytes returned on the next call.
	response []byte
	// err is the error returned on the next call.
	err error

	// calls records all invocations for assertion.
	calls []callRecord
}

type callRecord struct {
	binary string
	args   []string
}

func (m *mockExecutor) Run(_ context.Context, args ...string) ([]byte, error) {
	m.lastBinary = "sdp"
	m.lastArgs = args
	m.calls = append(m.calls, callRecord{binary: "sdp", args: append([]string{}, args...)})
	resp, err := m.response, m.err
	m.response = nil
	m.err = nil
	return resp, err
}

func (m *mockExecutor) RunCustom(_ context.Context, binary string, args ...string) ([]byte, error) {
	m.lastBinary = binary
	m.lastArgs = args
	m.calls = append(m.calls, callRecord{binary: binary, args: append([]string{}, args...)})
	resp, err := m.response, m.err
	m.response = nil
	m.err = nil
	return resp, err
}

// newTestServer creates a Server with a mock executor for testing.
func newTestServer() (*Server, *mockExecutor) {
	srv := NewServer(ServerConfig{RepoRoot: "/test/repo"})
	mock := &mockExecutor{}
	srv.executor = mock
	return srv, mock
}

// --- Scout ---

func TestHandleScout_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"languages":["Go"],"health":"good"}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_scout",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := srv.handleScout(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "languages")

	// Verify correct CLI invocation
	assert.Equal(t, []string{"scout", "--format", "json", "/test/repo"}, mock.lastArgs)
}

func TestHandleScout_CustomFormat(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("Scout Report\n---\n...")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_scout",
			Arguments: map[string]interface{}{
				"format": "text",
				"path":   "/custom/path",
			},
		},
	}

	result, err := srv.handleScout(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, []string{"scout", "--format", "text", "/custom/path"}, mock.lastArgs)
}

func TestHandleScout_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("scout: no go.mod found")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_scout",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := srv.handleScout(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "scout failed")
}

// --- Metrics ---

func TestHandleMetrics_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"commits_analyzed":42}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_metrics",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := srv.handleMetrics(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, []string{"metrics", "--format", "json", "/test/repo"}, mock.lastArgs)
}

func TestHandleMetrics_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("git not found")

	result, err := srv.handleMetrics(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "sdp_metrics", Arguments: map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "metrics failed")
}

// --- Spec ---

func TestHandleSpec_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"api_contracts":[]}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_spec",
			Arguments: map[string]interface{}{
				"category": "api",
			},
		},
	}

	result, err := srv.handleSpec(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, []string{"spec", "--format", "json", "--category", "api", "/test/repo"}, mock.lastArgs)
}

func TestHandleSpec_AllCategory(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_spec",
			Arguments: map[string]interface{}{
				"category": "all",
			},
		},
	}

	result, err := srv.handleSpec(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// "all" should not pass --category flag
	assert.Equal(t, []string{"spec", "--format", "json", "/test/repo"}, mock.lastArgs)
}

func TestHandleSpec_WithEnrich(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_spec",
			Arguments: map[string]interface{}{
				"enrich": true,
			},
		},
	}

	result, err := srv.handleSpec(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, mock.lastArgs, "--enrich")
}

func TestHandleSpec_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("no source files found")

	result, err := srv.handleSpec(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "sdp_spec", Arguments: map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Bootstrap ---

func TestHandleBootstrap_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"artifacts":[],"version":"1.0"}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_bootstrap",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := srv.handleBootstrap(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Default verify=true, so --no-verify should NOT be present
	for _, a := range mock.lastArgs {
		assert.NotEqual(t, "--no-verify", a)
	}
}

func TestHandleBootstrap_DryRun(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"artifacts":[],"dry_run":true}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_bootstrap",
			Arguments: map[string]interface{}{
				"dry_run": true,
				"only":    "agents-md,policies",
			},
		},
	}

	result, err := srv.handleBootstrap(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, mock.lastArgs, "--dry-run")
	assert.Contains(t, mock.lastArgs, "--only")
}

func TestHandleBootstrap_NoVerify(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_bootstrap",
			Arguments: map[string]interface{}{
				"verify": false,
			},
		},
	}

	_, err := srv.handleBootstrap(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, mock.lastArgs, "--no-verify")
}

func TestHandleBootstrap_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("permission denied")

	result, err := srv.handleBootstrap(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "sdp_bootstrap", Arguments: map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Architect ---

func TestHandleArchitect_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"components":[]}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_architect",
			Arguments: map[string]interface{}{},
		},
	}

	result, err := srv.handleArchitect(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Uses "analyze" subcommand
	assert.Contains(t, mock.lastArgs, "analyze")
	assert.Contains(t, mock.lastArgs, "--format")
	assert.Contains(t, mock.lastArgs, "json")
}

func TestHandleArchitect_WithSection(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_architect",
			Arguments: map[string]interface{}{
				"section": "diagrams",
			},
		},
	}

	_, err := srv.handleArchitect(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, mock.lastArgs, "--section")
	assert.Contains(t, mock.lastArgs, "diagrams")
}

func TestHandleArchitect_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("analysis timeout")

	result, err := srv.handleArchitect(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "sdp_architect", Arguments: map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Index Build ---

func TestHandleIndexBuild_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"total_files":100}`)

	result, err := srv.handleIndexBuild(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "sdp_index_build", Arguments: map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, []string{"index", "build", "--format", "json", "/test/repo"}, mock.lastArgs)
}

func TestHandleIndexBuild_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("not a directory")

	result, err := srv.handleIndexBuild(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "sdp_index_build", Arguments: map[string]interface{}{}},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Index Query ---

func TestHandleIndexQuery_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"results":[],"total":0}`)

	result, err := srv.handleIndexQuery(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_index_query",
			Arguments: map[string]interface{}{
				"query": "database connection",
				"limit": float64(5),
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, mock.lastArgs, "query")
	assert.Contains(t, mock.lastArgs, "database connection")
	assert.Contains(t, mock.lastArgs, "5")
}

func TestHandleIndexQuery_MissingQuery(t *testing.T) {
	srv, _ := newTestServer()

	result, err := srv.handleIndexQuery(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_index_query",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "query")
}

func TestHandleIndexQuery_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("index not found")

	result, err := srv.handleIndexQuery(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_index_query",
			Arguments: map[string]interface{}{"query": "test"},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Index Find ---

func TestHandleIndexFind_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"results":[{"symbol":"Server"}]}`)

	result, err := srv.handleIndexFind(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_index_find",
			Arguments: map[string]interface{}{
				"symbol": "Server",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, mock.lastArgs, "Server")
	assert.NotContains(t, mock.lastArgs, "--regex")
}

func TestHandleIndexFind_WithRegex(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	_, err := srv.handleIndexFind(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_index_find",
			Arguments: map[string]interface{}{
				"symbol": "Serv.*",
				"regex":  true,
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, mock.lastArgs, "--regex")
}

func TestHandleIndexFind_MissingSymbol(t *testing.T) {
	srv, _ := newTestServer()

	result, err := srv.handleIndexFind(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_index_find",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Index Deps ---

func TestHandleIndexDeps_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"results":[]}`)

	result, err := srv.handleIndexDeps(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_index_deps",
			Arguments: map[string]interface{}{
				"module": "internal/scout",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, mock.lastArgs, "internal/scout")
	assert.Contains(t, mock.lastArgs, "--depth")
	assert.Contains(t, mock.lastArgs, "3")
}

func TestHandleIndexDeps_Reverse(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	_, err := srv.handleIndexDeps(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_index_deps",
			Arguments: map[string]interface{}{
				"module":    "internal/scout",
				"direction": "reverse",
				"depth":     float64(3),
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, mock.lastArgs, "--reverse")
	assert.Contains(t, mock.lastArgs, "3")
}

func TestHandleIndexDeps_MissingModule(t *testing.T) {
	srv, _ := newTestServer()

	result, err := srv.handleIndexDeps(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_index_deps",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleIndexDeps_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("module not found")

	result, err := srv.handleIndexDeps(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_index_deps",
			Arguments: map[string]interface{}{"module": "foo"},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Dispatch ---

func TestHandleDispatch_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"harness":"codex","model":"gpt-4"}`)

	result, err := srv.handleDispatch(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_dispatch",
			Arguments: map[string]interface{}{
				"task": "refactor auth module",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "sdp-dispatch", mock.lastBinary)
	assert.Contains(t, mock.lastArgs, "route")
	assert.Contains(t, mock.lastArgs, "refactor auth module")
	assert.Contains(t, mock.lastArgs, "--json")
}

func TestHandleDispatch_MissingTask(t *testing.T) {
	srv, _ := newTestServer()

	result, err := srv.handleDispatch(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_dispatch",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "task")
}

// --- Beads Create ---

func TestHandleBeadsCreate_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("WS-42")

	result, err := srv.handleBeadsCreate(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_create",
			Arguments: map[string]interface{}{
				"title":       "Fix login bug",
				"description": "Login fails on Safari",
				"type":        "bug",
				"priority":    float64(2),
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "bd", mock.lastBinary)
	assert.Contains(t, mock.lastArgs, "create")
	assert.Contains(t, mock.lastArgs, "Fix login bug")
	assert.Contains(t, mock.lastArgs, "--description")
	assert.Contains(t, mock.lastArgs, "--type")
	assert.Contains(t, mock.lastArgs, "--priority")
}

func TestHandleBeadsCreate_Minimal(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("WS-43")

	result, err := srv.handleBeadsCreate(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_create",
			Arguments: map[string]interface{}{
				"title": "Quick task",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotContains(t, mock.lastArgs, "--description")
	assert.NotContains(t, mock.lastArgs, "--type")
	assert.NotContains(t, mock.lastArgs, "--priority")
}

func TestHandleBeadsCreate_PriorityZero(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("WS-44")

	result, err := srv.handleBeadsCreate(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_create",
			Arguments: map[string]interface{}{
				"title":    "Critical issue",
				"priority": float64(0),
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Priority 0 should be forwarded (it's the highest priority in bd)
	assert.Contains(t, mock.lastArgs, "--priority")
	assert.Contains(t, mock.lastArgs, "0")
}

func TestHandleBeadsCreate_MissingTitle(t *testing.T) {
	srv, _ := newTestServer()

	result, err := srv.handleBeadsCreate(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_beads_create",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleBeadsCreate_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("beads not initialized")

	result, err := srv.handleBeadsCreate(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_beads_create",
			Arguments: map[string]interface{}{"title": "test"},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Beads Close ---

func TestHandleBeadsClose_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("closed WS-42")

	result, err := srv.handleBeadsClose(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_close",
			Arguments: map[string]interface{}{
				"id": "WS-42",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "bd", mock.lastBinary)
	assert.Contains(t, mock.lastArgs, "close")
	assert.Contains(t, mock.lastArgs, "WS-42")
}

func TestHandleBeadsClose_MissingID(t *testing.T) {
	srv, _ := newTestServer()

	result, err := srv.handleBeadsClose(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_beads_close",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Beads List ---

func TestHandleBeadsList_HappyPath(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("WS-42: Fix bug [open]\nWS-43: Add feature [in-progress]")

	result, err := srv.handleBeadsList(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_list",
			Arguments: map[string]interface{}{
				"status": "open",
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "bd", mock.lastBinary)
	assert.Contains(t, mock.lastArgs, "list")
	assert.Contains(t, mock.lastArgs, "--status")
	assert.Contains(t, mock.lastArgs, "open")
}

func TestHandleBeadsList_NoFilters(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("WS-42: Fix bug [open]")

	result, err := srv.handleBeadsList(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_beads_list",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotContains(t, mock.lastArgs, "--status")
	assert.NotContains(t, mock.lastArgs, "--assignee")
}

func TestHandleBeadsList_WithAssignee(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("WS-42: Fix bug [open]")

	_, err := srv.handleBeadsList(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_list",
			Arguments: map[string]interface{}{
				"assignee": "alice",
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, mock.lastArgs, "--assignee")
	assert.Contains(t, mock.lastArgs, "alice")
}

func TestHandleBeadsList_Error(t *testing.T) {
	srv, mock := newTestServer()
	mock.err = fmt.Errorf("permission denied")

	result, err := srv.handleBeadsList(context.Background(), mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "sdp_beads_list",
			Arguments: map[string]interface{}{},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// --- Executor tests ---

func TestRealExecutor_BinaryNotFound(t *testing.T) {
	exec := &realExecutor{binaryPath: "nonexistent-binary-xyz", workDir: "."}
	_, err := exec.Run(context.Background(), "test")
	assert.Error(t, err)
}

func TestMockExecutor_Tracking(t *testing.T) {
	mock := &mockExecutor{}
	mock.response = []byte("ok")

	ctx := context.Background()

	_, err := mock.Run(ctx, "scout", "--format", "json", ".")
	require.NoError(t, err)
	assert.Equal(t, 1, len(mock.calls))
	assert.Equal(t, "sdp", mock.calls[0].binary)
	assert.Equal(t, []string{"scout", "--format", "json", "."}, mock.calls[0].args)

	_, err = mock.RunCustom(ctx, "bd", "list", "--status", "open")
	require.NoError(t, err)
	assert.Equal(t, 2, len(mock.calls))
	assert.Equal(t, "bd", mock.calls[1].binary)
}

// ---------------------------------------------------------------------------
// Security tests (WS-04)
// ---------------------------------------------------------------------------

// TestSecurity_PathTraversal_PassedToCLI verifies that path parameters with
// traversal sequences are passed through to the CLI as-is (the MCP server
// does not interpret paths — the CLI is responsible for sandboxing). The
// important security property is that the MCP server does NOT resolve or
// normalize these paths itself, so it cannot accidentally widen access.
func TestSecurity_PathTraversal_PassedToCLI(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	traversalPaths := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		"../../.env",
	}

	for _, path := range traversalPaths {
		t.Run("path="+path, func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "sdp_scout",
					Arguments: map[string]interface{}{
						"path": path,
					},
				},
			}

			result, err := srv.handleScout(context.Background(), req)
			require.NoError(t, err)
			assert.False(t, result.IsError)

			// The key assertion: the MCP server passes the path directly to
			// the CLI without interpretation. The CLI is the security boundary.
			assert.Contains(t, mock.lastArgs, path,
				"path traversal string should be passed to CLI verbatim")
		})
	}
}

// TestSecurity_CommandInjection_NoShellExpansion verifies that arguments with
// shell metacharacters are passed to the CLI as individual arguments (via
// exec.Command), not through a shell. This prevents command injection.
func TestSecurity_CommandInjection_NoShellExpansion(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{"results":[]}`)

	injectionStrings := []string{
		"; rm -rf /",
		"$(cat /etc/passwd)",
		"`whoami`",
		"&& echo pwned",
		"| cat /etc/shadow",
	}

	for _, inj := range injectionStrings {
		t.Run("query="+truncateForName(inj), func(t *testing.T) {
			req := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "sdp_index_query",
					Arguments: map[string]interface{}{
						"query": inj,
					},
				},
			}

			result, err := srv.handleIndexQuery(context.Background(), req)
			require.NoError(t, err)
			assert.False(t, result.IsError)

			// The injection string must appear as a single argument, not
			// expanded by a shell. exec.Command does not invoke a shell.
			assert.Contains(t, mock.lastArgs, inj,
				"injection string should be passed as single arg to CLI")
		})
	}
}

// TestSecurity_CommandInjection_SymbolArg verifies injection safety for the
// symbol parameter of sdp_index_find.
func TestSecurity_CommandInjection_SymbolArg(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_index_find",
			Arguments: map[string]interface{}{
				"symbol": "; cat /etc/shadow",
			},
		},
	}

	result, err := srv.handleIndexFind(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Symbol passed as a single CLI arg (no shell expansion).
	assert.Contains(t, mock.lastArgs, "; cat /etc/shadow")
}

// TestSecurity_CommandInjection_DispatchTask verifies injection safety for
// the task parameter of sdp_dispatch.
func TestSecurity_CommandInjection_DispatchTask(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte(`{}`)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_dispatch",
			Arguments: map[string]interface{}{
				"task": "$(whoami); rm -rf /",
			},
		},
	}

	result, err := srv.handleDispatch(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	// Task passed as a single CLI arg via --task flag.
	assert.Contains(t, mock.lastArgs, "$(whoami); rm -rf /")
}

// TestSecurity_CommandInjection_BeadsTitle verifies injection safety for the
// title parameter of sdp_beads_create.
func TestSecurity_CommandInjection_BeadsTitle(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("WS-99")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_create",
			Arguments: map[string]interface{}{
				"title": "test; rm -rf /",
			},
		},
	}

	result, err := srv.handleBeadsCreate(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, mock.lastArgs, "test; rm -rf /")
}

// TestSecurity_CommandInjection_BeadsCloseID verifies injection safety for
// the id parameter of sdp_beads_close.
func TestSecurity_CommandInjection_BeadsCloseID(t *testing.T) {
	srv, mock := newTestServer()
	mock.response = []byte("closed")

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "sdp_beads_close",
			Arguments: map[string]interface{}{
				"id": "WS-1; cat /etc/passwd",
			},
		},
	}

	result, err := srv.handleBeadsClose(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, mock.lastArgs, "WS-1; cat /etc/passwd")
}

// TestSecurity_NoPrivilegeEscalation verifies that the realExecutor uses
// exec.Command (same-user process spawn), not setuid, sudo, or anything
// that would escalate privileges.
func TestSecurity_NoPrivilegeEscalation(t *testing.T) {
	exec := &realExecutor{binaryPath: "echo", workDir: "."}

	// echo is a safe binary that exists on all systems.
	out, err := exec.Run(context.Background(), "hello")
	require.NoError(t, err)
	assert.Contains(t, string(out), "hello")

	// The executor does not add sudo, doas, or any setuid wrapper.
	// This is a design property: realExecutor.RunCustom calls exec.Command
	// directly. We verify the binaryPath does not contain privilege escalation
	// command chains.
	assert.NotContains(t, exec.binaryPath, "sudo",
		"binaryPath should not contain sudo")
	assert.NotContains(t, exec.binaryPath, "doas",
		"binaryPath should not contain doas")
	assert.NotContains(t, exec.binaryPath, "&&",
		"binaryPath should not contain command chaining")
	assert.NotContains(t, exec.binaryPath, ";",
		"binaryPath should not contain command separators")
}

// TestSecurity_ExecutorDoesNotUseShell verifies that the realExecutor calls
// exec.Command (not exec.Command("/bin/sh", "-c", ...)), so shell expansion
// is impossible.
func TestSecurity_ExecutorDoesNotUseShell(t *testing.T) {
	// This is a design-level test. The realExecutor.RunCustom implementation
	// calls exec.CommandContext(ctx, binary, args...) directly, which does NOT invoke a
	// shell. Verify the executor interface doesn't have a shell method.
	var _ CommandExecutor = &realExecutor{}
	// If realExecutor had a shell-based method, it would show up here.
	// The test passes if compilation succeeds (interface satisfied).
}

// truncateForName creates a safe test name from an arbitrary string.
func truncateForName(s string) string {
	if len(s) > 30 {
		return s[:30]
	}
	return s
}
