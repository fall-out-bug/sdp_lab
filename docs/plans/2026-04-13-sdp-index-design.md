# SDP Index: Codebase Memory & Indexing

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build persistent, queryable codebase index so agents never rescan from scratch. Three query modes: semantic ("how does routing work"), structural ("what depends on dispatch"), navigational ("where is ExecutorBridge").

**Architecture:** tree-sitter AST + embeddings + FTS5 in single SQLite file. Hybrid search with RRF fusion. Incremental updates via content hashing.

**Tech Stack:** Go, tree-sitter (go-tree-sitter), sqlite-vec (CGO), SQLite FTS5, Ollama (Jina Code 0.5B) or Voyage Code 3 API.

**Parent design:** `2026-04-13-sdp-toolkit-vision-design.md`

---

## Problem Statement

Every agent session starts from zero. Today's workflow:
1. Agent reads CLAUDE.md (stale, manual, incomplete)
2. Spawns 3-4 Explore subagents to scan the repo (4+ minutes)
3. Still misses 60% of what's there

After `sdp index`: Agent reads `.sdp/manifest.md` (auto-generated, complete, ≤2K tokens) and queries `.sdp/index.db` for specifics. Time: 0 seconds for context, <100ms per query.

## Three Levels

```
Level 3: ORGANIZATION (future)
  catalog-info.yaml per repo → unified service graph
  Cross-repo dependencies, ownership, API contracts

Level 2: MULTI-REPO (future)
  sdp index build <repo1> <repo2> → shared index
  Cross-repo imports, shared contracts, topology

Level 1: REPOSITORY (MVP — build first)
  sdp index build . → .sdp/index.db + .sdp/manifest.md
  AST graph + embeddings + keyword + metadata
```

This document covers Level 1 only.

## Architecture

```
sdp index build .
        │
        ├─ tree-sitter AST ──→ chunks table + edges table
        │  (go-tree-sitter)     functions, types, methods,
        │  22 languages          imports, call graph
        │
        ├─ Embedding ─────────→ chunks_vec table
        │  (Jina Code 0.5B      NL descriptions of chunks
        │   via Ollama           256 dims, Matryoshka
        │   or Voyage API)
        │
        ├─ FTS5 ──────────────→ chunks_fts table
        │  (SQLite built-in)     identifiers, paths, literals
        │
        ├─ PageRank ──────────→ rank column in chunks
        │  (from edges graph)    importance by structural centrality
        │
        └─ Metadata ──────────→ modules + files tables
           (git blame,           owners, purpose, LOC
            architect output)    conventions, summary
```

## Database Schema

Single file: `.sdp/index.db`

```sql
-- Source chunks (AST-extracted semantic units)
CREATE TABLE chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL,
    symbol_name TEXT,           -- "(*Router).ServeHTTP", "Config", "ErrNotFound"
    kind TEXT NOT NULL,         -- "function", "type", "method", "const", "interface", "file"
    scope TEXT,                 -- "internal/dispatch/router.go > Router"
    language TEXT,              -- "go", "python", "typescript"
    line_start INTEGER,
    line_end INTEGER,
    content TEXT NOT NULL,      -- raw source code
    description TEXT,           -- NL summary for embedding
    pagerank REAL DEFAULT 0,   -- structural importance
    hash TEXT NOT NULL          -- SHA256 of content (for incremental updates)
);

CREATE INDEX idx_chunks_file ON chunks(file_path);
CREATE INDEX idx_chunks_kind ON chunks(kind);
CREATE INDEX idx_chunks_symbol ON chunks(symbol_name);

-- Vector index for semantic search
CREATE VIRTUAL TABLE chunks_vec USING vec0(
    chunk_id INTEGER PRIMARY KEY,
    embedding FLOAT[256]
);

-- Full-text index for keyword search
CREATE VIRTUAL TABLE chunks_fts USING fts5(
    file_path, symbol_name, content, scope,
    content='chunks', content_rowid='id'
);

-- Structural edges (dependency graph)
CREATE TABLE edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL REFERENCES chunks(id),
    target_id INTEGER NOT NULL REFERENCES chunks(id),
    relation TEXT NOT NULL,    -- "calls", "imports", "implements", "contains", "uses"
    weight REAL DEFAULT 1.0
);

CREATE INDEX idx_edges_source ON edges(source_id);
CREATE INDEX idx_edges_target ON edges(target_id);

-- File metadata (for incremental indexing)
CREATE TABLE files (
    path TEXT PRIMARY KEY,
    hash TEXT NOT NULL,        -- SHA256 of file content
    last_indexed TEXT NOT NULL, -- ISO timestamp
    language TEXT,
    loc INTEGER,
    is_test BOOLEAN DEFAULT FALSE,
    is_generated BOOLEAN DEFAULT FALSE
);

-- Module metadata (aggregated)
CREATE TABLE modules (
    name TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    purpose TEXT,             -- NL description
    owner TEXT,               -- from git blame / CODEOWNERS
    bus_factor INTEGER,       -- from metrics (if available)
    files_count INTEGER,
    loc INTEGER,
    is_hotspot BOOLEAN DEFAULT FALSE
);

-- Index metadata
CREATE TABLE meta (
    key TEXT PRIMARY KEY,
    value TEXT
);
-- Keys: "version", "repo_name", "indexed_at", "commit_hash",
--        "total_chunks", "total_files", "embedding_model",
--        "embedding_dims", "languages"
```

