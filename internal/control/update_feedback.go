package control

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

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
