# LLM Council Report: StratAudit v1.1 Implementation Plan Audit

**Rounds:** 1 of 2
**Consensus:** PARTIAL (P0 fixes supported with conditions, P1 needs refinement)
**Convergence:** 6/6 models active, 2 SUPPORT, 4 CONDITIONAL on overall plan
**Models:** codex-rescue (Architect), gemini-3.1-pro (Critic), deepseek-v3.2 (Technician), kimi-k2.5 (Philosopher), minimax-m2.7 (Pragmatist), mimo-v2-pro (Engineer)

---

## Issue Ledger

| ID | Title | Severity | Verdict | Confidence |
|----|-------|----------|---------|------------|
| I1 | FIX-01 reasoning fallback | CRITICAL | SUPPORT w/ corrections | HIGH |
| I2 | FIX-02 outputDir | HIGH | SUPPORT (6/6) | HIGH |
| I3 | FIX-03 LLM verify cost/risk | HIGH | CONDITIONAL (6/6) | HIGH |
| I4 | FIX-04 JSON export | MEDIUM | SUPPORT (6/6) | HIGH |
| I5 | FIX-05 localization scope | MEDIUM | CONDITIONAL (5/6) | MEDIUM |
| I6 | FIX-08 exec safety | HIGH | CONDITIONAL (5/6) | HIGH |
| I7 | No acceptance criteria | HIGH | SUPPORT (3/6 raised) | HIGH |
| I8 | Similarity distribution unknown | CRITICAL | SUPPORT (Philosopher+Pragmatist) | HIGH |
| I9 | Encoding detection missing | HIGH | SUPPORT (Technician) | HIGH |
| I10 | O(n^2) scaling | MEDIUM | SUPPORT (Engineer) | MEDIUM |
| I11 | Observability gap | MEDIUM | SUPPORT (Architect) | HIGH |
| I12 | LLM cache TTL/persistence | LOW | SUPPORT (Critic) | MEDIUM |

---

## RESOLVED (consensus ≥80%)

### I1: FIX-01 Reasoning fallback — SUPPORT with corrections

**Consensus:** 6/6 SUPPORT with corrections. All models agree the fix is needed.

**Accepted corrections (integrate into WS-2):**
1. **Engineer:** Use `*string` for BOTH `Content` and `Reasoning` fields (both can be JSON null)
2. **Architect:** Add minimum reasoning length (50 chars) before fallback activates
3. **Architect:** Emit `slog.Warn` every time fallback fires (model name, original content status)
4. **Critic:** Validate that `<answer>` tag extraction doesn't create injection surface
5. **Engineer:** `extractFinalAnswer()` — use last non-empty paragraph, not "last 80%"

**Action:** Update WS-2 spec with these corrections before implementing.

### I2: FIX-02 outputDir — SUPPORT (6/6)

**Consensus:** 6/6 SUPPORT. Clean fix, no corrections needed.

**Action:** Implement as specified.

### I4: FIX-04 JSON export — SUPPORT (6/6)

**Consensus:** 6/6 SUPPORT. One addition:

**Architect:** Add `"schema_version": "1.1"` top-level field in AuditReport.

**Action:** Implement as specified + schema_version.

### I7: No acceptance criteria — RESOLVED

**Action:** Define concrete targets for each WS before implementation:
- FIX-01: "extract stage completes without error on deepseek-v3.2-speciale"
- FIX-02: "reports created in --dir/.strataudit/, not CWD"
- FIX-03: "trace count >10 (vs current 1)"
- FIX-04: "report.json contains entities[] and traces[]"
- FIX-05: "findings[0].title starts with 'Разрыв' or 'Сирота'"
- FIX-08: "sdp-strataudit run on directory with .pptx doesn't fail"

---

## CONDITIONAL (needs resolution)

### I3: FIX-03 LLM verify — needs diagnostics first

**Pragmatist (minimax-m2.7):** "Do NOT implement LLM verification yet. Instead, log similarity distribution for all candidate pairs. Then decide."

**Philosopher (kimi-k2.5):** "Before adding LLM verification, diagnose the similarity distribution: mean, median, P95. If bimodal, threshold tuning solves without LLM cost."

**Architect:** "No cost/latency controls. Add `link.llm_verify_budget: int` config. Add dry-run mode."

