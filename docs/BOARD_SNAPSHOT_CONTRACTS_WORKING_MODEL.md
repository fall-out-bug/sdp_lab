# Board Snapshot Contracts — Working Model

Status: working model
Date: 2026-03-22
Related:
- `schema/contracts/feature-card.schema.json`
- `schema/contracts/project-board-snapshot.schema.json`
- `schema/contracts/portfolio-board-snapshot.schema.json`
- `docs/PROJECT_CONTROL_PANEL_WORKING_MODEL.md`
- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`

## Purpose

Define the machine-readable board snapshot layer for the project control panel.

If `FeatureCard` is the core intake/shaping entity, board snapshots are the read models that let a UI or CLI show:
- one project board
- the whole portfolio
- waiting-on-human queues
- ready-to-execute queues
- blocked work

---

## 1. Contracts introduced

### `feature-card.schema.json`
Canonical contract for the board-level feature object.

### `project-board-snapshot.schema.json`
Read model for one project board with columns and execution summary.

### `portfolio-board-snapshot.schema.json`
Read model for the whole portfolio overview across projects.

---

## 2. Intent of each contract

### FeatureCard
Represents one feature/request as it matures from intake to execution.

### ProjectBoardSnapshot
Represents one project's current board state:
- cards by column
- per-column counts
- execution summary
- next recommended action

### PortfolioBoardSnapshot
Represents the whole portfolio:
- project summaries
- total counts
- cross-project queues
- top-level recommended next action

---

## 3. Why separate write model from read models

This separation is intentional.

### Write model
`FeatureCard`

### Read models
- `ProjectBoardSnapshot`
- `PortfolioBoardSnapshot`

Why:
- cards change individually
- boards are derived views
- portfolio view is an aggregation layer
- this keeps storage flexible later

---

## 4. Board-level queues that matter most

At portfolio level, the most valuable queues are:

### `waiting_on_human`
What needs an answer or decision from the human.

### `ready_to_execute`
What is sufficiently shaped and can move into execution.

### `blocked`
What cannot currently progress and why.

These are explicitly included in the portfolio snapshot contract.

---

## 5. Recommended implementation order

1. persist `FeatureCard`
2. derive `ProjectBoardSnapshot`
3. derive `PortfolioBoardSnapshot`
4. render UI/CLI on top

This keeps the system honest: the board is a projection, not the source of truth itself.

---

## 6. Near-term use

These contracts are enough to start building:
- a thin project board UI
- a portfolio home view
- a CLI status surface
- orchestrator recommendations based on current queues

They do not yet define storage. That is the next decision.

---

## 7. Short formula

- `FeatureCard` = write model for intake/shaping
- `ProjectBoardSnapshot` = one project board view
- `PortfolioBoardSnapshot` = portfolio control tower view

That is the control-panel contract stack.
