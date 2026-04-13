# SDP Spec: Specification Recovery from Code

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract implicit specifications from code — API contracts, business rules, invariants, SLA parameters — producing structured JSON that documents what the system actually enforces, not what someone wrote in a wiki.

**Architecture:** Two-phase pipeline: deterministic AST extraction (fast, always runs) + optional LLM enrichment (slow, higher quality). Go CLI produces JSON, skill interprets into spec documents.

**Tech Stack:** Go, go/ast (for Go code), tree-sitter (for other languages), no CGO for Phase 1.

**Parent design:** `2026-04-13-sdp-toolkit-vision-design.md`

---

## Problem Statement

Specifications live in code, not in documentation. The documentation says "POST /api/users creates a user" but the code enforces:
- Email must be valid and unique (validation tag)
- Password must be 8+ chars with uppercase+digit (guard clause)
- Rate limit: 10 requests/minute (middleware config)
- Response timeout: 30s (context deadline)
- Retry: 3 attempts with exponential backoff (retry config)

These are the **real** specifications. Spec recovery extracts them automatically.

**Before:** Read 50 files to understand what the API actually does.
**After:** `sdp spec .` → structured spec document with contracts, rules, invariants, SLA.

## Four Spec Categories

### 1. API Contracts

HTTP endpoints, gRPC services, GraphQL schemas, message queue topics.

**Sources:**
| Source | Method | Example |
|--------|--------|---------|
| OpenAPI/Swagger files | Parse YAML/JSON | `openapi.yaml`, `swagger.json` |
| Route registration | AST pattern matching | `r.GET("/api/users", handler)` |
| Proto files | Parse .proto | `service UserService { rpc Create(...) }` |
| GraphQL schemas | Parse .graphql | `type Query { user(id: ID!): User }` |
| Struct tags | Reflect on `json:"name"` | Field names, omitempty, required |
| Handler signatures | AST analysis | Request/response types |

**Go-specific route detection patterns:**

```go
// chi
r.Get("/api/users/{id}", getUser)
r.Route("/api/v2", func(r chi.Router) { ... })

// gin
r.GET("/api/users/:id", getUser)
group := r.Group("/api/v2")

// echo
e.GET("/api/users/:id", getUser)
g := e.Group("/api/v2")

// net/http (stdlib)
http.HandleFunc("/api/users/", handler)
mux.Handle("/api/users", handler)

// gorilla/mux
r.HandleFunc("/api/users/{id:[0-9]+}", getUser).Methods("GET")
```

**Output per endpoint:**
```jsonc
{
  "method": "GET",
  "path": "/api/users/{id}",
  "handler": "internal/api/users.go:GetUser",
  "request": {
    "path_params": [{"name": "id", "type": "string", "pattern": "[0-9]+"}],
    "query_params": [{"name": "include", "type": "string", "required": false}],
    "headers": [{"name": "Authorization", "required": true}]
  },
  "response": {
    "type": "User",
    "fields": [
      {"name": "id", "type": "string", "json": "id"},
      {"name": "email", "type": "string", "json": "email"},
      {"name": "created_at", "type": "time.Time", "json": "created_at"}
    ]
  },
  "middleware": ["auth", "rate-limit"],
  "source_file": "internal/api/routes.go",
  "source_line": 42
}
```

### 2. Business Rules

Validation logic, guard clauses, authorization checks, state machine transitions.

**Extraction patterns:**

| Pattern | Example | Rule extracted |
|---------|---------|----------------|
| Validation tags | `validate:"required,email,max=255"` | "email: required, valid email, max 255 chars" |
| Guard clauses | `if age < 18 { return ErrUnderage }` | "age must be >= 18" |
| Error constants | `ErrInsufficientBalance = errors.New(...)` | "balance check enforced" |
| Switch/case | `switch status { case "active": ... }` | "status enum: active, inactive, suspended" |
| Enum types | `type Status int; const Active Status = 1` | "Status: Active=1, Inactive=2, ..." |
| Auth checks | `if !user.HasRole("admin") { ... }` | "admin role required" |
| Nil checks | `if order.Payment == nil { return ... }` | "payment required for order" |

**Go-specific validation tag parsing:**

```go
// Popular tag formats to parse:
`validate:"required,min=8,max=128"`           // go-playground/validator
`binding:"required,email"`                     // gin
`json:"email" validate:"required,email"`       // combined
`db:"email" validate:"required,email,unique"`  // with DB tag
```

