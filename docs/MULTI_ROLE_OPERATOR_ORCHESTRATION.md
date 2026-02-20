# Multi-Role Operator Orchestration (SDP)

Status: draft design for O1-03 and O2-01

## Objective

Run multiple agent roles in Kubernetes via kubeopencode, while preserving SDP semantics:

- deterministic run lifecycle
- traceable role outputs
- reviewer gate before final PR publication

## Roles

- `analyst`: requirement decomposition and risk notes
- `coder`: implementation proposal and patch plan
- `reviewer`: synthesis, quality/risk verdict, next-action recommendation

## Orchestration model

1. Orchestrator creates a run ID (`run-<timestamp>`).
2. Spawn `analyst` and `coder` Tasks in parallel.
3. Wait for both to reach terminal phase (`Completed|Failed`).
4. Collect both outputs from task logs.
5. Spawn `reviewer` Task with aggregated context.
6. Persist run trace (`run-id`, task names, phases, outputs, verdict).

This keeps role execution in operator resources, while orchestration and policy stay in SDP.

## Inter-agent communication contract

Communication is mediator-based (orchestrator hub), not direct pod-to-pod chat.

Envelope format (embedded in task logs):

```json
{
  "run_id": "run-20260219-2300",
  "role": "analyst",
  "status": "ok",
  "summary": "...",
  "artifacts": [
    {"type": "risk_note", "content": "..."},
    {"type": "task_breakdown", "content": "..."}
  ]
}
```

Rules:

- each role emits one final envelope
- reviewer receives analyst/coder envelopes as input context
- orchestrator stores final trace for audit and PR linkage

## Why mediator pattern

- deterministic and auditable flow
- no direct network permissions between role pods
- easier policy enforcement at a single control point
- portable to upstream operator with minimal assumptions

## Mapping to SDP

- beads/FSM remains source of truth for issue progression
- kubeopencode Task/Agent CRDs become execution substrate
- strict evidence includes role envelopes and run trace links

## Immediate prototype scope

- install kubeopencode in `kubeopencode-system`
- create 3 Agents (`analyst`, `coder`, `reviewer`)
- run one probe with parallel analyst/coder + sequential reviewer
- capture logs and emit run summary
