package adapter

import (
	"fmt"
	"strings"

	"sdp_dev/internal/beads"
)

// TaskIntent represents the kubeopencode Task CRD payload derived from a Beads issue.
type TaskIntent struct {
	RunID      string
	IssueID    string
	Prompt     string
	Objective  string
	AgentRef   string
	Labels     map[string]string
	SpecHash   string
}

// IntentTranslator converts Beads issue/run intent into Task CRD payload.
type IntentTranslator struct{}

// NewIntentTranslator returns a new translator.
func NewIntentTranslator() *IntentTranslator {
	return &IntentTranslator{}
}

// Translate builds a TaskIntent from a Beads issue.
func (t *IntentTranslator) Translate(issue *beads.Issue, runID string) (*TaskIntent, error) {
	if issue == nil || issue.ID == "" {
		return nil, fmt.Errorf("issue required")
	}
	if runID == "" {
		runID = issue.ID + "-1"
	}
	prompt := strings.TrimSpace(issue.Title)
	if issue.Description != "" {
		prompt += "\n\n" + strings.TrimSpace(issue.Description)
	}
	objective := strings.TrimSpace(issue.AcceptanceCriteria)
	if objective == "" {
		objective = "Implement task per spec_id"
	}
	agentRef := "coder"
	for _, l := range issue.Labels {
		if strings.HasPrefix(l, "role:") {
			agentRef = strings.TrimPrefix(l, "role:")
			break
		}
	}
	labels := map[string]string{
		"beads.issue": issue.ID,
		"sdp.run_id":  runID,
	}
	return &TaskIntent{
		RunID:     runID,
		IssueID:   issue.ID,
		Prompt:    prompt,
		Objective: objective,
		AgentRef:  agentRef,
		Labels:    labels,
		SpecHash:  hashIntent(prompt, objective),
	}, nil
}

func hashIntent(prompt, objective string) string {
	// Minimal deterministic hash for intent; full implementation would use crypto/sha256
	s := prompt + "\x00" + objective
	h := uint64(0)
	for _, c := range s {
		h = h*31 + uint64(c)
	}
	return fmt.Sprintf("%016x", h)
}
