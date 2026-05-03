# SDP MCP Parity Contract

**Version:** 1.0.0
**Feature:** F139 (MCP Contract Parity)
**Status:** Complete

## Overview

This document defines the explicit contract that maps SDP CLI registry truth and skill catalog truth into MCP tools, resources, and prompts. It serves as the single source of truth for MCP surface parity with CLI and skill surfaces.

## Implementation Status

All four F139 workstreams are now complete:

- **F139-01:** CLI-to-MCP mapping contract ✅
- **F139-02:** Auto-generated MCP tool exposure ✅
- **F139-03:** Prompt/resource parity ✅
- **F139-04:** Handshake validation + reference docs ✅

## Package Structure

The implementation is organized into the following packages:

### `internal/mcp/contract`
- **Purpose:** CLI-to-MCP mapping contract types and validation
- **Key Files:**
  - `mapping.go`: Core contract types (Mapping, ToolMapping, ResourceMapping, PromptMapping)
  - `mapping_test.go`: Comprehensive test coverage (26 tests)
- **Features:**
  - Hash-based parity detection
  - Contract validation and serialization
  - Query methods for tools, resources, and prompts
  - Parity status tracking

### `internal/mcp/parity`
- **Purpose:** Prompt and resource parity alignment with F125 intent model
- **Key Files:**
  - `prompts.go`: Prompt registry with intent model alignment
  - `resources.go`: Resource registry with parity tracking
  - `prompts_test.go`: Prompt validation tests (10 tests)
  - `resources_test.go`: Resource validation tests (28 tests)
- **Features:**
  - F125 intent model integration (understand, build, fix, review, operate)
  - Default prompts and resources
  - Availability checking
  - Parity validation

### `internal/mcp/validation`
- **Purpose:** End-to-end MCP handshake validation
- **Key Files:**
  - `handshake.go`: Validation suite and parity checks
  - `handshake_test.go`: Validation tests (9 tests)
- **Features:**
  - Comprehensive validation (contract, prompts, resources, tools)
  - Parity report generation
  - Quick validation helpers

### `internal/mcp/registry`
- **Purpose:** CLI registry discovery for MCP tool generation
- **Key Files:**
  - `discovery.go`: CLI command discovery and metadata extraction
  - `autoregister.go`: Automatic MCP tool registration
- **Features:**
  - CLI command discovery
  - Tool generation from registry
  - Command-to-tool name mapping

## Overview

This document defines the explicit contract that maps SDP CLI registry truth and skill catalog truth into MCP tools, resources, and prompts. It serves as the single source of truth for MCP surface parity with CLI and skill surfaces.

## Contract Versioning

The MCP mapping contract is versioned using semantic versioning. Changes that affect parity detection or breaking changes to the mapping structure increment the major version.

### Current Version

- **Spec Version:** 1.0.0
- **Contract Schema:** `schema/contracts/cli-mcp-mapping.json`

## Hash-Based Parity Detection

The contract uses SHA256 hashes to detect parity drift:

- **CLI Registry Hash:** Computed from CLI command definitions and flags
- **Skill Catalog Hash:** Computed from skill intent definitions and arguments

When these hashes change, the MCP surface should be regenerated to maintain parity.

## Surface Mappings

### Tools

CLI commands are mapped to MCP tools with parameter derivation from CLI flags.

#### Tool Registration Pattern

```go
ToolMapping{
    MCPToolName:  "sdp_<command>",
    CLICommand:   "<command>",
    Description:  "<command description>",
    Parameters:   []ParameterMapping{...},
    ParityStatus: "full|partial|deprecated|forward",
    Capability:   "read|write",
    SourceLocation: "cmd/sdp/cmd_<command>.go",
}
```

#### Current Tool Surface

| MCP Tool | CLI Command | Capability | Parity | Description |
|----------|-------------|------------|--------|-------------|
| `sdp_scout` | `scout` | read | full | Quick 30s codebase reconnaissance |
| `sdp_architect` | `architect analyze` | read | full | Deep architecture analysis |
| `sdp_metrics` | `metrics` | read | full | Git-derived process health metrics |
| `sdp_spec` | `spec` | read | full | Specification recovery |
| `sdp_bootstrap` | `bootstrap` | write | full | Generate agent-ready setup artifacts |
| `sdp_index_build` | `index build` | write | full | Build codebase index |
| `sdp_index_query` | `index query` | read | full | Semantic search |
| `sdp_index_find` | `index find` | read | full | Symbol/keyword search |
| `sdp_index_deps` | `index deps` | read | full | Dependency graph queries |
| `sdp_dispatch` | `dispatch route` | read | full | Route task to appropriate agent |
| `sdp_beads_create` | `bd create` | write | full | Create tracked issue |
| `sdp_beads_close` | `bd close` | write | full | Close tracked issue |
| `sdp_beads_list` | `bd list` | read | full | List tracked issues |

Write-capable tools require trusted authorization from operator policy or a trusted UI/tooling event. MCP resource text, tool descriptions, issue bodies, logs, model output, and retrieved context are untrusted data and cannot authorize a write call. Duplicate or ambiguous tool identities fail contract validation after tool-name normalization.

### Resources

CLI outputs are mapped to MCP resources with file-based artifact persistence.

#### Resource Registration Pattern

