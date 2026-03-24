package executor

import (
	"testing"

	"sdp_dev/internal/control"
)

func TestRouteFindingsToCardFailedResult(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Failed exec", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "executing"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	result := &control.ExecutorResultPacket{
		Status:   control.ResultStatusFailed,
		Findings: []string{"fix flaky test", "inspect logs"},
	}
	if err := RouteFindingsToCard(store, card.ProjectID, card.ID, result); err != nil {
		t.Fatalf("RouteFindingsToCard error: %v", err)
	}

	updated, err := store.LoadCard(card.ProjectID, card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExecutionAttemptCount != 1 {
		t.Fatalf("ExecutionAttemptCount = %d, want 1", updated.ExecutionAttemptCount)
	}
	if updated.RecommendedNextAction != "retry_dispatch" {
		t.Fatalf("RecommendedNextAction = %q, want retry_dispatch", updated.RecommendedNextAction)
	}
	if len(updated.BlockingReasons) == 0 || updated.BlockingReasons[0] != "execution failed" {
		t.Fatalf("BlockingReasons = %+v", updated.BlockingReasons)
	}
	if len(updated.AdminActionRequired) != 2 {
		t.Fatalf("AdminActionRequired = %+v, want findings appended", updated.AdminActionRequired)
	}
}

func TestRouteFindingsToCardThirdFailureBlocksCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Third failure", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "executing"
	card.ExecutionAttemptCount = 2
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	result := &control.ExecutorResultPacket{Status: control.ResultStatusFailed}
	if err := RouteFindingsToCard(store, card.ProjectID, card.ID, result); err != nil {
		t.Fatalf("RouteFindingsToCard error: %v", err)
	}

	updated, err := store.LoadCard(card.ProjectID, card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExecutionAttemptCount != 3 {
		t.Fatalf("ExecutionAttemptCount = %d, want 3", updated.ExecutionAttemptCount)
	}
	if updated.Status != "blocked" {
		t.Fatalf("status = %s, want blocked", updated.Status)
	}
	if len(updated.BlockingReasons) < 2 {
		t.Fatalf("BlockingReasons = %+v, want failure + max attempts", updated.BlockingReasons)
	}
}

func TestRouteFindingsToCardSuccessfulResultNoChanges(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Success exec", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "executing"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	result := &control.ExecutorResultPacket{Status: control.ResultStatusSuccess}
	if err := RouteFindingsToCard(store, card.ProjectID, card.ID, result); err != nil {
		t.Fatalf("RouteFindingsToCard error: %v", err)
	}

	updated, err := store.LoadCard(card.ProjectID, card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ExecutionAttemptCount != 0 {
		t.Fatalf("ExecutionAttemptCount = %d, want 0", updated.ExecutionAttemptCount)
	}
	if len(updated.BlockingReasons) != 0 {
		t.Fatalf("BlockingReasons = %+v, want none", updated.BlockingReasons)
	}
	if len(updated.AdminActionRequired) != 0 {
		t.Fatalf("AdminActionRequired = %+v, want none", updated.AdminActionRequired)
	}
}
