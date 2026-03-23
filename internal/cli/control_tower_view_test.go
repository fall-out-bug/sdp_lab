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
			"waiting_on_human": {{ProjectID: "alpha", CardID: "feature-alpha-001", Title: "Need decision", RecommendedNextStep: "ask owner", NeedsFeedbackFrom: []string{"author"}}},
			"blocked":          {{ProjectID: "beta", CardID: "feature-beta-002", Title: "Blocked task", RecommendedNextStep: "resolve blocker"}},
			"ready_to_execute": {{ProjectID: "alpha", CardID: "feature-alpha-003", Title: "Ready task", RecommendedNextStep: "dispatch"}},
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
		"- alpha/feature-alpha-001 — Need decision | next: ask owner | waiting on: author",
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
			"needs_input": {{ID: "feature-alpha-001", Title: "Await answer", RecommendedNextStep: "ask owner", NeedsFeedbackFrom: []string{"admin"}}},
			"blocked":     {{ID: "feature-alpha-002", Title: "Blocked item", RecommendedNextStep: "resolve dependency"}},
			"ready":       {{ID: "feature-alpha-003", Title: "Ready item", RecommendedNextStep: "dispatch"}},
			"executing":   {{ID: "feature-alpha-004", Title: "Executing item", RecommendedNextStep: "wait for result", LinkedBeadsIDs: []string{"bd-123"}}},
		},
		NextAction: map[string]string{"recommended": "spawn_execution", "reason": "A ready card can move into execution", "target_card_id": "feature-alpha-003"},
	}

	out := RenderProjectBoard(snap)
	for _, want := range []string{
		"PROJECT BOARD — alpha",
		"Needs human input (1)",
		"- alpha/feature-alpha-001 — Await answer | next: ask owner | waiting on: admin",
		"Executing now (1)",
		"- alpha/feature-alpha-004 — Executing item | next: wait for result | beads: bd-123",
		"Target: alpha/feature-alpha-003",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q\nfull output:\n%s", want, out)
		}
	}
}
