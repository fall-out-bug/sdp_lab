package swarm

// Phase is the per-issue lifecycle phase.
type Phase string

const (
	PhaseClaimed     Phase = "claimed"
	PhaseAnalyzing   Phase = "analyzing"
	PhaseImplementing Phase = "implementing"
	PhaseReviewing   Phase = "reviewing"
	PhaseApproved    Phase = "approved"
	PhaseReworking   Phase = "reworking"
	PhaseMerging     Phase = "merging"
	PhasePublished   Phase = "published"
	PhaseDone        Phase = "done"
	PhaseBlocked     Phase = "blocked"
)

// State holds per-issue coordinator state.
type State struct {
	ProjectID string
	IssueID   string
	RunID     string
	Phase     Phase
	ReworkCount int
}

// Coordinator manages per-issue state machine.
type Coordinator struct {
	states map[string]*State
}

// NewCoordinator creates a Coordinator.
func NewCoordinator() *Coordinator {
	return &Coordinator{states: make(map[string]*State)}
}

// Claim transitions to claimed.
func (c *Coordinator) Claim(projectID, issueID, runID string) *State {
	key := projectID + ":" + issueID
	s := &State{ProjectID: projectID, IssueID: issueID, RunID: runID, Phase: PhaseClaimed}
	c.states[key] = s
	return s
}

// Transition moves to the next phase.
func (c *Coordinator) Transition(projectID, issueID string, next Phase) *State {
	key := projectID + ":" + issueID
	s := c.states[key]
	if s == nil {
		return nil
	}
	s.Phase = next
	return s
}

// Get returns state for a key.
func (c *Coordinator) Get(projectID, issueID string) *State {
	return c.states[projectID+":"+issueID]
}
