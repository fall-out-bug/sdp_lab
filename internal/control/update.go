package control

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

func (s *Store) DispatchCard(projectID, cardID string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}

	if card.Status != "ready" && card.Status != "executing" {
		return nil, fmt.Errorf("card must be ready or executing to dispatch, current status: %s", card.Status)
	}

	packet, err := s.BuildExecutionPacket(projectID, cardID)
	if err != nil {
		return nil, fmt.Errorf("build execution packet: %w", err)
	}

	if err := s.writeDispatchPacket(projectID, cardID, packet); err != nil {
		return nil, fmt.Errorf("write dispatch packet: %w", err)
	}

	now := time.Now().UTC()
	card.Status = "executing"
	card.DispatchedAt = now.Format(time.RFC3339)
	card.DispatchedTo = packet.ExecutorRole
	card.DispatchedPacketPath = s.dispatchPacketPath(projectID, cardID)
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

func (s *Store) writeDispatchPacket(projectID, cardID string, packet *ExecutionPacket) error {
	if err := os.MkdirAll(s.dispatchDir(projectID), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal packet: %w", err)
	}

	return os.WriteFile(s.dispatchPacketPath(projectID, cardID), data, 0o644)
}

func (s *Store) dispatchDir(projectID string) string {
	return filepath.Join(s.ControlRoot, "projects", projectID, "dispatches")
}

func (s *Store) dispatchPacketPath(projectID, cardID string) string {
	return filepath.Join(s.dispatchDir(projectID), cardID+".json")
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

type FeedbackPacket struct {
	CardID              string   `json:"card_id"`
	CardTitle           string   `json:"card_title"`
	ProjectID           string   `json:"project_id"`
	Status              string   `json:"status"`
	NeedsFeedbackFrom   []string `json:"needs_feedback_from,omitempty"`
	FeedbackRequest     []string `json:"feedback_request,omitempty"`
	DecisionRequired    []string `json:"decision_required,omitempty"`
	AuthorUpdate        []string `json:"author_update,omitempty"`
	AdminActionRequired []string `json:"admin_action_required,omitempty"`
	BlockingReasons     []string `json:"blocking_reasons,omitempty"`
	RecommendedNext     string   `json:"recommended_next_step,omitempty"`
	WaitingOn           []string `json:"waiting_on,omitempty"`
}

func (s *Store) GenerateFeedbackPacket(projectID, cardID string) (*FeedbackPacket, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}

	if card.Status != "needs_input" && card.Status != "blocked" {
		return nil, fmt.Errorf("card must be in needs_input or blocked state to generate feedback packet, current status: %s", card.Status)
	}

	packet := &FeedbackPacket{
		CardID:              card.ID,
		CardTitle:           card.Title,
		ProjectID:           card.ProjectID,
		Status:              card.Status,
		NeedsFeedbackFrom:   card.NeedsFeedbackFrom,
		FeedbackRequest:     card.FeedbackRequest,
		DecisionRequired:    card.DecisionRequired,
		AuthorUpdate:        card.AuthorUpdate,
		AdminActionRequired: card.AdminActionRequired,
		BlockingReasons:     card.BlockingReasons,
		RecommendedNext:     card.RecommendedNext,
		WaitingOn:           card.WaitingOn,
	}

	return packet, nil
}

type FeedbackAnswer struct {
	FeedbackAnswers    []string `json:"feedback_answers,omitempty"`
	DecisionAnswers    []string `json:"decision_answers,omitempty"`
	AuthorUpdates      []string `json:"author_updates,omitempty"`
	AdminActions       []string `json:"admin_actions,omitempty"`
	UnblockReasons     []string `json:"unblock_reasons,omitempty"`
	ResumeTargetStatus string   `json:"resume_target_status,omitempty"`
}

// ExportFeedbackPacket writes a feedback packet to a file for external messaging
func (s *Store) ExportFeedbackPacket(projectID, cardID, path string) (*FeedbackPacket, error) {
	packet, err := s.GenerateFeedbackPacket(projectID, cardID)
	if err != nil {
		return nil, err
	}
	if err := s.writeJSON(path, packet); err != nil {
		return nil, fmt.Errorf("write feedback packet to %s: %w", path, err)
	}
	return packet, nil
}

// ImportFeedbackAnswer reads a feedback answer from a file
func ImportFeedbackAnswer(path string) (*FeedbackAnswer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read feedback answer from %s: %w", path, err)
	}
	var answer FeedbackAnswer
	if err := json.Unmarshal(data, &answer); err != nil {
		return nil, fmt.Errorf("parse feedback answer: %w", err)
	}
	return &answer, nil
}

