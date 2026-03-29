package dispatch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DispatchDecision records which harness was selected for a task.
type DispatchDecision struct {
	Harness      string        `json:"harness"`
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	Score        float64       `json:"score"`
	Reason       string        `json:"reason,omitempty"`
	Timestamp    string        `json:"timestamp,omitempty"`
	Alternatives []Alternative `json:"alternatives,omitempty"`
	ColdStart    bool          `json:"cold_start,omitempty"`
	Staleness    string        `json:"staleness,omitempty"` // "fresh", "stale", "expired" — of the winning profile
}

// Alternative represents a harness that was considered but not selected.
type Alternative struct {
	Harness string  `json:"harness"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason,omitempty"`
}

const decisionRelPath = ".sdp/dispatch-decision.json"

// WriteDecision atomically writes dec to <projectRoot>/.sdp/dispatch-decision.json.
func WriteDecision(projectRoot string, dec *DispatchDecision) error {
	destPath := filepath.Join(projectRoot, decisionRelPath)
	destDir := filepath.Dir(destPath)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("dispatch: create dir %s: %w", destDir, err)
	}

	data, err := json.MarshalIndent(dec, "", "  ")
	if err != nil {
		return fmt.Errorf("dispatch: marshal decision: %w", err)
	}

	// Atomic write: write to temp file then rename.
	tmp, err := os.CreateTemp(destDir, "dispatch-decision-*.json.tmp")
	if err != nil {
		return fmt.Errorf("dispatch: create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("dispatch: write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("dispatch: close temp file: %w", err)
	}

	if err := os.Rename(tmpName, destPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("dispatch: rename to %s: %w", destPath, err)
	}

	return nil
}

// LoadDecision reads the dispatch decision from <projectRoot>/.sdp/dispatch-decision.json.
func LoadDecision(projectRoot string) (*DispatchDecision, error) {
	path := filepath.Join(projectRoot, decisionRelPath)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("dispatch: read %s: %w", path, err)
	}

	var dec DispatchDecision
	if err := json.Unmarshal(data, &dec); err != nil {
		return nil, fmt.Errorf("dispatch: unmarshal decision: %w", err)
	}

	return &dec, nil
}
