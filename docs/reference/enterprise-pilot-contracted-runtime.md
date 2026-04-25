# Contracted Runtime Pilot Package

> **Goal**: Extend the CI-gate-only pilot with runtime schema validation, orchestration events, and handoff contracts.

## Overview

The contracted runtime mode adds runtime governance to the CI gates established in the CI-gate-only pilot. It introduces three schema-validated event types that coordinate agent actions at runtime:

1. **Orchestration events** — task lifecycle, phase transitions, handoffs
2. **Runtime decisions** — allow/ask/deny governance decisions
3. **Evidence integration** — findings and verification results feed back into decisions

**Prerequisite**: CI-gate-only pilot complete (see [enterprise-pilot-ci-gate-only.md](./enterprise-pilot-ci-gate-only.md)).

## What Changes from CI-Only Mode

| Aspect | CI-Only | Contracted Runtime |
|--------|---------|-------------------|
| Gate enforcement | Merge time only | Merge + runtime |
| Schema validation | CI only | Ingest precondition |
| Event model | Evidence log only | Full orchestration events |
| Agent coordination | None | Handoff contracts |
| Decision model | Pass/fail gates | Allow/ask/deny |
| Rollback | Manual | Evidence-preserved automatic |

## Schema Validation as Ingest Precondition

In contracted runtime mode, all events must pass schema validation before being accepted into the event log. This prevents malformed events from polluting the audit trail.

### Validation Flow

```
Agent emits event
    ↓
Schema validation (orchestration-event.schema.json)
    ↓ valid?                ↓ invalid
Accepted into event log    Rejected with error details
    ↓
Runtime decision evaluation
    ↓
Decision recorded (runtime-decision.schema.json)
```

### Enabling Schema Validation

```bash
# In .sdp/config.yml
runtime:
  mode: contracted
  schema_validation: strict  # reject invalid events
  # schema_validation: advisory  # log warnings, don't reject (default during pilot)
```

With `strict` mode, any event that doesn't match the `orchestration-event.schema.json` is rejected at ingest. With `advisory` mode (recommended for pilot), invalid events are logged but still accepted.

### Validating Events Manually

```bash
# Validate a single event
sdp contract validate-event --schema schema/contracts/orchestration-event.schema.json \
  --event .sdp/events/my-event.json

# Validate all events in log
sdp contract validate-events --log .sdp/log/events.jsonl
```

## Reference Adapters and Events Payload Flow

### Minimal Event Sequence

Here is the minimal event flow for a single agent task:

```json
// 1. Task started
{
  "spec_version": "v1.0",
  "event_id": "550e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2026-04-25T10:00:00Z",
  "source": {"system": "sdp-lab", "component": "coder-agent"},
  "event_type": "task.started",
  "payload": {"ws_id": "00-081-02", "action": "implement"},
  "context": {"workstream_id": "00-081-02", "feature_id": "F081"}
}

// 2. Phase transition
{
  "spec_version": "v1.0",
  "event_id": "550e8400-e29b-41d4-a716-446655440002",
  "timestamp": "2026-04-25T10:05:00Z",
  "source": {"system": "sdp-lab", "component": "coder-agent"},
  "event_type": "phase.transition",
  "payload": {"from": "building", "to": "testing"},
  "metadata": {"correlation_id": "corr-001"}
}

// 3. Evidence generated
{
  "spec_version": "v1.0",
  "event_id": "550e8400-e29b-41d4-a716-446655440003",
  "timestamp": "2026-04-25T10:08:00Z",
  "source": {"system": "sdp-lab", "component": "coder-agent"},
  "event_type": "evidence.generated",
  "payload": {"type": "generation", "files_changed": ["internal/x/foo.go"]}
}

// 4. Quality gate passed
{
  "spec_version": "v1.0",
  "event_id": "550e8400-e29b-41d4-a716-446655440004",
  "timestamp": "2026-04-25T10:10:00Z",
  "source": {"system": "sdp-lab", "component": "reviewer-agent"},
  "event_type": "quality.gate.passed",
  "payload": {"gate": "evidence-gate", "findings_count": 0}
}

// 5. Task completed
{
  "spec_version": "v1.0",
  "event_id": "550e8400-e29b-41d4-a716-446655440005",
  "timestamp": "2026-04-25T10:12:00Z",
  "source": {"system": "sdp-lab", "component": "coder-agent"},
  "event_type": "task.completed",
  "payload": {"ws_id": "00-081-02", "duration_sec": 720}
}
```

### Handoff Event Flow

When one agent hands off to another:

