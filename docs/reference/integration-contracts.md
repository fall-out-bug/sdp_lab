# Integration Contracts Reference

SDP integration contracts define the protocol boundaries between components. Each contract is a JSON Schema that governs event exchange at runtime.

## Contract Catalog

| Contract | Schema | Used By | Mode |
|----------|--------|---------|------|
| Orchestration Event | `schema/contracts/orchestration-event.schema.json` | All agents | Contracted runtime+ |
| Runtime Decision | `schema/contracts/runtime-decision.schema.json` | Governance layer | Contracted runtime+ |
| Evidence Event | `schema/evidence.schema.json` | All skills | CI-only+ |
| Instructions | `schema/contracts/instructions.schema.json` | Harness adapters | All modes |
| Feature Card | `schema/contracts/feature-card.schema.json` | Planning tools | All modes |

## Event Types

### Orchestration Events

Emitted by agents during task execution. See [orchestration-event.schema.json](../../schema/contracts/orchestration-event.schema.json).

| Event Type | Description | Required Fields |
|------------|-------------|-----------------|
| `task.started` | Agent begins work on a task | `payload.ws_id`, `payload.action` |
| `task.completed` | Agent finishes a task | `payload.ws_id`, `payload.duration_sec` |
| `task.failed` | Agent encounters unrecoverable error | `payload.ws_id`, `payload.error` |
| `phase.transition` | Agent moves between phases | `payload.from`, `payload.to` |
| `handoff.initiated` | Agent hands work to another agent | `payload.to_agent`, `payload.artifacts` |
| `handoff.completed` | Receiving agent confirms handoff | `payload.from_agent`, `payload.result` |
| `decision.made` | Governance decision recorded | Links to runtime-decision schema |
| `evidence.generated` | Agent produces evidence artifact | `payload.type`, `payload.files_changed` |
| `quality.gate.passed` | Quality gate succeeds | `payload.gate`, `payload.findings_count` |
| `quality.gate.failed` | Quality gate fails | `payload.gate`, `payload.findings` |

### Runtime Decisions

Governance decisions with allow/ask/deny semantics. See [runtime-decision.schema.json](../../schema/contracts/runtime-decision.schema.json).

| Decision Type | Description | Typical Outcome |
|---------------|-------------|-----------------|
| `scope.boundary` | File change within declared scope | allow/deny |
| `test.coverage` | Test coverage meets threshold | allow/deny |
| `security.approval` | Security scan results acceptable | allow/ask/deny |
| `review.status` | Code review approved | allow/deny |
| `evidence.validity` | Evidence chain intact | allow/deny |
| `quality.gate` | Quality gate passed | allow/deny |
| `resource.limit` | Resource usage within bounds | allow/ask |
| `model.selection` | Model selection appropriate | allow/ask |

## Validation

All contracts use JSON Schema draft 2020-12. Validate with:

```bash
# Single event
sdp contract validate-event --event my-event.json

# Batch validation
sdp contract validate-events --log .sdp/log/events.jsonl

# Specific schema
sdp contract validate-event --schema schema/contracts/runtime-decision.schema.json \
  --event my-decision.json
```

## Versioning

Contracts follow semantic versioning via the `spec_version` field. Breaking changes increment the major version; additive changes increment the minor version.

Current versions:
- `orchestration-event`: v1.0
- `runtime-decision`: v1.0
- `evidence-event`: v1 (no spec_version field, uses schema $id)

## Examples

See `schema/contracts/examples/` for complete event samples:
- `task-lifecycle.json` — full task started → completed sequence
- `handoff-flow.json` — inter-agent handoff with correlation
- `runtime-decision-allow.json` — allow decision with evidence
- `runtime-decision-deny.json` — deny decision with reason
