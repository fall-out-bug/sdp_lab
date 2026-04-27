# Decision Record v1 — Design Document

**Feature:** F151-03 (sdplab-hfk0.3)
**Internal namespace:** sdp-pr-gate
**Status:** Design v1

## Overview

A Decision Record captures the readiness decision for a pull request. It is append-only: decisions are never mutated or deleted, only superseded by new decisions.

## Decision Lifecycle

```text
PR opens → scope confirmed → evidence collected → passport generated
  → system recommendation (hold/rework)
  → human reviews passport
  → decision_owner finalizes (merge/hold/rework/escalate)
  OR
  → decision_owner overrides (override with reason)
```

### State transitions

```text
(None) → hold          (system: insufficient evidence, unresolved findings)
(None) → rework        (system: findings require code changes)
(None) → merge         (system: all checks pass — auto-merge if enabled)
hold → merge           (human: evidence recovered, ready to merge)
hold → rework          (human: findings need addressing)
hold → escalate        (human: needs higher-level decision)
hold → override        (human: explicit override with reason)
rework → merge         (human: rework completed, ready)
rework → hold          (human: rework introduced new issues)
rework → override      (human: explicit override with reason)
escalate → merge       (admin: escalation resolved, merge approved)
escalate → hold        (admin: escalation requires more work)
escalate → override    (admin: explicit override with reason)
merge → (terminal)     (decision finalized)
override → (terminal)  (override finalized)
```

### Conflict resolution

When multiple decision owners attempt to decide simultaneously:
1. First write wins (append-only log)
2. Second writer sees conflict error with the first decision
3. Second writer must either accept the first decision or override it
4. Override creates a new decision record referencing the previous one

Override by a different actor requires `admin` role. Override by the same `decision_owner` is always allowed.

## Required Fields

| Field | Type | Required | Rationale |
|---|---|---|---|
| `decision_id` | UUID | yes | Unique per decision entry |
| `passport_id` | UUID | yes | Links to the passport this decision is for |
| `decision` | enum | yes | One of: merge, hold, rework, escalate, override |
| `decided_by` | actor | yes | Who made the decision (human or system) |
| `decided_at` | datetime | yes | When the decision was made |
| `reason` | string | yes | Why this decision was made. Always required — no decision without accountability. |
| `evidence_snapshot_ref` | hash | yes | Evidence state at decision time |
| `audit_ref` | string | yes | Append-only audit log reference |
| `signed_by` | object | yes | Cryptographic signature. Every persisted decision must be tamper-proof. |
| `override_detail` | object | conditional | Required when decision is "override". Must include: trigger, original_decision, previous_decision_id. |

## Override Mechanics

Override is a first-class decision type, not a special case:

1. `decision` is set to `"override"`
2. `override_detail` object is required with:
   - `trigger`: how the override was initiated (comment, api, manual)
   - `original_decision`: what the system recommended (hold, rework, escalate)
   - `previous_decision_id`: UUID of the decision being overridden
3. `reason` is required and non-empty
4. `evidence_snapshot_ref` binds the override to a specific evidence state

### Override visibility

Overrides produce a visible marker in the passport:
- The passport `decision.result` shows `"override"`
- The passport `decision.override` object contains the full override detail
- The audit log contains an immutable entry with owner, reason, and timestamp

## Decision Signing

Each decision is cryptographically signed:

- `signed_by.signer`: identifier of the signing entity
- `signed_by.algorithm`: ed25519 (preferred) or rsa-pss-256
- `signed_by.signature`: signature over the canonical decision record (all fields except `signed_by`)

Signing ensures decisions cannot be forged or tampered with after the fact. The signing key belongs to the sdp-pr-gate service, not the human decision maker.

### Verification

```text
verify(canonical_json(record without signed_by), public_key, signature) → true/false
```

Failed verification means the decision record has been tampered with.

## Evidence Snapshot Linkage

Each decision references a specific evidence snapshot via `evidence_snapshot_ref`. This means:

1. The decision is reproducible: given the same evidence snapshot, the same passport would be generated
2. If evidence changes after the decision, the passport is marked as drifted
3. The snapshot is content-addressable (SHA-256 hash), so it is immutable

### Example scenarios

**Scenario 1: New commit after hold**
```
Decision: hold (evidence_snapshot_ref: hash-A)
  → new commit pushed
  → evidence re-collected
  → new evidence_snapshot_ref: hash-B
  → Decision: merge (evidence_snapshot_ref: hash-B)
```

**Scenario 2: Override**
```
Decision: hold (evidence_snapshot_ref: hash-A, decision_id: uuid-1)
  → override comment posted
  → Decision: override (evidence_snapshot_ref: hash-A, decision_id: uuid-2, previous: uuid-1)
```

Note: the override references the SAME evidence snapshot as the hold. No new evidence was collected.

## Audit Trail

The audit trail is an append-only log stored separately from passports:

```text
audit/
  └── {passport_id}/
      ├── {decision_id-1}.json  (immutable)
      ├── {decision_id-2}.json  (immutable)
      └── ...
```

Properties:
- Append-only: no mutation, no deletion
- Tamper-evident: each entry is signed
- Chronological: entries are ordered by `decided_at`
- Linked: each entry references the passport and optionally a previous decision

## Examples

### Standard merge decision

```json
{
  "decision_id": "01912234-5678-7000-8000-000000000010",
  "passport_id": "01912234-5678-7000-8000-000000000001",
  "schema_version": "v1",
  "decision": "merge",
  "decided_by": { "id": "bob", "type": "human", "name": "Bob Manager", "role": "decision_owner" },
  "decided_at": "2026-05-01T10:05:00Z",
  "reason": "All checks pass. Scope confirmed. No unresolved findings.",
  "evidence_snapshot_ref": "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "audit_ref": "audit:01912234-5678-7000-8000-000000000001:01912234-5678-7000-8000-000000000010",
  "signed_by": {
    "signer": "sdp-pr-gate-service",
    "algorithm": "ed25519",
    "signature": "base64-encoded-signature"
  }
}
```

### Override with reason

```json
{
  "decision_id": "01912234-5678-7000-8000-000000000011",
  "passport_id": "01912234-5678-7000-8000-000000000002",
  "schema_version": "v1",
  "decision": "override",
  "decided_by": { "id": "bob", "type": "human", "name": "Bob Manager", "role": "decision_owner" },
  "decided_at": "2026-05-01T14:10:00Z",
  "reason": "Client deadline accepted; non-blocking scanner finding tracked in #350",
  "evidence_snapshot_ref": "sha256:f1e2d3c4b5a6f1e2d3c4b5a6f1e2d3c4b5a6f1e2d3c4b5a6f1e2d3c4b5a6f1e2",
  "audit_ref": "audit:01912234-5678-7000-8000-000000000002:01912234-5678-7000-8000-000000000011",
  "override_detail": {
    "trigger": "comment",
    "original_decision": "hold",
    "previous_decision_id": "01912234-5678-7000-8000-000000000009"
  },
  "signed_by": {
    "signer": "sdp-pr-gate-service",
    "algorithm": "ed25519",
    "signature": "base64-encoded-signature"
  }
}
```
