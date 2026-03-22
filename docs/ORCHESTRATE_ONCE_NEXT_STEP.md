# Orchestrate Once / Dispatch Next — Next Step

Status: working next step
Date: 2026-03-22
Scope: first narrow orchestrator cycle command on top of the current control-tower skeleton

## Goal

Move from separate lifecycle/dispatch/result commands to a first thin orchestrator cycle step.

This is not a full daemon or scheduler.
It is the smallest practical slice that lets SDP do one meaningful orchestration pass.

## Current baseline

Already implemented:
- control store
- lifecycle actions
- Beads bridge
- feedback/resume/correlation
- routing matrix
- execution packet skeleton
- dispatch skeleton
- result ingestion skeleton
- attention surface
- doctor control
- canonical `sdp` root command

## Next implementation target

Implement a narrow orchestrator cycle command.

### Minimum useful behavior
1. add one command such as `sdp dispatch next` or `sdp orchestrate once`
2. select one dispatchable card/task from current state
3. run the existing dispatch path
4. return a concise summary of what happened
5. if nothing is dispatchable, return a clear no-op result
6. update snapshots
7. add tests

## Constraints

- do not build a full runtime daemon
- do not add scheduling complexity beyond one-step selection
- do not add UI
- do not redesign control-store architecture
- reuse current routing, packet, dispatch, and snapshot logic
- keep it thin and practical

## Desired outcome

A single command exists that can perform one orchestrator step and answer:
- what did the system choose to dispatch?
- to which executor role?
- what packet/path was produced?
- if nothing happened, why not?
