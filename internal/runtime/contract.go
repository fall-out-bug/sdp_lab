// Package runtime defines the AutonomousRuntimeModule contract.
package runtime

// TaskContext holds execution context for a task.
type TaskContext struct {
	IssueID   string
	Branch    string
	RunID     string
	Model     string
	WorkDir   string
	EvidencePath string
}

// Plan holds execution plan for a task.
type Plan struct {
	IssueID   string
	SpecID    string
	Prompt    string
	Acceptance string
	Model     string
}

// AutonomousRuntimeModule is the shared contract for OpenCode and OpenClaw runtimes.
type AutonomousRuntimeModule interface {
	ClaimTask(issueID string) error
	LoadTask(issueID string) (*Plan, error)
	CreateBranch(issueID, slug string) (string, error)
	ExecuteTask(plan *Plan) (*TaskContext, error)
	RunVerification(ctx *TaskContext) (bool, error)
	BuildEvidence(ctx *TaskContext) (string, error)
	PublishPR(ctx *TaskContext) (string, error)
	UpdateTaskState(issueID, state string, payload map[string]any) error
	Escalate(issueID, reason string) error
}
