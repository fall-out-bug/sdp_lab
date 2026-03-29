package cli

import (
	"fmt"
	"strings"
)

type renderedSection struct {
	title string
	lines []string
}

type actionRecommendation struct {
	command string
	reason  string
}

func valueOrFallback(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func humanizeAction(action string) string {
	if action == "" {
		return ""
	}
	return strings.ReplaceAll(action, "_", " ")
}

func compactList(label string, items []string) string {
	if len(items) == 0 {
		return ""
	}
	return label + ": " + strings.Join(items, "; ")
}

func reviewHint(state string) string {
	if strings.TrimSpace(state) == "" {
		return ""
	}
	return "review: " + state
}

func deliveryHint(state, target string, hasRollback, hasFollowup bool) string {
	parts := []string{}
	if strings.TrimSpace(state) != "" {
		hint := "delivery: " + state
		if strings.TrimSpace(target) != "" {
			hint += " -> " + target
		}
		parts = append(parts, hint)
	}
	if hasRollback {
		parts = append(parts, "rollback recorded")
	}
	if hasFollowup {
		parts = append(parts, "follow-up linked")
	}
	return strings.Join(parts, " | ")
}

func frictionMarkers(clarify, blocked, execCount, review, rollback int) string {
	parts := []string{}
	if clarify > 0 {
		parts = append(parts, fmt.Sprintf("clarify:%d", clarify))
	}
	if blocked > 0 {
		parts = append(parts, fmt.Sprintf("blocked:%d", blocked))
	}
	if execCount > 0 {
		parts = append(parts, fmt.Sprintf("exec:%d", execCount))
	}
	if review > 0 {
		parts = append(parts, fmt.Sprintf("review:%d", review))
	}
	if rollback > 0 {
		parts = append(parts, fmt.Sprintf("rollback:%d", rollback))
	}
	if len(parts) == 0 {
		return ""
	}
	return "friction: " + strings.Join(parts, ", ")
}

func executionHint(beads []string, dispatchedTo, sessionID, heartbeatAt, runtimeState, progress, resultStatus, resultSummary, resultNext string) string {
	parts := []string{}
	if len(beads) > 0 {
		parts = append(parts, "beads: "+strings.Join(beads, ", "))
	}
	if dispatchedTo != "" {
		parts = append(parts, "dispatch: "+dispatchedTo)
	}
	if runtimeState != "" {
		parts = append(parts, "runtime: "+runtimeState)
	}
	if sessionID != "" {
		parts = append(parts, "session: "+sessionID)
	}
	if heartbeatAt != "" {
		parts = append(parts, "hb: "+heartbeatAt)
	}
	if progress != "" {
		parts = append(parts, "progress: "+progress)
	}
	if resultStatus != "" {
		hint := "result: " + resultStatus
		if resultSummary != "" {
			hint += " — " + resultSummary
		}
		parts = append(parts, hint)
	}
	if resultNext != "" {
		parts = append(parts, "result-next: "+resultNext)
	}
	return strings.Join(parts, " | ")
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func humanizeCheckID(id string) string {
	return strings.ReplaceAll(id, "-", " ")
}

func humanizeRecommendation(action string) string {
	switch action {
	case "surface_feedback_request", "request_human_input":
		return "Get the missing human/admin input"
	case "start_execution", "spawn_execution":
		return "Move the next ready card into execution"
	case "continue_clarification":
		return "Continue shaping the next unfinished card"
	case "idle":
		return "No immediate move needed"
	default:
		return strings.ReplaceAll(action, "_", " ")
	}
}
