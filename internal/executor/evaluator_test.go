package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/control"
)

func TestEvaluatorCriteria(t *testing.T) {
	cfg := DefaultEvaluatorConfig()
	passed := map[string]bool{
		"tests_pass":        true,
		"scope_adherence":   true,
		"evidence_complete": false,
		"code_quality":      true,
	}
	score, hardFailure := scoreCriteria(cfg.Criteria, passed)
	if hardFailure {
		t.Fatalf("hardFailure = true, want false")
	}
	if score != 0.80 {
		t.Fatalf("score = %.2f, want 0.80", score)
	}
}

func TestEvaluateBuild_NoEvidence(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Eval missing evidence", "check evaluation")
	if err != nil {
		t.Fatal(err)
	}
	card.NormalizedIntent = "check evaluation"
	card.ScopeIn = []string{"internal/executor"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	result, err := EvaluateBuild(context.Background(), store.ProjectRoot, card, DefaultEvaluatorConfig())
	if err != nil {
		t.Fatalf("EvaluateBuild error: %v", err)
	}
	if result.Verdict != evalVerdictBlocked {
		t.Fatalf("verdict = %s, want %s", result.Verdict, evalVerdictBlocked)
	}
}

func TestEvalResult_Verdict(t *testing.T) {
	if got := verdictForScore(0.90, false); got != evalVerdictPass {
		t.Fatalf("verdict(0.90,false) = %s, want %s", got, evalVerdictPass)
	}
	if got := verdictForScore(0.65, false); got != evalVerdictNeedsReview {
		t.Fatalf("verdict(0.65,false) = %s, want %s", got, evalVerdictNeedsReview)
	}
	if got := verdictForScore(0.65, true); got != evalVerdictFail {
		t.Fatalf("verdict(0.65,true) = %s, want %s", got, evalVerdictFail)
	}
	if got := verdictForScore(0.40, false); got != evalVerdictFail {
		t.Fatalf("verdict(0.40,false) = %s, want %s", got, evalVerdictFail)
	}
}

func TestServeBridgeEvaluateWritesEvidence(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Eval pass", "evaluate build")
	if err != nil {
		t.Fatal(err)
	}
	card.NormalizedIntent = "evaluate build"
	card.ScopeIn = []string{"internal/executor"}
	card.ScopeOut = []string{"deploy/**"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	artifactDir := filepath.Join(store.ProjectRoot, ".sdp", "artifacts", card.ID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{
		"phase":         "build",
		"card_id":       card.ID,
		"exit_code":     0,
		"status":        control.ResultStatusSuccess,
		"files_changed": []string{"internal/executor/evaluator.go"},
	}
	data, _ := json.Marshal(payload)
	if err := os.WriteFile(filepath.Join(artifactDir, "build.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// Ensure no LLM invoker is reachable: clear serve URL and
	// point PATH to an empty dir so DefaultLLMInvoker (exec) also fails fast.
	t.Setenv("OMO_SERVE_URL", "")
	t.Setenv("PATH", t.TempDir())

	bridge := NewServeBridge(store, store.ProjectRoot)
	result, err := bridge.Evaluate(context.Background(), card.ID)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}
	if result.Verdict != evalVerdictBlocked {
		t.Fatalf("verdict = %s, want %s when OmO is unavailable", result.Verdict, evalVerdictBlocked)
	}
	// evidence is written regardless of verdict
}
