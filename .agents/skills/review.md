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

## Modes

**code (default):** Correctness, style, maintainability, error handling, test coverage, docs. For standard PRs.
**architecture:** Design patterns, coupling/cohesion, scalability, integration points, tech debt. For structural changes.
**security:** Input validation, auth/authz, data handling, dependencies, secrets. For auth/system changes.
**performance:** Complexity, query efficiency, memory usage, caching, bottlenecks. For optimization work.
**readiness:** Quality gates, docs, migration guides, rollback plan, observability. For pre-release.
**reality:** Code matches docs, no hidden assumptions, no drift, tests actually test. For verification.

## Routing Rules

Mode based on: (1) Diff size: small (<50 lines) → code only, large → multiple modes.
(2) Risk profile: auth/system → security+architecture, UI → code+readiness.
(3) Explicit request: `@review --security`, `@review --arch`, `@review --mode readiness`.
(4) Context: production deployment? → readiness, new API? → security+performance.
(5) Default: code mode only unless context suggests otherwise.

**One skill, one entry point.** Modes are parameters, not separate skills.

## Input Expectations

- **Target:** PR URL, branch, or diff (required)
- **Mode:** Optional — auto-detected from change type and risk profile. Override with `--mode <name>`.
- **Severity threshold:** Optional — default warn on low/medium, block on critical/high. Override with `--severity <level>`.
- **Context:** Production deployment? Breaking change? API change? Helps auto-select modes.

## Legacy Aliases

@review → code mode, @review --arch → architecture mode, @review --security → security mode, @reality-check → reality mode, @verify-workstream → readiness mode.

All these route to ONE @review skill with mode parameters.

## Severity Levels

**critical (blocking):** Security vulns, data corruption, production outage, broken core functionality. Must fix before merge.
**high (should block):** Major architecture issues, significant performance problems, missing critical docs. Should fix or document exception.
**medium (warn):** Style inconsistencies, minor performance issues, documentation gaps, tech debt. Track in backlog if persistent.
**low (nit):** Formatting, naming suggestions, non-blocking optimizations. Optional improvements.

## Embedded Practices

**@guard:** Pre-commit quality gate runs automatically via hooks. NOT invoked manually.

## Artifacts Created

**Pass/fail verdict** with findings by mode and severity, blocking items list, non-blocking suggestions.
**For failures:** Beads issues created, specific remediation steps, re-review criteria.

## Acceptance Boundaries

NOT for: understanding (@understand), building (@build), fixing (@fix), deployment (@operate)

Quality gates: critical findings must be addressed, high findings should be addressed or documented, re-review process clear
