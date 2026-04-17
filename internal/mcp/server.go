// Package mcp implements an MCP (Model Context Protocol) server that exposes
// SDP toolkit commands as MCP tools. The server wraps the sdp CLI binary
// rather than importing business logic directly.
package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	ServerName    = "sdp-mcp"
	ServerVersion = "0.1.0"

	// DefaultBinary is the default CLI binary name looked up in PATH.
	DefaultBinary = "sdp"
)

// ServerConfig holds configuration for the MCP server.
type ServerConfig struct {
	// BinaryPath is the path to the sdp CLI binary.
	// If empty, the server uses DefaultBinary and relies on PATH lookup.
	BinaryPath string

	// RepoRoot is the default repository root passed to tools via --repo or as
	// a positional argument. Defaults to ".".
	RepoRoot string
}

// Server is the MCP server wrapping SDP CLI commands.
type Server struct {
	config   ServerConfig
	inner    *mcpserver.MCPServer
	executor CommandExecutor
}

// NewServer creates a new MCP server with all SDP tools registered.
func NewServer(cfg ServerConfig) *Server {
	if cfg.RepoRoot == "" {
		cfg.RepoRoot = "."
	}
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = DefaultBinary
	}

	inner := mcpserver.NewMCPServer(
		ServerName,
		ServerVersion,
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, true),
		mcpserver.WithPromptCapabilities(true),
	)

	s := &Server{
		config:   cfg,
		inner:    inner,
		executor: &realExecutor{binaryPath: cfg.BinaryPath, workDir: cfg.RepoRoot},
	}

	s.registerTools()
	s.registerResources()
	s.registerPrompts()
	return s
}

// Inner returns the underlying MCP server for use with transport functions.
func (s *Server) Inner() *mcpserver.MCPServer {
	return s.inner
}

// ToolCount returns the number of registered tools. Intended for testing.
func (s *Server) ToolCount() int {
	return len(s.inner.ListTools())
}

// registerTools registers all SDP MCP tools with the server.
func (s *Server) registerTools() {
	s.registerScout()
	s.registerArchitect()
	s.registerMetrics()
	s.registerSpec()
	s.registerBootstrap()
	s.registerIndexBuild()
	s.registerIndexQuery()
	s.registerIndexFind()
	s.registerIndexDeps()
	s.registerDispatch()
	s.registerBeadsCreate()
	s.registerBeadsClose()
	s.registerBeadsList()
}

// --- Scout ---

