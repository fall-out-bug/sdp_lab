# SDP Protocol Contracts

This document defines the versioned protocol contracts for orchestration, runtime decisions, status, instructions, and OpenSpec imports shared by OSS and enterprise surfaces.

## Overview

SDP uses five primary contract types:

1. **OrchestrationEvent** - Events that occur during workflow orchestration
2. **RuntimeDecision** - Governance decisions made during execution
3. **StatusView** - Machine-readable project state for guided next-step UX
4. **InstructionPayload** - Next-step instruction payload used by SDP agents
5. **OpenSpecImportPayload** - Normalized contract for importing OpenSpec artifacts

All contracts follow semantic versioning and are designed for backward compatibility.

## Schemas

- [OrchestrationEvent](../../schema/contracts/orchestration-event.schema.json)
- [RuntimeDecision](../../schema/contracts/runtime-decision.schema.json)
- [StatusView](../../schema/contracts/status-view.schema.json)
- [InstructionPayload](../../schema/contracts/instructions.schema.json)
- [OpenSpecImportPayload](../../schema/contracts/openspec-import.schema.json)

## Versioning Strategy

### Schema Version Format

All contracts use a `spec_version` field with format: `v{MAJOR}.{MINOR}`

Examples:
- `v1.0` - Initial version
- `v1.1` - Backward-compatible additions
- `v2.0` - Breaking changes

### Compatibility Rules

#### Backward-Compatible Changes (Minor version bump)

These changes are **safe** and maintain compatibility:

- Adding new optional fields
- Adding new enum values (with default handling)
- Expanding regex patterns (more permissive)
- Adding new event types or decision types
- Relaxing validation constraints

#### Breaking Changes (Major version bump)

These changes **require migration**:

- Removing required fields
- Changing field types
- Narrowing validation constraints
- Renaming fields
- Removing enum values

### Version Negotiation

Consumers should:

1. Check `spec_version` field
2. Reject unknown major versions
3. Accept minor versions >= their minimum supported version
4. Gracefully handle unknown optional fields

Example:
```go
func parseOrchestrationEvent(data []byte) (*OrchestrationEvent, error) {
    var event OrchestrationEvent
    if err := json.Unmarshal(data, &event); err != nil {
        return nil, err
    }
    
    // Version check
    major, minor, err := parseVersion(event.SpecVersion)
    if err != nil {
        return nil, fmt.Errorf("invalid spec_version: %w", err)
    }
    
    if major > 1 {
        return nil, fmt.Errorf("unsupported major version: %s", event.SpecVersion)
    }
    
    if major == 1 && minor < 0 {
        return nil, fmt.Errorf("unsupported version: %s", event.SpecVersion)
    }
    
    return &event, nil
}
```

## Evolution Guidelines

### Adding New Fields

When adding new optional fields:

1. Update schema with new field (optional)
2. Increment minor version
3. Document field in CHANGELOG
4. Ensure producers populate field when available
5. Ensure consumers handle missing field gracefully

### Deprecating Fields

When deprecating fields:

1. Mark field as `deprecated` in schema description
2. Keep field for at least 2 major versions
3. Provide migration path in documentation
4. Update examples to use new fields
5. Remove field in next major version

### Changing Semantics

When changing field meaning:

1. Create new field with clear name
2. Deprecate old field
3. Provide migration period
4. Update all producers first
5. Update consumers gradually

## Example Payloads

### OMO (OhMyOpenCode) Adapter

#### OrchestrationEvent Example

```json
{
  "spec_version": "v1.0",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-03-01T18:00:00Z",
  "source": {
    "system": "omo",
    "component": "analyst-agent",
    "version": "1.2.0"
  },
  "event_type": "task.completed",
  "payload": {
    "task_id": "F068-01-implementation",
    "result": "success",
    "artifacts": [
      "schema/contracts/orchestration-event.schema.json",
      "schema/contracts/runtime-decision.schema.json"
    ]
  },
  "metadata": {
    "correlation_id": "session-abc123",
    "labels": {
      "priority": "high",
      "workstream": "00-068-01"
    }
  },
  "context": {
    "workstream_id": "00-068-01",
    "feature_id": "F068",
    "beads_id": "sdplab-iix",
    "session_id": "ses_xyz789",
    "git_context": {
      "branch": "feature/F068-contracts",
      "commit_sha": "a1b2c3d4e5f",
      "repo_url": "https://github.com/fall-out-bug/sdp_lab"
    }
  }
}
```

#### RuntimeDecision Example

```json
{
  "spec_version": "v1.0",
  "decision_id": "660e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-03-01T18:05:00Z",
  "decision_type": "scope.boundary",
  "decision": "deny",
  "reason": "Proposed changes exceed workstream scope",
  "context": {
    "workstream_id": "00-068-01",
    "beads_id": "sdplab-iix"
  },
  "details": {
    "requested_files": [
      "internal/k8s/controller.go",
      "deploy/manifests/production.yaml"
    ],
    "scope_boundary": {
      "allowed_patterns": ["schema/**", "docs/protocol/**"],
      "denied_patterns": ["internal/k8s/**", "deploy/**"]
    }
  },
  "automated": true,
  "actor": {
    "type": "system",
    "id": "sdp-guard"
  }
}
```

### Gas Town (Enterprise) Adapter

#### OrchestrationEvent Example

