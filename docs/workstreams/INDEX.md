# Workstream Index

> **Updated:** 2026-02-24
> **Format:** `@build 00-FFF-SS` executes single workstream; `@review F00F` reviews all WS for feature F00F
> **Roadmap:** [ROADMAP.md](../roadmap/ROADMAP.md)

## Features

| Feature | Phase | Description | Workstreams |
|---------|-------|-------------|-------------|
| **F014** | 0 | CI Loop CLI | 00-014-01, 00-014-02 |
| **F015** | 0 | Stop Hook Gate | 00-015-01, 00-015-02 |
| **F016** | 0 | Oneshot Outer Loop | 00-016-01, 00-016-02, 00-016-03, 00-016-04 |
| **F017** | 0 | Skill Eval Suite | 00-017-01, 00-017-02 |
| **F018** | 0 | Dead Code Purge | 00-018-01, 00-018-02, 00-018-03 |
| **F019** | 0 | Skill Compression | 00-019-01, 00-019-02, 00-019-03 |
| **F020** | 0 | Build Scope Fix | 00-020-01 |
| **F021** | 0 | Language-Agnostic Skills | 00-021-01 |
| **F022** | 0 | Context Pre-Hydration | 00-022-01 |
| **F023** | 0 | Scope Enforcement | 00-023-01, 00-023-02 |
| **F024** | 0 | Phase Hooks | 00-024-01 |
| **F025** | 0 | Prompt Consolidation | 00-025-01 |
| **F027** | 0 | CI Deterministic Auto-Fixers | 00-027-01 |
| **F001** | 1 | Evidence Schema | 00-001-01, 00-001-02 |
| **F002** | 1 | Evidence CLI | 00-002-01, 00-002-02, 00-002-03 |
| **F026** | 1 | Prompt Provenance | 00-026-01 |
| **F003** | 2 | Handoff Artifact Schema | 00-003-01, 00-003-02 |
| **F004** | 2 | Sequential Reconciler | 00-004-01, 00-004-02, 00-004-03 |
| **F005** | 2 | Rework Loop | 00-005-01 |
| **F006** | 3 | JetStream Evidence Stream | 00-006-01, 00-006-02 |
| **F007** | 3 | Evidence Assembler | 00-007-01, 00-007-02 |
| **F008** | 4 | Model Policy Wiring | 00-008-01, 00-008-02 |
| **F009** | 4 | Intake Bridge | 00-009-01, 00-009-02 |
| **F010** | 4 | Dead Code Removal | 00-010-01 |
| **F011** | 5 | kubeopencode Upstream PRs | 00-011-01, 00-011-02 |
| **F012** | 5 | awesome-opencode | 00-012-01 |
| **F013** | 6 | 10 Consecutive E2E Runs | 00-013-01, 00-013-02, 00-013-03 |

## Workstream Status

### Phase 0: Agent Loop Reliability

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-014-01 | F014 | CI Loop CLI — Poll + Classify | Done |
| 00-014-02 | F014 | CI Loop CLI — Auto-Fix Engine | Done |
| 00-015-01 | F015 | Stop Hook — Cursor Implementation | Done |
| 00-015-02 | F015 | Stop Hook — Claude Code Implementation | Done |
| 00-016-01 | F016 | Oneshot Outer Loop — State Machine CLI | Done |
| 00-016-02 | F016 | Oneshot Outer Loop — Cursor Integration | Done |
| 00-016-03 | F016 | Oneshot Outer Loop — Claude Code Integration | Done |
| 00-016-04 | F016 | Oneshot Outer Loop — opencode Integration | Done |
| 00-017-01 | F017 | Skill Eval Suite — Framework + Core Evals | Done |
| 00-017-02 | F017 | Skill Eval Suite — CI Integration | Done |
| 00-018-01 | F018 | Delete Dead Skills + Agents | Backlog |
| 00-018-02 | F018 | Fix Python→Go + Phantom CLI + Branch Model | Done |
| 00-018-03 | F018 | Phantom sdp guard context/branch/complete/finding removal | Done |
| 00-019-01 | F019 | Compress Operational Skills | Backlog |
| 00-019-02 | F019 | Compress Planning & Design Skills | Backlog |
| 00-019-03 | F019 | Trim Bloated Agents + Sync Copies | Backlog |
| 00-020-01 | F020 | @build Scope Surgery | Backlog |
| 00-021-01 | F021 | Remove Go-Specific Commands from Universal Skills | Done |
| 00-022-01 | F022 | Context Pre-Hydration — gather context before LLM | Done |
| 00-023-01 | F023 | Scope Diff Checker — boundary validation | Done |
| 00-023-02 | F023 | Wire Scope Enforcement into Orchestrator | Done |
| 00-024-01 | F024 | Phase Hooks — pre/post hooks at phase transitions | Done |
| 00-025-01 | F025 | Prompt Consolidation — DRY prompt builders | Done |
| 00-027-01 | F027 | CI Deterministic Auto-Fixers — goimports/go mod tidy before LLM | Done |