**Technician:** "O(n^2) pairs with 30 req/min rate limit = minutes of wall clock. Needs concurrency control."

**Resolution:**
1. **Split FIX-03 into two phases:**
   - **FIX-03a (P0):** Log similarity distribution (mean, median, P95, histogram) for all candidate pairs. No LLM calls yet.
   - **FIX-03b (P1):** Based on distribution data, either adjust thresholds or implement LLM verification with budget controls.
2. Add `link.llm_verify_budget` config (max LLM calls per link pass)
3. Add semaphore for concurrency control matching `cfg.LLM.MaxConcurrent`
4. Fail-closed: LLM verification failure = pair rejected

### I5: FIX-05 localization — scope reduction

**Architect:** "Move locale maps to embedded YAML files (i18n/en.yaml, i18n/ru.yaml)"
**Philosopher:** "Localization should be semantic, not just string translation"
**Pragmatist:** "Scope to finding titles only for v1.1"

**Resolution:**
1. Keep map-based approach (not YAML files) — simpler for v1.1, can migrate later
2. Scope to title + description only (no semantic layer in v1.1)
3. Add fallback: missing key → English + log warning
4. Engineer's suggestion for BCP 47 tag: use `output.lang: "ru"` for now (not full BCP 47)

### I6: FIX-08 exec safety

**Critic:** "TOCTOU between filepath.Base check and exec call if symlinks"
**Architect:** "No timeout on exec.Command. Hung process blocks pipeline."

**Resolution:**
1. Add `context.WithTimeout` — 60s per file (configurable)
2. Startup probe: `exec.LookPath` for configured tool, warning if missing
3. Construct args as typed struct, not string concatenation
4. Resolve symlinks before Base(): `filepath.EvalSymlinks()` then `filepath.Base()`

---

## NEW ISSUES

### I8: Similarity distribution unknown (CRITICAL)

**Philosopher:** "Why did 110 entities produce only 1 trace? Before FIX-03, diagnose."
**Pragmatist:** "Log distribution before deciding on LLM verification."

**Action:** Create FIX-03a (diagnostics phase). Add to WS-4 spec.

### I9: Encoding detection (HIGH)

**Technician:** "Russian documents in Windows-1251 produce mojibake. Directly blocks FIX-05."

**Action:** Add encoding detection in `ingest.go` — `golang.org/x/net/html/charset` or `guessencoding`. Add to WS-7 (FIX-08) or as separate quick fix.

### I10: O(n^2) scaling (MEDIUM)

**Engineer:** "110 entities = ~5000 pairs. 1000 entities = 500K pairs. Consider HNSW index."

**Action:** Defer to v1.2. For v1.1, the corpus is small enough.

### I11: Observability gap (MEDIUM)

**Architect:** "No metrics: LLM call count, fallback trigger count, verification acceptance rate."

**Action:** Add structured logging in each fix. Not a separate WS.

### I12: LLM cache TTL (LOW)

**Critic:** "In-memory cache has no TTL, no cross-process persistence."

**Action:** Defer to v1.2.

---

## REVISED IMPLEMENTATION ORDER

```
Slice 1 (ship immediately, ~1.5h):
  FIX-01 (reasoning fallback)
  FIX-02 (outputDir)
  
Slice 2 (after Slice 1 data, ~2h):
  FIX-03a (similarity distribution logging + threshold diagnosis)
  
Slice 3 (based on Slice 2 data, ~3h):
  FIX-03b (LLM verification with budget controls)
  FIX-04 (JSON export)
  
Slice 4 (v1.1 completion, ~4h):
  FIX-05 (Russian localization, titles + descriptions)
  FIX-08 (Extractor interface + BridgeExtractor)
```

---

## Minority Reports

**Philosopher:** "The pipeline has no incremental re-run despite checkpoint infrastructure. After FIX-01, you cannot re-extract without re-ingesting 110 documents. This is a usability gap that will worsen with larger corpora."

**Pragmatist:** "FIX-04, FIX-05, FIX-08 are enhancements, not bugfixes. They should not be in the same release as P0 fixes. Consider versioning: v1.0.1 for P0 fixes, v1.1.0 for enhancements."

---

## Round Convergence

| Round | Resolved | New | Confidence Avg | Models Active |
|-------|----------|-----|----------------|---------------|
| 1     | 4/12     | 5   | HIGH           | 6/6           |
