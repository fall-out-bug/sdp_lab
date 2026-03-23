# Process Hygiene Telemetry Spec

Status: working spec
Date: 2026-03-23
Phase: 4
Scope: first thin operator-health slice on top of `sdp doctor control`

## Purpose

Phase 4 should make SDP visibly useful as an operator surface, not just a storage/lifecycle engine.

The first thin slice is **process hygiene telemetry**:
- surface stale work
- surface waiting-on-human debt
- surface blocked debt
- surface missing trace/evidence links
- do it through the existing `sdp doctor control` path first

This is intentionally not a dashboard project.
It is a narrow operational sharpness pass.

---

## Why this slice

SDP already has:
- control-store lifecycle
- stage packs
- template discipline
- baseline doctor checks

What it still lacks is a stronger answer to:

**What is silently going stale, blocked, or under-specified right now?**

That is the job of process hygiene telemetry.

---

## Core principle

Start with **operator-relevant debt detection**, not visualization glamour.

Meaning:
- detect debt first
- report it in a concise, machine-readable and operator-readable form
- only later build richer views on top

---

## Thin-slice target

Extend `sdp doctor control` with warning/error checks for the highest-value debt surfaces.

### First checks to add

#### 1. stale-ready-card
A card is `ready` but has not moved for too long.

Why it matters:
- shaped work is piling up
- orchestration/dispatch may be not happening
- the board may look healthy while execution stalls

Suggested default logic:
- status = `ready`
- `updated_at` older than a small threshold (for example 72h)
- severity = warning

#### 2. stale-needs-input-card
A card is `needs_input` and has been waiting too long.

Why it matters:
- feedback debt accumulates quietly
- waiting-on-human work disappears into the floorboards

Suggested default logic:
- status = `needs_input`
- `updated_at` older than threshold (for example 48h)
- severity = warning

#### 3. stale-blocked-card
A card is `blocked` and has been stuck too long.

Why it matters:
- blocked debt needs explicit operator attention
- otherwise the system pretends blocked cards are just “existing”

Suggested default logic:
- status = `blocked`
- `updated_at` older than threshold (for example 72h)
- severity = warning

#### 4. executing-without-dispatch-metadata
A card is `executing` but dispatch metadata is incomplete.

Suggested logic:
- status = `executing`
- one or more of these missing:
  - `dispatched_at`
  - `dispatched_to`
  - `dispatched_packet_path`
- severity = warning

This is distinct from `executing-without-beads`.
It checks execution trace completeness, not just Beads linkage.

#### 5. done-without-result-summary
A card is `done` but `executor_result` is missing.

Why it matters:
- the lifecycle claims completion without evidence of the terminal execution outcome

Suggested logic:
- status = `done`
- `executor_result == nil`
- severity = warning

---

## Threshold policy

For the first slice, keep thresholds simple and hardcoded.
Do not build config infrastructure yet.

Suggested defaults:
- `needs_input`: 48h
- `ready`: 72h
- `blocked`: 72h

These are not sacred. They are first operator-signal defaults.

---

## Report semantics

`DoctorReport` already has:
- `TotalChecks`
- `Passed`
- `Failed`
- `Checks[]`

For this slice, keep the report shape stable if possible.
Use `Severity` to distinguish:
- `error` = invariant/state correctness problem
- `warning` = process debt / hygiene signal

Interpretation:
- `Failed` should continue to mean emitted checks, including warnings if the current report model stays simple
- do not redesign the whole report contract unless truly needed for the thin slice

If future refinement is needed, add explicit warning counters in a later slice.
Not now unless implementation pressure makes it obviously worth it.

---

## CLI output goal

Operator should be able to run:

```bash
sdp doctor control
```

and get immediate visibility into:
- broken invariants
- cards quietly rotting in `ready`
- cards stuck in `needs_input`
- blocked debt
- execution trace gaps
- completion-without-result gaps

The output should remain concise.
No giant essay in CLI.

---

## Non-goals for this slice

Do not do these yet:
- dashboard or TUI
- configurable thresholds
- background scheduler
- notification engine
- portfolio scoring framework
- historical trend analytics
- debt scoring system

This slice is about **first useful telemetry**, not telemetry empire-building.

---

## Acceptance criteria

This slice is good enough when:
- `sdp doctor control` can detect at least 4-5 concrete debt surfaces beyond current invariant checks
- new checks are covered by tests
- existing report shape remains understandable
- docs mention the new telemetry role of doctor control
- the implementation stays thin and local to the current control-store model

---

## Recommended implementation order

1. add time parsing helper for `updated_at`
2. implement stale status checks for `ready`, `needs_input`, and `blocked`
3. implement trace-completeness checks for `executing` and `done`
4. add focused tests in `internal/control/control_test.go`
5. update `docs/CONTROL_STORE_SKELETON.md` and roadmap notes

---

## Short formula

- invariant checks catch **broken state**
- hygiene telemetry catches **rotting state**
- Phase 4 starts by adding both through `sdp doctor control`
