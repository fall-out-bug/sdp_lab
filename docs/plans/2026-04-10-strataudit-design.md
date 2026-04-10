# StratAudit — AI Strategy Audit Module: Design Specification v2

**Date:** 2026-04-10
**Status:** Post-council review — final
**Module:** `internal/strataudit`
**CLI:** `cmd/sdp-strataudit`
**Council models:** MiniMax M2.7, Kimi K2.5, Gemini 3.1 Pro, DeepSeek V3.2, Xiaomi Mimo V2 Pro

---

## Changelog (v1 → v2)

Based on LLM council review, the following changes were incorporated:

**Adopted:**
- Document chunking for large files before LLM extraction
- Prompt injection protection via XML tag boundaries and content sanitization
- LLM response caching table (`llm_cache`) for cost reduction on re-runs
- Entity provenance fields (`source_quote`, `extraction_model`, `page_number`)
- Embedding model version tracking in entities table
- Pipeline checkpoint/resume with `pipeline_state` table
- Rate limiting, concurrency semaphore, and token budget estimation
- Transaction boundaries for pipeline stages
- Additional composite indexes for common queries
- SQLite WAL mode enabled by default
- Paginated queries for large result sets (`EntityPage`, `TracePage`)
- LLM call audit table (`llm_invocations`) for budget tracking
- Configurable entity types in YAML
- Graceful shutdown with context.Context throughout
- Per-prompt temperature overrides
- Store all candidate pairs (not just above-threshold) for learning

**Deferred to v2:**
- REST API, webhook triggers, human-in-the-loop verification UI
- PostgreSQL/pgvector migration path
- Full delta comparison between audit runs
- Event-driven architecture redesign
- PII detection/redaction pipeline
- SQLCipher encryption at rest

**Rejected for v1 (over-engineering):**
- Full document versioning history table (version column suffices)
- Temporal entity tracking (`valid_from`/`valid_to`)
- Custom ontology DSL
- Local LLM/Ollama fallback
- Collaboration features (comments, approvals)

---

## 1. Problem Statement

Organizations produce strategy documents at multiple levels (vision, strategy, plans, initiatives, tasks), but no tool automatically verifies alignment between these levels. Strategic objectives lack operational support, operational work runs without strategic justification, and contradictions go undetected. Current solutions (Jira Align, Workboard, Quantive) require manual linking and provide no automated gap/conflict detection.

**StratAudit** solves this by ingesting strategy documents at any hierarchy level, extracting strategic entities via LLM, building a traceability graph in SQLite, and producing a comprehensive audit of alignment — including reverse inference of strategy from operational work when top-level documents are missing.

**Key capabilities not found in any existing tool:**
- Automated gap detection (strategic goals with no operational support)
- Automated orphan detection (work disconnected from strategy)
- Unknown rationale detection (work with unclear purpose)
- Reverse strategy inference (what strategy is being executed de-facto)
- Shadow strategy detection (unformalized goals driving actual work)
- Conflict detection (contradictions between levels)

---

## 2. Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Architecture | Built-in Go module | Maximum SDP kernel integration, no external dependencies |
| Storage | SQLite (relational tables, WAL mode) | Self-contained, one file per project, no external DB, SQL analytics |
| Data sources (v1) | Local files only | MVP scope: PDF, DOCX, Markdown, TXT |
| Output formats | HTML (Faust design), JSON/YAML, Mermaid, TUI | Full coverage of consumption patterns |
| Strategy levels | Configurable via YAML | Works in any project, any hierarchy model |
| LLM calls | Through modelgateway | Same interface as other SDP modules |
| Embeddings | Via modelgateway API, stored as float32 BLOB with dimension header | No local ML models, keeps binary pure-Go |
| SQLite driver | modernc.org/sqlite (pure Go) | No CGO dependency |
| Link algorithm | Brute-force cosine similarity + goroutine parallelism (v1), ANN (v2) | Acceptable up to ~5k entities, documented upgrade path |
| Document chunking | Sliding window (3000 tokens, 500 overlap) | Handles documents exceeding LLM context window |

---

## 3. Module Structure

```
internal/strataudit/
├── config.go              # StratAuditConfig, LevelConfig, YAML loading
├── pipeline.go             # Pipeline orchestrator with checkpoint/resume
├── store.go                # Store interface + SQLite implementation (WAL mode)
├── ingest.go               # File walker, level classifier, document loader
├── extract.go              # LLM entity extraction per document (with chunking)
├── link.go                 # Embedding similarity + LLM verification
├── analyze.go              # Gap/orphan/conflict/reverse inference engine
├── llmclient.go            # Unified LLM client with retry, rate limiting, caching
├── sanitize.go             # Content sanitization for prompt injection protection
├── report/
│   ├── html.go             # HTML report with Faust design system
│   ├── json.go             # JSON/YAML structured export
│   ├── diagram.go          # Mermaid diagram generator
│   └── tui.go              # Bubbletea TUI (minimal: level browser + findings list)
├── model/
│   ├── entity.go           # Entity, EntityType constants
│   ├── trace.go            # Trace, TraceRelation constants
│   ├── finding.go          # Finding, FindingType, Severity constants
│   └── document.go         # Document, Level types
├── extraction/
│   ├── txt.go              # .txt, .md extraction
│   ├── pdf.go              # .pdf extraction (with pdftotext fallback)
│   └── docx.go             # .docx extraction
└── prompts/
    ├── classify.tmpl       # Level classification prompt
    ├── extract.tmpl        # Entity extraction prompt
    ├── verify.tmpl         # Trace verification prompt
    ├── infer.tmpl          # Reverse strategy inference prompt
    └── query.tmpl          # RAG query prompt
```

