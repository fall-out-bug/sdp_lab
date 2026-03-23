package control

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIngestExecutorResultSuccess(t *testing.T) {
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

	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusSuccess,
		Summary:         "Implementation completed successfully",
		Artifacts:       []ExecutorArtifact{{Type: "code", Reference: "/path/to/code", Description: "Main implementation"}},
	}

	ingestedCard, err := store.IngestExecutorResult(result)
	if err != nil {
		t.Fatalf("IngestExecutorResult error: %v", err)
	}

	if ingestedCard.Status != "done" {
		t.Fatalf("status = %s, want done", ingestedCard.Status)
	}
	if ingestedCard.ExecutorResult == nil {
		t.Fatal("ExecutorResult should be set")
	}
	if ingestedCard.ExecutorResult.Status != "success" {
		t.Fatalf("executor_result.status = %s, want success", ingestedCard.ExecutorResult.Status)
	}
	if len(ingestedCard.LinkedArtifacts) == 0 {
		t.Fatal("LinkedArtifacts should contain result artifacts")
	}

	if _, err := os.Stat(filepath.Join(store.projectSnapshotsDir("openclaw"), "board.json")); err != nil {
		t.Fatalf("project snapshot not updated: %v", err)
	}

	if _, err := os.Stat(filepath.Join(store.ControlRoot, "portfolio", "snapshot.json")); err != nil {
		t.Fatalf("portfolio snapshot not updated: %v", err)
	}
}

func TestIngestExecutorResultBlocked(t *testing.T) {
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

	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusBlocked,
		Summary:         "Blocked by missing dependency",
		Findings:        []string{"Dependency X not available", "Resource Y exhausted"},
		OpenRisks:       []string{"Risk A remains unresolved"},
	}

	ingestedCard, err := store.IngestExecutorResult(result)
	if err != nil {
		t.Fatalf("IngestExecutorResult error: %v", err)
	}

	if ingestedCard.Status != "blocked" {
		t.Fatalf("status = %s, want blocked", ingestedCard.Status)
	}
	if ingestedCard.ExecutorResult == nil {
		t.Fatal("ExecutorResult should be set")
	}
	if ingestedCard.BlockedCycles != 1 {
		t.Fatalf("blocked_cycles = %d, want 1", ingestedCard.BlockedCycles)
	}
	if len(ingestedCard.BlockingReasons) < 2 {
		t.Fatalf("BlockingReasons should contain findings, got %v", ingestedCard.BlockingReasons)
	}
	if len(ingestedCard.OpenQuestions) < 1 {
		t.Fatalf("OpenQuestions should contain open risks, got %v", ingestedCard.OpenQuestions)
	}
}

func TestIngestExecutorResultNeedsReview(t *testing.T) {
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

	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "review",
		Status:          ResultStatusNeedsReview,
		Summary:         "Code review required for security concerns",
		Findings:        []string{"Potential XSS vulnerability in component X", "Missing input validation"},
	}

	ingestedCard, err := store.IngestExecutorResult(result)
	if err != nil {
		t.Fatalf("IngestExecutorResult error: %v", err)
	}

	if ingestedCard.Status != "needs_input" {
		t.Fatalf("status = %s, want needs_input", ingestedCard.Status)
	}
	if ingestedCard.ExecutorResult == nil {
		t.Fatal("ExecutorResult should be set")
	}
	if len(ingestedCard.FeedbackRequest) == 0 {
		t.Fatal("FeedbackRequest should contain summary")
	}
	if len(ingestedCard.DecisionRequired) == 0 {
		t.Fatal("DecisionRequired should contain findings")
	}
	if len(ingestedCard.NeedsFeedbackFrom) < 2 {
		t.Fatalf("NeedsFeedbackFrom should contain human and admin, got %v", ingestedCard.NeedsFeedbackFrom)
	}
	if ingestedCard.ReviewFailCount != 1 {
		t.Fatalf("review_fail_count = %d, want 1", ingestedCard.ReviewFailCount)
	}
	if ingestedCard.ReviewState != "needs_attention" {
		t.Fatalf("review_state = %s, want needs_attention", ingestedCard.ReviewState)
	}
	if ingestedCard.ReviewSummary != "Code review required for security concerns" {
		t.Fatalf("review_summary = %q", ingestedCard.ReviewSummary)
	}
	if ingestedCard.LastOrchestratorAction != "ingested_executor_result" {
		t.Fatalf("last_orchestrator_action = %s", ingestedCard.LastOrchestratorAction)
	}
}

func TestIngestExecutorResultNeedsInput(t *testing.T) {
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

	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusNeedsInput,
		Summary:         "Additional clarification needed on API contract",
		Findings:        []string{"Which endpoint version?", "Include backward compatibility?"},
	}

	ingestedCard, err := store.IngestExecutorResult(result)
	if err != nil {
		t.Fatalf("IngestExecutorResult error: %v", err)
	}

	if ingestedCard.Status != "needs_input" {
		t.Fatalf("status = %s, want needs_input", ingestedCard.Status)
	}
	if ingestedCard.ExecutorResult == nil {
		t.Fatal("ExecutorResult should be set")
	}
	if len(ingestedCard.FeedbackRequest) == 0 {
		t.Fatal("FeedbackRequest should contain summary")
	}
	if len(ingestedCard.AuthorUpdate) == 0 {
		t.Fatal("AuthorUpdate should contain findings")
	}
}

