# SDP Compliance Reference

> **Last updated:** 2026-04-26
> **Regulatory target:** EU AI Act (effective 2026-08), NIST AI RMF

## Overview

SDP provides **evidence collection** and **process enforcement** that supports compliance assessments. SDP is not itself a compliance certification. See [trust-guarantees.md](../reference/trust-guarantees.md) for the canonical statement of guarantees and limitations.

## Evidence Artifacts per Compliance Control

| Compliance Control | SDP Artifact | Gate | Local Reproduce |
|--------------------|--------------|------|-----------------|
| Test execution evidence | `.sdp/evidence/*.json` | evidence-gate | `go run ./cmd/sdp-evidence validate --require-pr-url=false <file>` |
| Scope containment evidence | `.sdp/checkpoints/*.json` | scope-gate | `go run ./cmd/sdp-guard --ws <ws-id>` |
| Contract compliance evidence | `.sdp/contracts/F*.json` + snapshots | protocol-compliance | `go run ./cmd/sdp-guard --check-contract --contract <file> --snapshot <file>` |
| Coverage metrics | `cov.out`, `.sdp/metrics/coverage.txt` | coverage-gate | `go test -tags sqlite_fts5 -coverprofile=cover.out ./... && go tool cover -func=cover.out` |
| Repo consistency results | `.sdp/findings/*.json` | consistency-gate | `python3 scripts/check_repo_consistency.py --strict-ac --json` |
| Signed attestation | `.sdp/attestations/ci-auto.json` + `.bundle` | auto-attestation | `go run ./internal/evidence/cmd/auto-attest --branch <branch>` |
| Policy evaluation | OPA evaluation log | policy-gate | `opa eval --data .sdp/policies/ --input <input.json> 'data.sdp.policies.effective_deny'` |

## Phase Gate Evidence Requirements

Per F134-03 (gate evidence enforcement), phase-typed gates require an `evidence.json` artifact:

- **Plan phase**: test coverage data + design checklist
- **Review phase**: spec-reviewer output + code-review verdict
- **Eval phase**: `go test` + `go vet` + `protocol-check` + smoke test results

## Customer Responsibilities

See [trust-guarantees.md](../reference/trust-guarantees.md) "Required Customer Controls" section.

---

*Source: [trust-guarantees.md](../reference/trust-guarantees.md), [maturity-matrix.md](../reference/maturity-matrix.md), [ci-gates-map.md](../reference/ci-gates-map.md)*
