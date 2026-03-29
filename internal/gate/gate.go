// Package gate implements human decision points (gates) in the SDP pipeline.
// Gates are represented as blocked beads that wait for human input before
// the pipeline can proceed.
package gate

import "time"

// Gate represents a human decision point in the pipeline.
type Gate struct {
	ID        string        `json:"id"`
	Question  string        `json:"question"`
	Context   string        `json:"context,omitempty"`
	Options   []string      `json:"options,omitempty"` // e.g. ["approve", "reject", "defer"]
	CreatedAt time.Time     `json:"created_at"`
	Timeout   time.Duration `json:"timeout,omitempty"` // 0 = no timeout

	// Resolution
	Answer     string     `json:"answer,omitempty"`
	Answerer   string     `json:"answerer,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// Status returns the gate status: "resolved", "timed_out", or "pending".
func (g *Gate) Status() string {
	if g.ResolvedAt != nil {
		return "resolved"
	}
	if g.Timeout > 0 && time.Since(g.CreatedAt) > g.Timeout {
		return "timed_out"
	}
	return "pending"
}

// IsBlocking returns true if the gate is still waiting for a human decision.
func (g *Gate) IsBlocking() bool {
	return g.Status() == "pending"
}
