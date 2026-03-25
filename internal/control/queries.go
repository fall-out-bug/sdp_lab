package control

import (
	"encoding/json"
	"fmt"
	"log"
)

// WhyBlocked explains why a given card is blocked.
// Returns blocker IDs and reasons.
func (s *Store) WhyBlocked(cardID string) ([]BlockerInfo, error) {
	if s.beadsRepo == nil {
		return nil, fmt.Errorf("why-blocked requires beads mode (current: %s)", s.RepoMode)
	}

	data, err := s.beadsRepo.runBD("blocked", cardID)
	if err != nil {
		return nil, fmt.Errorf("blocked query: %w", err)
	}

	// Parse bd blocked output
	issues, err := parseBdList(data)
	if err != nil {
		return nil, err
	}

	blockers := make([]BlockerInfo, 0, len(issues))
	for _, issue := range issues {
		blockers = append(blockers, BlockerInfo{
			ID:    issue.ID,
			Title: issue.Title,
			Notes: issue.Notes,
		})
	}

	return blockers, nil
}

// WhatNext returns the next actionable items (ready queue).
func (s *Store) WhatNext(limit int) ([]CardSummary, error) {
	cards, err := s.cardRepo().LoadCards("")
	if err != nil {
		return nil, err
	}

	var ready []CardSummary
	for _, c := range cards {
		if c.Status == "open" || c.Status == "ready" {
			ready = append(ready, summarize(c))
			if limit > 0 && len(ready) >= limit {
				break
			}
		}
	}

	return ready, nil
}

// WhatMissing returns items that lack evidence or have incomplete provenance.
func (s *Store) WhatMissing(projectID string) ([]MissingInfo, error) {
	cards, err := s.cardRepo().LoadCards(projectID)
	if err != nil {
		return nil, err
	}

	var missing []MissingInfo
	for _, c := range cards {
		m := MissingInfo{ID: c.ID, Title: c.Title}

		if c.ExecutorResult == nil && c.Status != "done" {
			m.MissingEvidence = true
		}
		if c.DispatchedPacketPath == "" && (c.Status == "executing" || c.Status == "reviewing") {
			m.MissingDispatch = true
		}
		if c.ExecutorRuntimeState == "" && c.Status == "executing" {
			m.MissingExecutorState = true
		}

		if m.MissingEvidence || m.MissingDispatch || m.MissingExecutorState {
			missing = append(missing, m)
		}
	}

	return missing, nil
}

// NeedsApproval returns items awaiting human gates.
func (s *Store) NeedsApproval() ([]CardSummary, error) {
	if s.beadsRepo == nil {
		return nil, fmt.Errorf("needs-approval requires beads mode (current: %s)", s.RepoMode)
	}

	data, err := s.beadsRepo.runBD("gate", "list")
	if err != nil {
		return nil, fmt.Errorf("gate list: %w", err)
	}

	issues, err := parseBdList(data)
	if err != nil {
		return nil, err
	}

	var approvals []CardSummary
	for _, issue := range issues {
		if issue.Status == "open" {
			for _, label := range issue.Labels {
				if label == "sdp:gate:human" {
					card := bdToCard(issue)
					approvals = append(approvals, summarize(*card))
					break
				}
			}
		}
	}

	return approvals, nil
}

// TraceFeature returns the full trace for a feature (all descendants).
func (s *Store) TraceFeature(featureID string) (*FeatureTrace, error) {
	if s.beadsRepo == nil {
		return nil, fmt.Errorf("trace requires beads mode (current: %s)", s.RepoMode)
	}

	data, err := s.beadsRepo.runBD("show", featureID, "--children")
	if err != nil {
		return nil, fmt.Errorf("show children: %w", err)
	}

	// bd show --children returns {"id": [...issues...]}
	var childrenMap map[string][]bdIssue
	if err := json.Unmarshal(data, &childrenMap); err != nil {
		// Fallback: try parsing as plain array
		issues, parseErr := parseBdList(data)
		if parseErr != nil {
			return nil, fmt.Errorf("parse children: %w", err)
		}
		trace := &FeatureTrace{Root: featureID, Children: make([]CardSummary, 0, len(issues))}
		for _, issue := range issues {
			trace.Children = append(trace.Children, summarize(*bdToCard(issue)))
		}
		return trace, nil
	}

	var issues []bdIssue
	for _, list := range childrenMap {
		issues = list
		break
	}

	trace := &FeatureTrace{
		Root:     featureID,
		Children: make([]CardSummary, 0, len(issues)),
	}

	for _, issue := range issues {
		trace.Children = append(trace.Children, summarize(*bdToCard(issue)))
	}

	return trace, nil
}

// BlockerInfo describes why a card is blocked.
type BlockerInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Notes string `json:"notes"`
}

// MissingInfo describes what information is missing from a card.
type MissingInfo struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	MissingEvidence     bool   `json:"missing_evidence"`
	MissingDispatch     bool   `json:"missing_dispatch"`
	MissingExecutorState bool   `json:"missing_executor_state"`
}

// FeatureTrace contains a feature and all its descendant issues.
type FeatureTrace struct {
	Root     string        `json:"root"`
	Children []CardSummary `json:"children"`
}

// PrintWhyBlocked is a convenience method for CLI output.
func PrintWhyBlocked(blockers []BlockerInfo, logger *log.Logger) {
	if logger == nil {
		logger = log.New(log.Writer(), "", 0)
	}
	if len(blockers) == 0 {
		logger.Println("No blockers found.")
		return
	}
	for _, b := range blockers {
		logger.Printf("  🔒 %s: %s", b.ID, b.Title)
		if b.Notes != "" {
			logger.Printf("     %s", b.Notes)
		}
	}
}

// PrintMissing is a convenience method for CLI output.
func PrintMissing(missing []MissingInfo, logger *log.Logger) {
	if logger == nil {
		logger = log.New(log.Writer(), "", 0)
	}
	if len(missing) == 0 {
		logger.Println("All items have complete information.")
		return
	}
	for _, m := range missing {
		var flags []string
		if m.MissingEvidence {
			flags = append(flags, "evidence")
		}
		if m.MissingDispatch {
			flags = append(flags, "dispatch")
		}
		if m.MissingExecutorState {
			flags = append(flags, "executor_state")
		}
		logger.Printf("  ⚠️  %s: %s [missing: %s]", m.ID, m.Title, fmt.Sprintf("%v", flags))
	}
}
