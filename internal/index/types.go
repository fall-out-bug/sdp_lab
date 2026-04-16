// Package index provides persistent codebase memory via local SQLite indexing.
// It supports tree-sitter-aware chunking, structural edge extraction,
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
	Duration    time.Duration `json:"duration"`
	Languages   []string      `json:"languages"`
	DBPath      string        `json:"db_path"`
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

// ── Store Interface for manifest/enrichment ──────────────────────────

// ManifestStore is the interface used by manifest generation and enrichment.
// It provides read/write access to modules, metadata, and entry points.
// Both the concrete SQLiteStore in store.go and the in-memory memStore
// in tests satisfy this interface.
type ManifestStore interface {
	// Meta operations (used by both manifest and enricher)
	GetMeta(key string) (string, error)
	SetMeta(key, value string) error
	SaveMeta(key, value string) error
	LoadMeta(keys ...string) (map[string]string, error)

	// Module operations
	ListModules() ([]ModuleMeta, error)
	LoadModules() ([]ModuleMeta, error)
	UpsertModuleMeta(mm ModuleMeta) error
	UpdateModules(modules []ModuleMeta) error

	// Entry points
	ListEntryPoints() ([]string, error)
	LoadEntryPoints() ([]string, error)

	// Stats
	LoadStats() (*IndexStats, error)
}
