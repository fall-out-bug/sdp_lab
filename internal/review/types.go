package review

// ReviewVerdict is a single reviewer's verdict.
type ReviewVerdict struct {
	PersonaID string
	Verdict   string // "approve", "needs_changes", "reject"
	Summary   string
	Comments  []string
}

// ConsensusResult aggregates multiple reviewer verdicts.
type ConsensusResult struct {
	Approved      bool
	NeedsChanges  bool
	Rejected      bool
	Verdicts      []ReviewVerdict
	Consensus     string // "approve", "needs_changes", "reject"
	Dissenting    []string
}

// ReworkRequest structures feedback for coder rework.
type ReworkRequest struct {
	ProjectID string
	IssueID   string
	RunID     string
	Comments  []string
	Personas  []string
}
