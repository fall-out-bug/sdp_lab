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
- Emit review delta with findings
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