func (s *Store) ApplyFeedback(projectID, cardID string, answer *FeedbackAnswer) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}

	if card.Status != "needs_input" && card.Status != "blocked" {
		return nil, fmt.Errorf("card must be in needs_input or blocked state to apply feedback, current status: %s", card.Status)
	}

	now := time.Now().UTC()

	if len(answer.FeedbackAnswers) > 0 {
		for _, ans := range answer.FeedbackAnswers {
			card.AuthorUpdate = cleanList(append(card.AuthorUpdate, fmt.Sprintf("[%s] Answer: %s", now.Format(time.RFC3339), ans)))
		}
	}

	if len(answer.DecisionAnswers) > 0 {
		for _, ans := range answer.DecisionAnswers {
			card.AuthorUpdate = cleanList(append(card.AuthorUpdate, fmt.Sprintf("[%s] Decision: %s", now.Format(time.RFC3339), ans)))
		}
	}

	if len(answer.AuthorUpdates) > 0 {
		for _, upd := range answer.AuthorUpdates {
			card.AuthorUpdate = cleanList(append(card.AuthorUpdate, fmt.Sprintf("[%s] Update: %s", now.Format(time.RFC3339), upd)))
		}
	}

	if len(answer.AdminActions) > 0 {
		for _, act := range answer.AdminActions {
			card.AdminActionRequired = cleanList(append(card.AdminActionRequired, fmt.Sprintf("[%s] Action taken: %s", now.Format(time.RFC3339), act)))
		}
	}

	if len(answer.UnblockReasons) > 0 {
		card.BlockingReasons = removeStrings(card.BlockingReasons, answer.UnblockReasons)
	}

	card.NeedsFeedbackFrom = nil
	card.FeedbackRequest = nil
	card.DecisionRequired = nil
	card.WaitingOn = nil

	targetStatus := answer.ResumeTargetStatus
	if targetStatus == "" {
		targetStatus = "clarifying"
	}

	if targetStatus != "clarifying" && targetStatus != "ready" {
		return nil, fmt.Errorf("invalid resume_target_status: %s, must be clarifying or ready", targetStatus)
	}

	if targetStatus == "ready" {
		if err := validateReady(card); err != nil {
			return nil, fmt.Errorf("cannot resume to ready: %w", err)
		}
	}

	card.Status = targetStatus
	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	card.UpdatedAt = now.Format(time.RFC3339)

	if err := s.SaveCard(card); err != nil {
		return nil, err
	}

	return card, nil
}

func removeStrings(from []string, remove []string) []string {
	if len(from) == 0 || len(remove) == 0 {
		return from
	}

	removeSet := make(map[string]struct{})
	for _, s := range remove {
		removeSet[s] = struct{}{}
	}

	result := make([]string, 0, len(from))
	for _, s := range from {
		if _, exists := removeSet[s]; !exists {
			result = append(result, s)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// OutboundMessageEnvelope is a normalized envelope for outbound messages with correlation tracking
type OutboundMessageEnvelope struct {
	CardID        string          `json:"card_id"`
	ProjectID     string          `json:"project_id"`
	CorrelationID string          `json:"correlation_id"`
	TargetRole    string          `json:"target_role"`
	Payload       *FeedbackPacket `json:"payload"`
}

// InboundReplyEnvelope is a normalized envelope for inbound replies with correlation tracking
type InboundReplyEnvelope struct {
	CardID             string   `json:"card_id"`
	ProjectID          string   `json:"project_id"`
	CorrelationID      string   `json:"correlation_id"`
	ReplyText          string   `json:"reply_text,omitempty"`
	Answers            []string `json:"answers,omitempty"`
	ResumeTargetStatus string   `json:"resume_target_status,omitempty"`
}

// GenerateCorrelationID creates a unique correlation ID for message tracking
func GenerateCorrelationID() string {
	return fmt.Sprintf("corr-%d", time.Now().UnixNano())
}

// ExportOutboundMessage creates a correlation-enabled outbound message envelope for a card
func (s *Store) ExportOutboundMessage(projectID, cardID, targetRole string) (*OutboundMessageEnvelope, error) {
	packet, err := s.GenerateFeedbackPacket(projectID, cardID)
	if err != nil {
		return nil, err
	}

	envelope := &OutboundMessageEnvelope{
		CardID:        packet.CardID,
		ProjectID:     packet.ProjectID,
		CorrelationID: GenerateCorrelationID(),
		TargetRole:    targetRole,
		Payload:       packet,
	}

	return envelope, nil
}

// IngestReply processes an inbound reply envelope and routes it to the correct card
func (s *Store) IngestReply(envelope *InboundReplyEnvelope) (*FeatureCard, error) {
	if envelope.CardID == "" || envelope.ProjectID == "" {
		return nil, fmt.Errorf("card_id and project_id are required in reply envelope")
	}

	answer := &FeedbackAnswer{
		ResumeTargetStatus: envelope.ResumeTargetStatus,
	}

	if envelope.ReplyText != "" {
		answer.FeedbackAnswers = []string{envelope.ReplyText}
	}
	if len(envelope.Answers) > 0 {
		answer.FeedbackAnswers = envelope.Answers
	}

	card, err := s.ApplyFeedback(envelope.ProjectID, envelope.CardID, answer)
	if err != nil {
		return nil, fmt.Errorf("apply feedback for card %s: %w", envelope.CardID, err)
	}

	return card, nil
}
