# Beads Autonomy Spec (Private)

Status: draft
Scope: L3 autonomous path to PR

## 0. Planning hierarchy

Use three levels in Beads:

- `epic` - program-level outcome and governance boundary
- `feature` - deliverable slice under epic
- `task` - executable autonomous unit

Autonomous workers only claim `task` issues.

## 1. Required issue fields

For every executable task used by swarm:

- type: `task` (or `feature` for parent)
- beads status: `open|in_progress|closed`
- protocol flow state (stored in `.sdp/runs/<issue-id>.json`): `open|in_progress|review|verified|done|blocked|escalated|cancelled`
- priority: P0-P3
- labels (required):
  - `autonomy`
  - `strict-evidence`
  - `risk:<low|medium|high|critical>`
  - `lane:<commit|explore>`
  - `model:<glm-5|glm-4.7>`
- spec-id: path to planning artifact
- acceptance: machine-checkable criteria
- dependencies: explicit blockers

Model constraint:

- only `glm-5` and `glm-4.7` are allowed by policy for now
- any other model request must set state to `escalated`

## 2. Branch policy

- Branch template: `feat/<issue-id>-<slug>`
- One active branch per claimed issue
- Direct push to protected branches forbidden

## 3. State machine

`open -> in_progress -> review -> verified -> done`

Implementation note:

- Beads native status tracks coarse state (`open|in_progress|closed`).
- Fine-grained protocol flow is tracked in run packets and validated by `cmd/beads-fsm`.

Allowed side paths:

- `in_progress -> blocked`
- `review -> blocked`
- `any -> escalated`
- `blocked -> in_progress` (only when blockers closed)
- `any -> cancelled`

## 4. Transition gates

- `open -> in_progress`
  - claim lock acquired
  - branch created
  - dependencies checked
  - selected model is policy-allowed (`glm-5` or `glm-4.7`)

- `in_progress -> review`
  - local tests/lint/contract checks pass
  - no unresolved TODO markers in changed files

- `review -> verified`
  - strict evidence sections complete
  - adversarial review completed

- `verified -> done`
  - PR exists and is linked in task notes
  - trace chain is complete

## 5. Strict evidence attachment

Attach a structured bundle with keys:

- `intent`
- `plan`
- `execution`
- `verification`
- `review`
- `risk_notes`
- `boundary`
- `provenance`
- `trace`

Any missing key blocks `verified`.

Boundary and provenance are mandatory SDP invariants for autonomous runs:

- `boundary.declared`: allowed/control/forbidden path prefixes and role/lane intent
- `boundary.observed`: touched paths and out-of-boundary paths
- `boundary.compliance`: boolean verdict with reason
- `provenance`: run_id, orchestrator/runtime/model identifiers, gate results, phase+role capture, source_issue_id, contract_version/hash_algorithm/sequence/payload_digest, and hash/hash_prev linkage

## 6. Retry and failure handling

Failure classes and retry limits:

- `transient` (timeout/429/5xx):
  - max retries: 3
  - backoff: 30s -> 90s -> 180s
  - then: `escalated`
- `tool_flake` (CI/linter/test infra instability):
  - max retries: 2
  - then: `blocked` + label `infra-flaky`
- `verification_fail` (real quality gate failure):
  - max retries: 1 auto-fix attempt
  - then: `review` + `escalated`
- `policy_conflict`:
  - max retries: 0
  - immediate: `escalated`
- `security_sensitive`:
  - max retries: 0
  - immediate: `escalated`

## 7. Escalation triggers

Immediate escalation to human:

- security-sensitive file patterns touched
- policy conflicts or uncertain branch target
- repeated verification failures over retry threshold
- unexpected dependency cycles
- mismatch between beads state and git/PR state
- missing strict evidence section

## 8. Risk class and gates

Risk classes:

- `low`: docs, non-critical refactor, tests-only behavior-safe changes
- `medium`: normal product logic and CLI updates
- `high`: orchestration, backlog state behavior, evidence pipeline, git safety
- `critical`: authn/authz, secrets, policy engine, compliance controls

Required gate stack by risk:

- `low`: unit + lint + contract basic
- `medium`: `low` + coverage gate + integration smoke
- `high`: `medium` + adversarial review + trace completeness hard check
- `critical`: `high` + manual security signoff before final PR publication

## 9. Minimal operational queries

- ready autonomous tasks:
  - `bd ready` filtered by `autonomy`
- blocked tasks:
  - `bd blocked`
- stale in-progress:
  - `bd stale --days 1`

## 10. Circuit breaker policy

- Breakers are per-agent backend, never global.
- Open breaker after 3 failures in 10 minutes.
- Half-open after 5 minutes.
- On open breaker:
  - route to fallback agent, or
  - set issue to `blocked` with reason.
- Every fallback event must be recorded in `trace`.

## 11. Acceptance examples

Good acceptance criteria:

- `AC1: all unit tests in package X pass`
- `AC2: coverage in package X >= 80%`
- `AC3: protocol contract validation returns zero errors`

Bad acceptance criteria:

- `make it better`
- `looks good`
