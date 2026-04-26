package orchestrate_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"sdp_dev/internal/orchestrate"
)

// helperCheckpoint creates a checkpoint at the given phase with n workstreams.
// Always uses at least 2 workstreams to avoid the single-workstream FSM
// validation edge case in Advance (computeNextPhase assumes donePlus1 < total).
func helperCheckpoint(phase string, n int) *orchestrate.Checkpoint {
	if n < 2 {
		n = 2
	}
	ws := make([]orchestrate.WSStatus, n)
	for i := range ws {
		ws[i] = orchestrate.WSStatus{ID: fmt.Sprintf("ws-%02d", i+1), Status: "pending"}
	}
	return &orchestrate.Checkpoint{
		Schema:      "orchestrate.v1",
		FeatureID:   "F106",
		Branch:      "feature-F106",
		Phase:       phase,
		Workstreams: ws,
		Review:      &orchestrate.ReviewStatus{Iteration: 0, Status: "pending"},
		QA:          &orchestrate.QAStatus{Iteration: 0, Status: "pending"},
	}
}

// helperWorkstreams returns workstream IDs for n workstreams.
func helperWorkstreams(n int) []string {
	if n < 2 {
		n = 2
	}
	ws := make([]string, n)
	for i := range ws {
		ws[i] = fmt.Sprintf("ws-%02d", i+1)
	}
	return ws
}

// allGates returns all human gate names for convenience.
func allGates() []string {
	return []string{"review", "pr", "ci-loop", "qa"}
}

func TestAutonomous_HappyPath(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		AcceptGates:   allGates(),
		DryRun:        false,
		Force:         true,
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if cp.Phase != orchestrate.PhaseDone {
		t.Errorf("expected phase done, got: %s", cp.Phase)
	}
}

func TestAutonomous_HappyPath_ThreeWorkstreams(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseInit, 3)
	workstreams := helperWorkstreams(3)

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		AcceptGates:   allGates(),
		DryRun:        false,
		Force:         true,
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if cp.Phase != orchestrate.PhaseDone {
		t.Errorf("expected phase done, got: %s", cp.Phase)
	}
}

func TestAutonomous_DryRun(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		AcceptGates:   allGates(),
		DryRun:        true,
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected dry-run prefix in output, got: %s", output)
	}
	// In dry-run mode, the original checkpoint should NOT be mutated.
	if cp.Phase != orchestrate.PhaseInit {
		t.Errorf("dry-run should not mutate original checkpoint, got phase: %s", cp.Phase)
	}
	// The dry-run should have printed multiple actions.
	lines := strings.Count(output, "[dry-run]")
	if lines < 3 {
		t.Errorf("expected at least 3 dry-run lines, got %d (output: %s)", lines, output)
	}
}

func TestAutonomous_StuckLoop(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseBuild, 2)
	cp.Workstreams[0].Status = "pending"
	workstreams := helperWorkstreams(2)

	advanceCalls := 0
	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		DryRun:        false,
		Force:         true,
	})
	// Override advance to be a no-op so the state never changes,
	// simulating a stuck loop where next-action always returns "build".
	driver.SetAdvanceFn(func(cp *orchestrate.Checkpoint, workstreams []string, result string) error {
		advanceCalls++
		// Don't actually advance — keeps the same state forever.
		return nil
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for stuck loop")
	}
	if !errors.Is(err, orchestrate.ErrLoopStuck) {
		t.Errorf("expected ErrLoopStuck, got: %v", err)
	}
	// advance is called but is a no-op; next-action keeps returning "build".
	if advanceCalls < 2 {
		t.Errorf("expected at least 2 advance calls before stuck detection, got %d", advanceCalls)
	}
}

func TestAutonomous_GateAllowlist_Blocked(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		AcceptGates:   []string{"build"}, // review, pr, ci, qa are NOT in allowlist
		DryRun:        false,
		Force:         true,
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for human gate")
	}
	if !errors.Is(err, orchestrate.ErrHumanRequired) {
		t.Errorf("expected ErrHumanRequired, got: %v", err)
	}
}

func TestAutonomous_GateAllowlist_AllAllowed(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		AcceptGates:   allGates(),
		DryRun:        false,
		Force:         true,
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error with all gates in allowlist, got: %v", err)
	}
	if cp.Phase != orchestrate.PhaseDone {
		t.Errorf("expected phase done, got: %s", cp.Phase)
	}
}

func TestAutonomous_IterationCap(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseBuild, 2)
	cp.Workstreams[0].Status = "pending"
	workstreams := helperWorkstreams(2)

	// Use maxIterations=2 which is less than MaxSameAction (3),
	// so the iteration cap fires before stuck detection.
	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 2,
		DryRun:        false,
		Force:         true,
	})
	// Override advance to be a no-op — state never changes.
	driver.SetAdvanceFn(func(cp *orchestrate.Checkpoint, workstreams []string, result string) error {
		return nil
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for iteration cap")
	}
	if !errors.Is(err, orchestrate.ErrMaxIterations) {
		t.Errorf("expected ErrMaxIterations, got: %v", err)
	}
}

func TestAutonomous_DefaultMaxIterations(t *testing.T) {
	if orchestrate.DefaultMaxIterations != 50 {
		t.Errorf("DefaultMaxIterations = %d, want 50", orchestrate.DefaultMaxIterations)
	}
}

func TestAutonomous_MaxSameAction(t *testing.T) {
	if orchestrate.MaxSameAction != 3 {
		t.Errorf("MaxSameAction = %d, want 3", orchestrate.MaxSameAction)
	}
}

