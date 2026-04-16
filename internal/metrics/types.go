// Package metrics defines the MetricsReport contract and git ingestion pipeline
// for SDP process health analysis.
package metrics

import "time"

// MetricsReport is the stable output of sdp metrics.
// All fields use explicit nil/empty to signal "unknown".
// Consumers must check for nil before dereferencing pointer fields.
type MetricsReport struct {
	Version         string       `json:"version"`
	GeneratedAt     time.Time    `json:"generated_at"`
	RepoPath        string       `json:"repo_path"`
	DurationMs      int64        `json:"duration_ms"`
	CommitsAnalyzed int          `json:"commits_analyzed"`
	Period          TimePeriod   `json:"period"`
	Hygiene         *Hygiene     `json:"hygiene,omitempty"`
	Waste           *Waste       `json:"waste,omitempty"`
	GitFlow         *GitFlow     `json:"git_flow,omitempty"`
	ReleaseQuality  *ReleaseQuality `json:"release_quality,omitempty"`
	Stabilization   *Stabilization  `json:"stabilization,omitempty"`
	KnowledgeRisk   *KnowledgeRisk  `json:"knowledge_risk,omitempty"`
	Decay           *Decay          `json:"decay,omitempty"`
}

// TimePeriod describes the analysis window.
type TimePeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// RawCommit is a parsed git commit with file-level change data.
// This is the shared raw data type consumed by all seven analyzers.
type RawCommit struct {
	Hash    string       `json:"hash"`
	Author  string       `json:"author"`
	Date    time.Time    `json:"date"`
	Subject string       `json:"subject"`
	Body    string       `json:"body"`
	Files   []FileChange `json:"files,omitempty"`
}

// FileChange represents a single file change within a commit (from git numstat).
type FileChange struct {
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
	Path    string `json:"path"`
}

// TagInfo holds parsed semver tag metadata.
type TagInfo struct {
	Tag      string    `json:"tag"`
	Date     time.Time `json:"date"`
	IsSemver bool      `json:"is_semver"`
}

// BranchInfo holds parsed remote branch metadata.
type BranchInfo struct {
	Name       string     `json:"name"`
	LastCommit *time.Time `json:"last_commit,omitempty"`
}

// GitData is the complete raw dataset produced by the collector.
// All seven analyzers consume this single structure.
type GitData struct {
	Commits       []RawCommit  `json:"commits"`
	Tags          []TagInfo    `json:"tags"`
	Branches      []BranchInfo `json:"branches"`
	MergeCount    int          `json:"merge_count"`
	ParseWarnings int          `json:"parse_warnings,omitempty"` // >0 when git log was truncated
}

// ── Category Result Types ─────────────────────────────────────────

// Hygiene holds commit hygiene metrics.
type Hygiene struct {
	TicketLinkedRatio       float64           `json:"ticket_linked_ratio"`
	TicketPatternsFound     []string          `json:"ticket_patterns_found"`
	ConventionalCommitsRatio float64           `json:"conventional_commits_ratio"`
	CommitTypeBreakdown     map[string]int    `json:"commit_type_breakdown"`
	FixToFeatureRatio       float64           `json:"fix_to_feature_ratio"`
	AvgMessageLength        float64           `json:"avg_message_length"`
	AvgFilesPerCommit       float64           `json:"avg_files_per_commit"`
	MonorepoStyleRatio      float64           `json:"monorepo_style_ratio"` // commits with >10 files
}

// Waste holds wasted-work metrics.
type Waste struct {
	ChurnRatio        float64           `json:"churn_ratio"`
	ChurnFilesTop     []ChurnFile       `json:"churn_files_top"`
	AbandonedBranches int               `json:"abandoned_branches"`
	AbandonedLinesEst int64             `json:"abandoned_lines_est"`
	RevertRate        float64           `json:"revert_rate"`
	RevertCount       int               `json:"revert_count"`
}

