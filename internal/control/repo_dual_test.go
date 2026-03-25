package control

import (
	"context"
	"testing"
	"time"
)

func TestDriftReport_EmptyStores(t *testing.T) {
	report := &DriftReport{
		GeneratedAt: time.Now().UTC(),
	}

	if report.TotalPrimary != 0 {
		t.Errorf("expected 0 primary, got %d", report.TotalPrimary)
	}
	if len(report.MissingShadow) != 0 {
		t.Errorf("expected no missing, got %d", len(report.MissingShadow))
	}
}

func TestDualWriteRepository_Compare_NilShadow(t *testing.T) {
	tmp := t.TempDir()
	primary := NewFileCardRepository(tmp, tmp, ProjectRegistry{})

	// Directly write a card file (bypass CreateCard ID requirement)
	card := &FeatureCard{
		ID:        "F-TEST-001",
		ProjectID: "test",
		Title:     "Only in primary",
		Status:    "open",
	}
	_ = primary.CreateCard("test", card)

	// nil shadow — should handle gracefully
	dual := NewDualWriteRepository(primary, nil, nil)

	report, err := dual.Compare(context.Background(), "test")
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	if report.TotalPrimary != 1 {
		t.Errorf("total primary: got %d, want 1", report.TotalPrimary)
	}
	if report.TotalShadow != 0 {
		t.Errorf("total shadow: got %d, want 0", report.TotalShadow)
	}
	if len(report.MissingShadow) != 1 {
		t.Errorf("missing in shadow: got %d, want 1", len(report.MissingShadow))
	}
	if report.MissingShadow[0] != "F-TEST-001" {
		t.Errorf("missing ID: got %s, want F-TEST-001", report.MissingShadow[0])
	}
}

func TestStatusMismatch_Struct(t *testing.T) {
	m := StatusMismatch{
		ID:      "test-123",
		Title:   "Test",
		Primary: "open",
		Shadow:  "closed",
	}

	if m.Primary != "open" || m.Shadow != "closed" {
		t.Error("mismatch values wrong")
	}
}
