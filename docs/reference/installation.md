# SDP MCP Server -- Installation and Configuration

> **Scope:** building, installing, and configuring the `sdp-mcp` MCP server for supported AI coding harnesses.
> **Tested matrix:** Claude Code, Cursor, VS Code (Copilot), OpenCode.

---

## Prerequisites

| Requirement | Minimum | Notes |
|-------------|---------|-------|
| Go | 1.24+ | Required to build `sdp-mcp` and the `sdp` CLI |
| Git | 2.30+ | Used by `sdp metrics` and `sdp scout` for repository analysis |
| `sdp` CLI | built from this repo | `sdp-mcp` wraps the CLI, not the other way around |

## Building sdp-mcp

From the repository root:

```bash
go build -o sdp-mcp ./cmd/sdp-mcp/
```

This produces a single binary called `sdp-mcp` in the current directory. The binary has no runtime dependencies beyond the `sdp` CLI being available in PATH (or specified via `--binary`).

## Installing to PATH

Move the binary somewhere on your PATH:

```bash
# Option 1: /usr/local/bin (system-wide)
sudo mv sdp-mcp /usr/local/bin/

# Option 2: ~/bin (user-local, make sure ~/bin is on PATH)
mkdir -p ~/bin
mv sdp-mcp ~/bin/

# Option 3: Go's default bin directory
go build -o $(go env GOPATH)/bin/sdp-mcp ./cmd/sdp-mcp/
```

Verify the install:

```bash
sdp-mcp --version
# Expected output: sdp-mcp 0.1.0
```

### Building the sdp CLI

The MCP server delegates tool execution to the `sdp` CLI binary. Build and install it:

```bash
go build -o sdp ./cmd/sdp/      # or the appropriate entry point
mv sdp /usr/local/bin/           # same PATH strategy as above
```

If `sdp` is installed under a different name or path, use the `--binary` flag when configuring `sdp-mcp`:

```bash
sdp-mcp --repo /path/to/project --binary /custom/path/to/sdp
```

## Configuring Your Harness

Reference configuration files live in `configs/mcp/` in the repository. Copy or adapt the one that matches your editor.

### Claude Code

**Config location:** `.claude/settings.json` (project-local) or `~/.claude/settings.json` (global).

