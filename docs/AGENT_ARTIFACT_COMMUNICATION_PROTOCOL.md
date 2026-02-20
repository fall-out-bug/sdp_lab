# Agent Artifact Communication Protocol (Draft)

Status: draft for `sdp_dev-2aq.10`

## Problem

Task phase `Completed` is not enough to prove useful agent output. We need semantic success and anti-slacking checks across multiple roles.

## Design

- Use a mediator pattern: agents do not chat directly.
- Each role writes a final JSON envelope artifact.
- Reviewer consumes analyst/coder envelopes and must reference their artifact IDs.
- Orchestrator decides run success only after envelope validation.

## Envelope schema (required)

```json
{
  "run_id": "run-...",
  "role": "analyst|coder|reviewer",
  "status": "ok|needs_changes",
  "summary": "short summary",
  "artifacts": [
    {"id": "a1", "type": "analysis|plan|patch|verdict", "content": "..."}
  ]
}
```

## Transport and storage

Phase 1 (prototype):

- envelope emitted in task log (single source for now)
- orchestrator parses and validates required keys

Phase 2 (hardened):

- write envelopes to immutable run storage (`ConfigMap` or object storage key by `run_id/role`)
- keep log copy only for debugging

## Validation gates

- infrastructure gate: Task phase is `Completed`
- semantic gate: envelope exists, schema valid, role matches, status valid
- dependency gate:
  - reviewer must run only after analyst+coder envelopes are valid
  - reviewer envelope must include references to consumed artifacts

## Failure policy

- missing/invalid envelope => run fails
- model/provider error in logs => run fails
- reviewer without role references => run fails

## Why this is reliable

- prevents false-positive `Completed`
- prevents silent no-op outputs
- keeps deterministic audit chain per run
