# Self-Improvement Protocol (Private)

Status: baseline v1
Scope: telemetry-driven feature discovery

## 1. Overview

The self-improvement agent analyzes telemetry from completed runs and creates Beads tasks for SDP improvements. It implements the loop from PRIVATE_BLUEPRINT section 9.

## 2. Data sources

- `.sdp/runs/*.json` — run traces with phase transitions and terminal reasons
- `.sdp/observability/intake.jsonl` — JSONL from internal/observability (phase, status, retries, escalation)
- Beads history — `bd list --status closed --json` for lead time and failure rate

## 3. Failure classification

Aligned with [RETRY_ESCALATION_POLICY.md](RETRY_ESCALATION_POLICY.md):

- `transient` — timeout, 429, 5xx
- `tool_flake` — infra flakiness
- `verification_fail` — quality gate failure
- `policy_conflict` — policy violation
- `security_sensitive` — security gate (blocked from auto-injection)

## 4. Weakness detection

- Repeated failures of same class across runs
- Escalation spike in telemetry window
- Boundary violations in evidence

## 5. Safety gate

- `security_sensitive` patterns are never auto-injected
- Max proposals per cycle (default 3)
- Risky proposals escalate for human approval

## 6. Created task format

Labels: `autonomy`, `strict-evidence`, `workstream:self-improvement`, `risk:medium`

## 7. Trigger

- CronJob or manual: `self-improve-agent --work-dir /workspace`
- In-cluster: Deployment in sdp-control with periodic execution

## 8. References

- [specs/self-improvement-contract.yaml](../specs/self-improvement-contract.yaml)
- [CI_LOCAL_BRIDGE.md](runbooks/CI_LOCAL_BRIDGE.md)
