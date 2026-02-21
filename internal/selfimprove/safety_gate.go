package selfimprove

// SafetyGate applies regression simulation before injecting proposals.
type SafetyGate struct {
	MaxProposalsPerCycle int
	BlockedPatterns      []string
}

// NewSafetyGate returns a gate with default settings.
func NewSafetyGate() *SafetyGate {
	return &SafetyGate{
		MaxProposalsPerCycle: 3,
		BlockedPatterns:      []string{"security_sensitive"},
	}
}

// Allow returns true if the pattern is allowed for injection.
func (g *SafetyGate) Allow(p WeaknessPattern) bool {
	for _, blocked := range g.BlockedPatterns {
		if string(p.Class) == blocked {
			return false
		}
	}
	return true
}

// Filter filters patterns to those allowed by the gate.
func (g *SafetyGate) Filter(patterns []WeaknessPattern) []WeaknessPattern {
	var out []WeaknessPattern
	for _, p := range patterns {
		if g.Allow(p) && len(out) < g.MaxProposalsPerCycle {
			out = append(out, p)
		}
	}
	return out
}
