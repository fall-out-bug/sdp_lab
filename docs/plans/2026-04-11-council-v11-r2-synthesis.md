# LLM Council Report: StratAudit v1.1 Plan Audit — Round 2

**Rounds:** 2 of 2
**Consensus:** REACHED (all issues resolved)
**Convergence:** 5/5 CONDITIONAL → RESOLVED
**Models:** 6/6 active

---

## RESOLVED (Round 2, consensus ≥80%)

### I3: FIX-03 split → CONFIRMED (6/6 SUPPORT)

Split into two phases:
- **FIX-03a (P0, Slice 2):** Log similarity distribution to `{output.dir}/similarity_distribution.json`
  - Schema: run_id, generated_at, level_pairs[{stats, histogram}], current_threshold, recommendation
  - Captures ALL pairwise scores, not just above-threshold
  - Inline in `LinkEntities()`, zero extra compute cost
  - Config: `link.emit_distribution: true`
- **FIX-03b (P1, Slice 3):** Based on 03a data, adjust thresholds or implement LLM verification
  - Budget: `link.llm_verify_budget: 50` (hard stop, ~$0.10 cost)
  - Fail-closed: verification failure = pair rejected
  - Semaphore matching `cfg.LLM.MaxConcurrent`

### I5: Localization scope → CONFIRMED (6/6 SUPPORT)

Map-based localization, scoped to title + description for v1.1:
- `finding.type` stays English (machine-readable)
- Fallback to English with `slog.Warn` for missing keys
- LLM extraction prompts remain English-only (mistranslation risk)
- Config: `output.lang: "ru"` (default)
- Philosopher reservation: "String substitution is not semantic localization — flag for v1.2 enrichment"

### I6: Exec safety → CONFIRMED (6/6 SUPPORT)

Three fixes for BridgeExtractor:
1. **Timeout: 180s** (not 60s — large PPTX can take 120s). Configurable via `extract.timeout_seconds`
2. **exec.CommandContext** handles process kill automatically (Go 1.20+)
3. **exec.LookPath at startup:** warn, don't fail (no PDF files = no pdftotext needed)
4. **filepath.EvalSymlinks()** before `filepath.Base()` to prevent symlink-based name spoofing

### I9: Encoding detection → CONFIRMED, standalone (6/5 SUPPORT)

- Standalone fix in `ingest.go`, NOT part of FIX-08 Extractor interface
- Library: `golang.org/x/net/html/charset`
- Apply only to `.txt/.md/.markdown` (PDF/DOCX handle encoding internally)
- `decodeBytes(data)` helper: check `utf8.Valid()` first, then charset detection
- Add to Slice 4, before FIX-05 (localization depends on correct encoding)

### Versioning → CONFIRMED (5/6 SUPPORT)

- **v1.0.1:** FIX-01, FIX-02, FIX-03a, encoding detection (bugfixes + diagnostics)
- **v1.1.0:** FIX-03b, FIX-04, FIX-05, FIX-08 (new capabilities)
- Philosopher opposed: "FIX-03a classification is blurry — diagnostic for a bug."
- Resolution: FIX-03a ships in v1.0.1 as diagnostic patch

---

## FINAL IMPLEMENTATION ORDER

```
v1.0.1 — Slice 1 (ship immediately, ~2h):
  FIX-01: Reasoning fallback (*string, min 50 chars, slog.Warn)
  FIX-02: Absolute outputDir (filepath.Abs in main.go)

v1.0.1 — Slice 2 (after Slice 1, ~2h):
  FIX-03a: Similarity distribution logging (JSON file)
  Encoding detection (standalone, golang.org/x/net/html/charset)

v1.1.0 — Slice 3 (based on Slice 2 data, ~5h):
  FIX-03b: LLM verification (budget=50, fail-closed, semaphore)
  FIX-04: Full JSON export (EntityReport, TraceReport, schema_version)

v1.1.0 — Slice 4 (v1.1 completion, ~5h):
  FIX-05: Russian localization (map-based, title+description)
  FIX-08: Extractor interface + BridgeExtractor (180s timeout, startup probe)
```

---

## ACCEPTANCE CRITERIA

| Fix | Criterion | Target |
|-----|-----------|--------|
| FIX-01 | Extract completes on reasoning models | No error with deepseek-v3.2-speciale |
| FIX-02 | Reports in correct directory | Files in --dir/.strataudit/, not CWD |
| FIX-03a | Distribution file generated | similarity_distribution.json exists with stats |
| FIX-03b | Trace count increases | >10 traces (vs current 1) |
| FIX-04 | Entities and traces in JSON | report.json contains entities[] and traces[] |
| FIX-05 | Russian findings | findings[0].title starts with "Разрыв" |
| FIX-08 | PPTX support | sdp-strataudit run on .pptx directory succeeds |

---

## DEFERRED TO v1.2

- I10: O(n^2) scaling (HNSW index)
- I11: Observability metrics
- I12: LLM cache TTL/persistence
- I8: Threshold calibration with manual labels
- Semantic localization layer (Philosopher)

---

## Round Convergence

| Round | Resolved | New | Confidence Avg | Models Active |
|-------|----------|-----|----------------|---------------|
| 1     | 4/12     | 5   | HIGH           | 6/6           |
| 2     | 12/12    | 0   | HIGH           | 6/6           |
