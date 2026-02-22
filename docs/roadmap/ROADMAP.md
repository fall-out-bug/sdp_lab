# sdp_lab Roadmap — Autonomous K8s Agent Swarm

> **Updated:** 2026-02-22
> **Direction:** Evidence layer + autonomous agent pipeline → issue in, PR with proof out
> **Design:** [Dream Swarm Design](../plans/2026-02-22-dream-swarm-design.md)

---

## Overview

Six phases. Each phase is independently valuable. Earlier phases are prerequisites for later ones but each ships something usable.

```mermaid
graph LR
    subgraph p1 ["Phase 1: Evidence Foundation"]
        F001["F001 Schema"]
        F002["F002 CLI"]
    end
    subgraph p2 ["Phase 2: Sequential Pipeline"]
        F003["F003 Handoff Schema"]
        F004["F004 Sequential Reconciler"]
        F005["F005 Rework Loop"]
    end
    subgraph p3 ["Phase 3: Evidence Stream"]
        F006["F006 JetStream Evidence"]
        F007["F007 Assembler"]
    end
    subgraph p4 ["Phase 4: Simplify & Wire"]
        F008["F008 Model Policy"]
        F009["F009 Intake Bridge"]
        F010["F010 Dead Code Removal"]
    end
    subgraph p5 ["Phase 5: Ecosystem"]
        F011["F011 kubeopencode PRs"]
        F012["F012 awesome-opencode"]
    end
    subgraph p6 ["Phase 6: E2E Dream"]
        F013["F013 10 Consecutive Runs"]
    end

    F001 --> F002
    F001 --> F003
    F002 --> F012
    F003 --> F004
    F004 --> F005
    F004 --> F006
    F006 --> F007
    F005 --> F013
    F007 --> F013
    F008 --> F009
    F009 --> F010
    F010 --> F013
    F011 --> F012
```

---

## Phase 1: Evidence Foundation

**Goal:** Evidence envelope formalized as JSON Schema. Standalone CLI released. Anyone can validate agent evidence without K8s.

| Feature | Workstreams | Size | Description |
|---------|-------------|------|-------------|
| **F001: Evidence Schema** | 00-001-01, 00-001-02 | M | Formalize 9-section envelope as JSON Schema from `specs/strict-evidence-template.json` and `internal/evidence/strict.go`. Publish in sdp repo `schema/evidence-envelope.schema.json`. |
| **F002: Evidence CLI** | 00-002-01, 00-002-02, 00-002-03 | L | Extract `cmd/pr-gate` into standalone `cmd/sdp-evidence` with `validate` and `inspect` subcommands. Goreleaser + GitHub Actions for binary releases. Zero K8s dependency. |

**Exit criteria:**
- `sdp-evidence validate` works as standalone binary
- JSON Schema published in sdp protocol repo
- Binary downloadable from GitHub Releases

**Delivers:** Evidence as a product anyone can use. CI/CD integration via single binary.

---

## Phase 2: Sequential Pipeline

**Goal:** AgentRunReconciler runs analyst → coder → reviewer sequentially with structured handoff artifacts. Reviewer can reject and trigger rework.

| Feature | Workstreams | Size | Description |
|---------|-------------|------|-------------|
| **F003: Handoff Artifact Schema** | 00-003-01, 00-003-02 | M | JSON Schema for `.sdp/handoff/<id>/analyst.json`, `coder.json`, `reviewer.json`. Each role writes structured output; next role reads it. Schema validates handoff integrity. |
| **F004: Sequential Reconciler** | 00-004-01, 00-004-02, 00-004-03 | L | Rewrite AgentRunReconciler phases: `""` → analyst only → `AnalystComplete` → coder (with analyst.json injected) → `CoderComplete` → reviewer (with both artifacts). Delete parallel analyst+coder creation. |
| **F005: Rework Loop** | 00-005-01 | S | Reviewer verdict `needs_changes` triggers coder retry with reviewer feedback injected. Max 2 rework iterations before failing the run. Track rework count in AgentRun status. |

**Exit criteria:**
- Analyst output feeds into coder prompt (verified by checking handoff file exists)
- Reviewer has access to both analyst and coder artifacts
- Rework loop demonstrated: reviewer rejects → coder fixes → reviewer approves