**CLI binary:**

```
cmd/sdp-strataudit/main.go
```

Commands:
```bash
sdp-strataudit init                    # Create strataudit.yaml template
sdp-strataudit run [--dir ./docs]      # Full pipeline
sdp-strataudit ingest [--dir ./docs]   # Ingest only
sdp-strataudit trace                   # Extract + Link (without re-ingesting)
sdp-strataudit analyze                 # Analyze only (on existing traces)
sdp-strataudit report --format html|json|mermaid|tui
sdp-strataudit query "..."             # RAG query against trace model
```

---

## 4. Data Model

### 4.1 SQLite Schema

```sql
-- Enable WAL mode for concurrent reads
PRAGMA journal_mode=WAL;

CREATE TABLE levels (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    rank        INTEGER NOT NULL UNIQUE,
    description TEXT,
    patterns    TEXT,  -- JSON array of glob patterns
    config      TEXT   -- JSON additional config
);

CREATE TABLE documents (
    id               TEXT PRIMARY KEY,
    path             TEXT NOT NULL UNIQUE,
    level_id         TEXT NOT NULL REFERENCES levels(id),
    content_hash     TEXT NOT NULL,
    content          TEXT NOT NULL,
    version          INTEGER NOT NULL DEFAULT 1,
    file_modified_at DATETIME,
    metadata         TEXT,  -- JSON
    ingested_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE entities (
    id                TEXT PRIMARY KEY,
    document_id       TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    level_id          TEXT NOT NULL REFERENCES levels(id),
    type              TEXT NOT NULL,
    title             TEXT NOT NULL,
    description       TEXT,
    source_quote      TEXT,      -- Direct quote from document
    page_number       INTEGER,   -- Page/location in source
    embedding         BLOB,      -- 4-byte dim count + []float32 serialized little-endian
    embedding_model   TEXT,      -- Model used for embedding (e.g., "text-embedding-3-small")
    embedding_dims    INTEGER,   -- Dimension count for validation
    extraction_model  TEXT,      -- LLM model that extracted this entity
    metadata          TEXT,      -- JSON
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (embedding IS NULL OR embedding_dims IS NOT NULL)
);

CREATE TABLE traces (
    id               TEXT PRIMARY KEY,
    source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation         TEXT NOT NULL,
    confidence       REAL NOT NULL DEFAULT 0.0,
    justification    TEXT,
    direction        TEXT NOT NULL CHECK (direction IN ('up', 'down', 'bidirectional')),
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trace_candidates (
    id               TEXT PRIMARY KEY,
    source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    similarity       REAL NOT NULL,
    verified         BOOLEAN DEFAULT FALSE,
    trace_id         TEXT REFERENCES traces(id),
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE findings (
    id             TEXT PRIMARY KEY,
    type           TEXT NOT NULL CHECK (type IN (
        'alignment', 'strong_trace', 'coverage', 'gap', 'orphan',
        'unknown_rationale', 'ambiguous_trace', 'conflict', 'weak_link',
        'stale', 'inferred_strategy', 'shadow_strategy'
    )),
    severity       TEXT NOT NULL CHECK (severity IN ('info', 'warn', 'critical')),
    entity_ids     TEXT,  -- JSON array of entity IDs
    title          TEXT NOT NULL,
    description    TEXT,
    recommendation TEXT,
    suppressed     BOOLEAN DEFAULT FALSE,  -- User can suppress known findings
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trace_coverage (
    id               TEXT PRIMARY KEY,
    level_id         TEXT NOT NULL REFERENCES levels(id),
    total_entities   INTEGER NOT NULL DEFAULT 0,
    traced_entities  INTEGER NOT NULL DEFAULT 0,
    coverage_pct     REAL NOT NULL DEFAULT 0.0,
    computed_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pipeline_state (
    id          TEXT PRIMARY KEY,
    stage       TEXT NOT NULL,      -- 'ingest', 'extract', 'link', 'analyze', 'report'
    status      TEXT NOT NULL,      -- 'running', 'completed', 'failed'
    checkpoint  TEXT,               -- JSON: last processed item ID / batch info
    error       TEXT,
    started_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE TABLE llm_invocations (
    id           TEXT PRIMARY KEY,
    stage        TEXT NOT NULL,
    model        TEXT NOT NULL,
    prompt_hash  TEXT NOT NULL,
    tokens_in    INTEGER,
    tokens_out   INTEGER,
    cost_usd     REAL,
    duration_ms  INTEGER,
    cached       BOOLEAN DEFAULT FALSE,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE llm_cache (
    prompt_hash  TEXT PRIMARY KEY,
    model        TEXT NOT NULL,
    response     TEXT NOT NULL,
    tokens_in    INTEGER,
    tokens_out   INTEGER,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_entities_level ON entities(level_id);
CREATE INDEX idx_entities_document ON entities(document_id);
CREATE INDEX idx_entities_type_level ON entities(type, level_id);
CREATE INDEX idx_traces_source ON traces(source_entity_id);
CREATE INDEX idx_traces_target ON traces(target_entity_id);
CREATE INDEX idx_traces_direction ON traces(direction);
CREATE INDEX idx_traces_relation_confidence ON traces(relation, confidence);
CREATE INDEX idx_traces_confidence ON traces(confidence);
CREATE INDEX idx_findings_type ON findings(type);
CREATE INDEX idx_findings_severity ON findings(severity);
CREATE INDEX idx_findings_suppressed ON findings(suppressed) WHERE suppressed = FALSE;
CREATE INDEX idx_documents_ingested ON documents(ingested_at);
CREATE INDEX idx_trace_candidates_source ON trace_candidates(source_entity_id);
CREATE INDEX idx_trace_candidates_verified ON trace_candidates(verified) WHERE verified = FALSE;
CREATE INDEX idx_llm_cache_hash ON llm_cache(prompt_hash);
CREATE INDEX idx_llm_invocations_stage ON llm_invocations(stage);
```

