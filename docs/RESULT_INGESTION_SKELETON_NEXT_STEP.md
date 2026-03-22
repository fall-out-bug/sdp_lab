# Result Ingestion Skeleton — Next Step

Status: working next step
Date: 2026-03-22
Scope: first narrow executor-result ingestion flow on top of dispatch skeleton

## Goal

Move from "can dispatch work and emit an execution packet" to "can ingest executor results back into control state".

This is not the full orchestrator runtime loop.
It is the smallest practical slice that lets the control tower:
- accept a result packet from an executor
- correlate it to dispatched work
- update card/control state
- reflect the new state in snapshots

## Current baseline

Already implemented:
- routing matrix
- execution packet skeleton
- dispatch skeleton
- dispatch metadata persisted in cards
- Beads bridge
- feedback/resume flow

## Next implementation target

Implement a narrow result-ingestion skeleton:

1. define a minimal executor result packet struct/contract in code
2. add one CLI command to ingest a result packet for a dispatched card
3. route the result back to the correct card/state via dispatch metadata/card id
4. update card state for at least these result statuses:
   - success
   - blocked
   - needs_review
   - needs_input
5. persist useful result metadata/artifact refs on the card
6. update snapshots
7. add tests

## Constraints

- do not build the full runtime loop yet
- do not add scheduling complexity
- do not add UI
- do not redesign the architecture
- reuse current control-store state and dispatch metadata
- keep implementation thin and practical

## Desired outcome

A command exists that can ingest an executor result packet and leave behind state that answers:
- what happened
- whether the work succeeded / blocked / needs review / needs input
- what artifacts/findings came back
- what the next visible state is
