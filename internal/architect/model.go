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
