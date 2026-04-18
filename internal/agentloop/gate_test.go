package agentloop

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sdp_dev/internal/harness"
)

// ---- helpers ----

// minimalContract returns a TaskContract with all gates disabled (any snapshot passes).
func minimalContract() *harness.TaskContract {
	return &harness.TaskContract{
		Version: "1.0",
		QualityGates: harness.QualityGates{
			Build:     false,
			Test:      false,
			Lint:      false,
			Typecheck: false,
		},
		// No required evidence, no required metrics, no acceptance criteria.
	}
}

// blockingContract returns a TaskContract requiring evidence that will never be in the snapshot.
func blockingContract() *harness.TaskContract {
	return &harness.TaskContract{
		Version:          "1.0",
		RequiredEvidence: []string{"proof_of_work"},
	}
}

func emptySnapshot(phase Role) PhaseSnapshot {
	return PhaseSnapshot{
		Phase:    phase,
		Evidence: nil,
		Quality:  make(map[string]bool),
	}
}

// ---- tests ----

// TestGateEngine_pass: compliance passes → GateResult.Escalated = false.
func TestGateEngine_pass(t *testing.T) {
	engine := NewGateEngine(minimalContract(), 5*time.Second)

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	assert.False(t, result.Escalated, "gate must not escalate when compliance passes")
	assert.False(t, result.Report.Blocked, "report must not be blocked when compliance passes")
}

// TestGateEngine_blocked: compliance is blocked → GateResult.Escalated = true.
func TestGateEngine_blocked(t *testing.T) {
	// blockingContract requires "proof_of_work" evidence — emptySnapshot has none.
	engine := NewGateEngine(blockingContract(), 5*time.Second)

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	assert.True(t, result.Report.Blocked, "report must be blocked when required evidence is missing")
	assert.True(t, result.Escalated, "blocked report must set Escalated=true (requires human decision)")
}

// TestGateEngine_timeout: evalFn takes longer than timeout → GateWarn violation, Escalated=true.
// Fix R2-3: timeout is NOT automatic pass — it triggers escalation.
func TestGateEngine_timeout(t *testing.T) {
	// Use a very short timeout to trigger it reliably in tests.
	engine := NewGateEngine(minimalContract(), 10*time.Millisecond)

	// Override the evalFn with one that blocks longer than the timeout.
	// We test via a custom engine variant with injectable eval function.
	var called atomic.Bool
	engine.evalFn = func(contract *harness.TaskContract, snap *harness.TaskSnapshot) harness.ComplianceReport {
		called.Store(true)
		time.Sleep(200 * time.Millisecond) // longer than 10ms timeout
		return harness.ComplianceReport{Blocked: false}
	}

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	// Timeout must escalate, NOT silently pass.
	assert.True(t, result.Escalated,
		"Fix R2-3: gate timeout must escalate (require human decision), not auto-pass")
	assert.False(t, result.Report.Blocked,
		"timeout sets Escalated without Blocked — human reviews, automation doesn't block")

	// Must have a GateWarn violation explaining the timeout.
	require.NotEmpty(t, result.Report.GateResults, "timeout result must contain gate results")
	found := false
	for _, gr := range result.Report.GateResults {
		if gr.Status == harness.GateWarn {
			for _, v := range gr.Violations {
				if v.Type == harness.DriftProcessIncomplete {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "timeout must produce a GateWarn violation with DriftProcessIncomplete type")
}

// TestGateEngine_nilContract: nil contract → auto-pass (MVP bypass mode).
// When no TaskContract is configured, the gate cannot evaluate compliance,
// so it passes unconditionally. Production deployments should provide a contract.
func TestGateEngine_nilContract(t *testing.T) {
	engine := NewGateEngine(nil, 5*time.Second)

	snap := emptySnapshot(RoleDiscover)
	result := engine.Evaluate(context.Background(), snap)

	// MVP bypass: nil contract auto-passes (not blocked, not escalated).
	assert.False(t, result.Report.Blocked)
	assert.False(t, result.Escalated)
}

// TestGateEngine_defaultTimeout: NewGateEngine with zero timeout uses 5s default.
func TestGateEngine_defaultTimeout(t *testing.T) {
	engine := NewGateEngine(minimalContract(), 0)
	assert.Equal(t, 5*time.Second, engine.timeout, "zero timeout must default to 5s")
}

// TestGateEngine_contextAlreadyCancelled: cancelled context returns escalated result immediately.
func TestGateEngine_contextAlreadyCancelled(t *testing.T) {
	engine := NewGateEngine(minimalContract(), 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling Evaluate

	snap := emptySnapshot(RoleDiscover)
	// Must not hang; returns escalated result.
	done := make(chan GateResult, 1)
	go func() {
		done <- engine.Evaluate(ctx, snap)
	}()

	select {
	case result := <-done:
		// Either the ctx cancellation was noticed immediately or EvaluateCompliance finished first.
		// Either way, function must return promptly.
		_ = result
	case <-time.After(2 * time.Second):
		t.Fatal("Evaluate blocked too long with cancelled context")
	}
}
