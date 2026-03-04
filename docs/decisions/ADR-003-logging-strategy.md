# ADR-003: Unified Logging Strategy

> **Status:** Accepted
> **Date:** 2026-02-25
> **Context:** F053 Phase 4 remediation — evidence, executor, verifier, orchestrate use slog inconsistently

## Decision

Use `log/slog` as the single logging backend. Establish conventions for levels, structured fields, and when to log vs return errors.

### Levels

| Level | Use When |
|-------|----------|
| **Debug** | Development/troubleshooting; skip in production unless `SDP_DEBUG=1` |
| **Info** | Normal flow milestones (phase start, checkpoint saved, completion) |
| **Warn** | Recoverable or degraded state (shutdown signal, retry, escalation) |
| **Error** | Failure requiring attention; include `error` field |

### Structured Field Names

Use consistent key names across packages:

| Key | Type | Meaning |
|-----|------|---------|
| `error` | string | Error value (from `err.Error()`) |
| `ws_id` | string | Workstream ID (e.g. `00-053-01`) |
| `feature` | string | Feature ID (e.g. `F053`) |
| `phase` | string | Phase name (e.g. `build`, `review`) |
| `msg` | string | Human-readable message from hook/callback |
| `attempt` | int | Retry attempt number |
| `max_retries` | int | Max retry count |

### When to Log vs Return

- **Return errors** for callers to handle (validation, I/O, business logic failures)
- **Log + return** when the error is surfaced to caller but we also want observability
- **Log only** for internal state (shutdown, checkpoint saved) — no error to propagate
- **Never** use `fmt.Print`/`log.Print` in internal packages — use slog or return

### CLI Output vs Logging

- **User-facing output** (CLI results, `sdp validate`, `sdp quality`): `fmt.Println` to stdout is acceptable
- **Orchestration markers** (`CI GREEN`, `INVOKE: @build`): stdout for CI/script consumption
- **Internal library code**: slog only; no direct stdout/stderr

## Context

Evidence, executor, verifier, and orchestrate packages used slog with inconsistent:
- Key names (`err` vs `error`, `ws` vs `ws_id`)
- Level choice (Info vs Warn for similar events)
- Mix of fmt.Print and slog in same flows

## Consequences

- New code follows conventions; existing code migrated incrementally
- Reference implementation: `internal/orchestrate/` (migrated first)
- CI can filter logs by `SDP_DEBUG` for verbose output

## References

- [slog package](https://pkg.go.dev/log/slog)
- F053 workstream 00-053-41
