# Data Model Implementation Spec

Date: 2026-04-10
Status: Draft
Owner: F105 (AI Architect)
Depends on: WS-02 (Core Types), C4 Graph Schema (Section 1)

---

## 1. DependencyInfo Split

The current `DependencyInfo` conflates manifest entries, graph topology, and LLM annotations. Split into four disjoint types:

### 1.1 ManifestDependency

Raw entry from a single manifest file. No resolution, no inference.

```go
type ManifestDependency struct {
    Name         string `json:"name"`           // e.g. "github.com/gin-gonic/gin"
    Version      string `json:"version"`        // semver string or commit hash
    ManifestPath string `json:"manifest_path"`  // e.g. "go.mod", "package.json"
    Constraint   string `json:"constraint,omitempty"` // e.g. "^1.2.0", ">=2.0.0"
    Direct       bool   `json:"direct"`         // true = declared in manifest, false = transitive
    Dev          bool   `json:"dev,omitempty"`  // devDependency / test-only
}
```

### 1.2 DependencyCorrelation

Cross-manifest deduplication result. Links the same logical dependency across ecosystems.

```go
type DependencyCorrelation struct {
    CanonicalName string               `json:"canonical_name"` // e.g. "lodash"
    Ecosystem     string               `json:"ecosystem"`      // "go", "npm", "maven", "pypi"
    Sources       []ManifestDependency `json:"sources"`        // all manifest entries that resolve here
    ResolvedID    string               `json:"resolved_id"`    // deterministic ID (see Section 3)
    IsInternal    bool                 `json:"is_internal"`    // references another module in this repo
}
```

### 1.3 StructuralEdge

Graph topology edge. Pure structural, no LLM data.

```go
type EdgeKind string

const (
    EdgeImport      EdgeKind = "import"       // compile-time import
    EdgeCall        EdgeKind = "call"         // runtime function/method call
    EdgeImplements  EdgeKind = "implements"   // contract fulfillment
    EdgePersistsTo  EdgeKind = "persists_to"  // data storage dependency
    EdgeExposes     EdgeKind = "exposes"      // public API surface
    EdgeContains    EdgeKind = "contains"     // parent/child ownership
)

type StructuralEdge struct {
    Source    string  `json:"source"`              // node ID
    Target    string  `json:"target"`              // node ID
    Kind      EdgeKind `json:"kind"`
    Weight    int     `json:"weight,omitempty"`    // import count / call frequency
    Protocol  string  `json:"protocol,omitempty"`  // HTTP, GRPC, Import, FunctionCall
    Schema    []string `json:"schema,omitempty"`   // table names for PersistsTo
    Path      string  `json:"path,omitempty"`      // URL path for Exposes
    Method    string  `json:"method,omitempty"`    // HTTP verb or gRPC method
    SpecType  string  `json:"spec_type,omitempty"` // OpenAPI, Proto, GraphQL
    SpecPath  string  `json:"spec_path,omitempty"` // path to spec file
    Confidence float64 `json:"confidence"`         // 0.0-1.0
}
```

### 1.4 LLMEnrichment

Semantic annotations only. NEVER read by structural algorithms.

```go
type LLMEnrichment struct {
    Description      string   `json:"description,omitempty"`
    TechnologyTags   []string `json:"technology_tags,omitempty"`  // ["Go", "Gin", "PostgreSQL"]
    BusinessPurpose  string   `json:"business_purpose,omitempty"`
    DataFlow         string   `json:"data_flow,omitempty"`       // edges only
}
```

Invariant: `LLMEnrichment` is attached via a separate map (`map[string]LLMEnrichment`, keyed by node/edge ID). It is never a struct field on `Module`, `Component`, or `StructuralEdge`.

---

## 2. Concrete Go Types

### 2.1 Module

```go
type Module struct {
    ID           string             `json:"id"`            // deterministic (Section 3)
    Language     string             `json:"language"`      // "go", "python", "java", "typescript", "sql"
    Path         string             `json:"path"`          // directory relative to repo root
    Name         string             `json:"name"`          // package/module name
    Dependencies []ModuleDependency `json:"dependencies,omitempty"`
    Files        []string           `json:"files,omitempty"`     // relative paths
    ContainerID  string            `json:"container_id,omitempty"` // parent container
    IsGenerated  bool              `json:"is_generated,omitempty"`
}
```

### 2.2 ModuleDependency

```go
type DepType string

const (
    DepImport   DepType = "import"   // static import statement
    DepRequire  DepType = "require"  // package manager require
    DepDynamic  DepType = "dynamic"  // importlib, require(variable), reflection
)

type ModuleDependency struct {
    TargetID   string `json:"target_id"`              // resolved module ID or external name
    Type       DepType `json:"type"`
    Line       int    `json:"line,omitempty"`          // source line number
    IsExternal bool   `json:"is_external"`            // true if outside this repo
}
```

