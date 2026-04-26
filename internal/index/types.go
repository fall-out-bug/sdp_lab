// Package index provides persistent codebase memory via local SQLite indexing.
// It supports regex-based language-aware chunking, structural edge extraction,
// FTS5 full-text search, and optional embedding-based semantic search.
package index

import "time"

// SchemaVersion is the current index database schema version.
const SchemaVersion = 1

// Chunk represents a semantic unit of source code extracted from a file.
type Chunk struct {
	ID          int64  `json:"id"`
	FilePath    string `json:"file_path"`
	SymbolName  string `json:"symbol_name,omitempty"`
	Kind        string `json:"kind"` // function, method, type, interface, const, var, file
	Scope       string `json:"scope,omitempty"`
	Language    string `json:"language"`
	LineStart   int    `json:"line_start"`
	LineEnd     int    `json:"line_end"`
	Content     string `json:"content"`
	Description string `json:"description,omitempty"`
	PageRank    float64 `json:"pagerank"`
	Hash        string `json:"hash"`
}

// Edge represents a structural relationship between two chunks.
type Edge struct {
	ID       int64   `json:"id"`
	SourceID int64   `json:"source_id"`
	TargetID int64   `json:"target_id"`
	Relation string  `json:"relation"` // calls, imports, implements, contains, uses
	Weight   float64 `json:"weight"`
}

// SymbolicEdge represents a relationship between two symbols by name rather than ID.
// It is produced by the parser and resolved to ID-based Edge values after all chunks
// for the repository have been inserted into the database.
type SymbolicEdge struct {
	SourceFile   string  // relative file path of the source symbol
	SourceSymbol string  // e.g. "Handler.Serve"
	TargetFile   string  // relative file path of the target symbol (empty for same-file)
	TargetSymbol string  // e.g. "ErrNotFound"
	Relation     string  // calls, uses, implements
	Weight       float64
}

// FileMeta holds per-file metadata for incremental indexing.
type FileMeta struct {
	Path         string `json:"path"`
	Hash         string `json:"hash"`
	LastIndexed  string `json:"last_indexed"`
	Language     string `json:"language,omitempty"`
	Loc          int    `json:"loc"`
	IsTest       bool   `json:"is_test"`
	IsGenerated  bool   `json:"is_generated"`
}

// ModuleMeta holds aggregated module metadata.
type ModuleMeta struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Purpose    string `json:"purpose,omitempty"`
	Owner      string `json:"owner,omitempty"`
	BusFactor  int    `json:"bus_factor"`
	FilesCount int    `json:"files_count"`
	Loc        int    `json:"loc"`
	IsHotspot  bool   `json:"is_hotspot"`
}

// IndexStats holds summary statistics of the index, used by manifest generation.
type IndexStats struct {
	RepoName    string `json:"repo_name"`
	TotalChunks int    `json:"total_chunks"`
	TotalFiles  int    `json:"total_files"`
	TotalEdges  int    `json:"total_edges"`
	Languages   []string `json:"languages"`
}

// MetaEntry is a key-value pair in the meta table.
type MetaEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// BuildResult holds the output of a cold build operation.
type BuildResult struct {
	TotalChunks int           `json:"total_chunks"`
	TotalFiles  int           `json:"total_files"`
	TotalEdges  int           `json:"total_edges"`
	FilesSkipped int          `json:"files_skipped"` // Files that failed to parse or produced no chunks
	Duration    time.Duration `json:"duration"`
	Languages   []string      `json:"languages"`
	DBPath      string        `json:"db_path"`
	Errors      []error       `json:"-"`
}

// BuildOptions configures the cold build behavior.
type BuildOptions struct {
	// RepoPath is the root of the repository to index.
	RepoPath string
	// DBPath is where .sdp/index.db will be written. Defaults to <RepoPath>/.sdp/index.db.
	DBPath string
	// MaxFileSizeBytes skips files larger than this threshold. Default 100KB.
	MaxFileSizeBytes int64
	// Languages restricts indexing to these languages. Empty means all supported.
	Languages []string
}

// RefreshOptions configures incremental refresh behavior.
type RefreshOptions struct {
	// RepoPath is the root of the repository to refresh.
	RepoPath string
	// DBPath is the existing index database. Defaults to <RepoPath>/.sdp/index.db.
	DBPath string
	// MaxFileSizeBytes skips files larger than this threshold. Default 100KB.
	MaxFileSizeBytes int64
	// Languages restricts indexing to these languages. Empty means all supported.
	Languages []string
}

