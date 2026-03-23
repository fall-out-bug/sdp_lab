# Process Hygiene Telemetry — Next Step

Status: working next step
Date: 2026-03-23
Phase: 4.1

## Goal

Add the first thin process-hygiene telemetry slice to `sdp doctor control`.

## Do now

1. extend `DoctorControl()` with warning-level debt checks for:
   - stale `ready` cards
   - stale `needs_input` cards
   - stale `blocked` cards
   - `executing` cards missing dispatch metadata
   - `done` cards missing executor result summary

2. keep thresholds simple and hardcoded for the first slice
3. add focused tests
4. update docs to reflect that doctor now surfaces both invariant failures and hygiene debt

## Constraints

- no new dashboard
- no scheduler/daemon
- no config system for thresholds yet
- no report-contract rewrite unless truly necessary
- keep implementation narrow and local

## Desired outcome

After this slice, `sdp doctor control` should answer not only:
- what is structurally broken?

but also:
- what is quietly going stale or under-traced?