### 4.2 Entity Types

Configurable in YAML (v1 defaults):

| Type | Description |
|------|-------------|
| `goal` | Strategic goal |
| `objective` | Operational objective |
| `kpi` | Measurable metric |
| `initiative` | Project or program |
| `task` | Specific work item |
| `principle` | Guiding principle or constraint |
| `stakeholder` | Person or group with interest |
| `capability` | Required organizational ability |

### 4.3 Trace Relations

| Relation | Direction | Meaning |
|----------|-----------|---------|
| `decomposes_into` | Down | Goal broken into sub-goals |
| `contributes_to` | Up | Entity contributes to achieving target |
| `measures` | Up | KPI measures target entity |
| `enables` | Up | Capability enables target entity |
| `conflicts_with` | Bidirectional | Contradiction between entities |
| `duplicates` | Bidirectional | Same intent expressed differently |
| `depends_on` | Up | Entity requires target for completion |

### 4.4 Finding Types

| Type | Severity Range | Meaning |
|------|---------------|---------|
| `alignment` | info | Confirmed trace both up and down |
| `strong_trace` | info | Trace with confidence > 0.85 |
| `coverage` | info/warn | Coverage metric per level |
| `gap` | warn/critical | Goal has no supporting entities below |
| `orphan` | warn | Entity has no trace upward |
| `unknown_rationale` | warn | Work exists but purpose is unclear |
| `ambiguous_trace` | warn | Multiple possible traces, none dominant |
| `conflict` | critical | Entities contradict each other |
| `weak_link` | warn | Trace with low confidence |
| `stale` | warn | Document not updated recently |
| `inferred_strategy` | info | Strategy inferred from bottom-up analysis |
| `shadow_strategy` | warn | Unformalized goals driving actual work |

---

## 5. Pipeline

### 5.1 Stage: Ingest

**Input:** Project directory with strategy documents
**Output:** `documents` table populated

1. Walk directories from `source_dirs` config (respecting `exclude` patterns)
2. Validate file paths stay within source dirs (prevent path traversal)
3. For each file, check content hash against existing `documents`:
   - **Unchanged:** skip
   - **Changed:** increment version, cascade-delete old entities/traces
   - **New:** classify and insert
4. Classify level:
   - First try glob patterns from level config
   - If no match, sample document (first 500 + middle 500 + last 500 tokens) for LLM classification
   - Store level assignment
5. Extract text content (format-specific extractors with fallback):
   - PDF: try `ledongthuc/pdf`, fallback to `pdftotext` CLI if available
   - DOCX: `nguyenthenguyen/docx`
   - TXT/MD: direct read
6. **Chunk large documents:** If content exceeds `chunk_token_limit` (default 3000 tokens), split into overlapping chunks (500 token overlap). Store chunks as separate entries linked to original document.
7. Store document with content hash, file modification time, version

### 5.2 Stage: Extract

**Input:** Documents from `documents` table
**Output:** `entities` table populated

For each document:
1. Check `llm_cache` for cached extraction (keyed by content hash + model + prompt hash). If hit, use cached result.
2. Send content to LLM with extraction prompt (using XML tag boundaries for injection protection):
   ```
   <document_content>
   {{content}}
   </document_content>
   ```
3. Use structured output mode (tool_use / JSON mode) where available. Fallback to text JSON with robust parsing.
4. Parse response with retry (up to 3 attempts with backoff):
   - Try direct JSON parse
   - Fallback: extract JSON from markdown code blocks
   - Fallback: regex extraction of entity blocks
5. Validate entity types against configured types
6. For each entity, store: title, description, `source_quote` (direct quote), `page_number` if available, `extraction_model`
7. Compute embeddings in batch (batch size configurable, default 100):
   - Input: title + description concatenated
   - Store with dimension header (4-byte int) + float32 array
   - Tag with `embedding_model` and `embedding_dims`
8. Log invocation to `llm_invocations` (tokens, cost, duration)

### 5.3 Stage: Link

**Input:** Entities from `entities` table
**Output:** `trace_candidates` and `traces` tables populated

