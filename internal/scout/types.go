// Package scout defines the ProjectCard contract for sdp scout.
package scout

import "time"

// ProjectCard is the stable output of sdp scout.
// All pointer and nullable fields use explicit nil/empty to signal "unknown".
// Consumers must check for nil before dereferencing.
type ProjectCard struct {
	Version    string          `json:"version"`
	ScannedAt  time.Time       `json:"scanned_at"`
	DurationMs int64           `json:"duration_ms"`
	Identity   Identity        `json:"identity"`
	Scale      Scale           `json:"scale"`
	Activity   Activity        `json:"activity"`
	Maturity   Maturity        `json:"maturity"`
	Build      Build           `json:"build"`
	Health     HealthSignals   `json:"health_signals"`
}

// Identity describes what the project is.
type Identity struct {
	Name           string                `json:"name"`
	Description    *string               `json:"description"`    // nil = not found
	RepoURL        *string               `json:"repo_url"`       // nil = not a git repo or no remote
	PrimaryLanguage string               `json:"primary_language"`
	Languages      map[string]LangStats  `json:"languages"`
	BuildSystem    *string               `json:"build_system"`   // nil = unknown
	BuildFiles     []string              `json:"build_files"`
	Monorepo       bool                  `json:"monorepo"`
}

// LangStats holds file count and ratio for a single language.
type LangStats struct {
	Files int     `json:"files"`
	Ratio float64 `json:"ratio"`
}

// Scale describes how big the project is.
type Scale struct {
	TotalFiles     int     `json:"total_files"`
	TotalLoc       int64   `json:"total_loc"`
	SourceFiles    int     `json:"source_files"`
	TestFiles      int     `json:"test_files"`
	TestRatio      float64 `json:"test_ratio"`
	GeneratedFiles int     `json:"generated_files"`
	VendorFiles    int     `json:"vendor_files"`
	MaxFileLoc     int     `json:"max_file_loc"`
	MedianFileLoc  int     `json:"median_file_loc"`
	Directories    int     `json:"directories"`
	DepthMax       int     `json:"depth_max"`
}

// Activity describes recent and historical commit patterns.
type Activity struct {
	FirstCommit          *string  `json:"first_commit"`           // RFC3339 date, nil = no git
	LastCommit           *string  `json:"last_commit"`            // RFC3339 date, nil = no git
	AgeMonths            int      `json:"age_months"`
	TotalCommits         int      `json:"total_commits"`
	Contributors         int      `json:"contributors"`
	ActiveContributors90d int     `json:"active_contributors_90d"`
	Commits30d           int      `json:"commits_30d"`
	Commits90d           int      `json:"commits_90d"`
	ActiveBranches       int      `json:"active_branches"`
}

// Maturity signals indicate project infrastructure completeness.
// Fields are explicit bool or nullable string — never guessed.
type Maturity struct {
	HasReadme      bool    `json:"has_readme"`
	HasLicense     bool    `json:"has_license"`
	HasCI          bool    `json:"has_ci"`
	CISystem       *string `json:"ci_system"` // nil = no CI detected
	HasTests       bool    `json:"has_tests"`
	HasLinter      bool    `json:"has_linter"`
	HasDocker      bool    `json:"has_docker"`
	HasReleases    bool    `json:"has_releases"`
	LatestRelease  *string `json:"latest_release"` // nil = no releases
	ReleaseCount   int     `json:"release_count"`
	HasCodeowners  bool    `json:"has_codeowners"`
	HasContributing bool   `json:"has_contributing"`
	HasChangelog   bool    `json:"has_changelog"`
}

// Build describes entry points and dependency surface.
type Build struct {
	EntryPoints     []string `json:"entry_points"`
	ConfigFiles     []string `json:"config_files"`
	PackageManager  *string  `json:"package_manager"`   // nil = unknown
	DependencyCount int      `json:"dependency_count"`
	DependencyFile  *string  `json:"dependency_file"`    // nil = unknown
}

// HealthSignals are lightweight heuristics derived from other fields.
// Enum-like fields use explicit string constants, never free-form text.
type HealthSignals struct {
	BusFactorEstimate int    `json:"bus_factor_estimate"`
	CommitFrequency   string `json:"commit_frequency"`    // "high", "medium", "low", "unknown"
	Staleness         string `json:"staleness"`           // "active", "recent", "stale", "dormant", "unknown"
	TestCoverageHint  string `json:"test_coverage_hint"`  // "good", "partial", "low", "none", "unknown"
	ComplexityHint    string `json:"complexity_hint"`     // "low", "medium", "high", "unknown"
}

// Health signal enum constants.
const (
	CommitFreqHigh   = "high"
	CommitFreqMedium = "medium"
	CommitFreqLow    = "low"

	StalenessActive  = "active"
	StalenessRecent  = "recent"
	StalenessStale   = "stale"
	StalenessDormant = "dormant"

	CovGood    = "good"
	CovPartial = "partial"
	CovLow     = "low"
	CovNone    = "none"

	ComplexityLow    = "low"
	ComplexityMedium = "medium"
	ComplexityHigh   = "high"

	Unknown = "unknown"
)
