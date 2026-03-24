# ADR: Beads as Operational Source of Truth

**Status:** Accepted
**Date:** 2026-03-24
**Supersedes:** None (new decision)
**Context:** Beads-First Control Tower pivot

---

## Decision

Beads is the **sole durable operational source of truth** for SDP Lab execution state.

Everything else — FeatureCard, control store, board snapshots — is a **semantic artifact, derived projection, or verification layer**.

## Motivation

SDP currently maintains split-brain state:
- `FeatureCard` + `.sdp/control/*.yaml` as an independent lifecycle store
- Beads as a dependency-aware durable graph
- Orchestration runtime state in separate artifacts
- Board/snapshot as yet another projection

This is acceptable as a transitional stage, but not as target architecture. Continuing with parallel state stores leads to:
- Status divergence (`ready` / `blocked` / `in_progress` disagreeing between layers)
- Orchestration policy duplicating Beads semantics
- Ambiguity about what constitutes truth
- Difficulty building A2A, federation, governance, and dogfooding

## Consequences

### What Beads owns (operational truth)
- Work item identity
- Status
- Priority
- Dependencies
- Readiness / blockers
- Gates
- Claim / assignment semantics
- Parent-child graph
- Labels
- Machine metadata (`sdp.*`)

### What SDP owns (governance + semantics)
- Intent normalization
- Clarification semantics
- Contract generation
- Provenance chain
- Evidence requirements
- Compliance evaluation
- Routing policy
- Human/admin UX
- Views and summaries

### What opencode owns (execution)
- Actual execution
- Code changes
- Tool use
- Runtime logs
- Local produced artifacts

### What A2A owns (transport)
- External task API
- Transport contract
- Authn/authz at API boundary

## Constraints

- **No shadow lifecycle state** — SDP must not maintain a parallel ready queue
- **No rewriting checkpoint loop** during storage migration
- **No second task graph** inside SDP
- **FeatureCard is a semantic artifact**, not a competing source of truth
- **Snapshots and boards are derived projections only**, never source of truth
- **Artifacts (evidence, provenance, contracts) remain file-based** — not stored as Beads blobs

## Migration Path

R0 → Canon (this ADR)
R1 → Repository boundary extraction (code)
R2 → Beads mapping canon (data model)
R3 → Beads adapter MVP
R4 → Dual-write / shadow read
R5 → Cutover to Beads-first
R6 → Contract / evidence tightening
R7 → Product surface cleanup
