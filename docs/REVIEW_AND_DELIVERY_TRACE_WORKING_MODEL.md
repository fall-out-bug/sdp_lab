# Review and Delivery Trace — Working Model

Status: working model
Date: 2026-03-23
Scope: close the visible control-tower chain from review through delivery and rollback/follow-up

## Purpose

Control Tower V2 is still incomplete if trace visibility ends at executor result.

The tower should also make visible:
- review outcomes
- verification/release readiness
- delivery/deploy state
- rollback/follow-up outcomes

This doc defines the first thin working model for that tail of the chain.

---

## Product rule

A colleague should be able to answer:
- did execution pass review?
- what happened at delivery time?
- was it deployed?
- did it roll back?
- was a follow-up/hotfix created?

If those answers live only in random chat logs or CI tabs, the control tower is lying by omission.

---

## Trace chain extension

The visible chain becomes:
- source / intake
- shaping
- execution
- review
- delivery
- rollback / follow-up

Execution result is no longer the end of the story.
It is only the handoff into the next visible risk area.

---

## 1. Review visibility model

### What should be visible
At minimum:
- last review outcome
- review summary
- review findings / concerns
- whether review caused rework or escalation
- review failure count

### Suggested status semantics
Current statuses do not need a major rewrite yet.
Thin first slice can work with:
- `reviewing` as an explicit state when a review step is active
- `needs_input` or `blocked` when review produced human/admin or process blockers
- `ready` or `executing` again if rework loop resumes
- `done` only when the current implementation loop is actually completed for the current scope

### Important rule
A failed review is not just another opaque result packet.
It is a visible event in the card’s story.

---

## 2. Delivery visibility model

### What should be visible
At minimum:
- delivery state
- delivery target/environment (if known)
- delivery summary / event ref
- whether deployment succeeded, failed, or is pending

### Suggested thin fields
A first slice can introduce simple card-level fields like:
- `delivery_state`
- `delivery_target`
- `delivery_summary`
- `delivery_ref`
- `delivered_at`

These should stay thin and optional.
Do not build a full deployment engine.

### Example `delivery_state` values
- `pending`
- `deployed`
- `failed`
- `rolled_back`

---

## 3. Rollback and follow-up visibility

### What should be visible
At minimum:
- rollback happened or not
- rollback summary/ref
- follow-up / hotfix linkage if one exists
- rollback count

### Suggested thin fields
- `rollback_ref`
- `rollback_summary`
- `followup_refs`

Keep these optional and honest.
Do not invent history when there is no real rollback path yet.

---

## 4. UI / control-tower visibility rules

### On board/project views
Show lightweight markers only:
- review status/hint
- delivery state if present
- rollback marker if present
- follow-up marker if present

### On card detail
Show a dedicated section for:
- review trace
- delivery trace
- rollback / follow-up trace

### UX rule
Board should signal that something happened.
Card detail should explain what happened.

---

## 5. Friction implications

Review and delivery are high-friction zones.
The control tower should start surfacing:
- repeated review failures
- delivery failures
- rollback presence
- follow-up creation as a sign of escaped issues

Not as a massive analytics system yet.
Just enough to make pain visible.

---

## 6. Thin implementation principle

Do first:
- add thin fields/contracts
- write them only where real paths exist
- surface them in board/card views
- add small helper commands if needed

Do not do yet:
- release orchestrator
- deployment automation framework
- incident management subsystem
- full event sourcing

---

## 7. Recommended first slice

1. Add thin review/delivery/rollback fields to the card and schemas
2. Update result-ingest logic so review outcomes are more explicit in visible fields
3. Add simple CLI commands to register delivery outcomes if needed
4. Surface review/delivery/rollback info in card detail and board hints
5. Keep everything local and honest

---

## Short formula

Execution trace is not enough.
A trustworthy control tower must also show:
**review outcome + delivery outcome + rollback/follow-up outcome**.
