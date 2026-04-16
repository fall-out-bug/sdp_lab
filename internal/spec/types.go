package spec

import "time"

// SpecReport is the top-level output of deterministic spec extraction.
type SpecReport struct {
	Version       string        `json:"version"`
	Repo          string        `json:"repo"`
	GeneratedAt   time.Time     `json:"generated_at"`
	DurationMs    int64         `json:"duration_ms"`
	APIContracts  APIContracts  `json:"api_contracts"`
	BusinessRules BusinessRules `json:"business_rules"`
	Coverage      Coverage      `json:"coverage"`
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
