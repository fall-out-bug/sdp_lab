package cli

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/control"
)

func RenderProjectBoard(snap *control.ProjectBoardSnapshot) string {
	var b strings.Builder
	projectID := snap.Project["project_id"]
	name := snap.Project["name"]
	if name == "" {
		name = projectID
	}

	fmt.Fprintf(&b, "PROJECT BOARD — %s\n", name)
	fmt.Fprintf(&b, "State: attention %d | waiting %d | blocked %d | ready %d | executing %d | done %d\n",
		snap.Counts["needs_input"], snap.Counts["clarifying"]+snap.Counts["inbox"], snap.Counts["blocked"], snap.Counts["ready"], snap.Counts["executing"], snap.Counts["done"])
	b.WriteString("\n")

	for _, section := range projectSections(snap) {
		b.WriteString(section.title + "\n")
		for _, line := range section.lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Next action\n")
	fmt.Fprintf(&b, "- %s\n", humanizeRecommendation(snap.NextAction["recommended"]))
	if reason := strings.TrimSpace(snap.NextAction["reason"]); reason != "" {
		b.WriteString("  " + reason + "\n")
	}
	if target := strings.TrimSpace(snap.NextAction["target_card_id"]); target != "" {
		b.WriteString("  Target: " + projectID + "/" + target + "\n")
	}
	for _, line := range renderProjectNextActionCommands(projectID, snap.NextAction) {
		b.WriteString(line + "\n")
	}
	if lines := renderProjectActionSurface(snap); len(lines) > 0 {
		b.WriteString("\nAction surface\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
	}

	return strings.TrimSpace(b.String())
}

func RenderPortfolioBoard(snap *control.PortfolioBoardSnapshot) string {
	var b strings.Builder
	b.WriteString("CONTROL TOWER\n")
	fmt.Fprintf(&b, "Attention now: %d | Waiting on human: %d | Blocked: %d | Ready: %d | Executing: %d | Done: %d\n",
		snap.Totals["needs_input"]+snap.Totals["blocked"], len(snap.Queues["waiting_on_human"]), len(snap.Queues["blocked"]), len(snap.Queues["ready_to_execute"]), snap.Totals["executing"], snap.Totals["done"])
	b.WriteString("\n")

	for _, section := range portfolioSections(snap) {
		b.WriteString(section.title + "\n")
		for _, line := range section.lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Projects\n")
	for _, line := range summarizeProjects(snap) {
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	b.WriteString("Next action\n")
	fmt.Fprintf(&b, "- %s\n", humanizeRecommendation(snap.NextAction["recommended"]))
	if reason := strings.TrimSpace(snap.NextAction["reason"]); reason != "" {
		b.WriteString("  " + reason + "\n")
	}
	if projectID := strings.TrimSpace(snap.NextAction["target_project_id"]); projectID != "" {
		target := projectID
		if cardID := strings.TrimSpace(snap.NextAction["target_card_id"]); cardID != "" {
			target += "/" + cardID
		}
		b.WriteString("  Target: " + target + "\n")
	}
	for _, line := range renderPortfolioNextActionCommands(snap.NextAction) {
		b.WriteString(line + "\n")
	}

	return strings.TrimSpace(b.String())
}

func RenderAttention(snap *control.PortfolioBoardSnapshot) string {
	var b strings.Builder
	b.WriteString("EXECUTIVE ATTENTION\n")
	fmt.Fprintf(&b, "Attention now: %d | Waiting on human: %d | Blocked: %d | Movement: %d | Delivery trouble: %d | Ready to move: %d\n",
		snap.Executive.AttentionNowCount,
		snap.Executive.WaitingOnHumanCount,
		snap.Executive.BlockedCount,
		snap.Executive.MovementCount,
		snap.Executive.DeliveryTroubleCount,
		snap.Executive.ReadyToMoveCount,
	)
	b.WriteString("\n")

	sections := []renderedSection{
		{title: fmt.Sprintf("Attention now (%d)", snap.Executive.AttentionNowCount), lines: renderQueueItems(snap.Executive.AttentionNow, "Nothing needs immediate operator attention.")},
		{title: fmt.Sprintf("Waiting on human (%d)", snap.Executive.WaitingOnHumanCount), lines: renderQueueItems(snap.Executive.WaitingOnHuman, "Nothing is waiting on a human right now.")},
		{title: fmt.Sprintf("Blocked (%d)", snap.Executive.BlockedCount), lines: renderQueueItems(snap.Executive.Blocked, "No blocked cards right now.")},
		{title: fmt.Sprintf("Movement (%d)", snap.Executive.MovementCount), lines: renderQueueItems(snap.Executive.Movement, "Nothing is actively moving right now.")},
		{title: fmt.Sprintf("Delivery trouble (%d)", snap.Executive.DeliveryTroubleCount), lines: renderQueueItems(snap.Executive.DeliveryTrouble, "No delivery trouble right now.")},
		{title: fmt.Sprintf("Ready to move (%d)", snap.Executive.ReadyToMoveCount), lines: renderQueueItems(snap.Executive.ReadyToMove, "No ready cards right now.")},
		{title: fmt.Sprintf("Friction hotspots (%d)", len(snap.Executive.FrictionHotspots)), lines: renderQueueItems(snap.Executive.FrictionHotspots, "No friction hotspots right now.")},
	}
	for _, section := range sections {
		b.WriteString(section.title + "\n")
		for _, line := range section.lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Next best action\n")
	fmt.Fprintf(&b, "- %s\n", humanizeRecommendation(snap.NextAction["recommended"]))
	if reason := strings.TrimSpace(snap.NextAction["reason"]); reason != "" {
		b.WriteString("  " + reason + "\n")
	}
	if projectID := strings.TrimSpace(snap.NextAction["target_project_id"]); projectID != "" {
		target := projectID
		if cardID := strings.TrimSpace(snap.NextAction["target_card_id"]); cardID != "" {
			target += "/" + cardID
		}
		b.WriteString("  Target: " + target + "\n")
	}
	for _, line := range renderPortfolioNextActionCommands(snap.NextAction) {
		b.WriteString(line + "\n")
	}
	if lines := renderExecutiveActionSurface(snap); len(lines) > 0 {
		b.WriteString("\nTop commands\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func renderExecutiveActionSurface(snap *control.PortfolioBoardSnapshot) []string {
	groups := [][]control.QueueItem{
		snap.Executive.AttentionNow,
		snap.Executive.ReadyToMove,
		snap.Executive.DeliveryTrouble,
		snap.Executive.Movement,
	}
	seen := map[string]bool{}
	lines := []string{}
	for _, group := range groups {
		for _, item := range group {
			key := item.ProjectID + "/" + item.CardID
			if seen[key] {
				continue
			}
			seen[key] = true
			lines = append(lines, renderItemActionLines(item, 1)...)
			if len(lines) >= 3 {
				return lines[:3]
			}
		}
	}
	return lines
}

func renderProjectActionSurface(snap *control.ProjectBoardSnapshot) []string {
	groups := [][]control.CardSummary{snap.Columns["needs_input"], snap.Columns["blocked"], snap.Columns["ready"], snap.Columns["executing"]}
	lines := []string{}
	for _, group := range groups {
		for _, item := range group {
			lines = append(lines, renderCardSummaryActionLines(snap.Project["project_id"], item, 1)...)
			if len(lines) >= 4 {
				return lines[:4]
			}
		}
	}
	return lines
}

func portfolioSections(snap *control.PortfolioBoardSnapshot) []renderedSection {
	return []renderedSection{
		{title: fmt.Sprintf("Needs human input (%d)", len(snap.Queues["waiting_on_human"])), lines: renderQueueItems(snap.Queues["waiting_on_human"], "Nothing is waiting on a human right now.")},
		{title: fmt.Sprintf("Blocked (%d)", len(snap.Queues["blocked"])), lines: renderQueueItems(snap.Queues["blocked"], "No blocked cards right now.")},
		{title: fmt.Sprintf("Ready to move (%d)", len(snap.Queues["ready_to_execute"])), lines: renderQueueItems(snap.Queues["ready_to_execute"], "No ready cards right now.")},
	}
}

func projectSections(snap *control.ProjectBoardSnapshot) []renderedSection {
	return []renderedSection{
		{title: fmt.Sprintf("Needs human input (%d)", len(snap.Columns["needs_input"])), lines: renderCardSummaries(snap.Project["project_id"], snap.Columns["needs_input"], "No cards are waiting on feedback.")},
		{title: fmt.Sprintf("Blocked (%d)", len(snap.Columns["blocked"])), lines: renderCardSummaries(snap.Project["project_id"], snap.Columns["blocked"], "No blocked cards.")},
		{title: fmt.Sprintf("Ready to execute (%d)", len(snap.Columns["ready"])), lines: renderCardSummaries(snap.Project["project_id"], snap.Columns["ready"], "No ready cards.")},
		{title: fmt.Sprintf("Executing now (%d)", len(snap.Columns["executing"])), lines: renderCardSummaries(snap.Project["project_id"], snap.Columns["executing"], "Nothing is executing right now.")},
	}
}

func renderQueueItems(items []control.QueueItem, empty string) []string {
	if len(items) == 0 {
		return []string{"- " + empty}
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		line := fmt.Sprintf("- %s/%s — %s", item.ProjectID, item.CardID, item.Title)
		detail := queueDetail(item)
		if detail != "" {
			line += " | " + detail
		}
		lines = append(lines, line)
	}
	return lines
}

func renderCardSummaries(projectID string, items []control.CardSummary, empty string) []string {
	if len(items) == 0 {
		return []string{"- " + empty}
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		line := fmt.Sprintf("- %s/%s — %s", projectID, item.ID, item.Title)
		detail := cardSummaryDetail(item)
		if detail != "" {
			line += " | " + detail
		}
		lines = append(lines, line)
	}
	return lines
}

func queueDetail(item control.QueueItem) string {
	parts := []string{}
	if item.RecommendedNextStep != "" {
		parts = append(parts, "next: "+item.RecommendedNextStep)
	}
	if item.RecommendedNextReason != "" {
		parts = append(parts, "why: "+item.RecommendedNextReason)
	}
	if item.LastOrchestratorAction != "" {
		parts = append(parts, "last: "+humanizeAction(item.LastOrchestratorAction))
	}
	if len(item.NeedsFeedbackFrom) > 0 {
		parts = append(parts, "waiting on: "+strings.Join(item.NeedsFeedbackFrom, ", "))
	}
	if len(item.AdminActionRequired) > 0 {
		parts = append(parts, "admin: "+strings.Join(item.AdminActionRequired, ", "))
	}
	if len(item.AuthorUpdate) > 0 {
		parts = append(parts, "update: "+strings.Join(item.AuthorUpdate, ", "))
	}
	if review := reviewHint(item.ReviewState); review != "" {
		parts = append(parts, review)
	}
	if delivery := deliveryHint(item.DeliveryState, item.DeliveryTarget, item.HasRollback, item.HasFollowup); delivery != "" {
		parts = append(parts, delivery)
	}
	if friction := frictionMarkers(item.ClarificationCycles, item.BlockedCycles, item.ExecutionAttemptCount, item.ReviewFailCount, item.RollbackCount); friction != "" {
		parts = append(parts, friction)
	}
	if hint := executionHint(item.LinkedBeadsIDs, item.DispatchedTo, item.ExecutorSessionID, item.LastExecutorHeartbeatAt, item.ExecutorRuntimeState, item.ExecutorProgressSummary, item.ExecutorResultStatus, item.ExecutorResultSummary, item.ExecutorNextHint); hint != "" {
		parts = append(parts, hint)
	}
	return strings.Join(parts, " | ")
}

func cardSummaryDetail(item control.CardSummary) string {
	parts := []string{}
	if item.RecommendedNextStep != "" {
		parts = append(parts, "next: "+item.RecommendedNextStep)
	}
	if item.RecommendedNextReason != "" {
		parts = append(parts, "why: "+item.RecommendedNextReason)
	}
	if item.LastOrchestratorAction != "" {
		parts = append(parts, "last: "+humanizeAction(item.LastOrchestratorAction))
	}
	if len(item.NeedsFeedbackFrom) > 0 {
		parts = append(parts, "waiting on: "+strings.Join(item.NeedsFeedbackFrom, ", "))
	}
	if review := reviewHint(item.ReviewState); review != "" {
		parts = append(parts, review)
	}
	if delivery := deliveryHint(item.DeliveryState, item.DeliveryTarget, item.HasRollback, item.HasFollowup); delivery != "" {
		parts = append(parts, delivery)
	}
	if friction := frictionMarkers(item.ClarificationCycles, item.BlockedCycles, item.ExecutionAttemptCount, item.ReviewFailCount, item.RollbackCount); friction != "" {
		parts = append(parts, friction)
	}
	if hint := executionHint(item.LinkedBeadsIDs, item.DispatchedTo, item.ExecutorSessionID, item.LastExecutorHeartbeatAt, item.ExecutorRuntimeState, item.ExecutorProgressSummary, item.ExecutorResultStatus, item.ExecutorResultSummary, item.ExecutorNextHint); hint != "" {
		parts = append(parts, hint)
	}
	return strings.Join(parts, " | ")
}

func summarizeProjects(snap *control.PortfolioBoardSnapshot) []string {
	if len(snap.Projects) == 0 {
		return []string{"- No registered projects."}
	}
	projects := append([]map[string]any(nil), snap.Projects...)
	slices.SortFunc(projects, func(a, b map[string]any) int {
		return cmp.Compare(projectUrgencyScore(b), projectUrgencyScore(a))
	})
	lines := make([]string, 0, len(projects))
	for _, proj := range projects {
		projectID, _ := proj["project_id"].(string)
		counts, _ := proj["counts"].(map[string]any)
		lines = append(lines, fmt.Sprintf("- %s — attention %d | blocked %d | ready %d | executing %d", projectID,
			intFromAny(counts["needs_input"])+intFromAny(counts["blocked"]),
			intFromAny(counts["blocked"]),
			intFromAny(counts["ready"]),
			intFromAny(counts["executing"])))
	}
	return lines
}

func projectUrgencyScore(project map[string]any) int {
	counts, _ := project["counts"].(map[string]any)
	return intFromAny(counts["needs_input"])*100 + intFromAny(counts["blocked"])*10 + intFromAny(counts["ready"])
}
