package cli

import (
	"strings"
	"testing"

	"sdp_dev/internal/control"
)

func TestRenderDoctorControl(t *testing.T) {
	report := &control.DoctorReport{
		TotalChecks: 4,
		Passed:      2,
		Failed:      2,
		Checks: []control.DoctorCheck{
			{CheckID: "stale-ready-card", Severity: "warning", Message: "ready card has been idle for more than 72h", ProjectID: "alpha", CardID: "feature-alpha-001"},
			{CheckID: "ready-gate-missing", Severity: "error", Message: "ready card fails ready gate: missing normalized_intent", ProjectID: "alpha", CardID: "feature-alpha-002"},
		},
	}

	out := RenderDoctorControl(report)
	for _, want := range []string{
		"Status: action needed (1 errors, 1 warnings)",
		"Issue groups:",
		"- ready gate missing: 1",
		"- stale ready card: 1",
		"Next: finish clarification, fill the ready-gate fields, then mark the card ready again.",
		"Next: dispatch this ready card or park it if it should not move yet.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q\nfull output:\n%s", want, out)
		}
	}
}

func TestRenderPortfolioBoard(t *testing.T) {
	snap := &control.PortfolioBoardSnapshot{
		Totals: map[string]int{"needs_input": 1, "blocked": 1, "ready": 2, "executing": 1, "done": 3},
		Queues: map[string][]control.QueueItem{
			"waiting_on_human": {{ProjectID: "alpha", CardID: "feature-alpha-001", Title: "Need decision", RecommendedNextStep: "ask owner", RecommendedNextReason: "Need product input", LastOrchestratorAction: "requested_input", NeedsFeedbackFrom: []string{"author"}, ClarificationCycles: 2, ExecutorResultStatus: "needs_input", ExecutorResultSummary: "Contract still ambiguous", ReviewState: "needs_attention"}},
			"blocked":          {{ProjectID: "beta", CardID: "feature-beta-002", Title: "Blocked task", RecommendedNextStep: "resolve blocker", LastOrchestratorAction: "ingested_executor_result", BlockedCycles: 1, LinkedBeadsIDs: []string{"bd-77"}, DeliveryState: "failed", DeliveryTarget: "staging", HasRollback: true, HasFollowup: true}},
			"ready_to_execute": {{ProjectID: "alpha", CardID: "feature-alpha-003", Title: "Ready task", RecommendedNextStep: "dispatch", LastOrchestratorAction: "marked_ready", ExecutionAttemptCount: 1}},
		},
		Projects: []map[string]any{
			{"project_id": "alpha", "counts": map[string]any{"needs_input": 1, "blocked": 0, "ready": 2, "executing": 1}},
			{"project_id": "beta", "counts": map[string]any{"needs_input": 0, "blocked": 1, "ready": 0, "executing": 0}},
		},
		NextAction: map[string]string{"recommended": "surface_feedback_request", "reason": "At least one card needs human/admin input", "target_project_id": "alpha", "target_card_id": "feature-alpha-001"},
	}

	out := RenderPortfolioBoard(snap)
	for _, want := range []string{
		"CONTROL TOWER",
		"Needs human input (1)",
		"- alpha/feature-alpha-001 — Need decision | next: ask owner | why: Need product input | last: requested input | waiting on: author | review: needs_attention | friction: clarify:2 | result: needs_input — Contract still ambiguous",
		"- beta/feature-beta-002 — Blocked task | next: resolve blocker | last: ingested executor result | delivery: failed -> staging | rollback recorded | follow-up linked | friction: blocked:1 | beads: bd-77",
		"Projects",
		"- alpha — attention 1 | blocked 0 | ready 2 | executing 1",
		"Target: alpha/feature-alpha-001",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q\nfull output:\n%s", want, out)
		}
	}
}

