package control

import (
	"fmt"
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
