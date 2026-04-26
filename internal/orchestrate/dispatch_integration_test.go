package orchestrate_test

import (
	"encoding/json"
	"testing"

	"sdp_dev/internal/dispatch"
	"sdp_dev/internal/orchestrate"
)

func TestWSDispatchInfo_JSON(t *testing.T) {
	info := &orchestrate.WSDispatchInfo{
		Harness:   "claude",
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-20250514",
		Score:     0.92,
		Reason:    "best fit",
		Timestamp: "2026-03-28T12:00:00Z",
		ColdStart: true,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got orchestrate.WSDispatchInfo
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Harness != info.Harness {
		t.Errorf("Harness = %q, want %q", got.Harness, info.Harness)
	}
	if got.Provider != info.Provider {
		t.Errorf("Provider = %q, want %q", got.Provider, info.Provider)
	}
	if got.Model != info.Model {
		t.Errorf("Model = %q, want %q", got.Model, info.Model)
	}
	if got.Score != info.Score {
		t.Errorf("Score = %f, want %f", got.Score, info.Score)
	}
	if got.Reason != info.Reason {
		t.Errorf("Reason = %q, want %q", got.Reason, info.Reason)
	}
	if got.Timestamp != info.Timestamp {
		t.Errorf("Timestamp = %q, want %q", got.Timestamp, info.Timestamp)
	}
	if got.ColdStart != info.ColdStart {
		t.Errorf("ColdStart = %v, want %v", got.ColdStart, info.ColdStart)
	}
}

func TestWSDispatchInfo_JSON_OmitsEmpty(t *testing.T) {
	info := &orchestrate.WSDispatchInfo{
		Harness:  "opencode",
		Provider: "openai",
		Model:    "gpt-4o",
		Score:    0.85,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify omitempty fields are absent
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	for _, key := range []string{"reason", "timestamp", "cold_start"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected key %q to be omitted", key)
		}
	}
}

func TestRecordDispatch(t *testing.T) {
	cp := &orchestrate.Checkpoint{
		FeatureID: "F100",
		Branch:    "feature/F100-test",
		Phase:     orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{
			{ID: "ws-alpha", Status: "pending"},
			{ID: "ws-beta", Status: "in_progress"},
		},
	}

	dec := &dispatch.DispatchDecision{
		Harness:   "claude",
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-20250514",
		Score:     0.95,
		Reason:    "high complexity",
		Timestamp: "2026-03-28T10:00:00Z",
	}

	found := orchestrate.RecordDispatch(cp, "ws-beta", dec)
	if !found {
		t.Fatal("RecordDispatch returned false, expected true")
	}

	// Verify the correct workstream was updated
	if cp.Workstreams[1].Dispatch == nil {
		t.Fatal("Dispatch field not set on ws-beta")
	}
	d := cp.Workstreams[1].Dispatch
	if d.Harness != "claude" {
		t.Errorf("Harness = %q, want %q", d.Harness, "claude")
	}
	if d.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", d.Provider, "anthropic")
	}
	if d.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", d.Model, "claude-sonnet-4-20250514")
	}
	if d.Score != 0.95 {
		t.Errorf("Score = %f, want %f", d.Score, 0.95)
	}

	// Verify other workstream was NOT updated
	if cp.Workstreams[0].Dispatch != nil {
		t.Error("ws-alpha should not have dispatch info")
	}
}

func TestRecordDispatch_NotFound(t *testing.T) {
	cp := &orchestrate.Checkpoint{
		FeatureID: "F100",
		Phase:     orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{
			{ID: "ws-alpha", Status: "pending"},
		},
	}

	dec := &dispatch.DispatchDecision{
		Harness:  "claude",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		Score:    0.9,
	}

	found := orchestrate.RecordDispatch(cp, "ws-nonexistent", dec)
	if found {
		t.Fatal("RecordDispatch returned true for unknown wsID")
	}
}

func TestCheckpoint_WithDispatch(t *testing.T) {
	dir := t.TempDir()

	cp := &orchestrate.Checkpoint{
		Schema:    "orchestrate.v1",
		FeatureID: "F200",
		Branch:    "feature/F200-dispatch",
		Phase:     orchestrate.PhaseBuild,
		Workstreams: []orchestrate.WSStatus{
			{
				ID:     "00-200-01",
				Status: "in_progress",
				Dispatch: &orchestrate.WSDispatchInfo{
					Harness:   "opencode",
					Provider:  "openai",
					Model:     "gpt-4o",
					Score:     0.88,
					Reason:    "default fallback",
					Timestamp: "2026-03-28T09:00:00Z",
				},
			},
			{
				ID:     "00-200-02",
				Status: "pending",
			},
		},
	}

	if err := orchestrate.SaveCheckpoint(dir, cp); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := orchestrate.LoadCheckpoint(dir, "F200")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded.Workstreams) != 2 {
		t.Fatalf("expected 2 workstreams, got %d", len(loaded.Workstreams))
	}

	ws0 := loaded.Workstreams[0]
	if ws0.Dispatch == nil {
		t.Fatal("00-200-01 dispatch info lost after save/load")
	}
	if ws0.Dispatch.Harness != "opencode" {
		t.Errorf("Harness = %q, want %q", ws0.Dispatch.Harness, "opencode")
	}
	if ws0.Dispatch.Score != 0.88 {
		t.Errorf("Score = %f, want %f", ws0.Dispatch.Score, 0.88)
	}
	if ws0.Dispatch.Reason != "default fallback" {
		t.Errorf("Reason = %q, want %q", ws0.Dispatch.Reason, "default fallback")
	}

	ws1 := loaded.Workstreams[1]
	if ws1.Dispatch != nil {
		t.Error("ws-two should have nil dispatch")
	}
}
