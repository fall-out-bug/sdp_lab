# Retry and Escalation Policy (Private)

Status: baseline v1
Scope: swarm execution in L3 mode

## 1. Failure classes

- `transient`: timeout, 429, 5xx, temporary remote errors
- `tool_flake`: infra flakiness in linters/tests/runner
- `verification_fail`: deterministic quality gate failure
- `policy_conflict`: action violates or conflicts with policy
- `security_sensitive`: security-critical context requires human gate

## 2. Retry rules

- `transient`
  - retries: 3
  - backoff: 30s, 90s, 180s
  - final state: `escalated`

- `tool_flake`
  - retries: 2
  - final state: `blocked`
  - required label: `infra-flaky`

- `verification_fail`
  - retries: 1 auto-fix attempt
  - if still failing: `review` then `escalated`

- `policy_conflict`
  - retries: 0
  - immediate `escalated`

- `security_sensitive`
  - retries: 0
  - immediate `escalated`

## 3. Escalation payload requirements

When moving to `escalated`, include:

- failure class
- attempted retries
- last failing gate or command
- recommended next action
- branch and task identifiers

## 4. Circuit breaker defaults

- per-agent breaker only
- open after 3 failures in 10 minutes
- half-open after 5 minutes
- route to fallback agent if available; otherwise mark `blocked`

## 5. Model fallback policy (current restriction)

- Primary model: `glm-5`
- Fallback model: `glm-4.7`
- Fallback chain is strictly `glm-5 -> glm-4.7 -> escalated`
- No other model is allowed without explicit policy update.

## 6. Non-negotiable rule

No silent retries and no silent fallbacks.
Every retry/fallback must be captured in evidence `trace`.
