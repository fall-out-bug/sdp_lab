---
name: review-phase
description: Execute the Review phase of the SDP pipeline with evidence enforcement.
version: 1.0.0
tags:
  - phase
  - review
  - compliance
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# review-phase

## Purpose

Execute the Review phase: verify implementation against plan, check evidence enforcement, open gate for review approval.

## MUST DO
- Load plan delta from previous phase
- Verify implementation matches plan artifacts
- Check evidence enforcement (F134-03)
- Emit review delta: `sdp phase review --feature-id <F> --evidence-path <path>`
  - `--evidence-path` points to a JSON file with keys: `spec_review_verdict`, `code_review_verdict`
  - `--strict` requires evidence and blocks on validation failure
  - Without `--strict`, the gate auto-approves (non-strict mode) but prints a warning if evidence is missing
- Delta artifact persisted to `.sdp/phases/<run_id>/review.delta.md`
- Trace record written to `.sdp/phases/<run_id>/trace.json`
- Wait for human approval before proceeding
- Record review decision rationale

## MUST NOT DO
- Skip human approval gate
- Proceed to eval phase without review approval
- Modify implementation during review phase (verification only)

## Response Format
- Review delta with VERIFIED/DEVIATIONS/MISSING sections
- Evidence compliance status
- Risk assessment
- Approval/rejection rationale
