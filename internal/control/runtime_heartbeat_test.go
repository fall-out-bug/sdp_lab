package control

import (
	"testing"
	"time"
)

func TestDispatchCardMarksRuntimePending(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Heartbeat feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "intent", "feature", "openclaw", "low", "dispatch", []string{"implementation"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkReady("openclaw", card.ID); err != nil {
		t.Fatal(err)
	}

	defer SetCreateBeadsIssueFn(createBeadsIssue)
	SetCreateBeadsIssueFn(MockCreateBeadsIssue("bd-heartbeat-1"))

	card, err = store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if card.ExecutorRuntimeState != ExecutorRuntimePending {
		t.Fatalf("executor_runtime_state = %s, want %s", card.ExecutorRuntimeState, ExecutorRuntimePending)
	}
	if card.ExecutorProgressSummary == "" {
		t.Fatal("executor_progress_summary should be set on dispatch")
	}
}

func TestRecordExecutorHeartbeatUpdatesRuntimeFields(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Heartbeat feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "intent", "feature", "openclaw", "low", "dispatch", []string{"implementation"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkReady("openclaw", card.ID); err != nil {
		t.Fatal(err)
	}
	defer SetCreateBeadsIssueFn(createBeadsIssue)
	SetCreateBeadsIssueFn(MockCreateBeadsIssue("bd-heartbeat-1"))
	if _, err := store.DispatchCard("openclaw", card.ID); err != nil {
		t.Fatal(err)
	}

	card, err = store.RecordExecutorHeartbeat("openclaw", card.ID, "sess-123", ExecutorRuntimeRunning, "Halfway through implementation")
	if err != nil {
		t.Fatal(err)
	}
	if card.ExecutorSessionID != "sess-123" {
		t.Fatalf("executor_session_id = %s", card.ExecutorSessionID)
	}
	if card.ExecutorStartedAt == "" {
		t.Fatal("executor_started_at should be set")
	}
	if card.LastExecutorHeartbeatAt == "" {
		t.Fatal("last_executor_heartbeat_at should be set")
	}
	if card.ExecutorRuntimeState != ExecutorRuntimeRunning {
		t.Fatalf("executor_runtime_state = %s", card.ExecutorRuntimeState)
	}
	if card.ExecutorProgressSummary != "Halfway through implementation" {
		t.Fatalf("executor_progress_summary = %q", card.ExecutorProgressSummary)
	}
}

func TestDoctorControlReportsRuntimeHeartbeatIssues(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Heartbeat feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "intent", "feature", "openclaw", "low", "dispatch", []string{"implementation"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MarkReady("openclaw", card.ID); err != nil {
		t.Fatal(err)
	}
	defer SetCreateBeadsIssueFn(createBeadsIssue)
	SetCreateBeadsIssueFn(MockCreateBeadsIssue("bd-heartbeat-1"))
	card, err = store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().UTC().Add(-90 * time.Minute).Format(time.RFC3339)
	card.DispatchedAt = old
	card.ExecutorSessionID = "sess-123"
	card.ExecutorStartedAt = old
	card.LastExecutorHeartbeatAt = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	card.ExecutorRuntimeState = ExecutorRuntimeLost
	overwriteCard(t, store, card)

	report, err := store.DoctorControl()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, check := range report.Checks {
		seen[check.CheckID] = true
	}
	for _, want := range []string{"stale-executor-heartbeat", "executing-runtime-lost"} {
		if !seen[want] {
			t.Fatalf("expected check %s in %#v", want, report.Checks)
		}
	}
}
