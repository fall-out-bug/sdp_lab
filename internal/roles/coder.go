package roles

import (
	"context"
	"fmt"
	"os/exec"

	"sdp_dev/internal/llm"
)

// CoderStrategy implements code generation via builder workstream.
type CoderStrategy struct{}

// Execute runs the builder workstream (opencode) for the task.
func (c *CoderStrategy) Execute(ctx context.Context, input TaskInput) (TaskResult, error) {
	issue := input.FederatedTask.Issue
	workDir := input.FederatedTask.Workspace

	model := "glm-4.7"
	if input.Ctx != nil && input.Ctx.Policy != nil {
		model = input.Ctx.Policy.SelectModelSimple()
	}

	boundary, err := llm.LoadBoundary(workDir, "builder")
	if err != nil {
		return TaskResult{Err: err}, err
	}

	req := llm.ExecuteRequest{
		IssueID:            issue.ID,
		Title:              issue.Title,
		Description:        issue.Description,
		AcceptanceCriteria: issue.AcceptanceCriteria,
		SpecID:             issue.SpecID,
		Model:              model,
		WorkDir:             workDir,
		Boundary:           boundary,
	}
	res, err := llm.Execute(ctx, req)
	if err != nil {
		if res.BoundaryViolation != nil {
			resetCmd := exec.Command("git", "reset", "--hard", "HEAD")
			resetCmd.Dir = workDir
			_ = resetCmd.Run()
		}
		return TaskResult{Err: err, ChangedFiles: res.ChangedFiles}, err
	}

	return TaskResult{
		ChangedFiles: res.ChangedFiles,
		Summary:      fmt.Sprintf("Implemented: %d files changed", len(res.ChangedFiles)),
	}, nil
}
