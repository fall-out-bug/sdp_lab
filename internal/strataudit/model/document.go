package model

import "time"

type Level struct {
	ID          string
	Name        string
	Rank        int
	Description string
	Patterns    []string
}

type Document struct {
	ID              string
	Path            string
	LevelID         string
	ContentHash     string
	Content         string
	Version         int
	FileModifiedAt  time.Time
	Metadata        map[string]string
	IngestedAt      time.Time
}

type Coverage struct {
	ID             string
	LevelID        string
	TotalEntities  int
	TracedEntities int
	CoveragePct    float64
	ComputedAt     time.Time
}

type PipelineState struct {
	ID          string
	Stage       string
	Status      string
	Checkpoint  string
	Error       string
	StartedAt   time.Time
	CompletedAt time.Time
}

type Candidate struct {
	ID              string
	SourceEntityID  string
	TargetEntityID  string
	Similarity      float64
	Verified        bool
	TraceID         string
}

type EntityScore struct {
	Entity Entity
	Score  float64
}

type Page struct {
	Offset int
	Limit  int
}
