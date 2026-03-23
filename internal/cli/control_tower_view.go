package cli

import (
	"fmt"
	"sort"
	"strings"

	"sdp_dev/internal/control"
)

func RenderDoctorControl(report *control.DoctorReport) string {
	var b strings.Builder
	issues := append([]control.DoctorCheck(nil), report.Checks...)
	sortDoctorChecks(issues)

	warnings := 0
	errors := 0
	byCheck := map[string]int{}
	for _, check := range issues {
		byCheck[check.CheckID]++
		if check.Severity == "error" {
			errors++
		} else {
			warnings++
		}
	}

	b.WriteString("DOCTOR CONTROL\n")
	b.WriteString(fmt.Sprintf("Checks: %d total | %d passed | %d issues\n", report.TotalChecks, report.Passed, report.Failed))

	if len(issues) == 0 {
		b.WriteString("Status: healthy\n")
		b.WriteString("Next action: none — control store looks clean.\n")
		return strings.TrimSpace(b.String())
	}

	b.WriteString(fmt.Sprintf("Status: action needed (%d errors, %d warnings)\n", errors, warnings))
	b.WriteString("Next action: fix errors first, then clear the oldest/stalest warnings.\n")
	b.WriteString("\nIssue groups:\n")
	for _, line := range summarizeCheckCounts(byCheck) {
		b.WriteString("- " + line + "\n")
	}

	b.WriteString("\nTop issues:\n")
	for _, check := range issues {
		b.WriteString(fmt.Sprintf("- [%s] %s", strings.ToUpper(check.Severity), humanizeCheckID(check.CheckID)))
		if check.ProjectID != "" || check.CardID != "" {
			b.WriteString(" — ")
			if check.ProjectID != "" {
				b.WriteString(check.ProjectID)
			}
			if check.CardID != "" {
				if check.ProjectID != "" {
					b.WriteString("/")
				}
				b.WriteString(check.CardID)
			}
		}
		b.WriteString("\n")
		b.WriteString("  " + check.Message + "\n")
		b.WriteString("  Next: " + recommendedDoctorAction(check) + "\n")
	}

	return strings.TrimSpace(b.String())
}

