# ADR-0001: Go-First Stack with Python Research Lane

Status: accepted
Date: 2026-02-19

## Context

The private autonomy system must run L3 delivery (`task -> PR`) in a stable, secure, and operable way on k8s.
We need clear language/runtime boundaries so agents do not drift into mixed-stack operational risk.

## Decision

1. `Go` is mandatory for all production execution-path components:
   - orchestration runtime
   - brain gateway
   - Beads FSM guards
   - evidence and verification gates
   - PR publication pipeline
2. `Python` is allowed only in private research lane:
   - offline analytics
   - model/prompt experiments
   - non-blocking learning analysis jobs
3. Research-lane Python must never be required to complete `task -> PR` path.

## Consequences

- Positive:
  - one production runtime standard
  - simpler k8s operations
  - better alignment with SDP Go-centric tooling
- Tradeoff:
  - some experimentation may be slower in Go
  - requires explicit handoff from research output to Go implementation

## Enforcement

- Any new production tool under this repo must be Go unless explicitly approved as research-only.
- Production critical paths cannot import or shell out to Python tools.
