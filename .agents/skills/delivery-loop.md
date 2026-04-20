---
name: delivery-loop
description: Autonomous delivery cycle — build all WS in subagents, review, fix design gaps, repeat until APPROVED, create PR, codex review with tests, repeat until clean.
version: 1.0.0
tags:
  - delivery
  - orchestration
  - loop
requires_cli:
  - bd
  - git
  - go
compatibility:
  - claude-code
  - opencode
  - cursor
  - codex
---

# Delivery Loop

## Purpose

Run the full feature delivery cycle autonomously without user intervention.
Replaces manual "build ws в сабагентах → review → fix → build → ... → PR → codex → fix".

**Entry condition:** worktree exists, feature claimed, workstreams defined in `docs/workstreams/backlog/`.
**Exit condition:** PR merged OR PR approved + all codex findings fixed + tests green.

## Loop Structure

```
PHASE 1: BUILD LOOP
  repeat:
    1. Dispatch @build subagents (one per WS, parallel, haiku/sonnet)
    2. Dispatch @review subagent (fresh context, sonnet)
    3. If APPROVED (zero findings including P3) → break
    4. If findings exist:
       - P1/P2 → dispatch @fix subagents per finding (haiku)
       - P3 → dispatch @fix subagents per finding (haiku)
       - Design gaps (new requirements revealed) → dispatch @design → new WS files → continue loop
  until: APPROVED with zero findings

PHASE 2: PR CREATION
  1. @review --dimension impact (check blast radius before PR)
  2. If impact review passes → create PR: gh pr create
  3. Run quality gates locally: ./scripts/run_go_quality_gates.sh

PHASE 3: CODEX REVIEW LOOP
  repeat:
    1. /codex:rescue "Review PR #<N>. Steps: (1) read all changed files, (2) run ./scripts/run_go_quality_gates.sh, (3) report all failing tests and all code findings. Do not skip tests."
    2. Wait for codex output
    3. If "no findings + tests pass" → done
    4. Dispatch @fix subagents per finding (haiku/sonnet)
    5. Run quality gates locally
    6. git push
  until: codex reports zero findings AND tests pass
```

## Subagent Model Policy

| Task | Model |
|------|-------|
| @build per WS | haiku |
| @fix per finding | haiku |
| @review | sonnet |
| @design (new WS) | sonnet |
| @review --dimension impact | sonnet |
| codex:rescue | default (Codex CLI) |

## Rules

**Never skip P3.** All findings from @review block the loop, including P3 nitpicks.

**Never create PR with red tests.** `./scripts/run_go_quality_gates.sh` must be green before `gh pr create`.

**Codex must run tests.** The codex:rescue prompt MUST include: "run ./scripts/run_go_quality_gates.sh and report failures". Never send codex a code-only review prompt.

**Independent WS → parallel subagents.** Check for file overlap before dispatching parallel @build agents. Overlapping WS → sequential.

**Max parallel subagents: 5.** Queue beyond that.

## Compaction Recovery

If loop is interrupted (compaction, crash):
1. `cat .sdp/checkpoint.json` → read loop phase + last completed step
2. `bd list --status=in_progress` → verify claim
3. Check which WS are implemented: `git diff main --name-only`
4. Resume from last incomplete phase — do NOT restart from WS 1

Update `.sdp/checkpoint.json` at the start of each phase:
```json
{"skill":"delivery-loop","feature_id":"<id>","phase":1,"step":"build","ws_done":["00-FFF-01"],"ts":"<iso>"}
```

## Checkpoint Update Points

- Phase 1 start: `{"phase":1,"step":"build","ws_done":[]}`
- After each WS built: add to `ws_done`
- After @review: `{"step":"review","verdict":"<APPROVED|FINDINGS>","findings_count":<N>}`
- Phase 2 start: `{"phase":2,"step":"impact-review"}`
- After PR created: `{"phase":3,"step":"codex","pr":"<N>"}`
- After each codex cycle: `{"step":"codex","cycle":<N>,"findings":<N>}`

## Input

Called with feature ID or auto-detected from `.sdp/checkpoint.json` / `bd list --status=in_progress`.

```
@delivery-loop [feature_id]
@delivery-loop --resume   # force checkpoint recovery
```

## Output (per phase)

```
## Phase 1: Build Loop — Cycle N
WS built: 00-FFF-01, 00-FFF-02 (parallel, haiku)
@review verdict: FINDINGS (3 findings: 1xP2, 2xP3)
  P2: [file:line] description → dispatching @fix
  P3: [file:line] description → dispatching @fix

## Phase 1: Build Loop — Cycle N+1
@review verdict: APPROVED — zero findings

## Phase 2: PR
@review --impact: OK (blast radius: 2 packages, 0 exported symbols changed)
Tests: go test ./... — 47 passed, 0 failed
PR: #<N> created

## Phase 3: Codex Review — Cycle 1
Codex findings: 2 (1 test failure, 1 code issue)
  test: TestFoo panics — dispatching @fix
  code: [file:line] — dispatching @fix
Tests after fix: 47 passed, 0 failed
Push: done

## Phase 3: Codex Review — Cycle 2
Codex: zero findings, tests pass
Done. PR #<N> ready for merge.
```

## References

- `docs/reference/go-patterns.md` — Go conventions for @build subagents
- `AGENTS.md` — beads workflow, quality gates
- `.agents/skills/build.md` — build skill (worktree bootstrap)
- `.agents/skills/review.md` — review dimensions
- `scripts/run_go_quality_gates.sh` — quality gate script
