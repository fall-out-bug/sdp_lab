package cli

import (
	"fmt"
	"html/template"
	"strings"

	"sdp_dev/internal/control"
)

type canonicalOwner struct {
	Flow   string
	Owner  string
	Detail string
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

func RenderCardDetailHTML(card *control.FeatureCard) string {
	text := RenderCardDetail(card)
	return `<!doctype html><html><head><meta charset="utf-8"><title>` + template.HTMLEscapeString(card.ID) + `</title><style>body{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;background:#0b1020;color:#e5e7eb;margin:0;padding:24px}main{max-width:1100px;margin:0 auto}pre{white-space:pre-wrap;line-height:1.45;background:#111827;border:1px solid #374151;border-radius:12px;padding:20px}.muted{color:#9ca3af;margin-bottom:12px}</style></head><body><main><div class="muted">SDP control tower card detail</div><pre>` + template.HTMLEscapeString(text) + `</pre></main></body></html>`
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