### Phase 1: Evidence Foundation

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-001-01 | F001 | Extract JSON Schema from strict.go + template | Backlog |
| 00-001-02 | F001 | Publish schema to sdp protocol repo | Backlog |
| 00-002-01 | F002 | Refactor pr-gate into sdp-evidence CLI | Backlog |
| 00-002-02 | F002 | Add `inspect` subcommand | Backlog |
| 00-002-03 | F002 | Goreleaser + GitHub Actions releases | Backlog |
| 00-026-01 | F026 | Prompt Provenance — prompt_hash + context_sources in evidence | Done |

### Phase 2: Sequential Pipeline

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-003-01 | F003 | Define analyst/coder/reviewer handoff JSON Schema | Done |
| 00-003-02 | F003 | Validation library for handoff artifacts | Done |
| 00-004-01 | F004 | Rewrite AgentRunReconciler phases to sequential | Backlog |
| 00-004-02 | F004 | Inject handoff paths into Task CRD annotations | Backlog |
| 00-004-03 | F004 | Integration test: analyst output feeds coder prompt | Backlog |
| 00-005-01 | F005 | Reviewer verdict → coder rework loop (max 2) | Backlog |

### Phase 3: Evidence Stream

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-006-01 | F006 | NATS JetStream EVIDENCE stream + subject design | Backlog |
| 00-006-02 | F006 | Evidence fragment publisher library for agent pods | Backlog |
| 00-007-01 | F007 | EvidenceAssembler: subscribe + collect + validate | Backlog |
| 00-007-02 | F007 | Materialize envelope to filesystem + pr-gate integration | Backlog |

### Phase 4: Simplify & Wire

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-008-01 | F008 | Wire model-policy ConfigMap into AgentRunReconciler | Backlog |
| 00-008-02 | F008 | Persistent budget tracking + auto-downgrade | Backlog |
| 00-009-01 | F009 | beads-bridge CronJob: bd ready → AgentRun CRD | Backlog |
| 00-009-02 | F009 | Multi-project routing from project-registry.yaml | Backlog |
| 00-010-01 | F010 | Delete ~5.7K LOC: orchestrator, swarm, worker, intake | Backlog |

### Phase 5: Ecosystem

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-011-01 | F011 | kubeopencode UP-001 retry budget PR | Backlog |
| 00-011-02 | F011 | kubeopencode UP-003 evidence hooks proposal | Backlog |
| 00-012-01 | F012 | awesome-opencode submission + blog post | Backlog |

### Phase 6: E2E Dream

| WS | Feature | Title | Status |
|----|---------|-------|--------|
| 00-013-01 | F013 | E2E test harness: create issues, verify PRs | Backlog |
| 00-013-02 | F013 | Run 10 consecutive, fix failures | Backlog |
| 00-013-03 | F013 | Document: swarm operations runbook | Backlog |

## Workstream ID Format

`PP-FFF-SS` — Project (00), Feature (001–027), Step (01, 02, …)

Example: `00-004-02` = sdp_lab, F004 Sequential Reconciler, step 2 (inject handoff paths)
Example: `00-014-01` = sdp_lab, F014 CI Loop CLI, step 1 (poll + classify)
