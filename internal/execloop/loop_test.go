package execloop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"sdp_dev/internal/build"
)

// --- Test sandboxes ---

// successSandbox always succeeds on build and test.
type successSandbox struct{}

func (s *successSandbox) Build(_ context.Context, _ string) (*build.SandboxResult, error) {
	return &build.SandboxResult{Success: true, ExitCode: 0}, nil
}
func (s *successSandbox) Test(_ context.Context, _ string) (*build.SandboxResult, error) {
	return &build.SandboxResult{Success: true, ExitCode: 0}, nil
}
func (s *successSandbox) Cleanup() error { return nil }

// failSandbox always succeeds on build but fails on test.
type failSandbox struct{}

func (s *failSandbox) Build(_ context.Context, _ string) (*build.SandboxResult, error) {
	return &build.SandboxResult{Success: true, ExitCode: 0}, nil
}
func (s *failSandbox) Test(_ context.Context, _ string) (*build.SandboxResult, error) {
	return &build.SandboxResult{
		Success:  false,
		ExitCode: 1,
		Stderr:   "test failure",
	}, nil
}
func (s *failSandbox) Cleanup() error { return nil }

// counterSandbox fails test for the first N attempts, then succeeds.
// Used to simulate the agent fixing code between attempts.
type counterSandbox struct {
	failCount int32 // number of Test calls that fail before success
	calls     int32 // number of Test calls so far (atomic)
}

func (s *counterSandbox) Build(_ context.Context, _ string) (*build.SandboxResult, error) {
	return &build.SandboxResult{Success: true, ExitCode: 0}, nil
}
func (s *counterSandbox) Test(_ context.Context, _ string) (*build.SandboxResult, error) {
	n := atomic.AddInt32(&s.calls, 1)
	if n <= s.failCount {
		return &build.SandboxResult{
			Success:  false,
			ExitCode: 1,
			Stderr:   fmt.Sprintf("test failure (attempt %d)", n),
		}, nil
	}
	return &build.SandboxResult{Success: true, ExitCode: 0}, nil
}
func (s *counterSandbox) Cleanup() error { return nil }

// buildFailSandbox always fails on build.
type buildFailSandbox struct{}

func (s *buildFailSandbox) Build(_ context.Context, _ string) (*build.SandboxResult, error) {
	return &build.SandboxResult{
		Success:  false,
		ExitCode: 2,
		Stderr:   "build error",
	}, nil
}
func (s *buildFailSandbox) Test(_ context.Context, _ string) (*build.SandboxResult, error) {
	return &build.SandboxResult{Success: true, ExitCode: 0}, nil
}
func (s *buildFailSandbox) Cleanup() error { return nil }

// errorSandbox returns an error from Build.
type errorSandbox struct{}

func (s *errorSandbox) Build(_ context.Context, _ string) (*build.SandboxResult, error) {
	return nil, fmt.Errorf("sandbox infrastructure error")
}
func (s *errorSandbox) Test(_ context.Context, _ string) (*build.SandboxResult, error) {
	return nil, nil
}
func (s *errorSandbox) Cleanup() error { return nil }

// --- Config validation tests ---

func TestNewDefaultLoop_RejectsZeroAttempts(t *testing.T) {
	cfg := Config{
		MaxAttempts: 0,
		Sandbox:     &successSandbox{},
	}
	_, err := NewDefaultLoop(cfg)
	if err == nil {
		t.Fatal("expected error for MaxAttempts=0")
	}
}

func TestNewDefaultLoop_RejectsNegativeAttempts(t *testing.T) {
	cfg := Config{
		MaxAttempts: -1,
		Sandbox:     &successSandbox{},
	}
	_, err := NewDefaultLoop(cfg)
	if err == nil {
		t.Fatal("expected error for MaxAttempts=-1")
	}
}

func TestNewDefaultLoop_RejectsNilSandbox(t *testing.T) {
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     nil,
	}
	_, err := NewDefaultLoop(cfg)
	if err == nil {
		t.Fatal("expected error for nil Sandbox")
	}
}

func TestNewDefaultLoop_ValidConfig(t *testing.T) {
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &successSandbox{},
		RunID:       "test-valid",
	}
	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loop == nil {
		t.Fatal("loop should not be nil")
	}
	if loop.cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", loop.cfg.MaxAttempts)
	}
}

// --- Convergence tests ---