## Chunking Strategy (tree-sitter based)

Priority order:
1. **Functions/methods** — primary chunks. Include signature + body.
2. **Types/classes/interfaces** — with all method signatures (bodies truncated if >512 tokens).
3. **Constants/variables** — top-level only, grouped by file.
4. **File-level** — for files without parseable structure (config, scripts).

Each chunk gets context prepended:
```
// File: internal/dispatch/router.go
// Package: dispatch
// Imports: executor, orchestrate, beads
// Scope: Router > Route

func (r *Router) Route(card *Card) (*Executor, error) {
    ...
}
```

**Size limits:**
- Target: 200-500 tokens per chunk
- Max: 1024 tokens (split recursively if exceeded)
- Min: 20 tokens (skip trivial getters/setters)

**Exclusions:**
- `vendor/`, `node_modules/`, `.git/`
- Generated files (detected by `internal/architect/extract/generated.go` patterns)
- Binary files
- Files >100KB (likely generated/data)

## NL Description Generation

Key insight from Greptile research: code↔NL semantic gap is 12%. Embedding NL descriptions instead of raw code improves retrieval significantly.

**Strategy:** For each chunk, generate a one-line NL description.

**Without LLM (fast, default):**
- Function: `"{name} in {package} — {signature}"`
- Type: `"{name} — {kind} with {N} fields/methods in {package}"`
- File: `"{path} — {language}, {LOC} lines, {N} functions"`

**With LLM (optional, higher quality):**
- Batch chunks and ask LLM for one-line descriptions
- Cache descriptions in `description` column
- Only re-generate for changed chunks

## Embedding Pipeline

```
Chunk content + NL description
        │
        ├─ Without Ollama: embed NL description only
        │  (simple, fast, reasonable quality)
        │
        └─ With Ollama: embed "description\n\ncode_snippet"
           (higher quality, ~60s for 100K LOC)

Embedding model priority:
  1. Ollama + jina/jina-embeddings-v3 (local, code-optimized, 256 dims)
  2. Ollama + nomic-embed-text (local, general, 256 dims)
  3. Voyage Code 3 API (cloud, best quality, 256 dims via Matryoshka)
  4. Skip embeddings (structural + FTS only — still useful)
```

**Matryoshka truncation:** All models produce high-dim vectors. We truncate to 256 dims.
- Storage: 256 * 4 bytes = 1KB per chunk
- 10K chunks (100K LOC) = ~10MB vectors
- 100K chunks (1M LOC) = ~100MB vectors

## Three Query Modes

### 1. Semantic Query (hybrid: vec + FTS + rerank)

```bash
sdp index query "how does task routing work"
```

