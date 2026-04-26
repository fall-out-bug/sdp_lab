package promote

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/build"
)

// TestBuildPromoteIntegration tests the full flow:
// vibecode build → evidence → promote → Phase FSM artifacts.
func TestBuildPromoteIntegration(t *testing.T) {
	tmp := t.TempDir()
	runID := "integration-test-run-001"
	evDir := filepath.Join(tmp, ".sdp", "evidence", runID)
	phaseDir := filepath.Join(tmp, ".sdp", "phases", runID)

	// Simulate a completed vibecode build by writing evidence.
	ev := &build.BuildEvidence{
		RunID:     runID,
		Idea:      "add user authentication",
		Timestamp: "2026-04-25T12:00:00Z",
		Status:    "success",
		Stages: []build.StageEvidence{
			{Name: "dispatch", Status: "success", Duration: "50ms"},
			{Name: "sandbox", Status: "success", Duration: "3.2s"},
			{Name: "commit", Status: "success", Duration: "10ms"},
		},
		Dispatch: build.DispatchEvidence{
			Harness:  "claude-code",
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			Score:    0.92,
			Reason:   "auto-classified as feature/medium",
		},
		Sandbox: build.SandboxEvidence{
			Type:    "docker",
			BuildOK: true,
			TestsOK: true,
		},
	}

	if err := os.MkdirAll(evDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, _ := json.MarshalIndent(ev, "", "  ")
	if err := os.WriteFile(filepath.Join(evDir, "evidence.json"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Promote to strict Phase FSM.
	result, err := PromoteFromRun(PromoteOptions{
		RunID:       runID,
		FeatureID:   "F135-05",
		EvidenceDir: evDir,
		PhaseDir:    phaseDir,
	})
	if err != nil {
		t.Fatalf("PromoteFromRun: %v", err)
	}

	// AC2: plan/review/eval delta.md files must exist.
	for _, phase := range []string{"plan", "review", "eval"} {
		deltaPath := filepath.Join(phaseDir, phase+".delta.md")
		if _, err := os.Stat(deltaPath); os.IsNotExist(err) {
			t.Errorf("AC2: delta file missing: %s", deltaPath)
		}
		content, err := os.ReadFile(deltaPath)
		if err != nil {
			t.Errorf("read delta: %v", err)
			continue
		}
		// Delta must contain vibecode-promoted source label.
		s := string(content)
		if !contains(s, "vibecode-promoted") {
			t.Errorf("AC2: %s delta missing source label", phase)
		}
	}

	// AC3: Gates must have evidence paths pointing to promoted evidence.
	for _, g := range result.Gates {
		gateData, err := os.ReadFile(g.Path)
		if err != nil {
			t.Fatalf("read gate %s: %v", g.Path, err)
		}
		var gateMap map[string]interface{}
		_ = json.Unmarshal(gateData, &gateMap)

		evPath, _ := gateMap["evidence_path"].(string)
		if evPath == "" {
			t.Errorf("AC3: gate %s has no evidence_path", g.ID)
		}

		// Gate must be in pending state (awaiting human approval).
		if _, ok := gateMap["resolved_at"]; ok {
			t.Errorf("AC3: gate %s should be AWAITING (no resolved_at)", g.ID)
		}
	}

	// AC4: Per-phase evidence files must be valid JSON with required keys.
	for _, phase := range []string{"plan", "review", "eval"} {
		evPath := filepath.Join(phaseDir, phase+".evidence.json")
		evData, err := os.ReadFile(evPath)
		if err != nil {
			t.Errorf("AC4: evidence file missing: %s", evPath)
			continue
		}
		var evMap map[string]interface{}
		if err := json.Unmarshal(evData, &evMap); err != nil {
			t.Errorf("AC4: invalid JSON in %s: %v", evPath, err)
			continue
		}
		// All promoted evidence must have source label.
		if src, _ := evMap["source"].(string); src != "vibecode-promoted" {
			t.Errorf("AC4: %s evidence missing source label", phase)
		}
	}

	// AC5: No work duplication — evidence dir is only read, not modified.
	evOriginal, _ := os.ReadFile(filepath.Join(evDir, "evidence.json"))
	if string(evOriginal) != string(data) {
		t.Error("AC5: original evidence file was modified (work duplication)")
	}

	// Verify promotion trace.
	tracePath := filepath.Join(phaseDir, "promotion-trace.json")
	traceData, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("trace file missing: %v", err)
	}
	var trace map[string]interface{}
	_ = json.Unmarshal(traceData, &trace)
	if deltas, _ := trace["deltas"].(float64); int(deltas) != 3 {
		t.Errorf("trace deltas = %v, want 3", deltas)
	}
	if gates, _ := trace["gates"].(float64); int(gates) != 3 {
		t.Errorf("trace gates = %v, want 3", gates)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
