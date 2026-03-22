# Attention Surface — Next Step

Status: working next step
Date: 2026-03-22
Scope: operator-facing attention/queue view over the existing control tower state

## Goal

Build the next thin practical layer on top of the control store so the orchestrator/human/admin can quickly see:
- what needs human input now
- what is ready to execute now
- what is blocked now
- what is actively executing now
- what the next recommended action is

## Current baseline

Already implemented:
- FeatureCard write model
- project/portfolio snapshots
- card lifecycle actions
- Beads bridge for ready cards
- feedback/export/import flow
- reply ingestion / resume path
- message-correlation bridge

What is still weak is the operator-facing surface for immediate attention.

## Next implementation target

Implement a thin attention surface in `sdp_lab`:

1. add a dedicated CLI status/attention view over existing snapshots
2. emphasize queues:
   - waiting_on_human
   - ready_to_execute
   - blocked
   - executing
3. include active agents and next recommended action in the output
4. keep this read-only and derived from current state
5. add tests
6. update docs if command surface changes

## Constraints

- do not redesign architecture
- do not add UI/dashboard framework yet
- do not make the board/source views mutable
- keep provider-agnostic and file-backed approach
- prefer thin practical implementation over abstraction

## Desired outcome

One command should give a useful control-tower attention view that answers:
- what needs attention right now?
- who needs to respond?
- what can execute immediately?
- what is blocked and why?
- what should the orchestrator do next?