```json
// Agent A initiates handoff
{
  "spec_version": "v1.0",
  "event_id": "660e8400-...",
  "timestamp": "2026-04-25T10:15:00Z",
  "source": {"system": "sdp-lab", "component": "coder-agent"},
  "event_type": "handoff.initiated",
  "payload": {
    "to_agent": "reviewer-agent",
    "artifacts": ["internal/x/foo.go", "internal/x/foo_test.go"],
    "context": {"ws_id": "00-081-02", "summary": "Implementation complete"}
  },
  "metadata": {"correlation_id": "handoff-001"}
}

// Agent B completes handoff
{
  "spec_version": "v1.0",
  "event_id": "660e8401-...",
  "timestamp": "2026-04-25T10:20:00Z",
  "source": {"system": "sdp-lab", "component": "reviewer-agent"},
  "event_type": "handoff.completed",
  "payload": {
    "from_agent": "coder-agent",
    "result": "approved",
    "findings_count": 0
  },
  "metadata": {"correlation_id": "handoff-001"}
}
```

### Runtime Decision Example

```json
{
  "spec_version": "v1.0",
  "decision_id": "770e8400-e29b-41d4-a716-446655440001",
  "timestamp": "2026-04-25T10:25:00Z",
  "decision_type": "scope.boundary",
  "decision": "allow",
  "reason": {
    "code": "SCOPE_VALID",
    "message": "All changed files are within declared workstream scope"
  },
  "context": {
    "request": {
      "action": "write",
      "resource": "internal/x/foo.go"
    },
    "actor": {"type": "agent", "id": "coder-agent"},
    "workstream_id": "00-081-02"
  },
  "evidence": [
    {
      "type": "test_result",
      "reference": ".sdp/evidence/00-081-02-generation.json",
      "summary": "All tests passing"
    }
  ]
}
```

## Migration from CI-Gate-Only Mode

### Step 1: Update Configuration

```bash
# In .sdp/config.yml, add:
runtime:
  mode: contracted
  schema_validation: advisory  # start with advisory during pilot
```

### Step 2: Add Event Emission Points

In your agent scripts, emit events at key lifecycle points:

```bash
# Task start
sdp event emit --type task.started \
  --source '{"system":"sdp-lab","component":"my-agent"}' \
  --payload '{"ws_id":"00-001-01","action":"implement"}'

# ... agent work ...

# Task complete
sdp event emit --type task.completed \
  --source '{"system":"sdp-lab","component":"my-agent"}' \
  --payload '{"ws_id":"00-001-01","duration_sec":300}'
```

### Step 3: Add Handoff Coordination

For multi-agent workflows:

```bash
# Agent A hands off to Agent B
sdp event emit --type handoff.initiated \
  --source '{"system":"sdp-lab","component":"agent-a"}' \
  --payload '{"to_agent":"agent-b","artifacts":["file1.go"]}' \
  --correlation-id "$(uuidgen)"
```

### Step 4: Validate Integration

```bash
# Check all events validate against schemas
sdp contract validate-events --log .sdp/log/events.jsonl

# Verify runtime decisions are being recorded
sdp log show --type decision
```

### Step 5: Switch to Strict Mode

After validating during pilot:

```yaml
# .sdp/config.yml
runtime:
  mode: contracted
  schema_validation: strict  # now reject invalid events
```

## Measurable Success Signals

The contracted runtime pilot is successful when:

| Signal | Measurement | Target |
|--------|-------------|--------|
| Schema validation pass rate | `% of events passing schema validation` | > 99% |
| Handoff completion rate | `% of handoffs completed within 5 min` | > 95% |
| Decision latency | `P50 time from event to decision` | < 100ms |
| Event coverage | `% of agent actions producing events` | > 80% |
| Zero invalid events in strict mode | `count of rejected events in strict mode` | 0 |

### Measuring During Pilot

```bash
# Schema pass rate
sdp contract validate-events --log .sdp/log/events.jsonl --stats

# Handoff metrics
sdp log show --type handoff | sdp metrics handoff

# Decision latency
sdp log show --type decision | sdp metrics latency
```

## Schema Reference

| Schema | File | Purpose |
|--------|------|---------|
| Orchestration Event | `schema/contracts/orchestration-event.schema.json` | Task lifecycle, handoff, phase events |
| Runtime Decision | `schema/contracts/runtime-decision.schema.json` | Allow/ask/deny governance decisions |
| Evidence Event | `schema/evidence.schema.json` | Plan, generation, verification, approval events |

## Next Steps

After completing the contracted runtime pilot:

1. **Full orchestration** — enable event-driven agent loop with automatic phase transitions
2. **Staged rollout** — use `sdp deploy --staged` for canary deployments with metric confirmation
3. **Rollback playbook** — see [enterprise-pilot-rollback.md](./enterprise-pilot-rollback.md)
