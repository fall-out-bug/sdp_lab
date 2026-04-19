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

1. **Emit Delta**: Write phase delta artifact to `.sdp/phases/<run_id>/<phase>.delta.md`
2. **Open Gate**: Create gate via `bd human <issue-id>` or SDP gate command
3. **Wait**: Block until gate is resolved (human approves/rejects)
4. **Read Verdict**: 
   - Approved → continue to next phase
   - Rejected → return to previous phase with feedback
   - Deferred → pause execution, save checkpoint

## Gate Types
- Plan gate: approve/reject/defer
- Review gate: approve/reject/request-changes
- Eval gate: approve/reject/retry

## Evidence Enforcement

Phase gates require evidence.json present (F134-03). Gates without evidence are blocked.
