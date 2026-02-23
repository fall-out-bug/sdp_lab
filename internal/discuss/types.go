package discuss

import "time"

// SessionPhase is the current phase of a discussion session.
type SessionPhase string

const (
	PhaseCreated   SessionPhase = "created"
	PhaseAnalyzing SessionPhase = "analyzing"
	PhaseReady     SessionPhase = "ready"   // analysis complete, awaiting approval
	PhaseApproved  SessionPhase = "approved" // Beads issues created
	PhaseFailed    SessionPhase = "failed"
)

// DiscussRequest is the payload for POST /api/v1/discuss.
type DiscussRequest struct {
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
	UserID      string `json:"user_id,omitempty"`
}

// SubtaskProposal is a proposed subtask from LLM analysis.
type SubtaskProposal struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Acceptance     string `json:"acceptance"`
	DependsOnIndex int    `json:"depends_on_index,omitempty"`
}

// AnalysisResult is the LLM-generated analysis of a feature idea.
type AnalysisResult struct {
	Scope       string            `json:"scope"`
	Risks       []string          `json:"risks"`
	Subtasks    []SubtaskProposal  `json:"subtasks"`
	ModelUsed   string            `json:"model_used,omitempty"`
	AnalyzedAt  string            `json:"analyzed_at"`
}

// Session holds a discussion session and its analysis.
type Session struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"project_id"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Source       string         `json:"source"`
	UserID       string         `json:"user_id"`
	Phase        SessionPhase    `json:"phase"`
	Analysis     *AnalysisResult `json:"analysis,omitempty"`
	CreatedIssueIDs []string    `json:"created_issue_ids,omitempty"`
	Error        string         `json:"error,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}
