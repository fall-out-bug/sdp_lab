# OpenSpec Integration Plan for SDP

> **Status:** Draft design
> **Date:** 2026-03-08
> **Goal:** Adopt the best OpenSpec planning ergonomics in SDP without weakening evidence, policy, or execution governance

---

## Context

OpenSpec is strong at planning UX:

- action-oriented workflow
- ready/blocked/next-action state
- agent-friendly JSON instructions
- brownfield-friendly change artifacts

SDP is strong at trust and control:

- evidence and attestations
- policy enforcement
- review and provenance
- workstream and Beads execution discipline

The design goal is not to make SDP "become OpenSpec". The goal is to let SDP borrow OpenSpec's best planning surfaces while keeping SDP as the canonical trust layer.

---

## Summary

We should integrate OpenSpec ideas in three layers:

1. **Contract layer** - versioned JSON contracts for status, instructions, and OpenSpec import payloads.
2. **UX layer** - OpenSpec-style next-step guidance in SDP CLI output and first-run flows.
3. **Interop layer** - optional import from OpenSpec change artifacts into SDP planning primitives.

We should not:

- replace Beads or SDP workstreams with OpenSpec folders
- make OpenSpec a runtime dependency for core SDP flows
- downgrade evidence, review, or policy gates to match OpenSpec's lighter model

---

## Current-State Drift

`docs/reference/2026-02-14-competitive-analysis-sdp-vs-oss.md` maps OpenSpec-inspired work to `00-068-03`, `00-069-01`, `00-069-03`, and `00-072-02`. That mapping is stale.

Those IDs are now used for different scope in the current backlog:

- `docs/workstreams/backlog/00-068-03.md`
- `docs/workstreams/backlog/00-069-01.md`
- `docs/workstreams/backlog/00-069-03.md`
- `docs/workstreams/backlog/00-072-02.md`

Decision: this plan must not reuse those historical mappings. New implementation work should be attached as new child workstreams under the correct feature families during backlog grooming.

---

## Key Decisions

| ID | Decision | Why |
|----|----------|-----|
| D1 | Treat OpenSpec as an **optional planning surface**, not a core SDP dependency | Keeps SDP portable and governance-first |
| D2 | Keep **SDP workstreams + Beads** as execution source of truth | Preserves existing execution discipline and trace model |
| D3 | MVP interop is **import-only** | Avoids two-way sync conflicts before semantics are proven |
| D4 | MVP import target is **planner DAG / planning payload**, not direct Beads mutation | Minimizes risk and preserves human review before execution |
| D5 | OpenSpec-inspired UX ships only after versioned contracts exist | Prevents UI drift and ad hoc formats |
| D6 | New backlog work must attach under **F068, F069, F072** with fresh IDs | Avoids roadmap confusion and stale references |

---

## Adopt / Adapt / Reject

| OpenSpec pattern | Decision | SDP scope boundary | Rationale |
|------------------|----------|--------------------|-----------|
| Ready/blocked/next-action state | Adopt | CLI status and orchestrator guidance only | Strong fit for reducing "what now?" friction |
| Agent-readable instructions payload | Adopt | Structured CLI contract, not hidden prompt magic | Good automation surface for agents and adapters |
| Action-oriented wording (`propose`, `apply`, `verify`, `archive`) | Adapt | Use SDP-native command tree and vocabulary | Improves UX without copying product language blindly |
| Project context + per-artifact rules injection | Adapt | Feed SDP context packets and workstream rules | Useful for consistency across runs |
| Delta specs (`ADDED`, `MODIFIED`, `REMOVED`) | Adapt | Import/export format for planning artifacts only | Valuable for brownfield diffs, but not SDP execution truth |
| Artifact graph as primary source of execution truth | Reject | SDP keeps Beads/workstreams/evidence as primary | OpenSpec is optimized for planning, not governed execution |
| Lightweight verify-before-archive as main gate | Reject | SDP keeps explicit review, policy, and evidence gates | Trust layer must stay stronger than planning layer |

