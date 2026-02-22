package federation

import "sdp_dev/internal/beads"

// FederatedTask is a ready task from any project, with workspace path.
type FederatedTask struct {
	ProjectID string
	Issue     beads.Issue
	Workspace string
}

// IntakeBatchItem is a single feature or subtask in a batch intake.
type IntakeBatchItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Acceptance  string `json:"acceptance,omitempty"`
}

// IntakeDepEdge describes a dependency between subtasks (blocked depends on blocker).
type IntakeDepEdge struct {
	Blocked int `json:"blocked"`
	Blocker int `json:"blocker"`
}

// IntakeBatchPayload is the payload for batch intake over NATS (feature + subtasks + deps).
type IntakeBatchPayload struct {
	ProjectID string            `json:"project_id"`
	Feature   IntakeBatchItem   `json:"feature"`
	Subtasks  []IntakeBatchItem `json:"subtasks"`
	DepEdges  []IntakeDepEdge   `json:"dep_edges"`
}