func RenderProjectBoard(snap *control.ProjectBoardSnapshot) string {
	var b strings.Builder
	projectID := snap.Project["project_id"]
	name := snap.Project["name"]
	if name == "" {
		name = projectID
	}

	b.WriteString(fmt.Sprintf("PROJECT BOARD — %s\n", name))
	b.WriteString(fmt.Sprintf("State: attention %d | waiting %d | blocked %d | ready %d | executing %d | done %d\n",
		snap.Counts["needs_input"], snap.Counts["clarifying"]+snap.Counts["inbox"], snap.Counts["blocked"], snap.Counts["ready"], snap.Counts["executing"], snap.Counts["done"]))
	b.WriteString("\n")

	for _, section := range projectSections(snap) {
		b.WriteString(section.title + "\n")
		for _, line := range section.lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Next action\n")
	b.WriteString(fmt.Sprintf("- %s\n", humanizeRecommendation(snap.NextAction["recommended"])))
	if reason := strings.TrimSpace(snap.NextAction["reason"]); reason != "" {
		b.WriteString("  " + reason + "\n")
	}
	if target := strings.TrimSpace(snap.NextAction["target_card_id"]); target != "" {
		b.WriteString("  Target: " + projectID + "/" + target + "\n")
	}

	return strings.TrimSpace(b.String())
}

func RenderPortfolioBoard(snap *control.PortfolioBoardSnapshot) string {
	var b strings.Builder
	b.WriteString("CONTROL TOWER\n")
	b.WriteString(fmt.Sprintf("Attention now: %d | Waiting on human: %d | Blocked: %d | Ready: %d | Executing: %d | Done: %d\n",
		snap.Totals["needs_input"]+snap.Totals["blocked"], len(snap.Queues["waiting_on_human"]), len(snap.Queues["blocked"]), len(snap.Queues["ready_to_execute"]), snap.Totals["executing"], snap.Totals["done"]))
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
	b.WriteString(fmt.Sprintf("- %s\n", humanizeRecommendation(snap.NextAction["recommended"])))
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

	return strings.TrimSpace(b.String())
}

func RenderAttention(snap *control.PortfolioBoardSnapshot) string {
	var b strings.Builder
	b.WriteString("ATTENTION\n")
	b.WriteString(fmt.Sprintf("Now: %d needs input/blocker items | %d ready | %d executing\n", len(snap.Queues["waiting_on_human"])+len(snap.Queues["blocked"]), len(snap.Queues["ready_to_execute"]), snap.Totals["executing"]))
	b.WriteString("\n")

	for _, section := range portfolioSections(snap) {
		b.WriteString(section.title + "\n")
		for _, line := range section.lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Next action\n")
	b.WriteString(fmt.Sprintf("- %s\n", humanizeRecommendation(snap.NextAction["recommended"])))
	if reason := strings.TrimSpace(snap.NextAction["reason"]); reason != "" {
		b.WriteString("  " + reason + "\n")
	}
	return strings.TrimSpace(b.String())
}

type renderedSection struct {
	title string
	lines []string
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
	if len(item.NeedsFeedbackFrom) > 0 {
		parts = append(parts, "waiting on: "+strings.Join(item.NeedsFeedbackFrom, ", "))
	}
	if len(item.AdminActionRequired) > 0 {
		parts = append(parts, "admin: "+strings.Join(item.AdminActionRequired, ", "))
	}
	if len(item.AuthorUpdate) > 0 {
		parts = append(parts, "update: "+strings.Join(item.AuthorUpdate, ", "))
	}
	return strings.Join(parts, " | ")
}

func cardSummaryDetail(item control.CardSummary) string {
	parts := []string{}
	if item.RecommendedNextStep != "" {
		parts = append(parts, "next: "+item.RecommendedNextStep)
	}
	if len(item.NeedsFeedbackFrom) > 0 {
		parts = append(parts, "waiting on: "+strings.Join(item.NeedsFeedbackFrom, ", "))
	}
	if len(item.LinkedBeadsIDs) > 0 {
		parts = append(parts, "beads: "+strings.Join(item.LinkedBeadsIDs, ", "))
	}
	return strings.Join(parts, " | ")
}

func summarizeProjects(snap *control.PortfolioBoardSnapshot) []string {
	if len(snap.Projects) == 0 {
		return []string{"- No registered projects."}
	}
	projects := append([]map[string]any(nil), snap.Projects...)
	sort.Slice(projects, func(i, j int) bool {
		return projectUrgencyScore(projects[i]) > projectUrgencyScore(projects[j])
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

func sortDoctorChecks(checks []control.DoctorCheck) {
	sort.Slice(checks, func(i, j int) bool {
		if severityRank(checks[i].Severity) != severityRank(checks[j].Severity) {
			return severityRank(checks[i].Severity) < severityRank(checks[j].Severity)
		}
		if checks[i].CheckID != checks[j].CheckID {
			return checks[i].CheckID < checks[j].CheckID
		}
		if checks[i].ProjectID != checks[j].ProjectID {
			return checks[i].ProjectID < checks[j].ProjectID
		}
		return checks[i].CardID < checks[j].CardID
	})
}

func severityRank(severity string) int {
	if severity == "error" {
		return 0
	}
	return 1
}

func summarizeCheckCounts(byCheck map[string]int) []string {
	keys := make([]string, 0, len(byCheck))
	for key := range byCheck {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %d", humanizeCheckID(key), byCheck[key]))
	}
	return lines
}

func humanizeCheckID(id string) string {
	return strings.ReplaceAll(id, "-", " ")
}

func recommendedDoctorAction(check control.DoctorCheck) string {
	scope := check.ProjectID
	if check.CardID != "" {
		scope += "/" + check.CardID
	}
	switch check.CheckID {
	case "missing-intake-artifact", "intake-artifact-not-found":
		return "restore or regenerate the intake artifact so the card has traceable intake context."
	case "ready-gate-missing":
		return "finish clarification, fill the ready-gate fields, then mark the card ready again."
	case "executing-without-beads":
		return "re-dispatch or repair Beads linkage before trusting this execution state."
	case "executing-without-dispatch-metadata":
		return "write dispatched_at / dispatched_to / dispatched_packet_path so operators can trace execution."
	case "needs-input-without-questions":
		return "add explicit feedback questions or decisions so the human knows what to answer."
	case "stale-ready-card":
		return "dispatch this ready card or park it if it should not move yet."
	case "stale-needs-input-card":
		return "nudge the requested human/admin and capture the missing answer."
	case "stale-blocked-card":
		return "make the blocking reason explicit and decide whether to unblock, escalate, or park it."
	case "done-without-result-summary":
		return "ingest or write the executor result summary so completion is auditable."
	default:
		if scope != "" {
			return "inspect " + scope + " and clear this hygiene issue."
		}
		return "inspect the affected card and clear this hygiene issue."
	}
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
