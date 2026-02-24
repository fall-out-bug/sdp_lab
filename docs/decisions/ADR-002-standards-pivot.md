# ADR-002: Standards Pivot — in-toto, OPA, Sigstore

> **Status:** Accepted
> **Date:** 2026-02-24
> **Context:** Phase 0 enforcement audit, enforcement hypotheses research

## Decision

Pivot SDP from custom evidence/enforcement to standards-based architecture:

- **Evidence format:** in-toto attestation with custom predicate type (`coding-workflow/v1`)
- **Policy engine:** OPA/Rego (declarative, executable, versioned)
- **Signing:** Sigstore (keyless, OIDC-based)
- **Enforcement:** CI gates + GitHub branch protection (server-side, bypass-proof)
- **K8s code:** Archived to `archive/k8s-v0` branch for future rebuild on standards

## Context

Phase 0 (14 features, all Done) built tools but not enforcement. Audit showed:
- 7% real enforcement, 43% cleanup, 50% potential
- All controls inside orchestration pipeline; any path around it bypasses everything
- CI has no evidence validation; branch protection not configured
- Evidence envelope is custom 9-section JSON (3,400 LOC) when in-toto attestation is the industry standard
- Policies are markdown, not executable

Research found the problem is solved in adjacent domains:
- Supply chain security: SLSA, in-toto, Tekton Chains, Sigstore
- Policy-as-code: OPA/Rego, Kyverno
- Agent governance: MI9, AgentSpec, A2AS (academic, not deployed)
- Commercial: GLACIS, TrustPlane (LLM API calls, not coding workflow)

## Consequences

- K8s/swarm code archived, not deleted — domain knowledge preserved for rebuild
- Evidence envelope rewritten as in-toto predicate — 9 sections become predicate, DSSE handles signing
- Enforcement moves to merge boundary — CI gates + branch protection
- Policies become executable — OPA/Rego replaces quality-gates.md
- K8s dream stays — rebuilt properly in Phases 8-9 with Kyverno admission, Tekton Chains auto-attestation

## Alternatives Considered

1. **Fix current system:** Add CI gates without changing formats. Rejected — custom format limits ecosystem integration.
2. **Adopt GLACIS/TrustPlane:** Commercial tools for LLM API governance. Rejected — wrong layer (inference vs development process).
3. **Build everything custom:** Continue current path. Rejected — reinventing solved problems.

## References

- [Phase 0 Enforcement Audit](../plans/2026-02-24-phase0-enforcement-audit.md)
- [Enforcement Hypotheses](../plans/2026-02-24-enforcement-hypotheses.md)
- [Standards Pivot Plan](.cursor/plans/sdp_standards_pivot_*.plan.md)
