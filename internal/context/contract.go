// Package sdpcontext provides the v1 API contract for SDP context operations.
// This package defines interfaces and types for repository mapping, diff retrieval,
// prompt budgeting, and cache hashing — core substrate services for SDP runtime.
package sdpcontext

import (
	"context"
	"time"
)

// ContractVersion is the semver version of this contract.
// All implementations must satisfy this interface contract.
const ContractVersion = "v1.0.0"

// RepoMapper defines the contract for mapping repository structure.
// Produces RepoMap as the primary context object for SDP runtime.
type RepoMapper interface {
	// Map analyzes the repository at root and produces a structured map.
	Map(ctx context.Context, root string) (*RepoMap, error)
	// FileEntries returns the discovered file entries from the last Map operation.
	FileEntries() []FileEntry
}

// DiffRetriever defines the contract for retrieving repository diffs.
// Provides change-aware context for incremental processing.
type DiffRetriever interface {
	// Diff computes the diff between base and head refs.
	Diff(ctx context.Context, base, head string) (*DiffResult, error)
	// Hunks returns diff hunks for a specific file from the last Diff operation.
	Hunks(file string) []DiffHunk
}

// PromptBudgeter defines the contract for managing token budgets.
// Allocates prompt space across layers while reserving capacity for injected context.
type PromptBudgeter interface {
	// Budget returns the total token budget for the specified model.
	Budget(model string) int
	// Allocate calculates tokens consumed by task and layers, returns allocation.
	Allocate(task string, layers []PromptLayer) int
	// Remaining returns unallocated tokens from the budget.
	Remaining() int
}

// CacheHasher defines the contract for computing cache keys.
// Produces deterministic keys for input-based cache invalidation.
type CacheHasher interface {
	// Hash generates a cache key from the provided inputs.
	Hash(inputs ...string) CacheKey
	// Validate checks if the key matches the hash of inputs.
	Validate(key CacheKey, inputs ...string) bool
}

// RepoMap represents the complete structure of a repository.
// Primary context object for SDP runtime operations.
type RepoMap struct {
	Root              string                 // Repository root path
	Files             []FileEntry            // All discovered files
	TotalFiles        int                    // Total number of files
	TotalLines        int                    // Total lines across all files
	LanguageBreakdown map[string]int         // Lines per language
	Metadata          map[string]interface{} // Optional implementation metadata
}

// FileEntry represents a single file in the repository.
type FileEntry struct {
	Path         string    // Relative path from root
	Language     string    // Detected programming language
	Lines        int       // Line count
	Hash         string    // Content hash for change detection
	LastModified time.Time // File modification timestamp
}

// DiffResult represents the complete diff between two refs.
type DiffResult struct {
	Base  string                 // Base commit/ref
	Head  string                 // Head commit/ref
	Files []string               // Changed file paths
	Hunks map[string][]DiffHunk  // Hunks per file
	Stats DiffStats              // Aggregated statistics
}

// DiffHunk represents a contiguous change section within a file.
type DiffHunk struct {
	File     string // File path (redundant with parent map key)
	StartLine int   // 1-based start line in original file
	EndLine   int   // 1-based end line in original file
	Added     int   // Lines added in this hunk
	Removed   int   // Lines removed in this hunk
	Content   string // Raw diff content for this hunk
}

// DiffStats provides summary statistics for a diff operation.
type DiffStats struct {
	FilesChanged int // Total files with changes
	LinesAdded   int // Total lines added across all files
	LinesRemoved int // Total lines removed across all files
}

// PromptBudget represents token allocation configuration.
type PromptBudget struct {
	TotalTokens    int     // Total budget for the model
	ContextPct     float64 // Percentage reserved for injected context (0.0-1.0)
	AllocatedTokens int    // Tokens currently allocated
	Model          string  // Model identifier (e.g., "gpt-4", "claude-opus")
}

// PromptLayer represents a single prompt section for budgeting.
type PromptLayer struct {
	Name    string // Layer identifier (e.g., "system", "task", "context")
	Content string // Layer content for token estimation
	Tokens  int    // Estimated or actual token count
}

// CacheKey is a hash-based identifier for cache entries.
type CacheKey string

// PromptBudgetContract documents the budget allocation contract.
// Implements charsPerToken=4 heuristic and layer ordering rules.
type PromptBudgetContract struct {
	// CharsPerToken is the heuristic for converting characters to tokens.
	// Standard average: 4 characters per token for ASCII text.
	CharsPerToken int

	// ContextPct reserves N% of total budget for injected context.
	// Prevents prompt overflow when runtime context is added.
	// Recommended: 0.15 (15%) to 0.25 (25%).
	ContextPct float64

	// LayerInjectionOrder defines the order layers are concatenated.
	// Layers are injected in array order: [0] first, [n] last.
	// Truncation removes oldest layers (lowest indices) first.
	LayerInjectionOrder []string

	// TruncationStrategy defines how to handle budget overruns.
	// "oldest" removes lowest-index layers first.
	// "largest" removes layers with highest token count first.
	TruncationStrategy string
}

// CacheContract documents the cache invalidation contract.
// Implements deterministic hashing and invalidation rules.
type CacheContract struct {
	// HashAlgorithm is the algorithm used for key generation.
	// Must be deterministic and collision-resistant.
	// Standard: SHA-256.
	HashAlgorithm string

	// InputSorting requires inputs to be sorted before hashing.
	// Ensures ["a", "b"] and ["b", "a"] produce the same key.
	InputSorting bool

	// InvalidationTriggers define when cache entries are invalidated.
	// "file-change": content hash mismatch
	// "config-change": configuration parameters modified
	// "time-based": TTL expiration (optional)
	InvalidationTriggers []string

	// KeyEncoding specifies how hash bytes are encoded as CacheKey.
	// Standard: hex (lowercase).
	KeyEncoding string
}
