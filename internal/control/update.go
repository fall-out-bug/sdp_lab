package control

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	ExecutorRuntimePending   = "pending"
	ExecutorRuntimeRunning   = "running"
	ExecutorRuntimeStale     = "stale"
	ExecutorRuntimeLost      = "lost"
	ExecutorRuntimeCompleted = "completed"
	ExecutorRuntimeFailed    = "failed"
)

func validExecutorRuntimeState(state string) bool {
	switch state {
	case ExecutorRuntimePending, ExecutorRuntimeRunning, ExecutorRuntimeStale, ExecutorRuntimeLost, ExecutorRuntimeCompleted:
		return true
	default:
		return false
	}
}

func (s *Store) RecordExecutorHeartbeat(projectID, cardID, sessionID, runtimeState, progress string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}
	if card.Status != "executing" {
		return nil, fmt.Errorf("card must be executing to record heartbeat, current status: %s", card.Status)
	}
	runtimeState = strings.TrimSpace(runtimeState)
	if runtimeState == "" {
		runtimeState = ExecutorRuntimeRunning
	}
	if !validExecutorRuntimeState(runtimeState) {
		return nil, fmt.Errorf("invalid executor runtime state: %s", runtimeState)
	}
	now := time.Now().UTC()
	if sid := strings.TrimSpace(sessionID); sid != "" {
		card.ExecutorSessionID = sid
		if card.ExecutorStartedAt == "" {
			card.ExecutorStartedAt = now.Format(time.RFC3339)
		}
	}
	card.LastExecutorHeartbeatAt = now.Format(time.RFC3339)
	card.ExecutorRuntimeState = runtimeState
	if progress := strings.TrimSpace(progress); progress != "" {
		card.ExecutorProgressSummary = progress
	}
	setOrchestratorTrace(card, "recorded_executor_heartbeat", "Recorded a manual/interim executor heartbeat for runtime reconciliation", "await_executor_result", "Keep watching runtime heartbeat freshness until a result arrives", now)
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

func setOrchestratorTrace(card *FeatureCard, action, reason, recommendedAction, recommendedReason string, at time.Time) {
	if card == nil {
		return
	}
	card.LastOrchestratorAction = action
	card.LastOrchestratorReason = reason
	card.LastOrchestratorAt = at.Format(time.RFC3339)
	card.RecommendedNextAction = recommendedAction
	card.RecommendedNextReason = recommendedReason
}

func incrementCycleOnStatusEntry(card *FeatureCard, fromStatus, toStatus string) {
	if card == nil || fromStatus == toStatus {
		return
	}
	switch toStatus {
	case "clarifying":
		card.ClarificationCycles++
	case "blocked":
		card.BlockedCycles++
	case "executing":
		card.ExecutionAttemptCount++
	}
}

func maybeIncrementReviewFailCount(card *FeatureCard, result *ExecutorResultPacket) {
	if card == nil || result == nil {
		return
	}
	if result.ExecutorRole == string(ExecutorRoleReview) && (result.Status == ResultStatusNeedsReview || result.Status == ResultStatusFailed) {
		card.ReviewFailCount++
	}
}

// updateReviewTrace sets explicit review trace fields when result comes from review role
func updateReviewTrace(card *FeatureCard, result *ExecutorResultPacket) {
	if card == nil || result == nil {
		return
	}
	if result.ExecutorRole != string(ExecutorRoleReview) {
		return
	}
	// Map result status to review state
	switch result.Status {
	case ResultStatusSuccess:
		card.ReviewState = "passed"
	case ResultStatusNeedsReview:
		card.ReviewState = "needs_attention"
	case ResultStatusFailed:
		card.ReviewState = "failed"
	case ResultStatusBlocked:
		card.ReviewState = "blocked"
	case ResultStatusNeedsInput:
		card.ReviewState = "needs_input"
	}
	if result.Summary != "" {
		card.ReviewSummary = result.Summary
	}
	// Store first artifact reference as review ref if available
	if len(result.Artifacts) > 0 && result.Artifacts[0].Reference != "" {
		card.ReviewRef = result.Artifacts[0].Reference
	}
}

