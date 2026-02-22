package federation

import "sdp_dev/internal/beads"

// FederatedTask is a ready task from any project, with workspace path.
type FederatedTask struct {
	ProjectID string
	Issue     beads.Issue
	Workspace string
}
