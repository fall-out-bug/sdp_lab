package bridge

import (
	"sync"
)

type FindingSourceType string

const (
	FindingSourceReview  FindingSourceType = "review"
	FindingSourceCI      FindingSourceType = "ci"
	FindingSourceGitHub  FindingSourceType = "github"
	FindingSourceDrift   FindingSourceType = "drift"
	FindingSourceQA      FindingSourceType = "qa"
)

type TypedFinding struct {
	Source       FindingSourceType
	FeatureID    string
	WSID         string
	Blocking     bool
	Title        string
	Summary      string
	Description  string
	Severity     string
	Priority     int
	PRURL        string
	ArtifactRef  string
	EvidenceRef  string
	TraceRef     string
	DriftVerdict string
	DedupKey     string
}

type ReviewFindingInput struct {
	FeatureID    string
	WSID         string
	Blocking     bool
	Role         string
	Title        string
	Summary      string
	Description  string
	Severity     string
	Priority     int
	PRURL        string
	ArtifactRef  string
	EvidenceRef  string
	TraceRef     string
	DriftVerdict string
	DedupKey     string
}

type QAFindingInput struct {
	FeatureID       string
	WSID            string
	Blocking        bool
	Scenario        string
	FailedStep      string
	Title           string
	Summary         string
	Description     string
	Severity        string
	Priority        int
	PRURL           string
	ArtifactRef     string
	EvidenceRef     string
	TraceRef        string
	ExpectedOutcome string
	ActualOutcome   string
	DedupKey        string
}

// SyncStats tracks synchronization statistics.
type SyncStats struct {
	Processed int `json:"processed"`
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Skipped   int `json:"skipped"`
	Failed    int `json:"failed"`
}

type beadsIssueSummary struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Labels []string `json:"labels"`
}

// BeadsSink creates and updates Beads tasks from findings.
type BeadsSink struct {
	mu     sync.RWMutex
	prefix string // Issue prefix (e.g., "sdplab-")
	dryRun bool
	labels []string
	stats  SyncStats
	dedupe *DedupeStore
}

// NewBeadsSink creates a new Beads sink.
func NewBeadsSink(prefix string, dryRun bool, defaultLabels []string) *BeadsSink {
	return &BeadsSink{
		prefix: prefix,
		dryRun: dryRun,
		labels: defaultLabels,
		dedupe: NewDedupeStore(),
	}
}

// GetStats returns the current sync statistics.
func (s *BeadsSink) GetStats() SyncStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}