**Output per rule:**
```jsonc
{
  "category": "validation",
  "description": "Email must be valid and unique",
  "enforcement": "struct-tag",
  "location": "internal/model/user.go:15",
  "field": "User.Email",
  "constraints": [
    {"type": "required", "value": true},
    {"type": "format", "value": "email"},
    {"type": "unique", "value": true}
  ],
  "error_on_violation": "ErrInvalidEmail"
}
```

### 3. Invariants

Database constraints, type system guarantees, assertion patterns, architectural boundaries.

**Extraction patterns:**

| Pattern | Example | Invariant extracted |
|---------|---------|---------------------|
| SQL migrations | `NOT NULL`, `UNIQUE`, `CHECK(...)` | DB-level constraints |
| Foreign keys | `REFERENCES users(id)` | Referential integrity |
| Type assertions | `amount.(decimal.Decimal)` | Type safety boundary |
| Context deadlines | `context.WithTimeout(ctx, 30*time.Second)` | 30s operation timeout |
| Mutex patterns | `sync.Mutex` guarding field access | Concurrency invariant |
| Interface compliance | `var _ Interface = (*Impl)(nil)` | Type must implement interface |
| Build constraints | `//go:build !integration` | Build-time boundaries |

**Migration file parsing:**

Scan `migrations/`, `db/migrate/`, `sql/` directories for SQL files:
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    balance DECIMAL(10,2) NOT NULL CHECK (balance >= 0),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','active','suspended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Extracted invariants:
- `users.email`: NOT NULL, UNIQUE
- `users.balance`: NOT NULL, >= 0
- `users.status`: NOT NULL, enum('pending','active','suspended'), default 'pending'

### 4. SLA Parameters

Timeouts, retries, rate limits, circuit breakers, resource limits.

**Extraction patterns:**

| Pattern | Example | SLA extracted |
|---------|---------|---------------|
| Timeout config | `Timeout: 30 * time.Second` | "30s request timeout" |
| Retry config | `MaxRetries: 3, Backoff: exponential` | "3 retries, exponential backoff" |
| Rate limit | `rate.NewLimiter(10, 1)` | "10 req/sec rate limit" |
| Circuit breaker | `gobreaker.NewCircuitBreaker(settings)` | "circuit breaker with 5 failure threshold" |
| Pool size | `MaxOpenConns: 25` | "DB pool: 25 max connections" |
| Cache TTL | `cache.Set(key, val, 5*time.Minute)` | "5-minute cache TTL" |
| Queue config | `MaxSize: 1000, Workers: 4` | "Queue: 1000 buffer, 4 workers" |
| Health check | `/health`, `/ready` endpoints | "health check at /health" |

**Output per parameter:**
```jsonc
{
  "category": "timeout",
  "component": "HTTP client",
  "value": "30s",
  "location": "internal/client/http.go:23",
  "context": "Default timeout for external API calls",
  "configurable": true,
  "env_var": "HTTP_TIMEOUT"
}
```

## JSON Output Contract

```jsonc
{
  "version": "1.0.0",
  "repo": "owner/repo-name",
  "generated_at": "2026-04-13T15:30:00Z",
  "duration_ms": 4200,

  "api_contracts": {
    "http_endpoints": [ /* endpoint objects */ ],
    "grpc_services": [ /* service objects */ ],
    "graphql_schemas": [ /* schema objects */ ],
    "message_queues": [ /* topic/queue objects */ ],
    "total": 47,
    "documented_ratio": 0.34   // endpoints with existing OpenAPI vs discovered
  },

  "business_rules": {
    "validations": [ /* validation rule objects */ ],
    "guard_clauses": [ /* guard clause objects */ ],
    "auth_checks": [ /* authorization rule objects */ ],
    "state_machines": [ /* state transition objects */ ],
    "total": 89
  },

  "invariants": {
    "database": [ /* DB constraint objects */ ],
    "type_system": [ /* type assertion objects */ ],
    "concurrency": [ /* mutex/lock pattern objects */ ],
    "architectural": [ /* boundary objects */ ],
    "total": 34
  },

  "sla_parameters": {
    "timeouts": [ /* timeout objects */ ],
    "retries": [ /* retry config objects */ ],
    "rate_limits": [ /* rate limit objects */ ],
    "circuit_breakers": [ /* breaker config objects */ ],
    "resource_pools": [ /* pool config objects */ ],
    "health_checks": [ /* health endpoint objects */ ],
    "total": 21
  },

  "coverage": {
    "files_scanned": 312,
    "files_with_specs": 87,
    "spec_density": 0.28,        // files_with_specs / files_scanned
    "undocumented_endpoints": 31, // discovered but no OpenAPI
    "implicit_rules": 89,        // rules only in code, not in docs
    "explicit_specs": 16          // existing OpenAPI, proto, graphql files
  }
}
```

