package review

import (
	"testing"
)

func TestAggregateFeedback_empty(t *testing.T) {
	req := AggregateFeedback("proj1", "i1", "run-1", nil)
	if req.ProjectID != "proj1" || req.IssueID != "i1" || req.RunID != "run-1" {
		t.Errorf("got %+v", req)
	}
	if len(req.Comments) != 0 || len(req.Personas) != 0 {
		t.Errorf("expected empty comments/personas: %+v", req)
	}
}

func TestAggregateFeedback_needsChanges(t *testing.T) {
	verdicts := []ReviewVerdict{
		{PersonaID: "qa", Verdict: "needs_changes", Comments: []string{"fix test"}},
		{PersonaID: "security", Verdict: "approve"},
	}
	req := AggregateFeedback("p", "i", "r", verdicts)
	if len(req.Comments) != 1 || req.Comments[0] != "fix test" {
		t.Errorf("Comments: %v", req.Comments)
	}
	if len(req.Personas) != 1 || req.Personas[0] != "qa" {
		t.Errorf("Personas: %v", req.Personas)
	}
}

func TestAggregateFeedback_dedup(t *testing.T) {
	verdicts := []ReviewVerdict{
		{PersonaID: "qa", Verdict: "reject", Comments: []string{"same", "same"}},
	}
	req := AggregateFeedback("p", "i", "r", verdicts)
	if len(req.Comments) != 1 {
		t.Errorf("expected dedup comments: %v", req.Comments)
	}
}
