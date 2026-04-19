// Package bootstrap provides non-destructive planning and dry-run for SDP
// project bootstrapping. It discovers available .sdp/ inputs and repo
// conventions, then generates a plan without mutating any files.
package bootstrap

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// BootstrapConfig holds the user-supplied configuration for a bootstrap run.
type BootstrapConfig struct {
	// RepoPath is the absolute path to the target repository.
	RepoPath string
	// DryRun reports what would be generated without writing files.
	DryRun bool
	// Force overwrites existing user content.
	Force bool
	// Only restricts generation to specific artifact types.
	// Valid values: "claude-md", "agents-md", "policies", "hooks", "beads".
	Only []string
	// NoVerify skips build/test/lint command verification.
	NoVerify bool
	// Beads enables beads initialization (opt-in, default false).
	Beads bool
	// UseDraft prefixes generated files with "DRAFT-" and injects review
	// headers. The CLI defaults this to true; set to false via --yes or
	// --auto-curate flags to produce final artifacts without DRAFT prefix.
	UseDraft bool
}

// ShouldUseDraft returns true if artifacts should be written with DRAFT prefix.
func (c BootstrapConfig) ShouldUseDraft() bool {
	return c.UseDraft
}

// DataSourceInfo describes the analysis data available in .sdp/.
type DataSourceInfo struct {
	// Scout is the scout.json data. Required for bootstrap.
	Scout *ScoutData `json:"scout,omitempty"`
	// Architect is the architect/report.json data. Optional.
	Architect *ArchitectData `json:"architect,omitempty"`
	// Metrics is the metrics report data. Optional.
	Metrics *MetricsData `json:"metrics,omitempty"`
	// Spec holds specification data. Optional.
	Spec *SpecData `json:"spec,omitempty"`
	// Index holds index data. Optional.
	Index *IndexData `json:"index,omitempty"`
}

// ScoutData represents the subset of scout.json used by bootstrap.
type ScoutData struct {
	PrimaryLanguage string            `json:"primary_language"`
	BuildSystem     string            `json:"build_system"`
	Languages       map[string]float64 `json:"languages"`
	HasTests        bool              `json:"has_tests"`
	HasCI           bool              `json:"has_ci"`
	CISystem        string            `json:"ci_system"`
	HasLinter       bool              `json:"has_linter"`
	Monorepo        bool              `json:"monorepo"`
	TestRatio       float64           `json:"test_ratio"`
	TotalFiles      int               `json:"total_files"`
}

// ArchitectData represents the architect/report.json used by bootstrap.
type ArchitectData struct {
	Components []string          `json:"components"`
	Decisions []ArchDecision    `json:"decisions"`
	Patterns  []string          `json:"patterns"`
}

// ArchDecision represents a single architecture decision.
type ArchDecision struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

// MetricsData represents the metrics report used by bootstrap.
type MetricsData struct {
	BusFactor      int     `json:"bus_factor"`
	CommitFreq     string  `json:"commit_frequency"`
	Staleness      string  `json:"staleness"`
	TestCovHint    string  `json:"test_coverage_hint"`
	ComplexityHint string  `json:"complexity_hint"`
	ChurnRatio     float64 `json:"churn_ratio"`
}

// SpecData represents specification data used by bootstrap.
type SpecData struct {
	Files []SpecFile `json:"files"`
}

// SpecFile represents a single spec file entry.
type SpecFile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// IndexData represents index data used by bootstrap.
type IndexData struct {
	Symbols  int    `json:"symbols"`
	Files    int    `json:"files"`
	Language string `json:"language"`
}

// PlannedArtifact describes a single artifact that bootstrap will produce.
type PlannedArtifact struct {
	// Type is the artifact category: "claude_md", "agents_md", "policy", "hook", "beads".
	Type string `json:"type"`
	// Path is the target file path relative to the repo root.
	Path string `json:"path"`
	// Action is "create", "merge", or "skip".
	Action string `json:"action"`
	// Description is a human-readable explanation of what will happen.
	Description string `json:"description"`
}

// BootstrapPlan describes everything bootstrap intends to do, before doing it.
type BootstrapPlan struct {
	// WillCreate lists artifacts that will be created from scratch.
	WillCreate []PlannedArtifact `json:"will_create"`
	// WillMerge lists artifacts that will be merged into existing files.
	WillMerge []PlannedArtifact `json:"will_merge"`
	// WillSkip lists artifacts that already exist and will be left alone.
	WillSkip []PlannedArtifact `json:"will_skip"`
	// DataSources describes which .sdp/ inputs were found.
	DataSources DataSourceInfo `json:"data_sources"`
	// Commands holds the detected build/test/lint commands.
	Commands BuildCommands `json:"commands"`
}

