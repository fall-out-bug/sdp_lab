package evals

// UnsafeDemoRunner is intentionally vulnerable test machinery for F165 demos.
// It simulates naive agent behavior that ignores trust boundaries.
//
// SAFETY: This code is deterministic, uses mock state only, and cannot call
// live Beads, Git, network, or filesystem-write tools. It is clearly labeled
// unsafe and is intended to produce expected-fail outputs for defensive tests.
//
// Do NOT use this in production code paths.
type UnsafeDemoRunner struct{}

// NewUnsafeDemoRunner creates a new unsafe demo runner.
func NewUnsafeDemoRunner() *UnsafeDemoRunner {
	return &UnsafeDemoRunner{}
}

// RunCase deterministically produces the naive (unsafe) output for a fixture.
// It ignores trust boundaries, normalization, and validation — simulating
// what would happen if untrusted narrative were treated as instruction.
func (u *UnsafeDemoRunner) RunCase(c IndirectPICase) DefendedResult {
	// Naive behavior: trust the untrusted artifact and produce the unsafe claim.
	return DefendedResult{
		Verdict:       "clean",
		BlockedReason: "",
		TrustedEvidenceRef: "",
	}
}

// UnsafeDemoAction records what action the naive runner would take.
// This is for test assertions only.
func (u *UnsafeDemoRunner) UnsafeDemoAction(c IndirectPICase) string {
	return c.ExpectedUnsafeResult.UnsafeAction
}

// UnsafeDemoClaim records what claim the naive runner would emit.
// This is for test assertions only.
func (u *UnsafeDemoRunner) UnsafeDemoClaim(c IndirectPICase) string {
	return c.ExpectedUnsafeResult.UnsafeClaim
}

// UnsafeDemoToolRegistry is a fake tool registry used to prove isolation.
// All write-capable tools record a denied mock event instead of mutating state.
type UnsafeDemoToolRegistry struct {
	Calls []string
}

// NewUnsafeDemoToolRegistry creates a new fake tool registry.
func NewUnsafeDemoToolRegistry() *UnsafeDemoToolRegistry {
	return &UnsafeDemoToolRegistry{Calls: []string{}}
}

// Call records the tool name and returns a mock denied result for write tools.
func (r *UnsafeDemoToolRegistry) Call(toolName string) string {
	r.Calls = append(r.Calls, toolName)
	if isWriteAction(toolName) {
		return "DENIED_MOCK"
	}
	return "ALLOWED_MOCK"
}

// HasCall reports whether the registry recorded a call to the given tool.
func (r *UnsafeDemoToolRegistry) HasCall(toolName string) bool {
	for _, c := range r.Calls {
		if c == toolName {
			return true
		}
	}
	return false
}
