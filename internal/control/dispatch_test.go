package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDispatchCardForReadyCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	defer SetCreateBeadsIssueFn(createBeadsIssue)
	SetCreateBeadsIssueFn(MockCreateBeadsIssue("bd-test-123"))

	result, err := store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatalf("DispatchCard error: %v", err)
	}

	if result.Status != "executing" {
		t.Fatalf("status = %s, want executing", result.Status)
	}
	if result.ExecutionAttemptCount != 1 {
		t.Fatalf("execution_attempt_count = %d, want 1", result.ExecutionAttemptCount)
	}
	if result.LastOrchestratorAction != "dispatched_execution" {
		t.Fatalf("last_orchestrator_action = %s", result.LastOrchestratorAction)
	}
	if result.DispatchedAt == "" {
		t.Fatal("DispatchedAt should be set")
	}
	if result.DispatchedTo == "" {
		t.Fatal("DispatchedTo should be set")
	}
	if result.DispatchedPacketPath == "" {
		t.Fatal("DispatchedPacketPath should be set")
	}
	if len(result.ActiveAgents) == 0 {
		t.Fatal("ActiveAgents should contain executor")
	}
	if len(result.LinkedBeadsIDs) != 1 {
		t.Fatalf("linked_beads_ids count = %d, want 1", len(result.LinkedBeadsIDs))
	}
	if result.LinkedBeadsIDs[0] != "bd-test-123" {
		t.Fatalf("beads id = %s, want bd-test-123", result.LinkedBeadsIDs[0])
	}

	packetPath := filepath.Join(store.ControlRoot, "projects", "openclaw", "dispatches", card.ID+".json")
	if _, err := os.Stat(packetPath); err != nil {
		t.Fatalf("dispatch packet file not created: %v", err)
	}
}

func TestDispatchCardForExecutingCard(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ExecuteCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatalf("DispatchCard error: %v", err)
	}

	if result.Status != "executing" {
		t.Fatalf("status = %s, want executing", result.Status)
	}

	if _, err := os.Stat(filepath.Join(store.ControlRoot, "projects", "openclaw", "snapshots", "board.json")); err != nil {
		t.Fatalf("project snapshot not updated: %v", err)
	}

	if _, err := os.Stat(filepath.Join(store.ControlRoot, "portfolio", "snapshot.json")); err != nil {
		t.Fatalf("portfolio snapshot not updated: %v", err)
	}
}

func TestDispatchCardFailsForNonDispatchableStatus(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card.Status = "inbox"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	_, err = store.DispatchCard("openclaw", card.ID)
	if err == nil {
		t.Fatal("expected error for non-dispatchable card")
	}
}

func TestDispatchCardFailsForNonexistentCard(t *testing.T) {
	store := setupStore(t)
	_, err := store.DispatchCard("openclaw", "nonexistent-card-id")
	if err == nil {
		t.Fatal("expected error for nonexistent card")
	}
}

func TestDispatchCardGeneratesValidPacket(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code", "tests"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatalf("DispatchCard error: %v", err)
	}

	if result.Status != "executing" {
		t.Fatalf("status = %s, want executing", result.Status)
	}
	if result.DispatchedAt == "" {
		t.Fatal("DispatchedAt should be set")
	}
	if result.DispatchedTo == "" {
		t.Fatal("DispatchedTo should be set")
	}
	if result.DispatchedPacketPath == "" {
		t.Fatal("DispatchedPacketPath should be set")
	}
	if len(result.ActiveAgents) == 0 {
		t.Fatal("ActiveAgents should contain executor")
	}

	packetPath := filepath.Join(store.ControlRoot, "projects", "openclaw", "dispatches", card.ID+".json")
	data, err := os.ReadFile(packetPath)
	if err != nil {
		t.Fatalf("read dispatch packet: %v", err)
	}

	var packet ExecutionPacket
	if err := json.Unmarshal(data, &packet); err != nil {
		t.Fatalf("parse dispatch packet: %v", err)
	}

	if packet.ParentFeatureID != card.ID {
		t.Fatalf("ParentFeatureID = %s, want %s", packet.ParentFeatureID, card.ID)
	}
	if packet.ExecutorRole != string(ExecutorRoleOmOImplementation) {
		t.Fatalf("ExecutorRole = %s, want %s", packet.ExecutorRole, ExecutorRoleOmOImplementation)
	}
	if packet.Objective != card.NormalizedIntent {
		t.Fatalf("Objective = %s, want %s", packet.Objective, card.NormalizedIntent)
	}
	if len(packet.ScopeIn) == 0 {
		t.Fatal("ScopeIn should not be empty")
	}
	if len(packet.ScopeOut) == 0 {
		t.Fatal("ScopeOut should not be empty")
	}
}
