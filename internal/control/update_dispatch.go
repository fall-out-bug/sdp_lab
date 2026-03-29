package control

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
		return fmt.Errorf("create dispatch dir: %w", err)
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
