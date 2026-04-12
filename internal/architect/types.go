// Package architect provides types and interfaces for AI-powered architecture
// analysis of external repositories. It produces architectural hypotheses,
// C4 models, contract catalogs, and conformance reports.
package architect

import "time"

// ArchStyle represents a hypothesis about a repository's architectural style.
type ArchStyle string

const (
	StyleLayered             ArchStyle = "layered"
	StyleModular             ArchStyle = "modular"
	StyleMicroservices       ArchStyle = "microservices"
	StyleEventDriven         ArchStyle = "event_driven"
	StyleServerless          ArchStyle = "serverless"
	StyleMonorepoMultiSvc    ArchStyle = "monorepo_multi_service"
	StyleLibrary             ArchStyle = "library"
	StyleInfraRepo           ArchStyle = "infra_repo"
)

// StyleScore pairs an architecture style with a confidence score and evidence.
type StyleScore struct {
	Style      ArchStyle `json:"style"`
	Confidence float64   `json:"confidence"`
	Evidence   []string  `json:"evidence,omitempty"`
}

// StyleHypothesis is a scored profile of architectural styles, not a single label.
type StyleHypothesis struct {
	Styles           []StyleScore `json:"styles"`
	HumanInputNeeded []string     `json:"human_input_needed,omitempty"`
}

// DetectedPattern represents a design pattern found in the codebase.
type DetectedPattern struct {
	Category   string   `json:"category"`   // "gof", "ddd", "infrastructure"
	Name       string   `json:"name"`       // "repository", "circuit_breaker", etc.
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence,omitempty"`
	Location   string   `json:"location,omitempty"`
}

// SpecArtifact represents an architectural specification found in the repo.
type SpecArtifact struct {
	Kind    string `json:"kind"`              // "openapi", "asyncapi", "protobuf", "graphql", "adr", "dockerfile", "terraform", "ci_cd", "migration"
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

// InfraArtifact represents an infrastructure definition found in the repo.
type InfraArtifact struct {
	Kind     string   `json:"kind"`               // "dockerfile", "compose", "terraform", "helm", "kustomize"
	Path     string   `json:"path"`
	Services []string `json:"services,omitempty"`
}

// Severity classifies the severity of a risk.
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// ArchRisk represents an architectural risk or gap.
type ArchRisk struct {
	Severity    Severity `json:"severity"`
	Category    string   `json:"category"`    // "missing_contract", "circular_dependency", "pii_exposure", etc.
	Description string   `json:"description"`
	Affected    []string `json:"affected,omitempty"`
}

// CodeMetrics holds quantitative metrics about the codebase.
type CodeMetrics struct {
	TotalFiles          int                `json:"total_files"`
	TotalLOC            int                `json:"total_loc"`
	TestRatio           float64            `json:"test_ratio"`
	LanguagesCount      int                `json:"languages_count"`
	ContainersDetected  int                `json:"containers_detected"`
	ComponentsDetected  int                `json:"components_detected"`
	ContractsDiscovered int                `json:"contracts_discovered"`
	ContractsMissing    int                `json:"contracts_missing_estimated"`
	GeneratedExcluded   int                `json:"generated_files_excluded"`
	LanguageBreakdown   map[string]int     `json:"language_breakdown,omitempty"` // ext -> file count (e.g. ".scala": 1823)
}

// LanguageInfo describes the language distribution of a repository.
type LanguageInfo struct {
	Primary      string             `json:"primary"`
	All          []string           `json:"all"`
	Distribution map[string]float64 `json:"distribution,omitempty"`
}

// ConfidenceSummary provides an overall confidence assessment.
type ConfidenceSummary struct {
	Overall            float64 `json:"overall"`
	StructuralAnalysis float64 `json:"structural_analysis"`
	StyleHypothesis    float64 `json:"style_hypothesis"`
	ContractCoverage   float64 `json:"contract_coverage"`
	Note               string  `json:"note,omitempty"`
}

// ArchitectureReport is the main output of an architecture analysis.
type ArchitectureReport struct {
	Version            string            `json:"version"`
	AnalyzedAt         time.Time         `json:"analyzed_at"`
	RepoRoot           string            `json:"repo_root"`
	AnalysisDurationS  float64           `json:"analysis_duration_seconds"`
	LLMCostUSD         float64           `json:"llm_cost_usd"`
	Languages          LanguageInfo      `json:"languages"`
	StyleHypothesis    StyleHypothesis   `json:"style_hypothesis"`
	PatternsDetected   []DetectedPattern `json:"patterns_detected,omitempty"`
	SpecsDiscovered    []SpecArtifact    `json:"specs_discovered,omitempty"`
	Risks              []ArchRisk        `json:"risks,omitempty"`
	Metrics            CodeMetrics       `json:"metrics"`
	ConfidenceSummary  ConfidenceSummary `json:"confidence_summary"`
}
