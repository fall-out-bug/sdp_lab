# Execution Heartbeat & Runtime Reconciliation — Next Step

Status: urgent next step
Date: 2026-03-23
Scope: first thin heartbeat/reconciliation slice for executing cards

## Goal

Make execution state honest.

## Do now

1. Add runtime heartbeat fields to FeatureCard + schemas.
2. Mark dispatch-created executions as `executor_runtime_state=pending`.
3. Add a thin method/CLI path to record executor heartbeat updates.
   Suggested shape:
   - `sdp card heartbeat --project <id> --id <card-id> --session <session-id> --state running --progress "..."`
4. Mark runtime `completed` when result is ingested.
5. Add doctor checks for missing/stale runtime heartbeat.
6. Surface runtime status in board/card/executive views.

## Constraints

- thin slice only
- no full job scheduler
- no full OmO launch engine yet
- no fake heartbeat generation beyond explicit manual/interim updates

## Desired outcome

After this slice, `executing` cards should no longer be ambiguous.
The operator should be able to see whether execution is pending, truly running, stale, lost, or completed.
