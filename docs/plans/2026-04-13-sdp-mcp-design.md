# SDP MCP Server: Universal Agent Interface

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Single MCP server that gives any AI agent (Claude Code, Cursor, Codex, OpenCode) full SDP toolkit access — tools, resources, and intent-based prompts — through the standard Model Context Protocol.

**Architecture:** Go MCP server (stdio transport) wrapping existing CLI tools. Three MCP primitives: Tools (actions), Resources (data), Prompts (intents). One server, all capabilities.

**Tech Stack:** Go, MCP SDK (github.com/mark3labs/mcp-go), stdio transport, JSON-RPC 2.0.

**Parent design:** `2026-04-13-sdp-toolkit-vision-design.md`, `2026-04-13-sdp-skill-architecture-design.md`

---

## Problem Statement

Today SDP capabilities are scattered across interfaces:
- CLI commands: `sdp architect`, `sdp dispatch`, etc.
- Skill files: 26 markdown prompts in `prompts/skills/`
- Agent configs: CLAUDE.md, .opencode/agents/, .cursor/commands/
- Each harness needs its own integration code

Result: every new tool × every harness = N×M integration work.

**MCP collapses this to 1:**

```
Before:                          After:
  Claude Code ─── skills/        Claude Code ─── MCP ─── SDP
  Cursor ──────── commands/      Cursor ──────── MCP ─── SDP
  Codex ───────── agents/        Codex ───────── MCP ─── SDP
  OpenCode ────── agents/        OpenCode ────── MCP ─── SDP

  4 integrations × N tools       1 server × M clients
```

## MCP Primitives Mapping

### Tools — Actions the agent can perform

```jsonc
// Analysis tools
sdp_scout        — Quick 30s codebase reconnaissance
sdp_architect    — Deep architecture analysis
sdp_metrics      — Git-derived process health metrics
sdp_spec         — Specification recovery from code
sdp_bootstrap    — Generate agent-ready setup artifacts

// Index tools
sdp_index_build  — Build/refresh codebase index
sdp_index_query  — Semantic search ("how does routing work")
sdp_index_find   — Symbol/keyword search ("ExecutorBridge")
sdp_index_deps   — Dependency graph queries

// Workflow tools
sdp_dispatch     — Route task to best agent/model
sdp_beads_create — Create tracked issue
sdp_beads_close  — Close tracked issue
sdp_beads_list   — List issues by status
```

### Resources — Data the agent can read

```jsonc
// Always available (auto-discovered)
sdp://manifest        — .sdp/manifest.md (context primer, ≤2K tokens)
sdp://scout           — .sdp/scout.json (project card)

// Available after analysis
sdp://architect       — .sdp/architecture/report.json
sdp://metrics         — .sdp/metrics/report.json
sdp://spec            — .sdp/specs/report.json
sdp://bootstrap       — .sdp/bootstrap-report.json

// Dynamic resources (query-based)
sdp://index/modules   — Module list with metadata
sdp://index/stats     — Index statistics
```

### Prompts — Intent-based interaction patterns

```jsonc
understand  — "Analyze and understand this codebase"
             Arguments: depth (quick|standard|deep), focus (area)

build       — "Create a new feature or component"
             Arguments: scope (idea|feature|prototype), description

fix         — "Diagnose and fix a problem"
             Arguments: severity (critical|normal|low), description

review      — "Review changes for quality"
             Arguments: scope (code|arch|security|readiness)

operate     — "Deployment, CI, and backlog management"
             Arguments: mode (deploy|triage|plan)
```

## Tool Definitions

### sdp_scout

```jsonc
{
  "name": "sdp_scout",
  "description": "Quick 30-second codebase reconnaissance. Returns project card with language, scale, build system, activity, maturity signals. No LLM calls, pure filesystem + git analysis.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "Repository path to scout. Defaults to current directory.",
        "default": "."
      },
      "format": {
        "type": "string",
        "enum": ["json", "text", "card"],
        "default": "json",
        "description": "Output format. 'json' for structured data, 'text' for human-readable, 'card' for compact summary."
      }
    }
  }
}
```

### sdp_architect

```jsonc
{
  "name": "sdp_architect",
  "description": "Deep architecture analysis producing C4 models, pattern detection, risk assessment, and tech debt identification. Takes 5-15 minutes. Uses LLM for synthesis.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "default": "."
      },
      "section": {
        "type": "string",
        "description": "Specific section to analyze: 'modules', 'patterns', 'risks', 'contracts', 'all'",
        "default": "all"
      },
      "fast": {
        "type": "boolean",
        "description": "Fast mode: skip LLM synthesis, extractors only",
        "default": false
      }
    }
  }
}
```

### sdp_metrics