**Phase 1 — Candidate Generation (parallelized):**
- For each pair of adjacent levels (rank N and rank N+1):
  - Load embeddings for both levels into memory
  - Compute cosine similarity using goroutine pool (configurable concurrency, default 8)
  - Store ALL candidates in `trace_candidates` (similarity score, verified=false)
  - Flag candidates above `thresholds.similarity` (default 0.5) for verification
- Also generate skip-level candidates (rank N and rank N+2) with threshold × 1.5

**Phase 2 — LLM Verification (batched, rate-limited):**
- Process candidates in batches (default 20 per batch)
- Rate limit: configurable `requests_per_minute` (default 30), `max_concurrent` (default 5)
- For each candidate, check `llm_cache` first
- Call LLM with verification prompt (XML boundaries, sanitized content)
- LLM returns: `{relation, confidence, justification}`
- Use categorical confidence (`HIGH`/`MEDIUM`/`LOW`) → map to (0.9/0.7/0.4)
- Store verified traces where mapped confidence >= `thresholds.trace_confidence` (default 0.6)
- Mark candidates as verified, link to trace_id
- Log all invocations to `llm_invocations`

**Scalability note:** Brute-force O(n²) is acceptable up to ~5,000 entities per level pair (~25M comparisons at ~100ns each in Go ≈ 2.5 seconds). For larger corpora, v2 will add ANN indexing.

### 5.4 Stage: Analyze

**Input:** Traces and entities
**Output:** `findings` and `trace_coverage` tables populated

All queries within a single SQLite transaction. Sequential analysis:

1. **Gap detection:** For each entity at level N, check if any trace exists with direction "down" from it. If not → `gap` finding. Severity: `critical` if rank <= 1, `warn` otherwise.

2. **Orphan detection:** For each entity at level N > 0, check if any trace exists with direction "up" from it. If not → `orphan` finding.

3. **Unknown rationale:** For each orphan entity at level N > 0, send to LLM with context to determine if purpose can be inferred. If not → `unknown_rationale` finding. Skip if entity count > threshold (batch reasoning).

4. **Ambiguous trace:** For each entity with multiple upward traces where top-2 confidence difference < 0.15 → `ambiguous_trace` finding.

5. **Conflict detection:** For each pair of entities linked to the same parent where one trace has `conflicts_with` relation → `conflict` finding with severity `critical`. Also: use LLM to detect conflicts between entities at same level sharing a parent (not caught by embedding similarity).

6. **Weak link:** Traces with confidence < `thresholds.trace_confidence` + 0.1 → `weak_link` finding.

7. **Stale documents:** Documents not modified in > `thresholds.stale_days` → `stale` finding.

8. **Coverage computation:** For each level, calculate `(traced_entities / total_entities) * 100`. Store in `trace_coverage`. Flag as `coverage` finding if below `thresholds.coverage_warn`.

9. **Reverse inference:** If level 0 or 1 has no documents, or if coverage is below 50%, run reverse inference:
   - Collect all entities from bottom levels (sample if > 500)
   - Send to LLM with inference prompt (XML boundaries)
   - LLM returns: `{inferred_vision, inferred_goals, shadow_themes, missing_from_strategy}`
   - Store as `inferred_strategy` and `shadow_strategy` findings
   - These findings are marked ephemeral — regenerated each run

### 5.5 Stage: Report

**Input:** All tables
**Output:** Generated reports in `.strataudit/`

Four outputs (HTML and JSON always generated; Mermaid and TUI opt-in):

1. **HTML (Faust design):** Self-contained HTML with:
   - Geologica Variable font (Google Fonts CDN, with system-ui fallback)
   - Dark-first with light toggle
   - Faust design tokens (`var(--cw-*)`)
   - Coverage cards with progress bars
   - Alignment chains (expandable)
   - Findings with severity color coding and type filters
   - SVG trace graph
   - Responsive layout (mobile → desktop)
   - Inline JS for filtering, theme toggle, graph interaction
   - All dynamic content escaped via `html/template`

2. **JSON/YAML:** Complete model export with metadata, levels, entities, traces, findings, coverage.

3. **Mermaid diagrams:** Trace graph, coverage heatmap, alignment chains.

4. **TUI (minimal):** Level browser with coverage, findings list with severity filter, basic entity detail view. No graph visualization in TUI.

---

## 6. LLM Prompts

All prompts are Go templates embedded via `//go:embed prompts/*.tmpl`.

### 6.1 Level Classification (Ingest fallback)

```
System: You are a strategic document classifier. Determine the hierarchy level
of the document. Available levels and descriptions:
<levels>
{{.Levels | json}}
</levels>

Classify the following document:
<document filename="{{.Filename}}">
{{.Content}}
</document>

Response format (JSON):
{"level": "<level_name>", "confidence": "HIGH|MEDIUM|LOW", "reason": "..."}

IMPORTANT: Respond ONLY with the JSON object. No other text.
```

### 6.2 Entity Extraction (Extract)

