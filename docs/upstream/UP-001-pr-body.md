# Add generic retry budget and terminal reason contract to Task API

## Problem

Today, when a Task's Pod fails, the controller immediately sets `Phase=Failed` with no retry and no structured terminal reason. Operators and higher-level orchestrators (e.g. SDP, CI pipelines) need:

1. **Bounded retries** — Transient infrastructure failures (OOMKilled, Evicted, ImagePullBackOff) should be retried before giving up.
2. **Terminal reason taxonomy** — A normalized `code` and `message` when a Task reaches a terminal state (Completed or Failed), so consumers can programmatically decide escalation, alerting, or remediation.

## Proposed changes

### 1. Task spec (optional, additive)

```yaml
spec:
  retry:
    maxAttempts: 7      # Total attempts including first (default: 1 = no retry)
    profile: linear     # linear | exponential (backoff between attempts)
```

- **maxAttempts**: Total attempts. `1` = current behavior (no retry). Omitted = no retry.
- **profile**: Backoff schedule between attempts:
  - `linear`: Fixed 30s delay between attempts
  - `exponential`: 2^n * 5s (5s, 10s, 20s, 40s, ...)

### 2. Task status (additive)

```yaml
status:
  terminalReason:
    code: InfrastructureError    # Normalized taxonomy
    message: "Pod OOMKilled"   # Human-readable
  retryAttempt: 3              # Current attempt (1-based)
```

**terminalReason** — Set when `Phase=Completed` or `Phase=Failed`:
- `code`: One of `Success`, `InfrastructureError`, `AgentExitNonZero`, `Timeout`, `UserStopped`, `RetryExhausted`, etc.
- `message`: From Pod/Condition or derived

**retryAttempt** — Current attempt number (1 = first run). Only present when `spec.retry` is set.

### 3. Controller behavior

- When Pod fails and `spec.retry` is set and `retryAttempt < maxAttempts`: delete Pod, increment `retryAttempt`, requeue with backoff, create new Pod.
- When retries exhausted or `spec.retry` not set: set `Phase=Failed`, populate `terminalReason`.
- When Pod succeeds: set `Phase=Completed`, `terminalReason.code=Success`.

### 4. Retriable vs non-retriable

**Retriable by default:** Pod phase `Failed` with container reason in `OOMKilled`, `Evicted`, `Error` (infrastructure).  
**Non-retriable:** Agent exit code != 0 (business logic failure). Configurable via `retriableReasons` (future) if needed.

## Backward compatibility

- **Old manifests:** Valid. Omission of `spec.retry` = no retry, behavior unchanged.
- **Default:** When `retry` omitted, `maxAttempts` effectively 1.
- **Status:** New fields are additive. Old clients ignore `terminalReason` and `retryAttempt`.

## Upgrade / migration

- No CRD migration required.
- No breaking changes to existing Task behavior.
- Opt-in: set `spec.retry` to enable retries.

## Test coverage

- [ ] Unit: retry exhaustion sets `terminalReason.code=RetryExhausted`
- [ ] Unit: terminal reason populated on first-attempt failure
- [ ] Integration: Task with `maxAttempts: 2`, force Pod fail twice → Phase=Failed, terminalReason set
- [ ] Integration: Task without retry → single attempt, same as today

## References

- SDP adapter contract: `specs/runtime/kubeopencode-sdp-adapter-contract.json`
- Fit-gap: `specs/runtime/kubeopencode-sdp-fit-gap.json`
- Beads: `sdp_dev-4py`, `sdp_dev-2aq.7.2`
