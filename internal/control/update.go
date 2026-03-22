package control

import (
	"fmt"
	"strings"
)

func (s *Store) LoadCard(projectID, cardID string) (*FeatureCard, error) {
	cards, err := s.LoadCards(projectID)
	if err != nil {
		return nil, err
	}
	for _, c := range cards {
		if c.ID == cardID {
			card := c
			return &card, nil
		}
	}
	return nil, fmt.Errorf("card not found: %s", cardID)
}

func (s *Store) ClarifyCard(projectID, cardID, normalizedIntent, taskType, targetRepo, riskLevel, nextStep string, scopeIn, scopeOut []string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}
	card.Status = "clarifying"
	if normalizedIntent != "" {
		card.NormalizedIntent = normalizedIntent
	}
	if taskType != "" {
		card.TaskType = taskType
	}
	if targetRepo != "" {
		card.TargetRepo = targetRepo
	}
	if riskLevel != "" {
		card.RiskLevel = riskLevel
	}
	if nextStep != "" {
		card.RecommendedNext = nextStep
	}
	if len(scopeIn) > 0 {
		card.ScopeIn = cleanList(scopeIn)
	}
	if len(scopeOut) > 0 {
		card.ScopeOut = cleanList(scopeOut)
	}
	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	if err := s.SaveCard(card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *Store) MarkNeedsInput(projectID, cardID string, needsFeedbackFrom, feedbackRequest, decisionRequired, authorUpdate, adminActionRequired []string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}
	card.Status = "needs_input"
	card.NeedsFeedbackFrom = cleanList(needsFeedbackFrom)
	card.FeedbackRequest = cleanList(feedbackRequest)
	card.DecisionRequired = cleanList(decisionRequired)
	card.AuthorUpdate = cleanList(authorUpdate)
	card.AdminActionRequired = cleanList(adminActionRequired)
	card.WaitingOn = ensureContains(card.WaitingOn, "human")
	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	if err := s.SaveCard(card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *Store) MarkReady(projectID, cardID string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}
	if err := validateReady(card); err != nil {
		return nil, err
	}
	card.Status = "ready"
	card.WaitingOn = nil
	card.NeedsFeedbackFrom = nil
	card.FeedbackRequest = nil
	card.DecisionRequired = nil
	card.AdminActionRequired = nil
	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	if err := s.SaveCard(card); err != nil {
		return nil, err
	}
	return card, nil
}

func (s *Store) ParkCard(projectID, cardID string, reason string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}
	card.Status = "parked"
	if reason != "" {
		card.AuthorUpdate = cleanList([]string{reason})
	}
	if err := s.SaveCard(card); err != nil {
		return nil, err
	}
	return card, nil
}

func validateReady(card *FeatureCard) error {
	if strings.TrimSpace(card.NormalizedIntent) == "" {
		return fmt.Errorf("cannot mark ready: normalized_intent is required")
	}
	if strings.TrimSpace(card.TaskType) == "" {
		return fmt.Errorf("cannot mark ready: task_type is required")
	}
	if strings.TrimSpace(card.TargetRepo) == "" {
		return fmt.Errorf("cannot mark ready: target_repo is required")
	}
	if len(cleanList(card.ScopeIn)) == 0 {
		return fmt.Errorf("cannot mark ready: scope_in is required")
	}
	if strings.TrimSpace(card.RiskLevel) == "" {
		return fmt.Errorf("cannot mark ready: risk_level is required")
	}
	if strings.TrimSpace(card.RecommendedNext) == "" {
		return fmt.Errorf("cannot mark ready: recommended_next_step is required")
	}
	return nil
}

func cleanList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ensureContains(items []string, want string) []string {
	for _, item := range items {
		if item == want {
			return items
		}
	}
	return append(items, want)
}
