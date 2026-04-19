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
3. **Strict Mode**: Use `--strict` to enforce evidence validation. Gate is blocked if evidence is missing or invalid. Without `--strict`, gates auto-approve but print a warning if evidence is absent.
4. **Trace**: Each run writes a trace record to `.sdp/phases/<run_id>/trace.json`
5. **Open Gate**: Create gate via `bd human <issue-id>` or SDP gate command
6. **Wait**: Block until gate is resolved (human approves/rejects)
7. **Read Verdict**:
   - Approved -> continue to next phase
   - Rejected -> return to previous phase with feedback
   - Deferred -> pause execution, save checkpoint

## Gate Types
- Plan gate: approve/reject/defer
- Review gate: approve/reject/request-changes
- Eval gate: approve/reject/retry

## Evidence Enforcement

Phase gates require `--evidence-path` to point to a valid JSON file with the required keys for that gate type. In strict mode (`--strict`), evidence is mandatory and the gate blocks on validation failure. In non-strict mode, gates auto-approve but warn when evidence is missing.