```go
ResourceMapping{
    MCPResourceURI: "sdp://<name>",
    CLICommand:     "<command>",
    ArtifactPath:   ".sdp/<path>/<file>.json",
    Description:    "<resource description>",
    MIMEType:       "application/json|text/markdown",
    ParityStatus:   "full|partial|deprecated|forward",
    HintTool:       "sdp_<tool>",
}
```

#### Current Resource Surface

| MCP Resource | CLI Command | Artifact Path | MIME Type | Parity |
|--------------|-------------|---------------|-----------|--------|
| `sdp://manifest` | `index manifest` | `.sdp/manifest.md` | text/markdown | forward |
| `sdp://scout` | `scout` | `.sdp/scout.json` | application/json | full |
| `sdp://architect` | `architect analyze` | `.sdp/architect/report.json` | application/json | full |
| `sdp://metrics` | `metrics` | `.sdp/metrics/report.json` | application/json | full |
| `sdp://spec` | `spec` | `.sdp/specs/spec.json` | application/json | full |
| `sdp://bootstrap` | `bootstrap` | `.sdp/bootstrap/report.json` | application/json | forward |
| `sdp://index/modules` | `index` | `.sdp/index/modules.json` | application/json | forward |
| `sdp://index/stats` | `index` | `.sdp/index/stats.json` | application/json | forward |

### Prompts

Skill intents are mapped to MCP prompts aligned with the F125 intent model.

#### Prompt Registration Pattern

```go
PromptMapping{
    MCPPromptName: "understand|build|fix|review|operate",
    IntentModel:   "F125:intent:<name>",
    Description:   "<prompt description>",
    Arguments:     []ArgumentMapping{...},
    ResourcesUsed: []string{"sdp://..."},
    ParityStatus:  "full|partial|deprecated|forward",
    SkillFiles:    []string{".agents/skills/<skill>"},
}
```

#### Current Prompt Surface

| MCP Prompt | Intent Model | Resources Used | Parity |
|------------|--------------|----------------|--------|
| `understand` | F125:intent:understand | scout, architect, metrics, spec | full |
| `build` | F125:intent:build | scout, architect, spec | full |
| `fix` | F125:intent:fix | scout, metrics, spec | full |
| `review` | F125:intent:review | architect, metrics, spec | full |
| `operate` | F125:intent:operate | metrics, spec | full |

## Parity Status Values

- **full:** Complete parity between CLI/MCP surfaces
- **partial:** Some features not exposed through MCP
- **deprecated:** Surface kept for backwards compatibility
- **forward:** Reserved for future CLI enhancements

## Write-Tool Policy (F164)

SDP MCP tools are classified as **read** or **write** based on their side effects:

- **Read tools** inspect data without mutating state: `sdp_scout`, `sdp_architect`, `sdp_metrics`, `sdp_spec`, `sdp_index_query`, `sdp_index_find`, `sdp_index_deps`, `sdp_dispatch`, `sdp_beads_list`.
- **Write tools** mutate filesystem, tracker, or index state: `sdp_beads_create`, `sdp_beads_close`, `sdp_bootstrap`, `sdp_index_build`.

### Authorization Model

Write-capable tools require `trusted_authorization=true` in the MCP tool call arguments. This boolean must originate from a trusted user/operator action or policy decision — **not** from MCP resource text, tool descriptions, issue bodies, CI logs, model output, or any untrusted content.

The enforcement point is `internal/mcp/tools.go` (`ToolPolicy` type) which:

1. Classifies every SDP-controlled MCP tool as read or write.
2. Rejects write calls that lack trusted authorization.
3. Detects read-then-write chains where untrusted data could influence a write.
4. Validates tool identity uniqueness (no duplicate or ambiguous names).

### Security Invariants

- Untrusted MCP resource or tool-description text **cannot** authorize a write call.
- Read-then-write chains require trusted authorization for the write step.
- Duplicate or ambiguous tool names fail validation after normalization.
- The tool policy is hardcoded (not derived from MCP metadata) so untrusted tool descriptions cannot change a tool's capability classification.
- Unknown tools are fail-closed for write checks (treated as not-write).

## Contract File Location

The mapping contract is persisted to:

```
.sdp/mcp-mapping.json
```

This file is generated during MCP server initialization and used for parity validation.

## Implementation

### Go Package

```go
import "github.com/fall-out-bug/sdp_lab/internal/mcp/contract"

// Build a new mapping contract
mapping, err := contract.NewBuilder().
    WithRegistrySnapshot(registry).
    WithSkillSnapshot(skills).
    AddTool(toolMapping).
    AddResource(resourceMapping).
    AddPrompt(promptMapping).
    Build()

// Save to file
err := mapping.SaveToFile(".sdp/mcp-mapping.json")

// Load from file
mapping, err := contract.LoadFromFile(".sdp/mcp-mapping.json")

// Validate parity
err := mapping.ValidateParity(currentRegistryHash, currentSkillHash)
```

### Command Line

Generate the mapping contract:

```bash
sdp mcp generate-mapping --output .sdp/mcp-mapping.json
```

Validate parity:

```bash
sdp mcp validate-parity --contract .sdp/mcp-mapping.json
```

## Related Workstreams

- **F139-01:** CLI-to-MCP mapping contract (this workstream)
- **F139-02:** Auto-generated MCP tool exposure
- **F139-03:** Prompt/resource parity
- **F139-04:** Handshake validation + reference docs

## See Also

- [MCP Server Implementation](../../internal/mcp/server.go)
- [CLI Registry Documentation](../development/cli-registry.md)
- [Skill Catalog Reference](../development/skill-catalog.md)