---

## Target Architecture

```text
OpenSpec change folder (optional input)
        |
        v
  OpenSpec import adapter
        |
        v
  SDP planning payload / planner DAG
        |
        +--> next-step contract + instructions contract
        |
        v
  SDP workstreams + Beads + evidence + policy
        |
        v
     PR with proof
```

### Boundary rule

OpenSpec can help SDP decide **what should happen next**.

SDP remains responsible for proving **what actually happened**.

---

## Contract Layer (F068)

### Objective

Create versioned machine-readable contracts that support OpenSpec-style guidance without coupling SDP to OpenSpec internals.

### Proposed contract surfaces

1. `status` contract
   - ready items
   - blocked items
   - next recommended action
   - explanation fields for why something is blocked

2. `instructions` contract
   - action identifier
   - required context
   - optional context
   - suggested command or execution surface
   - policy/evidence expectations

3. `openspec import` contract
   - normalized representation of proposal/spec/design/tasks input
   - source metadata
   - mapping confidence
   - unresolved ambiguity list

### Transport decision

These are CLI-first contracts, not service APIs. The primary surfaces should be machine-readable command output, for example:

- `sdp status --format json`
- `sdp instructions --format json`
- future: `sdp import openspec --format json`

### Versioning decision

Define explicit schema versions in `schema/contracts/` before UX work begins. This keeps F069 and F072 from inventing one-off payloads.

---

## UX Layer (F069)

### Objective

Make SDP feel more obvious on first use by adding OpenSpec-style next-step guidance on top of SDP's existing control model.

### Scope

- improve help/status outputs
- show ready/blocked/next in one place
- explain why an action is recommended
- keep evidence/policy expectations visible in guidance

### Non-goals

- no OpenSpec command aliases as the main SDP UX
- no hidden state machine that bypasses workstream discipline
- no requirement that users install Node or OpenSpec to use SDP

### UX rules

- guidance should be deterministic
- every recommendation should be traceable to known state
- failures should return a structured next step, not just an error

---

## Interop Layer (F072)

### Objective

Allow teams that already use OpenSpec to feed planning artifacts into SDP without rewriting them by hand.

### MVP direction

Import only. No export and no bidirectional sync in phase 1.

### MVP target

Import into a normalized SDP planning payload or planner DAG.

Why not direct workstream or Beads creation on day one:

- imported plans can be incomplete or ambiguous
- workstreams carry stronger execution commitments
- direct issue mutation increases failure blast radius

### Phase-2 extension

After import quality is proven, generate draft workstream proposals from the normalized payload. Human review still approves them before execution.

### Conflict and idempotency policy

- same OpenSpec change imported twice must produce the same normalized payload ID
- import must record source path and source hash
- imports never overwrite existing executed SDP evidence
- ambiguous mappings must surface as unresolved items, not silent guesses

---

## Delivery Plan

### Phase 0 - Roadmap reconciliation

Goal: stop doc drift before implementation starts.

Actions:

- mark the old OpenSpec workstream mapping as historical in planning docs
- attach new implementation work under the right feature families
- avoid reusing repurposed IDs from the old competitive analysis

Exit criteria:

- this design becomes the canonical planning reference
- backlog grooming uses fresh child workstreams under F068/F069/F072

### Phase 1 - F068 contracts first

Goal: freeze the machine-readable surfaces.

Actions:

- define schemas for `status`, `instructions`, and `openspec import`
- define versioning and compatibility policy
- add compatibility fixtures to contract tests

Exit criteria:

- contracts are versioned
- fixtures exist for golden-path payloads
- F069/F072 can consume contracts without guessing field shapes

### Phase 2 - F069 guided next-step UX

Goal: improve first-run ergonomics using the new contracts.

Actions:

