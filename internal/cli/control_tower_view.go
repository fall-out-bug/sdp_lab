package cli

import (
	"fmt"
	"html/template"
	"sort"
	"strconv"
	"strings"

	"sdp_dev/internal/control"
)

type canonicalOwner struct {
	Flow   string
	Owner  string
	Detail string
}

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
	for _, line := range renderPortfolioNextActionCommands(snap.NextAction) {
		b.WriteString(line + "\n")
	}

	return strings.TrimSpace(b.String())
}

func RenderAttention(snap *control.PortfolioBoardSnapshot) string {
	var b strings.Builder
	b.WriteString("EXECUTIVE ATTENTION\n")
	b.WriteString(fmt.Sprintf("Attention now: %d | Waiting on human: %d | Blocked: %d | Movement: %d | Delivery trouble: %d | Ready to move: %d\n",
		snap.Executive.AttentionNowCount,
		snap.Executive.WaitingOnHumanCount,
		snap.Executive.BlockedCount,
		snap.Executive.MovementCount,
		snap.Executive.DeliveryTroubleCount,
		snap.Executive.ReadyToMoveCount,
	))
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

func canonicalOwners(card *control.FeatureCard) []canonicalOwner {
	owners := []canonicalOwner{{Flow: "Project", Owner: card.ProjectID, Detail: "Feature card and project board are the canonical project surface."}}
	sessionOwner := strings.TrimSpace(card.DispatchedPacketPath)
	if sessionOwner == "" {
		sessionOwner = strings.TrimSpace(card.DispatchedTo)
	}
	sessionDetail := "Dispatch packet is not recorded yet."
	if sessionOwner != "" {
		sessionDetail = "Dispatch/runtime handoff is anchored here."
	}
	owners = append(owners, canonicalOwner{Flow: "Session", Owner: valueOrFallback(sessionOwner, "unassigned"), Detail: sessionDetail})
	markdownOwner := ""
	if len(card.IntakeArtifact) > 0 {
		markdownOwner = card.IntakeArtifact[0]
	} else if len(card.SourceRefs) > 0 {
		markdownOwner = strings.Join(card.SourceRefs, ", ")
	}
	owners = append(owners, canonicalOwner{Flow: "Markdown", Owner: valueOrFallback(markdownOwner, "missing intake artifact"), Detail: "Intake markdown is the canonical authored request surface."})
	agentOwner := strings.TrimSpace(card.DispatchedTo)
	if agentOwner == "" && len(card.ActiveAgents) > 0 {
		agentOwner = strings.Join(card.ActiveAgents, ", ")
	}
	owners = append(owners, canonicalOwner{Flow: "Agents", Owner: valueOrFallback(agentOwner, "not dispatched"), Detail: "Dispatch target / active agents own execution."})
	artifactRefs := []string{}
	artifactRefs = append(artifactRefs, card.LinkedArtifacts...)
	if card.ReviewRef != "" {
		artifactRefs = append(artifactRefs, card.ReviewRef)
	}
	if card.DeliveryRef != "" {
		artifactRefs = append(artifactRefs, card.DeliveryRef)
	}
	if card.RollbackRef != "" {
		artifactRefs = append(artifactRefs, card.RollbackRef)
	}
	artifactOwner := ""
	if len(artifactRefs) > 0 {
		artifactOwner = strings.Join(artifactRefs, ", ")
	}
	owners = append(owners, canonicalOwner{Flow: "Artifacts / materials", Owner: valueOrFallback(artifactOwner, "none linked"), Detail: "Linked artifacts and delivery/review refs are the canonical proof surface."})
	return owners
}

func valueOrFallback(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func RenderCardDetail(card *control.FeatureCard) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("CARD — %s\n", card.Title))
	b.WriteString(fmt.Sprintf("ID: %s/%s | Status: %s", card.ProjectID, card.ID, card.Status))
	if risk := strings.TrimSpace(card.RiskLevel); risk != "" {
		b.WriteString(" | Risk: " + risk)
	}
	b.WriteString("\n")

	if raw := strings.TrimSpace(card.RawRequest); raw != "" {
		b.WriteString("Request\n")
		b.WriteString("- " + raw + "\n\n")
	}

	if len(card.SourceRefs) > 0 || card.NormalizedIntent != "" || len(card.ScopeIn) > 0 || len(card.ScopeOut) > 0 {
		b.WriteString("Shape\n")
		for _, line := range cardShapeLines(card) {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("Canonical owners\n")
	for _, owner := range canonicalOwners(card) {
		b.WriteString(fmt.Sprintf("- %s: %s — %s\n", owner.Flow, owner.Owner, owner.Detail))
	}
	b.WriteString("\n")

	b.WriteString("Control\n")
	for _, line := range cardControlLines(card) {
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	if lines := cardExecutionLines(card); len(lines) > 0 {
		b.WriteString("Execution\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if lines := cardReviewLines(card); len(lines) > 0 {
		b.WriteString("Review\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if lines := cardDeliveryLines(card); len(lines) > 0 {
		b.WriteString("Delivery\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if lines := cardRollbackLines(card); len(lines) > 0 {
		b.WriteString("Rollback\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if lines := cardFrictionLines(card); len(lines) > 0 {
		b.WriteString("Friction\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	if lines := cardActionLines(card); len(lines) > 0 {
		b.WriteString("Action surface\n")
		for _, line := range lines {
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}

type renderedSection struct {
	title string
	lines []string
}

type actionRecommendation struct {
	command string
	reason  string
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
			for _, line := range renderItemActionLines(item, 1) { //nolint:gosimple
				lines = append(lines, line)
			}
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
			for _, line := range renderCardSummaryActionLines(snap.Project["project_id"], item, 1) { //nolint:gosimple
				lines = append(lines, line)
			}
			if len(lines) >= 4 {
				return lines[:4]
			}
		}
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

func cardShapeLines(card *control.FeatureCard) []string {
	lines := []string{}
	if len(card.SourceRefs) > 0 {
		lines = append(lines, "- Source: "+strings.Join(card.SourceRefs, ", "))
	}
	if intent := strings.TrimSpace(card.NormalizedIntent); intent != "" {
		lines = append(lines, "- Intent: "+intent)
	}
	if scope := compactList("Scope in", card.ScopeIn); scope != "" {
		lines = append(lines, "- "+scope)
	}
	if scope := compactList("Scope out", card.ScopeOut); scope != "" {
		lines = append(lines, "- "+scope)
	}
	return lines
}

func cardControlLines(card *control.FeatureCard) []string {
	lines := []string{}
	if card.LastOrchestratorAction != "" {
		line := "- Last orchestrator: " + humanizeAction(card.LastOrchestratorAction)
		if reason := strings.TrimSpace(card.LastOrchestratorReason); reason != "" {
			line += " — " + reason
		}
		if at := strings.TrimSpace(card.LastOrchestratorAt); at != "" {
			line += " (" + at + ")"
		}
		lines = append(lines, line)
	}
	if card.RecommendedNextAction != "" {
		line := "- Next: " + humanizeAction(card.RecommendedNextAction)
		if reason := strings.TrimSpace(card.RecommendedNextReason); reason != "" {
			line += " — " + reason
		}
		lines = append(lines, line)
	}
	if len(card.WaitingOn) > 0 {
		lines = append(lines, "- Waiting on: "+strings.Join(card.WaitingOn, ", "))
	}
	if len(card.BlockingReasons) > 0 {
		lines = append(lines, "- Blockers: "+strings.Join(card.BlockingReasons, "; "))
	}
	if len(card.NeedsFeedbackFrom) > 0 {
		lines = append(lines, "- Feedback from: "+strings.Join(card.NeedsFeedbackFrom, ", "))
	}
	if len(lines) == 0 {
		return []string{"- No control metadata yet."}
	}
	return lines
}

func cardExecutionLines(card *control.FeatureCard) []string {
	lines := []string{}
	if len(card.LinkedBeadsIDs) > 0 {
		lines = append(lines, "- Beads: "+strings.Join(card.LinkedBeadsIDs, ", "))
	}
	if card.DispatchedTo != "" || card.DispatchedAt != "" {
		line := "- Dispatch"
		if card.DispatchedTo != "" {
			line += ": " + card.DispatchedTo
		}
		if card.DispatchedAt != "" {
			line += " @ " + card.DispatchedAt
		}
		lines = append(lines, line)
	}
	if card.DispatchedPacketPath != "" {
		lines = append(lines, "- Packet: "+card.DispatchedPacketPath)
	}
	if card.ExecutorRuntimeState != "" || card.ExecutorSessionID != "" || card.ExecutorStartedAt != "" || card.LastExecutorHeartbeatAt != "" || card.ExecutorProgressSummary != "" {
		if card.ExecutorRuntimeState != "" {
			lines = append(lines, "- Runtime: "+card.ExecutorRuntimeState)
		}
		if card.ExecutorSessionID != "" {
			lines = append(lines, "- Session: "+card.ExecutorSessionID)
		}
		if card.ExecutorStartedAt != "" {
			lines = append(lines, "- Started: "+card.ExecutorStartedAt)
		}
		if card.LastExecutorHeartbeatAt != "" {
			lines = append(lines, "- Last heartbeat: "+card.LastExecutorHeartbeatAt)
		}
		if card.ExecutorProgressSummary != "" {
			lines = append(lines, "- Progress: "+card.ExecutorProgressSummary)
		}
	}
	if result := card.ExecutorResult; result != nil {
		line := "- Result: " + result.Status
		if result.Summary != "" {
			line += " — " + result.Summary
		}
		lines = append(lines, line)
		if result.RecommendedNextStep != "" {
			lines = append(lines, "- Result next: "+result.RecommendedNextStep)
		}
		if len(result.Findings) > 0 {
			lines = append(lines, "- Findings: "+strings.Join(result.Findings, "; "))
		}
		if len(result.OpenRisks) > 0 {
			lines = append(lines, "- Open risks: "+strings.Join(result.OpenRisks, "; "))
		}
	}
	if card.ReviewState != "" || card.ReviewSummary != "" || card.ReviewRef != "" {
		line := "- Review"
		if card.ReviewState != "" {
			line += ": " + card.ReviewState
		}
		if card.ReviewSummary != "" {
			line += " — " + card.ReviewSummary
		}
		lines = append(lines, line)
		if card.ReviewRef != "" {
			lines = append(lines, "- Review ref: "+card.ReviewRef)
		}
	}
	if card.DeliveryState != "" || card.DeliveryTarget != "" || card.DeliverySummary != "" || card.DeliveryRef != "" {
		line := "- Delivery"
		if card.DeliveryState != "" {
			line += ": " + card.DeliveryState
		}
		if card.DeliveryTarget != "" {
			line += " -> " + card.DeliveryTarget
		}
		if card.DeliverySummary != "" {
			line += " — " + card.DeliverySummary
		}
		lines = append(lines, line)
		if card.DeliveryRef != "" {
			lines = append(lines, "- Delivery ref: "+card.DeliveryRef)
		}
		if card.DeliveredAt != "" {
			lines = append(lines, "- Delivered at: "+card.DeliveredAt)
		}
	}
	if card.RollbackRef != "" || card.RollbackSummary != "" || len(card.FollowupRefs) > 0 {
		line := "- Rollback"
		if card.RollbackSummary != "" {
			line += ": " + card.RollbackSummary
		}
		lines = append(lines, line)
		if card.RollbackRef != "" {
			lines = append(lines, "- Rollback ref: "+card.RollbackRef)
		}
		if len(card.FollowupRefs) > 0 {
			lines = append(lines, "- Follow-up refs: "+strings.Join(card.FollowupRefs, ", "))
		}
	}
	return lines
}

func cardFrictionLines(card *control.FeatureCard) []string {
	lines := []string{}
	if card.ClarificationCycles > 0 {
		lines = append(lines, fmt.Sprintf("- clarification_cycles: %d", card.ClarificationCycles))
	}
	if card.BlockedCycles > 0 {
		lines = append(lines, fmt.Sprintf("- blocked_cycles: %d", card.BlockedCycles))
	}
	if card.ExecutionAttemptCount > 0 {
		lines = append(lines, fmt.Sprintf("- execution_attempt_count: %d", card.ExecutionAttemptCount))
	}
	if card.ReviewFailCount > 0 {
		lines = append(lines, fmt.Sprintf("- review_fail_count: %d", card.ReviewFailCount))
	}
	if card.RollbackCount > 0 {
		lines = append(lines, fmt.Sprintf("- rollback_count: %d", card.RollbackCount))
	}
	return lines
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

func humanizeAction(action string) string {
	if action == "" {
		return ""
	}
	return strings.ReplaceAll(action, "_", " ")
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
	case "executing-without-session":
		return "record the executor session id once a real runtime exists, or reconcile why execution is still only pending."
	case "executing-without-heartbeat":
		return "record the first executor heartbeat or relaunch/reconcile the runtime if nothing actually started."
	case "stale-executor-heartbeat":
		return "refresh the executor heartbeat or mark the runtime lost if the session is gone."
	case "executing-runtime-lost":
		return "either relaunch the executor and heartbeat it, or move the card out of executing so the board stops lying."
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

func cardReviewLines(card *control.FeatureCard) []string {
	lines := []string{}
	if card.ReviewState != "" {
		lines = append(lines, "- State: "+card.ReviewState)
	}
	if card.ReviewSummary != "" {
		lines = append(lines, "- Summary: "+card.ReviewSummary)
	}
	if card.ReviewRef != "" {
		lines = append(lines, "- Ref: "+card.ReviewRef)
	}
	return lines
}

func cardDeliveryLines(card *control.FeatureCard) []string {
	lines := []string{}
	if card.DeliveryState != "" {
		lines = append(lines, "- State: "+card.DeliveryState)
	}
	if card.DeliveryTarget != "" {
		lines = append(lines, "- Target: "+card.DeliveryTarget)
	}
	if card.DeliverySummary != "" {
		lines = append(lines, "- Summary: "+card.DeliverySummary)
	}
	if card.DeliveryRef != "" {
		lines = append(lines, "- Ref: "+card.DeliveryRef)
	}
	if card.DeliveredAt != "" {
		lines = append(lines, "- Delivered at: "+card.DeliveredAt)
	}
	return lines
}

func cardRollbackLines(card *control.FeatureCard) []string {
	lines := []string{}
	if card.RollbackRef != "" {
		lines = append(lines, "- Rollback ref: "+card.RollbackRef)
	}
	if card.RollbackSummary != "" {
		lines = append(lines, "- Rollback summary: "+card.RollbackSummary)
	}
	if len(card.FollowupRefs) > 0 {
		lines = append(lines, "- Follow-ups: "+strings.Join(card.FollowupRefs, ", "))
	}
	return lines
}

func RenderCardDetailHTML(card *control.FeatureCard) string {
	text := RenderCardDetail(card)
	return `<!doctype html><html><head><meta charset="utf-8"><title>` + template.HTMLEscapeString(card.ID) + `</title><style>body{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;background:#0b1020;color:#e5e7eb;margin:0;padding:24px}main{max-width:1100px;margin:0 auto}pre{white-space:pre-wrap;line-height:1.45;background:#111827;border:1px solid #374151;border-radius:12px;padding:20px}.muted{color:#9ca3af;margin-bottom:12px}</style></head><body><main><div class="muted">SDP control tower card detail</div><pre>` + template.HTMLEscapeString(text) + `</pre></main></body></html>`
}