Algorithm:
1. Embed query with same model → query_vec
2. Vector search: top-100 from chunks_vec (cosine similarity)
3. Keyword search: top-100 from chunks_fts (BM25)
4. RRF fusion: `score = Σ(1/(60+rank_i))` across both result sets
5. Return top-10 with file paths, line numbers, snippets

```sql
-- Simplified hybrid search
WITH vec_results AS (
    SELECT chunk_id, distance FROM chunks_vec
    WHERE embedding MATCH ?query_vec
    ORDER BY distance LIMIT 100
),
fts_results AS (
    SELECT rowid FROM chunks_fts
    WHERE chunks_fts MATCH ?query_text
    ORDER BY rank LIMIT 100
)
SELECT c.file_path, c.symbol_name, c.line_start, c.description
FROM chunks c
JOIN (
    SELECT chunk_id as id, ROW_NUMBER() OVER (ORDER BY distance) as rank
    FROM vec_results
    UNION ALL
    SELECT rowid as id, ROW_NUMBER() OVER (ORDER BY rank) as rank
    FROM fts_results
) ranked ON c.id = ranked.id
GROUP BY c.id
ORDER BY SUM(1.0 / (60 + ranked.rank)) DESC
LIMIT 10;
```

### 2. Structural Query (graph traversal)

```bash
sdp index deps internal/dispatch
sdp index deps --reverse internal/executor  # what depends ON executor
```

Algorithm:
1. Find all chunks in module
2. Follow edges (imports, calls, uses) to find dependencies
3. Aggregate at module level
4. Include metadata: bus factor, hotspot status, LOC

### 3. Navigational Query (FTS)

```bash
sdp index find "ExecutorBridge"
sdp index find "func.*Route" --regex
```

Pure FTS5 search. Fast, exact. For identifiers, paths, literals.

## Manifest Generation

```bash
sdp index manifest  # regenerate .sdp/manifest.md
```

Template-driven from aggregated index data:

```markdown
# {repo_name} — {primary_language} {arch_style}

{one_paragraph_summary}

## Modules ({count})
| Module | Purpose | LOC | Owner | Bus Factor |
|--------|---------|-----|-------|------------|
{for each module, sorted by LOC desc}

## Entry Points
{list main() functions and CLI commands}

## Conventions
- {commit style}
- {test framework}
- {build system}
- {key patterns}

## Active Work
- Last commit: {date} by {author}
- Active branches: {count}
- Open issues: {count from beads if available}
```

Target: ≤2K tokens. Loaded into agent context automatically.

## Incremental Updates

```bash
sdp index refresh  # incremental, <3 seconds for 10 changed files
```

Algorithm:
1. Walk all source files, compute SHA256 hash
2. Compare with `files.hash` in index
3. For changed files:
   a. Re-parse AST, extract new chunks
   b. Delete old chunks for this file (CASCADE edges)
   c. Insert new chunks
   d. Re-embed only new/changed chunks
   e. Rebuild FTS5 for affected rows
4. Re-run PageRank on full graph (fast: ~100ms for 10K nodes)
5. Regenerate manifest.md

**Triggers:**
- `sdp index refresh` — manual
- `post-commit` hook — after every commit (via `.sdp/hooks/`)
- `post-merge` hook — after pull/merge
- `sdp index build --watch` — background daemon, checks every 5 min

## Enrichment from Other Tools

Index gets richer when other SDP tools have run:

| Source | What it adds | Where stored |
|--------|-------------|--------------|
| `@architect` output | Module purposes, C4 data, arch style | modules.purpose, meta |
| `@metrics` output | Bus factor, hotspots, ownership | modules.bus_factor, modules.owner |
| `CODEOWNERS` | File ownership patterns | modules.owner |
| `git blame` | Per-file primary author | modules.owner (fallback) |
| beads | Active work items per module | Not stored — queried live |

Enrichment is additive and optional. Index works without any of these.

## CLI Interface

