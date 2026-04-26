package promote

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/build"
)

func TestPromoteFromRun_Success(t *testing.T) {
	tmp := t.TempDir()
	runID := "test-run-001"
	evDir := filepath.Join(tmp, ".sdp", "evidence", runID)
	phaseDir := filepath.Join(tmp, ".sdp", "phases", runID)

	// Create vibecode evidence.
	ev := &build.BuildEvidence{
		RunID:     runID,
		Idea:      "test feature",
		Timestamp: "2026-04-25T12:00:00Z",
		Status:    "success",
		Stages: []build.StageEvidence{
			{Name: "dispatch", Status: "success", Duration: "100ms"},
			{Name: "sandbox", Status: "success", Duration: "5s"},
			{Name: "commit", Status: "success", Duration: "50ms"},
		},
		Dispatch: build.DispatchEvidence{
			Harness:  "claude-code",
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			Score:    0.85,
			Reason:   "auto-classified",
		},
		Sandbox: build.SandboxEvidence{
			Type:    "docker",
			BuildOK: true,
			TestsOK: true,
		},
	}

	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatalf("create ev dir: %v", err)
	}
	evData, _ := json.MarshalIndent(ev, "", "  ")
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), evData, 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}

	result, err := PromoteFromRun(PromoteOptions{
		RunID:       runID,
		FeatureID:   "F135-05",
		EvidenceDir: evDir,
		PhaseDir:    phaseDir,
	})
	if err != nil {
		t.Fatalf("PromoteFromRun: %v", err)
	}

	// Verify result structure.
	if result.RunID != runID {
		t.Errorf("RunID = %q, want %q", result.RunID, runID)
	}
	if len(result.Deltas) != 3 {
		t.Errorf("Deltas count = %d, want 3", len(result.Deltas))
	}
	if len(result.Gates) != 3 {
		t.Errorf("Gates count = %d, want 3", len(result.Gates))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}

	// Verify delta files exist.
	for _, d := range result.Deltas {
		if _, err := os.Stat(d.Path); os.IsNotExist(err) {
			t.Errorf("delta file missing: %s", d.Path)
		}
	}

	// Verify gate files exist and have correct structure.
	for _, g := range result.Gates {
		data, err := os.ReadFile(g.Path)
		if err != nil {
			t.Fatalf("read gate %s: %v", g.Path, err)
		}
		var gateMap map[string]interface{}
		if err := json.Unmarshal(data, &gateMap); err != nil {
			t.Fatalf("parse gate %s: %v", g.Path, err)
		}
		if id, _ := gateMap["id"].(string); id != g.ID {
			t.Errorf("gate id = %q, want %q", id, g.ID)
		}
	}

	// Verify per-phase evidence files exist and contain required keys.
	for _, phase := range []string{"plan", "review", "eval"} {
		evPath := filepath.Join(phaseDir, phase+".evidence.json")
		data, err := os.ReadFile(evPath)
		if err != nil {
			t.Fatalf("read %s evidence: %v", phase, err)
		}
		var evMap map[string]interface{}
		if err := json.Unmarshal(data, &evMap); err != nil {
			t.Fatalf("parse %s evidence: %v", phase, err)
		}
		if src, _ := evMap["source"].(string); src != SourceLabel {
			t.Errorf("%s evidence source = %q, want %q", phase, src, SourceLabel)
		}
	}

	// Verify promotion trace.
	tracePath := filepath.Join(phaseDir, "promotion-trace.json")
	traceData, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	var trace map[string]interface{}
	if err := json.Unmarshal(traceData, &trace); err != nil {
		t.Fatalf("parse trace: %v", err)
	}
	if src, _ := trace["source"].(string); src != SourceLabel {
		t.Errorf("trace source = %q, want %q", src, SourceLabel)
	}
}