## Two-Phase Pipeline

### Phase 1: Deterministic Extraction (always runs, <10s)

Pure AST analysis — no LLM, no external calls.

```
Source files
    │
    ├── Go files ──→ go/ast parser
    │   Routes, struct tags, guard clauses,
    │   error constants, timeouts, config
    │
    ├── Proto files ──→ text parser
    │   Services, messages, options
    │
    ├── SQL migrations ──→ SQL parser
    │   Tables, constraints, indexes
    │
    ├── OpenAPI/Swagger ──→ YAML/JSON parser
    │   Existing documented endpoints
    │
    ├── GraphQL schemas ──→ text parser
    │   Types, queries, mutations
    │
    └── Config files ──→ YAML/JSON parser
        Timeouts, retries, pool sizes
```

**Language-specific extractors:**

| Language | AST Tool | What's Extracted |
|----------|----------|------------------|
| Go | `go/ast` (stdlib) | Routes, struct tags, guard clauses, errors, config |
| Python | tree-sitter | Flask/FastAPI routes, Pydantic models, decorators |
| TypeScript | tree-sitter | Express/Nest routes, Zod schemas, decorators |
| Java | tree-sitter | Spring annotations, JPA entities, Bean Validation |
| SQL | regex-based | DDL constraints, indexes, triggers |
| Proto | regex-based | Services, messages, options |
| Config | YAML/JSON parser | Timeouts, retries, pool sizes, feature flags |

Phase 1 is Go-focused for MVP. Other languages get basic regex extraction.

### Phase 2: LLM Enrichment (optional, 30-60s)

For richer descriptions and missed patterns:

1. **Summarize business rules:** Send guard clauses to LLM for natural language descriptions.
2. **Infer missing specs:** For endpoints without OpenAPI, generate draft spec from handler code.
3. **Cross-reference:** Match API contracts with business rules affecting same entities.
4. **Gap analysis:** Identify endpoints with no validation, no auth, no rate limiting.

Phase 2 is optional and invoked via `sdp spec --enrich` or by the `@spec` skill.

## Go Route Extraction Algorithm

The most valuable extraction for Go codebases. Algorithm:

```
1. Parse all .go files with go/ast
2. For each file, find import statements
   - Match against known router packages (chi, gin, echo, gorilla, stdlib)
3. For each function body, find method calls matching router patterns:
   - chi:    r.Get|Post|Put|Delete|Route|Group
   - gin:    r.GET|POST|PUT|DELETE|Group
   - echo:   e.GET|POST|PUT|DELETE|Group
   - stdlib: http.HandleFunc|Handle, mux.HandleFunc|Handle
   - gorilla: r.HandleFunc|Handle + .Methods()
4. Extract:
   - HTTP method (from function name)
   - Path pattern (first string argument)
   - Handler reference (function name or inline)
5. Resolve handler to function definition:
   - Follow the reference to find the actual handler function
   - Extract request/response types from signature
6. For each handler, find:
   - Struct tags on request/response types (json, validate, binding)
   - Guard clauses (if/return error patterns)
   - Middleware applied (from router chain)
```

## Middleware Detection

Common middleware patterns to detect:

```go
// chi middleware chain
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(auth.Required)
r.Use(ratelimit.Limit(10))

// gin middleware
r.Use(gin.Logger())
r.Use(cors.Default())
r.Use(auth.JWT())

// Wrapping patterns
handler = auth.Wrap(handler)
handler = ratelimit.New(handler, 10)
```

Each detected middleware becomes a spec annotation on the affected endpoints.

## CLI Interface