### 2.3 Component

```go
type ComponentType string

const (
    CompService     ComponentType = "service"
    CompLibrary     ComponentType = "library"
    CompApplication ComponentType = "application"
)

// Component is the canonical type. The C4 spec's C4Component alias is
// type C4Component = Component (same underlying type, zero conversion cost).
type Component struct {
    ID       string         `json:"id"`
    Name     string         `json:"name"`
    Modules  []string       `json:"modules"`          // module IDs
    Type     ComponentType  `json:"type"`
    Path     string         `json:"path"`             // primary directory
    Confidence float64     `json:"confidence"`
}
```

### 2.4 APISurface

```go
type APISurface struct {
    Path         string `json:"path"`                    // "/api/v1/users"
    Method       string `json:"method"`                  // GET, POST, or gRPC method path
    Handler      string `json:"handler"`                 // function/method reference
    RequestType  string `json:"request_type,omitempty"`  // Go struct / TS interface name
    ResponseType string `json:"response_type,omitempty"`
    ComponentID  string `json:"component_id,omitempty"`
}
```

### 2.5 ModuleBoundary

```go
type ModuleBoundary struct {
    Name             string   `json:"name"`              // e.g. "auth", "billing"
    Pattern          string   `json:"pattern"`           // glob: "internal/auth/**"
    EntryFiles       []string `json:"entry_files"`       // public API files
    PublicInterfaces []string `json:"public_interfaces"` // exported types/functions
}
```

### 2.6 LayerAssignment

```go
type LayerAssignment struct {
    Layer        string   `json:"layer"`         // "presentation", "business", "data", "infrastructure"
    Directories  []string `json:"directories"`    // ["internal/handlers", "api/"]
    FilePatterns []string `json:"file_patterns"`  // ["*_handler.go", "*_controller.py"]
    Confidence   float64  `json:"confidence"`
}
```

---

## 3. Deterministic ID Scheme

Format: `"<language>\x00<package-path>\x00<module-name>"` — uses null byte (`\x00`) separator.

**Why null byte:** `\x00` is invalid in POSIX paths, NPM package names, Maven coordinates, Cargo crate names, PyPI package names, and all target ecosystems. Unlike `::` (which collides with C++ scope resolution, Rust path separators, and Java method references), `\x00` has zero collision risk across all programming languages and package managers. Go strings can contain null bytes; the delimiter is safe for in-process use.

**Canonicalization invariant:** `JoinID(SplitID(id)) == id` for any well-formed ID. SplitID and JoinID are the SOLE canonical functions for ID construction and parsing. All ID comparisons must use normalized IDs (join after split). A well-formed ID never contains a literal `\x00` except as the delimiter between segments.

**Encoding rule:** All segments are stored verbatim (no URL-encoding, no base64). The null byte is the only reserved character; any segment containing `\x00` has it replaced with `%00` during JoinID (and `%00` in a segment is decoded back to `\x00` during SplitID). This normalization happens ONLY inside JoinID/SplitID — no other code may perform this encoding/decoding.

Rules:
- Language: lowercase ecosystem tag (`go`, `python`, `java`, `typescript`, `sql`)
- Package path: relative to repo root, forward slashes, no leading slash; if empty, use `_`. NPM scoped packages use their natural `/` separator (e.g., `@types/node` → `typescript\x00@types/node\x00node`)
- Module name: last segment of import path or directory name
- For external dependencies: `"ext\x00<ecosystem>\x00<name>"` (no path segment)
- For containers: `"container\x00<container-name>"` (derived from deploy config)
- For components: `"<container-id>\x00<component-slug>"`
- For Maven coordinates (`groupId:artifactId`): `java\x00<group-path>\x00<artifactId>` where `group-path` replaces `.` with `/` (e.g., `com.google.guava` → `java\x00com/google/guava\x00guava`)

Examples (shown with `[NUL]` for null byte readability):
```
go[NUL]internal/architect[NUL]architect
typescript[NUL]src/api[NUL]api
typescript[NUL]@types/node[NUL]node
ext[NUL]npm[NUL]lodash
container[NUL]auth-service
container[NUL]auth-service[NUL]user-handlers
java[NUL]com/google/guava[NUL]guava
```

Content hash disambiguation: when path alone is ambiguous (multiple modules at same path), the module-name segment is suffixed with `~<sha256(canonical_json)[:8]>` (e.g., `architect~a1b2c3d4`). This keeps the ID at exactly 3 segments. Canonical JSON: keys sorted, no whitespace, UTF-8.

Validation: split on `\x00`, expect exactly 3 segments for module IDs. First segment must match `^[a-z][a-z0-9]*$` or be `ext`/`container`. No segment may be empty.

