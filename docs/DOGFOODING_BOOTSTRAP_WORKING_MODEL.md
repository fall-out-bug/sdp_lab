# SDP Dogfooding Bootstrap — Working Model

Status: working model
Date: 2026-03-23
Scope: move SDP from self-hosted design work into real operational dogfooding

## Goal

Use SDP on SDP itself.

That means:
- real work gets captured as FeatureCards
- cards are grouped into coherent feature slices / streams
- execution gets bridged into Beads
- orchestrator becomes the active loop over real work
- readiness and progress get reported from the control tower instead of from ad hoc chat memory

---

## Product rule

Until SDP is planning and tracking its own work through the tower, we are still building a demo, not a control system.

---

## Dogfood target state

For the first practical dogfood pass, we want:
- one real project lane: `sdp_dev`
- a small set of real next slices as FeatureCards
- each card clarified enough to be board-visible and dispatchable
- each active card bridged into Beads when ready
- executive / board / card surfaces reflecting actual work
- orchestrator used as the canonical next-step picker

---

## Dogfood loop

1. Capture next real SDP work as FeatureCards
2. Tag/group those cards by feature slice / stream in the card metadata
3. Clarify them enough for ready-gate
4. Bridge ready cards into Beads
5. Use `sdp attention`, `sdp board show`, `sdp card show`, `sdp orchestrate once`, and `sdp dispatch next` as the operating loop
6. Report readiness/progress from the tower, not from scattered chat context

---

## First bootstrap scope

Use current obvious next work as the initial dogfood backlog.

Suggested first slices:
- friction intelligence surface
- operator action-loop refinement
- stream/feature grouping discipline
- readiness reporting / orchestrator reporting loop

Do not try to ingest the whole universe on day one.
Start with a real but small operational backlog.

---

## Stream discipline (thin first slice)

The current model does not need a large workstream subsystem rewrite.
For dogfooding, it is enough to:
- keep cards as the primary unit
- use `linked_workstreams` or equivalent metadata where useful
- group related cards under a named feature slice / stream theme

Example themes:
- `stream:control-tower-ux`
- `stream:orchestrator-ops`
- `stream:friction-intelligence`
- `stream:dogfood-bootstrap`

---

## Beads discipline

Dogfooding should not stop at board-only cards.
If a slice is real execution work, it should get a Beads anchor once it is ready.

Thin rule:
- clarify first
- mark ready
- execute/dispatch
- keep Beads linkage visible on the card

If Beads integration fails in practice, that failure is part of the dogfood signal.

---

## Reporting discipline

The tower should become the default reporting surface.

At minimum, the operator loop should be able to answer:
- what cards are ready?
- what cards are executing?
- what is blocked on human/admin?
- what is the next card to move?
- what finished recently?
- what is not actually ready despite appearing alive?

---

## First bootstrap deliverables

1. Dogfood bootstrap doc (this file)
2. Initial `sdp_dev` FeatureCards for the next real slices
3. Snapshot rebuild after card creation
4. Ready/dispatch path on at least one real card if feasible
5. A short operating rhythm for using the tower daily

---

## Daily operating rhythm (thin)

### Start
- `sdp attention`
- `sdp doctor control`
- `sdp board show --project sdp_dev`

### Move work
- clarify / needs-input / ready / dispatch based on actual queue state
- run `sdp orchestrate once` when useful

### Check active items
- `sdp card show --project sdp_dev --id <card-id>`

### Close/update
- record delivery/review outcomes when they happen

---

## Non-goals

- full migration of all historical tasks into SDP right now
- large workstream taxonomy design
- perfect automation before first dogfood usage

The goal is to start using the system for real work immediately and learn from the friction.

---

## Short formula

Dogfooding starts when SDP work itself is:
**captured as cards, grouped into feature slices, bridged into Beads, and driven by the orchestrator loop**.