```bash
# Build & maintain
sdp index build <repo-path>              # Full index (cold start)
sdp index refresh                        # Incremental update
sdp index build --watch                  # Background daemon
sdp index manifest                       # Regenerate manifest.md
sdp index stats                          # Show index statistics

# Query
sdp index query "how does routing work"  # Semantic search
sdp index find "ExecutorBridge"          # Keyword/symbol search
sdp index deps <module>                  # Dependencies
sdp index deps --reverse <module>        # Reverse dependencies
sdp index modules                        # List all modules with metadata

# Configuration
sdp index config                         # Show current config
sdp index config --embedding ollama      # Set embedding backend
sdp index config --embedding voyage      # Set embedding backend
sdp index config --embedding none        # Disable embeddings (FTS + graph only)
```

## Performance Targets

| Operation | 100K LOC | 1M LOC | Target |
|-----------|----------|--------|--------|
| Cold build (no embeddings) | <5 sec | <30 sec | Parse + FTS + graph |
| Cold build (Ollama local) | <90 sec | <15 min | + embedding generation |
| Cold build (API) | <20 sec | <3 min | + embedding generation |
| Incremental (10 files) | <3 sec | <3 sec | Always fast |
| Semantic query | <200ms | <500ms | RRF hybrid search |
| Keyword query | <50ms | <100ms | FTS5 only |
| Structural query | <100ms | <200ms | Graph traversal |
| Manifest generation | <1 sec | <2 sec | Template fill |
| Index file size | ~15 MB | ~150 MB | With 256-dim vectors |

## Go Package Structure

```
internal/index/
  ├── builder.go          # Orchestrates full and incremental builds
  ├── builder_test.go
  ├── parser.go           # tree-sitter AST → chunks + edges
  ├── parser_test.go
  ├── embedder.go         # Embedding backend interface + implementations
  ├── embedder_test.go
  ├── store.go            # SQLite schema, CRUD, queries
  ├── store_test.go
  ├── search.go           # Hybrid search (RRF), structural queries, FTS
  ├── search_test.go
  ├── pagerank.go         # PageRank on edges graph
  ├── pagerank_test.go
  ├── manifest.go         # Manifest.md generation
  ├── manifest_test.go
  ├── enricher.go         # Ingests architect/metrics/CODEOWNERS data
  ├── enricher_test.go
  ├── types.go            # Chunk, Edge, Module, SearchResult, Config
  └── languages.go        # tree-sitter language registry + query files

cmd/sdp/
  └── cmd_index.go        # CLI subcommand (~200 LOC)
```

## Dependencies

```
github.com/smacker/go-tree-sitter       # AST parsing (CGO)
github.com/asg017/sqlite-vec-go-bindings # Vector search (CGO)
github.com/mattn/go-sqlite3              # SQLite driver (CGO, already in project?)
```

Optional:
```
Ollama (external process)  # For local embeddings
Voyage API key             # For cloud embeddings
```

## Testing Strategy

1. **Parser tests:** Create source files in temp dir, verify chunks extracted correctly.
   - Go: functions, methods, interfaces, types
   - Python: functions, classes, imports
   - TypeScript: functions, classes, exports

2. **Store tests:** In-memory SQLite, test CRUD operations.

3. **Search tests:** Pre-populated index, verify:
   - Semantic query returns relevant chunks (with mock embeddings)
   - FTS query finds exact identifiers
   - Structural query follows edges correctly
   - RRF fusion ranks correctly

4. **Incremental tests:** Build index, modify files, refresh, verify only changed chunks re-indexed.

5. **Manifest tests:** Build index, generate manifest, verify format and content.

## Security

- No secrets in index (skip `.env`, `*.pem`, `credentials.*`)
- Embeddings are one-way (cannot reconstruct code from vector)
- Index is local-only (no network calls except optional embedding API)
- `.sdp/index.db` added to `.gitignore` by default (contains local state)

## Future: Level 2 (Multi-Repo)

When ready:
- `sdp index build repo1/ repo2/` → shared index with cross-repo edges
- Cross-repo imports detected via package names / API specs
- Shared modules table with repo prefix
- Query scoping: `sdp index query "auth" --repo payment-service`

## Future: Level 3 (Organization)

When ready:
- `catalog-info.yaml` in each repo → org-wide service catalog
- `sdp index org-build` → aggregated service graph
- Runtime dependencies from OpenTelemetry traces
- Team→service→repo ownership mapping
- API contract registry across repos