**Runtime helpers:**

```go
// SplitID splits a deterministic ID into its segments.
// Decodes %00 back to \x00 within segments.
// Returns an error if the ID format is invalid.
// Idempotent: JoinID(SplitID(id)) == id for well-formed IDs.
func SplitID(id string) (segments []string, err error)

// JoinID constructs a deterministic ID from segments.
// Encodes any \x00 within segments as %00 before joining.
// Rejects empty segments.
// Idempotent: SplitID(JoinID(segs)) == segs for valid segments.
func JoinID(segments ...string) (string, error)

// NormalizeID parses and re-joins an ID to its canonical form.
// All ID comparisons MUST use NormalizeID on both sides.
func NormalizeID(id string) (string, error)
```

---

## 4. Merge Strategy: ProfileFragment to CodebaseProfile

Each extractor emits a `ProfileFragment`. The assembler merges fragments into a `CodebaseProfile`.

```go
type ProfileFragment struct {
    Source         string               `json:"source"`          // extractor name
    Modules        []Module             `json:"modules,omitempty"`
    Dependencies   []ManifestDependency `json:"dependencies,omitempty"`
    Correlations   []DependencyCorrelation `json:"correlations,omitempty"`
    Edges          []StructuralEdge     `json:"edges,omitempty"`
    APISurfaces    []APISurface         `json:"api_surfaces,omitempty"`
    Boundaries     []ModuleBoundary     `json:"boundaries,omitempty"`
    Layers         []LayerAssignment    `json:"layers,omitempty"`
    Containers     []C4Container        `json:"containers,omitempty"`
    Components     []C4Component        `json:"components,omitempty"`
}
```

Per-field merge rules:

| Field | Rule | Collision resolution |
|-------|------|---------------------|
| `modules` | Union by `id` | Highest-precedence-wins (higher tier overwrites lower, same tier: deterministic sort by extractor name) |
| `dependencies` | Union by `(name, manifest_path)` | Highest-precedence-wins |
| `edges` | Union by `(source, target, kind, protocol)` | Increment `weight`, keep higher `confidence` |
| `api_surfaces` | Union by `(path, method)` | Highest-precedence-wins |
| `boundaries` | Union by `name` | Merge: union `entry_files` and `public_interfaces` |
| `layers` | Union by `layer` | Keep highest `confidence` entry per layer name |
| `containers` | Union by `id` | Merge: union `components` arrays |
| `components` | Union by `id` | Highest-precedence-wins |
| `correlations` | Union by `canonical_name` | Merge: union `sources` arrays |

Merge order (extractor precedence, HIGHER number = HIGHER precedence, wins on conflict):
1. FileTreeAnalyzer (lowest)
2. DependencyManifestParser
3. SpecInventoryScanner
4. InfraExtractor
5. Language extractors (explicit index-based precedence: Go=0, Python=1, Java=2, TypeScript=3, SQL=4 — NOT alphabetical)
6. ImportGraphExtractor (highest)

**Invariant:** Once a higher-precedence extractor has populated a field for a given key, no lower-precedence extractor may overwrite it, regardless of execution order. This prevents race conditions from async extractor execution.

---

## 5. Serialization Format

All types serialize as JSON with these conventions:
- `omitempty` on all optional fields (already in struct tags)
- No `null` values in output — omit the key entirely
- Arrays never null — use empty `[]` if needed
- `confidence` always present on nodes/edges, range `[0.0, 1.0]`
- Timestamps in RFC3339 (`generated_at`, etc.)
- IDs are strings, never integers

### CodebaseProfile (top-level output)

```go
type CodebaseProfile struct {
    Version        string               `json:"version"`
    GeneratedAt    string               `json:"generated_at"`
    AnalyzedCommit string               `json:"analyzed_commit,omitempty"`
    System         SystemInfo           `json:"system"`
    Modules        []Module             `json:"modules"`
    Containers     []C4Container        `json:"containers"`
    Components     []C4Component        `json:"components"`
    Edges          []StructuralEdge     `json:"edges"`
    Dependencies   []ManifestDependency `json:"dependencies"`
    Correlations   []DependencyCorrelation `json:"correlations,omitempty"`
    APISurfaces    []APISurface         `json:"api_surfaces,omitempty"`
    Boundaries     []ModuleBoundary     `json:"boundaries,omitempty"`
    Layers         []LayerAssignment    `json:"layers,omitempty"`
    Enrichment     map[string]LLMEnrichment `json:"enrichment,omitempty"` // keyed by node/edge ID
}
```

File output: `{target-repo}/.sdp/architecture/codebase-profile.json`
Schema: `architecture-model.schema.json` (registered in `sdp/schema/index.json`)
