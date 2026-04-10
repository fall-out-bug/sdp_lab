package architect

// Actor represents a user or external actor in the C4 system context.
type Actor struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
}

// ExternalSystem represents an external system in the C4 system context.
type ExternalSystem struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
	Technology  string `json:"technology,omitempty"`
	Evidence    string `json:"evidence,omitempty"`
}

// C4Component represents a component within a container (C4 Level 3).
type C4Component struct {
	ID          string  `json:"id"`
	Path        string  `json:"path"`
	Description string  `json:"description,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// C4Container represents a deployable unit (C4 Level 2).
type C4Container struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Technology       string        `json:"technology,omitempty"`
	Description      string        `json:"description,omitempty"`
	HumanDescription string        `json:"human_description,omitempty"` // filled by team
	Source           string        `json:"source,omitempty"`
	Deploy           string        `json:"deploy,omitempty"`
	Components       []C4Component `json:"components,omitempty"`
}

// --- Data Model types (Council Round 2 I1 resolution) ---

// EdgeKind describes the type of a structural graph edge.
type EdgeKind string

const (
	EdgeImport     EdgeKind = "import"
	EdgeCall       EdgeKind = "call"
	EdgeImplements EdgeKind = "implements"
	EdgePersistsTo EdgeKind = "persists_to"
	EdgeExposes    EdgeKind = "exposes"
	EdgeContains   EdgeKind = "contains"
)

// ManifestDependency is a raw entry from a single manifest file.
type ManifestDependency struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	ManifestPath string `json:"manifest_path"`
	Constraint   string `json:"constraint,omitempty"`
	Direct       bool   `json:"direct"`
	Dev          bool   `json:"dev,omitempty"`
}

// DependencyCorrelation is a cross-manifest deduplication result.
type DependencyCorrelation struct {
	CanonicalName string               `json:"canonical_name"`
	Ecosystem     string               `json:"ecosystem"`
	Sources       []ManifestDependency `json:"sources"`
	ResolvedID    string               `json:"resolved_id"`
	IsInternal    bool                 `json:"is_internal"`
}

// StructuralEdge is a graph topology edge. Pure structural, no LLM data.
type StructuralEdge struct {
	Source     string   `json:"source"`
	Target     string   `json:"target"`
	Kind       EdgeKind `json:"kind"`
	Weight     int      `json:"weight,omitempty"`
	Protocol   string   `json:"protocol,omitempty"`
	Schema     []string `json:"schema,omitempty"`
	Path       string   `json:"path,omitempty"`
	Method     string   `json:"method,omitempty"`
	SpecType   string   `json:"spec_type,omitempty"`
	SpecPath   string   `json:"spec_path,omitempty"`
	Confidence float64  `json:"confidence"`
}

// LLMEnrichment holds semantic annotations. NEVER read by structural algorithms.
// Attached via separate map[string]LLMEnrichment keyed by node/edge ID.
type LLMEnrichment struct {
	Description     string   `json:"description,omitempty"`
	TechnologyTags  []string `json:"technology_tags,omitempty"`
	BusinessPurpose string   `json:"business_purpose,omitempty"`
	DataFlow        string   `json:"data_flow,omitempty"`
}

// DepType describes how a module dependency is expressed.
type DepType string

const (
	DepImport  DepType = "import"
	DepRequire DepType = "require"
	DepDynamic DepType = "dynamic"
)

// ModuleDependency describes a single dependency within a module.
type ModuleDependency struct {
	TargetID   string  `json:"target_id"`
	Type       DepType `json:"type"`
	Line       int     `json:"line,omitempty"`
	IsExternal bool    `json:"is_external"`
}

// Module represents a language-level module/package.
type Module struct {
	ID           string             `json:"id"`
	Language     string             `json:"language"`
	Path         string             `json:"path"`
	Name         string             `json:"name"`
	Dependencies []ModuleDependency `json:"dependencies,omitempty"`
	Files        []string           `json:"files,omitempty"`
	ContainerID  string             `json:"container_id,omitempty"`
	IsGenerated  bool               `json:"is_generated,omitempty"`
}

// ComponentType classifies the kind of component.
type ComponentType string

const (
	CompService     ComponentType = "service"
	CompLibrary     ComponentType = "library"
	CompApplication ComponentType = "application"
)

// Component is a logical grouping of modules (C4 Level 3).
// Note: C4Component (defined above) is used in C4 rendering code.
// Component is the canonical data model type for extractor assembly.
type Component struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Modules    []string      `json:"modules"`
	Type       ComponentType `json:"type"`
	Path       string        `json:"path"`
	Confidence float64       `json:"confidence"`
}

// APISurface describes a single exposed API endpoint or method.
type APISurface struct {
	Path         string `json:"path"`
	Method       string `json:"method"`
	Handler      string `json:"handler"`
	RequestType  string `json:"request_type,omitempty"`
	ResponseType string `json:"response_type,omitempty"`
	ComponentID  string `json:"component_id,omitempty"`
}

// ModuleBoundary defines a logical boundary within the codebase.
type ModuleBoundary struct {
	Name             string   `json:"name"`
	Pattern          string   `json:"pattern"`
	EntryFiles       []string `json:"entry_files"`
	PublicInterfaces []string `json:"public_interfaces"`
}

// LayerAssignment maps directories to an architectural layer.
type LayerAssignment struct {
	Layer        string   `json:"layer"`         // "presentation", "business", "data", "infrastructure"
	Directories  []string `json:"directories"`   // ["internal/handlers", "api/"]
	FilePatterns []string `json:"file_patterns"` // ["*_handler.go", "*_controller.py"]
	Confidence   float64  `json:"confidence"`
}

// --- End Data Model types ---

// C4Relationship represents a dependency between C4 elements.
type C4Relationship struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`     // "sync", "async", "data"
	Contract    string `json:"contract,omitempty"` // contract ID or spec path
	Risk        string `json:"risk,omitempty"`     // e.g. "circular_dependency"
}

// SystemInfo describes the top-level system.
type SystemInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ModelState indicates whether the model is auto-generated or human-approved.
type ModelState string

const (
	ModelObserved  ModelState = "observed"
	ModelProposed  ModelState = "proposed"
	ModelReference ModelState = "reference"
)

// ReferenceModel is a C4-oriented architecture model of the repository.
type ReferenceModel struct {
	Version         string            `json:"version"`
	State           ModelState        `json:"state"`
	GeneratedAt     string            `json:"generated_at,omitempty"`
	AnalyzedCommit  string            `json:"analyzed_commit,omitempty"`
	System          SystemInfo        `json:"system"`
	Actors          []Actor           `json:"actors,omitempty"`
	ExternalSystems []ExternalSystem  `json:"external_systems,omitempty"`
	Containers      []C4Container     `json:"containers,omitempty"`
	Relationships   []C4Relationship  `json:"relationships,omitempty"`
}
