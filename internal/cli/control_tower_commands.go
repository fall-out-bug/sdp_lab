package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/control"
)

func recommendationForAction(projectID, cardID, action string) actionRecommendation {
	switch action {
	case "surface_feedback_request", "request_human_input":
		return actionRecommendation{command: feedbackCommand(projectID, cardID), reason: "Generate the feedback/resume path for the top waiting card."}
	case "start_execution", "spawn_execution":
		return actionRecommendation{command: dispatchCardCommand(projectID, cardID), reason: "Dispatch the highest-priority ready card directly."}
	case "continue_clarification":
		return actionRecommendation{command: showCommand(projectID, cardID), reason: "Open the top unfinished card and continue shaping it."}
	default:
		return actionRecommendation{}
	}
}

func showCommand(projectID, cardID string) string {
	return fmt.Sprintf("sdp card show --project %s --id %s", projectID, cardID)
}

func feedbackCommand(projectID, cardID string) string {
	return fmt.Sprintf("sdp card feedback --project %s --id %s", projectID, cardID)
}

func readyCommand(projectID, cardID string) string {
	return fmt.Sprintf("sdp card ready --project %s --id %s", projectID, cardID)
}

func dispatchCardCommand(projectID, cardID string) string {
	return fmt.Sprintf("sdp dispatch card --project %s --id %s", projectID, cardID)
}

func heartbeatCommand(projectID, cardID string) string {
	return fmt.Sprintf("sdp card heartbeat --project %s --id %s --session <session-id> --state running --progress \"...\"", projectID, cardID)
}

func dispatchNextCommand() string    { return "sdp dispatch next" }
func orchestrateOnceCommand() string { return "sdp orchestrate once" }

func deliverCommand(projectID, cardID, state string) string {
	return fmt.Sprintf("sdp card deliver --project %s --id %s --state %s", projectID, cardID, state)
}

func recommendCommandsForQueueItem(item control.QueueItem) []actionRecommendation {
	return recommendationsForState(item.ProjectID, item.CardID, item.Status, item.RecommendedNextAction, item.ReviewState, item.DeliveryState, item.HasRollback, len(item.AdminActionRequired) > 0, item.ExecutorResultStatus, item.ExecutorRuntimeState)
}

func recommendCommandsForCardSummary(projectID string, item control.CardSummary) []actionRecommendation {
	return recommendationsForState(projectID, item.ID, item.Status, item.RecommendedNextAction, item.ReviewState, item.DeliveryState, item.HasRollback, len(item.AdminActionRequired) > 0, item.ExecutorResultStatus, item.ExecutorRuntimeState)
}

func recommendCommandsForCard(card *control.FeatureCard) []actionRecommendation {
	executorStatus := ""
	if card.ExecutorResult != nil {
		executorStatus = card.ExecutorResult.Status
	}
	return recommendationsForState(card.ProjectID, card.ID, card.Status, card.RecommendedNextAction, card.ReviewState, card.DeliveryState, card.RollbackRef != "", len(card.AdminActionRequired) > 0, executorStatus, card.ExecutorRuntimeState)
}