// ChurnFile describes a high-churn file.
type ChurnFile struct {
	Path      string `json:"path"`
	Added     int    `json:"added"`
	Deleted   int    `json:"deleted"`
	Commits   int    `json:"commits"`
}

// GitFlow holds branch-model detection results.
type GitFlow struct {
	DetectedModel           string   `json:"detected_model"`
	Confidence              float64  `json:"confidence"`
	Evidence                []string `json:"evidence"`
	BranchLifetimeMedianH   float64  `json:"branch_lifetime_median_hours"`
	BranchLifetimeP95H      float64  `json:"branch_lifetime_p95_hours"`
	MergeFrequencyPerWeek   float64  `json:"merge_frequency_per_week"`
	LongLivedBranches       int      `json:"long_lived_branches"`
}

// ReleaseQuality holds release quality metrics.
type ReleaseQuality struct {
	ReleasesAnalyzed          int              `json:"releases_analyzed"`
	AvgTimeToFirstHotfixH     float64          `json:"avg_time_to_first_hotfix_hours"`
	Releases                  []ReleaseInfo    `json:"releases,omitempty"`
}

// ReleaseInfo describes a single release.
type ReleaseInfo struct {
	Tag              string    `json:"tag"`
	Date             time.Time `json:"date"`
	Fixes7d          int       `json:"fixes_7d"`
	Fixes14d         int       `json:"fixes_14d"`
	Fixes30d         int       `json:"fixes_30d"`
	TimeToFirstFixH  float64   `json:"time_to_first_fix_hours"`
}

// Stabilization holds release stabilization metrics.
type Stabilization struct {
	AvgPatchesToStable float64            `json:"avg_patches_to_stable"`
	Trend              string             `json:"trend"`
	Releases           []StabilizedRelease `json:"releases,omitempty"`
}

// StabilizedRelease describes stabilization of a release line.
type StabilizedRelease struct {
	Base              string `json:"base"`
	StabilizedAtPatch int    `json:"stabilized_at_patch"`
	PatchesTotal      int    `json:"patches_total"`
}

// KnowledgeRisk holds knowledge risk metrics.
type KnowledgeRisk struct {
	OverallBusFactor        int             `json:"overall_bus_factor"`
	GiniCoefficient         float64         `json:"gini_coefficient"`
	BusFactorByModule       []ModuleRisk    `json:"bus_factor_by_module,omitempty"`
	FormerContributorRatio  float64         `json:"former_contributor_ratio"`
	FormerContributors      []string        `json:"former_contributors,omitempty"`
}

// ModuleRisk holds bus factor info for a directory.
type ModuleRisk struct {
	Module             string  `json:"module"`
	BusFactor          int     `json:"bus_factor"`
	PrimaryAuthor      string  `json:"primary_author"`
	PrimaryAuthorRatio float64 `json:"primary_author_ratio"`
	FilesCount         int     `json:"files_count"`
}

// Decay holds code decay metrics.
type Decay struct {
	ShotgunSurgeryRatio  float64          `json:"shotgun_surgery_ratio"`
	ShotgunCommits       int              `json:"shotgun_commits"`
	MonotonicGrowthFiles []MonotonicFile  `json:"monotonic_growth_files,omitempty"`
	FixRecurrence        []FixRecurrenceEntry `json:"fix_recurrence,omitempty"`
}

// MonotonicFile represents a file that keeps growing without refactoring.
type MonotonicFile struct {
	Path          string `json:"path"`
	MonthsGrowing int    `json:"months_growing"`
	StartLOC      int    `json:"start_loc"`
	CurrentLOC    int    `json:"current_loc"`
	ZeroRefactor  bool   `json:"zero_refactor_events"`
}

// FixRecurrenceEntry describes files with repeated fix commits.
type FixRecurrenceEntry struct {
	Path         string  `json:"path"`
	FixCount     int     `json:"fix_count"`
	TotalCommits int     `json:"total_commits"`
	FixDensity   float64 `json:"fix_density"`
}
