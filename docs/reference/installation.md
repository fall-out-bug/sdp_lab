# SDP MCP Server -- Installation and Configuration

> **Scope:** building, installing, and configuring the `sdp-mcp` MCP server for supported AI coding harnesses.
> **Tested matrix:** Config formats verified; end-to-end harness testing pending.
> **First-run note:** this page is not the primary Toolkit onboarding path. For
> a cold downstream repo, start with [Quickstart](../QUICKSTART.md) and its
> repo-local `./.sdp/bin/sdp` binary. Use this MCP guide only when you are
> intentionally wiring MCP tools into a harness.

---

## Prerequisites

| Requirement | Minimum | Notes |
|-------------|---------|-------|
| Go | 1.26+ | Required to build `sdp-mcp` and the `sdp` CLI |
| Git | 2.30+ | Used by `sdp metrics` and `sdp scout` for repository analysis |
| `sdp` CLI | built from this repo | `sdp-mcp` wraps the CLI, not the other way around |

## Runtime Dependencies

The MCP server delegates tool execution to external CLI binaries. All three must be installed and available on PATH (or specified via flags):

| Binary | Used by tools | Required | Notes |
|--------|---------------|----------|-------|
| `sdp` | `sdp_scout`, `sdp_architect`, `sdp_metrics`, `sdp_spec`, `sdp_bootstrap`, `sdp_index_*` | Yes | Core SDP CLI; built from `cmd/sdp/` |
| `sdp-dispatch` | `sdp_dispatch` | For dispatch only | Dispatch routing binary; built from `cmd/sdp-dispatch/` |
| `bd` | `sdp_beads_create`, `sdp_beads_close`, `sdp_beads_list` | For beads only | Beads issue tracker CLI |

If a binary is missing, the corresponding tool calls will return an error. The `--binary` flag on `sdp-mcp` controls the `sdp` CLI path; `sdp-dispatch` and `bd` are always looked up in PATH.

## Building sdp-mcp

The `sdp-mcp` binary itself is standalone. However, the MCP tools it exposes shell out to additional CLIs -- see the Runtime Dependencies table above.

From the repository root:

```bash
go build -o sdp-mcp ./cmd/sdp-mcp/
```

This produces a single binary called `sdp-mcp` in the current directory.

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
go build -tags sqlite_fts5 -o sdp ./cmd/sdp/   # FTS5 tag required for index tools
mv sdp /usr/local/bin/                          # same PATH strategy as above
```

> **Why `-tags sqlite_fts5`?** The `sdp` CLI uses SQLite with the FTS5 (Full-Text Search 5) extension for index operations (`sdp index build`, `sdp index query`, `sdp index find`, `sdp index deps`). Without this build tag, the binary compiles but index tools will fail at runtime. The CI pipeline, Makefile, and quality gate scripts all use this tag.

If `sdp` is installed under a different name or path, use the `--binary` flag when configuring `sdp-mcp`:

```bash
sdp-mcp --repo /path/to/project --binary /custom/path/to/sdp
```

## Configuring Your Harness

Reference configuration files live in `configs/mcp/` in the repository. Copy or adapt the one that matches your editor.

### Claude Code

**Config locations:**

- **Project-scoped (shared):** `.mcp.json` in the project root. This is the recommended location because it can be checked into version control and shared with the team.
- **Local-scoped:** Stored in `~/.claude.json` under the project's path. Private to you and the current project.
- **User-scoped:** `~/.claude.json`. Available across all projects, private to your user account.

```jsonc
// .mcp.json (project-scoped, recommended)
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

**Config locations:**

- **Project-scoped:** `opencode.json` in the project root.
- **Global:** `~/.config/opencode/opencode.json`.

```jsonc
// opencode.json (project root)
{
  "mcp": {
    "sdp": {
      "type": "local",
      "command": ["sdp-mcp", "--repo", "."],
      "environment": {}
    }
  }
}
```

OpenCode uses a top-level `"mcp"` key (not `"mcpServers"`). Each server entry has `"type": "local"`, `"command"` as a string array (binary + args), and `"environment"` as an object for environment variables.

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
| `sdp_index_build` | Build codebase index |
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
Run sdp_scout on this project with format=json.
```

If the tool executes and returns a JSON project report, the server is correctly configured and the `sdp` CLI is reachable. To get a human-readable card instead, use `format=card`.

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

### Resources return "resource not available: .sdp/... not found"

This is expected when the CLI has not been run with the appropriate `--output` flag. The MCP server does **not** persist artifacts — it wraps CLI commands and returns their stdout. Resources read pre-existing files from `.sdp/`. To populate resources, run CLI commands directly with `--output` to write artifacts to disk. For example:

- `sdp_scout` calls `sdp scout`, which produces `.sdp/scout.json` when run with `--output .sdp/`. The `sdp://scout` resource reads that file.
- `sdp_index_build` calls `sdp index build`, which produces `.sdp/index.db`. The manifest is generated by a separate command: `sdp index manifest <repo-path>` writes `.sdp/manifest.md`. The `sdp://manifest` resource reads that file.
- `sdp_architect` calls `sdp architect analyze`, which produces `.sdp/architect/report.json` when run with `--output .sdp/architect/report.json`. The `sdp://architect` resource reads that file.
- `sdp_metrics` calls `sdp metrics`, which produces `.sdp/metrics/report.json` when run with `--output .sdp/metrics`.
- `sdp_spec` calls `sdp spec`, which produces `.sdp/specs/spec.json` when run with `--output .sdp/specs`.
- `sdp://bootstrap` is a forward-compatible resource. The bootstrap CLI writes AGENTS.md, hooks, and policies but does not produce a JSON report file.
- `sdp://index/modules` and `sdp://index/stats` are forward-compatible resources for planned index enhancements. No current MCP tool produces these files.