func recommendationsForState(projectID, cardID, status, recommendedAction, reviewState, deliveryState string, hasRollback, hasAdminAction bool, executorResultStatus, runtimeState string) []actionRecommendation {
	appendUnique := func(items []actionRecommendation, rec actionRecommendation) []actionRecommendation {
		if rec.command == "" {
			return items
		}
		for _, existing := range items {
			if existing.command == rec.command {
				return items
			}
		}
		return append(items, rec)
	}
	recs := []actionRecommendation{}
	switch status {
	case "needs_input":
		recs = appendUnique(recs, actionRecommendation{command: feedbackCommand(projectID, cardID), reason: "This card is waiting on explicit human/admin input."})
		recs = appendUnique(recs, actionRecommendation{command: showCommand(projectID, cardID), reason: "Open the full card before sending or importing the reply."})
	case "ready":
		recs = appendUnique(recs, actionRecommendation{command: dispatchCardCommand(projectID, cardID), reason: "This card already passed the ready gate."})
		recs = appendUnique(recs, actionRecommendation{command: dispatchNextCommand(), reason: "Use the portfolio dispatcher if you want SDP to pick the next ready card."})
	case "blocked":
		recs = appendUnique(recs, actionRecommendation{command: showCommand(projectID, cardID), reason: "Inspect blockers, result summary, and trace before changing state."})
		if hasAdminAction || recommendedAction == "resolve_blocker" {
			recs = appendUnique(recs, actionRecommendation{command: feedbackCommand(projectID, cardID), reason: "If the blocker needs a human/admin answer, generate the explicit feedback packet."})
		}
	case "clarifying", "inbox":
		recs = appendUnique(recs, actionRecommendation{command: showCommand(projectID, cardID), reason: "Open the full card while you shape the missing intent/scope fields."})
		recs = appendUnique(recs, actionRecommendation{command: readyCommand(projectID, cardID), reason: "Use this once the ready-gate fields are complete."})
	case "executing":
		recs = appendUnique(recs, actionRecommendation{command: showCommand(projectID, cardID), reason: "Check the live execution trace, packet path, and latest result hints."})
		if runtimeState == control.ExecutorRuntimePending || runtimeState == control.ExecutorRuntimeStale || runtimeState == control.ExecutorRuntimeLost {
			recs = appendUnique(recs, actionRecommendation{command: heartbeatCommand(projectID, cardID), reason: "Record or reconcile executor runtime heartbeat so the execution state stays honest."})
		}
		if executorResultStatus != "" {
			recs = appendUnique(recs, actionRecommendation{command: orchestrateOnceCommand(), reason: "Let the orchestrator ingest or react to the latest execution result."})
		}
	case "reviewing":
		recs = appendUnique(recs, actionRecommendation{command: showCommand(projectID, cardID), reason: "Review state is visible on the card detail surface."})
	}
	if reviewState == "needs_attention" || reviewState == "failed" {
		recs = appendUnique(recs, actionRecommendation{command: showCommand(projectID, cardID), reason: "Review feedback is the first thing to inspect before resuming work."})
	}
	if deliveryState == "failed" || deliveryState == "rolled_back" || hasRollback {
		recs = appendUnique(recs, actionRecommendation{command: deliverCommand(projectID, cardID, "rolled_back"), reason: "Record the rollback/delivery outcome explicitly when delivery goes sideways."})
	}
	if len(recs) == 0 {
		recs = appendUnique(recs, actionRecommendation{command: showCommand(projectID, cardID), reason: "Open the detailed control object for the next move."})
	}
	return recs
}

func cardActionLines(card *control.FeatureCard) []string {
	recs := recommendCommandsForCard(card)
	if len(recs) == 0 {
		return nil
	}
	lines := make([]string, 0, len(recs)+1)
	for i, rec := range recs {
		label := "- Primary"
		if i > 0 {
			label = "- Fallback " + strconv.Itoa(i)
		}
		line := label + ": `" + rec.command + "`"
		if rec.reason != "" {
			line += " — " + rec.reason
		}
		lines = append(lines, line)
	}
	return lines
}

func renderPortfolioNextActionCommands(next map[string]string) []string {
	projectID := strings.TrimSpace(next["target_project_id"])
	cardID := strings.TrimSpace(next["target_card_id"])
	if projectID == "" || cardID == "" {
		return nil
	}
	rec := recommendationForAction(projectID, cardID, next["recommended"])
	if rec.command == "" {
		return nil
	}
	lines := []string{"  Command: `" + rec.command + "`"}
	if rec.reason != "" {
		lines = append(lines, "  Why this command: "+rec.reason)
	}
	return lines
}

func renderProjectNextActionCommands(projectID string, next map[string]string) []string {
	cardID := strings.TrimSpace(next["target_card_id"])
	if projectID == "" || cardID == "" {
		return nil
	}
	rec := recommendationForAction(projectID, cardID, next["recommended"])
	if rec.command == "" {
		return nil
	}
	lines := []string{"  Command: `" + rec.command + "`"}
	if rec.reason != "" {
		lines = append(lines, "  Why this command: "+rec.reason)
	}
	return lines
}

func renderItemActionLines(item control.QueueItem, max int) []string {
	recs := recommendCommandsForQueueItem(item)
	if len(recs) == 0 {
		return nil
	}
	if max > 0 && len(recs) > max {
		recs = recs[:max]
	}
	lines := make([]string, 0, len(recs))
	for i, rec := range recs {
		prefix := fmt.Sprintf("- %s/%s", item.ProjectID, item.CardID)
		if i == 0 {
			prefix += " — primary: `" + rec.command + "`"
		} else {
			prefix += " — fallback: `" + rec.command + "`"
		}
		if rec.reason != "" {
			prefix += " | why: " + rec.reason
		}
		lines = append(lines, prefix)
	}
	return lines
}

func renderCardSummaryActionLines(projectID string, item control.CardSummary, max int) []string {
	recs := recommendCommandsForCardSummary(projectID, item)
	if len(recs) == 0 {
		return nil
	}
	if max > 0 && len(recs) > max {
		recs = recs[:max]
	}
	lines := make([]string, 0, len(recs))
	for i, rec := range recs {
		prefix := fmt.Sprintf("- %s/%s", projectID, item.ID)
		if i == 0 {
			prefix += " — primary: `" + rec.command + "`"
		} else {
			prefix += " — fallback: `" + rec.command + "`"
		}
		if rec.reason != "" {
			prefix += " | why: " + rec.reason
		}
		lines = append(lines, prefix)
	}
	return lines
}
