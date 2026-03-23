# Execution Heartbeat & Runtime Reconciliation Spec

Status: urgent working spec
Date: 2026-03-23
Scope: make `executing` mean a live executor runtime rather than a stale optimistic state

## Problem

Current SDP control state can say `executing` after dispatch + Beads linkage even when no live OmO/executor session exists.

That creates a trust gap:
- tower says work is running
- but there may be no actual runtime behind it

This spec closes that gap with a thin heartbeat + reconciliation layer.

---

## Goal

For every `executing` card, SDP should be able to answer:
- is there a real executor session?
- when did it start?
- when was the last heartbeat?
- what is the runtime state?
- is execution healthy, stale, or lost?

---

## Thin-slice model

### Add executor runtime visibility to FeatureCard
Suggested first fields:
- `executor_session_id`
- `executor_started_at`
- `last_executor_heartbeat_at`
- `executor_runtime_state`
- `executor_progress_summary`

### Runtime state values
For first slice:
- `pending`
- `running`
- `stale`
- `lost`
- `completed`

Do not overbuild a job orchestration engine yet.

---

## State rules

### When dispatch happens
If dispatch is only packet creation:
- `executor_runtime_state = pending`
- no heartbeat yet

If/when a real OmO session is launched:
- set `executor_session_id`
- set `executor_started_at`
- set `last_executor_heartbeat_at`
- set `executor_runtime_state = running`

### When heartbeat arrives
- update `last_executor_heartbeat_at`
- optionally update `executor_progress_summary`
- keep `executor_runtime_state = running`

### When result is ingested
- set `executor_runtime_state = completed`
- keep final heartbeat/session metadata for trace

### When heartbeat expires
If `executing` card has no fresh heartbeat for threshold:
- mark `executor_runtime_state = stale` or `lost`
- surface in doctor + attention
- recommend relaunch / reconciliation

---

## Thresholds

Keep simple and hardcoded first.
Suggested:
- missing initial heartbeat after dispatch: 10 minutes → warning
- stale heartbeat after runtime start: 20 minutes → warning
- long stale heartbeat: 60 minutes → error/lost

---

## Doctor checks to add

### 1. executing-without-session
`executing` + no `executor_session_id`
Severity: warning

### 2. executing-without-heartbeat
`executing` + no `last_executor_heartbeat_at` after threshold
Severity: warning

### 3. stale-executor-heartbeat
`executing` + heartbeat older than threshold
Severity: warning/error depending on age

### 4. executing-runtime-lost
`executing` + runtime state marked `lost`
Severity: error

---

## UI / control-tower visibility

### Executive / board
Show compact runtime markers:
- runtime: pending
- runtime: running
- runtime: stale
- runtime: lost
- hb: 5m ago

### Card detail
Show full runtime block:
- session id
- started at
- last heartbeat
- runtime state
- progress summary

---

## Manual interim policy

Until automatic OmO launch/heartbeat exists, the operator may act as heartbeat initiator by checking executing cards and updating runtime state manually/reconciliator-side.

This is acceptable only as an interim bridge.
The long-term target is real executor-driven heartbeat updates.

---

## Recommended implementation order

1. extend FeatureCard + schemas with runtime heartbeat fields
2. add helper to record/update executor heartbeat
3. update dispatch path to mark runtime `pending`
4. update result-ingest path to mark runtime `completed`
5. add doctor checks for missing/stale runtime heartbeat
6. show runtime markers in executive/board/card views
7. add a thin CLI surface for manual heartbeat updates/reconciliation if needed

---

## Short formula

`executing` should mean:
**there is a real runtime, a recent heartbeat, and a reconcilable session behind the card**.
