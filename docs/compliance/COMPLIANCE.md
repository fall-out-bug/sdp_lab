# SDP Compliance Reference

> **Last updated:** 2026-04-26
> **Regulatory target:** EU AI Act (effective 2026-08), NIST AI RMF

## Overview

SDP provides **evidence collection** and **process enforcement** that supports compliance assessments. SDP is not itself a compliance certification. See [trust-guarantees.md](../reference/trust-guarantees.md) for the canonical statement of guarantees and limitations.

## Evidence Artifacts per Compliance Control

| Compliance Control | SDP Artifact | Gate | Local Reproduce |
|--------------------|--------------|------|-----------------|
| Test execution evidence | `.sdp/evidence/*.json` (type: verification) | evidence-gate | `./scripts/run_go_quality_gates.sh` |
| Scope containment evidence | `.sdp/evidence/*.json` (type: plan) | scope-gate | `sdp guard check` |
| Policy conformance evidence | Policy summary JSON | policy-gate | `sdp gate status` |
| Coverage metrics | Coverage report | coverage-gate | `go test -coverprofile=cover.out ./...` |
| Consistency check results | Guard-rules report | consistency-gate | `sdp verify` |

## Phase Gate Evidence Requirements

Per F134-03 (gate evidence enforcement), phase-typed gates require an `evidence.json` artifact:

- **Plan phase**: test coverage data + design checklist
- **Review phase**: spec-reviewer output + code-review verdict
- **Eval phase**: `go test` + `go vet` + `protocol-check` + smoke test results

## Customer Responsibilities

See [trust-guarantees.md](../reference/trust-guarantees.md) "Required Customer Controls" section.

---

*Source: [trust-guarantees.md](../reference/trust-guarantees.md), [maturity-matrix.md](../reference/maturity-matrix.md)*