**Delivers:** Quality pipeline. The analyst's risk analysis actually affects the coder's behavior. Reviewer feedback isn't thrown away.

---

## Phase 3: Evidence Stream

**Goal:** Evidence fragments collected across pod boundaries via NATS JetStream. Assembled into a single validated envelope by a dedicated component.

| Feature | Workstreams | Size | Description |
|---------|-------------|------|-------------|
| **F006: JetStream Evidence Stream** | 00-006-01, 00-006-02 | M | Create `EVIDENCE` JetStream stream. Define subjects: `sdp.evidence.<issueID>.<section>`. Evidence fragment publisher library that agent pods use via `bus.Publish()`. Each fragment includes provenance hash for chain validation. |
| **F007: Evidence Assembler** | 00-007-01, 00-007-02 | L | New component (in adapter-controller or standalone) subscribing to `sdp.evidence.<issueID>.>`. Collects fragments, feeds into `BusService.Ingest()` for hash chain validation, materializes assembled envelope to `.sdp/evidence/<issueID>.json`. Handles out-of-order arrival and assembler restarts via JetStream replay. |

**Exit criteria:**
- Evidence fragments published from separate pods arrive in JetStream
- Assembler produces a valid 9-section envelope from fragments
- Hash chain validates end-to-end across fragments
- `pr-gate` runs unchanged against assembled file

**Delivers:** Cross-pod evidence collection. The hard part of autonomous agent swarms — collecting proof across independent processes.

---

## Phase 4: Simplify & Wire

**Goal:** Delete ~5.7K LOC of dead orchestration code. Wire model policy. Replace NATS dispatch with CRD-only intake.

| Feature | Workstreams | Size | Description |
|---------|-------------|------|-------------|
| **F008: Model Policy Wiring** | 00-008-01, 00-008-02 | M | Wire existing `model-policy` ConfigMap into AgentRunReconciler. Controller resolves `spec.workstream` → role → model. `status.resolvedModel` for audit. Budget tracking persisted in `budget-status` ConfigMap (replace in-memory `BudgetTracking`). Auto-downgrade at 80% daily budget. |
| **F009: Intake Bridge** | 00-009-01, 00-009-02 | M | `beads-bridge` CronJob: polls `bd ready` per project from `project-registry.yaml`, creates AgentRun CRDs for ready issues. ~50 LOC Go. Replaces swarm-orchestrator + feature-orchestrator + NATS intake path. |
| **F010: Dead Code Removal** | 00-010-01 | L | Delete: `orchestrator/`, `parallel/`, `swarm/`, `roles/`, `agent/`, `cmd/swarm-worker`, `cmd/swarm-orchestrator`, `cmd/feature-orchestrator`, `cmd/autonomy-worker`, `cmd/intake-gateway`. Update `go.mod`, Dockerfiles, CI. Verify remaining code compiles and tests pass. ~5,753 LOC removed. |

**Exit criteria:**
- Model resolved from ConfigMap, visible in `kubectl describe agentrun`
- Budget enforced: run rejected when daily limit exceeded
- `beads-bridge` CronJob creates AgentRun for each ready issue
- All deleted packages gone, `go build ./...` clean, tests pass
- Binary count: 27 → 4 (adapter-controller, sdp-evidence, beads-fsm, beads-bridge)

**Delivers:** Simplicity. 6K LOC that does what 25K LOC used to do, with ecosystem handling the rest.

---

## Phase 5: Ecosystem

**Goal:** Visible in the opencode ecosystem. Contributing upstream.

| Feature | Workstreams | Size | Description |
|---------|-------------|------|-------------|
| **F011: kubeopencode Upstream PRs** | 00-011-01, 00-011-02 | M | Push UP-001 retry budget PR. Write UP-003 evidence hooks proposal. Contribute evidence bridge pattern upstream so any kubeopencode user can project evidence. |
| **F012: awesome-opencode** | 00-012-01 | S | Submit SDP protocol + `sdp-evidence` CLI to awesome-opencode. Write a blog post or README section: "Evidence for Autonomous Agent Swarms." |

**Exit criteria:**
- At least one kubeopencode PR merged or in active review
- Listed in awesome-opencode
- Blog post published