```bash
sdp spec <repo-path>                    # Full extraction, JSON to stdout
sdp spec --format text <repo-path>      # Human-readable summary
sdp spec --format markdown <repo-path>  # Markdown for skill
sdp spec --category api <repo-path>     # Only API contracts
sdp spec --category rules <repo-path>   # Only business rules
sdp spec --enrich <repo-path>           # Phase 2: LLM enrichment
sdp spec --output .sdp/specs/ <repo>    # Write to directory
sdp spec --diff <old.json> <new.json>   # Compare spec versions (drift detection)
```

### Spec Diff (Drift Detection)

Compare two spec versions to detect drift:

```bash
sdp spec --diff .sdp/specs/v1.json .sdp/specs/v2.json
```

Output:
```
 API Contracts:
   + POST /api/v2/orders (new endpoint)
   - GET /api/users/search (removed)
   ~ PUT /api/users/{id}: added field "role" to request body

 Business Rules:
   + User.Password: min length increased 8→12
   ~ Order.Amount: max changed from 10000 to 50000

 SLA Parameters:
   ~ HTTP timeout: 30s → 45s
   + Rate limit on /api/orders: 5 req/sec (new)
```

## Performance Targets

| Operation | 10K LOC | 100K LOC | 1M LOC | Target |
|-----------|---------|----------|--------|--------|
| Phase 1 (deterministic) | <2s | <10s | <60s | AST only |
| Phase 2 (LLM enrichment) | <15s | <60s | <5min | Optional |
| Spec diff | <1s | <2s | <5s | JSON comparison |

## Go Package Structure

```
internal/spec/
  ├── spec.go             # Orchestrates extraction pipeline
  ├── spec_test.go
  ├── api_extract.go      # API contract extraction (routes, proto, graphql)
  ├── api_extract_test.go
  ├── rules_extract.go    # Business rule extraction (validation, guards)
  ├── rules_extract_test.go
  ├── invariant_extract.go # Invariant extraction (DB, types, concurrency)
  ├── invariant_extract_test.go
  ├── sla_extract.go      # SLA parameter extraction (timeouts, retries)
  ├── sla_extract_test.go
  ├── go_routes.go        # Go-specific route extraction (chi, gin, echo, stdlib)
  ├── go_routes_test.go
  ├── go_tags.go          # Go struct tag parsing (validate, json, db)
  ├── go_tags_test.go
  ├── sql_parse.go        # SQL migration constraint extraction
  ├── sql_parse_test.go
  ├── config_parse.go     # Config file parameter extraction
  ├── config_parse_test.go
  ├── diff.go             # Spec version comparison
  ├── diff_test.go
  └── types.go            # SpecReport, Endpoint, Rule, Invariant, SLAParam

cmd/sdp/
  └── cmd_spec.go         # CLI subcommand (~150 LOC)
```

## Testing Strategy

1. **Route extraction tests:** Create Go files with known route patterns for each framework.
   - chi, gin, echo, gorilla/mux, stdlib
   - Nested groups, middleware chains, path parameters
   - Edge cases: inline handlers, method chaining

2. **Validation tag tests:** Create structs with various validate/binding tags.
   - Required, email, min/max, custom validators
   - Nested structs, embedded structs

3. **Guard clause tests:** Create functions with known guard patterns.
   - `if err != nil`, `if x < threshold`, `switch/case`
   - Multiple guards in one function

4. **SQL migration tests:** Create migration files with known constraints.
   - NOT NULL, UNIQUE, CHECK, FOREIGN KEY, DEFAULT
   - Multiple dialects (PostgreSQL, MySQL, SQLite)

5. **Integration test:** Run against sdp_lab repo, verify discovered routes match actual CLI commands.

## Relationship to Existing Code

- **Extends:** `@reality` skill concept (code→spec inversion)
- **Reuses:** Architect's extractor pattern (`internal/architect/extract/`)
- **Does NOT copy:** `git_extract.go` — separate concerns, different output
- **Feeds into:** `sdp index` (API contracts enrich chunk metadata)
- **Consumed by:** `@spec` skill (JSON→markdown), `@landscape` meta-skill (gap analysis)

## Security

- Spec extraction only reads code — no execution, no network calls
- Extracted specs may reveal API structure — `.sdp/specs/` in `.gitignore` by default
- LLM enrichment (Phase 2) sends code snippets to LLM — warn user about sensitive code
- Never extract literal secrets (env vars, API keys, passwords) — detect and skip
