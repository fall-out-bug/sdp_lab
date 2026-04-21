// Package gate implements human decision points (gates) in the SDP pipeline.
// Gates are represented as blocked beads that wait for human input before
// the pipeline can proceed.
package gate

import (
	"fmt"
	"time"
)

// GateType distinguishes phase gates from manual gates.
type GateType string

const (
	GateTypeManual GateType = "manual"
	GateTypePlan   GateType = "plan"
	GateTypeReview GateType = "review"
	GateTypeEval   GateType = "eval"
)

// RequireEvidenceError is returned when a phase gate is resolved without evidence.
type RequireEvidenceError struct {
	GateType GateType
}

func (e *RequireEvidenceError) Error() string {
	return fmt.Sprintf("phase gate of type %s requires evidence", e.GateType)
}

// EvidenceNotFoundError is returned when the specified evidence file does not exist.
type EvidenceNotFoundError struct {
	Path string
}

func (e *EvidenceNotFoundError) Error() string {
	return fmt.Sprintf("evidence file not found: %s", e.Path)
}

// InvalidEvidenceError is returned when the evidence file is not valid JSON.
type InvalidEvidenceError struct {
	Path string
	Err  error
}

func (e *InvalidEvidenceError) Error() string {
	return fmt.Sprintf("evidence file is not valid JSON: %s: %v", e.Path, e.Err)
}

func (e *InvalidEvidenceError) Unwrap() error {
	return e.Err
}

// Gate represents a human decision point in the pipeline.
type Gate struct {
	ID        string        `json:"id"`
	Question  string        `json:"question"`
	Context   string        `json:"context,omitempty"`
	Options   []string      `json:"options,omitempty"` // e.g. ["approve", "reject", "defer"]
	CreatedAt time.Time     `json:"created_at"`
	Timeout   time.Duration `json:"timeout,omitempty"` // 0 = no timeout
	Type      GateType      `json:"type,omitempty"`    // manual, plan, review, eval

	// Resolution
	Answer       string     `json:"answer,omitempty"`
	Answerer     string     `json:"answerer,omitempty"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
	EvidencePath string     `json:"evidence_path,omitempty"` // path to evidence file
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

// RequireEvidence returns true for phase-typed gates (plan, review, eval).
func (g *Gate) RequireEvidence() bool {
	switch g.Type {
	case GateTypePlan, GateTypeReview, GateTypeEval:
		return true
	default:
		return false
	}
}

// ResolveWithEvidence resolves the gate with evidence validation.
// For phase-typed gates, evidencePath must be non-empty and point to an existing file
// that contains valid JSON. Manual gates do not require evidence but can optionally
// provide it.
// Returns an error if a phase gate requires evidence but none is provided,
// or if the evidence file does not exist or is not valid JSON.
func (g *Gate) ResolveWithEvidence(answer, answerer, evidencePath string) error {
	if g.RequireEvidence() {
		if evidencePath == "" {
			return &RequireEvidenceError{GateType: g.Type}
		}

		// Validate evidence schema (existence + JSON + required keys)
		if err := ValidateEvidenceSchema(g.Type, evidencePath); err != nil {
			return err
		}

		// Store evidence path for phase gates
		g.EvidencePath = evidencePath
	} else if evidencePath != "" {
		// For manual gates, optionally store evidence if provided
		g.EvidencePath = evidencePath
	}

	// Set resolution fields
	now := time.Now()
	g.Answer = answer
	g.Answerer = answerer
	g.ResolvedAt = &now

	return nil
}
