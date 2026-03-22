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

## Read as needed for control tower work

- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/PROJECT_CONTROL_PANEL_WORKING_MODEL.md`
- `docs/ORCHESTRATOR_ACTIONS_AND_FEEDBACK_CONTRACT.md`
- `docs/BEADS_GASTOWN_SDP_CONTROL_TOWER_INTEGRATION_PLAN.md`

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
