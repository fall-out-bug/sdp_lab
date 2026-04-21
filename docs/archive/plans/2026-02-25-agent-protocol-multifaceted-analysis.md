# 2026-02-25 Agent Protocol Multifaceted Analysis

Status: baseline
Scope: dual-surface SDP strategy for OSS and enterprise profiles

## Context

This analysis captures how SDP should inherit useful ideas from OhMyOpenCode, Gas Town, and Beads without coupling to any one runtime. It supports the roadmap references for F054 and later F068-F077 workstreams.

## Key outcomes

- Define two product surfaces:
  - OSS combine profile: easy bootstrap and high observability for experimentation.
  - Enterprise pack profile: governed K8s runtime, BYOM routing, and strict policy/evidence controls.
- Decommission primitive Ralph-loop behavior for enterprise orchestration.
- Standardize integrations around contracts (orchestration, runtime, policy, evidence) rather than tool-specific glue.

## Integration principles

- Keep Beads as backlog/dependency source of truth for local execution loops.
- Treat GitHub CI as the remote sensor layer for attestations and findings.
- Bridge CI findings into local Beads queue through deterministic sync adapters.
- Keep evidence and policy standards first (in-toto, OPA, Sigstore) across both surfaces.

## Related workstreams

- F054: Continuous Protocol Improvement
- F068-F075: Dual-surface productization and enterprise pack
- F076: Documentation automation agent
- F077: CI-to-local bridge for autonomous improvement loop

## References

- [ECOSYSTEM_SYNERGIES.md](../../integrations/ECOSYSTEM_SYNERGIES.md)
- [ROADMAP.md](../../roadmap/ROADMAP.md)
- [CI_LOCAL_BRIDGE.md](../../runbooks/CI_LOCAL_BRIDGE.md)