```jsonc
{
  "name": "sdp_metrics",
  "description": "Extract process health metrics from git history. 7 categories: commit hygiene, wasted work, git flow, release quality, stabilization, knowledge risk, code decay. Pure git analysis, no LLM.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "default": "."
      },
      "category": {
        "type": "array",
        "items": {
          "type": "string",
          "enum": ["hygiene", "waste", "gitflow", "release", "stabilization", "knowledge", "decay"]
        },
        "description": "Specific categories to analyze. Defaults to all."
      },
      "period": {
        "type": "integer",
        "description": "Analysis period in years",
        "default": 2
      }
    }
  }
}
```

### sdp_index_query

```jsonc
{
  "name": "sdp_index_query",
  "description": "Semantic search over codebase index. Hybrid vector + keyword search with RRF fusion. Returns relevant code chunks with file paths and line numbers. Requires index to be built first (sdp_index_build).",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Natural language query about the codebase"
      },
      "limit": {
        "type": "integer",
        "default": 10,
        "description": "Maximum number of results"
      },
      "kind": {
        "type": "string",
        "enum": ["function", "type", "method", "interface", "file", "all"],
        "default": "all",
        "description": "Filter by chunk kind"
      }
    },
    "required": ["query"]
  }
}
```

### sdp_index_find

```jsonc
{
  "name": "sdp_index_find",
  "description": "Fast symbol/keyword search in codebase index. FTS5-based exact matching for identifiers, paths, literals. Sub-100ms response.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "symbol": {
        "type": "string",
        "description": "Symbol name, path fragment, or keyword to find"
      },
      "regex": {
        "type": "boolean",
        "default": false,
        "description": "Treat symbol as regex pattern"
      }
    },
    "required": ["symbol"]
  }
}
```

### sdp_index_deps

```jsonc
{
  "name": "sdp_index_deps",
  "description": "Query dependency graph from codebase index. Shows what a module depends on or what depends on it.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "module": {
        "type": "string",
        "description": "Module path to query dependencies for"
      },
      "direction": {
        "type": "string",
        "enum": ["forward", "reverse"],
        "default": "forward",
        "description": "'forward': what does this depend on. 'reverse': what depends on this."
      },
      "depth": {
        "type": "integer",
        "default": 1,
        "description": "Traversal depth (1 = direct deps only)"
      }
    },
    "required": ["module"]
  }
}
```

### sdp_spec

```jsonc
{
  "name": "sdp_spec",
  "description": "Extract implicit specifications from code: API contracts, business rules, invariants, SLA parameters. Deterministic AST extraction by default, optional LLM enrichment.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "default": "."
      },
      "category": {
        "type": "string",
        "enum": ["api", "rules", "invariants", "sla", "all"],
        "default": "all"
      },
      "enrich": {
        "type": "boolean",
        "default": false,
        "description": "Enable LLM enrichment for richer descriptions"
      }
    }
  }
}
```

### sdp_bootstrap

```jsonc
{
  "name": "sdp_bootstrap",
  "description": "Generate agent-ready setup artifacts: CLAUDE.md, policies, hooks, beads init. Data-driven from analysis results, no LLM guessing. Runs scout automatically if needed.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "default": "."
      },
      "only": {
        "type": "string",
        "enum": ["claude-md", "policies", "hooks", "all"],
        "default": "all"
      },
      "dry_run": {
        "type": "boolean",
        "default": false,
        "description": "Show what would be generated without writing files"
      },
      "verify": {
        "type": "boolean",
        "default": true,
        "description": "Verify build/test/lint commands work"
      }
    }
  }
}
```

## Prompt Definitions

### understand

```jsonc
{
  "name": "understand",
  "description": "Analyze and understand a codebase at the desired depth. Automatically selects which tools to run based on what's already available and the requested depth.",
  "arguments": [
    {
      "name": "depth",
      "description": "Analysis depth: 'quick' (scout only, 30s), 'standard' (scout+architect+metrics, 10min), 'deep' (all tools, 20min)",
      "required": false
    },
    {
      "name": "focus",
      "description": "Optional focus area: 'architecture', 'health', 'api', 'security', 'team'",
      "required": false
    }
  ]
}
```

**Prompt template (understand/standard):**

```
You are analyzing the {{repo_name}} codebase to build a comprehensive understanding.

## Available Data
{{#if scout_json}}
### Project Card (from scout)
{{scout_json}}
{{else}}
Run sdp_scout first to get basic project information.
{{/if}}

{{#if architect_report}}
### Architecture (from architect)
{{architect_summary}}
{{/if}}

{{#if metrics_report}}
### Process Health (from metrics)
{{metrics_summary}}
{{/if}}

## Your Task
1. If any analysis data is missing for the requested depth, run the appropriate tools
2. Synthesize findings into a coherent narrative
3. Highlight the top 3 risks and top 3 strengths
4. Generate actionable recommendations
5. Update .sdp/manifest.md with current understanding

Depth: {{depth}}
{{#if focus}}Focus area: {{focus}}{{/if}}
```

