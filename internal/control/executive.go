package control

import (
	"sort"
	"strings"
)

type ExecutiveSummary struct {
	AttentionNowCount    int               `json:"attention_now_count"`
	WaitingOnHumanCount  int               `json:"waiting_on_human_count"`
	BlockedCount         int               `json:"blocked_count"`
	MovementCount        int               `json:"movement_count"`
	DeliveryTroubleCount int               `json:"delivery_trouble_count"`
	ReadyToMoveCount     int               `json:"ready_to_move_count"`
	AttentionNow         []QueueItem       `json:"attention_now,omitempty"`
	WaitingOnHuman       []QueueItem       `json:"waiting_on_human,omitempty"`
	Blocked              []QueueItem       `json:"blocked,omitempty"`
	Movement             []QueueItem       `json:"movement,omitempty"`
	DeliveryTrouble      []QueueItem       `json:"delivery_trouble,omitempty"`
	ReadyToMove          []QueueItem       `json:"ready_to_move,omitempty"`
	FrictionHotspots     []QueueItem       `json:"friction_hotspots,omitempty"`
	NextAction           map[string]string `json:"next_action,omitempty"`
}

func newQueueItem(projectID string, c CardSummary) QueueItem {
	return QueueItem{
		ProjectID:              projectID,
		CardID:                 c.ID,
		Title:                  c.Title,
		Status:                 c.Status,
		RecommendedNextStep:    c.RecommendedNextStep,
		RecommendedNextAction:  c.RecommendedNextAction,
		RecommendedNextReason:  c.RecommendedNextReason,
		LastOrchestratorAction: c.LastOrchestratorAction,
		LastOrchestratorReason: c.LastOrchestratorReason,
		ClarificationCycles:    c.ClarificationCycles,
		BlockedCycles:          c.BlockedCycles,
		ExecutionAttemptCount:  c.ExecutionAttemptCount,
		ReviewFailCount:        c.ReviewFailCount,
		RollbackCount:          c.RollbackCount,
		ActiveAgents:           c.ActiveAgents,
		NeedsFeedbackFrom:      c.NeedsFeedbackFrom,
		AuthorUpdate:           c.AuthorUpdate,
		AdminActionRequired:    c.AdminActionRequired,
		LinkedBeadsIDs:         c.LinkedBeadsIDs,
		DispatchedTo:           c.DispatchedTo,
		ExecutorResultStatus:   c.ExecutorResultStatus,
		ExecutorResultSummary:  c.ExecutorResultSummary,
		ExecutorNextHint:       c.ExecutorNextHint,
		ReviewState:            c.ReviewState,
		DeliveryState:          c.DeliveryState,
		DeliveryTarget:         c.DeliveryTarget,
		RollbackRef:            c.RollbackRef,
		FollowupRefs:           c.FollowupRefs,
		HasRollback:            c.HasRollback,
		HasFollowup:            c.HasFollowup,
	}
}

func deriveExecutiveSummary(queues map[string][]QueueItem, nextAction map[string]string) ExecutiveSummary {
	return ExecutiveSummary{
		AttentionNowCount:    len(queues["attention_now"]),
		WaitingOnHumanCount:  len(queues["waiting_on_human"]),
		BlockedCount:         len(queues["blocked"]),
		MovementCount:        len(queues["movement"]),
		DeliveryTroubleCount: len(queues["delivery_trouble"]),
		ReadyToMoveCount:     len(queues["ready_to_execute"]),
		AttentionNow:         limitQueue(queues["attention_now"], 5),
		WaitingOnHuman:       limitQueue(queues["waiting_on_human"], 5),
		Blocked:              limitQueue(queues["blocked"], 5),
		Movement:             limitQueue(queues["movement"], 5),
		DeliveryTrouble:      limitQueue(queues["delivery_trouble"], 5),
		ReadyToMove:          limitQueue(queues["ready_to_execute"], 5),
		FrictionHotspots:     limitQueue(queues["friction_hotspots"], 5),
		NextAction:           nextAction,
	}
}

func limitQueue(items []QueueItem, n int) []QueueItem {
	if len(items) <= n {
		return append([]QueueItem(nil), items...)
	}
	return append([]QueueItem(nil), items[:n]...)
}

func sortExecutiveQueues(queues map[string][]QueueItem) {
	for key := range queues {
		sort.SliceStable(queues[key], func(i, j int) bool {
			return queuePriority(queues[key][i]) > queuePriority(queues[key][j])
		})
	}
}

func queuePriority(item QueueItem) int {
	score := frictionScore(item)*10 + baseStatusScore(item.Status)
	if needsHumanAttention(item) {
		score += 1000
	}
	if isDeliveryTrouble(item) {
		score += 700
	}
	if item.Status == "blocked" {
		score += 500
	}
	if item.Status == "reviewing" || strings.Contains(item.ReviewState, "fail") || strings.Contains(item.ReviewState, "attention") {
		score += 250
	}
	return score
}

func baseStatusScore(status string) int {
	switch status {
	case "needs_input":
		return 90
	case "blocked":
		return 80
	case "reviewing":
		return 70
	case "executing":
		return 60
	case "ready":
		return 50
	case "done":
		return 40
	default:
		return 0
	}
}

func frictionScore(item QueueItem) int {
	return item.ClarificationCycles + item.BlockedCycles + item.ExecutionAttemptCount + item.ReviewFailCount*2 + item.RollbackCount*3
}

func needsHumanAttention(item QueueItem) bool {
	return item.Status == "needs_input" || len(item.NeedsFeedbackFrom) > 0 || len(item.AdminActionRequired) > 0 || len(item.AuthorUpdate) > 0
}

func isDeliveryTrouble(item QueueItem) bool {
	return item.DeliveryState == "failed" || item.DeliveryState == "rolled_back" || item.HasRollback || item.HasFollowup
}

func inMovement(item QueueItem) bool {
	if item.Status == "executing" || item.Status == "reviewing" {
		return true
	}
	switch item.DeliveryState {
	case "pending", "deployed":
		return true
	default:
		return false
	}
}

func needsAttentionNow(item QueueItem) bool {
	return needsHumanAttention(item) || item.Status == "blocked" || isDeliveryTrouble(item) || item.Status == "reviewing" || strings.Contains(item.ReviewState, "fail") || strings.Contains(item.ReviewState, "attention")
}