func TestLoop_ConvergesOnFirstAttempt(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &successSandbox{},
		RunID:       "test-converge-1",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "converged" {
		t.Errorf("Status = %q, want %q", result.Status, "converged")
	}
	if result.Winner != 1 {
		t.Errorf("Winner = %d, want 1", result.Winner)
	}
	if len(result.Attempts) != 1 {
		t.Fatalf("len(Attempts) = %d, want 1", len(result.Attempts))
	}
	if result.Attempts[0].Status != "success" {
		t.Errorf("Attempts[0].Status = %q, want %q", result.Attempts[0].Status, "success")
	}
	if !result.Attempts[0].BuildOK {
		t.Error("Attempts[0].BuildOK should be true")
	}
	if !result.Attempts[0].TestsOK {
		t.Error("Attempts[0].TestsOK should be true")
	}
	if result.TotalTime == 0 {
		t.Error("TotalTime should be non-zero")
	}
}

func TestLoop_ConvergesOnSecondAttempt(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &counterSandbox{failCount: 1},
		RunID:       "test-converge-2",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "converged" {
		t.Errorf("Status = %q, want %q", result.Status, "converged")
	}
	if result.Winner != 2 {
		t.Errorf("Winner = %d, want 2", result.Winner)
	}
	if len(result.Attempts) != 2 {
		t.Fatalf("len(Attempts) = %d, want 2", len(result.Attempts))
	}
	if result.Attempts[0].Status != "failed" {
		t.Errorf("Attempts[0].Status = %q, want %q", result.Attempts[0].Status, "failed")
	}
	if result.Attempts[1].Status != "success" {
		t.Errorf("Attempts[1].Status = %q, want %q", result.Attempts[1].Status, "success")
	}
}

// --- Max attempts tests ---

func TestLoop_ExitsOnMaxAttempts(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 2,
		Sandbox:     &failSandbox{},
		RunID:       "test-max-attempts",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "max_attempts" {
		t.Errorf("Status = %q, want %q", result.Status, "max_attempts")
	}
	if result.Winner != 0 {
		t.Errorf("Winner = %d, want 0 (no winner)", result.Winner)
	}
	if len(result.Attempts) != 2 {
		t.Fatalf("len(Attempts) = %d, want 2", len(result.Attempts))
	}
	for i, ar := range result.Attempts {
		if ar.Status != "failed" {
			t.Errorf("Attempts[%d].Status = %q, want %q", i, ar.Status, "failed")
		}
	}
}

func TestLoop_BuildFailure(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 2,
		Sandbox:     &buildFailSandbox{},
		RunID:       "test-build-fail",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "max_attempts" {
		t.Errorf("Status = %q, want %q", result.Status, "max_attempts")
	}
	for _, ar := range result.Attempts {
		if ar.BuildOK {
			t.Error("BuildOK should be false")
		}
	}
}

// --- Context cancellation tests ---

func TestLoop_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 5,
		Sandbox:     &failSandbox{},
		RunID:       "test-cancel",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	result, err := loop.Run(ctx)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if result == nil {
		t.Fatal("result should not be nil even on cancellation")
	}
	if result.Status != "error" {
		t.Errorf("Status = %q, want %q", result.Status, "error")
	}
}

// --- Scope creep tests ---

func TestLoop_ScopeCreepDetection(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Create files and commit them so git has a clean baseline.
	scopeFile := filepath.Join(dir, "expected.go")
	if err := os.WriteFile(scopeFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write expected.go: %v", err)
	}
	outsideFile := filepath.Join(dir, "outside.go")
	if err := os.WriteFile(outsideFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write outside.go: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	// Modify the outside file so it shows up in git diff --name-only.
	if err := os.WriteFile(outsideFile, []byte("package main\n// modified\n"), 0o644); err != nil {
		t.Fatalf("write modified outside.go: %v", err)
	}

	outDir := filepath.Join(dir, ".sdp", "evidence")
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &successSandbox{},
		RunID:       "test-scope-creep",
		OutputDir:   outDir,
		RepoPath:    dir,
		ScopeFiles:  []string{"expected.go"},
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "scope_creep" {
		t.Errorf("Status = %q, want %q", result.Status, "scope_creep")
	}
	if len(result.Attempts) != 1 {
		t.Fatalf("len(Attempts) = %d, want 1 (scope creep stops immediately)", len(result.Attempts))
	}
	if result.Attempts[0].Status != "scope_creep" {
		t.Errorf("Attempts[0].Status = %q, want %q", result.Attempts[0].Status, "scope_creep")
	}
	if result.Attempts[0].Diff == "" {
		t.Error("Attempts[0].Diff should not be empty when scope creep detected")
	}
}

func TestLoop_NoScopeCreepCheckWhenScopeFilesEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &successSandbox{},
		RunID:       "test-no-scope",
		OutputDir:   dir,
		ScopeFiles:  nil,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "converged" {
		t.Errorf("Status = %q, want %q", result.Status, "converged")
	}
}

