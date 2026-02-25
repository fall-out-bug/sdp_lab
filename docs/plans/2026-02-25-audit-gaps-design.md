# Audit Engineering Gaps — Design Document

> **Date:** 2026-02-25
> **Source:** `docs/drafts/AUDIT-ENGINEERING-GAPS.md` + external prompt reviews (pt1, pt2)
> **Scope:** sdp/ submodule (sdp-plugin Go code, schemas, prompts)
> **Branch:** `fix/audit-engineering-gaps`

---

## Summary

4-expert parallel analysis of 7 audit findings + 2 prompt review documents. All findings confirmed real. Key architectural insight: SDP's "inverted architecture" pushes enforcement to CLI/schemas/hooks — many audit criticisms about prompt engineering are misplaced but some are genuine gaps.

## Decisions

### D1: Evidence Layer — Singleton Writer + flock (Option B)

**Expert:** Distributed Systems (Kleppmann principles)

| Bug | Root Cause | Fix |
|-----|-----------|-----|
| Silent error drop (emitter.go:24-28) | `go func() { err; return }` | Add `slog.Error` |
| In-process race (writer.go + emitter.go) | `NewWriter` per call, mutex is per-instance | Singleton via `sync.Once` |
| Inter-process race | No file lock | `syscall.Flock` on `.lock` file |
| File open/close per event | `appendToFile` reopens every time | Singleton holds state in memory |

**Implementation sequence:**
1. `slog.Error` in `Emit()` goroutine (30 min)
2. Singleton Writer in `emitSync()` (1 hour)
3. `flock` in `Append()` with re-read-under-lock (2 hours)
4. Concurrency test (30 min)

**Future:** Flip `Emit()` to sync-by-default in v1.0 (breaking API change).

### D2: in-toto Schema — Pragmatic v1 (Option D)

**Expert:** Supply-Chain Security (Torres-Arias principles)

| Change | What | Why |
|--------|------|-----|
| Statement v1 | `StatementInTotoV01` → `StatementInTotoV1` | v0.1 deprecated, all tooling expects v1 |
| Predicate schema | Create missing `coding-workflow-predicate.schema.json` | goreleaser references it, doesn't exist |
| Version field | Add `version: "1.0"` to predicate | Future schema evolution |
| digestSet | Provenance hash chain → `{"sha256": "..."}` | Standard pattern, algorithm agility |
| snake_case | **Keep** — document as intentional | Convention, not requirement. lowerCamelCase is only needed for registry |

**Not doing:** Full lowerCamelCase rename (Option C). Too much churn for cosmetic gain. If needed later, `version` field enables it.

### D3: Inverted Architecture — Selective Prompt Improvements

**Expert:** AI Systems Architecture (Hightower principles)

**Audit criticisms reclassified:**

| Criticism | Belongs in | Status |
|-----------|-----------|--------|
| Chain-of-thought | PROMPT (3-5 skills only) | @review only |
| Few-shot examples | PROMPT (highest ROI) | @review, @build, @idea |
| Output validation | CLI/SCHEMA | New schemas needed |
| Hallucination detection | CLI/HOOKS | Already exists (guard, constraints) |
| Confidence thresholds | NOT NEEDED | System is binary (PASS/FAIL) |
| Retry/fallback | CLI | Already exists (orchestrator loop) |
| Token budget | CLI | Already exists (hydrate layer) |

**Top 5 prompt improvements:**
1. Few-shot examples for judgment skills (50 LOC/skill)
2. Explicit artifact contracts in each skill
3. Structured reasoning for @review subagents
4. Context hydration declarations
5. Adversarial synthesis in @review

**New schemas needed:**
- `schemas/review-verdict.schema.json` — 7 required reviewer roles
- `schemas/ws-verdict.schema.json` — quality gates, AC evidence

### D4: Go Code Quality — Prioritized Fix Plan

**Expert:** Go Engineering (Cheney principles)

| Priority | Issue | Fix | Effort |
|----------|-------|-----|--------|
| P0 | Mock in retry.go | Interface `WorkstreamRunner`, move mock to test | 3-4h |
| P0 | Evidence silent error | slog.Error + singleton Writer | 30m + 3h |
| P1 | Context propagation | Fix verifier.go, coverage_lang.go | 2-3h |
| P1 | Coverage placeholder | Wire existing quality.Checker | 1-2h |
| P2 | Hardcoded timeouts | Config-driven + env override | 3-4h |
| P2 | fmt.Printf → slog | Evidence + executor packages only | 1 day |

---

## Beads (Issues)

| ID | Title | Priority | Depends |
|----|-------|----------|---------|
| B1 | Evidence: add slog.Error to Emit() | P0 | — |
| B2 | Evidence: singleton Writer + flock | P0 | B1 |
| B3 | Executor: replace mock with WorkstreamRunner interface | P0 | — |
| B4 | in-toto: upgrade Statement to v1 | P1 | — |
| B5 | Schema: create coding-workflow-predicate.schema.json (Statement wrapper) | P1 | B4 |
| B6 | Verifier: fix context propagation + defer-in-loop | P1 | — |
| B7 | Verifier: wire real coverage checking | P1 | B6 |
| B8 | Schema: create review-verdict.schema.json | P1 | — |
| B9 | Schema: create ws-verdict.schema.json | P1 | — |
| B10 | Prompts: few-shot examples for @review, @build, @idea | P1 | B8, B9 |
| B11 | Prompts: structured reasoning + adversarial synthesis for @review | P1 | B10 |
| B12 | Config: configurable timeouts with env override | P2 | — |
| B13 | Logging: slog migration for evidence + executor packages | P2 | B1 |

**Total estimated effort: ~4 working days**

---

## Implementation Order

**Phase 1 — Credibility Fixes (P0):** B1, B2, B3 — 1 day
**Phase 2 — Standards + Contracts (P1):** B4, B5, B6, B7, B8, B9 — 1.5 days
**Phase 3 — Prompt Engineering (P1):** B10, B11 — 1 day
**Phase 4 — Polish (P2):** B12, B13 — 0.5 day

---

## Risks

- **flock not on Windows**: Evidence file lock requires UNIX (macOS/Linux). Windows is not supported. SDP targets macOS/Linux CI.
- **Singleton Writer stale config**: Acceptable for CLI (one invocation = one process).
- **in-toto v1 Go types are protobuf**: Keep using deprecated `StatementHeader` for JSON serialization (works). Add `nolint:staticcheck` comments.
- **Few-shot examples need maintenance**: Tie to schema versions.
- **Mock removal cascades to 7 test files**: Do as 2 commits (extract interface, then replace impl).
