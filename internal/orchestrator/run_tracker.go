package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunDoc is the schema for .sdp/runs/<run-id>.json
type RunDoc struct {
	RunID          string    `json:"run_id"`
	IssueID        string    `json:"issue_id"`
	Orchestrator   string    `json:"orchestrator"`
	StartedAt      string    `json:"started_at"`
	Events         []RunEvent `json:"events"`
	LastPhase      string    `json:"last_phase,omitempty"`
	LastState      string    `json:"last_state,omitempty"`
	PRURL          string    `json:"pr_url,omitempty"`
	Host           string    `json:"host,omitempty"`
	PollSeconds    int       `json:"poll_seconds,omitempty"`
	TimeoutSeconds int       `json:"timeout_seconds,omitempty"`
}

// RunEvent is a single phase transition.
type RunEvent struct {
	At      string `json:"at"`
	Phase   string `json:"phase"`
	State   string `json:"state"`
	Message string `json:"message"`
	PRURL   string `json:"pr_url,omitempty"`
}

// RunTracker manages run trace artifacts.
type RunTracker struct {
	workDir string
	runsDir string
}

// NewRunTracker returns a tracker for the given working directory.
func NewRunTracker(workDir string) *RunTracker {
	return &RunTracker{
		workDir: workDir,
		runsDir: filepath.Join(workDir, ".sdp", "runs"),
	}
}

// Create creates a new run document and returns its path.
func (rt *RunTracker) Create(runID, issueID, orchestrator string, host string, pollSec, timeoutSec int) (string, error) {
	if err := os.MkdirAll(rt.runsDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rt.runsDir, runID+".json")
	doc := RunDoc{
		RunID:          runID,
		IssueID:        issueID,
		Orchestrator:   orchestrator,
		StartedAt:      time.Now().UTC().Format(time.RFC3339),
		Events:         nil,
		Host:           host,
		PollSeconds:    pollSec,
		TimeoutSeconds: timeoutSec,
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// AppendPhase appends a phase event to the run document.
func (rt *RunTracker) AppendPhase(runID, phase, state, message, prURL string) error {
	path := filepath.Join(rt.runsDir, runID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc RunDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	doc.Events = append(doc.Events, RunEvent{
		At:      time.Now().UTC().Format(time.RFC3339),
		Phase:   phase,
		State:   state,
		Message: message,
		PRURL:   prURL,
	})
	doc.LastPhase = phase
	doc.LastState = state
	if prURL != "" {
		doc.PRURL = prURL
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// RunFilePath returns the path for a run ID.
func (rt *RunTracker) RunFilePath(runID string) string {
	return filepath.Join(rt.runsDir, runID+".json")
}

// RunDir returns the runs directory.
func (rt *RunTracker) RunDir() string {
	return rt.runsDir
}

// EnsureRunsDir creates the runs directory if needed.
func (rt *RunTracker) EnsureRunsDir() error {
	return os.MkdirAll(rt.runsDir, 0o755)
}

// RunTrackerFromWorkDir is a convenience constructor.
func RunTrackerFromWorkDir(workDir string) (*RunTracker, error) {
	wd := workDir
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getwd: %w", err)
		}
	}
	return NewRunTracker(wd), nil
}
