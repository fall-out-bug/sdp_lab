package spec

import "time"

// SpecReport is the top-level output of deterministic spec extraction.
type SpecReport struct {
	Version       string         `json:"version"`
	Repo          string         `json:"repo"`
	GeneratedAt   time.Time      `json:"generated_at"`
	DurationMs    int64          `json:"duration_ms"`
	APIContracts  APIContracts   `json:"api_contracts"`
	BusinessRules BusinessRules  `json:"business_rules"`
	Invariants    Invariants     `json:"invariants"`
	SLAParameters SLAParameters  `json:"sla_parameters"`
	Coverage      Coverage       `json:"coverage"`
	Enrichment    *EnrichmentInfo `json:"enrichment,omitempty"`
}

// EnrichmentInfo records whether optional LLM enrichment was attempted.
// Enrichment is always opt-in and never the default path.
type EnrichmentInfo struct {
	Attempted bool   `json:"attempted"`
	Status    string `json:"status"` // "not_configured", "available"
	Note      string `json:"note"`
}

// APIContracts holds all extracted HTTP endpoint definitions.
type APIContracts struct {
	HTTPEndpoints []Endpoint `json:"http_endpoints"`
	Total         int        `json:"total"`
}

// Endpoint describes a single HTTP route registration found in source code.
type Endpoint struct {
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	Handler    string   `json:"handler"`
	Middleware []string `json:"middleware,omitempty"`
	SourceFile string   `json:"source_file"`
	SourceLine int      `json:"source_line"`
}

// BusinessRules holds all extracted validation and constraint rules.
type BusinessRules struct {
	Validations []ValidationRule `json:"validations"`
	Total       int              `json:"total"`
}

// ValidationRule describes a single business rule discovered from source code.
type ValidationRule struct {
	Category    string       `json:"category"`
	Description string       `json:"description"`
	Enforcement string       `json:"enforcement"`
	Location    string       `json:"location"`
	Field       string       `json:"field"`
	Constraints []Constraint `json:"constraints"`
}

// Constraint represents a single constraint on a field.
type Constraint struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// SQLConstraint represents a constraint extracted from SQL migrations.
type SQLConstraint struct {
	Table       string `json:"table"`
	Column      string `json:"column"`
	Type        string `json:"type"`        // NOT NULL, UNIQUE, CHECK, FOREIGN KEY, DEFAULT
	Value       string `json:"value,omitempty"`
	References  string `json:"references,omitempty"` // For FK: referenced table(column)
	SourceFile  string `json:"source_file"`
	SourceLine  int    `json:"source_line"`
}

// Coverage reports how much of the repo was examined.
type Coverage struct {
	FilesScanned   int     `json:"files_scanned"`
	FilesWithSpecs int     `json:"files_with_specs"`
	SpecDensity    float64 `json:"spec_density"`
}

// Invariants holds all extracted system invariants.
type Invariants struct {
	Database      []DBInvariant    `json:"database"`
	TypeSystem    []TypeInvariant  `json:"type_system"`
	Concurrency   []ConcInvariant  `json:"concurrency"`
	Architectural []ArchInvariant  `json:"architectural"`
	Total         int              `json:"total"`
}

// DBInvariant represents a database-level invariant.
type DBInvariant struct {
	Table      string `json:"table"`
	Column     string `json:"column"`
	Constraint string `json:"constraint"`
	Detail     string `json:"detail"`
	Location   string `json:"location"`
}

// TypeInvariant represents a type-system invariant.
type TypeInvariant struct {
	Category string `json:"category"` // type_assertion, interface_compliance
	Detail   string `json:"detail"`
	Location string `json:"location"`
}

// ConcInvariant represents a concurrency invariant.
type ConcInvariant struct {
	Category string `json:"category"` // mutex_guard, channel_sync
	Detail   string `json:"detail"`
	Location string `json:"location"`
}

// ArchInvariant represents an architectural boundary.
type ArchInvariant struct {
	Category string `json:"category"` // build_constraint, interface_boundary
	Detail   string `json:"detail"`
	Location string `json:"location"`
}

// SLAParameters holds all extracted SLA-related configuration.
type SLAParameters struct {
	Timeouts        []SLAParam `json:"timeouts"`
	Retries         []SLAParam `json:"retries"`
	RateLimits      []SLAParam `json:"rate_limits"`
	CircuitBreakers []SLAParam `json:"circuit_breakers"`
	ResourcePools   []SLAParam `json:"resource_pools"`
	HealthChecks    []SLAParam `json:"health_checks"`
	Total           int        `json:"total"`
}

// SLAParam describes a single SLA-related parameter.
type SLAParam struct {
	Category     string `json:"category"`
	Component    string `json:"component"`
	Value        string `json:"value"`
	Location     string `json:"location"`
	Context      string `json:"context,omitempty"`
	Configurable bool   `json:"configurable"`
	EnvVar       string `json:"env_var,omitempty"`
}

// SpecDiff holds the result of comparing two spec snapshots.
type SpecDiff struct {
	Version     string      `json:"version"`
	OldSnapshot string      `json:"old_snapshot"`
	NewSnapshot string      `json:"new_snapshot"`
	GeneratedAt time.Time   `json:"generated_at"`
	APIChanges  []Change    `json:"api_changes"`
	RuleChanges []Change    `json:"rule_changes"`
	InvChanges  []Change    `json:"invariant_changes"`
	SLAChanges  []Change    `json:"sla_changes"`
	Summary     DiffSummary `json:"summary"`
}

// Change represents a single difference between two spec snapshots.
type Change struct {
	Category string `json:"category"` // "added", "removed", "modified"
	Key      string `json:"key"`      // e.g. "POST /api/users"
	Old      string `json:"old,omitempty"`
	New      string `json:"new,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// DiffSummary counts changes by category.
type DiffSummary struct {
	Added    int `json:"added"`
	Removed  int `json:"removed"`
	Modified int `json:"modified"`
}
