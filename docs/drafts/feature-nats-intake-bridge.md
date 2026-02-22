# Feature: NATS Intake → Adapter Bridge (FR-005)

Priority: P1
Effort: 1 day
Depends on: FR-001

## Problem

adapter-controller is not connected to the intake flow. Intake events arrive via NATS (sdp.intake.*), but adapter does not listen to them. Bridge publishes sdp.beads.*.ready, but the recipient is swarm-orchestrator (Path A), not adapter.

## Scope

1. adapter-controller subscribes to `sdp.beads.*.ready`
2. On receiving ready event → IntentTranslator → Task CRD creation
3. On terminal reconcile → publishes `sdp.status.*` to NATS
4. federation/bridge remains as intake layer (poll Beads → NATS)

## Acceptance Criteria

- intake-gateway POST → NATS → adapter creates Task CRD
- Terminal status → NATS event → bridge updates Beads
- Multi-project: per-project bridge, all feed into adapter
