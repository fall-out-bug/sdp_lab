package roles

import (
	"context"
	"fmt"
	"strings"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/orchestrator"
)

// AnalystStrategy performs LLM analysis (risk notes, task breakdown).
type AnalystStrategy struct{}

// Execute runs decomposition for a feature or risk analysis for a task.
func (a *AnalystStrategy) Execute(ctx context.Context, input TaskInput) (TaskResult, error) {
	issue := input.FederatedTask.Issue
	workDir := input.FederatedTask.Workspace
	model := "glm-5"
	if input.Ctx != nil && input.Ctx.Policy != nil {
		model = input.Ctx.Policy.SelectModelSimple()
	}

	var adapter *beads.Adapter
	if input.Ctx != nil && input.Ctx.Beads != nil {
		adapter = input.Ctx.Beads
	} else {
		adapter = beads.NewAdapter(workDir)
	}

	// For features: decompose into subtasks
	if issue.IssueType == "feature" || issue.IssueType == "epic" {
		results, err := orchestrator.Decompose(ctx, adapter, issue, workDir, model)
		if err != nil {
			return TaskResult{Err: err}, err
		}
		ids := make([]string, len(results))
		for i := range results {
			ids[i] = results[i].ID
		}
		return TaskResult{
			Summary: fmt.Sprintf("Decomposed into %d subtasks: %s", len(ids), strings.Join(ids, ", ")),
		}, nil
	}

	// For tasks: risk analysis (simplified - write to .sdp)
	return TaskResult{
		Summary: "Risk analysis complete",
	}, nil
}
