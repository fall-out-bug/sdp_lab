# Execution Bridge Stage Pack

Status: v1
Date: 2026-03-23
Scope: `ready` → `executing` → result ingestion

## Purpose

The execution bridge stage connects the human-facing control object (`FeatureCard`) to Beads-backed execution.

This stage should:
- bridge a `ready` card into execution
- preserve trace via `linked_beads_ids`
- keep the card as the board-level control object while execution runs
- ingest executor outcomes back into the control-store lifecycle

This stage should **not** replace Beads or turn the board into the execution system.

---

## Entry / exit

### Entry
A card is in `ready` and passes the ready gate.

### Exit
Two main outcomes:
- the card moves to `executing` with `linked_beads_ids` populated
- later, an executor result is ingested and the card advances to `done`, `blocked`, `reviewing`, or `needs_input`

When the card needs human/admin help, continue via `../feedback-loop/PACK.md`.

---

## Canonical commands

### Bridge a ready card into execution
```bash
sdp card execute --project <project-id> --id <card-id>
```

Current behavior documented in the control-store skeleton:
- creates a feature-level Beads issue
- persists returned Beads IDs on the card
- moves the card to `executing`
- rebuilds project and portfolio snapshots

### Show current board state
```bash
sdp board show --project <project-id>
```

### Run one dispatch step
```bash
sdp dispatch next
```

### Run one orchestration step
```bash
sdp orchestrate once
```

### Ingest executor result packet
```bash
sdp result ingest --input <result-packet.json>
```

### Hygiene check
```bash
sdp doctor control
```

---

## Bridge rules

### 1. Bridge, do not replace
Execution happens in Beads / executor land.
The control tower keeps the lifecycle summary, linkage, and operator-facing state.

### 2. Keep linkage explicit
`linked_beads_ids` is not a nice-to-have.
It is the trace connection between the card and execution objects.

### 3. The card remains the control object during execution
Even after dispatch, the card is still what the board and operator attention model reason about.

### 4. Result ingestion is part of the stage
Dispatch without ingest is an incomplete loop.
The bridge is only useful if execution outcomes return to the card lifecycle.

---

## Result-ingest guidance

Executor result packets should update the card rather than bypass it.
Per the current skeleton, result ingestion can move the card based on outcome, including:
- `success` → `done`
- `blocked` → `blocked`
- `needs_review` → `reviewing`
- `needs_input` → `needs_input`
- `failed` → `blocked`

That means this pack covers both:
- outbound bridge into execution
- inbound bridge back from execution

---

## Common patterns

### Direct bridge of a ready card
```bash
sdp card execute --project myapp --id feature-myapp-20260323-001
sdp board show --project myapp
```

### Operator-driven dispatch step
```bash
sdp dispatch next
```

Use when you want one explicit dispatch action from the current portfolio state.

### Orchestrator pass
```bash
sdp orchestrate once
```

Use when you want one meaningful control action from the current state, which may include dispatch or result handling.

### Ingest a returned result
```bash
sdp result ingest --input /path/to/result.json
```

---

## Anti-patterns

### Bypassing the card
Bad:
- creating execution work directly with no control-store card
- treating Beads as the only state that matters

Good:
- card first
- bridge second
- ingest outcomes back into the card lifecycle

### Losing correlation
Bad:
- execution objects with no durable linkage back to the card

Good:
- persist `linked_beads_ids`
- ingest results using the control-store contract

### Treating the board as an agent workbench
Bad:
- using the board as the native execution interface

Good:
- keep the board as visualization / operator surface
- let execution systems do execution work

---

## Related docs

- `docs/CONTROL_STORE_SKELETON.md`
- `docs/BEADS_GASTOWN_SDP_CONTROL_TOWER_INTEGRATION_PLAN.md`
- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `../feedback-loop/PACK.md`
- `../README.md`
