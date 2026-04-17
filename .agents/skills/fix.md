---
name: fix
description: Diagnose and resolve bugs, errors, and failures — from quick hotfixes to systematic investigations.
version: 1.0.0
tags:
  - debug
  - bugfix
  - troubleshooting
requires_cli:
  - sdp
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# fix

## Purpose

Something is broken — fix it. From quick hotfixes to systematic debugging.
Absorbs: @hotfix, @bugfix, @issue, @debug.

## When to Use

Known bugs with reproduction steps, test failures, CI breaks, production incidents, error logs available.

## Modes

**quick:** Known cause, minimal change, instant PR. Output: minimal fix + regression test. For: hotfixes, trivial bugs, known workarounds.
**investigate:** Unknown cause, needs diagnosis. Output: root cause, reproduction steps, proposed fix. For: "debug this...", "why is...", unclear failures.
**systematic:** Known issue, planned fix with full investigation. Output: complete fix + tests + docs + RCA. For: tracked issues, complex bugs, production incidents.

## Routing Rules

Approach based on: (1) Severity: `--severity critical` → quick mode (fast path to resolution). (2) Known vs unknown cause. (3) Reproduction available? (4) User preference: "just fix it"→quick, "proper fix"→systematic. (5) Issue tracking: beads issue exists?→systematic.

## Input Expectations

- **Problem:** Clear description (error message, stack trace, behavior)
- **Context:** When it happens, how to reproduce, impact
- **Severity:** Optional — auto-detected (critical, production, blocking)
- **Mode:** Optional — auto-detected from available information

## Legacy Aliases

@hotfix → quick mode, @bugfix → systematic mode, @issue → systematic mode + issue tracker, @debug → investigate mode

## Embedded Practices

**@tdd:** Regression test BEFORE fix. Write failing test → verify fails → implement → verify passes → check regressions. NOT a separate skill — embedded in @fix workflow.

**@guard:** Pre-commit quality gate runs automatically via hooks. NOT invoked manually.

## Severity Levels

**critical:** Production outage, data loss, security vuln, blocking all users. Forces quick mode with minimal fix first.
**high:** Major feature broken, significant user impact, CI completely blocked. Defaults to systematic unless urgency specified.
**medium:** Single feature affected, workarounds available, partial CI failure. Mode based on context.
**low:** Minor issues, edge cases, nice-to-have. Can backlog via @operate plan mode.

**Severity as parameter:** `@fix --severity critical "production 500"`, `@fix --severity low "cosmetic issue"`

## Artifacts Created

**quick:** Minimal fix (1-5 lines), regression test, PR, brief RCA
**investigate:** Reproduction steps, root cause analysis, proposed fix approach
**systematic:** Complete fix, comprehensive tests, doc updates, RCA document, beads issue

## Acceptance Boundaries

NOT for: understanding (@understand), building features (@build), non-failure review (@review), deployment (@operate)

Quality gates: reproduction test exists, minimal (quick) or comprehensive (systematic), no regressions, docs updated
