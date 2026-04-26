# SDP Trust Guarantees

> **Canonical source** for security and integrity claims. All product documentation, marketing, and compliance materials MUST reference this document rather than making independent claims.
> **Last updated:** 2026-04-26

## What SDP Guarantees

These guarantees hold for all GA-maturity components (see [maturity-matrix.md](./maturity-matrix.md)).

### Evidence Integrity

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Evidence artifacts are tamper-evident | in-toto attestation format with SHA-256 content hashes | `go run ./cmd/sdp-evidence validate <file>` |
| Evidence schema is validated before gate passage | JSON Schema validation in `evidence-gate` CI job | Schema at `schema/evidence.schema.json` |
| Auto-attestations are signed | Sigstore keyless signing of attestation bundles in `auto-attestation` CI job | `.sdp/attestations/ci-auto.bundle` |

### Scope Enforcement

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Changes outside declared WS scope are detected | `scope-gate` CI job runs `sdp-guard --ws <ws>` per workstream | CI job output |
| Contract compliance is verified | `protocol-compliance` CI job validates contracts against snapshots | `go run ./cmd/sdp-guard --check-contract` |

### CI Gate Consistency

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Build and test gates are reproducible locally | `./scripts/run_go_quality_gates.sh` runs `go build`, `go test`, `go vet` locally | See [ci-gates-map.md](./ci-gates-map.md) |
| CI workflow is version-controlled | `.github/workflows/ci.yml` in repo | `git log .github/workflows/ci.yml` |
| Policy enforcement is configurable | OPA/Rego policies in `.sdp/policies/`, mode via `SDP_POLICY_ENFORCEMENT_MODE` env var | `opa eval --data .sdp/policies/` |

### Audit Trail

| Guarantee | Mechanism | Verification |
|-----------|-----------|--------------|
| Every merge has a PR with gate evidence | Push protection + `required-checks` gate validates all 12 CI jobs | GitHub branch protection |
| Attestations are persisted as CI artifacts | `auto-attestation` uploads signed bundles (90-day retention) | `.sdp/attestations/ci-auto.json` |
| Coverage baselines are tracked | `coverage-gate` compares against `.sdp/metrics/coverage.txt`, auto-updates on main push | `git log .sdp/metrics/coverage.txt` |

## What SDP Does NOT Guarantee

Honest disclosure of limitations. These must not be overclaimed in any documentation.

### Not Tamper-Proof

SDP uses **tamper-evident** mechanisms (hashes, attestations), not tamper-proof ones. An adversary with write access to the git repository or CI environment can modify artifacts. The guarantee is detectability, not prevention.

### Not Non-Repudiation

SDP evidence artifacts identify the actor (human or agent) that produced them, but do not provide cryptographic non-repudiation. Auto-attestations are signed with CI identity (Sigstore keyless), not individual actor keys. Legal non-repudiation requires additional controls outside SDP scope.

### Not a Compliance Framework

SDP provides **evidence** and **process enforcement** that supports compliance with frameworks like EU AI Act (2026-08) and NIST AI RMF. SDP is not itself a compliance certification or legal opinion. Customers must perform their own compliance assessment using SDP outputs as inputs.

### Not Secret Management

SDP does not include a dedicated secret scanning CI gate. Secret detection is the responsibility of external tools (e.g., GitHub secret scanning, truffleHog). Integrating secret scanning into the SDP gate workflow is a planned enhancement.

### Not a Substitute for Code Review

SDP gates automate mechanical checks (format, lint, test, scope, schema). They do not evaluate code quality, architectural decisions, or business logic correctness. Human code review remains required.

### Policy Enforcement is Advisory by Default

The `policy-gate` CI job evaluates OPA policies but defaults to `advisory` mode. Denials are logged but do not block merge unless `SDP_POLICY_ENFORCEMENT_MODE=blocking` is set. Customers who need enforced policy must configure this explicitly.

## Required Customer Controls

For the guarantees above to hold, customers MUST:

1. **Enable branch protection** on the main branch with required status checks matching the `required-checks` job.
2. **Set `SDP_POLICY_ENFORCEMENT_MODE=blocking`** if policy gate denials should block merge.
3. **Maintain CI runner security** — SDP evidence is only as trustworthy as the CI environment that produces it.
4. **Maintain `.sdp/metrics/coverage.txt`** baseline — the coverage gate fails if no baseline exists.
5. **Review evidence artifacts** before relying on them for compliance submissions — SDP collects and validates evidence; humans must interpret it.

## Maturity Dependencies

| Guarantee | Minimum Component Maturity |
|-----------|---------------------------|
| Evidence integrity | GA (evidence-gate, auto-attestation, sdp-evidence) |
| Scope enforcement | GA (scope-gate, protocol-compliance, sdp-guard) |
| CI consistency | GA (build-test, quality gates script, ci.yml) |
| Audit trail | GA (required-checks, push-protection, auto-attestation) |

For Beta and Experimental components, guarantees are best-effort. See [maturity-matrix.md](./maturity-matrix.md) for component-level status.

## Canonical Wording

Use these phrases exactly in product docs, README, and compliance materials:

- **Instead of** "tamper-proof" → "tamper-evident with SHA-256 content hashes"
- **Instead of** "guarantees compliance" → "produces evidence supporting compliance assessment"
- **Instead of** "fully automated quality" → "automated mechanical checks; human review required"
- **Instead of** "secure by default" → "configurable policy enforcement with OPA, advisory by default"

---

*Related: [maturity-matrix.md](./maturity-matrix.md), [ci-gates-map.md](./ci-gates-map.md), [quality-gates.md](./quality-gates.md), [EVIDENCE-COVERAGE.md](./EVIDENCE-COVERAGE.md)*
