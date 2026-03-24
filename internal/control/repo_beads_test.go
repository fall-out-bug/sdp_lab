//go:build integration

package control

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestBeadsCardRepository_Integration tests against the real Beads database.
// Requires: bd daemon running, SDP project database present.
// Run with: go test -tags=integration ./internal/control/ -run Integration
func TestBeadsCardRepository_Integration(t *testing.T) {
	if os.Getenv("SDP_TEST_BEADS") == "" {
		t.Skip("set SDP_TEST_BEADS=1 to run Beads integration tests")
	}

	repo := NewBeadsCardRepository("", nil)

	// Test LoadCards — query existing SDP cards
	t.Run("ListCards", func(t *testing.T) {
		cards, err := repo.LoadCards("default")
		if err != nil {
			t.Logf("LoadCards error (may be expected if no cards): %v", err)
			return
		}
		t.Logf("Found %d cards", len(cards))
		for _, c := range cards {
			t.Logf("  %s: %s [%s]", c.ID, c.Title, c.Status)
		}
	})

	// Test QueryReady
	t.Run("QueryReady", func(t *testing.T) {
		cards, err := repo.QueryReady()
		if err != nil {
			t.Logf("QueryReady error: %v", err)
			return
		}
		t.Logf("Ready items: %d", len(cards))
	})

	// Test LoadCardByID — load a real card
	t.Run("LoadByID", func(t *testing.T) {
		// Get any card first
		data, err := repo.runBD("list", "--limit", "1")
		if err != nil {
			t.Fatalf("bd list: %v", err)
		}
		var issues []bdIssue
		if err := json.Unmarshal(data, &issues); err != nil {
			t.Fatalf("parse: %v", err)
		}
		if len(issues) == 0 {
			t.Skip("no issues in database")
		}

		card, err := repo.LoadCardByID(issues[0].ID)
		if err != nil {
			t.Fatalf("LoadCardByID: %v", err)
		}
		if card.ID != issues[0].ID {
			t.Errorf("ID mismatch: got %s, want %s", card.ID, issues[0].ID)
		}
		if card.Title != issues[0].Title {
			t.Errorf("Title mismatch: got %s, want %s", card.Title, issues[0].Title)
		}
		t.Logf("Loaded: %s — %s", card.ID, card.Title)
	})

	// Test CreateCard + LoadCardByID round-trip
	t.Run("CreateAndLoad", func(t *testing.T) {
		card := &FeatureCard{
			ProjectID:       "test",
			Title:           "[TEST] BeadsCardRepository integration test",
			Status:          "open",
			NormalizedIntent: "This is a test card created by integration test. Should be deleted.",
			ExecutionMode:   "2",
			TaskType:        "build",
		}

		if err := repo.CreateCard("test", card); err != nil {
			t.Fatalf("CreateCard: %v", err)
		}
		if card.ID == "" {
			t.Fatal("CreateCard did not return an ID")
		}
		t.Logf("Created: %s", card.ID)

		// Load it back
		loaded, err := repo.LoadCardByID(card.ID)
		if err != nil {
			t.Fatalf("LoadCardByID: %v", err)
		}
		if loaded.Title != card.Title {
			t.Errorf("Title: got %s, want %s", loaded.Title, card.Title)
		}
		if loaded.TaskType != "build" {
			t.Errorf("TaskType: got %s, want build", loaded.TaskType)
		}

		// Cleanup: close the test card
		_ = repo.SaveCard(&FeatureCard{ID: card.ID, Status: "closed"})
		t.Logf("Cleaned up: %s", card.ID)
	})

	// Test SetState
	t.Run("SetState", func(t *testing.T) {
		data, err := repo.runBD("list", "--limit", "1")
		if err != nil {
			t.Skipf("bd list: %v", err)
		}
		var issues []bdIssue
		json.Unmarshal(data, &issues)
		if len(issues) == 0 {
			t.Skip("no issues")
		}

		err = repo.SetState(issues[0].ID, "test_dim", "test_value", "integration test")
		if err != nil {
			t.Fatalf("SetState: %v", err)
		}
		t.Logf("Set state on %s: test_dim=test_value", issues[0].ID)
	})

	// Test Gate lifecycle
	t.Run("GateLifecycle", func(t *testing.T) {
		data, err := repo.runBD("list", "--limit", "1")
		if err != nil {
			t.Skipf("bd list: %v", err)
		}
		var issues []bdIssue
		json.Unmarshal(data, &issues)
		if len(issues) == 0 {
			t.Skip("no issues")
		}

		gateID, err := repo.CreateGate(issues[0].ID, "test_human")
		if err != nil {
			t.Fatalf("CreateGate: %v", err)
		}
		if gateID == "" {
			t.Fatal("CreateGate returned empty ID")
		}
		if !strings.HasPrefix(gateID, "sdplab-") {
			t.Errorf("unexpected gate ID: %s", gateID)
		}
		t.Logf("Created gate: %s", gateID)

		err = repo.ResolveGate(gateID)
		if err != nil {
			t.Fatalf("ResolveGate: %v", err)
		}
		t.Logf("Resolved gate: %s", gateID)
	})
}
