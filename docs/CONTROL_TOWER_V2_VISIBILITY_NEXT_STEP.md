# Control Tower V2 Visibility — Next Step

Status: working next step
Date: 2026-03-23
Scope: first board/card visibility slice on top of observable V2 fields

## Goal

Take the observable-card fields that now exist in the model and make them legible in user-facing control-tower surfaces.

The immediate target is not more hidden metadata.
It is making the board and card views answer:
- what did the orchestrator last do?
- why is this card in its current situation?
- what is the next action?
- where is friction already accumulating?
- what execution/result trace is already visible?

## Do now

1. Add a human-readable card detail surface in the main `sdp` CLI.

Suggested shape:
- `sdp card show --project <id> --id <card-id>`
- optional `--json`

Card detail should show at least:
- title / status / project
- raw request
- source refs (if present)
- normalized intent / scope / risk
- last orchestrator action / reason / at
- recommended next action / reason
- waiting/blocking state
- linked Beads / dispatch metadata
- latest executor result summary
- friction counters

2. Upgrade board renderers to expose more V2 visibility.

For project / portfolio views, show more than title + next step:
- last orchestrator action
- recommended next action
- friction markers (clarify/block/exec/review counters when non-zero)
- execution/result hints when present

3. Keep the output compact and action-oriented.
- one-screen useful where possible
- more detail only in card view
- no raw YAML dumps

4. Keep command surface thin and stable.
- additive is good (`card show`)
- avoid broad CLI redesign

## Constraints

- thin slice only
- no web UI yet
- no giant rendering framework
- no event-history viewer yet
- use the observable data that already exists

## Desired outcome

After this slice, a colleague should be able to:
- scan the board and see not just state, but orchestrator intent and friction hints
- open one card and understand the current control situation without reading raw YAML