```
System: Extract all strategic entities from the document.
Entity types: {{.EntityTypes}}

For each entity specify:
- type: entity type (must be one of the listed types)
- title: short name
- description: description (up to 200 words)
- source_quote: exact quote from the document (verbatim)
- page_number: page number if available
- relations: relations to other entities INSIDE this document
  [{target_title, relation_type}]

Extract from:
<document level="{{.Level}}">
{{.Content}}
</document>

Response format (JSON array):
[{"type": "goal", "title": "...", "description": "...", "source_quote": "...",
  "page_number": null, "relations": [{"target_title": "...", "relation": "contributes_to"}]}]

IMPORTANT: Respond ONLY with the JSON array. No markdown, no explanations.
```

### 6.3 Trace Verification (Link)

```
System: Determine if there is a strategic relationship between two entities
from different hierarchy levels.

Source level: {{.SourceLevel}} (rank {{.SourceRank}})
Target level: {{.TargetLevel}} (rank {{.TargetRank}})

<source_entity>
[{{.SourceType}}] {{.SourceTitle}}: {{.SourceDescription}}
</source_entity>

<target_entity>
[{{.TargetType}}] {{.TargetTitle}}: {{.TargetDescription}}
</target_entity>

Relation types: decomposes_into, contributes_to, measures, enables,
conflicts_with, duplicates, depends_on, none

Response format (JSON):
{"relation": "<type>", "confidence": "HIGH|MEDIUM|LOW",
 "justification": "why this relation exists (or not)"}

IMPORTANT: Respond ONLY with the JSON object.
```

### 6.4 Reverse Strategy Inference (Analyze)

```
System: Based on lower hierarchy levels (projects and tasks), determine
what strategy is being executed de-facto. Formulate:
1. inferred_vision — what vision is visible from the work
2. inferred_goals — what goals are visible from tasks (3-7 items)
3. shadow_themes — recurring themes without formal declaration
4. gaps — what explicit strategies/goals are ABSENT in upper documents

Analyze:
<entities_by_level>
{{.EntitiesGroupedByLevel}}
</entities_by_level>

Response format (JSON):
{"inferred_vision": "...", "inferred_goals": [...],
 "shadow_themes": [...], "missing_from_strategy": [...]}

IMPORTANT: Respond ONLY with the JSON object.
```

### 6.5 RAG Query

```
System: You are a strategic traceability analyst. Answer questions
based ONLY on the trace model data. Support every claim with a reference
to a specific entity or trace.

Trace context:
Levels: {{.Levels}}
Coverage: {{.CoverageSummary}}

<retrieved_entities>
{{.RetrievedEntities}}
</retrieved_entities>

<related_traces>
{{.RetrievedTraces}}
</related_traces>

<relevant_findings>
{{.RelevantFindings}}
</relevant_findings>

User question: {{.Query}}

Rules:
- If data is insufficient, say so honestly — do not fabricate
- Indicate hierarchy level for each entity
- Show trace chain: "Task X → Initiative Y → Goal Z"
- Mention findings (gap, conflict, orphan) when relevant
- If reverse inference is needed, explain the logic
```

---

## 7. Configuration

```yaml
# strataudit.yaml
version: "1"

project:
  name: "Project Name"
  description: "Strategy traceability audit"

  source_dirs:
    - docs/strategy
    - docs/plans
    - docs/roadmap

  exclude:
    - "*.tmp"
    - ".git/**"
    - "node_modules/**"

levels:
  - name: vision
    rank: 0
    description: "Company vision and mission"
    patterns: ["*vision*", "*mission*"]
  - name: strategy
    rank: 1
    description: "Strategic goals and directions"
    patterns: ["*strategy*", "*стратег*"]
  - name: plan
    rank: 2
    description: "Plans and roadmaps"
    patterns: ["*roadmap*", "*plan*"]
  - name: initiative
    rank: 3
    description: "Initiatives and projects"
    patterns: ["*initiative*", "*project*"]
  - name: task
    rank: 4
    description: "Tasks and backlog items"
    patterns: ["*sprint*", "*backlog*"]

# Customizable entity types (extend or override defaults)
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
  model: "anthropic/claude-sonnet-4-20250514"
  extract_model: "anthropic/claude-sonnet-4-20250514"
  embedding_model: "openai/text-embedding-3-small"
  embedding_dims: 1536
  temperature: 0.1
  max_tokens: 4096
  # Per-prompt temperature overrides
  temperatures:
    classify: 0.0
    extract: 0.1
    verify: 0.0
    infer: 0.3
    query: 0.2
  # Rate limiting
  requests_per_minute: 30
  max_concurrent: 5
  # Retry
  max_retries: 3
  retry_base_delay_ms: 1000

thresholds:
  similarity: 0.5
  trace_confidence: 0.6
  coverage_warn: 0.7
  stale_days: 90
  chunk_token_limit: 3000
  chunk_overlap_tokens: 500

output:
  dir: ".strataudit"
  formats: [html, json]
```

---

## 8. Interfaces