Run the tool first, then query the resource.

### Server starts but tools return errors

Common causes:
1. **Not inside a git repository.** Most tools require a git repo. Navigate to one or pass `--repo /path/to/repo`.
2. **Missing `.sdp/` directory.** Some tools (index queries) require an index to be built first. Run `sdp_index_build` before `sdp_index_query` or `sdp_index_find`.
3. **Insufficient permissions.** The MCP server runs CLI commands with the same permissions as the parent process. Ensure the user has read access to the repository.

### Environment variable SDP_LOG_LEVEL has no effect

This variable controls the MCP server's own logging (written to stderr). It does not affect tool output. Valid values: `debug`, `info`, `warn`, `error`. Default: `warn`.

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
| Claude Code | `.mcp.json` (project) or `~/.claude.json` (user/local) | config only |
| Cursor | `.cursor/mcp.json` | config only |
| VS Code (Copilot) | `.vscode/mcp.json` | config only |
| OpenCode | `opencode.json` (project root) or `~/.config/opencode/opencode.json` (global) | config only |

Other MCP-capable clients may work with the `stdio` transport but are not tested. See [WS-04](../../docs/workstreams/backlog/00-126-04.md) for the protocol consistency verification status and the deferred end-to-end harness testing plan.

## Related Documentation

- [MCP Design](../plans/2026-04-13-sdp-mcp-design.md)
- [MCP Implementation Plan](../plans/2026-04-13-sdp-mcp-implementation-plan.md)
- [Project Map](project-map.md)
- [Installation Profiles](../plans/2026-04-13-sdp-mcp-implementation-plan.md) (broader packaging model)

---

## Security

This section documents what the MCP layer can and cannot access, its security boundaries, and the threat model it operates under.

### What the MCP Layer Can Access

| Capability | Mechanism | Scope |
|------------|-----------|-------|
| Repository files | Tool handlers wrap `sdp` CLI commands | Files the `sdp` binary can read (same user) |
| `.sdp/` artifacts | Resource handlers read from disk | Only files under `<repo>/.sdp/` |
| Intent templates | Prompt handlers render embedded `.tmpl` files | Data from `.sdp/` artifacts only |
| Shell commands | `exec.Command` via tool handlers | Same process user, no shell expansion |

### What the MCP Layer Cannot Access

- **No network access.** The server communicates exclusively over stdio (JSON-RPC 2.0). No HTTP, no TCP, no DNS.
- **No auth tokens.** The server does not store, read, or transmit authentication tokens, API keys, or credentials.
- **No privilege escalation.** All tool handlers use `exec.Command` (same-user process spawn). No `sudo`, `doas`, `setuid`, or privilege-elevation mechanisms.
- **No secrets exposure.** Resource handlers read only from `.sdp/` directory files. Files like `.env`, `credentials.json`, SSH keys, or any file outside `.sdp/` are not accessible through the MCP resource interface. If `.sdp/` itself is a symlink pointing outside the repository, the server will reject the read.
- **No shell interpretation.** Tool arguments are passed to the CLI binary as individual arguments via `exec.Command(binary, args...)`. Shell metacharacters (`;`, `&&`, `|`, `$()`, etc.) are not expanded.

### stdio Transport Security Model

The `sdp-mcp` server uses stdio transport: the parent process (the harness) spawns the server as a child process and communicates over stdin/stdout. This means:

1. **Parent controls access.** Only processes that can spawn `sdp-mcp` can use it. There is no network listener to attack.
2. **No remote attack surface.** The server is not reachable from the network. An attacker would need local code execution to interact with it.
3. **Process lifecycle tied to parent.** When the harness session ends, the server exits. No orphaned background processes.
4. **Same user permissions.** The server runs with the same UID/GID as the harness. It cannot escalate privileges beyond what the user can already do.

### Path Handling Guarantees

- Tool `path` parameters are passed verbatim to the CLI binary. The MCP server does not resolve, normalize, or validate paths.
- Resource URIs use the `sdp://` scheme and are mapped to hardcoded relative paths under `.sdp/`. There is no mechanism for an MCP client to request arbitrary file paths through resources.
- Resource paths are hardcoded to `.sdp/...` relative paths. The server resolves symlinks and rejects any path that escapes the `.sdp/` subtree. If `.sdp/` itself is a symlink (even within the repo), the server rejects the read.

### Rate Limiting Considerations

The current implementation does not include built-in rate limiting. However:

- The MCP protocol itself is request-response: the harness controls how frequently it calls tools.
- Expensive tools (e.g., `sdp_architect`) naturally limit themselves through execution time.
- If runaway agent loops are a concern, configure the harness's own rate limiting or request budget.
- Future work (see MCP design doc) may add optional per-session rate limiting as middleware.

### Threat Model Summary

| Threat | Mitigation |
|--------|-----------|
| Path traversal via tool arguments | Not applicable: the CLI is the security boundary, and the MCP server passes paths through without interpretation |
| Command injection via tool arguments | `exec.Command` does not invoke a shell; arguments are passed as-is to the binary |
| Secrets leak via resources | Resource handlers read only from `.sdp/`; the `staticResources` table is hardcoded and cannot be modified by MCP clients |
| Privilege escalation | No setuid/sudo/doas; process runs with parent's UID |
| Remote code execution | No network listener; stdio-only transport |
| Template injection in prompts | Go `text/template` does not double-evaluate; arguments are data, not template code |
