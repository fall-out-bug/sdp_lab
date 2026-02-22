# Feature: adapter-controller Reconcile Loop (FR-001)

Priority: P0
Effort: 3 days
Depends on: FR-003

## Problem

adapter-controller — placeholder (82 lines, demo). No watch, no informer, no reconcile. Adapter layer (5 components) is tested, but not connected to real CRD events.

## Scope

1. Replace demo main.go with controller-runtime Manager
2. TaskReconciler: watch Task CRDs
   - On Pending: IntentTranslator + PolicyGate + RunLockManager → claim Beads issue
   - On Running: heartbeat trace
   - On Succeeded/Completed: EvidenceProjector + quality pipeline + FSM review
   - On Failed: LifecycleReconciler → blocked/escalated
3. NATS subscriber for sdp.intake.* → Task CRD creation
4. Status write-back to Task CRD annotations

## Acceptance Criteria

- controller starts, connects to kubeopencode-system
- On Task CRD creation — reconcile invokes adapter components
- On terminal status — Beads issue is updated via FSM
- Evidence envelope is written to .sdp/evidence/
- envtest test passes

## Architecture

```
adapter-controller (controller-runtime Manager)
  ├── TaskReconciler (watches Task CRD)
  │   ├── IntentTranslator
  │   ├── PolicyGate
  │   ├── RunLockManager
  │   ├── LifecycleReconciler
  │   └── EvidenceProjector
  └── NATS subscriber (sdp.intake.*)
      └── creates Task CRD
```
