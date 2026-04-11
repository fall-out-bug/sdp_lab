# LLM Council Round 2 Synthesis: AI Architect on Apache Spark

## Council Composition
| Role | Model | Vote |
|------|-------|------|
| Critic | Gemini 2.5 Pro | **VETO** (D3 relationship IDs, D4 zero-edge clusters) |
| Technician | DeepSeek v3.2 | **CONVERGE** |
| Philosopher | DeepSeek R1 | **VETO** (container/module confusion, broken relationships) |
| Pragmatist | Grok 3 | **VETO** (import graph P1, relationship naming) |
| Architect | GPT-4.1 | **CONVERGE** |

**Consensus: 3/5 VETO, 2 CONVERGE. Average score: 3.3/5 (up from 1.45)**

## Dimension Ratings
| Dimension | R1 Avg | R2 Avg | Critic | Tech | Phil | Prag | Arch | Delta |
|-----------|--------|--------|--------|------|------|------|------|-------|
| D1 Language | 1.0 | 4.1 | 5 | 4 | 3.5 | 4 | 4 | +3.1 |
| D2 Containers | 1.4 | 3.6 | 4 | 4 | 2 | 3.5 | 4.2 | +2.2 |
| D3 Relationships | 1.0 | 2.9 | 2 | 3 | 1.5 | 3 | 4 | +1.9 |
| D4 Import Graph | 1.4 | 2.9 | — | 3 | 2 | 2.5 | 3.8 | +1.5 |
| D5 External | 3.4 | 4.5 | — | 5 | 4 | 4.5 | 4.7 | +1.1 |
| D6 Naming | 1.2 | 3.7 | — | 4 | 2.5 | 4 | 4 | +2.5 |
| D7 SQL | 1.8 | 4.3 | — | 5 | 4.5 | 3 | 5 | +2.5 |
| D8 Overall | 1.0 | 3.4 | — | 4 | 2 | 3.5 | 4.2 | +2.4 |

## Key Improvements (Council consensus)
1. **D1 Language**: 0→5 languages detected. Fallback from ExtCounts works perfectly. (5/5 Critic)
2. **D5 External**: +15 big-data signals. "Excellent big-data ecosystem detection." (5/5 Technician)
3. **D7 SQL**: 364→0 false positives. "Perfect handling of SQL test paths." (5/5 Technician)
4. **D6 Naming**: unknown→"Spark Project Parent POM". Fixed. (4/5 majority)
5. **D2 Containers**: Maven modules properly wired. 48 modules as containers.

## New Issues (Round 2)

### P1-1: Relationship IDs are opaque placeholders
- **All 5 models flagged**
- `from: "module_4"` instead of `from: "sketch"` — IDs don't match container names
- 2256 relationships all use `module_N` IDs, making them untraceable
- **Root cause**: Pipeline creates containers with `fmt.Sprintf("module_%d", idx)` but doesn't use the actual Maven module name as ID
- **Fix**: Use `filepath.Base(child)` as container ID instead of `module_N` counter

### P1-2: Import graph clusters have zero edges
- **Philosopher + Pragmatist flagged**
- All sample clusters show `internal_edges: 0, external_edges: 0`
- "Architecturally impossible for functional code" — Philosopher
- **Root cause**: JavaAdapter `javaImportPrefix` groups by 3 segments, but package directories don't map 1:1 to import prefixes, so internal/external edge classification produces zeros
- **Fix**: Debug the cluster→package→edge counting logic in `convertJavaResult`

### P1-3: N^2 relationship explosion (48 modules → 2256 edges)
- **Critic + Philosopher flagged**
- Every module connected to every other module = noise, not signal
- "The output is architecturally invalid" — Philosopher
- **Fix**: Read actual Maven `<dependency>` from each module's pom.xml to create directed, non-complete graph. Or at minimum: cap relationships to top-K by inferred weight.

### P2-1: Container/module boundary confusion
- **Philosopher flagged (severity P1)**
- Maven modules elevated to C4 Containers (should be Components)
- "Violates C4's container definition (runnable/deployable units)"
- **Counter-argument** (Architect, Technician): For multi-module Maven projects, modules ARE deployable units
- **Fix**: Add container type field (infra vs module) and consider mapping modules as components within a single application container for L1/L2 views

### P2-2: Duplicate container names
- **Critic + Pragmatist flagged**
- `spark-rm` appears twice (Dockerfile and Dockerfile.base)
- **Fix**: Disambiguate by appending source path or variant

### P2-3: Self-referencing external system
- **Philosopher flagged**
- "spark" listed as external system — but Spark IS the system being analyzed
- `"evidence": "inferred from dependency: org.spark-project.spark:unused"`
- **Fix**: Add self-reference filter — skip external systems whose name matches the project name

### P2-4: Primary language still null
- **Philosopher + Pragmatist flagged**
- `primary_language: null` despite 5 languages detected
- Scala is clearly dominant (5813 files) but not surfaced
- **Fix**: Set primary_language from highest-count language in ExtCounts fallback

### P3-1: All relationships type="sync"
- **Technician flagged**
- No differentiation between compile, runtime, test dependencies
- **Fix**: Parse Maven dependency scope and map to relationship types

## Score Progression

| Round | D1 | D2 | D3 | D4 | D5 | D6 | D7 | D8 | Avg |
|-------|----|----|----|----|----|----|----|----|-----|
| R1 | 1.0 | 1.4 | 1.0 | 1.4 | 3.4 | 1.2 | 1.8 | 1.0 | **1.45** |
| R2 | 4.1 | 3.6 | 2.9 | 2.9 | 4.5 | 3.7 | 4.3 | 3.4 | **3.30** |
| Delta | +3.1 | +2.2 | +1.9 | +1.5 | +1.1 | +2.5 | +2.5 | +2.4 | **+1.85** |

## Prioritized Fix Plan (Round 2)

| # | Fix | Effort | Impact |
|---|-----|--------|--------|
| 1 | Use module names as container IDs | ~10 lines | Fixes D3 (relationships become traceable) |
| 2 | Set primary_language from ExtCounts | ~5 lines | Fixes D1/D6 (Philosopher, Pragmatist) |
| 3 | Filter self-referencing external systems | ~5 lines | Fixes D5 false positive |
| 4 | Deduplicate container names | ~10 lines | Fixes D2 (spark-rm duplicate) |
| 5 | Debug import cluster zero-edge issue | ~30 lines | Fixes D4 |
| 6 | Cap N^2 relationships or use actual Maven deps | ~25 lines | Fixes D3 quality |
| **Total** | **~85 lines** | | |

## Council Quotes

> "The tool has made monumental progress since Round 1, moving from a state of complete failure (1.45/5) to one of partial success." — Critic (Gemini 2.5 Pro)

> "The tool has addressed all critical Round 1 issues and now provides a fundamentally sound architectural analysis." — Technician (DeepSeek v3.2)

> "The tool shows improvement in peripheral areas but regressed in core architectural modeling." — Philosopher (DeepSeek R1)

> "While the tool has improved significantly from Round 1 (1.45/5 to 3.5/5), critical issues in import graph quality prevent it from being fully actionable." — Pragmatist (Grok 3)

> "All critical architectural extraction and mapping issues are resolved. The tool now provides actionable, accurate, and scalable architectural insights." — Architect (GPT-4.1)
