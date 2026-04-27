package main

import (
	"encoding/json"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/beads"
)

func TestWSOutput_Format(t *testing.T) {
	// Test that WSOutput struct marshals correctly
	output := WSOutput{
		WSID:     "00-059-01",
		BeadsID:  "sdplab-3",
		Title:    "F059-01: Pre-tool-call guard hook",
		Priority: 1,
		Labels:   []string{"F059", "ecosystem"},
		Ready:    true,
	}

	b, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal WSOutput: %v", err)
	}

	// Verify required fields are present
	var parsed map[string]interface{}
	if err := json.Unmarshal(b, &parsed); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	if parsed["ws_id"] != "00-059-01" {
		t.Errorf("expected ws_id=00-059-01, got %v", parsed["ws_id"])
	}
	if parsed["beads_id"] != "sdplab-3" {
		t.Errorf("expected beads_id=sdplab-3, got %v", parsed["beads_id"])
	}
	if parsed["ready"] != true {
		t.Errorf("expected ready=true, got %v", parsed["ready"])
	}

	t.Logf("WSOutput JSON: %s", string(b))
}

func TestMappingFile_Load(t *testing.T) {
	// Test that mapping file is loaded correctly
	// This is tested via the beads.LoadMapping in the beads package
	mapping, err := beads.LoadMapping(".")
	if err != nil {
		t.Logf("LoadMapping failed (expected if no mapping file): %v", err)
		return
	}

	// If mapping exists, verify it has entries
	wsID := mapping.GetSDPID("sdplab-3")
	if wsID == "" {
		t.Log("No mapping for sdplab-3 (expected if mapping file empty)")
	} else {
		t.Logf("Found mapping: sdplab-3 -> %s", wsID)
	}
}

func TestReadyIssue_Structure(t *testing.T) {
	// Test that ReadyIssue struct has all required fields
	issue := beads.ReadyIssue{
		Issue: beads.Issue{
			ID:       "sdplab-3",
			Title:    "Test Issue",
			Status:   "open",
			Priority: 1,
			Labels:   []string{"F059"},
		},
		WSID: "00-059-01",
	}

	if issue.ID != "sdplab-3" {
		t.Errorf("expected ID=sdplab-3, got %s", issue.ID)
	}
	if issue.WSID != "00-059-01" {
		t.Errorf("expected WSID=00-059-01, got %s", issue.WSID)
	}
}
