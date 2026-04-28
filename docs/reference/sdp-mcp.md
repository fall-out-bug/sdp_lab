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
    SourceLocation: "cmd/sdp/cmd_<command>.go",
}
```

#### Current Tool Surface

| MCP Tool | CLI Command | Parity | Description |
|----------|-------------|--------|-------------|
| `sdp_scout` | `scout` | full | Quick 30s codebase reconnaissance |
| `sdp_architect` | `architect analyze` | full | Deep architecture analysis |
| `sdp_metrics` | `metrics` | full | Git-derived process health metrics |
| `sdp_spec` | `spec` | full | Specification recovery |
| `sdp_bootstrap` | `bootstrap` | full | Generate agent-ready setup artifacts |
| `sdp_index_build` | `index build` | full | Build codebase index |
| `sdp_index_query` | `index query` | full | Semantic search |
| `sdp_index_find` | `index find` | full | Symbol/keyword search |
| `sdp_index_deps` | `index deps` | full | Dependency graph queries |
| `sdp_dispatch` | `dispatch route` | full | Route task to appropriate agent |
| `sdp_beads_create` | `bd create` | full | Create tracked issue |
| `sdp_beads_close` | `bd close` | full | Close tracked issue |
| `sdp_beads_list` | `bd list` | full | List tracked issues |

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