# Findings Schema Specification

## Overview

The Findings Schema defines a stable format for GitHub CI outputs that can be ingested by local agents for the SDP improvement loop. This enables deterministic processing of protocol and documentation violations.

## Schema Files

| Schema | Purpose | Location |
|--------|---------|----------|
| Protocol Findings | Protocol violations from `sdp-protocol-check` | `schema/findings/protocol-findings.schema.json` |
| Docs Findings | Documentation violations from `sdp-doc-sync` | `schema/findings/docs-findings.schema.json` |

## Versioning Strategy

### Version Format

Schemas use semantic versioning with a `v` prefix:

```
vMAJOR.MINOR
```

- **MAJOR**: Breaking changes (field removal, type changes, required field additions)
- **MINOR**: Non-breaking additions (new optional fields, new enum values)

### Current Version

All schemas are at **v1.0**.

### Compatibility Rules

1. **Producers** (CI checks) must specify the schema version they output
2. **Consumers** (local agents) must support all versions within a major version
3. Unknown fields should be ignored (forward compatibility)
4. Missing optional fields should use defaults (backward compatibility)

## Core Structure

Both schemas share a common base structure:

```json
{
  "spec_version": "v1.0",
  "findings_id": "uuid-v4",
  "timestamp": "2024-01-15T10:30:00Z",
  "source": {
    "check_name": "sdp-protocol-check",
    "workflow": "CI",
    "run_id": 1234567890,
    "repository": "fall-out-bug/sdp_lab",
    "branch": "feature/F077-findings-schema",
    "commit_sha": "abc123"
  },
  "configuration": { ... },
  "findings": [ ... ],
  "summary": { ... }
}
```

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `spec_version` | string | Schema version (e.g., `v1.0`) |
| `findings_id` | uuid | Unique identifier for this report |
| `timestamp` | datetime | When findings were generated |
| `source` | object | CI check metadata |
| `findings` | array | List of individual findings |

## Finding Structure

Each finding in the `findings` array has:

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `finding_key` | string | 16-char hex hash for deduplication |
| `severity` | enum | `error`, `warning`, `info`, `hint` |
| `category` | enum | See categories below |
| `file` | string | Relative file path |
| `message` | string | Human-readable description |

### Deduplication Key

The `finding_key` enables idempotent processing:

```
finding_key = hash(file + category + line + code)[:16]
```

This ensures:
- Same issue in same location = same key
- Different runs produce consistent keys
- Agents can track finding resolution over time

### Severity Levels

| Severity | Meaning | Action |
|----------|---------|--------|
| `error` | Must fix before merge | Block PR |
| `warning` | Should fix soon | Create Beads issue |
| `info` | FYI, no action required | Log only |
| `hint` | Suggestion for improvement | Optional follow-up |

### Categories (Protocol)

| Category | Description |
|----------|-------------|
| `frontmatter` | YAML frontmatter issues |
| `beads` | Beads section validation |
| `acceptance_criteria` | AC section problems |
| `feature_consistency` | Feature ID mismatches |
| `workstream_structure` | Workstream file structure |
| `roadmap` | ROADMAP.md consistency |
| `index` | INDEX.md consistency |
| `documentation` | General docs issues |

### Categories (Docs)

| Category | Description |
|----------|-------------|
| `broken_link` | Dead or invalid links |
| `missing_section` | Required sections absent |
| `outdated_reference` | References to moved/removed content |
| `formatting` | Markdown formatting issues |
| `changelog` | CHANGELOG.md problems |
| `consistency` | Cross-file inconsistencies |
| `accessibility` | Accessibility concerns |
| `spelling` | Typos and spelling errors |
| `structure` | Document structure issues |

## Remediation Hints

The `remediation` object provides actionable guidance:

```json
{
  "remediation": {
    "hint": "Add a bullet item with sdplab-<id> reference",
    "action": "add",
    "template": "- sdplab-abc123: F077-01 findings-schema",
    "doc_url": "https://sdp.dev/docs/beads-section"
  }
}
```

### Action Types

| Action | Description |
|--------|-------------|
| `add` | Add new content |
| `update` | Modify existing content |
| `remove` | Delete content |
| `rename` | Rename file/section |
| `fix` | Fix error/typo |
| `create` | Create new file |

## Example Payloads

### Protocol Findings Example

Generated from `sdp-protocol-check`:

```json
{
  "spec_version": "v1.0",
  "findings_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "timestamp": "2024-01-15T10:30:00Z",
  "source": {
    "check_name": "sdp-protocol-check",
    "workflow": "CI",
    "run_id": 1234567890,
    "run_number": 42,
    "repository": "fall-out-bug/sdp_lab",
    "branch": "feature/F077-findings-schema",
    "commit_sha": "abc123def456"
  },
  "configuration": {
    "strict_beads": true,
    "strict_all": false
  },
  "findings": [
    {
      "finding_key": "a1b2c3d4e5f67890",
      "severity": "error",
      "category": "beads",
      "code": "PLACEHOLDER_ID",
      "file": "docs/workstreams/backlog/00-077-01.md",
      "line": 20,
      "message": "Beads entry must reference concrete issue id (sdplab-<id>) in strict mode - placeholder detected",
      "remediation": {
        "hint": "Replace sdplab-XX with actual Beads issue ID",
        "action": "update",
        "template": "- sdplab-abc123: F077-01 findings-schema"
      },
      "context": {
        "feature_id": "F077",
        "ws_id": "00-077-01"
      }
    },
    {
      "finding_key": "b2c3d4e5f6789012",
      "severity": "warning",
      "category": "acceptance_criteria",
      "code": "MISSING_AC",
      "file": "docs/workstreams/backlog/00-078-01.md",
      "message": "Acceptance Criteria section must contain at least one checkbox item",
      "remediation": {
        "hint": "Add checkbox items for each acceptance criterion",
        "action": "add",
        "template": "- [ ] Criterion description"
      }
    }
  ],
  "summary": {
    "total": 2,
    "by_severity": {
      "error": 1,
      "warning": 1,
      "info": 0,
      "hint": 0
    },
    "by_category": {
      "beads": 1,
      "acceptance_criteria": 1
    }
  }
}
```

