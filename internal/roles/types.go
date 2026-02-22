package roles

import (
	"context"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/federation"
)

// TaskInput is the input for a role execution.
type TaskInput struct {
	FederatedTask federation.FederatedTask
	Ctx           *agent.Context
}

// TaskResult is the output of a role execution.
type TaskResult struct {
	ChangedFiles []string
	Summary      string
	Verdict      string // for reviewer: "approve", "needs_changes", "reject"
	Comments     []string
	Err          error
}

// RoleStrategy executes work for a role.
type RoleStrategy interface {
	Execute(ctx context.Context, input TaskInput) (TaskResult, error)
}