**Delivers:** Community awareness. Users outside our own project trying the evidence CLI.

---

## Phase 6: E2E Dream

**Goal:** 10 consecutive issue → PR with evidence runs. The dream works.

| Feature | Workstreams | Size | Description |
|---------|-------------|------|-------------|
| **F013: 10 Consecutive Runs** | 00-013-01, 00-013-02, 00-013-03 | XL | End-to-end validation. Create 10 beads issues of varying complexity. beads-bridge dispatches them as AgentRun CRDs. Sequential pipeline (analyst→coder→reviewer) runs. Evidence fragments stream via JetStream. Assembler materializes envelopes. PR gate validates. PR published. 10/10 succeed. Fix whatever breaks. |

**Exit criteria:**
- 10 consecutive runs, different issue types
- Each produces a valid evidence envelope with complete hash chain
- Each produces a merged PR
- Budget stayed within limits
- No manual intervention required

**Delivers:** The dream. Issue in, PR with proof out.

---

## Feature Index

| Feature | Phase | Size | Status | Workstreams | Depends On |
|---------|-------|------|--------|-------------|------------|
| F001 Evidence Schema | 1 | M | Backlog | 00-001-01, 00-001-02 | — |
| F002 Evidence CLI | 1 | L | Backlog | 00-002-01, 00-002-02, 00-002-03 | F001 |
| F003 Handoff Schema | 2 | M | Backlog | 00-003-01, 00-003-02 | F001 |
| F004 Sequential Reconciler | 2 | L | Backlog | 00-004-01, 00-004-02, 00-004-03 | F003 |
| F005 Rework Loop | 2 | S | Backlog | 00-005-01 | F004 |
| F006 JetStream Evidence | 3 | M | Backlog | 00-006-01, 00-006-02 | F004 |
| F007 Evidence Assembler | 3 | L | Backlog | 00-007-01, 00-007-02 | F006 |
| F008 Model Policy | 4 | M | Backlog | 00-008-01, 00-008-02 | — |
| F009 Intake Bridge | 4 | M | Backlog | 00-009-01, 00-009-02 | F008 |
| F010 Dead Code Removal | 4 | L | Backlog | 00-010-01 | F009 |
| F011 kubeopencode PRs | 5 | M | Backlog | 00-011-01, 00-011-02 | — |
| F012 awesome-opencode | 5 | S | Backlog | 00-012-01 | F002, F011 |
| F013 10 Consecutive Runs | 6 | XL | Backlog | 00-013-01, 00-013-02, 00-013-03 | F005, F007, F010 |

---

## Dependency Graph (Critical Path)

```
F001 ──→ F002 ──→ F012 (publish evidence CLI → get listed)
  │
  └──→ F003 ──→ F004 ──→ F005 ──→ F013 (pipeline → rework → dream)
                  │
                  └──→ F006 ──→ F007 ──→ F013 (evidence stream → dream)

F008 ──→ F009 ──→ F010 ──→ F013 (model policy → intake → cleanup → dream)

F011 ──→ F012 (upstream PRs → awesome listing)
```

**Critical path to the dream:** F001 → F003 → F004 → F006 → F007 → F013

**Parallelizable work:** F008-F010 (simplify) can run alongside F003-F007 (pipeline + stream). F011-F012 (ecosystem) can run anytime after F002.

---

## Size Guide

| Size | Workstreams | Estimated Effort | Example |
|------|-------------|------------------|---------|
| S | 1 | 1-2 sessions | Rework loop, awesome-opencode submission |
| M | 2 | 2-4 sessions | JSON Schema, model policy, intake bridge |
| L | 2-3 | 4-6 sessions | Evidence CLI, sequential reconciler, assembler, dead code removal |
| XL | 3+ | 6-10 sessions | E2E validation (fixes everything that breaks) |

**Total: 13 features, 26 workstreams, estimated 40-60 sessions.**

---

## References

- [Dream Swarm Design](../plans/2026-02-22-dream-swarm-design.md) — architecture decisions, expert analysis
- [Manifesto](../MANIFESTO.md) — what SDP is and where it fits
- [Workstream Index](../workstreams/INDEX.md) — workstream ID format and current state