### build

```jsonc
{
  "name": "build",
  "description": "Create a new feature, component, or prototype. Follows TDD, uses project conventions from CLAUDE.md.",
  "arguments": [
    {
      "name": "description",
      "description": "What to build",
      "required": true
    },
    {
      "name": "scope",
      "description": "'idea' (brainstorm only), 'feature' (full cycle), 'prototype' (fast build)",
      "required": false
    }
  ]
}
```

### fix

```jsonc
{
  "name": "fix",
  "description": "Diagnose and fix a problem. Adapts approach based on severity and available information.",
  "arguments": [
    {
      "name": "description",
      "description": "What's broken or needs fixing",
      "required": true
    },
    {
      "name": "severity",
      "description": "'critical' (production down), 'normal' (standard bug), 'low' (minor issue)",
      "required": false
    },
    {
      "name": "issue",
      "description": "Beads issue ID if tracking",
      "required": false
    }
  ]
}
```

## Server Architecture

```
cmd/sdp-mcp/main.go          — Entry point, stdio transport
internal/mcp/
  ├── server.go               — MCP server setup, tool/resource/prompt registration
  ├── server_test.go
  ├── tools.go                — Tool handlers (wrap CLI commands)
  ├── tools_test.go
  ├── resources.go            — Resource handlers (read .sdp/ files)
  ├── resources_test.go
  ├── prompts.go              — Prompt handlers (intent templates)
  ├── prompts_test.go
  └── templates/
      ├── understand.tmpl     — @understand prompt template
      ├── build.tmpl          — @build prompt template
      ├── fix.tmpl            — @fix prompt template
      ├── review.tmpl         — @review prompt template
      └── operate.tmpl        — @operate prompt template
```

### Tool Handler Pattern

Each tool wraps an existing CLI command:

```go
func (s *Server) handleScout(ctx context.Context, params map[string]any) (*mcp.CallToolResult, error) {
    path := stringParam(params, "path", ".")
    format := stringParam(params, "format", "json")

    // Run CLI command
    out, err := exec.CommandContext(ctx, "sdp", "scout", "--format", format, path).Output()
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("scout failed: %v", err)), nil
    }

    return mcp.NewToolResultText(string(out)), nil
}
```

For tools that produce large output, return a summary + resource URI:

```go
func (s *Server) handleArchitect(ctx context.Context, params map[string]any) (*mcp.CallToolResult, error) {
    // ... run sdp architect analyze ...

    // Return summary + pointer to full report
    summary := extractSummary(report)
    return mcp.NewToolResultText(fmt.Sprintf(
        "%s\n\nFull report: sdp://architect",
        summary,
    )), nil
}
```

### Resource Handler Pattern

Resources read from `.sdp/` directory:

```go
func (s *Server) handleManifestResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
    content, err := os.ReadFile(filepath.Join(s.repoRoot, ".sdp", "manifest.md"))
    if err != nil {
        return nil, fmt.Errorf("manifest not found: run sdp_index_build first")
    }

    return &mcp.ReadResourceResult{
        Contents: []mcp.ResourceContents{
            mcp.TextResourceContents{
                URI:      uri,
                MIMEType: "text/markdown",
                Text:     string(content),
            },
        },
    }, nil
}
```

### Prompt Handler Pattern

Prompts assemble context from available resources:

```go
func (s *Server) handleUnderstandPrompt(ctx context.Context, args map[string]string) (*mcp.GetPromptResult, error) {
    depth := args["depth"]
    if depth == "" {
        depth = "standard"
    }

    // Collect available data
    data := s.collectAvailableData()

    // Render template
    tmpl := s.templates.Lookup("understand.tmpl")
    var buf bytes.Buffer
    tmpl.Execute(&buf, map[string]any{
        "depth":      depth,
        "focus":      args["focus"],
        "repo_name":  data.RepoName,
        "scout_json": data.ScoutJSON,
        "architect":  data.ArchitectSummary,
        "metrics":    data.MetricsSummary,
    })

    return &mcp.GetPromptResult{
        Description: "Understand this codebase",
        Messages: []mcp.PromptMessage{
            {Role: "user", Content: mcp.TextContent{Text: buf.String()}},
        },
    }, nil
}
```

## Client Configuration

### Claude Code

```jsonc
// .claude/settings.json or ~/.claude/settings.json
{
  "mcpServers": {
    "sdp": {
      "command": "sdp-mcp",
      "args": ["--repo", "."],
      "env": {
        "SDP_LOG_LEVEL": "warn"
      }
    }
  }
}
```