func TestRenderProjectBoard(t *testing.T) {
	snap := &control.ProjectBoardSnapshot{
		Project: map[string]string{"project_id": "alpha", "name": "alpha"},
		Counts:  map[string]int{"needs_input": 1, "clarifying": 1, "inbox": 1, "blocked": 1, "ready": 1, "executing": 1, "done": 2},
		Columns: map[string][]control.CardSummary{
			"needs_input": {{ID: "feature-alpha-001", Title: "Await answer", RecommendedNextStep: "ask owner", RecommendedNextReason: "Missing answer", LastOrchestratorAction: "requested_input", NeedsFeedbackFrom: []string{"admin"}, ClarificationCycles: 1, ReviewState: "needs_attention"}},
			"blocked":     {{ID: "feature-alpha-002", Title: "Blocked item", RecommendedNextStep: "resolve dependency", LastOrchestratorAction: "ingested_executor_result", BlockedCycles: 2, DeliveryState: "failed", DeliveryTarget: "staging", HasRollback: true}},
			"ready":       {{ID: "feature-alpha-003", Title: "Ready item", RecommendedNextStep: "dispatch", LastOrchestratorAction: "marked_ready"}},
			"executing":   {{ID: "feature-alpha-004", Title: "Executing item", RecommendedNextStep: "wait for result", LastOrchestratorAction: "dispatched_execution", LinkedBeadsIDs: []string{"bd-123"}, DispatchedTo: "omo-implementation", ExecutionAttemptCount: 1, ExecutorResultStatus: "blocked", ExecutorResultSummary: "CI is red"}},
		},
		NextAction: map[string]string{"recommended": "spawn_execution", "reason": "A ready card can move into execution", "target_card_id": "feature-alpha-003"},
	}

	out := RenderProjectBoard(snap)
	for _, want := range []string{
		"PROJECT BOARD — alpha",
		"Needs human input (1)",
		"- alpha/feature-alpha-001 — Await answer | next: ask owner | why: Missing answer | last: requested input | waiting on: admin | review: needs_attention | friction: clarify:1",
		"- alpha/feature-alpha-002 — Blocked item | next: resolve dependency | last: ingested executor result | delivery: failed -> staging | rollback recorded | friction: blocked:2",
		"Executing now (1)",
		"- alpha/feature-alpha-004 — Executing item | next: wait for result | last: dispatched execution | friction: exec:1 | beads: bd-123 | dispatch: omo-implementation | result: blocked — CI is red",
		"Target: alpha/feature-alpha-003",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q\nfull output:\n%s", want, out)
		}
	}
}

func TestRenderCardDetail(t *testing.T) {
	card := &control.FeatureCard{
		ID:                     "feature-alpha-007",
		ProjectID:              "alpha",
		Title:                  "Tighten board visibility",
		Status:                 "blocked",
		RiskLevel:              "medium",
		RawRequest:             "Make the card more observable without going full web UI.",
		SourceRefs:             []string{"ticket:ALPHA-7", "chat:thread-42"},
		NormalizedIntent:       "Improve control tower card visibility",
		ScopeIn:                []string{"card show command", "board hints"},
		ScopeOut:               []string{"web UI"},
		LastOrchestratorAction: "ingested_executor_result",
		LastOrchestratorReason: "Execution surfaced a real blocker that needs orchestration",
		LastOrchestratorAt:     "2026-03-23T08:00:00Z",
		RecommendedNextAction:  "resolve_blocker",
		RecommendedNextReason:  "Review the blocker details and unblock or replan",
		WaitingOn:              []string{"orchestrator"},
		BlockingReasons:        []string{"CI secret missing"},
		NeedsFeedbackFrom:      []string{"admin"},
		LinkedBeadsIDs:         []string{"bd-314"},
		DispatchedTo:           "omo-implementation",
		DispatchedAt:           "2026-03-23T07:40:00Z",
		DispatchedPacketPath:   ".sdp/control/projects/alpha/dispatches/feature-alpha-007.json",
		ExecutorResult: &control.ExecutorResultSummary{
			Status:              "blocked",
			Summary:             "CI secret missing in staging",
			RecommendedNextStep: "restore secret and retry",
			Findings:            []string{"staging deploy token not present"},
			OpenRisks:           []string{"Release stays blocked"},
		},
		ClarificationCycles:   2,
		BlockedCycles:         1,
		ExecutionAttemptCount: 1,
		ReviewFailCount:       1,
		ReviewState:           "failed",
		ReviewSummary:         "Security review found a release blocker",
		ReviewRef:             "artifacts/review-note.md",
		DeliveryState:         "failed",
		DeliveryTarget:        "staging",
		DeliverySummary:       "Smoke tests failed after rollout",
		DeliveryRef:           "deploy:staging-42",
		RollbackSummary:       "Rolled back staging deployment",
		RollbackRef:           "rollback:staging-42",
		FollowupRefs:          []string{"followup:hotfix-99"},
	}

	out := RenderCardDetail(card)
	for _, want := range []string{
		"CARD — Tighten board visibility",
		"ID: alpha/feature-alpha-007 | Status: blocked | Risk: medium",
		"- Source: ticket:ALPHA-7, chat:thread-42",
		"- Last orchestrator: ingested executor result — Execution surfaced a real blocker that needs orchestration (2026-03-23T08:00:00Z)",
		"- Next: resolve blocker — Review the blocker details and unblock or replan",
		"- Beads: bd-314",
		"- Result: blocked — CI secret missing in staging",
		"- Result next: restore secret and retry",
		"- Review: failed — Security review found a release blocker",
		"- Review ref: artifacts/review-note.md",
		"- Delivery: failed -> staging — Smoke tests failed after rollout",
		"- Delivery ref: deploy:staging-42",
		"- Rollback: Rolled back staging deployment",
		"- Follow-up refs: followup:hotfix-99",
		"- blocked_cycles: 1",
		"- review_fail_count: 1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q\nfull output:\n%s", want, out)
		}
	}
}
