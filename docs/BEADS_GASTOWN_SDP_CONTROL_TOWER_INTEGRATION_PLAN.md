# Beads + Gastown + SDP Control Tower Integration Plan

Status: working plan
Date: 2026-03-22
Scope: portfolio/project control panel without rebuilding existing substrates
Related:
- `docs/PROJECT_CONTROL_PANEL_WORKING_MODEL.md`
- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/BOARD_SNAPSHOT_CONTRACTS_WORKING_MODEL.md`
- `schema/contracts/feature-card.schema.json`
- `schema/contracts/project-board-snapshot.schema.json`
- `schema/contracts/portfolio-board-snapshot.schema.json`

## Goal

Build the SDP control tower without inventing a new task tracker or a new dashboard stack from scratch.

The correct approach is:
- reuse **Beads** for execution graph and dependency state
- reuse **Gastown** patterns for dashboard/control surface
- add only the missing **SDP intake + FeatureCard + trace spine** layer

---

## 1. Current reality

We already have three different strengths available.

### A. Beads is strong at execution state
Beads already gives us:
- issues/tasks
- dependencies
- ready queue
- claim/in-progress/close flow
- JSON outputs
- durable git-backed state
- graph structure for findings and follow-up work

### B. Gastown is strong at control-surface UX
Gastown already gives us:
- dashboard mindset
- project/rig/workspace visibility
- queue panels
- convoy/work tracking views
- next-action/operator-centric workflows
- live read-model fetching patterns
- browser dashboard implementation

### C. SDP is strong at process/evidence/trace
SDP already gives us:
- process layer
- artifact taxonomy
- handoff rules
- task envelopes
- evidence concepts
- trace/evidence expectations
- the principle that process trace should start at intake

### Key gap
What is still missing is a strong **pre-execution feature shaping layer** that:
- starts trace at intake
- lets the orchestrator mature raw feature ideas
- only later bridges into Beads execution objects

That is why `FeatureCard` exists.

---

## 2. Design principle

### Do not replace what already works

#### Do not replace Beads with a new task graph
That would be wasted effort and likely worse.

#### Do not ignore Gastown's dashboard patterns
That would mean rebuilding a control surface from zero for no reason.

#### Do not force Beads to be the raw inbox for immature feature ideas
That makes intake too heavy and pollutes execution tracking.

### Therefore
The control tower should be a **thin orchestration/intake layer** over:
- Beads as execution substrate
- Gastown-inspired dashboard/read-model patterns
- SDP as process/trace spine

---

## 3. Target layered architecture

### Layer 1 — Intake / shaping / trace
**Owned by SDP control tower layer**

Entities:
- `FeatureCard`
- intake artifact / task brief
- clarification history
- readiness state

Responsibilities:
- cheap feature intake
- trace from first contact
- orchestrator clarification
- readiness gate
- bridge preparation

### Layer 2 — Execution graph
**Owned by Beads**

Entities:
- feature-level execution issues
- child tasks
- dependencies
- findings loop
- ready/blocked/in_progress queue

Responsibilities:
- durable execution graph
- dependency tracking
- backlog and follow-up work
- execution state transitions

### Layer 3 — Dashboard / control surface
**Inspired by Gastown patterns**

Entities:
- project board snapshot
- portfolio board snapshot
- waiting_on_human queue
- ready_to_execute queue
- blocked queue

Responsibilities:
- show portfolio state
- show project board state
- show next recommended action
- show execution summary from Beads
- allow orchestrator actions

### Layer 4 — Agent execution
**Owned by OmO + project-local layers**

Responsibilities:
- planning
- coding
- review
- repo-local guidance
- execution artifacts and validation inputs

---

## 4. What to reuse as-is

### Reuse from Beads as-is
- issue/task graph
- dependency edges
- ready queue logic
- claim/in-progress/close lifecycle
- JSON status/query outputs
- findings/follow-up issue re-entry patterns

### Reuse from Gastown as-is or almost as-is conceptually
- dashboard as a read-model projection, not source of truth
- operator/Mayor-centric control surface
- panels for queues and work state
- next-action emphasis
- project/workspace overview patterns
- issue/activity/queue fetcher mentality

### Reuse from SDP as-is
- artifact taxonomy
- handoff contract
- task envelope vocabulary
- artifact bundle mapping
- evidence/trace-first philosophy

---

## 5. What to adapt, not rewrite

### Adapt Beads usage
Beads should be adapted to sit **under** FeatureCard, not instead of it.

Meaning:
- raw intake starts as `FeatureCard`
- when mature enough, it spawns/links Beads objects
- Beads stays execution truth, not raw feature inbox

### Adapt Gastown dashboard ideas
Gastown concepts should be adapted from:
- convoy-centric workspace tracking

to:
- project/portfolio feature-board tracking
- control-tower queues
- execution summaries derived from Beads

We should borrow the dashboard/read-model style, not blindly force convoys to become FeatureCards.

---

## 6. What must be built new

### New piece 1: FeatureCard persistence layer
Need a durable store for:
- raw request
- normalized intent
- clarification history
- status
- intake artifact link
- bridge links to Beads and SDP artifacts

### New piece 2: Intake-to-ready orchestrator actions
Need first-class actions such as:
- clarify
- ask_human
- mark_ready
- park
- reopen
- spawn_execution

### New piece 3: Projection layer
Need code that derives:
- `ProjectBoardSnapshot`
- `PortfolioBoardSnapshot`

from:
- FeatureCard state
- Beads execution state
- optional artifact links

---

## 7. Recommended implementation sequence

### Phase 1 — Storage and projections
Build:
- FeatureCard persistence
- project board snapshot derivation
- portfolio snapshot derivation

Without this, the dashboard has nothing clean to render.

### Phase 2 — Intake flow
Build:
- create card
- clarify card
- needs_input handling
- ready gate

### Phase 3 — Beads bridge
Build:
- ready card -> feature-level Beads issue
- link child tasks/workstreams later
- sync execution summary back into board snapshots

### Phase 4 — UI surface
Build or adapt dashboard views for:
- portfolio home
- project board
- card detail drawer
- waiting_on_human queue
- ready_to_execute queue

### Phase 5 — Deeper orchestration
Build:
- orchestrator recommendations
- stale-card detection
- automatic next-action suggestions
- optional agent-assisted clarification

---

## 8. Storage recommendation

### Recommendation: start with file-backed FeatureCards
Prefer a git-friendly file-backed store first.

Why:
- transparent
- easy to inspect
- easy to diff
- easy to bootstrap
- consistent with current document-first architecture work

Possible forms:
- file-per-card YAML
- or JSON/YAML under project-specific control-panel directories

### Not recommended first
- building a new full DB-backed task system
- pushing raw feature intake directly into Beads

---

## 9. Suggested directory strategy

Example only; exact layout can change.

```text
.sdp/control/
  projects/
    openclaw/
      cards/
        feature-openclaw-2026-03-22-001.yaml
      snapshots/
        board.json
    opencode/
      cards/
      snapshots/
  portfolio/
    snapshot.json
