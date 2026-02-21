package adapter

import (
	"testing"

	"sdp_dev/internal/beads"
)

func TestNewIntentTranslator(t *testing.T) {
	tx := NewIntentTranslator()
	if tx == nil {
		t.Fatal("NewIntentTranslator returned nil")
	}
}

func TestIntentTranslator_Translate(t *testing.T) {
	tx := NewIntentTranslator()

	_, err := tx.Translate(nil, "r1")
	if err == nil || err.Error() != "issue required" {
		t.Errorf("Translate(nil): got %v", err)
	}

	_, err = tx.Translate(&beads.Issue{ID: ""}, "r1")
	if err == nil {
		t.Error("expected error for empty issue ID")
	}

	intent, err := tx.Translate(&beads.Issue{
		ID:                "i1",
		Title:             "Fix bug",
		Description:       "Details",
		AcceptanceCriteria: "AC1",
		Labels:            []string{"role:coder"},
	}, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if intent.IssueID != "i1" || intent.RunID != "run-1" || intent.AgentRef != "coder" {
		t.Errorf("got %+v", intent)
	}
	if intent.Prompt == "" || intent.Objective == "" || intent.SpecHash == "" {
		t.Error("expected prompt, objective, specHash")
	}
}
