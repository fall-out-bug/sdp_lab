# Passport Schema v1 — Design Rationale

**Feature:** F151-01 (sdplab-hfk0.1)
**Internal namespace:** sdp-pr-gate
**Display name:** ChangePassport
**Status:** Design v1

## Overview

A Change Passport is an evidence-backed merge-readiness record for one pull request. It contains 7 sections derived from the ChangePassport manifesto v2 §"The Change Passport":

1. **Intent** — what the change claims to solve
2. **Scope** — what is in, out, and unknown
3. **Actors** — humans, agents, tools, and systems that touched the change
4. **Evidence** — summary of collected evidence events and provider states
5. **Findings** — failures, fixes, unresolved objections, blocked checks
6. **Risk** — known uncertainty and accepted risk
7. **Decision** — merge, hold, rework, escalate, or explicit override

## Schema Location

`schema/sdp-pr-gate/passport.schema.json`
`$id`: `https://sdp.dev/schema/sdp-pr-gate/passport/v1`

## Field-by-Field Rationale

### Root-level identity

| Field | Type | Required | Rationale |
|---|---|---|---|
| `passport_id` | UUID | yes | Unique instance identifier. Enables cross-referencing between systems. |
| `schema_version` | "v1" | yes | Locked to "v1" for this schema. Enables safe evolution. |
| `created_at` | datetime | yes | First creation timestamp. Immutable. |
| `generated_at` | datetime | yes | Timestamp of this specific generation. Changes on regeneration. |
| `repository` | object | yes | Identifies the repository. Platform-agnostic. |
| `pull_request` | object | yes | Identifies the PR. Includes head SHA for integrity binding. |

### Annotations (not observed facts)

The `annotations` array is deliberately separate from all 7 sections. This enforces the manifesto principle: **manual annotations cannot overwrite observed facts.** Annotations are additive-only, timestamped, and attributed.

### Intent

Derived from PR template, commit messages, linked issues, or labels. The `source` field records where the intent came from, enabling traceability and trust assessment.

`claimed_type` uses the author's claim, not a system classification. This preserves the author's intent without imposing external interpretation.

### Scope

Each scope item has a `classification`: `in_scope`, `out_of_scope`, or `unknown`. The `unknown` classification is critical — it forces the decision owner to explicitly resolve ambiguity rather than silently accepting scope gaps.

`confirmed_by` and `confirmed_at` track human confirmation. Unconfirmed scope is treated as `unknown` by default.

`scope_delta` captures changes detected after initial scope confirmation (e.g., new files added in later commits).

### Actors

Every contributor is classified as `human`, `agent`, `system`, or `tool`. This supports the manifesto principle of plural execution with singular accountability.

`contribution_type` captures how the actor participated: author, committer, reviewer, agent, CI system, scanner, or tool. One actor can have multiple entries with different contribution types.

`decision_owner` is a single person. The manifesto requires: "One accountable owner must decide whether it is ready."

### Evidence

The evidence section is a **summary**, not the full event stream. It lists providers with their ingestion status, event counts, and any errors.

`collection_status` is the aggregate state: `complete`, `partial`, `degraded`, or `pending`. This is what drives the passport-level assessment.

`evidence_snapshot_ref` is a content-addressable hash pointing to the full evidence bundle. This enables:
- Reproducibility: the passport can be regenerated from the snapshot
- Integrity: hash comparison detects drift
- Audit: the snapshot is immutable once sealed

### Findings

Each finding has a `status`: `open`, `resolved`, `accepted_risk`, `deferred`, or `false_positive`. The `resolution` object captures who resolved it, when, and why.

Findings reference their source provider, enabling cross-reference with evidence events.

### Risk

Risk items are categorized: `scope_violation`, `missing_evidence`, `unresolved_finding`, `actor_unknown`, `test_gap`, `dependency_risk`, or `other`. Each category maps to a concrete concern.

Risk acceptance requires `accepted_by` and `accepted_at`. Unsigned risk is treated as unresolved.

### Decision

The decision enum is fixed: `merge`, `hold`, `rework`, `escalate`, `override`. No other values are possible.

When `result` is `override`, the `override` object is required with `reason`, `evidence_snapshot_ref`, `audit_ref`, and `trigger` (how the override was initiated).

`check_status` maps to the GitHub Check status: `ready` → pass, `hold` → action_required, `rework` → failure, `escalate` → neutral.

### Integrity

The integrity block ensures the passport is tamper-evident:

- `passport_hash`: hash of the passport content (excluding the integrity block itself)
- `evidence_snapshot_hash`: hash of the evidence bundle used to generate this passport
- `previous_passport_hash`: enables chain verification when passports are regenerated
- `drift_detected` + `drift_reason`: surface when reproduced state differs from referenced evidence

## Observed Facts vs Annotations

The schema enforces separation at the structural level:

| Section | Type | Mutable |
|---|---|---|
| evidence.providers[].status | observed | no (set by ingestion) |
| evidence.providers[].event_count | observed | no (set by ingestion) |
| findings.items[].status | observed (initially) | yes (via resolution) |
| findings.items[].resolution | human action | yes (append-only) |
| decision.result | decision | yes (single write) |
| annotations[] | human annotation | yes (append-only) |

Observed fields are set by the ingestion pipeline and cannot be modified by annotations. Annotations can add context but never overwrite.

## Versioning Strategy

### When does v2 happen?

v2 is required when ANY of:
- A new required field is added
- An existing field semantics change meaningfully
- A new section is added to the passport
- An enum value is removed

v2 is NOT required for:
- New optional fields
- New enum values (additive)
- New metadata properties
- Documentation changes

### Backwards compatibility

Consumers MUST:
- Ignore unknown properties (JSON Schema `additionalProperties: false` applies to producers, not consumers)
- Treat new enum values as "unknown" rather than erroring
- Accept missing optional fields

### Schema evolution path

```text
v1 (this schema)
  → v1.1 (optional fields, new enums, metadata extensions)
  → v2 (breaking: new required fields, semantic changes)
```

## Hash Reproducibility

The passport hash is computed over a canonical JSON representation:

1. Remove the `integrity` block
2. Sort object keys lexicographically
3. Encode as UTF-8 JSON without whitespace
4. Compute SHA-256

The evidence snapshot hash follows the same canonical form applied to the full evidence event bundle.

Drift detection: if the passport is regenerated from the same evidence snapshot but produces a different hash, `drift_detected` is set to `true` with an explanation in `drift_reason`.
