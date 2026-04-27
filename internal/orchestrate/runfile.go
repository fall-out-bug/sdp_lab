package orchestrate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
	"github.com/fall-out-bug/sdp_lab/internal/sdputil"
)

type runFileJSON struct {
	RunID        kernel.RunID      `json:"run_id"`
	FeatureID    string            `json:"feature_id"`
	Orchestrator string            `json:"orchestrator"`
	Branch       string            `json:"branch"`
	StartedAt    string            `json:"started_at"`
	Events       []kernel.TraceEvent `json:"events"`
	LastPhase    string            `json:"last_phase"`
	LastState    string            `json:"last_state"`
}

// EnsureRunFile creates the initial run file for a feature (atomic write).
func EnsureRunFile(dir, featureID, branch string) error {
	if err := sdputil.ValidateFeatureID(featureID); err != nil {
		return fmt.Errorf("validate feature id: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	runID := kernel.RunID(fmt.Sprintf("oneshot-%s-%s", featureID, time.Now().UTC().Format("20060102T150405Z")))
	path := filepath.Join(dir, string(runID)+".json")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir runs dir: %w", err)
	}
	rf := runFileJSON{
		RunID:        runID,
		FeatureID:    featureID,
		Orchestrator: "sdp-orchestrate",
		Branch:       branch,
		StartedAt:    now,
		Events:       []kernel.TraceEvent{{RunID: runID, Phase: "init", At: now}},
		LastPhase:    "init",
		LastState:    "ok",
	}
	return sdputil.AtomicWriteJSON(path, rf)
}
