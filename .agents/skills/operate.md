---
name: operate
description: Deploy, monitor, and maintain systems — CI/CD, triage, and operational tasks.
version: 1.0.0
tags:
  - devops
  - deployment
  - ci-cd
requires_cli:
  - sdp
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# operate

## Purpose

Keep it running. Deployment, CI triage, monitoring, backlog planning.
Absorbs: @deploy, @ci-triage, @plan.

## When to Use

Deploying to production/staging, CI failures need investigation, system monitoring/alerts, converting insights to backlog.

## Modes

**deploy:** Release prep and execution. Pre-deploy checks, rollback plan, deployment, smoke tests, monitoring verification.
**triage:** CI failure diagnosis. Log analysis, test categorization (flaky/dep/env/code), CI health.
**plan:** Session & backlog management. Checkpoint work state, resume from checkpoint, decompose backlog, create beads issues, map dependencies. Replaces standalone @plan skill; absorbs @oneshot checkpoint/resume.

## Routing Rules

Mode based on: (1) Context: PR merged?→deploy, CI red?→triage, insights gathered?→plan.
(2) Explicit request: "Deploy"→deploy, "Triaging CI"→triage.
(3) System state: production alert?→deploy (rollback) or triage (investigate).
(4) Work completion: feature done?→plan or deploy.

## Input Expectations

**deploy:** Target environment, version/commit, release notes (optional), rollback plan (optional)
**triage:** CI failure logs/job ID, failure description, recent changes context
**plan:** Design docs, investigation findings, raw insights, existing backlog (optional), scope constraints

## Plan Mode Details

**Purpose:** Convert raw insights into structured backlog (replaces standalone @plan skill).

**Inputs:** Design docs, investigation findings, architecture decisions, team notes, tech debt discoveries, feature requests.

**Process:** Gather insights → identify work items → group into workstreams → create beads issues → assign priorities/sizes → map to SDP stages.

**Outputs:** Workstream files, beads issues with dependencies, priority-ranked backlog, size estimates.

**Not for:** Initial brainstorming (@build idea mode), understanding codebase (@understand).

## Legacy Aliases

@deploy → deploy mode, @ci-triage → triage mode, @plan → plan mode (NOT standalone planning session)

**@guard:** Pre-deployment quality gate. All tests pass, no critical security findings, docs updated, rollback plan exists. Automatic via hooks — NOT a user-facing skill.

## Artifacts Created

**deploy:** Deployment record, smoke test results, monitoring baseline, rollback plan
**triage:** Failure categorization, assigned issues (code→@fix, flaky→infrastructure), CI health metrics
**plan:** Beads issues with dependencies, workstream breakdown, priority assignments, size estimates

## Acceptance Boundaries

NOT for: understanding (@understand), building (@build), fixing code bugs (@fix), code review (@review)

Quality gates for deploy: all quality gates pass, review approved, tests passing, documentation complete, rollback plan documented

## Success Criteria

**deploy:** System deployed, smoke tests pass, monitoring confirms health
**triage:** Failures categorized, issues assigned, CI health improved/degraded identified
**plan:** Backlog structured, dependencies mapped, priorities clear