### Docs Findings Example

Generated from `sdp-doc-sync`:

```json
{
  "spec_version": "v1.0",
  "findings_id": "c3d4e5f6-7890-abcd-ef12-34567890abcd",
  "timestamp": "2024-01-15T10:35:00Z",
  "source": {
    "check_name": "sdp-doc-sync",
    "workflow": "CI",
    "run_id": 1234567890,
    "run_number": 42,
    "repository": "fall-out-bug/sdp_lab",
    "branch": "feature/F077-findings-schema",
    "commit_sha": "abc123def456"
  },
  "configuration": {
    "check_links": true,
    "strict_mode": false,
    "changelog_mode": true
  },
  "findings": [
    {
      "finding_key": "d4e5f67890abcdef",
      "severity": "error",
      "category": "broken_link",
      "code": "LINK_404",
      "file": "docs/protocol/CONTRACTS.md",
      "line": 45,
      "column": 1,
      "message": "Link to './schema/contracts/old.schema.json' does not exist",
      "remediation": {
        "hint": "Update link to point to correct schema file",
        "action": "update",
        "suggested_fix": "../../schema/contracts/runtime-decision.schema.json"
      },
      "context": {
        "link_target": "./schema/contracts/old.schema.json",
        "link_text": "old schema",
        "section": "Contract Schemas"
      }
    },
    {
      "finding_key": "e5f67890abcdef12",
      "severity": "warning",
      "category": "changelog",
      "code": "MISSING_CHANGELOG_ENTRY",
      "file": "docs/CHANGELOG.md",
      "message": "No changelog entry for commit abc123 on feature branch",
      "remediation": {
        "hint": "Add changelog entry following Keep a Changelog format",
        "action": "add"
      }
    }
  ],
  "summary": {
    "total": 2,
    "by_severity": {
      "error": 1,
      "warning": 1,
      "info": 0,
      "hint": 0
    },
    "by_category": {
      "broken_link": 1,
      "changelog": 1
    },
    "links_checked": 156,
    "files_checked": 42
  }
}
```

## CI Integration

### Output Location

CI checks should write findings to:

```
.sdp/findings/{check_name}-{timestamp}.json
```

Example:
```
.sdp/findings/sdp-protocol-check-20240115-103000Z.json
.sdp/findings/sdp-doc-sync-20240115-103500Z.json
```

### GitHub Actions Step

```yaml
- name: Run Protocol Check
  run: |
    mkdir -p .sdp/findings
    sdp-protocol-check --format json > .sdp/findings/sdp-protocol-check-$(date -u +%Y%m%d-%H%M%SZ).json
    if [ $? -ne 0 ]; then
      echo "::error::Protocol validation failed"
      exit 1
    fi
```

### Artifact Upload

```yaml
- name: Upload Findings
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: sdp-findings
    path: .sdp/findings/*.json
```

## Local Agent Consumption

Local agents can consume findings:

1. **Pull artifacts** from GitHub Actions run
2. **Parse JSON** using schema validation
3. **Deduplicate** using `finding_key`
4. **Create Beads issues** for actionable findings
5. **Track resolution** over multiple runs

### Example Agent Flow

```go
func ProcessFindings(findingsDir string) error {
    files, _ := filepath.Glob(filepath.Join(findingsDir, "*.json"))
    
    for _, file := range files {
        data, _ := os.ReadFile(file)
        
        var findings FindingsReport
        json.Unmarshal(data, &findings)
        
        for _, f := range findings.Findings {
            if f.Severity == "error" || f.Severity == "warning" {
                // Check if already tracked
                if !beads.ExistsByFindingKey(f.FindingKey) {
                    // Create new issue
                    beads.Create(beads.Issue{
                        Title: fmt.Sprintf("[%s] %s", f.Category, f.Message),
                        Body:  formatFindingBody(f),
                        Labels: []string{f.Severity, f.Category},
                        FindingKey: f.FindingKey,
                    })
                }
            }
        }
    }
    return nil
}
```

## Evolution Guidelines

When modifying schemas:

1. **Add optional fields** - Always allowed (minor version bump)
2. **Add enum values** - Always allowed (minor version bump)
3. **Remove fields** - Breaking change (major version bump)
4. **Change field types** - Breaking change (major version bump)
5. **Make optional required** - Breaking change (major version bump)

### Deprecation Process

1. Mark field as deprecated in documentation
2. Add `deprecated: true` to schema description
3. Maintain support for one major version
4. Remove in next major version

## Reference

- [JSON Schema specification](https://json-schema.org/)
- [CONTRACTS.md](./CONTRACTS.md) - Contract schemas
- [ROADMAP.md](../roadmap/ROADMAP.md) - Feature roadmap