func TestIngestExecutorResultFailed(t *testing.T) {
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

	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusFailed,
		Summary:         "Implementation failed due to technical constraint",
		Findings:        []string{"Library X does not support required feature Y"},
	}

	ingestedCard, err := store.IngestExecutorResult(result)
	if err != nil {
		t.Fatalf("IngestExecutorResult error: %v", err)
	}

	if ingestedCard.Status != "clarifying" {
		t.Fatalf("status = %s, want clarifying", ingestedCard.Status)
	}
	if ingestedCard.ExecutorResult == nil {
		t.Fatal("ExecutorResult should be set")
	}
	if len(ingestedCard.AuthorUpdate) == 0 {
		t.Fatal("AuthorUpdate should contain failure details")
	}
	if len(ingestedCard.OpenQuestions) == 0 {
		t.Fatal("OpenQuestions should contain findings")
	}
}

func TestIngestExecutorResultFailsForNilPacket(t *testing.T) {
	store := setupStore(t)
	_, err := store.IngestExecutorResult(nil)
	if err == nil {
		t.Fatal("expected error for nil packet")
	}
}

func TestIngestExecutorResultFailsForMissingParentFeatureID(t *testing.T) {
	store := setupStore(t)
	result := &ExecutorResultPacket{
		BeadsTaskID:  "test-task-123",
		ExecutorRole: "omo-implementation",
		Status:       ResultStatusSuccess,
		Summary:      "test",
	}
	_, err := store.IngestExecutorResult(result)
	if err == nil {
		t.Fatal("expected error for missing parent_feature_id")
	}
}

func TestIngestExecutorResultFailsForUnknownCard(t *testing.T) {
	store := setupStore(t)
	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: "nonexistent-card",
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusSuccess,
		Summary:         "test",
	}
	_, err := store.IngestExecutorResult(result)
	if err == nil {
		t.Fatal("expected error for unknown card")
	}
}

func TestIngestExecutorResultFailsForUnknownStatus(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Test feature", "test")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "test intent", "feature", "openclaw", "low", "execute", []string{"implementation"}, []string{"code"})
	if err != nil {
		t.Fatal(err)
	}

	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ExecutorResultStatus("unknown"),
		Summary:         "test",
	}
	_, err = store.IngestExecutorResult(result)
	if err == nil {
		t.Fatal("expected error for unknown status")
	}
}

func TestIngestExecutorResultWithArtifacts(t *testing.T) {
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

	result := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusSuccess,
		Summary:         "Completed with artifacts",
		Artifacts: []ExecutorArtifact{
			{Type: "code", Reference: "/path/to/main.go", Description: "Main implementation"},
			{Type: "test", Reference: "/path/to/test.go", Description: "Unit tests"},
			{Type: "doc", Reference: "/path/to/README.md", Description: "Documentation"},
		},
	}

	ingestedCard, err := store.IngestExecutorResult(result)
	if err != nil {
		t.Fatalf("IngestExecutorResult error: %v", err)
	}

	if len(ingestedCard.LinkedArtifacts) != 3 {
		t.Fatalf("LinkedArtifacts should contain 3 artifacts, got %d", len(ingestedCard.LinkedArtifacts))
	}

	if ingestedCard.ExecutorResult == nil {
		t.Fatal("ExecutorResult should be set")
	}

	if len(ingestedCard.ExecutorResult.Artifacts) != 3 {
		t.Fatalf("ExecutorResult.Artifacts should contain 3 artifacts, got %d", len(ingestedCard.ExecutorResult.Artifacts))
	}

	expectedArtifacts := []string{"code: /path/to/main.go", "test: /path/to/test.go", "doc: /path/to/README.md"}
	for i, art := range ingestedCard.ExecutorResult.Artifacts {
		if art != expectedArtifacts[i] {
			t.Fatalf("Artifact %d: got %s, want %s", i, art, expectedArtifacts[i])
		}
	}
}

func TestIngestExecutorResultFromJSONFile(t *testing.T) {
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

	resultPacket := &ExecutorResultPacket{
		BeadsTaskID:     "test-task-123",
		ParentFeatureID: card.ID,
		ExecutorRole:    "omo-implementation",
		Status:          ResultStatusSuccess,
		Summary:         "Completed successfully",
		Artifacts:       []ExecutorArtifact{{Type: "code", Reference: "/path/to/code"}},
	}

	tempDir := t.TempDir()
	resultPath := filepath.Join(tempDir, "result.json")
	data, err := json.MarshalIndent(resultPacket, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loadedResult := &ExecutorResultPacket{}
	data, err = os.ReadFile(resultPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, loadedResult); err != nil {
		t.Fatal(err)
	}

	ingestedCard, err := store.IngestExecutorResult(loadedResult)
	if err != nil {
		t.Fatalf("IngestExecutorResult error: %v", err)
	}

	if ingestedCard.Status != "done" {
		t.Fatalf("status = %s, want done", ingestedCard.Status)
	}

	ingestedCardNow, _ := store.LoadCardByID(ingestedCard.ID)
	if ingestedCardNow.ExecutorResult == nil {
		t.Fatal("ExecutorResult should persist after card reload")
	}

	receivedAt, err := time.Parse(time.RFC3339, ingestedCardNow.ExecutorResult.ReceivedAt)
	if err != nil {
		t.Fatalf("parse received_at: %v", err)
	}

	if time.Since(receivedAt) > time.Minute {
		t.Fatal("ReceivedAt should be recent")
	}
}

func TestRemoveAgent(t *testing.T) {
	agents := []string{"orchestrator", "executor", "reviewer"}
	result := removeAgent(agents, "executor")
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	for _, agent := range result {
		if agent == "executor" {
			t.Fatal("executor should be removed")
		}
	}
}

func TestRemoveAgentHandlesEmpty(t *testing.T) {
	agents := []string{"executor"}
	result := removeAgent(agents, "executor")
	if result != nil {
		t.Fatal("result should be nil when all agents removed")
	}
}