### Cursor

```jsonc
// .cursor/mcp.json
{
  "mcpServers": {
    "sdp": {
      "command": "sdp-mcp",
      "args": ["--repo", "."]
    }
  }
}
```

### VS Code (Copilot)

```jsonc
// .vscode/mcp.json
{
  "servers": {
    "sdp": {
      "type": "stdio",
      "command": "sdp-mcp",
      "args": ["--repo", "."]
    }
  }
}
```

## Transport

**stdio** (primary): Agent spawns `sdp-mcp` as child process. Simple, no auth needed, works everywhere.

**SSE** (future): For shared/remote setups where multiple clients connect to one index. Requires auth token.

```
Phase 1: stdio only (covers all local use cases)
Phase 2: SSE for team/CI scenarios
```

## Lifecycle

```
Agent starts session
    │
    ├─ MCP client connects to sdp-mcp (stdio)
    │
    ├─ Server discovers .sdp/ state
    │   ├─ manifest.md exists? → register as resource
    │   ├─ scout.json exists?  → register as resource
    │   ├─ index.db exists?    → enable query tools
    │   └─ reports exist?      → register as resources
    │
    ├─ Server registers all tools (always available)
    │
    ├─ Agent reads sdp://manifest (if exists)
    │   → immediate context about the repo
    │
    ├─ Agent calls tools as needed
    │   sdp_scout → sdp_index_query → sdp_metrics → ...
    │
    ├─ Agent uses prompts for complex workflows
    │   understand(depth=standard) → orchestrated analysis
    │
    └─ Session ends, server exits
```

## Performance

| Operation | Target | Notes |
|-----------|--------|-------|
| Server startup | <200ms | Register tools + discover .sdp/ state |
| Tool call overhead | <50ms | Process spawn + JSON marshaling |
| Resource read | <10ms | File read + marshal |
| Prompt render | <20ms | Template execution |

Tool execution time is dominated by the underlying CLI command, not MCP overhead.

## Testing Strategy

1. **Tool handler tests:** Mock CLI execution, verify input/output mapping.
2. **Resource handler tests:** Create temp .sdp/ directory, verify resource discovery and reading.
3. **Prompt handler tests:** Verify template rendering with various data combinations.
4. **Integration test:** Start server, connect MCP client, exercise full tool→resource→prompt flow.
5. **Multi-client test:** Verify stdio transport handles concurrent tool calls correctly.

## Security

- **No auth for stdio.** Parent process controls access — if you can spawn sdp-mcp, you can use it.
- **SSE requires token.** Environment variable `SDP_MCP_TOKEN` for SSE transport.
- **No secrets in resources.** `.sdp/` files are analysis output, not credentials.
- **Tool sandboxing.** Tools run CLI commands with same permissions as the MCP server process. No privilege escalation.
- **Rate limiting.** Optional rate limit on expensive tools (architect, metrics) to prevent runaway agent loops.

## Dependencies

```
github.com/mark3labs/mcp-go    # Go MCP SDK (MIT license)
```

No other new dependencies. All tool handlers call existing `sdp` CLI binary.

## Relationship to Existing Systems

```
sdp CLI (existing)
  ├── sdp architect    ←── sdp_architect tool wraps this
  ├── sdp dispatch     ←── sdp_dispatch tool wraps this
  └── sdp discover     ←── not wrapped (LLM-dependent, interactive)

sdp CLI (new, from design docs)
  ├── sdp scout        ←── sdp_scout tool
  ├── sdp metrics      ←── sdp_metrics tool
  ├── sdp spec         ←── sdp_spec tool
  ├── sdp index        ←── sdp_index_* tools
  └── sdp bootstrap    ←── sdp_bootstrap tool

Skills (current 26 files)
  └── Migrate to MCP prompts (5 intents)

Agent configs (.claude/, .opencode/, .cursor/)
  └── Replace with MCP server config (1 config per harness)
```

## Design Decisions

1. **Wrap CLI, don't embed.** MCP server calls `sdp` binary, doesn't import Go packages directly. This means CLI and MCP always behave identically, and CLI remains usable standalone.

2. **stdio first.** SSE adds auth complexity. stdio covers the primary use case (local agent) perfectly.

3. **Resources are files.** No database queries through resources — use tools for that. Resources are for pre-computed, file-based data.

4. **Prompts are templates, not code.** Easy to customize, version, and review. Stored as .tmpl files.

5. **All tools always registered.** Even if index isn't built, `sdp_index_query` is registered — it returns a helpful error telling the agent to run `sdp_index_build` first. This avoids dynamic tool registration complexity.

6. **One binary.** `sdp-mcp` is a single Go binary. No Node.js runtime, no Python, no Docker. Install and go.