func (s *Store) LoadCard(projectID, cardID string) (*FeatureCard, error) {
	return s.cardRepo().LoadCard(projectID, cardID)
}

func (s *Store) ClarifyCard(projectID, cardID, normalizedIntent, taskType, targetRepo, riskLevel, nextStep string, scopeIn, scopeOut []string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}
	prevStatus := card.Status
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
	now := time.Now().UTC()
	incrementCycleOnStatusEntry(card, prevStatus, card.Status)
	setOrchestratorTrace(card, "clarified_card", "Orchestrator captured the current shape of the request", "continue_clarification", "Keep shaping until the card can be marked ready or needs explicit input", now)
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
	prevStatus := card.Status
	card.Status = "needs_input"
	card.NeedsFeedbackFrom = cleanList(needsFeedbackFrom)
	card.FeedbackRequest = cleanList(feedbackRequest)
	card.DecisionRequired = cleanList(decisionRequired)
	card.AuthorUpdate = cleanList(authorUpdate)
	card.AdminActionRequired = cleanList(adminActionRequired)
	card.WaitingOn = ensureContains(card.WaitingOn, "human")
	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	now := time.Now().UTC()
	incrementCycleOnStatusEntry(card, prevStatus, card.Status)
	setOrchestratorTrace(card, "requested_input", "The orchestrator needs external clarification or approval before advancing", "await_human_input", "A human or admin reply is required to resume the card", now)
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
	prevStatus := card.Status
	card.Status = "ready"
	card.WaitingOn = nil
	card.NeedsFeedbackFrom = nil
	card.FeedbackRequest = nil
	card.DecisionRequired = nil
	card.AdminActionRequired = nil
	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	now := time.Now().UTC()
	incrementCycleOnStatusEntry(card, prevStatus, card.Status)
	setOrchestratorTrace(card, "marked_ready", "The card satisfies the current ready gate", "dispatch_execution", "The card is shaped enough to hand to an executor", now)
	if err := s.SaveCard(card); err != nil {
		return nil, err
	}
	if contractPath, err := s.writeGeneratedContract(card); err != nil {
		log.Printf("warning: failed to auto-generate task contract for %s: %v", card.ID, err)
	} else {
		card.RequiredArtifacts = cleanList(append(card.RequiredArtifacts, contractPath))
		if err := s.SaveCard(card); err != nil {
			log.Printf("warning: failed to persist generated contract path for %s: %v", card.ID, err)
		}
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

	prevStatus := card.Status
	card.LinkedBeadsIDs = cleanList(append(card.LinkedBeadsIDs, beadsID))
	card.Status = "executing"
	card.WaitingOn = nil
	card.ActiveAgents = ensureContains(card.ActiveAgents, "executor")
	now := time.Now().UTC()
	incrementCycleOnStatusEntry(card, prevStatus, card.Status)
	setOrchestratorTrace(card, "linked_execution", "The card was linked to Beads execution work", "dispatch_execution", "Write and send the execution packet to the selected executor", now)
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

	if card.Status == "ready" {
		card, err = s.ExecuteCard(projectID, cardID)
		if err != nil {
			return nil, fmt.Errorf("link execution before dispatch: %w", err)
		}
	}

	packet, err := s.BuildExecutionPacket(projectID, cardID)
	if err != nil {
		return nil, fmt.Errorf("build execution packet: %w", err)
	}

	if err := s.writeDispatchPacket(projectID, cardID, packet); err != nil {
		return nil, fmt.Errorf("write dispatch packet: %w", err)
	}

	prevStatus := card.Status
	now := time.Now().UTC()
	card.Status = "executing"
	card.DispatchedAt = now.Format(time.RFC3339)
	card.DispatchedTo = packet.ExecutorRole
	card.DispatchedPacketPath = s.dispatchPacketPath(projectID, cardID)
	card.ExecutorRuntimeState = ExecutorRuntimePending
	card.ExecutorProgressSummary = "Dispatch packet created; awaiting first executor heartbeat"
	card.ActiveAgents = ensureContains(card.ActiveAgents, "executor")
	incrementCycleOnStatusEntry(card, prevStatus, card.Status)
	setOrchestratorTrace(card, "dispatched_execution", "The orchestrator produced an execution packet and routed the card to an executor", "await_executor_result", "Wait for executor output or a follow-up result packet", now)

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

	prevStatus := card.Status
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
	incrementCycleOnStatusEntry(card, prevStatus, card.Status)
	if targetStatus == "ready" {
		setOrchestratorTrace(card, "resumed_after_feedback", "Feedback resolved the outstanding questions and the card can proceed", "dispatch_execution", "The card can return to execution", now)
	} else {
		setOrchestratorTrace(card, "resumed_after_feedback", "Feedback resolved the current blocker and the card returned to clarification", "continue_clarification", "Incorporate the new answers into shaping", now)
	}
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

func (s *Store) IngestExecutorResult(result *ExecutorResultPacket) (*FeatureCard, error) {
	if result == nil {
		return nil, fmt.Errorf("nil result packet")
	}

	if result.ParentFeatureID == "" {
		return nil, fmt.Errorf("parent_feature_id is required in result packet")
	}

	card, err := s.LoadCardByID(result.ParentFeatureID)
	if err != nil {
		return nil, fmt.Errorf("load card %s: %w", result.ParentFeatureID, err)
	}

	prevStatus := card.Status
	now := time.Now().UTC()

	summary := &ExecutorResultSummary{
		Status:              string(result.Status),
		Summary:             result.Summary,
		ReceivedAt:          now.Format(time.RFC3339),
		Findings:            result.Findings,
		OpenRisks:           result.OpenRisks,
		RecommendedNextStep: result.RecommendedNextStep,
	}

	if len(result.Artifacts) > 0 {
		artifactRefs := make([]string, 0, len(result.Artifacts))
		for _, art := range result.Artifacts {
			artifactRefs = append(artifactRefs, fmt.Sprintf("%s: %s", art.Type, art.Reference))
		}
		summary.Artifacts = artifactRefs

		linkedArtifacts := make([]string, 0, len(card.LinkedArtifacts)+len(result.Artifacts))
		for _, art := range card.LinkedArtifacts {
			if art != "" {
				linkedArtifacts = append(linkedArtifacts, art)
			}
		}
		for _, art := range result.Artifacts {
			if art.Reference != "" {
				linkedArtifacts = append(linkedArtifacts, art.Reference)
			}
		}
		card.LinkedArtifacts = linkedArtifacts
	}

	card.ExecutorResult = summary
	card.ExecutorRuntimeState = ExecutorRuntimeCompleted
	if card.LastExecutorHeartbeatAt == "" {
		card.LastExecutorHeartbeatAt = now.Format(time.RFC3339)
	}
	if card.ExecutorProgressSummary == "" && strings.TrimSpace(result.Summary) != "" {
		card.ExecutorProgressSummary = result.Summary
	}
	maybeIncrementReviewFailCount(card, result)
	// Update review trace fields when result comes from review role
	updateReviewTrace(card, result)

	switch result.Status {
	case ResultStatusSuccess:
		card.Status = "done"
		card.ActiveAgents = removeAgent(card.ActiveAgents, "executor")
		card.WaitingOn = nil

	case ResultStatusBlocked:
		card.Status = "blocked"
		if len(result.Findings) > 0 {
			card.BlockingReasons = cleanList(append(card.BlockingReasons, result.Findings...))
		}
		if len(result.OpenRisks) > 0 {
			card.OpenQuestions = cleanList(append(card.OpenQuestions, result.OpenRisks...))
		}
		card.WaitingOn = []string{"orchestrator"}

	case ResultStatusNeedsReview:
		card.Status = "needs_input"
		if result.Summary != "" {
			card.FeedbackRequest = cleanList(append(card.FeedbackRequest, result.Summary))
		}
		if len(result.Findings) > 0 {
			card.DecisionRequired = cleanList(append(card.DecisionRequired, result.Findings...))
		}
		card.NeedsFeedbackFrom = []string{"human", "admin"}
		card.WaitingOn = []string{"human"}

	case ResultStatusNeedsInput:
		card.Status = "needs_input"
		if result.Summary != "" {
			card.FeedbackRequest = cleanList(append(card.FeedbackRequest, result.Summary))
		}
		if len(result.Findings) > 0 {
			card.AuthorUpdate = cleanList(append(card.AuthorUpdate, result.Findings...))
		}
		card.NeedsFeedbackFrom = []string{"human"}
		card.WaitingOn = []string{"human"}

	case ResultStatusFailed:
		card.Status = "clarifying"
		if result.Summary != "" {
			card.AuthorUpdate = cleanList(append(card.AuthorUpdate, fmt.Sprintf("[%s] Failed: %s", now.Format(time.RFC3339), result.Summary)))
		}
		if len(result.Findings) > 0 {
			card.OpenQuestions = cleanList(append(card.OpenQuestions, result.Findings...))
		}
		card.ActiveAgents = removeAgent(card.ActiveAgents, "executor")
		card.WaitingOn = []string{"orchestrator"}

	default:
		return nil, fmt.Errorf("unknown result status: %s", result.Status)
	}

	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	incrementCycleOnStatusEntry(card, prevStatus, card.Status)
	switch result.Status {
	case ResultStatusSuccess:
		setOrchestratorTrace(card, "ingested_executor_result", "Execution completed successfully and the card can be closed", "none", "No immediate follow-up is required", now)
	case ResultStatusBlocked:
		setOrchestratorTrace(card, "ingested_executor_result", "Execution reported a real blocker that requires orchestration", "resolve_blocker", "Review the blocker details and decide how to unblock the card", now)
	case ResultStatusNeedsReview:
		setOrchestratorTrace(card, "ingested_executor_result", "Execution surfaced a review or approval loop", "request_review_input", "A human or admin decision is needed before continuing", now)
	case ResultStatusNeedsInput:
		setOrchestratorTrace(card, "ingested_executor_result", "Execution needs more clarification before it can continue", "request_human_input", "Collect the missing answers and resume the card", now)
	case ResultStatusFailed:
		setOrchestratorTrace(card, "ingested_executor_result", "Execution failed and the card returned to clarification", "replan_execution", "Incorporate the failure findings before another attempt", now)
	}
	card.UpdatedAt = now.Format(time.RFC3339)

	if err := s.SaveCard(card); err != nil {
		return nil, fmt.Errorf("save card: %w", err)
	}

	if _, err := s.BuildProjectSnapshot(card.ProjectID); err != nil {
		return nil, fmt.Errorf("update project snapshot: %w", err)
	}

	if _, err := s.BuildPortfolioSnapshot(); err != nil {
		return nil, fmt.Errorf("update portfolio snapshot: %w", err)
	}

	return card, nil
}

func removeAgent(agents []string, toRemove string) []string {
	result := make([]string, 0, len(agents))
	for _, agent := range agents {
		if agent != toRemove {
			result = append(result, agent)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// DeliveryState values for recording delivery outcomes
const (
	DeliveryStatePending    = "pending"
	DeliveryStateDeployed   = "deployed"
	DeliveryStateFailed     = "failed"
	DeliveryStateRolledBack = "rolled_back"
)

// RecordDelivery records a delivery outcome for a card.
// This is a thin honest method - it only writes what is explicitly provided.
func (s *Store) RecordDelivery(projectID, cardID, state, target, summary, ref string, followupRefs []string) (*FeatureCard, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	// Update delivery fields
	if state != "" {
		card.DeliveryState = state
	}
	if target != "" {
		card.DeliveryTarget = target
	}
	if summary != "" {
		card.DeliverySummary = summary
	}
	if ref != "" {
		card.DeliveryRef = ref
	}

	// If delivery state is deployed, set delivered_at
	if state == DeliveryStateDeployed && card.DeliveredAt == "" {
		card.DeliveredAt = now.Format(time.RFC3339)
	}

	// If delivery state is rolled_back, update rollback fields
	if state == DeliveryStateRolledBack {
		card.RollbackCount++
		if summary != "" {
			card.RollbackSummary = summary
		}
		if ref != "" {
			card.RollbackRef = ref
		}
	}

	// Add follow-up refs if provided
	if len(followupRefs) > 0 {
		card.FollowupRefs = cleanList(append(card.FollowupRefs, followupRefs...))
	}

	card.ActiveAgents = ensureContains(card.ActiveAgents, "orchestrator")
	setOrchestratorTrace(card, "recorded_delivery", fmt.Sprintf("Recorded delivery outcome: %s", state), "continue", "Delivery outcome recorded", now)
	card.UpdatedAt = now.Format(time.RFC3339)

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

// DispatchResult is a summary of a single dispatch operation
type DispatchResult struct {
	// Success indicates whether dispatch succeeded
	Success bool `json:"success"`

	// Message is a human-readable summary of what happened
	Message string `json:"message"`

	// ProjectID is the project containing the dispatched card
	ProjectID string `json:"project_id,omitempty"`

	// CardID is the ID of the dispatched card
	CardID string `json:"card_id,omitempty"`

	// CardTitle is the title of the dispatched card
	CardTitle string `json:"card_title,omitempty"`

	// ExecutorRole is the role the card was dispatched to
	ExecutorRole string `json:"executor_role,omitempty"`

	// PacketPath is the path to the written dispatch packet
	PacketPath string `json:"packet_path,omitempty"`

	// NoDispatchableReason is set when nothing could be dispatched
	NoDispatchableReason string `json:"no_dispatchable_reason,omitempty"`
}

// SelectDispatchableCard selects one dispatchable card from all projects.
// It checks in order: ready cards, executing cards (for re-dispatch), then returns nil if none.
func (s *Store) SelectDispatchableCard() (*FeatureCard, error) {
	portfolio, err := s.BuildPortfolioSnapshot()
	if err != nil {
		return nil, fmt.Errorf("build portfolio: %w", err)
	}

	for _, item := range portfolio.Queues["ready_to_execute"] {
		card, err := s.LoadCard(item.ProjectID, item.CardID)
		if err != nil {
			continue
		}
		return card, nil
	}

	for _, proj := range portfolio.Projects {
		projectID, ok := proj["project_id"].(string)
		if !ok {
			continue
		}
		projSnap, err := s.BuildProjectSnapshot(projectID)
		if err != nil {
			continue
		}
		for _, cardSum := range projSnap.Columns["executing"] {
			card, err := s.LoadCard(projectID, cardSum.ID)
			if err != nil {
				continue
			}
			return card, nil
		}
	}

	return nil, nil
}

// DispatchNext performs one orchestration step: selects dispatchable card, dispatches it using existing logic, returns summary.
// If nothing is dispatchable, returns a clear no-op result.
func (s *Store) DispatchNext() (*DispatchResult, error) {
	card, err := s.SelectDispatchableCard()
	if err != nil {
		return nil, fmt.Errorf("select dispatchable card: %w", err)
	}

	if card == nil {
		result := &DispatchResult{
			Success:              false,
			Message:              "No dispatchable cards found",
			NoDispatchableReason: "No cards in ready or executing state across all projects",
		}
		return result, nil
	}

	resultCard, err := s.DispatchCard(card.ProjectID, card.ID)
	if err != nil {
		return &DispatchResult{
			Success: false,
			Message: fmt.Sprintf("Failed to dispatch card %s/%s: %v", card.ProjectID, card.ID, err),
		}, nil
	}

	result := &DispatchResult{
		Success:      true,
		Message:      fmt.Sprintf("Dispatched card [%s/%s] %s to %s", card.ProjectID, card.ID, card.Title, resultCard.DispatchedTo),
		ProjectID:    card.ProjectID,
		CardID:       card.ID,
		CardTitle:    card.Title,
		ExecutorRole: resultCard.DispatchedTo,
		PacketPath:   resultCard.DispatchedPacketPath,
	}

	return result, nil
}
