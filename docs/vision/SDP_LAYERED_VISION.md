# SDP Layered Vision

Status: active
Updated: 2026-03-03

## North Star

SDP is a trust layer for AI-driven software delivery: every material agent action must be explainable, reproducible, and verifiable through provenance and evidence.

Product thesis:

- OSS-first around OpenCode for broad adoption.
- Enterprise controls as stricter upper layers, not a separate product identity.
- Portable architecture: each layer can be adopted independently.

## Layer Model

### Layer 1: Protocol

Scope:

- Skill, command, and agent contracts.
- Input/output schemas.
- Self-check obligations.
- Subagent orchestration contracts for complex and parallel execution.

Design rule: protocol is declarative and portable across runtimes.

### Layer 2: Runtime Quality + Governance

Scope:

- Hooks and baseline CLI guardrails.
- Drift control and promise verification.
- Provenance + evidence policy enforcement.

Design rule: no silent degradation of requirements, metrics, or claims.

### Layer 3: Orchestration Fabric

Scope:

- Multi-agent orchestration.
- Integrations with OhMyOpenCode and Gas Town.
- Message bus, Beads tracker, vibe-kanban coordination.

Design rule: orchestrator owns phase transitions and gate decisions.

### Layer 4: Enterprise Trust Envelope

Scope:

- Strict provenance and evidence chain.
- Digital signature standards for evidence envelopes.
- Compliance-grade auditability.

Design rule: same core protocol, stronger controls and policy posture.

### Layer 5: OSS Harness Runtime

Scope:

- Kubernetes-native harness for agent execution.
- Evolution of OpenCode operator model.
- Scale, reliability, and observability for autonomous delivery loops.

Design rule: operational runtime remains OSS while enterprise policy remains optional add-on.

## Design Principles

- Contract-first behavior over prompt-only behavior.
- Evidence-default assertions (no evidence, no claim).
- Governance as code (versioned and reviewable).
- Layer independence with explicit interfaces.
- Compatibility-first evolution (backward-compatible schema additions by default).

## Success Criteria

- 100% of completed runs include a valid evidence envelope.
- 100% of delivery claims are evidence-linked.
- Drift incidents are detected before phase completion.
- Protocol portability: same contracts run across at least two runtimes (OpenCode + one adapter surface).
- CI enforces mandatory trust checks for protected branches.

Related:

- `docs/roadmap/UNIFIED_VISION_ROADMAP_2026-03-03.md`
- `docs/vision/REPO_PROMOTION_VISION.md`