func (s *Server) registerScout() {
	tool := mcp.NewTool("sdp_scout",
		mcp.WithDescription("Quick 30s codebase reconnaissance. Returns a snapshot of languages, frameworks, structure, and health signals."),
		mcp.WithString("path",
			mcp.Description("Repository root path (default: server --repo)"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: json, text, or card"),
			mcp.Enum("json", "text", "card"),
		),
	)
	s.inner.AddTool(tool, s.handleScout)
}

func (s *Server) handleScout(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := s.repoPath(req.GetString("path", ""))
	format := req.GetString("format", "json")

	out, err := s.executor.Run(ctx, "scout", "--format", format, path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("scout failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Architect ---

func (s *Server) registerArchitect() {
	tool := mcp.NewTool("sdp_architect",
		mcp.WithDescription("Deep architecture analysis of a codebase. Returns C4 models, dependency graphs, and quality metrics."),
		mcp.WithString("path",
			mcp.Description("Repository root path (default: server --repo)"),
		),
		mcp.WithString("section",
			mcp.Description("Output section filter: profile, report, model, diagrams, summary (default: all)"),
			mcp.Enum("profile", "report", "model", "diagrams", "summary"),
		),
	)
	s.inner.AddTool(tool, s.handleArchitect)
}

func (s *Server) handleArchitect(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := s.repoPath(req.GetString("path", ""))
	section := req.GetString("section", "")

	args := []string{"architect", "analyze", "--format", "json"}
	if section != "" {
		args = append(args, "--section", section)
	}
	args = append(args, path)

	out, err := s.executor.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("architect failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Metrics ---

func (s *Server) registerMetrics() {
	tool := mcp.NewTool("sdp_metrics",
		mcp.WithDescription("Git-derived process health metrics: activity, hygiene, release patterns, and team dynamics."),
		mcp.WithString("path",
			mcp.Description("Repository root path (default: server --repo)"),
		),
		mcp.WithString("format",
			mcp.Description("Output format: json, text, or markdown"),
			mcp.Enum("json", "text", "markdown"),
		),
	)
	s.inner.AddTool(tool, s.handleMetrics)
}

func (s *Server) handleMetrics(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := s.repoPath(req.GetString("path", ""))
	format := req.GetString("format", "json")

	out, err := s.executor.Run(ctx, "metrics", "--format", format, path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("metrics failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Spec ---

func (s *Server) registerSpec() {
	tool := mcp.NewTool("sdp_spec",
		mcp.WithDescription("Specification recovery: extract API contracts, business rules, invariants, and SLA parameters from code."),
		mcp.WithString("path",
			mcp.Description("Repository root path (default: server --repo)"),
		),
		mcp.WithString("category",
			mcp.Description("Filter category: all, api, rules, invariants, sla"),
			mcp.Enum("all", "api", "rules", "invariants", "sla"),
		),
		mcp.WithBoolean("enrich",
			mcp.Description("Enable LLM enrichment (opt-in)"),
		),
	)
	s.inner.AddTool(tool, s.handleSpec)
}

func (s *Server) handleSpec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := s.repoPath(req.GetString("path", ""))
	category := req.GetString("category", "all")
	enrich := req.GetBool("enrich", false)

	args := []string{"spec", "--format", "json"}
	if category != "all" {
		args = append(args, "--category", category)
	}
	if enrich {
		args = append(args, "--enrich")
	}
	args = append(args, path)

	out, err := s.executor.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("spec failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Bootstrap ---

func (s *Server) registerBootstrap() {
	tool := mcp.NewTool("sdp_bootstrap",
		mcp.WithDescription("Generate agent-ready setup artifacts (AGENTS.md, hooks, policies) for a repository."),
		mcp.WithString("path",
			mcp.Description("Repository root path (default: server --repo)"),
		),
		mcp.WithString("only",
			mcp.Description("Generate only these artifacts (comma-separated: claude-md, agents-md, policies, hooks, beads)"),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description("Show what would be generated without writing"),
		),
		mcp.WithBoolean("verify",
			mcp.Description("Run build/test/lint verification after generation (default: true)"),
		),
	)
	s.inner.AddTool(tool, s.handleBootstrap)
}

func (s *Server) handleBootstrap(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := s.repoPath(req.GetString("path", ""))
	only := req.GetString("only", "")
	dryRun := req.GetBool("dry_run", false)
	verify := req.GetBool("verify", true)

	args := []string{"bootstrap", "--format", "json"}
	if only != "" && only != "all" {
		args = append(args, "--only", only)
	}
	if dryRun {
		args = append(args, "--dry-run")
	}
	if !verify {
		args = append(args, "--no-verify")
	}
	args = append(args, path)

	out, err := s.executor.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("bootstrap failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Index Build ---

func (s *Server) registerIndexBuild() {
	tool := mcp.NewTool("sdp_index_build",
		mcp.WithDescription("Build or refresh the codebase index for semantic search and dependency queries."),
		mcp.WithString("path",
			mcp.Description("Repository root path (default: server --repo)"),
		),
	)
	s.inner.AddTool(tool, s.handleIndexBuild)
}

func (s *Server) handleIndexBuild(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := s.repoPath(req.GetString("path", ""))

	out, err := s.executor.Run(ctx, "index", "build", "--format", "json", path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("index build failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Index Query ---

func (s *Server) registerIndexQuery() {
	tool := mcp.NewTool("sdp_index_query",
		mcp.WithDescription("Semantic search across the indexed codebase using full-text search."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query string"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 10)"),
		),
	)
	s.inner.AddTool(tool, s.handleIndexQuery)
}

func (s *Server) handleIndexQuery(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path := s.repoPath(req.GetString("path", ""))
	limit := req.GetInt("limit", 10)

	out, execErr := s.executor.Run(ctx, "index", "query", "--format", "json", "--limit", fmt.Sprintf("%d", limit), path, query)
	if execErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("index query failed: %v", execErr)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Index Find ---

func (s *Server) registerIndexFind() {
	tool := mcp.NewTool("sdp_index_find",
		mcp.WithDescription("Fast symbol/keyword search in the indexed codebase."),
		mcp.WithString("symbol",
			mcp.Required(),
			mcp.Description("Symbol or keyword to search for"),
		),
		mcp.WithBoolean("regex",
			mcp.Description("Treat the symbol as a regex pattern (default: false)"),
		),
	)
	s.inner.AddTool(tool, s.handleIndexFind)
}

func (s *Server) handleIndexFind(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	symbol, err := req.RequireString("symbol")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path := s.repoPath(req.GetString("path", ""))
	regex := req.GetBool("regex", false)

	args := []string{"index", "find", "--format", "json"}
	if regex {
		args = append(args, "--regex")
	}
	args = append(args, path, symbol)

	out, execErr := s.executor.Run(ctx, args...)
	if execErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("index find failed: %v", execErr)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Index Deps ---

func (s *Server) registerIndexDeps() {
	tool := mcp.NewTool("sdp_index_deps",
		mcp.WithDescription("Dependency graph queries: forward or reverse dependency traversal."),
		mcp.WithString("module",
			mcp.Required(),
			mcp.Description("Module path to query dependencies for"),
		),
		mcp.WithString("direction",
			mcp.Description("Traversal direction: forward or reverse (default: forward)"),
			mcp.Enum("forward", "reverse"),
		),
		mcp.WithNumber("depth",
			mcp.Description("Maximum traversal depth (default: 1)"),
		),
	)
	s.inner.AddTool(tool, s.handleIndexDeps)
}

func (s *Server) handleIndexDeps(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	module, err := req.RequireString("module")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path := s.repoPath(req.GetString("path", ""))
	direction := req.GetString("direction", "forward")
	depth := req.GetInt("depth", 1)

	args := []string{"index", "deps", "--format", "json", "--depth", fmt.Sprintf("%d", depth)}
	if direction == "reverse" {
		args = append(args, "--reverse")
	}
	args = append(args, path, module)

	out, execErr := s.executor.Run(ctx, args...)
	if execErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("index deps failed: %v", execErr)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Dispatch ---

func (s *Server) registerDispatch() {
	tool := mcp.NewTool("sdp_dispatch",
		mcp.WithDescription("Route a task to the best agent/model combination, or manage dispatch state."),
		mcp.WithString("task",
			mcp.Description("Task description to route (for routing)"),
		),
	)
	s.inner.AddTool(tool, s.handleDispatch)
}

func (s *Server) handleDispatch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Dispatch is exposed through a separate binary (sdp-dispatch) with
	// subcommands: route, limits, profile, bench, compare, status.
	// For MCP, the primary use case is "route".
	task := req.GetString("task", "")

	if task == "" {
		return mcp.NewToolResultError("dispatch requires a 'task' parameter"), nil
	}

	args := []string{"route", "--task", task, "--json"}

	out, err := s.executor.RunCustom(ctx, "sdp-dispatch", args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("dispatch failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Beads Create ---

func (s *Server) registerBeadsCreate() {
	tool := mcp.NewTool("sdp_beads_create",
		mcp.WithDescription("Create a tracked issue (bead) in the project's issue tracker."),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Issue title"),
		),
		mcp.WithString("description",
			mcp.Description("Issue description"),
		),
		mcp.WithString("type",
			mcp.Description("Issue type (task, bug, feature, epic)"),
		),
		mcp.WithNumber("priority",
			mcp.Description("Priority level (0-4, where 0 is highest)"),
		),
	)
	s.inner.AddTool(tool, s.handleBeadsCreate)
}

func (s *Server) handleBeadsCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Beads uses shell commands via 'bd' CLI.
	// Map MCP params to the bd create CLI.
	args := []string{"create", "--title", title}

	if desc := req.GetString("description", ""); desc != "" {
		args = append(args, "--description", desc)
	}
	if typ := req.GetString("type", ""); typ != "" {
		args = append(args, "--type", typ)
	}
	if priority, ok := req.Params.Arguments.(map[string]interface{}); ok {
		if p, exists := priority["priority"]; exists {
			args = append(args, "--priority", fmt.Sprintf("%v", p))
		}
	}

	out, execErr := s.executor.RunCustom(ctx, "bd", args...)
	if execErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("beads create failed: %v", execErr)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Beads Close ---

func (s *Server) registerBeadsClose() {
	tool := mcp.NewTool("sdp_beads_close",
		mcp.WithDescription("Close a tracked issue (bead) by ID."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Issue ID to close"),
		),
	)
	s.inner.AddTool(tool, s.handleBeadsClose)
}

func (s *Server) handleBeadsClose(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	out, execErr := s.executor.RunCustom(ctx, "bd", "close", id)
	if execErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("beads close failed: %v", execErr)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// --- Beads List ---

func (s *Server) registerBeadsList() {
	tool := mcp.NewTool("sdp_beads_list",
		mcp.WithDescription("List tracked issues (beads) with optional status and assignee filters."),
		mcp.WithString("status",
			mcp.Description("Filter by status (open, in_progress, closed)"),
		),
		mcp.WithString("assignee",
			mcp.Description("Filter by assignee"),
		),
	)
	s.inner.AddTool(tool, s.handleBeadsList)
}

func (s *Server) handleBeadsList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := []string{"list"}
	if status := req.GetString("status", ""); status != "" {
		args = append(args, "--status", status)
	}
	if assignee := req.GetString("assignee", ""); assignee != "" {
		args = append(args, "--assignee", assignee)
	}

	out, err := s.executor.RunCustom(ctx, "bd", args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("beads list failed: %v", err)), nil
	}
	return mcp.NewToolResultText(string(out)), nil
}

// repoPath returns the effective repository path: if the tool-level path
// parameter is non-empty it is used, otherwise the server-level default is used.
func (s *Server) repoPath(toolPath string) string {
	if toolPath != "" {
		return toolPath
	}
	return s.config.RepoRoot
}

// MeasureStartup measures the time to create a server instance.
// Useful for verifying the <200ms startup target.
func MeasureStartup(cfg ServerConfig) time.Duration {
	start := time.Now()
	_ = NewServer(cfg)
	return time.Since(start)
}
