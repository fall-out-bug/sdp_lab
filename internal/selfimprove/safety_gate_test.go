package selfimprove

import (
	"testing"
)

func TestNewSafetyGate(t *testing.T) {
	g := NewSafetyGate()
	if g == nil || g.MaxProposalsPerCycle != 3 {
		t.Fatalf("NewSafetyGate: got %+v", g)
	}
	if len(g.BlockedPatterns) == 0 {
		t.Error("expected default BlockedPatterns")
	}
}

func TestSafetyGate_Allow(t *testing.T) {
	g := NewSafetyGate()

	if !g.Allow(WeaknessPattern{Class: ClassTransient}) {
		t.Error("Transient should be allowed")
	}
	if g.Allow(WeaknessPattern{Class: ClassSecuritySensitive}) {
		t.Error("SecuritySensitive should be blocked")
	}
}

func TestSafetyGate_Filter(t *testing.T) {
	g := NewSafetyGate()
	patterns := []WeaknessPattern{
		{ID: "1", Class: ClassTransient},
		{ID: "2", Class: ClassSecuritySensitive},
		{ID: "3", Class: ClassToolFlake},
	}
	out := g.Filter(patterns)
	if len(out) != 2 {
		t.Errorf("expected 2 (SecuritySensitive blocked), got %d", len(out))
	}
	for _, p := range out {
		if p.Class == ClassSecuritySensitive {
			t.Error("SecuritySensitive should be filtered out")
		}
	}
}
