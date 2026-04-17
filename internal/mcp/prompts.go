package mcp

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// promptData holds data available to all prompt templates.
// It is assembled from whatever .sdp/ artifacts exist on disk.
type promptData struct {
	RepoName         string
	ScoutJSON        string
	ArchitectSummary string
	MetricsSummary   string
	SpecSummary      string
	BootstrapSummary string

	// Intent-specific fields
	Depth       string
	Focus       string
	Description string
	Scope       string
	Severity    string
	Issue       string
	Mode        string
}

// collectAvailableData reads .sdp/ artifacts from disk and returns a promptData
// with whatever content is available. Missing artifacts result in empty strings
// (templates conditionally omit those sections). Uses safeReadFile to prevent
// symlink-based path escapes.
//
// NOTE: Artifacts are produced by CLI commands as a side effect, not by MCP tool
// handlers directly. See the Server struct doc comment for the full contract.
func (s *Server) collectAvailableData() promptData {
	data := promptData{
		RepoName: filepath.Base(s.config.RepoRoot),
	}

	readFile := func(relPath string) string {
		content, err := safeReadFile(s.config.RepoRoot, relPath)
		if err != nil {
			return ""
		}
		return string(content)
	}

	data.ScoutJSON = readFile(".sdp/scout.json")
	data.ArchitectSummary = readFile(".sdp/architect/report.json")
	data.MetricsSummary = readFile(".sdp/metrics/report.json")
	data.SpecSummary = readFile(".sdp/specs/spec.json")
	data.BootstrapSummary = readFile(".sdp/bootstrap/report.json")

	return data
}

// loadTemplates parses all .tmpl files from the embedded filesystem.
func loadTemplates() (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/*.tmpl")
}

// registerPrompts registers all five intent prompts with the MCP server.
func (s *Server) registerPrompts() {
	s.registerUnderstand()
	s.registerBuild()
	s.registerFix()
	s.registerReview()
	s.registerOperate()
}

func (s *Server) registerUnderstand() {
	prompt := mcp.NewPrompt("understand",
		mcp.WithPromptDescription("Analyze and understand this codebase at the desired depth."),
		mcp.WithArgument("depth",
			mcp.ArgumentDescription("Analysis depth: quick, standard, or deep"),
		),
		mcp.WithArgument("focus",
			mcp.ArgumentDescription("Optional focus area (e.g. 'security', 'performance', 'api')"),
		),
	)
	s.inner.AddPrompt(prompt, s.handleUnderstandPrompt)
}

func (s *Server) handleUnderstandPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	depth := getArg(args, "depth", "standard")
	focus := getArg(args, "focus", "")

	data := s.collectAvailableData()
	data.Depth = depth
	data.Focus = focus

	text, err := s.executeTemplate("understand.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("understand prompt: %w", err)
	}

	return mcp.NewGetPromptResult("Understand this codebase", []mcp.PromptMessage{
		{Role: mcp.RoleUser, Content: mcp.TextContent{Text: text}},
	}), nil
}

func (s *Server) registerBuild() {
	prompt := mcp.NewPrompt("build",
		mcp.WithPromptDescription("Create a new feature, component, or prototype."),
		mcp.WithArgument("description",
			mcp.ArgumentDescription("What to build"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("scope",
			mcp.ArgumentDescription("Scope: idea, feature, or prototype"),
		),
	)
	s.inner.AddPrompt(prompt, s.handleBuildPrompt)
}

func (s *Server) handleBuildPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	description := getArg(args, "description", "")
	if description == "" {
		return nil, fmt.Errorf("build prompt: 'description' argument is required")
	}
	scope := getArg(args, "scope", "feature")

	data := s.collectAvailableData()
	data.Description = description
	data.Scope = scope

	text, err := s.executeTemplate("build.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("build prompt: %w", err)
	}

	return mcp.NewGetPromptResult("Build a new feature", []mcp.PromptMessage{
		{Role: mcp.RoleUser, Content: mcp.TextContent{Text: text}},
	}), nil
}

func (s *Server) registerFix() {
	prompt := mcp.NewPrompt("fix",
		mcp.WithPromptDescription("Diagnose and fix a problem."),
		mcp.WithArgument("description",
			mcp.ArgumentDescription("Description of the problem"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("severity",
			mcp.ArgumentDescription("Severity level: critical, normal, or low"),
		),
		mcp.WithArgument("issue",
			mcp.ArgumentDescription("Optional known issue identifier or URL"),
		),
	)
	s.inner.AddPrompt(prompt, s.handleFixPrompt)
}

func (s *Server) handleFixPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	description := getArg(args, "description", "")
	if description == "" {
		return nil, fmt.Errorf("fix prompt: 'description' argument is required")
	}
	severity := getArg(args, "severity", "normal")
	issue := getArg(args, "issue", "")

	data := s.collectAvailableData()
	data.Description = description
	data.Severity = severity
	data.Issue = issue

	text, err := s.executeTemplate("fix.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("fix prompt: %w", err)
	}

	return mcp.NewGetPromptResult("Diagnose and fix a problem", []mcp.PromptMessage{
		{Role: mcp.RoleUser, Content: mcp.TextContent{Text: text}},
	}), nil
}

func (s *Server) registerReview() {
	prompt := mcp.NewPrompt("review",
		mcp.WithPromptDescription("Review changes for quality."),
		mcp.WithArgument("scope",
			mcp.ArgumentDescription("Review scope: code, arch, security, or readiness"),
		),
	)
	s.inner.AddPrompt(prompt, s.handleReviewPrompt)
}

func (s *Server) handleReviewPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	scope := getArg(args, "scope", "code")

	data := s.collectAvailableData()
	data.Scope = scope

	text, err := s.executeTemplate("review.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("review prompt: %w", err)
	}

	return mcp.NewGetPromptResult("Review changes for quality", []mcp.PromptMessage{
		{Role: mcp.RoleUser, Content: mcp.TextContent{Text: text}},
	}), nil
}

func (s *Server) registerOperate() {
	prompt := mcp.NewPrompt("operate",
		mcp.WithPromptDescription("Deployment, CI, and backlog management."),
		mcp.WithArgument("mode",
			mcp.ArgumentDescription("Operation mode: deploy, triage, or plan"),
		),
	)
	s.inner.AddPrompt(prompt, s.handleOperatePrompt)
}

func (s *Server) handleOperatePrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	mode := getArg(args, "mode", "triage")

	data := s.collectAvailableData()
	data.Mode = mode

	text, err := s.executeTemplate("operate.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("operate prompt: %w", err)
	}

	return mcp.NewGetPromptResult("Operational task", []mcp.PromptMessage{
		{Role: mcp.RoleUser, Content: mcp.TextContent{Text: text}},
	}), nil
}

// executeTemplate renders a named template from the embedded filesystem with
// the given data.
func (s *Server) executeTemplate(name string, data any) (string, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return "", fmt.Errorf("load templates: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}
	return buf.String(), nil
}

// getArg retrieves an argument from the map, returning the default if missing.
func getArg(args map[string]string, key, defaultVal string) string {
	if args == nil {
		return defaultVal
	}
	if v, ok := args[key]; ok && v != "" {
		return v
	}
	return defaultVal
}
