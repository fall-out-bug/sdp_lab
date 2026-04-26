# SDP Trust Guarantees

> **Canonical source** for security and integrity claims. All product documentation, marketing, and compliance materials MUST reference this document rather than making independent claims.
> **Last updated:** 2026-04-26

## What SDP Guarantees (P0)

These guarantees hold for all GA-maturity components (see [maturity-matrix.md](./maturity-matrix.md)).

### Evidence Integrity

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Evidence artifacts are tamper-evident | in-toto attestation format with SHA-256 content hashes | `sdp-evidence verify` |
| Evidence schema is validated before gate passage | JSON Schema validation in `evidence-gate` CI job | Schema at `schema/evidence.schema.json` |
| Gate passage is recorded, not just gate configuration | Gate resolution writes outcome to `.sdp/gates/` | `sdp gate status` |

### Scope Enforcement

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Changes outside declared WS scope are detected | `scope-gate` CI job compares diff against WS `touches:` | CI job output |
| Out-of-scope edits block merge (fail-closed) | Policy gate aggregates scope-gate result | `sdp guard check` |
| WS scope is declared upfront, not retrofitted | WS frontmatter `touches:` field | WS file validation |

### CI Gate Consistency

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Gates run the same checks locally and in CI | `./scripts/run_go_quality_gates.sh` is the canonical gate runner | See [ci-gates-map.md](./ci-gates-map.md) |
| Gate configuration is version-controlled | `.sdp/guard-rules.yml` in repo root | `git log .sdp/guard-rules.yml` |
| Gate failures are never silently ignored | Fail-closed mode for all GA gates; fail-open only for Beta gates with explicit annotation | `.sdp/guard-rules.yml` mode field |

### Audit Trail

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Every merge has a PR with gate evidence | Push protection + required status checks | GitHub branch protection |
| Evidence artifacts are immutable after gate passage | Content-addressable storage (SHA-256 keyed) | `sdp-evidence verify` |
| Phase transitions require evidence | `gate.ResolveWithEvidence()` for phase-typed gates | See [EVIDENCE-COVERAGE.md](./EVIDENCE-COVERAGE.md) |

## What SDP Does NOT Guarantee

Honest disclosure of limitations. These must not be overclaimed in any documentation.

### Not Tamper-Proof

SDP uses **tamper-evident** mechanisms (hashes, attestations), not tamper-proof ones. An adversary with write access to the git repository or CI environment can modify artifacts. The guarantee is detectability, not prevention.

### Not Non-Repudiation

SDP evidence artifacts identify the actor (human or agent) that produced them, but do not provide cryptographic non-repudiation. Evidence is signed with repository-level keys, not individual actor keys. Legal non-repudiation requires additional controls outside SDP scope.

### Not a Compliance Framework

SDP provides **evidence** and **process enforcement** that supports compliance with frameworks like EU AI Act (2026-08) and NIST AI RMF. SDP is not itself a compliance certification or legal opinion. Customers must perform their own compliance assessment using SDP outputs as inputs.

### Not Secret Management

SDP scans for leaked credentials (`secretscan` gate) but does not manage secrets, rotate keys, or enforce secret access policies. Use a dedicated secrets manager (HashiCorp Vault, AWS Secrets Manager, etc.) for secret lifecycle management.

### Not a Substitute for Code Review

SDP gates automate mechanical checks (format, lint, test, scope, schema). They do not evaluate code quality, architectural decisions, or business logic correctness. Human code review remains required.

## Required Customer Controls

For the guarantees above to hold, customers MUST:

1. **Enable branch protection** on the main branch with required status checks matching SDP gates.
2. **Restrict write access** to `.sdp/guard-rules.yml` to authorized operators only.
3. **Run `sdp doctor`** periodically to detect configuration drift between local and CI environments.
4. **Maintain CI runner security** — SDP evidence is only as trustworthy as the CI environment that produces it.
5. **Review evidence artifacts** before relying on them for compliance submissions — SDP collects and validates evidence; humans must interpret it.

## Maturity Dependencies

| Guarantee | Minimum Component Maturity |
|-----------|---------------------------|
| Evidence integrity | GA (evidence-gate, sdp-evidence) |
| Scope enforcement | GA (scope-gate, sdp-guard) |
| CI consistency | GA (build-test, quality gates script) |
| Audit trail | GA (policy-gate, push-protection) |

For Beta and Experimental components, guarantees are best-effort. See [maturity-matrix.md](./maturity-matrix.md) for component-level status.

## Canonical Wording

Use these phrases exactly in product docs, README, and compliance materials:

- **Instead of** "tamper-proof" → "tamper-evident with SHA-256 content hashes"
- **Instead of** "guarantees compliance" → "produces evidence supporting compliance assessment"
- **Instead of** "fully automated quality" → "automated mechanical checks; human review required"
- **Instead of** "secure by default" → "fail-closed gates with version-controlled policy"

---

*Related: [maturity-matrix.md](./maturity-matrix.md), [ci-gates-map.md](./ci-gates-map.md), [quality-gates.md](./quality-gates.md), [EVIDENCE-COVERAGE.md](./EVIDENCE-COVERAGE.md)*
