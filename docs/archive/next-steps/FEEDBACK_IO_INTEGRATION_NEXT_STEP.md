# Feedback I/O Integration — Next Step

Status: working next step
Date: 2026-03-22
Scope: bridge internal feedback/resume flow to external author/admin communication loop

## Goal

Extend the current control-store feedback/resume model so that it can participate in a real external loop:
- generate outbound feedback/update packets
- accept normalized inbound answers
- map those answers back to a `FeatureCard`
- resume orchestration with minimal manual glue

## Current baseline

Already implemented:
- `card-needs-input`
- `card-feedback`
- `card-resume`
- feedback-related card fields
- ready gate
- Beads bridge for `ready -> executing`

What is still missing is the practical I/O edge.

## Next implementation target

Implement a thin external feedback integration layer in `sdp_lab`:

1. define a normalized outbound feedback packet shape
2. define a normalized inbound answer/update shape
3. add CLI surfaces/helpers to emit and ingest these packets
4. wire ingested answers into `card-resume`
5. preserve orchestrator-centric ownership of state transitions

## Constraints

- do not redesign architecture
- do not couple directly to one messaging provider
- keep the implementation provider-agnostic
- board remains visualization, not source of truth
- prefer thin practical implementation over framework-building

## Desired outcome

A control card can:
- produce a message-ready feedback packet for author/admin
- accept a normalized answer payload later
- resume automatically back into clarifying/ready flow

That closes the next major orchestration gap.
