package orchestrate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/sdputil"

)

// Checkpoint is the .sdp/checkpoints/F{NNN}.json schema for the orchestrate state machine.
// Compatible with ciloop.Checkpoint for pr_number, feature_id, branch (used by sdp-ci-loop and stop gate).
type Checkpoint struct {
	Schema      string        `json:"schema"`
	FeatureID   string        `json:"feature_id"`
	Branch      string        `json:"branch"`
	PRNumber    *int          `json:"pr_number,omitempty"`
	PRURL       string        `json:"pr_url,omitempty"`
	Phase       string        `json:"phase"`
	CreatedAt   string        `json:"created_at,omitempty"`
	UpdatedAt   string        `json:"updated_at,omitempty"`
	Workstreams []WSStatus    `json:"workstreams,omitempty"`
	Review      *ReviewStatus `json:"review,omitempty"`
	QA          *QAStatus     `json:"qa,omitempty"`
}

// WSStatus tracks a single workstream's execution.
type WSStatus struct {
	ID          string          `json:"id"`
	Status      string          `json:"status"` // pending, in_progress, done
	VerdictFile string          `json:"verdict_file,omitempty"`
	Commit      string          `json:"commit,omitempty"`
	Attempts    int             `json:"attempts,omitempty"`
	Dispatch    *WSDispatchInfo `json:"dispatch,omitempty"`
}

// WSDispatchInfo is a lightweight record of which harness/model was dispatched
// for a workstream. Defined locally to avoid importing the dispatch package.
type WSDispatchInfo struct {
	Harness   string  `json:"harness"`
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	Score     float64 `json:"score"`
	Reason    string  `json:"reason,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
	ColdStart bool    `json:"cold_start,omitempty"`
}

// ReviewStatus tracks review phase state.
type ReviewStatus struct {
	Iteration   int    `json:"iteration"`
	VerdictFile string `json:"verdict_file,omitempty"`
	Status      string `json:"status"` // pending, approved
}

type QAStatus struct {
	Iteration   int    `json:"iteration"`
	VerdictFile string `json:"verdict_file,omitempty"`
	Status      string `json:"status"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

// Phases in order.
const (
	PhaseInit   = "init"
	PhaseBuild  = "build"
	PhaseReview = "review"
	PhasePR     = "pr"
	PhaseCI     = "ci"
	PhaseQA     = "qa"
	PhaseDone   = "done"
)

// LoadCheckpoint reads the orchestrate checkpoint for a feature.
func LoadCheckpoint(dir, featureID string) (*Checkpoint, error) {
	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, featureID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint %s: %w", path, err)
	}
	var cp Checkpoint
	if err := sdputil.UnmarshalJSON(data, &cp); err != nil {
		return nil, fmt.Errorf("parse checkpoint %s: %w", path, err)
	}
	return &cp, nil
}

// SaveCheckpoint writes the checkpoint to disk atomically.
func SaveCheckpoint(dir string, cp *Checkpoint) error {
	if err := sdputil.ValidateFeatureID(cp.FeatureID); err != nil {
		return fmt.Errorf("validate feature id: %w", err)
	}
	cp.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	path := filepath.Join(dir, cp.FeatureID+".json")
	return sdputil.AtomicWriteJSON(path, cp)
}
