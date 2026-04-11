# SDP Dev — Product Requirements Document

Status: active
Updated: 2026-02-21

## Functional Requirements

### P0 — Must Have (block launch)

1. **FR-001: adapter-controller reconcile loop**
   - Watch kubeopencode Task CRDs via informer
   - Reconcile Task status → SDP FSM via LifecycleReconciler
   - Create Task CRDs via IntentTranslator when ready issues appear
   - Pre-dispatch policy gate (model allowlist, risk check)
   - Post-reconcile evidence projection

2. **FR-002: AgentRun CRD**
   - Spec: issueId, repo, baseBranch, model, workstream, timeoutSec
   - Status: phase, conditions, workerTask, reviewerTask, prUrl, lastError
   - Controller: create worker Task → wait → create reviewer Task → terminal status
   - Multi-role: analyst+coder parallel, reviewer sequential

3. **FR-003: CRD type definitions + code generation**
   - api/v1alpha1/types.go with kubebuilder markers
   - DeepCopy, typed clientsets, informers, listers
   - controller-runtime Reconciler

4. **FR-004: Remove Path A**
   - Remove internal/pipeline/k8s_dispatch.go
   - Remove cmd/swarm-orchestrator/ dispatch via pipeline.ExecuteTask
   - Mark SDP_DISPATCH_MODE=k8s as deprecated
   - Keep executor.go quality pipeline as shared library

### P1 — Should Have

5. **FR-005: NATS intake → adapter-controller bridge**
   - adapter-controller subscribes to sdp.intake.* via NATS
   - Creates Task CRD when receiving intake event
   - Publishes sdp.status.* on terminal reconcile

6. **FR-006: Handoff validation (10 consecutive runs)**
   - Each run via operator path (not via kubectl exec)
   - Emit .sdp/runs/orchestrate-*.json
   - Strict evidence valid (pr-gate passes)
   - Duplicate dispatch prevention active

7. **FR-007: Quality pipeline extraction**
   - Extract tests, evidence, provenance, PR gate from executor.go
   - Package as shared package (internal/quality/)
   - Use from adapter post-reconcile hook

### P2 — Nice to Have

8. **FR-008: UP-001 upstream merge follow-up**
   - Monitor PR #50 and feedback
   - Switch adapter from CRD shim to real upstream types

9. **FR-009: UP-002 multi-role dependency gating**
   - Upstream PR for generic task dependency contract
   - DAG-like dependency semantics in operator

10. **FR-010: Observability integration**
    - Traces from adapter → OpenTelemetry Collector
    - CRD event → Loki via controller logs
    - Grafana dashboard for agent runs

## Non-Functional Requirements

| Category | Requirement |
|----------|-------------|
| Availability | Agent runs survive pod restarts (reconcile loop idempotent) |
| Latency | Issue→PR p95 < 20 min |
| Security | RBAC least-privilege, no kubectl exec in production |
| Auditability | Every run has provenance chain, evidence envelope, trace links |
| Testability | Adapter components unit-tested, controller tested via envtest |
| Portability | CRD shim decouples from upstream merge timeline |

## Layered Architecture Requirements (OSS-First)

Reference: `docs/vision/SDP_LAYERED_VISION.md`

### L1 — Protocol Contracts

11. **FR-011: skill/agent IO contracts**
    - Every skill/command/agent defines stable input/output contract.
    - Complex tasks define subagent fan-out/fan-in behavior.
    - Self-check criteria are explicit and machine-testable.

### L2 — Runtime Governance

12. **FR-012: contract drift enforcement at harness level**
    - Immutable task contract baseline for feature runs.
    - Gate blocks on acceptance criteria or metric degradation.
    - Clarifications classified as no-impact/additive/reductive/policy-sensitive.

13. **FR-013: provenance + evidence default policy**
    - No completion claim without evidence references.
    - CI-level protocol compliance check required on PRs with contract artifacts.

### L3 — Orchestration Fabric

14. **FR-014: orchestrator-managed phase gating**
    - Phase transitions must pass contract gates before advance.
    - Drift and gate outcomes persisted as machine-readable reports.

15. **FR-015: ecosystem bridge contracts**
    - Interop contracts for OpenCode and Gas Town adapters.
    - Beads + vibe-kanban integration points remain protocol-boundary compatible.

### L4 — Enterprise Trust Envelope

16. **FR-016: signed evidence envelope standards**
    - Enterprise mode enforces stricter signature and audit requirements.
    - Same protocol semantics as OSS, stronger policy defaults.

### L5 — OSS Harness Runtime

17. **FR-017: portable harness runtime**
    - K8s/operator execution path remains OSS.
    - Runtime can be deployed independently from enterprise controls.
