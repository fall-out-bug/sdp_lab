# Dispatch Skeleton — Next Step

Status: working next step
Date: 2026-03-22
Scope: first narrow dispatch flow on top of routing matrix + execution packet skeleton

## Goal

Move from "can build an execution packet" to "can actually perform a thin dispatch step".

This is not the full orchestrator runtime.
It is the smallest practical slice that lets the control tower:
- select a dispatchable card/task
- route it to an executor role
- produce an execution packet
- record dispatch metadata back into state
- reflect the dispatch in snapshots/status

## Current baseline

Already implemented:
- FeatureCard/control store
- lifecycle actions
- Beads bridge
- feedback/resume/correlation
- doctor control
- routing matrix skeleton
- execution packet skeleton

## Next implementation target

Implement a narrow dispatch skeleton:

1. add a CLI dispatch command (for example: `dispatch-card` or `dispatch-next`)
2. select a dispatchable card/task in a narrow useful way
3. compute executor role via existing routing logic
4. produce execution packet via existing packet logic
5. write dispatch metadata back into card/state
6. update snapshots
7. add tests

## Constraints

- do not build the full runtime loop yet
- do not add scheduling complexity
- do not add UI
- do not replace current control-store architecture
- keep it thin, practical, and incremental

## Desired outcome

A command exists that can perform the first real orchestrator dispatch step and leave behind state that answers:
- what was dispatched
- to which executor role
- what execution packet was produced
- what the next visible state is
