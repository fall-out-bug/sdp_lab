# QA Command Full Specification

This document contains the full specification for `@qa`.

## Overview

`@qa` runs `QA/UAT` after engineering gates are clean.

Its job is to validate behavior against `feature` intent, not just technical correctness.

## Canonical Outputs

`@qa` must emit exactly one of:

- `qa:pass`
- `qa:fail`

And it must write:

- `.sdp/qa_verdict.json`

## Verdict Artifact Requirements

The QA verdict artifact must contain:

- `feature`
- `verdict`
- `iteration`
- `timestamp`

Optional but expected fields:

- `finding_ids`
- `blocking_ids`
- `summary`
- `evidence_ref`
- `pr_url`

## Typed Findings Rule

If `QA/UAT` fails, the failure must return to SDP as typed `beads issue` findings with:

- `source = qa`
- linked `feature`
- linked `workstream`
- `blocking = true`
- `PR` or UAT artifact reference when available

## UAT Evidence Rule

`qa:pass` must point to concrete `UAT evidence`.

The artifact reference can be a report, log bundle, trace, or structured test result, but it must be explicit.

## Rerun Control

If blocking findings already exist for the `feature`, orchestrator may route back to `build` before re-running `@qa`.

In that case, `.sdp/qa_verdict.json` remains the canonical place to record the blocked QA state and linked `blocking_ids`.
