# StratAudit — Strategy Traceability Audit

**Module:** `internal/strataudit`
**CLI:** `cmd/sdp-strataudit`
**Version:** 1.2

## Overview

StratAudit audits alignment between strategy documents at different hierarchy levels. It ingests documents, extracts strategic entities via LLM, builds a traceability graph, and produces a gap analysis report.

**Pipeline:** Ingest → Extract → Link → Analyze → Report

## Quick Start

```bash
# Set API key
export OPENROUTER_API_KEY=sk-...

# Initialize config
sdp-strataudit init --dir /path/to/project

# Edit strataudit.yaml (levels, patterns, thresholds)

# Run full audit
sdp-strataudit run --dir /path/to/project

# Resume after interruption (skips completed stages)
sdp-strataudit run --dir /path/to/project --resume
```

Output lands in `.strataudit/`: `report.html`, `report.json`, `similarity_distribution.json`, `strataudit.db`.

## Configuration

Config file: `strataudit.yaml`

```yaml
version: "1"

project:
  name: "Project Name"
  description: "Strategy traceability audit"
  source_dirs:
    - docs/strategy
    - docs/plans
  exclude:
    - "*.7z"
    - "*.png"
    - ".DS_Store"

levels:
  - name: strategy
    rank: 0
    description: "Strategy and vision"
    patterns:
      - "*стратег*"
      - "*vision*"

  - name: architecture
    rank: 1
    description: "Architecture and standards"
    patterns:
      - "*Архитекту*"
      - "*HLD*"

  - name: design
    rank: 2
    description: "Detailed design (LLD)"
    patterns:
      - "*LLD*"
      - "*ТЗ*"

  - name: implementation
    rank: 3
    description: "Implementation and API"
    patterns:
      - "*API*"
      - "*Подключ*"

entity_types:
  - goal
  - objective
  - kpi
  - initiative
  - task
  - principle
  - stakeholder
  - capability

llm:
  model: "deepseek/deepseek-v3.2"
  extract_model: "deepseek/deepseek-v3.2"
  embedding_model: "openai/text-embedding-3-small"
  embedding_dims: 1536
  temperature: 0.1
  temperatures:
    classify: 0.0
    extract: 0.1
    verify: 0.0
    infer: 0.3
  requests_per_minute: 30
  max_concurrent: 5
  max_retries: 3
  retry_base_delay_ms: 1000

thresholds:
  similarity: 0.5
  trace_confidence: 0.6
  auto_verify_similarity: 0.85
  llm_verify_budget: 50
  coverage_warn: 70
  stale_days: 90
  chunk_token_limit: 3000
  chunk_overlap_tokens: 500
  emit_distribution: true
  max_chunks_per_document: 100
  adaptive_similarity: true

output:
  dir: ".strataudit"
  formats:
    - html
    - json
  lang: "ru"

extractors:
  external_command: "textutil"    # macOS; use "libreoffice" on Linux
  extensions:
    - ".doc"
    - ".rtf"
```

## Pipeline Stages

### 1. Ingest

Walks `source_dirs`, extracts text from documents, classifies each into a level by filename glob patterns. Deterministic classification: levels sorted by rank, lowest rank wins on multi-match.

**Supported formats:** `.txt`, `.md`, `.pdf`, `.docx` (built-in), `.doc`, `.rtf` (via external bridge to `textutil`/`libreoffice`).

Documents are deduplicated by content hash. Unchanged documents are skipped on re-runs.

### 2. Extract

For each document, splits content into overlapping chunks (configurable `chunk_token_limit`/`chunk_overlap_tokens`). Large documents are sampled to `max_chunks_per_document` chunks (first + last + uniform middle).

Each chunk is sent to the LLM with an extraction prompt. The LLM returns structured JSON entities with type, title, description, and source quote.

Entities are deduplicated by (type + title) per document. Embeddings are generated in batches of 20.

### 3. Link

For each pair of adjacent levels, computes cosine similarity between all entity embeddings. Uses adaptive threshold: if <2% of pairs exceed the configured threshold, auto-lowers to `max(p95, 0.2)`.

**Two-tier verification:**
- **Auto-verified:** similarity >= `auto_verify_similarity` (default 0.85) — no LLM call needed
- **LLM-verified:** similarity 0.5–0.85 — LLM confirms relation, filtered by `trace_confidence` (default 0.6)

