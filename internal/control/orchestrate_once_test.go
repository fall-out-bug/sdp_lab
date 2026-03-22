package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOrchestrateOnceIngestsResult(t *testing.T) {
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
	_, err = store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	resultPacket := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusSuccess,
		Summary:         "Implementation completed successfully",
		Artifacts:       []ExecutorArtifact{{Type: "code", Reference: "/path/to/code", Description: "Main implementation"}},
	}

	resultsDir := store.executorResultsDir()
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	resultPath := filepath.Join(resultsDir, "result-001.json")
	data, err := json.MarshalIndent(resultPacket, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result.Action != "ingested" {
		t.Fatalf("Action = %s, want ingested", result.Action)
	}
	if result.IngestedCard == nil {
		t.Fatal("IngestedCard should be set")
	}
	if result.IngestedCard.ID != card.ID {
		t.Fatalf("IngestedCard.ID = %s, want %s", result.IngestedCard.ID, card.ID)
	}

	if _, err := os.Stat(resultPath); err == nil {
		t.Fatal("Result file should be removed after ingestion")
	}
}

func TestOrchestrateOnceDispatchesWhenNoResults(t *testing.T) {
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

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result.Action != "dispatched" {
		t.Fatalf("Action = %s, want dispatched", result.Action)
	}
	if result.DispatchedCard == nil {
		t.Fatal("DispatchedCard should be set")
	}
	if result.DispatchedCard.ID != card.ID {
		t.Fatalf("DispatchedCard.ID = %s, want %s", result.DispatchedCard.ID, card.ID)
	}
	if result.ExecutorRole == "" {
		t.Fatal("ExecutorRole should be set")
	}
	if result.PacketPath == "" {
		t.Fatal("PacketPath should be set")
	}
}

func TestOrchestrateOnceNoAction(t *testing.T) {
	store := setupStore(t)

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result.Action != "no_action" {
		t.Fatalf("Action = %s, want no_action", result.Action)
	}
	if result.NoActionReason == "" {
		t.Fatal("NoActionReason should be set when nothing to do")
	}
}

func TestOrchestrateOncePrefersIngestOverDispatch(t *testing.T) {
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
	_, err = store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	resultPacket := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusSuccess,
		Summary:         "Implementation completed successfully",
		Artifacts:       []ExecutorArtifact{{Type: "code", Reference: "/path/to/code", Description: "Main implementation"}},
	}

	resultsDir := store.executorResultsDir()
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	resultPath := filepath.Join(resultsDir, "result-001.json")
	data, err := json.MarshalIndent(resultPacket, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result.Action != "ingested" {
		t.Fatalf("Action = %s, want ingested (prefers result ingestion over dispatch)", result.Action)
	}
}

func TestOrchestrateOnceSkipsAlreadyIngested(t *testing.T) {
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
	_, err = store.DispatchCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	resultPacket := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusSuccess,
		Summary:         "First result",
	}

	resultsDir := store.executorResultsDir()
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	resultPath := filepath.Join(resultsDir, "result-001.json")
	data, err := json.MarshalIndent(resultPacket, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result.Action != "ingested" {
		t.Fatalf("Action = %s, want ingested", result.Action)
	}

	result2, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result2.Action != "no_action" {
		t.Fatalf("Action = %s, want no_action (already ingested)", result2.Action)
	}
}

func TestOrchestrateOnceSkipsHiddenFiles(t *testing.T) {
	store := setupStore(t)

	resultsDir := store.executorResultsDir()
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	hiddenFile := filepath.Join(resultsDir, ".hidden.json")
	if err := os.WriteFile(hiddenFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result.Action != "no_action" {
		t.Fatalf("Action = %s, want no_action (only hidden file exists)", result.Action)
	}
}

func TestOrchestrateOnceSkipsNonJSONFiles(t *testing.T) {
	store := setupStore(t)

	resultsDir := store.executorResultsDir()
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	textFile := filepath.Join(resultsDir, "result.txt")

	if err := os.WriteFile(textFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := store.OrchestrateOnce()
	if err != nil {
		t.Fatalf("OrchestrateOnce error: %v", err)
	}

	if result.Action != "no_action" {
		t.Fatalf("Action = %s, want no_action (no JSON files)", result.Action)
	}
}
