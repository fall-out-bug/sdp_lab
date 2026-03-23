# Control Tower Executive Surface — Next Step

Status: working next step
Date: 2026-03-23
Scope: first thin operator-home slice for the control tower

## Goal

Create the first real executive operator surface for SDP.

## Do now

1. Upgrade `sdp attention` into the executive summary surface.
2. Make it show, in compact order:
   - attention now
   - waiting on human
   - blocked
   - executing / review / delivery movement
   - delivery trouble
   - ready to move
   - friction hotspots
   - next best action
3. Reuse existing portfolio snapshot/read-model data where possible.
4. Add only thin derived summary logic if needed.
5. Keep `--json` support.

## Constraints

- thin slice only
- no huge dashboard work
- no heavy analytics layer
- no broad command-surface churn unless clearly worth it

## Desired outcome

A colleague should be able to run one command and immediately understand:
- what matters now
- what is broken or risky
- what is moving
- what they should do next
