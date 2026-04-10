package model

import "testing"

func TestIsValidEntityType(t *testing.T) {
	tests := []struct {
		input    EntityType
		expected bool
	}{
		{EntityGoal, true},
		{EntityTask, true},
		{EntityType("unknown"), false},
		{EntityType(""), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			if got := IsValidEntityType(tt.input); got != tt.expected {
				t.Errorf("IsValidEntityType(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFinding_ComputeConfidence_Structural(t *testing.T) {
	f := &Finding{Type: FindingGap}
	score := f.ComputeConfidence()
	if score != 1.0 {
		t.Errorf("structural finding confidence = %f, want 1.0", score)
	}
}

func TestFinding_ComputeConfidence_LLMHigh(t *testing.T) {
	f := &Finding{
		Type:             FindingAlignment,
		LLMScore:         LLMScoreHigh,
		EvidenceVerified: true,
		EvidenceCount:    2,
		SupportRatio:     0.8,
		CrossModelStatus: CrossModelConfirmed,
	}
	score := f.ComputeConfidence()
	if score < 0.7 {
		t.Errorf("high confidence finding = %f, want >= 0.7", score)
	}
}

func TestFinding_ComputeConfidence_Abstain(t *testing.T) {
	f := &Finding{Type: FindingConflict, LLMScore: LLMScoreAbstain}
	score := f.ComputeConfidence()
	if score != 0.0 {
		t.Errorf("abstain confidence = %f, want 0.0", score)
	}
}

func TestFinding_ComputeConfidence_InferredCap(t *testing.T) {
	f := &Finding{
		Type:             FindingInferredStrategy,
		LLMScore:         LLMScoreHigh,
		EvidenceVerified: true,
		EvidenceCount:    3,
		SupportRatio:     1.0,
		CrossModelStatus: CrossModelConfirmed,
	}
	score := f.ComputeConfidence()
	if score > 0.7 {
		t.Errorf("inferred strategy cap violated: %f > 0.7", score)
	}
}

func TestFinding_Tier(t *testing.T) {
	tests := []struct {
		score float64
		tier  ConfidenceTier
	}{
		{0.8, TierHigh},
		{0.5, TierMedium},
		{0.2, TierLow},
	}
	for _, tt := range tests {
		f := &Finding{ConfidenceScore: tt.score}
		if got := f.Tier(); got != tt.tier {
			t.Errorf("Tier(%f) = %q, want %q", tt.score, got, tt.tier)
		}
	}
}