```jsonc
// .claude/settings.json
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

Claude Code spawns `sdp-mcp` as a child process per session. The `"."` in `--repo` resolves to the project root where Claude Code was launched.

**Reference config:** [`configs/mcp/claude-code.json`](../../configs/mcp/claude-code.json)

### Cursor

**Config location:** `.cursor/mcp.json` in the project root.

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

Cursor uses the same `mcpServers` schema as Claude Code. The `env` field is optional.

**Reference config:** [`configs/mcp/cursor.json`](../../configs/mcp/cursor.json)

### VS Code (Copilot)

**Config location:** `.vscode/mcp.json` in the project root.

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

Note the structural difference: VS Code uses `"servers"` (not `"mcpServers"`) and requires an explicit `"type": "stdio"` field.

**Reference config:** [`configs/mcp/vscode.json`](../../configs/mcp/vscode.json)

### OpenCode

**Config location:** `.opencode.json` in the project root, or `~/.opencode.json` globally.

```jsonc
// .opencode.json
{
  "mcpServers": {
    "sdp": {
      "type": "stdio",
      "command": "sdp-mcp",
      "args": ["--repo", "."],
      "env": []
    }
  }
}
```

OpenCode requires the `"type"` field and expects `"env"` as an array (which can be empty).

**Reference config:** [`configs/mcp/opencode.json`](../../configs/mcp/opencode.json)

## Absolute Path Alternative

All examples above assume `sdp-mcp` is on PATH. If you prefer an absolute path (common for CI or when the binary lives inside the repo):

```jsonc
{
  "command": "/home/user/projects/sdp_lab/sdp-mcp",
  "args": ["--repo", "/home/user/projects/my-project"]
}
```

Replace the paths with the actual locations on your system.

## Verifying Installation

After configuration, restart your editor/harness and verify the MCP server is connected.

### Method 1: Check tool listing

Ask the AI agent to list available SDP tools. You should see these registered tools:

| Tool | Purpose |
|------|---------|
| `sdp_scout` | Quick 30s codebase reconnaissance |
| `sdp_architect` | Deep architecture analysis |
| `sdp_metrics` | Git-derived process health metrics |
| `sdp_spec` | Specification recovery from code |
| `sdp_bootstrap` | Generate agent-ready setup artifacts |
| `sdp_index_build` | Build or refresh codebase index |
| `sdp_index_query` | Semantic search across indexed codebase |
| `sdp_index_find` | Fast symbol/keyword search |
| `sdp_index_deps` | Dependency graph queries |
| `sdp_dispatch` | Route task to best agent/model |
| `sdp_beads_create` | Create a tracked issue |
| `sdp_beads_close` | Close a tracked issue |
| `sdp_beads_list` | List tracked issues |

### Method 2: Quick smoke test

Ask the agent:

```
Run sdp_scout on this project.
```

If the tool executes and returns a project card, the server is correctly configured and the `sdp` CLI is reachable.

### Method 3: Version check from the harness

Some harnesses show connected MCP servers in their UI. Look for `sdp-mcp` with version `0.1.0`.

## Troubleshooting

### "sdp-mcp: command not found"

The binary is not on PATH. Either:
- Install it to a directory on your PATH (see [Installing to PATH](#installing-to-path)), or
- Use an absolute path in the config (see [Absolute Path Alternative](#absolute-path-alternative)).

### Tool call returns "scout failed: exec: \"sdp\": executable file not found"

The `sdp` CLI binary is not on PATH. The MCP server wraps the CLI, so both binaries must be accessible:
- `sdp-mcp` -- the MCP server itself
- `sdp` -- the CLI that performs the actual work

Either install `sdp` to PATH or pass `--binary /path/to/sdp` in the server args.

### Server starts but tools return errors

Common causes:
1. **Not inside a git repository.** Most tools require a git repo. Navigate to one or pass `--repo /path/to/repo`.
2. **Missing `.sdp/` directory.** Some tools (index queries) require an index to be built first. Run `sdp_index_build` before `sdp_index_query` or `sdp_index_find`.
3. **Insufficient permissions.** The MCP server runs CLI commands with the same permissions as the parent process. Ensure the user has read access to the repository.

### Environment variable SDP_LOG_LEVEL has no effect

This variable controls the MCP server's own logging (written to stderr). It does not affect tool output. Valid values: `debug`, `info`, `warn`, `error`. Default: `info`.

### Multiple projects

The `--repo` flag defaults to `"."` (current working directory of the spawned process). For most harnesses this resolves to the project root. If you work with multiple repos, create per-project configs with explicit `--repo` paths.

## Packaging Notes

- `sdp-mcp` is a single statically-linked Go binary. No runtime, no Docker, no Node.js.
- The server communicates over stdio (JSON-RPC 2.0). No ports, no network.
- Both `sdp-mcp` and `sdp` must be built for the target platform. Cross-compile with `GOOS`/`GOARCH` as needed.
- The server expects a local filesystem path to a git repository. Remote repository URLs are not supported.

## Supported Harnesses

Only the following harnesses have verified MCP integration configs in this repository:

| Harness | Config Format | Verified |
|---------|---------------|----------|
| Claude Code | `.claude/settings.json` | config only |
| Cursor | `.cursor/mcp.json` | config only |
| VS Code (Copilot) | `.vscode/mcp.json` | config only |
| OpenCode | `.opencode.json` | config only |

Other MCP-capable clients may work with the `stdio` transport but are not tested. See [WS-04](../../docs/workstreams/backlog/00-126-04.md) for the cross-harness verification plan.

## Related Documentation

- [MCP Design](../plans/2026-04-13-sdp-mcp-design.md)
- [MCP Implementation Plan](../plans/2026-04-13-sdp-mcp-implementation-plan.md)
- [Project Map](project-map.md)
- [Installation Profiles](#) (broader packaging model)
