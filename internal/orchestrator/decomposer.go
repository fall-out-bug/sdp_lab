package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/llm"
)

// DecomposeResult holds a single subtask definition from decomposition.
type DecomposeResult struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Acceptance     string `json:"acceptance"`
	DependsOnIndex int    `json:"depends_on_index,omitempty"` // 0-based index of blocking task
}

// Decompose uses opencode run to analyze a feature and produce subtask definitions.
// Returns CreateOpts for each subtask, suitable for adapter.Create with --parent.
// DecomposeResultWithID pairs a created subtask ID with its opts.
type DecomposeResultWithID struct {
	ID   string
	Opts beads.CreateOpts
}

// Decompose uses opencode run to analyze a feature and produce subtask definitions.
// Creates beads issues and returns their IDs.
func Decompose(ctx context.Context, adapter *beads.Adapter, feature beads.Issue, workDir, model string) ([]DecomposeResultWithID, error) {
	outputPath := filepath.Join(workDir, ".sdp", "decompose-"+feature.ID+".json")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir decompose output: %w", err)
	}

	prompt := buildDecomposePrompt(feature, outputPath)
	boundary := llm.BoundarySpec{
		AllowedPathPrefixes:   []string{".sdp/", "internal/", "cmd/", "docs/", "specs/"},
		ControlPathPrefixes:   llm.DefaultControlPaths,
		ForbiddenPathPrefixes: llm.DefaultForbiddenPaths,
	}
	req := llm.ExecuteRequest{
		IssueID:            feature.ID,
		Title:              feature.Title,
		Description:        prompt,
		AcceptanceCriteria: "",
		SpecID:             feature.SpecID,
		Model:              model,
		WorkDir:             workDir,
		Boundary:           boundary,
		Timeout:             5 * time.Minute,
		OpencodeBinary:     "opencode",
	}
	res, err := llm.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("opencode decompose: %w", err)
	}
	_ = res

	b, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read decompose output: %w", err)
	}
	var results []DecomposeResult
	if err := json.Unmarshal(b, &results); err != nil {
		return nil, fmt.Errorf("parse decompose JSON: %w", err)
	}

	labels := []string{"autonomy", "strict-evidence", "workstream:builder", "lane:commit"}
	if model != "" {
		labels = append(labels, "model:"+model)
	}

	out := make([]DecomposeResultWithID, 0, len(results))
	createdIDs := make([]string, len(results))
	for i, r := range results {
		opts := beads.CreateOpts{
			Title:       r.Title,
			Type:        "task",
			Priority:    feature.Priority,
			Description: r.Description,
			Acceptance:  r.Acceptance,
			SpecID:      feature.SpecID,
			Labels:      labels,
			ParentID:    feature.ID,
		}
		id, err := adapter.Create(opts)
		if err != nil {
			return nil, fmt.Errorf("create subtask %d: %w", i, err)
		}
		createdIDs[i] = id
		out = append(out, DecomposeResultWithID{ID: id, Opts: opts})
	}
	for i, r := range results {
		idx := r.DependsOnIndex
		if idx >= 0 && idx < len(createdIDs) && idx != i {
			blocked, blocker := createdIDs[i], createdIDs[idx]
			_ = adapter.DepAdd(blocked, blocker)
		}
	}
	return out, nil
}

func buildDecomposePrompt(feature beads.Issue, outputPath string) string {
	var b strings.Builder
	b.WriteString("Analyze this feature and decompose it into implementable subtasks.\n\n")
	b.WriteString("## Feature\n\n")
	b.WriteString("**ID:** " + feature.ID + "\n\n")
	b.WriteString("**Title:** " + feature.Title + "\n\n")
	if feature.Description != "" {
		b.WriteString("**Description:**\n")
		b.WriteString(feature.Description)
		b.WriteString("\n\n")
	}
	if feature.AcceptanceCriteria != "" {
		b.WriteString("**Acceptance Criteria:**\n")
		b.WriteString(feature.AcceptanceCriteria)
		b.WriteString("\n\n")
	}
	b.WriteString("## Task\n\n")
	b.WriteString("Write a JSON file to " + outputPath + " with an array of subtask objects.\n")
	b.WriteString("Each object must have: title, description, acceptance (string), and optionally depends_on_index (0-based index of a prior subtask that must complete first).\n")
	b.WriteString("Example format:\n")
	b.WriteString(`[{"title":"Subtask 1","description":"...","acceptance":"..."},{"title":"Subtask 2","description":"...","acceptance":"...","depends_on_index":0}]` + "\n")
	b.WriteString("Create 2-8 subtasks. Each subtask should be independently implementable by a builder agent.\n")
	return b.String()
}
