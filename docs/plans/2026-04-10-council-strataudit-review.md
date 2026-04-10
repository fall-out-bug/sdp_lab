# LLM Council Report: StratAudit Code Review

**Date:** 2026-04-10
**Rounds:** 1 of 2 (convergence at 80%+)
**Consensus:** REACHED
**Models:** google/gemini-3.1-pro-preview (Critic), deepseek/deepseek-v3.2-speciale (Technician, failed), qwen/qwen3.6-plus (Engineer), minimax/minimax-m2.7 (Pragmatist), codex-rescue (Architect)
**Quorum:** 4/5 active (80%)

## Issue Ledger

| ID | Title | Severity | Architect | Critic | Engineer | Pragmatist | Consensus |
|----|-------|----------|-----------|--------|----------|------------|-----------|
| I1 | LLM cache grows without bound | HIGH | SUPPORT | SUPPORT | SUPPORT | SUPPORT | RESOLVED |
| I2 | Pipeline has no resume capability | HIGH | CONDITIONAL | SUPPORT | SUPPORT | SUPPORT | DEFERRED |
| I3 | break in analyze.go skips trace variants | HIGH | SUPPORT | SUPPORT | SUPPORT | SUPPORT | RESOLVED |
| I4 | Embedding error silently ignored | MEDIUM→HIGH | SUPPORT(critical) | SUPPORT(upgrade) | SUPPORT | SUPPORT | RESOLVED |
| I5 | Store methods in wrong file | MEDIUM | CONDITIONAL(minor) | CONDITIONAL | SUPPORT | CONDITIONAL | RESOLVED(low) |
| I6 | Coverage finding ID non-deterministic | MEDIUM | CONDITIONAL(major) | SUPPORT | SUPPORT | SUPPORT | RESOLVED |
| I7 | Sanitization bypass via Unicode homoglyphs | MEDIUM | SUPPORT(major) | SUPPORT(upgrade) | SUPPORT | SUPPORT(medium) | RESOLVED |
| I8 | LLM retry config parsed but unused | MEDIUM | SUPPORT(major) | SUPPORT | SUPPORT | SUPPORT | RESOLVED |
| I9 | No SQLite busy timeout | MEDIUM | SUPPORT(major) | SUPPORT | SUPPORT | SUPPORT | RESOLVED |
| I10 | Level classification drops documents silently | LOW→MEDIUM | SUPPORT(critical) | SUPPORT(upgrade) | SUPPORT | SUPPORT(medium) | RESOLVED |
| I11 | ChunkContent overlap can infinite-loop | LOW→HIGH | SUPPORT(critical) | SUPPORT(upgrade) | CONDITIONAL | SUPPORT | RESOLVED |
| I12 | Finding types declared but not produced | LOW | CONDITIONAL(major) | CONDITIONAL | SUPPORT | CONDITIONAL | RESOLVED |
| I13 | Doc version always resets to 1 on content change | LOW | CONDITIONAL(minor) | SUPPORT(upgrade) | SUPPORT | CONDITIONAL(medium) | DEFERRED |

## New Issues Raised

| ID | Title | Severity | Raised By |
|----|-------|----------|-----------|
| N1 | Pipeline success semantics broken | CRITICAL | Architect |
| N2 | Coverage findings can overwrite each other | CRITICAL | Architect |
| N3 | Entity dedupe semantically unsafe (title-only) | MAJOR | Architect |
| N4 | Missing context propagation in pipeline stages | MEDIUM | Engineer, Pragmatist |
| N5 | No metrics/observability | LOW | Pragmatist |

## Ship Gate

### Must Fix (blocks v1)
1. I3 — remove `break`, evaluate all traces
2. I4 — embedding error → stage-fatal
3. N1 — pipeline verdict semantics
4. N2 — coverage finding key by level_id
5. I11 — validate overlap < chunkSize

### Should Fix (before production)
6. I1 — bounded LRU cache
7. I8 — retry with exponential backoff
8. I9 — busy_timeout=5000
9. I7 — NFKC normalization + Unicode-aware regex
10. I10 — document disposition metrics

### Defer
11. I2 — implement resume or remove LoadPipelineState
12. I5 — move store methods (cosmetic)
13. I12 — remove or implement missing finding types
14. I13 — fix DocumentByPath error swallowing
