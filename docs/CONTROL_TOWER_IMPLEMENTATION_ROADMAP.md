# Control Tower Implementation Roadmap

Status: working roadmap
Date: 2026-03-22
Owner: Клавдий (orchestration) + OmO (implementation)

## Goal

Перейти от архитектурных документов и минимального skeleton к рабочему control tower для AI PDLC/SDLC:
- intake starts trace immediately via SDP
- orchestrator matures requests into ready state
- Beads becomes execution graph
- board remains a human/admin visualization and feedback surface

---

## Current baseline

Already done:
- OmO / SDP / orchestrator / project-local boundary model
- SDP artifact taxonomy, templates, mapping
- control panel working model
- FeatureCard working model + schema
- project/portfolio board snapshot schemas
- storage/layout proposal
- orchestrator actions + feedback contract
- file-backed control store skeleton
- card lifecycle actions (`create`, `clarify`, `needs-input`, `ready`, `park`)

This means the next steps should be implementation-heavy, not more architecture-only churn.

---

## Phase 1 — Control Store MVP (finish the core write/read model)

### Goal
Turn the current skeleton into a usable local control-state engine.

### Scope
1. Add card update/load helpers beyond current lifecycle basics
2. Add stronger ready-gate and transition validation
3. Add snapshot rebuild convenience helpers
4. Add richer status rendering for CLI use
5. Add explicit intake artifact writing/updating helpers

### Deliverable
A reliable local control-state layer that can:
- persist FeatureCards
- derive project/portfolio snapshots
- expose human/admin-relevant queues

### Exit criteria
- can create, clarify, request input, ready, and park cards reliably
- snapshots reflect queues correctly
- CLI output is usable for orchestration and human visibility

---

## Phase 2 — Beads Bridge (first real execution integration)

### Goal
Bridge `ready` FeatureCards into Beads execution objects.

### Scope
1. `card-execute` / `card-bridge` action
2. Create feature-level Beads issue from a ready card
3. Write back `linked_beads_ids`
4. Seed labels/description/spec references from card data
5. Optionally attach initial workstream hints or placeholders

### Deliverable
First working path:
`FeatureCard (ready)` -> `Beads feature issue`

### Exit criteria
- ready card can create linked Beads issue
- control store reflects the linkage
- snapshot can show execution linkage

---

## Phase 3 — Orchestrator Loop Integration

### Goal
Make orchestration actions operate on the control store as a first-class system.

### Scope
1. Orchestrator helpers over card lifecycle
2. Recommendation engine for next action
3. Feedback packet generation for author/admin
4. Resume flow after feedback/decision arrival
5. Blocked / waiting-on-human / ready-to-execute prioritization

### Deliverable
The orchestrator can autonomously move cards and surface only meaningful exceptions.

### Exit criteria
- orchestrator can own state transitions by default
- author/admin only get targeted feedback requests or updates
- feedback answers can resume flow automatically

---

## Phase 4 — Human/Admin Surface

### Goal
Expose control state as a useful board/dashboard without making it the system of record.

### Scope
1. CLI-friendly board/status views
2. thin board rendering over snapshots
3. project board view
4. portfolio control tower view
5. waiting_on_human / blocked / ready_to_execute visibility
6. show active agents and human/admin requests clearly

### Deliverable
A usable visualization surface for humans/admins.

### Exit criteria
- portfolio and project views are readable
- can see what is happening, which agent is doing what, and what is needed from humans/admins

---

## Phase 5 — Advanced execution shaping

### Goal
Mature the bridge between planning, execution, and evidence.

### Scope
1. richer Beads decomposition from FeatureCard
2. automatic artifact expectation attachment
3. findings loop integration back into cards
4. release/review gating integration
5. improved cross-project prioritization

### Deliverable
A more complete AI PDLC/SDLC loop with less manual glue.

---

## Immediate implementation recommendation

Do not split effort across all phases at once.

### Recommended next implementation chunk for OmO
Focus now on:
- **Phase 2: Beads Bridge**
- with just enough supporting polish from Phase 1 to make it solid

### Why
Because the architecture is already strong enough, and the biggest missing capability is the first real bridge:

`FeatureCard -> Beads execution graph`

Without that, the control store remains a good planning shell but not yet a true execution control tower.

---

## OmO implementation brief

### Assignment
Implement the next major slice in `sdp_lab`:

1. add a control action for bridging a ready FeatureCard into a Beads feature issue
2. persist the returned Beads ID(s) back into the card
3. ensure board snapshots reflect execution linkage
4. keep the implementation thin and aligned with the current file-backed control-store architecture
5. do not redesign the architecture or replace Beads

### Constraints
- preserve existing contracts and schemas unless a small compatibility fix is clearly needed
- prefer incremental implementation over framework-building
- do not add database-first infrastructure
- keep board as visualization, not source of truth
- keep orchestrator-centric philosophy intact

### Suggested command surface
Possible CLI shape:
- `sdp-control card-execute --project <id> --id <card-id>`

### Expected output of the OmO pass
- code changes
- tests
- updated docs if the CLI surface changes
- concise summary of what was implemented and what remains