```go
// Store — storage abstraction with context support
type Store interface {
    // Transaction support
    BeginTx(ctx context.Context) (Store, error)
    Commit() error
    Rollback() error

    // Write operations
    SaveDocuments(ctx context.Context, docs []model.Document) error
    SaveEntities(ctx context.Context, entities []model.Entity) error
    SaveTraces(ctx context.Context, traces []model.Trace) error
    SaveFindings(ctx context.Context, findings []model.Finding) error
    SaveCoverage(ctx context.Context, coverage []model.Coverage) error
    SaveCandidates(ctx context.Context, candidates []model.Candidate) error
    DeleteEntitiesForDocument(ctx context.Context, docID string) error

    // Read operations (paginated)
    Documents(ctx context.Context, page Page) ([]model.Document, error)
    EntitiesByLevel(ctx context.Context, levelID string, page Page) ([]model.Entity, error)
    EntityByID(ctx context.Context, id string) (*model.Entity, error)
    TracesForEntity(ctx context.Context, entityID string) ([]model.Trace, error)
    FindingsByType(ctx context.Context, ft model.FindingType, page Page) ([]model.Finding, error)
    FindingsBySeverity(ctx context.Context, sev model.Severity, page Page) ([]model.Finding, error)
    CoverageByLevel(ctx context.Context) ([]model.Coverage, error)
    CandidatesUnverified(ctx context.Context, page Page) ([]model.Candidate, error)

    // Counts (for reporting without full load)
    CountEntities(ctx context.Context) (int64, error)
    CountTraces(ctx context.Context) (int64, error)
    CountFindingsByType(ctx context.Context, ft model.FindingType) (int64, error)
    CountEntitiesByLevel(ctx context.Context, levelID string) (int64, error)

    // RAG support (brute-force in-memory for v1)
    EntitiesByEmbedding(ctx context.Context, embedding []float32, topK int) ([]model.EntityScore, error)

    // Pipeline state
    SavePipelineState(ctx context.Context, state model.PipelineState) error
    LoadPipelineState(ctx context.Context, stage string) (*model.PipelineState, error)
}

// Page — pagination
type Page struct {
    Offset int
    Limit  int
}

// LLMClient — unified LLM interface with caching and rate limiting
type LLMClient interface {
    Complete(ctx context.Context, req LLMRequest) (*LLMResponse, error)
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type LLMRequest struct {
    Model       string
    System      string
    User        string
    Temperature float64
    MaxTokens   int
    JSONMode    bool
}

type LLMResponse struct {
    Content      string
    TokensIn     int
    TokensOut    int
    Cached       bool
    Model        string
}

// Extractor — LLM entity extraction
type Extractor interface {
    Extract(ctx context.Context, doc model.Document, level model.Level) ([]model.Entity, error)
}

// Linker — trace creation
type Linker interface {
    GenerateCandidates(ctx context.Context, source, target []model.Entity) ([]model.Candidate, error)
    Verify(ctx context.Context, candidates []model.Candidate) ([]model.Trace, error)
}

// Analyzer — finding detection
type Analyzer interface {
    Analyze(ctx context.Context, store Store) ([]model.Finding, error)
}

// Reporter — output generation
type Reporter interface {
    Generate(ctx context.Context, store Store, config *Config) ([]byte, error)
    Format() string
}

// Pipeline — orchestrator with checkpoint/resume
type Pipeline interface {
    Run(ctx context.Context, opts PipelineOpts) error
    Resume(ctx context.Context) error
}

// PipelineOpts — pipeline configuration
type PipelineOpts struct {
    Dir         string
    Stages      []string  // which stages to run (empty = all)
    Force       bool      // force re-run even if stage completed
}
```

---

## 9. Confidence & Hallucination Protection

LLM-generated findings carry different hallucination risks depending on type. Structural findings (`gap`, `orphan`, `stale`, `coverage`) are deterministic — they don't use LLM. But `unknown_rationale`, `inferred_strategy`, `shadow_strategy`, and `conflict` involve LLM reasoning and must be protected.

### 9.1 Five-Layer Defense Pipeline

```
LLM generates finding with evidence_quotes[]
        ↓
  [Abstention gate] → ABSTAIN? → no finding created, logged
        ↓
  [Evidence verification] → quotes found in source? → verified: true/false
        ↓
  [Structural confidence] → composite score from 4 factors
        ↓
  [Adversarial verification] → only for high-risk finding types
        ↓
  Finding stored with full confidence metadata
```

### 9.2 Layer 1: Abstention Protocol

All LLM prompts include explicit abstention option. LLM may return `ABSTAIN` instead of a judgment. This is not an error — it's an honest answer. No finding is created for ABSTAIN responses, but the attempt is logged in `llm_invocations`.

Prompts include: `"If evidence is insufficient, respond with {\"result\": \"ABSTAIN\", \"reason\": \"...\"}. This is preferred over guessing."`

### 9.3 Layer 2: Evidence Grounding

Every LLM-generated finding must carry `evidence_quotes[]` — verbatim quotes from source documents. The system verifies each quote:

```go
func verifyEvidence(quote string, sourceContent string) bool {
    // Normalize whitespace and compare
    normalized := strings.Join(strings.Fields(quote), " ")
    return strings.Contains(
        strings.Join(strings.Fields(sourceContent), " "),
        normalized,
    )
}
```

Findings with no verifiable evidence get `evidence_verified: false` and are automatically downgraded to `low` confidence tier.

### 9.4 Layer 3: Structural Confidence Score

Composite score computed from 4 factors:

