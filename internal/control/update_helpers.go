package control

import (
	"fmt"
	"strings"
	"time"
)

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
