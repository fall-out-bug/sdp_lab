# Session Start Canon

Status: canonical quick-start
Date: 2026-03-22

## Purpose

Минимальный канон для любой новой реализационной сессии в `sdp_lab`.

Если новая сессия не знает, что такое OmO, SDP, Beads, Gastown, FeatureCard, control tower и как всё это связано — она должна прочитать этот файл и указанные рядом документы.

---

## Read first

1. `/home/fall_out_bug/.openclaw/workspace/CONTROL_TOWER_CANON.md`
2. `docs/CONTROL_TOWER_IMPLEMENTATION_ROADMAP.md`
3. `docs/CONTROL_STORE_SKELETON.md`

## Standard launch brief rule

When spawning a new implementation session for control-tower work, the brief should explicitly tell the agent to read the three files above first.
Use `docs/templates/control-tower-launch-brief.template.md` as the canonical launch-brief source when you want a reusable brief structure.

Minimal pattern:
- read `CONTROL_TOWER_CANON.md`
- read `docs/CONTROL_TOWER_IMPLEMENTATION_ROADMAP.md`
- read `docs/CONTROL_STORE_SKELETON.md`
- if a written launch brief is needed, structure it from `docs/templates/control-tower-launch-brief.template.md`
- only then inspect local code and implement the narrow slice

### Template discipline rule

For the current thin Phase 3 slice:
- `docs/templates/` = canonical source templates
- generated run output does **not** belong in `docs/templates/`
- CLI/packet/board renderings are delivery surfaces, not canonical templates
- packs must reference real template files only

Reference: `docs/TEMPLATE_INVENTORY_AND_CLASSIFICATION.md`

## Read as needed for control tower work

- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/PROJECT_CONTROL_PANEL_WORKING_MODEL.md`
- `docs/ORCHESTRATOR_ACTIONS_AND_FEEDBACK_CONTRACT.md`
- `docs/BEADS_GASTOWN_SDP_CONTROL_TOWER_INTEGRATION_PLAN.md`
- `packs/README.md`

### Stage-pack rule

When the work is clearly about a specific control-tower stage, read `packs/README.md` and then the relevant stage pack before implementing:

- intake work → `packs/intake/PACK.md`
- shaping / ready-gate work → `packs/shaping/PACK.md`
- execution bridge / dispatch / result-ingest work → `packs/execution-bridge/PACK.md`
- feedback / resume / external reply loops → `packs/feedback-loop/PACK.md`

The stage pack does not replace the canon or contracts.
It packages the practical process guidance for that stage so implementation sessions do not have to reconstruct it from scattered docs.
Use packs as the thin stage-oriented guide layered on top of the canon, not as a separate framework.

---

## Canonical understanding

### OmO
Universal agent layer for planning/coding/review/execution.
Not the process layer.

### SDP
Trace/process/artifact layer.
Trace starts at intake.

### Beads
Execution/dependency graph.
Bridge into it after feature shaping reaches the right maturity.

### Board
Human/admin visualization surface.
Not the native execution interface for agents.

### Orchestrator
Primary state mover.
Humans/admins enter mainly through feedback, decision, and approval loops.

---

## Do not do these

- Do not redesign the architecture casually.
- Do not replace Beads.
- Do not collapse OmO into SDP.
- Do not rebuild a second universal agent system inside a repo.
- Do not assume all prior chat context is magically available.

---

## Short implementation rule

Prefer thin practical slices that extend the current control-store architecture over broad framework-building.
