# Orchestrate Once Continuation — Next Step

Status: working next step
Date: 2026-03-22
Scope: first thin continuation of the orchestrator cycle beyond dispatch-next

## Goal

Move from a one-step dispatch command to a first useful `orchestrate once` command that can make one meaningful control decision per pass.

This is still not a daemon or full scheduler.
It is a narrow next slice.

## Current baseline

Already implemented:
- `sdp dispatch next`
- dispatch skeleton
- result-ingest skeleton
- attention surface
- doctor control
- feedback/resume/correlation
- Beads bridge

## Next implementation target

Implement a first `orchestrate once` style command.

### Minimum useful behavior
On each invocation, do one of these in order of opportunity:
1. if there is an ingestable executor result waiting in a narrow useful path, ingest it and report what happened
2. else if there is dispatchable work, dispatch one card/task and report what happened
3. else return a clear no-op / nothing-to-do summary

### Additional requirements
- reuse existing dispatch and result-ingest logic
- keep the decision simple and deterministic
- update snapshots
- add tests
- update docs if command surface changes

## Constraints

- do not build a daemon
- do not add scheduler complexity
- do not redesign architecture
- keep implementation thin and practical
- reuse current control-store state and existing commands/helpers

## Desired outcome

A single `sdp` command exists that can perform one orchestrator pass and answer:
- did the system ingest a result?
- else did it dispatch new work?
- else why was there nothing to do?
