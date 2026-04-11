# LLM Council Round 3 Synthesis: AI Architect on Apache Spark

## Council Composition
| Role | Model | Vote |
|------|-------|------|
| Critic | Gemini 3.1 Pro | **CONVERGE** (3.5 avg) |
| Philosopher | Claude Sonnet 4 | **VETO** (2.5 avg) — import graph P0, container/module confusion |
| Pragmatist | MiniMax M2.7 | **VETO** (2.75 avg) — graph broken, SQL missing |
| Architect | GPT-4.1 | **VETO** (3.46 avg) — import graph P1, SQL P2 |

**Consensus: 3/4 VETO, 1 CONVERGE. Average score: ~3.0/5 (regression from R2: 3.30)**

## Dimension Ratings
| Dimension | R1 Avg | R2 Avg | R3 Avg | Critic | Phil | Prag | Arch | Delta R2→R3 |
|-----------|--------|--------|--------|--------|------|------|------|-------------|
| D1 Language | 1.0 | 4.1 | 4.6 | 5 | 4 | 4 | 4.9 | +0.5 |
| D2 Containers | 1.4 | 3.6 | 3.3 | 5 | 2 | 3 | 4.2 | -0.3 |
| D3 Relationships | 1.0 | 2.9 | 3.0 | 3 | 2 | 3 | 3.5 | +0.1 |
| D4 Import Graph | 1.4 | 2.9 | 1.8 | — | 1 | 2 | 2.0 | -1.1 |
| D5 External | 3.4 | 4.5 | 4.5 | — | 4 | — | 5.0 | 0.0 |
| D6 Naming | 1.2 | 3.7 | 4.5 | — | 4 | 4 | 4.6 | +0.8 |
| D7 SQL | 1.8 | 4.3 | 2.3 | — | 3 | 2 | 2.0 | -2.0 |
| D8 Overall | 1.0 | 3.4 | 2.8 | — | 2 | 3 | 3.5 | -0.6 |

## Key Improvements (Council consensus)
1. **D1 Language**: primary_language="Scala" now correct. ExtCounts realistic.
2. **D6 Naming**: Semantic slug IDs (sketch, kvstore) work well. System name accurate.
3. **D3 Relationships**: N^2 capped to 168. Format clean.
4. **D5 External Systems**: Excellent detection (5/5 Architect). All major Spark systems found.

## Regressions (Round 3)

### P0-1: Import graph clusters ALL show zero edges (CRITICAL)
- **Philosopher + Pragmatist + Architect all flagged as P0/P1**
- 56 clusters, all `internal_edges: 0, external_edges: 0`
- 5037 total edges exist but aren't attributed to any cluster
- "Mathematically impossible" — Pragmatist
- **Root cause**: `convertJavaResult()` builds clusters but edge counting logic doesn't classify imports into internal/external per cluster
- **Fix**: Debug the cluster→package→edge attribution in `convertJavaResult()`

### P2-1: SQL tables = 0 despite 534 .sql files
- **Philosopher + Pragmatist + Architect flagged**
- Spark is a SQL engine — zero SQL detection is a major gap
- `isTestPath()` filter is too aggressive for Spark's test-heavy SQL files
- **Fix**: Relax SQL test path filter; extract table names from non-test SQL files

### P2-2: Bidirectional Maven dependencies
- **Critic + Pragmatist flagged**
- `sketch→kvstore` AND `kvstore→sketch` — Maven requires DAG
- Suggests naive relationship generation rather than actual dependency parsing
- **Fix**: Parse actual Maven `<dependency>` direction or ensure unidirectional edges

### P2-3: Generated/binary files in ext_counts
- **Critic flagged**
- .bin (728), .crc (630), .out (806), .explain (725) inflate file counts
- These are build artifacts, not source files
- **Fix**: Add build artifact filter to ext_counts

### P2-4: Temporal hallucination
- **Critic flagged**
- `ubuntu:jammy-20250819` — future date tag
- LLM-generated artifact, not tool bug per se
- **Fix**: Validate date tags against reasonable ranges

## Score Progression

| Round | D1 | D2 | D3 | D4 | D5 | D6 | D7 | D8 | Avg |
|-------|----|----|----|----|----|----|----|----|-----|
| R1 | 1.0 | 1.4 | 1.0 | 1.4 | 3.4 | 1.2 | 1.8 | 1.0 | **1.45** |
| R2 | 4.1 | 3.6 | 2.9 | 2.9 | 4.5 | 3.7 | 4.3 | 3.4 | **3.30** |
| R3 | 4.6 | 3.3 | 3.0 | 1.8 | 4.5 | 4.5 | 2.3 | 2.8 | **3.0** |
| Delta | +0.5 | -0.3 | +0.1 | -1.1 | 0.0 | +0.8 | -2.0 | -0.6 | **-0.3** |

## Prioritized Fix Plan (Round 3)

| # | Fix | Effort | Impact |
|---|-----|--------|--------|
| 1 | Debug import cluster zero-edge bug | ~40 lines | Fixes D4 (P0 blocker) |
| 2 | Relax SQL test path filter | ~15 lines | Fixes D7 (P2) |
| 3 | Make Maven relationships unidirectional | ~20 lines | Fixes D3 (P2) |
| 4 | Filter build artifacts from ext_counts | ~10 lines | Fixes data quality |
| **Total** | **~85 lines** | | |

## Council Quotes

> "The structural and schema-breaking issues from previous rounds are fixed. The remaining issues are semantic hallucinations typical of LLM-based extraction." — Critic (Gemini 3.1 Pro, CONVERGE)

> "The tool shows incremental improvements but retains fundamental architectural misunderstandings." — Philosopher (Claude Sonnet 4, VETO)

> "Score improved on formatting but regressed on correctness. D2 (Correctness) is 2/5 because the import_graph edge distribution is impossible." — Pragmatist (MiniMax M2.7, VETO)

> "Core architectural facts are now correct: modules, relationships, languages, external systems. However, import graph and SQL semantics remain lacking." — Architect (GPT-4.1, VETO)