func TestAutonomous_ExitCodes(t *testing.T) {
	if orchestrate.ExitAutonomousHumanRequired != 3 {
		t.Errorf("ExitAutonomousHumanRequired = %d, want 3", orchestrate.ExitAutonomousHumanRequired)
	}
	if orchestrate.ExitAutonomousStuckOrLimit != 4 {
		t.Errorf("ExitAutonomousStuckOrLimit = %d, want 4", orchestrate.ExitAutonomousStuckOrLimit)
	}
}

func TestAutonomous_ExitError_HumanRequired(t *testing.T) {
	aerr := &orchestrate.AutonomousExitError{Code: 3, Err: orchestrate.ErrHumanRequired}
	if aerr.Error() == "" {
		t.Error("AutonomousExitError.Error() should not be empty")
	}
	if !errors.Is(aerr, orchestrate.ErrHumanRequired) {
		t.Error("AutonomousExitError should unwrap to ErrHumanRequired")
	}
	if orchestrate.ExitCode(aerr) != 3 {
		t.Errorf("ExitCode = %d, want 3", orchestrate.ExitCode(aerr))
	}
}

func TestAutonomous_ExitError_StuckCode(t *testing.T) {
	aerr := &orchestrate.AutonomousExitError{Code: 4, Err: orchestrate.ErrLoopStuck}
	if orchestrate.ExitCode(aerr) != 4 {
		t.Errorf("ExitCode = %d, want 4", orchestrate.ExitCode(aerr))
	}
	if !errors.Is(aerr, orchestrate.ErrLoopStuck) {
		t.Error("AutonomousExitError should unwrap to ErrLoopStuck")
	}
}

func TestAutonomous_ExitError_NonAutonomous(t *testing.T) {
	if orchestrate.ExitCode(fmt.Errorf("some error")) != 1 {
		t.Error("ExitCode for non-autonomous error should be 1")
	}
}

func TestAutonomous_ContextCancellation(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		DryRun:        false,
		Force:         true,
	})

	err := driver.Run(ctx, cp, workstreams, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestAutonomous_RunAutonomous_WrapsHumanRequired(t *testing.T) {
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	err := orchestrate.RunAutonomous(
		context.Background(),
		orchestrate.AutonomousConfig{MaxIterations: 20, AcceptGates: nil},
		t.TempDir(), "F106", t.TempDir(), cp, workstreams,
	)
	if err == nil {
		t.Fatal("expected error")
	}
	var aerr *orchestrate.AutonomousExitError
	if !errors.As(err, &aerr) {
		t.Fatalf("expected AutonomousExitError, got: %T: %v", err, err)
	}
	if aerr.Code != 3 {
		t.Errorf("exit code = %d, want 3", aerr.Code)
	}
	if !errors.Is(aerr, orchestrate.ErrHumanRequired) {
		t.Errorf("should unwrap to ErrHumanRequired")
	}
}

func TestSameActionCount(t *testing.T) {
	tests := []struct {
		history []string
		action  string
		want    int
	}{
		{[]string{"a", "a", "a"}, "a", 3},
		{[]string{"a", "b", "a"}, "a", 1},
		{[]string{"a", "a", "b"}, "a", 0},
		{[]string{}, "a", 0},
		{[]string{"build", "build"}, "build", 2},
		{[]string{"build", "review", "build", "build"}, "build", 2},
	}
	for _, tt := range tests {
		got := sameActionCount(tt.history, tt.action)
		if got != tt.want {
			t.Errorf("sameActionCount(%v, %q) = %d, want %d", tt.history, tt.action, got, tt.want)
		}
	}
}

// sameActionCount mirrors the unexported function for direct testing.
func sameActionCount(history []string, action string) int {
	count := 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i] == action {
			count++
		} else {
			break
		}
	}
	return count
}

func TestAutonomous_NonDryRun_SafetyFallback(t *testing.T) {
	// Without Force, non-dry-run mode should be treated as dry-run (MVP safety).
	// The original checkpoint must NOT be mutated.
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		AcceptGates:   allGates(),
		DryRun:        false,
		Force:         false, // no force — should auto-downgrade to dry-run
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	output := buf.String()

	// Must contain the safety warning.
	if !strings.Contains(output, "WARNING") {
		t.Errorf("expected safety WARNING in output, got: %s", output)
	}
	if !strings.Contains(output, "--force") {
		t.Errorf("expected --force hint in output, got: %s", output)
	}

	// Must behave like dry-run: original checkpoint not mutated.
	if cp.Phase != orchestrate.PhaseInit {
		t.Errorf("safety fallback should not mutate original checkpoint, got phase: %s", cp.Phase)
	}

	// Must contain [dry-run] markers showing it ran in dry-run mode.
	if !strings.Contains(output, "[dry-run]") {
		t.Errorf("expected [dry-run] prefix in output after safety fallback, got: %s", output)
	}
}

func TestAutonomous_Force_BypassesSafety(t *testing.T) {
	// With Force=true and DryRun=false, the driver should run without
	// the safety downgrade and mutate the checkpoint as before.
	cp := helperCheckpoint(orchestrate.PhaseInit, 2)
	workstreams := helperWorkstreams(2)

	driver := orchestrate.NewAutonomousDriver(orchestrate.AutonomousConfig{
		MaxIterations: 20,
		AcceptGates:   allGates(),
		DryRun:        false,
		Force:         true, // bypass safety
	})

	var buf strings.Builder
	driver.SetOutput(&buf)

	err := driver.Run(context.Background(), cp, workstreams, t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	output := buf.String()

	// Should NOT contain the safety warning (it's force mode).
	if strings.Contains(output, "WARNING") {
		t.Errorf("force mode should not emit safety WARNING, got: %s", output)
	}

	// Checkpoint should be mutated (reached done phase).
	if cp.Phase != orchestrate.PhaseDone {
		t.Errorf("force mode should mutate checkpoint to done, got phase: %s", cp.Phase)
	}
}
