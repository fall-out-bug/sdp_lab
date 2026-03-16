# Beads Findings Contract

Status: draft
Scope: typed `beads issue` contract for review, CI, `drift`, and `QA/UAT` findings in SDP workflow

Canonical references:

- `AGENTS.md`
- `docs/plans/2026-03-15-canonical-sdp-loop-and-agent-stack.md`
- `docs/BEADS_AUTONOMY_SPEC.md`

## Purpose

In SDP, findings do not live outside the main loop.

Any actionable finding from:

- review
- CI
- `drift`
- `QA/UAT`

must re-enter execution as a typed `beads issue`.

This contract defines the minimum fields and behavior for that issue.

## Contract Rule

If a finding can block, change, or reopen work on a `feature`, it must become a `beads issue`.

Free-form comments, notes, or artifacts are not enough.

## Required Fields

Every finding-backed `beads issue` must include:

- `source`
  - `review | ci | drift | qa`
- linked `feature`
- linked `workstream`
- `blocking`
  - `true | false`
- finding title
- finding description
- severity or priority
- `PR` link or artifact reference

## Canonical Beads Mapping

Typed findings should map into Beads using the following SDP-friendly layout.

### Core issue fields

- `title`
  - short finding summary
  - should identify `feature` impact first, then the finding
- `description`
  - canonical finding body
  - should contain the typed fields in readable form
- `priority`
  - derived from severity or workflow impact
- `status`
  - normal Beads lifecycle status

### Labels

Required label groups:

- finding marker
  - `review-finding` for `source = review`
  - `ci-finding` for `source = ci`
  - `drift-finding` for `source = drift`
  - `qa-finding` for `source = qa`
- linked `feature`
  - for example `F054`
- linked `workstream`
  - for example `00-054-03`
- blocking flag
  - `blocking` when `blocking = true`
  - `non-blocking` when `blocking = false`

Optional labels:

- severity label such as `P0`, `P1`, `P2`, `P3`
- source role or check name label when stable
- deduplication hash label

### Description body

The description should include, in a predictable order:

1. `source`
2. linked `feature`
3. linked `workstream`
4. `blocking`
5. severity or priority
6. summary of the finding
7. evidence or artifact reference
8. `PR` reference
9. `trace` or `drift` reference when relevant

### Notes

Beads notes are allowed for:

- callback reports
- source payload excerpts
- repeated verification history
- reopen or regression notes

But notes are not the primary typed contract.

The contract must remain reconstructible from the issue fields, labels, and canonical description.

## Strongly Recommended Fields

When available, include:

- `evidence_ref`
- `trace_ref`
- `drift_verdict`
- source run or check identifier
- deduplication key
- owner or next action hint

## Source-Specific Expectations

### `source = review`

Use for:

- reviewer findings
- code review comments that require work
- engineering review verdict gaps

Expected references:

- review artifact or PR comment link
- reviewer role when relevant

### `source = ci`

Use for:

- failing tests
- gate failures
- generated findings from CI artifacts

Expected references:

- CI run identifier
- check name or job name
- artifact path when present

### `source = drift`

Use for:

- mismatch between `feature`, `workstream`, `execution`, `evidence`, or `trace`
- merge-readiness blocks caused by unaccepted `drift`

Expected references:

- `drift` verdict
- affected `feature` or `workstream` expectation

### `source = qa`

Use for:

- `qa:fail`
- UAT behavior mismatch
- acceptance failure found after engineering gates passed

Expected references:

- `QA/UAT` artifact
- user scenario or acceptance step that failed

## Blocking Semantics

- `blocking = true`
  - the active `PR` is not clean
  - the finding must return to the ready queue and be resolved before merge-ready state
- `blocking = false`
  - the finding is tracked follow-up work
  - it does not block current merge-ready state

Rule:

- any finding that breaks acceptance, gate status, `drift`, or `QA/UAT` must be `blocking = true`

## Required Linkage

Every typed finding must link back into SDP state:

- `feature`
- `workstream`
- current `PR` or review artifact

If any of those links is missing, the finding is incomplete and should not be treated as canonical SDP state.

## Deduplication Rule

Before creating a new finding issue, SDP should check for an existing open issue with the same:

- `source`
- `feature`
- `workstream`
- equivalent finding title or deduplication key

Do not create parallel copies of the same finding unless the old one is closed and the finding truly regressed.

Preferred deduplication tuple:

- `source`
- `feature`
- `workstream`
- finding title or stable finding key

## Lifecycle Behavior

Default path:

1. finding appears in review, CI, `drift`, or `QA/UAT`
2. finding is normalized into typed fields
3. finding becomes a `beads issue`
4. orchestrator returns it to the ready graph
5. implementer resolves it
6. reviewer or `qa` confirms resolution
7. issue closes or reopens if the finding persists

## Minimal Example Shape

```yaml
title: "F054: missing ws-verdict evidence"
source: review
feature: F054
workstream: 00-054-03
blocking: true
priority: P1
description: "Review found that required ws-verdict artifacts are missing for the current workstream."
pr_url: "https://github.com/org/repo/pull/123"
evidence_ref: "docs/reviews/F054-REVIEW-SUMMARY.md"
```

## SDP Workflow Effect

This contract means:

- `@review` must emit typed findings
- `@qa` must emit typed findings on `qa:fail`
- gate failures must be mappable into typed findings
- the orchestrator must consume typed findings from the same `beads issue` graph as implementation work

That is what keeps findings inside the canonical SDP workflow instead of turning them into side-channel comments.
