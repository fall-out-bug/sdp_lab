# Control Tower V2 — Next Step

Status: working next step
Date: 2026-03-23
Scope: first observable-card / orchestrator-trace slice for Control Tower V2

## Goal

Take the first thin implementation slice required by `docs/CONTROL_TOWER_V2_WORKING_MODEL.md`.

The immediate target is:
- make the card more observable as a control object
- make orchestrator actions/reasons visible
- add the smallest useful trace/friction fields needed for that visibility

## Do now

1. Extend the `FeatureCard` contract/schema/store with a thin set of observable-V2 fields.

Suggested first fields:
- `source_refs` — references to original ticket/chat/source objects
- `last_orchestrator_action`
- `last_orchestrator_reason`
- `last_orchestrator_at`
- `recommended_next_action`
- `recommended_next_reason`
- `clarification_cycles`
- `blocked_cycles`
- `execution_attempt_count`
- `review_fail_count`
- `rollback_count`

2. Update existing lifecycle mutations so the observable fields are actually written when useful:
- clarify
- needs_input
- ready
- execute / dispatch
- result ingest
- blocked transitions

3. Surface the new fields in board/card-oriented read models where it is cheap and useful.

4. Keep friction counters minimal and honest:
- increment only where behavior already exists in the current flow
- do not invent fake delivery history if deploy/rollback integration does not yet exist
- placeholders like `rollback_count` may exist but should remain zero unless a real path writes them

5. Update docs/contracts/tests.

## Constraints

- thin slice only
- no event-sourcing rewrite
- no giant lifecycle-event engine yet
- no fake release/deploy implementation
- do not explode the schema with speculative fields

## Desired outcome

After this slice, SDP should be able to show, at minimum:
- where the work came from
- what the orchestrator last did
- why it did it
- what it recommends next
- basic visible friction counters for shaping/execution/review loops

That is enough to begin making the control tower feel observable rather than magical.
