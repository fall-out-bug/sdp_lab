---
name: eval-phase
description: Execute the Eval phase of the SDP pipeline with evidence enforcement.
version: 1.0.0
tags:
  - phase
  - eval
  - compliance
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# eval-phase

## Purpose

Execute the Eval phase: run quality gates, check attestation, aggregate quality metrics, open gate for eval approval.

## MUST DO
- Run quality gates: `./scripts/run_go_quality_gates.sh`
- Check attestation evidence (F134-03)
- Aggregate quality metrics from all phases
- Emit eval delta: `sdp phase eval --feature-id <F> --evidence-path <path>`
  - `--evidence-path` points to a JSON file with keys: `go_test`, `go_vet`, `protocol_check`, `smoke`
  - `--strict` requires evidence and blocks on validation failure
  - Without `--strict`, the gate auto-approves (non-strict mode) but prints a warning if evidence is missing
- Delta artifact persisted to `.sdp/phases/<run_id>/eval.delta.md`
- Trace record written to `.sdp/phases/<run_id>/trace.json`
- Wait for human approval before completion
- Record eval decision rationale

## MUST NOT DO
- Skip human approval gate
- Complete feature without eval approval
- Modify implementation during eval phase (assessment only)

## Response Format
- Eval delta with QUALITY/ATTESTATION/METRICS sections
- Quality gate pass/fail status
- Attestation completeness check
- Approval/rejection rationale