```

### Principle
- write model = cards
- read models = snapshots
- execution graph remains in Beads

---

## 10. How Gastown should influence the implementation

### What to copy conceptually
- one-screen dashboard mentality
- queue-oriented operator view
- explicit next-action guidance
- project/workspace read-model aggregation
- live-ish refresh pattern

### What NOT to copy blindly
- convoy as the top-level feature object
- direct equivalence between Gastown issue model and FeatureCard
- rig/crew/polecat concepts where they do not fit the SDP control tower mental model

---

## 11. How Beads should influence the implementation

### What to copy conceptually
- graph-first execution thinking
- durable issue state
- dependency-aware readiness
- team sync branch workflows
- JSON-friendly CLI interface

### What NOT to force
- every raw idea must be a Beads issue immediately
- FeatureCard lifecycle must be flattened into Beads statuses

---

## 12. Product recommendation in one sentence

Build the SDP control tower as a **FeatureCard + trace-first intake layer** that projects into a **Gastown-style dashboard** and bridges into **Beads for execution**.

That is the non-stupid path.

---

## 13. Immediate next actionable step

Do not jump straight into a full dashboard.

First implement:
1. FeatureCard file storage layout
2. ProjectBoardSnapshot derivation
3. PortfolioBoardSnapshot derivation
4. minimal CLI/status view over those snapshots

Only after that decide whether to:
- adapt Gastown dashboard code directly
- or build a thinner SDP-specific dashboard using the same architectural pattern