func TestPromoteFromRun_PartialFailure(t *testing.T) {
	tmp := t.TempDir()
	runID := "test-run-partial"
	evDir := filepath.Join(tmp, ".sdp", "evidence", runID)
	phaseDir := filepath.Join(tmp, ".sdp", "phases", runID)

	// Create evidence with failed status.
	ev := &build.BuildEvidence{
		RunID:     runID,
		Idea:      "broken feature",
		Timestamp: "2026-04-25T12:00:00Z",
		Status:    "partial",
		Stages: []build.StageEvidence{
			{Name: "dispatch", Status: "success"},
			{Name: "sandbox", Status: "failed"},
		},
		Sandbox: build.SandboxEvidence{
			Type:    "docker",
			BuildOK: true,
			TestsOK: false,
		},
	}

	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatalf("create ev dir: %v", err)
	}
	evData, _ := json.MarshalIndent(ev, "", "  ")
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), evData, 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}

	result, err := PromoteFromRun(PromoteOptions{
		RunID:       runID,
		FeatureID:   "F135-05",
		EvidenceDir: evDir,
		PhaseDir:    phaseDir,
	})
	if err != nil {
		t.Fatalf("PromoteFromRun should not error on partial evidence: %v", err)
	}

	// Should still produce 3 deltas and 3 gates.
	if len(result.Deltas) != 3 {
		t.Errorf("Deltas count = %d, want 3 (even on partial)", len(result.Deltas))
	}
	if len(result.Gates) != 3 {
		t.Errorf("Gates count = %d, want 3 (even on partial)", len(result.Gates))
	}

	// Verify eval evidence reflects test failure.
	evalEvPath := filepath.Join(phaseDir, "eval.evidence.json")
	data, err := os.ReadFile(evalEvPath)
	if err != nil {
		t.Fatalf("read eval evidence: %v", err)
	}
	var evalMap map[string]interface{}
	_ = json.Unmarshal(data, &evalMap)
	if goTest, _ := evalMap["go_test"].(string); goTest == "pass" {
		t.Error("eval go_test should reflect sandbox failure")
	}
}

func TestPromoteFromRun_MissingRunID(t *testing.T) {
	_, err := PromoteFromRun(PromoteOptions{FeatureID: "F135-05"})
	if err == nil {
		t.Fatal("expected error for missing RunID")
	}
}

func TestPromoteFromRun_MissingFeatureID(t *testing.T) {
	_, err := PromoteFromRun(PromoteOptions{RunID: "run-001"})
	if err == nil {
		t.Fatal("expected error for missing FeatureID")
	}
}

func TestPromoteFromRun_EvidenceNotFound(t *testing.T) {
	tmp := t.TempDir()
	_, err := PromoteFromRun(PromoteOptions{
		RunID:       "nonexistent",
		FeatureID:   "F135-05",
		EvidenceDir: filepath.Join(tmp, "no-such-dir"),
	})
	if err == nil {
		t.Fatal("expected error for missing evidence")
	}
}

func TestBuildPhaseEvidence_Mapping(t *testing.T) {
	ev := &build.BuildEvidence{
		RunID:  "run-abc",
		Status: "success",
		Dispatch: build.DispatchEvidence{
			Harness: "claude-code",
			Model:   "sonnet",
			Score:   0.9,
		},
		Sandbox: build.SandboxEvidence{
			BuildOK: true,
			TestsOK: true,
		},
	}

	phaseEv := buildPhaseEvidence(ev)

	// Plan evidence should have required keys.
	plan := phaseEv["plan"]
	if _, ok := plan["test_coverage"]; !ok {
		t.Error("plan evidence missing test_coverage")
	}
	if _, ok := plan["design_checklist"]; !ok {
		t.Error("plan evidence missing design_checklist")
	}

	// Review evidence should have required keys.
	review := phaseEv["review"]
	if _, ok := review["spec_review_verdict"]; !ok {
		t.Error("review evidence missing spec_review_verdict")
	}
	if _, ok := review["code_review_verdict"]; !ok {
		t.Error("review evidence missing code_review_verdict")
	}

	// Eval evidence should have required keys.
	eval := phaseEv["eval"]
	if _, ok := eval["go_test"]; !ok {
		t.Error("eval evidence missing go_test")
	}
	if _, ok := eval["go_vet"]; !ok {
		t.Error("eval evidence missing go_vet")
	}
	if _, ok := eval["protocol_check"]; !ok {
		t.Error("eval evidence missing protocol_check")
	}
	if _, ok := eval["smoke"]; !ok {
		t.Error("eval evidence missing smoke")
	}

	// All phases should have source label.
	for _, phase := range []string{"plan", "review", "eval"} {
		if src, _ := phaseEv[phase]["source"].(string); src != SourceLabel {
			t.Errorf("%s source = %q, want %q", phase, src, SourceLabel)
		}
	}
}

func TestPromoteFromRun_RunIDMismatch(t *testing.T) {
	tmp := t.TempDir()
	evDir := filepath.Join(tmp, ".sdp", "evidence", "wrong-run-id")

	ev := &build.BuildEvidence{
		RunID:     "wrong-run-id",
		Idea:      "test",
		Timestamp: "2026-04-25T12:00:00Z",
		Status:    "success",
	}
	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.MarshalIndent(ev, "", "  ")
	_ = os.WriteFile(filepath.Join(evDir, "evidence.json"), data, 0o644)

	_, err := PromoteFromRun(PromoteOptions{
		RunID:       "correct-run-id",
		FeatureID:   "F135-05",
		EvidenceDir: evDir,
	})
	if err == nil {
		t.Fatal("expected error for RunID mismatch")
	}
}
