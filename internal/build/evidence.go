package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BuildEvidence represents the collected evidence for a build run.
// Compatible with the evidenceenv attestation structure (JSON serialisation).
type BuildEvidence struct {
	RunID     string            `json:"run_id"`
	Idea      string            `json:"idea"`
	Timestamp string            `json:"timestamp"`
	Status    string            `json:"status"`
	Stages    []StageEvidence   `json:"stages"`
	Dispatch  DispatchEvidence  `json:"dispatch,omitempty"`
	Sandbox   SandboxEvidence   `json:"sandbox,omitempty"`
}

// StageEvidence represents evidence for a single stage.
type StageEvidence struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Output   string `json:"output,omitempty"`
}

// DispatchEvidence records the dispatch decision.
type DispatchEvidence struct {
	Harness  string  `json:"harness,omitempty"`
	Provider string  `json:"provider,omitempty"`
	Model    string  `json:"model,omitempty"`
	Score    float64 `json:"score,omitempty"`
	Reason   string  `json:"reason,omitempty"`
}

// SandboxEvidence records sandbox execution results.
type SandboxEvidence struct {
	Type       string `json:"type"`
	BuildOK    bool   `json:"build_ok"`
	TestsOK    bool   `json:"tests_ok"`
	TestOutput string `json:"test_output,omitempty"`
}

// NewBuildEvidence creates a BuildEvidence with initial values from the config.
func NewBuildEvidence(config BuildConfig) *BuildEvidence {
	return &BuildEvidence{
		RunID:     config.RunID,
		Idea:      config.Idea,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "running",
		Stages:    []StageEvidence{},
	}
}

// AddStage appends evidence for a completed stage.
func (e *BuildEvidence) AddStage(stage StageEvidence) {
	e.Stages = append(e.Stages, stage)
}

// Write serialises the evidence to evidence.json in the given directory.
// Uses atomic write (temp file + rename) to avoid leaving a corrupted file
// if the process crashes mid-write.
func (e *BuildEvidence) Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("build: create evidence dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Errorf("build: marshal evidence: %w", err)
	}
	data = append(data, '\n')

	// Atomic write: temp file + rename
	target := filepath.Join(dir, "evidence.json")
	f, err := os.CreateTemp(dir, ".evidence-*.tmp")
	if err != nil {
		return fmt.Errorf("build: create temp file: %w", err)
	}
	tmpName := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpName)
		return fmt.Errorf("build: write evidence temp: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("build: close evidence temp: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("build: rename evidence temp: %w", err)
	}
	return nil
}

// WriteBuildResult converts a BuildResult into BuildEvidence and writes it.
func WriteBuildResult(result *BuildResult, dir string) error {
	ev := NewBuildEvidence(result.Config)
	ev.Status = result.Status

	for _, s := range result.Stages {
		se := StageEvidence{
			Name:     s.Stage,
			Status:   s.Status,
			Duration: s.Duration.String(),
			Output:   s.Output,
		}
		ev.AddStage(se)

		// Extract stage-specific evidence into top-level fields.
		switch s.Stage {
		case "dispatch":
			if dec, ok := s.Evidence["decision"]; ok {
				if b, err := json.Marshal(dec); err == nil {
					var de DispatchEvidence
					if json.Unmarshal(b, &de) == nil {
						ev.Dispatch = de
					}
				}
			}
		case "sandbox":
			if buildOK, ok := s.Evidence["build_ok"].(bool); ok {
				ev.Sandbox.BuildOK = buildOK
			}
			if testsOK, ok := s.Evidence["tests_ok"].(bool); ok {
				ev.Sandbox.TestsOK = testsOK
			}
			if sandboxType, ok := s.Evidence["sandbox_type"].(string); ok {
				ev.Sandbox.Type = sandboxType
			}
			// Capture test output from failed runs.
			if s.Status == "failed" && s.Error != "" {
				ev.Sandbox.TestOutput = s.Error
			}
		}
	}

	return ev.Write(dir)
}
