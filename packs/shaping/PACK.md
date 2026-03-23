# Shaping Stage Pack

Status: v1
Date: 2026-03-23
Scope: `inbox` / `clarifying` / `needs_input` → `ready`

## Purpose

Shaping turns a raw request into a card that is ready to bridge into execution.

This stage should:
- clarify intent without over-designing the solution
- fill the practical shaping fields that the control tower needs
- decide when human/admin input is required
- enforce the ready gate before execution

This stage should **not** try to replace execution planning with a giant spec.

---

## Entry / exit

### Entry
The card exists, usually in `inbox`, but is not yet safely executable.

### Exit
The card is either:
- `ready` with a credible shaping payload, or
- explicitly `needs_input`, or
- `parked` if it should not move now

Next stage after `ready`: `../execution-bridge/PACK.md`
Cross-cutting loop when blocked on humans: `../feedback-loop/PACK.md`

---

## Canonical commands

### Add or refine shaping fields
```bash
sdp card clarify \
  --project <project-id> \
  --id <card-id> \
  --intent "<normalized intent>" \
  --task-type <type> \
  --target-repo <repo-or-path> \
  --risk <low|medium|high|unknown> \
  --next "<recommended next step>" \
  --scope-in "item 1;item 2" \
  --scope-out "item 1;item 2"
```

### Mark the card as needing input
```bash
sdp card needs-input \
  --project <project-id> \
  --id <card-id> \
  --needs "author;admin" \
  --feedback "question 1;question 2" \
  --decision "decision needed" \
  --update "author update needed" \
  --admin-action "approval needed"
```

All fields are optional except `--project` and `--id`; use only the ones that fit the actual gap.

### Mark ready after the ready gate passes
```bash
sdp card ready --project <project-id> --id <card-id>
```

### Park a card
```bash
sdp card park --project <project-id> --id <card-id> --reason "<reason>"
```

### Hygiene check
```bash
sdp doctor control
```

---

## Ready-gate guidance

The ready gate should answer a practical question:

**Is this card shaped enough to bridge into execution without dumping ambiguity downstream?**

Use the shaping fields defined in `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`, especially:
- `normalized_intent`
- `task_type`
- `target_repo`
- `scope_in`
- `scope_out`
- `risk_level`
- `recommended_next_step`

Not every card needs exhaustive detail.
But a `ready` card should have enough shape that execution can start without reconstructing the assignment from scratch.

---

## Stage rules

### 1. Clarify the work, not the entire universe
Shaping is about enough fidelity for dispatch, not perfect product specification.

### 2. Ask for input when the card is genuinely blocked on human knowledge or authority
Use `needs_input` when missing context, decisions, approvals, or operator routing matter.
Do not smuggle silent uncertainty into `ready`.

### 3. `ready` is a real gate
Do not use `ready` to mean “probably fine.”
It means the card is shaped enough to bridge cleanly.

### 4. Parking is valid
Some cards should be parked instead of endlessly clarified.
That is process hygiene, not failure.

---

## Common patterns

### Minimal shaping pass
```bash
sdp card clarify \
  --project myapp \
  --id feature-myapp-20260323-001 \
  --intent "Add a user-selectable dark theme in settings" \
  --task-type feature \
  --target-repo /path/to/myapp \
  --risk medium \
  --next "Bridge to execution once UI scope is confirmed" \
  --scope-in "settings toggle;theme persistence" \
  --scope-out "full visual redesign"
```

### Escalate for missing product or operator input
```bash
sdp card needs-input \
  --project myapp \
  --id feature-myapp-20260323-001 \
  --needs "author;admin" \
  --feedback "Which user segment is in scope?" \
  --decision "Should this land behind a feature flag?"
```

### Mark ready only after shaping is done
```bash
sdp card ready --project myapp --id feature-myapp-20260323-001
```

---

## Anti-patterns

### Fake-ready cards
Bad:
- using `ready` as a parking place for ambiguity
- dispatching work that still depends on unstated assumptions

Good:
- either shape it properly
- or move it to `needs_input`
- or park it

### Endless clarifying loops
Bad:
- keeping a card in clarifying forever because “more thinking is always better”

Good:
- make the next state explicit: `ready`, `needs_input`, or `parked`

### Turning shaping into architecture theater
Bad:
- broad redesign before a narrow work slice even starts

Good:
- define enough scope and constraints to execute the next thin slice

---

## Related docs

- `docs/FEATURE_CARD_CONTRACT_WORKING_MODEL.md`
- `docs/ORCHESTRATOR_ACTIONS_AND_FEEDBACK_CONTRACT.md`
- `docs/CONTROL_STORE_SKELETON.md`
- `../feedback-loop/PACK.md`
- `../README.md`

## Useful templates

- `docs/templates/task-brief.template.md` for shaping the ask into a stable execution-facing brief
- `docs/templates/implementation-plan.template.md` when the card needs a thin explicit execution plan before dispatch
- `docs/templates/control-tower-launch-brief.template.md` for a clean implementation-session kickoff once the card is mature enough

Use these as canonical source templates only when they genuinely help.
Do not pad shaping with paperwork just because a template exists.