```go
func computeConfidence(f Finding) float64 {
    score := 0.0

    // Factor 1: LLM self-assessment (0-30 points)
    switch f.LLMScore {
    case "HIGH":   score += 30
    case "MEDIUM": score += 20
    case "LOW":    score += 10
    case "ABSTAIN": return 0.0
    }

    // Factor 2: Evidence grounding (0-30 points)
    if f.EvidenceVerified && f.EvidenceCount > 0 {
        score += min(f.EvidenceCount * 10, 30)
    }

    // Factor 3: Support ratio (0-20 points)
    // What fraction of relevant entities support this finding
    score += f.SupportRatio * 20

    // Factor 4: Adversarial verification (0-20 points)
    switch f.CrossModelStatus {
    case "confirmed": score += 20
    case "disputed":  score += 5
    case "refuted":   score += 0
    }

    return score / 100  // normalize to 0.0-1.0
}
```

### 9.5 Layer 4: Adversarial Verification

Applied selectively — only for high-risk finding types:

| Finding type | Adversarial check? | Reason |
|---|---|---|
| `unknown_rationale` | Yes | LLM infers absence of purpose |
| `inferred_strategy` | Yes | LLM synthesizes strategy from fragments |
| `shadow_strategy` | Yes | LLM claims unformalized goals exist |
| `conflict` | Yes | LLM detects contradiction |
| `gap`, `orphan`, `stale`, `coverage` | No | Structural, deterministic |
| `alignment`, `strong_trace`, `weak_link` | Optional (`--verify-all` flag) | LLM-assessed but grounded in trace data |

**Verification prompt:**
```
System: You are verifying a strategic audit finding. Determine if it is
supported by the provided evidence.

<finding type="{{.FindingType}}" confidence="{{.LLMScore}}">
{{.FindingTitle}}: {{.FindingDescription}}
</finding>

<evidence>
{{.EvidenceQuotes}}
</evidence>

<source_entities>
{{.SupportingEntities}}
</source_entities>

Assess:
1. Is the finding directly supported by the evidence?
2. Are there counter-arguments from the same data?
3. Is this a reasonable inference or a speculation?

Response (JSON):
{"status": "confirmed|disputed|refuted",
 "counter_arguments": ["..."],
 "verifier_confidence": "HIGH|MEDIUM|LOW"}
```

### 9.6 Layer 5: Reverse Inference Constraints

Specific constraints for the highest-risk analysis (reverse strategy inference):

- Each inferred item MUST include: `supporting_entity_ids`, `evidence_quotes`, `support_count`
- System computes `support_ratio = support_count / total_analyzed_entities`
- If `support_ratio < 0.15` for any item → item dropped, not surfaced as finding
- Empty results are valid: `inferred_goals: []` means "insufficient data"
- All reverse inference findings are tagged `ephemeral: true` — regenerated each run, never persisted across audits
- Reverse inference findings always start at `medium` confidence tier maximum (capped)

### 9.7 Presentation Tiers

| confidence_score | tier | HTML visual | default behavior |
|---|---|---|---|
| ≥ 0.7 | `high` | Green, no warning | Shown as fact |
| 0.4–0.69 | `medium` | Yellow, "requires verification" badge | Shown with caveat |
| < 0.4 | `low` | Gray, "insufficient data" badge | Hidden by default, expand on click |

### 9.8 Schema Additions

```sql
ALTER TABLE findings ADD COLUMN llm_score           TEXT;      -- HIGH/MEDIUM/LOW/ABSTAIN
ALTER TABLE findings ADD COLUMN evidence_quotes      TEXT;     -- JSON array of verbatim quotes
ALTER TABLE findings ADD COLUMN evidence_verified    BOOLEAN DEFAULT FALSE;
ALTER TABLE findings ADD COLUMN evidence_count       INTEGER DEFAULT 0;
ALTER TABLE findings ADD COLUMN support_ratio        REAL DEFAULT 0.0;
ALTER TABLE findings ADD COLUMN cross_model_status   TEXT;     -- confirmed/disputed/refuted/null
ALTER TABLE findings ADD COLUMN verification_passed  BOOLEAN;
ALTER TABLE findings ADD COLUMN confidence_score     REAL DEFAULT 0.0;
ALTER TABLE findings ADD COLUMN ephemeral            BOOLEAN DEFAULT FALSE;
ALTER TABLE findings ADD COLUMN confidence_tier      TEXT GENERATED ALWAYS AS (
    CASE
        WHEN confidence_score >= 0.7 THEN 'high'
        WHEN confidence_score >= 0.4 THEN 'medium'
        ELSE 'low'
    END
) STORED;
```

### 9.9 Risk Classification by Finding Type

| Type | Risk | Uses LLM | Needs evidence | Needs adversarial | Confidence cap |
|---|---|---|---|---|---|
| `gap` | None | No | N/A | No | 1.0 |
| `orphan` | None | No | N/A | No | 1.0 |
| `stale` | None | No | N/A | No | 1.0 |
| `coverage` | None | No | N/A | No | 1.0 |
| `alignment` | Low | Partial | Yes | Optional | 1.0 |
| `strong_trace` | Low | Partial | Yes | Optional | 1.0 |
| `weak_link` | Low | Partial | Yes | Optional | 1.0 |
| `ambiguous_trace` | Low | No | N/A | No | 1.0 |
| `conflict` | Medium | Yes | Yes | Yes | 0.9 |
| `unknown_rationale` | High | Yes | Yes | Yes | 0.7 |
| `inferred_strategy` | High | Yes | Yes | Yes | 0.7 |
| `shadow_strategy` | High | Yes | Yes | Yes | 0.7 |

