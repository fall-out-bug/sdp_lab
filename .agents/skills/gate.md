---
name: gate
description: Shared gate boundary protocol for SDP phase skills.
version: 1.0.0
tags:
  - gate
  - boundary
  - compliance
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# gate

## Purpose

Shared gate boundary protocol invoked by ANY phase skill at phase boundary.

## Protocol

1. **Emit Delta**: Run `sdp phase <plan|review|eval> --feature-id <F> --evidence-path <path>` to create the gate. Delta artifact is persisted to `.sdp/phases/<run_id>/<phase>.delta.md`
2. **Evidence**: Provide `--evidence-path <path>` pointing to a JSON file with the required keys for the phase gate. Required keys:
   - Plan: `test_coverage`, `design_checklist`
   - Review: `spec_review_verdict`, `code_review_verdict`
   - Eval: `go_test`, `go_vet`, `protocol_check`, `smoke`
3. **Strict Mode**: Use `--strict` to enforce evidence validation. Gate is blocked if evidence is missing or invalid. The gate object is persisted to `.sdp/phases/<run_id>/gate.json` in AWAITING state. To approve: edit gate.json to set `answer` (approve/reject), `answerer` (human name), and `resolved_at` (ISO timestamp). Without `--strict`, gates auto-approve but print a warning if evidence is absent.
4. **Trace**: Each run writes a trace record to `.sdp/phases/<run_id>/trace.json`
5. **Gate Persistence**: Gate object persisted to `.sdp/phases/<run_id>/gate.json`. In strict mode, the gate stays AWAITING until a human edits gate.json or beads integration is complete.

## Gate Types
- Plan gate: approve/reject/defer
- Review gate: approve/reject/request-changes
- Eval gate: approve/reject/retry

## Evidence Enforcement

Phase gates require `--evidence-path` to point to a valid JSON file with the required keys for that gate type. In strict mode (`--strict`), evidence is mandatory and the gate blocks on validation failure. In non-strict mode, gates auto-approve but warn when evidence is missing.