- add ready/blocked/next output to status/help flows
- add structured failure guidance
- update demo/bootstrap flow to show guided next actions

Exit criteria:

- a new user can see the next action without reading multiple docs
- guidance output is available in JSON and text modes

### Phase 3 - F072 OpenSpec import MVP

Goal: import OpenSpec planning artifacts into SDP planning payloads.

Actions:

- parse proposal/spec/design/tasks into normalized import payload
- map to planner DAG
- surface unresolved mappings explicitly

Exit criteria:

- import is deterministic
- import never mutates Beads or execution state directly
- imported payload is reviewable before execution

### Phase 4 - Optional ecosystem packaging

Goal: treat OpenSpec as a documented ecosystem entry point once the first three phases are stable.

Actions:

- add OpenSpec to ecosystem/integration docs
- add demo materials only after real interop exists

Exit criteria:

- docs describe a real supported flow, not an aspirational one

---

## Feature Placement

| Feature family | OpenSpec-related scope | Why |
|----------------|------------------------|-----|
| F068 Unified Integration Contracts | `status`, `instructions`, import payload schemas | This is contract work first |
| F069 OSS Combine Bootstrap | next-step UX, guided status/help/demo story | This is onboarding and first-run ergonomics |
| F072 Advanced Agent Architecture | import adapter and reconciliation into planning graph | This is planning-system interop |

Recommendation: do not create a separate OpenSpec feature yet. First prove the contract, UX, and import slices inside existing feature families.

Canonical child workstreams created from this plan:

- `00-068-04` and `00-068-05` for contract surfaces
- `00-069-04` and `00-069-05` for guided UX and failure walkthroughs
- `00-072-05` and `00-072-06` for import and draft workstream generation

---

## Guardrails

- Core SDP flows must work when OpenSpec is absent.
- Beads remains the source of truth for ready/blocked execution work.
- Evidence requirements do not become optional just because planning data came from OpenSpec.
- Import may suggest work; it may not silently start work.
- JSON contracts must be stable enough for agent automation and tests.

---

## Success Metrics

| Metric | Definition | Source |
|--------|------------|--------|
| First-run friction | Median time from `sdp` entry command to first successful guided next action | CLI telemetry or structured local logs |
| Guidance coverage | Percentage of blocked/error states that return structured next-step JSON | contract tests + CLI integration tests |
| Import determinism | Same OpenSpec input produces same normalized import payload ID | import fixture tests |
| Governance preservation | Imported flows still emit required evidence and pass policy checks | evidence tests + policy gate tests |
| Adoption fit | Number of demo or real workflows that use import without manual rewrite | examples and dogfood runs |

---

## What This Plan Deliberately Does Not Do

- It does not replace SDP workstreams with OpenSpec changes.
- It does not define a bidirectional sync engine.
- It does not change SDP's enforcement-first product thesis.
- It does not promise ecosystem docs before real interop exists.

---

## Follow-up Work

1. Implement `00-068-04`, `00-068-05`, `00-069-04`, `00-069-05`, `00-072-05`, and `00-072-06`.
2. Future update to `docs/integrations/ECOSYSTEM_SYNERGIES.md` once implementation exists.

---

## References

- `docs/reference/2026-02-14-competitive-analysis-sdp-vs-oss.md`
- `docs/integrations/ECOSYSTEM_SYNERGIES.md`
- `docs/roadmap/ROADMAP.md`
- `docs/workstreams/INDEX.md`
- `docs/workstreams/backlog/00-068-03.md`
- `docs/workstreams/backlog/00-069-01.md`
- `docs/workstreams/backlog/00-069-03.md`
- `docs/workstreams/backlog/00-072-02.md`
- `https://github.com/Fission-AI/OpenSpec/blob/main/docs/opsx.md`
- `https://github.com/Fission-AI/OpenSpec/blob/main/docs/workflows.md`
- `https://github.com/Fission-AI/OpenSpec/blob/main/docs/concepts.md`
