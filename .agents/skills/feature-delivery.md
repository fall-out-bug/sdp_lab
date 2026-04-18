---
name: feature-delivery
description: Implement one SDP leaf workstream end-to-end — AC-first, TDD, quality gates, beads close.
version: 1.0.0
tags: [delivery, tdd, workstream]
requires_cli: [go, bd, sdp-protocol-check]
compatibility: [claude-code, opencode, cursor, codex]
---

# Feature Delivery

## Purpose

Implement a single `leaf workstream` from acceptance criteria to merged, tested, documented code.
The most common SDP task pattern: claim → read AC → TDD → gates → close.

**Outcome:** all AC checkboxes satisfied, tests pass, quality gates green, beads issue closed.

## Use When

- A beads issue is ready (`bd ready`) and maps to a leaf workstream file.
- WS file exists in `docs/workstreams/backlog/00-FFF-SS.md`.
- AC checkboxes are explicit and testable.

**Do not use when:** the workstream is an `aggregate` (not a leaf), or AC is missing — shape the feature first.

## MUST DO

1. **Claim first** — `bd update <id> --claim` before any code.
2. **Read AC completely** — `bd show <id>` + WS file; list every checkbox.
3. **Read `docs/reference/go-patterns.md`** — before writing any Go code.
4. **Read related code** — all files in WS "Scope Files" section.
5. **Write tests before implementation** (TDD) — failing test first, then code.
6. **Run quality gates** — `go build ./...` + `go test ./...` + `go vet ./...`.
7. **Verify every AC checkbox** — one-line evidence per checkbox.
8. **Close beads** — `bd close <id> --reason="AC: all N checkboxes satisfied"`.

## MUST NOT DO

- Start coding before claiming the issue.
- Implement beyond the AC scope — if AC doesn't require it, don't add it.
- Skip tests to "save time".
- Close beads before all AC checkboxes are satisfied.
- Add features the workstream doesn't describe.

## Response Format

```
## Workstream
WS: 00-FFF-SS — <title>
Beads: <id>

## AC Coverage
- [x] <checkbox 1> → <evidence: test name or file:line>
- [x] <checkbox 2> → ...
- [ ] <failing checkbox if any> → <blocker>

## Implementation
Files changed:
  - internal/foo/bar.go — <what changed>
  - ...
Key design decision: <if any non-obvious choice was made>

## Quality Gates
go build ./...: OK
go test ./...:  N passed, 0 failed (coverage: X%)
go vet ./...:   OK

## Beads
bd close <id> --reason="..."
```

## References

- `docs/reference/go-patterns.md` — naming, patterns, antipatterns, file template
- `AGENTS.md` — beads workflow, quality gates, session close protocol
