package ciloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunEvent is a single event appended to a run file.
type RunEvent struct {
	At    string `json:"at"`
	Phase string `json:"phase"`
	State string `json:"state"`
	Notes string `json:"notes,omitempty"`
}

// RunFile mirrors the .sdp/runs/{run-id}.json schema.
type RunFile struct {
	RunID        string     `json:"run_id"`
	FeatureID    string     `json:"feature_id"`
	Orchestrator string     `json:"orchestrator"`
	Branch       string     `json:"branch"`
	StartedAt    string     `json:"started_at"`
	Events       []RunEvent `json:"events"`
	LastPhase    string     `json:"last_phase"`
	LastState    string     `json:"last_state"`
}

// AppendRunEvent finds the latest run file for featureID in dir and appends an event.
func AppendRunEvent(dir, featureID, phase, state, notes string) error {
	path, err := findRunFile(dir, featureID)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read run file: %w", err)
	}
	var rf RunFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return fmt.Errorf("parse run file: %w", err)
	}
	rf.Events = append(rf.Events, RunEvent{
		At:    time.Now().UTC().Format(time.RFC3339),
		Phase: phase,
		State: state,
		Notes: notes,
	})
	rf.LastPhase = phase
	rf.LastState = state
	out, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run file: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0o644); err != nil {
		return fmt.Errorf("write run file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename run file: %w", err)
	}
	return nil
}

func findRunFile(dir, featureID string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read runs dir %s: %w", dir, err)
	}
	prefix := "oneshot-" + featureID + "-"
	var latest string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".json") {
			if e.Name() > latest {
				latest = e.Name()
			}
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no run file found for feature %s in %s", featureID, dir)
	}
	return filepath.Join(dir, latest), nil
}
