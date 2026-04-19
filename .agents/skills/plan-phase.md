---
name: plan-phase
description: Execute the Plan phase of the SDP pipeline with evidence enforcement.
version: 1.0.0
tags:
  - phase
  - planning
  - compliance
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# plan-phase

## Purpose

Execute the Plan phase: analyze requirements, design approach, emit plan delta artifact, open gate for human approval.

## MUST DO
- Load feature context from beads issue and workstream files
- Analyze existing codebase to inform plan
- Emit plan delta artifact: `sdp phase plan --feature-id <F> --strict`
- Wait for human approval before proceeding
- Record plan decision rationale

## MUST NOT DO
- Skip human approval gate
- Proceed to build phase without plan approval
- Modify code during plan phase (analysis only)

## Response Format
- Plan delta with ADDED/MODIFIED/REMOVED sections
- Rationale for design decisions
- Risk assessment
- Files affected list