Similarity distribution is emitted to `similarity_distribution.json` for diagnostics.

### 4. Analyze

Generates findings from the trace graph:

| Finding Type | Severity | Meaning |
|---|---|---|
| `gap` | warn/critical | Entity has no downward trace (no supporting work) |
| `orphan` | warn | Entity has no upward trace (disconnected from strategy) |
| `coverage` | info/warn/critical | Level coverage below threshold |
| `strong_trace` | info | Trace with confidence > 0.85 |
| `ambiguous_trace` | warn | Multiple upward traces, no clear winner |
| `weak_link` | warn | Low-confidence trace |

Findings are localized to `output.lang` (default: `ru`).

### 5. Report

Generates output files in `.strataudit/`:
- **HTML** — self-contained dark-theme report with coverage cards, findings list, trace chains
- **JSON** — complete data export (entities, traces, findings, coverage, levels)

## Architecture

```
internal/strataudit/
  config.go            Configuration loading and validation (206 lines)
  pipeline.go          Pipeline orchestration with checkpoint/resume (136 lines)
  ingest.go            File walking, text extraction, level classification (334 lines)
  extract_llm.go       LLM entity extraction with chunking and sampling (274 lines)
  extract.go           PDF/DOCX text extraction stubs (44 lines)
  extract_docx.go      DOCX ZIP parsing (54 lines)
  extractor.go         Pluggable extractor registry (95 lines)
  extractor_bridge.go  External command bridge for .doc/.rtf (154 lines)
  link.go              Cosine similarity + LLM verification (559 lines)
  analyze.go           Finding detection engine (318 lines)
  llmclient.go         OpenRouter API client with rate limiting and cache (349 lines)
  store.go             SQLite persistence layer (431 lines)
  sanitize.go          Prompt injection protection (104 lines)
  report_builder.go    Report data aggregation (90 lines)
  report/
    html.go            HTML report generation (141 lines)
    json.go            JSON report generation (93 lines)
  model/
    entity.go          Entity types and struct (47 lines)
    document.go        Document, Level, Coverage, PipelineState (61 lines)
    finding.go         Finding types with confidence scoring (145 lines)
    trace.go           Trace relations and directions (33 lines)

cmd/sdp-strataudit/
  main.go              CLI entry point (135 lines)
```

**Total:** ~5,500 lines of Go.

## Data Model

SQLite tables: `levels`, `documents`, `entities`, `traces`, `findings`, `trace_coverage`, `pipeline_state`.

**Entity types:** goal, objective, kpi, initiative, task, principle, stakeholder, capability

**Trace relations:** contributes_to, decomposes_into, measures, enables, conflicts_with, duplicates

## Key Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| LLM API | OpenRouter | Model-agnostic, supports deepseek/openai/anthropic |
| Embeddings | text-embedding-3-small | Cost-effective, 1536 dims |
| Storage | SQLite (WAL mode) | Self-contained, no external DB, portable |
| Level classification | Glob patterns | Deterministic, no LLM cost, configurable |
| Chunk sampling | Uniform (first+last+middle) | Preserves document boundaries |
| Similarity | Cosine on title+description embeddings | Fast, no additional LLM calls |
| Prompt safety | Unicode normalization + XML escaping | Prevents injection via document content |

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `OPENROUTER_API_KEY` | Yes | API key for OpenRouter |

## CLI Reference

```
sdp-strataudit <command> [options]

Commands:
  init    Create strataudit.yaml template
  run     Run full audit pipeline

Options (run):
  --dir     Project root directory (default: .)
  --config  Config file name (default: strataudit.yaml)
  --resume  Skip completed stages (auto-detect from checkpoints)
```

## Performance

Benchmark on 50 documents (UBRIR dataset, ~35M chars total):

| Metric | Value |
|---|---|
| Ingest | ~30 seconds |
| Extract | ~80 minutes (LLM-bound) |
| Link | ~5 minutes |
| Analyze + Report | ~1 minute |
| Total | ~90 minutes |

Chunk sampling reduces large-document processing from ~44 min to ~3 min per document.

## Limitations

- Strategy↔Architecture linking depends on document quality: if strategy documents contain operational details rather than strategic goals, traces will be sparse
- No incremental extraction: all entities for a document are regenerated on re-run
- Embedding model must be OpenAI-compatible (via OpenRouter)
- No web UI (static HTML report only)