// ArtifactResult records the outcome of generating a single artifact.
type ArtifactResult struct {
	Type       string `json:"type"`
	Path       string `json:"path"`
	Action     string `json:"action"`
	Status     string `json:"status"` // "ok", "skipped", "error"
	Message    string `json:"message,omitempty"`
}

// BootstrapReport is the final output of a bootstrap run.
type BootstrapReport struct {
	Version      string           `json:"version"`
	GeneratedAt  time.Time        `json:"generated_at"`
	Repo         string           `json:"repo"`
	Artifacts    []ArtifactResult `json:"artifacts"`
	DataSources  map[string]bool  `json:"data_sources"`
	Confidence   map[string]float64 `json:"confidence"`
	Notes        []string         `json:"notes,omitempty"`
	DurationMs   int64            `json:"duration_ms"`
	Verification []VerifyResult   `json:"verification,omitempty"`
	Kept         []string         `json:"kept,omitempty"`
	Updated      []string         `json:"updated,omitempty"`
}

// BuildCommands holds the detected build, test, lint, and run commands.
type BuildCommands struct {
	Build string `json:"build"`
	Test  string `json:"test"`
	Lint  string `json:"lint"`
	Run   string `json:"run,omitempty"`
}

// BootstrapStatus describes the current bootstrap state of a repository.
type BootstrapStatus struct {
	RepoPath      string   `json:"repo_path"`
	Bootstrapped  bool     `json:"bootstrapped"`
	ExistingFiles []string `json:"existing_files"`
	MissingFiles  []string `json:"missing_files"`
	DataSources   map[string]bool `json:"data_sources"`
	Suggestions   []string `json:"suggestions"`
}

// ParseScoutData parses raw JSON bytes into a ScoutData struct.
func ParseScoutData(data []byte) (*ScoutData, error) {
	var s ScoutData
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ParseArchitectData parses raw JSON bytes into an ArchitectData struct.
func ParseArchitectData(data []byte) (*ArchitectData, error) {
	var a ArchitectData
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// ParseMetricsData parses raw JSON bytes into a MetricsData struct.
func ParseMetricsData(data []byte) (*MetricsData, error) {
	var m MetricsData
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ClaudeMDTemplateData holds the data for rendering CLAUDE.md templates.
type ClaudeMDTemplateData struct {
	Name            string
	Description     string
	PrimaryLanguage string
	LanguageVersion string
	BuildSystem     string
	BuildCommand    string
	TestCommand     string
	LintCommand     string
	RunCommand      string
	CISystem        string
	ArchSummary     string
	Modules         []ModuleInfo
	CommitStyle     string
	GitFlow         string
	TestPattern     string
	ErrorPattern    string
	Antipatterns    []string
}

// ModuleInfo describes a single module for the CLAUDE.md template.
type ModuleInfo struct {
	Path    string
	Purpose string
	LOC     int
	Owner   string
}

// AgentsMDTemplateData holds the data for rendering AGENTS.md templates.
type AgentsMDTemplateData struct {
	Name          string
	Description   string
	Agents        []AgentInfo
	CommitStyle   string
	BranchPattern string
	AgentNotes    []AgentNote
}

// AgentInfo describes a single agent for the AGENTS.md template.
type AgentInfo struct {
	Name        string
	Description string
	BestFor     string
	ConfigPath  string
}

// AgentNote holds agent-specific notes for the AGENTS.md template.
type AgentNote struct {
	Agent string
	Notes string
}

// DraftHeader returns the DRAFT header comment to prepend to generated files.
// The date is formatted as YYYY-MM-DD.
func DraftHeader(date string) string {
	return "<!-- DRAFT: generated by sdp bootstrap " + date + ". Review, curate, then rename (remove DRAFT- prefix) before committing. -->\n"
}

// DraftPath prefixes the filename with "DRAFT-" for the given relative path.
func DraftPath(relPath string) string {
	dir := filepath.Dir(relPath)
	base := filepath.Base(relPath)
	if dir == "." {
		return "DRAFT-" + base
	}
	return filepath.Join(dir, "DRAFT-"+base)
}

// InjectTODOAfterMarkers inserts TODO markers after each generated section
// end marker in the content. The marker is placed immediately after
// "<!-- end generated by sdp bootstrap -->" lines.
func InjectTODOAfterMarkers(content string) string {
	endMarkerLine := "<!-- end generated by sdp bootstrap -->"
	todoLine := "\n<!-- TODO: verify this rule matches intended behavior -->"
	return strings.ReplaceAll(content, endMarkerLine, endMarkerLine+todoLine)
}