```json
{
  "spec_version": "v1.0",
  "event_id": "770e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-03-01T19:00:00Z",
  "source": {
    "system": "gas-town",
    "component": "k8s-agent-controller",
    "version": "2.1.0"
  },
  "event_type": "phase.transition",
  "payload": {
    "from_phase": "analyst",
    "to_phase": "coder",
    "handoff": {
      "analyst_output": "/artifacts/analyst/F068-01.json",
      "coder_prompt": "/prompts/coder/feature-implementation.yaml"
    }
  },
  "metadata": {
    "correlation_id": "workflow-xyz456",
    "trace_context": {
      "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e473-00f067a1f2e3d4c5b6a7b8-01",
      "tracestate": "gas-town=v1,gateway=prod"
    }
  },
  "context": {
    "workstream_id": "00-068-01",
    "feature_id": "F068",
    "git_context": {
      "branch": "main",
      "commit_sha": "b2c3d4e5f6a7",
      "repo_url": "https://github.com/fall-out-bug/sdp_lab"
    }
  }
}
```

#### RuntimeDecision Example

```json
{
  "spec_version": "v1.0",
  "decision_id": "880e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-03-01T19:10:00Z",
  "decision_type": "model.selection",
  "decision": "allow",
  "reason": "Model requested by enterprise customer within allowed list",
  "context": {
    "workstream_id": "00-068-01"
  },
  "details": {
    "requested_model": {
      "provider": "anthropic",
      "model_id": "claude-3-opus",
      "capabilities": ["reasoning", "code-generation"]
    },
    "policy": {
      "allowed_providers": ["openai", "anthropic", "azure-openai"],
      "allowed_models": ["gpt-4", "claude-3-opus", "claude-3-sonnet"],
      "byom_enabled": true
    }
  },
  "automated": true,
  "actor": {
    "type": "system",
    "id": "gas-town-policy-engine"
  },
  "policy_reference": {
    "opa_policy": "enterprise/model-selection.rego",
    "policy_version": "v1.2.0"
  }
}
```

## Integration Checklist

When integrating these contracts:

- [ ] Validate incoming events against JSON schema
- [ ] Check `spec_version` before processing
- [ ] Handle unknown optional fields gracefully
- [ ] Include correlation IDs for tracing
- [ ] Set appropriate decision values (allow/ask/deny)
- [ ] Provide clear reasons for decisions
- [ ] Include policy references for governance
- [ ] Test backward compatibility with older versions
- [ ] Document adapter-specific payload schemas
- [ ] Implement proper error handling for unknown versions

## Status and Instructions Contracts

### StatusView

The `StatusView` contract provides a machine-readable representation of project state used by `sdp status --json` and consumed by guided next-step UX.

Key fields:
- `version` - Contract version
- `environment` - Project environment status (git, SDP config, directories)
- `workstreams` - Workstream counts and categorized lists (ready, in_progress, blocked, failed)
- `next_action` - Human-readable description of next action
- `next_step` - Machine-readable `InstructionPayload` for execution

Example usage:
```go
status := &StatusView{
    Version: "1.0",
    HasGit: true,
    HasSDP: true,
    Workstreams: WorkstreamStatus{
        Ready: []WorkstreamStatus{
            {ID: "00-068-02", Status: "ready", Priority: 1},
        },
    },
    NextAction: "Implement Adapter SDK",
    NextStep: InstructionPayload{
        Command: "sdp build 00-068-02",
        Reason: "Ready to implement",
        Confidence: 0.95,
    },
}
```

### InstructionPayload

The `InstructionPayload` contract provides next-step instructions that can be executed by agents or displayed to users.

Key fields:
- `action_id` - Unique identifier for this instruction
- `command` - Executable command string
- `reason` - Human-readable explanation
- `confidence` - Confidence score (0-1)
- `category` - Instruction category (execution, recovery, planning, information, setup)
- `alternatives` - Optional alternative commands
- `required_context` - Required context for execution
- `policy_expectations` - Relevant policy constraints
- `evidence_expectations` - Required evidence artifacts

## OpenSpec Import Contract

### OpenSpecImportPayload

The `OpenSpecImportPayload` contract provides a normalized format for importing OpenSpec proposal, spec, design, and task artifacts into SDP.

Key features:
- **Source metadata** - Tracks source location, hash, and version for change detection
- **Mapping confidence** - Provides confidence scores for field mappings
- **Ambiguity tracking** - Lists unresolved mapping ambiguities requiring human review
- **Artifact normalization** - Normalizes proposal, spec, design, and task artifacts
- **Validation results** - Includes validation outcomes and errors
- **Deterministic identity** - Source hash enables repeatable import detection

Example usage:
```go
import := &OpenSpecImportPayload{
    SpecVersion: "v1.0",
    ImportID: "import-001",
    SourceMetadata: SourceMetadata{
        SourceType: "openspec",
        SourceLocation: "https://github.com/example/proposals",
        SourceHash: "a3d5e9f4...",
    },
    MappingConfidence: MappingConfidence{
        OverallScore: 0.92,
        ConfidenceLevel: "high",
    },
    Artifacts: Artifacts{
        Proposal: ProposalArtifact{...},
        Spec: SpecArtifact{...},
        Design: DesignArtifact{...},
        Tasks: []TaskArtifact{...},
    },
}
```

## Changelog

### v1.1 (2026-04-26)
- Added StatusView contract for machine-readable project state
- Added InstructionPayload contract for next-step instructions
- Added OpenSpecImportPayload contract for OpenSpec artifact imports
- Added SDK examples for OMO, Beads, and Gas Town adapters
- Enhanced documentation with new contract sections

### v1.0 (2026-03-01)
- Initial release
- OrchestrationEvent contract
- RuntimeDecision contract
- OMO and Gas Town adapter examples
- Backward compatibility guidelines