func TestLoop_ScopeCreepAllFilesInScope(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	inScopeFile := filepath.Join(dir, "inscope.go")
	if err := os.WriteFile(inScopeFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write inscope.go: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial")

	// Modify the in-scope file.
	if err := os.WriteFile(inScopeFile, []byte("package main\n// modified\n"), 0o644); err != nil {
		t.Fatalf("write modified inscope.go: %v", err)
	}

	outDir := filepath.Join(dir, ".sdp", "evidence")
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &successSandbox{},
		RunID:       "test-scope-ok",
		OutputDir:   outDir,
		RepoPath:    dir,
		ScopeFiles:  []string{"inscope.go"},
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "converged" {
		t.Errorf("Status = %q, want %q", result.Status, "converged")
	}
}

// --- Evidence tests ---

func TestLoop_EvidenceFilesWritten(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &counterSandbox{failCount: 1},
		RunID:       "test-evidence",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "converged" {
		t.Fatalf("Status = %q, want %q", result.Status, "converged")
	}

	// Verify attempt files.
	for i := 1; i <= 2; i++ {
		path := filepath.Join(dir, "attempts", fmt.Sprintf("%d.json", i))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read attempt %d: %v", i, err)
		}
		var ar AttemptResult
		if err := json.Unmarshal(data, &ar); err != nil {
			t.Fatalf("unmarshal attempt %d: %v", i, err)
		}
		if ar.Attempt != i {
			t.Errorf("Attempt = %d, want %d", ar.Attempt, i)
		}
	}

	// Verify loop-result.json.
	loopResultPath := filepath.Join(dir, "loop-result.json")
	data, err := os.ReadFile(loopResultPath)
	if err != nil {
		t.Fatalf("read loop-result.json: %v", err)
	}
	var lr LoopResult
	if err := json.Unmarshal(data, &lr); err != nil {
		t.Fatalf("unmarshal loop-result: %v", err)
	}
	if lr.RunID != "test-evidence" {
		t.Errorf("RunID = %q, want %q", lr.RunID, "test-evidence")
	}
	if lr.Status != "converged" {
		t.Errorf("Status = %q, want %q", lr.Status, "converged")
	}
	if lr.Winner != 2 {
		t.Errorf("Winner = %d, want 2", lr.Winner)
	}
}

func TestLoop_EvidenceNoOutputDir(t *testing.T) {
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &successSandbox{},
		RunID:       "test-no-outdir",
		OutputDir:   "",
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "converged" {
		t.Errorf("Status = %q, want %q", result.Status, "converged")
	}
}

func TestLoop_AttemptDurationRecorded(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &successSandbox{},
		RunID:       "test-duration",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(result.Attempts) == 0 {
		t.Fatal("no attempts recorded")
	}
	if result.Attempts[0].Duration == 0 {
		t.Error("Attempt duration should be non-zero")
	}
}

// --- Error sandbox test ---

func TestLoop_SandboxError(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 1,
		Sandbox:     &errorSandbox{},
		RunID:       "test-sandbox-error",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from sandbox")
	}
	if result.Status != "error" {
		t.Errorf("Status = %q, want %q", result.Status, "error")
	}
	if len(result.Attempts) != 1 {
		t.Fatalf("len(Attempts) = %d, want 1", len(result.Attempts))
	}
	if result.Attempts[0].Error == "" {
		t.Error("Attempt error should not be empty")
	}
}

// --- Default max attempts test ---

func TestDefaultMaxAttempts(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 3,
		Sandbox:     &failSandbox{},
		RunID:       "test-default-max",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	result, err := loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Attempts) != 3 {
		t.Errorf("len(Attempts) = %d, want 3", len(result.Attempts))
	}
}

// --- Atomic write verification ---

func TestLoop_AtomicWriteNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		MaxAttempts: 1,
		Sandbox:     &successSandbox{},
		RunID:       "test-atomic",
		OutputDir:   dir,
	}

	loop, err := NewDefaultLoop(cfg)
	if err != nil {
		t.Fatalf("NewDefaultLoop: %v", err)
	}

	_, err = loop.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if filepath.Ext(info.Name()) == ".tmp" {
			t.Errorf("stale temp file found: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
}

// --- Helpers ---

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@sdp.dev")
	runGit(t, dir, "config", "user.name", "Test")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}
