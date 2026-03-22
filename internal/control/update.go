package control

import (
	"encoding/json"
	"fmt"
	"os/exec"
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

var createBeadsIssueFn = createBeadsIssue

func (s *Store) ExecuteCard(projectID, cardID string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}
	if card.Status != "ready" {
		return nil, fmt.Errorf("card must be ready to execute, current status: %s", card.Status)
	}

	beadsID, err := createBeadsIssueFn(card)
	if err != nil {
		return nil, fmt.Errorf("create Beads issue: %w", err)
	}

	if beadsID == "" {
		return nil, fmt.Errorf("empty Beads ID returned")
	}

	card.LinkedBeadsIDs = cleanList(append(card.LinkedBeadsIDs, beadsID))
	card.Status = "executing"
	card.WaitingOn = nil
	card.ActiveAgents = ensureContains(card.ActiveAgents, "executor")
	if err := s.SaveCard(card); err != nil {
		return nil, err
	}

	if _, err := s.BuildProjectSnapshot(projectID); err != nil {
		return nil, fmt.Errorf("update project snapshot: %w", err)
	}

	if _, err := s.BuildPortfolioSnapshot(); err != nil {
		return nil, fmt.Errorf("update portfolio snapshot: %w", err)
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

type bdCreateOutput struct {
	ID string `json:"id"`
}

func createBeadsIssue(card *FeatureCard) (string, error) {
	title := fmt.Sprintf("%s [%s]", card.Title, card.ID)
	description := buildBeadsDescription(card)

	priority := "2"
	switch strings.ToLower(card.RiskLevel) {
	case "high":
		priority = "1"
	case "low":
		priority = "3"
	}

	args := []string{
		"create", title,
		"--description=" + description,
		"--type=feature",
		"--priority=" + priority,
		"--json",
	}

	cmd := exec.Command("bd", args...)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	jsonOutput := findJSONInOutput(outputStr)
	if jsonOutput == "" {
		if err != nil {
			return "", fmt.Errorf("bd create failed: %w, output: %s", err, outputStr)
		}
		return "", fmt.Errorf("no JSON found in bd create output: %s", outputStr)
	}

	var result bdCreateOutput
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		return "", fmt.Errorf("parse bd create output: %w, output: %s", err, jsonOutput)
	}

	return result.ID, nil
}

func findJSONInOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "{") {
			var buf strings.Builder
			for j := i; j < len(lines); j++ {
				buf.WriteString(lines[j])
				if strings.HasSuffix(strings.TrimSpace(lines[j]), "}") {
					break
				}
				if j < len(lines)-1 {
					buf.WriteString("\n")
				}
			}
			candidate := buf.String()
			if json.Valid([]byte(candidate)) {
				return candidate
			}
		}
	}
	return ""
}

func buildBeadsDescription(card *FeatureCard) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("FeatureCard: %s\n\n", card.ID))
	sb.WriteString(fmt.Sprintf("Project: %s\n\n", card.ProjectID))

	if card.NormalizedIntent != "" {
		sb.WriteString(fmt.Sprintf("Normalized Intent: %s\n\n", card.NormalizedIntent))
	}

	if card.TaskType != "" {
		sb.WriteString(fmt.Sprintf("Task Type: %s\n", card.TaskType))
	}
	if card.ExecutionMode != "" {
		sb.WriteString(fmt.Sprintf("Execution Mode: %s\n", card.ExecutionMode))
	}
	if card.TargetRepo != "" {
		sb.WriteString(fmt.Sprintf("Target Repo: %s\n", card.TargetRepo))
	}
	if card.TargetArea != "" {
		sb.WriteString(fmt.Sprintf("Target Area: %s\n", card.TargetArea))
	}
	sb.WriteString("\n")

	if len(card.ScopeIn) > 0 {
		sb.WriteString("Scope In:\n")
		for _, item := range card.ScopeIn {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
		sb.WriteString("\n")
	}

	if len(card.ScopeOut) > 0 {
		sb.WriteString("Scope Out:\n")
		for _, item := range card.ScopeOut {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
		sb.WriteString("\n")
	}

	if len(card.NonGoals) > 0 {
		sb.WriteString("Non-goals:\n")
		for _, item := range card.NonGoals {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
		sb.WriteString("\n")
	}

	if len(card.AcceptanceShape) > 0 {
		sb.WriteString("Acceptance Criteria:\n")
		for _, item := range card.AcceptanceShape {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
		sb.WriteString("\n")
	}

	if card.RecommendedNext != "" {
		sb.WriteString(fmt.Sprintf("Recommended Next Step: %s\n", card.RecommendedNext))
	}

	return sb.String()
}

func SetCreateBeadsIssueFn(fn func(*FeatureCard) (string, error)) {
	createBeadsIssueFn = fn
}

func MockCreateBeadsIssue(id string) func(*FeatureCard) (string, error) {
	return func(*FeatureCard) (string, error) { return id, nil }
}