---

## 10. HTML Report Design

The HTML report uses the Faust Consultant Workspace design system.

**Design tokens:**
- Dark-first: `#0f0f0f` background, `#1a1a1a` elevated
- Accent: `#4760F3` primary, `#8082F4` lavender, `#A8B5FF` light
- Font: Geologica Variable (Google Fonts CDN, system-ui fallback)
- Semantic: `#FF8080` danger, `#fbbf24` warning, `#34d399` success
- Shadows: brand-tinted `rgba(71,96,243,opacity)`
- Radius: 6px/8px/16px/20px scale

**Components:**
- Header with project name, date, stats badges
- Coverage grid: cards per level with progress bars
- Stats row: key metrics (total traces, coverage %, gaps, conflicts, inferred)
- Alignment chains: expandable trace paths from vision to task
- Findings: filterable list with severity dots, type badges, descriptions, recommendations, source quotes
- Trace graph: inline SVG with nodes colored by level, edges by confidence, gap nodes dashed
- Footer with generation metadata

**Security:** All dynamic content escaped via `html/template`. No raw `template.HTML` for user-derived content. Inline CSS/JS kept minimal.

**Template:** Go `html/template` with CSS/JS in separate `//go:embed` files for maintainability. Single self-contained HTML output.

---

## 11. Security Measures

1. **Prompt injection protection:**
   - All document content wrapped in XML tags (`<document>`, `<source_entity>`)
   - Content sanitization: strip instruction-like patterns from document content
   - System prompts explicitly instruct to ignore meta-instructions in document text

2. **Path traversal prevention:**
   - Validate all file paths resolved within `source_dirs` using `filepath.Rel`
   - Reject symlinks pointing outside source directories

3. **File validation:**
   - Enforce maximum file size (configurable, default 50MB)
   - Validate file extensions against allowlist (.pdf, .docx, .md, .txt)
   - Skip files with suspicious patterns (hidden, executable)

4. **Output security:**
   - HTML reports use `html/template` auto-escaping
   - SQLite file permissions: 0600
   - `.strataudit/` added to `.gitignore`

---

## 12. Dependencies

```
# Go modules
modernc.org/sqlite                    # Pure-Go SQLite (no CGO)
gopkg.in/yaml.v3                      # Config parsing (already in project)
github.com/ledongthuc/pdf             # PDF text extraction (with pdftotext fallback)
github.com/nguyenthenguyen/docx       # DOCX text extraction
github.com/charmbracelet/bubbletea    # TUI framework
github.com/charmbracelet/lipgloss     # TUI styling
github.com/cheggaaa/lb/v3             # Progress bar
golang.org/x/time/rate                # Rate limiting

# Internal (SDP)
internal/modelgateway                 # LLM + embedding calls
internal/kernel                       # Core types, SessionState
```

---

## 13. Implementation Phases

| Phase | Scope | Key Deliverables |
|-------|-------|------------------|
| **P0** | Foundation | `model` types, `store` SQLite (WAL, transactions), `config` loader, `llmclient` (retry, cache, rate-limit), `sanitize`, `pipeline` skeleton with checkpoint/resume |
| **P1** | Data Input | `ingest` (file walker + LLM classifier + chunking), `extraction` (pdf/docx/txt + fallbacks), `extract` (LLM entity extraction with caching) |
| **P2** | Core Analysis | `link` (brute-force similarity + goroutine pool + batched LLM verify), `analyze` (all 12 finding types + reverse inference) |
| **P3** | Output | `report/html` (Faust template), `report/json` |
| **P4** | Extended Output | `report/diagram` (Mermaid), `report/tui` (minimal: levels + findings) |
| **P5** | Interactive | `query` command (RAG against trace model) |

---

## 14. Testing Strategy

- **Unit tests:** Each stage tested independently with mock Store and mock LLMClient. Deterministic test fixtures.
- **Integration tests:** Full pipeline on fixture directory (5-10 documents) with known expected findings.
- **Golden tests:** JSON report output compared against approved golden files. HTML tested structurally (CSS selectors), not byte-for-byte.
- **LLM prompt tests:** Verify structured output parsing with 10+ example LLM responses (including malformed ones).
- **Evaluation tests (separate suite):** Run with real LLM against golden data, measure precision/recall >80%. Not in CI — run manually before releases.
- **Performance benchmarks:** Cosine similarity benchmark at 100/1000/5000 entities. Pipeline end-to-end timing.

---

## 15. Future Extensions (out of scope for v1)

- Wiki/Confluence API connectors
- Jira/YouTrack/GitHub Issues connectors
- Git commit analysis (trace strategy to code)
- Delta reports (comparison between audit runs)
- Web UI (beyond static HTML)
- sqlite-vss or dedicated vector DB for ANN search
- Multi-project portfolio view
- Scheduled/CI-integrated audits
- REST API for integration
- Human-in-the-loop trace verification
- PII detection/redaction pipeline
- Local LLM fallback (Ollama)
- PostgreSQL/pgvector production path
