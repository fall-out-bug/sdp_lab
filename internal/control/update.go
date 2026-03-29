package control

import (
	"fmt"
	"log"
	"strings"
	"time"
)

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
