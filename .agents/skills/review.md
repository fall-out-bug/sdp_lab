---
name: review
description: Engineering quality review — code, architecture, security, and release readiness.
version: 1.0.0
tags:
  - review
  - quality-gates
  - verification
requires_cli:
  - sdp
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# review

## Purpose

Is this good enough? Engineering review across multiple dimensions.
Absorbs: @review (all roles), @reality-check, @verify-workstream.

## When to Use

PR ready for engineering review, before merging to main, architecture/security concerns, release readiness.

## Review Dimensions

**code (default):** Correctness, style, maintainability, error handling, test coverage, docs.
**architecture:** Design patterns, coupling/cohesion, scalability, integration points, tech debt.
**security:** Input validation, auth/authz, data handling, dependencies, secrets.
**performance:** Complexity, query efficiency, memory usage, caching, bottlenecks.
**readiness:** Quality gates, docs, migration guides, rollback plan, observability.
**reality:** Code matches docs, no hidden assumptions, no drift, tests actually test.

## Routing Rules

Dimensions based on: (1) Diff size: small (<50 lines) → code only, large → multiple.
(2) Risk profile: auth/system → security+architecture, UI → code+readiness.
(3) Explicit request: `--security`, `--arch`, etc.
(4) Context: production deployment? → readiness, new API? → security+performance.

## Input Expectations

- **Target:** PR URL, branch, or diff
- **Dimensions:** Optional — auto-detected from change type
- **Severity threshold:** Optional — default warn on low, block on critical
- **Context:** Production deployment? Breaking change? API change?

## Legacy Aliases

@review→code, @review --arch→architecture, @review --security→security, @reality-check→reality, @verify-workstream→readiness

## Severity Levels

**critical (blocking):** Security vulns, data corruption, production outage, broken core functionality.
**high (should block):** Major architecture issues, significant performance problems, missing critical docs.
**medium (warn):** Style inconsistencies, minor performance issues, documentation gaps, tech debt.
**low (nit):** Formatting, naming suggestions, non-blocking optimizations.

## Artifacts Created

**Pass/fail verdict** with findings by dimension and severity, blocking items list, non-blocking suggestions.
**For failures:** Beads issues created, specific remediation steps, re-review criteria.

## Acceptance Boundaries

NOT for: understanding (@understand), building (@build), fixing (@fix), deployment (@operate)

Quality gates: critical findings must be addressed, high findings should be addressed or documented, re-review process clear