// RefreshResult holds the output of an incremental refresh operation.
type RefreshResult struct {
	// FilesChecked is the total number of source files examined.
	FilesChecked int `json:"files_checked"`
	// FilesUpdated is the number of files whose content changed and were re-indexed.
	FilesUpdated int `json:"files_updated"`
	// FilesAdded is the number of new files not previously in the index.
	FilesAdded int `json:"files_added"`
	// FilesRemoved is the number of files that were deleted from disk but still in the index.
	FilesRemoved int `json:"files_removed"`
	// TotalChunks is the total number of chunks in the index after refresh.
	TotalChunks int `json:"total_chunks"`
	// TotalFiles is the total number of files in the index after refresh.
	TotalFiles int `json:"total_files"`
	// Duration is the wall-clock time for the refresh.
	Duration time.Duration `json:"duration"`
	// DBPath is the path to the index database.
	DBPath string `json:"db_path"`
	// Errors holds non-fatal errors encountered during refresh.
	Errors []error `json:"-"`
}

// ── Manifest Types ──────────────────────────────────────────────────

// ManifestData holds all data needed to render the manifest.md template.
type ManifestData struct {
	RepoName        string
	PrimaryLanguage string
	ArchStyle       string
	Summary         string
	Modules         []ModuleMeta
	EntryPoints     []string
	Conventions     ConventionSet
	ActiveWork      ActiveWork
}

// ConventionSet describes detected project conventions.
type ConventionSet struct {
	CommitStyle   string
	TestFramework string
	BuildSystem   string
	KeyPatterns   []string
}

// ActiveWork summarizes current repository activity.
type ActiveWork struct {
	LastCommitDate string
	LastAuthor     string
	ActiveBranches int
	OpenIssues     int
}

// ── Enrichment Types ────────────────────────────────────────────────

// EnrichmentInput holds optional data from neighboring toolkit artifacts.
// Every field is optional — enrichment degrades gracefully when absent.
type EnrichmentInput struct {
	// ArchitectReport comes from sdp architect output.
	ArchitectReport *ArchitectEnrichment
	// MetricsReport comes from sdp metrics output.
	MetricsReport *MetricsEnrichment
	// ScoutCard comes from sdp scout output.
	ScoutCard *ScoutEnrichment
	// CodeOwners is the raw CODEOWNERS file content.
	CodeOwners string
	// GitBlame holds file -> primary author mappings.
	GitBlame map[string]string
}

// ArchitectEnrichment holds data extracted from an architect report.
type ArchitectEnrichment struct {
	ArchStyle      string
	ModulePurposes map[string]string // module path -> purpose
	Patterns       []string
}

// MetricsEnrichment holds data extracted from a metrics report.
type MetricsEnrichment struct {
	BusFactor      int
	ModuleRisks    []ModuleRiskEntry
	CommitStyle    string
	ActiveBranches int
}

// ModuleRiskEntry is a simplified view of bus-factor risk per module.
type ModuleRiskEntry struct {
	Module        string
	BusFactor     int
	PrimaryAuthor string
}

// ScoutEnrichment holds data extracted from a scout project card.
type ScoutEnrichment struct {
	PrimaryLanguage string
	BuildSystem     string
	TestFramework   string
	TotalLOC        int64
	TestFiles       int
	TotalFiles      int
	EntryPoints     []string
}

// ── Query Types ────────────────────────────────────────────────────────

// SearchResult represents a single matched chunk from a query.
type SearchResult struct {
	Chunk    Chunk   `json:"chunk"`
	Score    float64 `json:"score"`
	MatchSrc string  `json:"match_src"` // "fts", "vector", "fused"
}

// SearchResponse is the unified response for all query modes.
type SearchResponse struct {
	Query    string         `json:"query"`
	Mode     string         `json:"mode"` // "semantic", "deps", "find"
	Results  []SearchResult `json:"results"`
	Total    int            `json:"total"`
	Duration string         `json:"duration,omitempty"`
}

// DepsResult represents a module-level dependency entry.
type DepsResult struct {
	ModuleName string `json:"module_name"`
	Path       string `json:"path"`
	LOC        int    `json:"loc"`
	IsHotspot  bool   `json:"is_hotspot"`
	BusFactor  int    `json:"bus_factor"`
	Relation   string `json:"relation"` // "forward" or "reverse"
}

// DepsResponse is the response for dependency queries.
type DepsResponse struct {
	Module  string       `json:"module"`
	Depth   int          `json:"depth"`
	Results []DepsResult `json:"results"`
}

// ── Store Interface for manifest/enrichment ──────────────────────────

// ManifestStore is the interface used by manifest generation and enrichment.
// It provides read/write access to modules, metadata, and entry points.
// Both the concrete SQLiteStore in store.go and the in-memory memStore
// in tests satisfy this interface.
type ManifestStore interface {
	// Meta operations (used by both manifest and enricher)
	GetMeta(key string) (string, error)
	SetMeta(key, value string) error
	LoadMeta(keys ...string) (map[string]string, error)

	// Module operations
	LoadModules() ([]ModuleMeta, error)
	UpsertModuleMeta(mm ModuleMeta) error
	UpdateModules(modules []ModuleMeta) error

	// Entry points
	LoadEntryPoints() ([]string, error)

	// Stats
	LoadStats() (*IndexStats, error)
}
