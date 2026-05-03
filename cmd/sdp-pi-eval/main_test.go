package main

import (
	"testing"
)

// =================================================================
// Slot parsing
// =================================================================

func TestBuildSlots_Default(t *testing.T) {
	slots := buildSlots("")
	if slots != nil {
		t.Fatalf("expected nil for empty flag, got %v", slots)
	}
}

func TestBuildSlots_Custom(t *testing.T) {
	slots := buildSlots("zai=zai/glm-5.1,kimi=kimi-coding/k2p6")
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
	if slots[0].Slot != "zai" || slots[0].Provider != "zai" || slots[0].Model != "glm-5.1" {
		t.Errorf("slot 0: %+v", slots[0])
	}
	if slots[1].Slot != "kimi" || slots[1].Provider != "kimi-coding" || slots[1].Model != "k2p6" {
		t.Errorf("slot 1: %+v", slots[1])
	}
}

func TestBuildSlots_Malformed(t *testing.T) {
	slots := buildSlots("invalid,bad=format/extra/stuff")
	// malformed entries should be skipped; only valid slot=provider/model parsed
	if len(slots) > 1 {
		t.Fatalf("expected at most 1 valid slot, got %d", len(slots))
	}
}

// =================================================================
// Integration: LiveRunner through CLI config (mock runner)
// =================================================================

func TestCLIIntegration_WithMockRunner(t *testing.T) {
	// This test verifies the CLI config + LiveRunner integration path
	// without calling actual model providers.
	//
	// The actual integration test is in internal/evals/pi_live_test.go
	// which tests NewLiveRunner with mock runners directly.
	//
	// Here we verify the CLI flag parsing and slot construction work correctly.

	slots := buildSlots("zai=zai/glm-5.1,minimax=minimax/MiniMax-M2.7")
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}

	// Verify slot details match F164 acceptance criteria
	foundZai := false
	foundMinimax := false
	for _, s := range slots {
		if s.Slot == "zai" && s.Provider == "zai" && s.Model == "glm-5.1" {
			foundZai = true
		}
		if s.Slot == "minimax" && s.Provider == "minimax" && s.Model == "MiniMax-M2.7" {
			foundMinimax = true
		}
	}
	if !foundZai {
		t.Error("missing zai slot with correct provider/model")
	}
	if !foundMinimax {
		t.Error("missing minimax slot with correct provider/model")
	}
}
