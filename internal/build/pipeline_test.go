package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewDefaultPipeline(t *testing.T) {
	cfg := BuildConfig{
		Idea:     "add auth middleware",
		Sandbox:  "none",
		RepoPath: t.TempDir(),
	}

	p, err := NewDefaultPipeline(cfg)
	if err != nil {
		t.Fatalf("NewDefaultPipeline: %v", err)
	}
	if p.config.RunID == "" {
		t.Error("RunID should be auto-generated")
	}
	if p.config.OutputDir == "" {
		t.Error("OutputDir should be auto-generated")
	}
	if p.sandbox == nil {
		t.Error("sandbox should not be nil")
	}
}

func TestNewDefaultPipeline_InvalidSandbox(t *testing.T) {
	cfg := BuildConfig{
		Idea:    "test",
		Sandbox: "invalid-type",
	}

	_, err := NewDefaultPipeline(cfg)
	if err == nil {
		t.Fatal("expected error for invalid sandbox type")
	}
}

func TestNewDefaultPipeline_Docker(t *testing.T) {
	cfg := BuildConfig{
		Idea:     "test",
		Sandbox:  "docker",
		RepoPath: t.TempDir(),
	}

	p, err := NewDefaultPipeline(cfg)
	if err != nil {
		// Docker not available on this system — that is acceptable.
		if dockerUnavailableError(err) {
			t.Skipf("docker not available: %v", err)
		}
		t.Fatalf("NewDefaultPipeline: %v", err)
	}
	if p == nil {
		t.Fatal("pipeline should not be nil")
	}
}

func TestNewDefaultPipeline_TestcontainersStub(t *testing.T) {
	cfg := BuildConfig{
		Idea:    "test",
		Sandbox: "testcontainers",
	}

	_, err := NewDefaultPipeline(cfg)
	if err == nil {
		t.Fatal("expected error for testcontainers sandbox stub")
	}
}

func TestDryRun(t *testing.T) {
	cfg := BuildConfig{
		Idea:     "refactor auth module",
		Sandbox:  "none",
		RunID:    "test-dry-run",
		RepoPath: t.TempDir(),
	}

	p, err := NewDefaultPipeline(cfg)
	if err != nil {
		t.Fatalf("NewDefaultPipeline: %v", err)
	}

	result, err := p.DryRun()
	if err != nil {
		t.Fatalf("DryRun: %v", err)
	}

	if result.RunID != "test-dry-run" {
		t.Errorf("RunID = %q, want %q", result.RunID, "test-dry-run")
	}
	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
	}
	if len(result.Stages) != 4 {
		t.Fatalf("len(Stages) = %d, want 4", len(result.Stages))
	}
	for _, s := range result.Stages {
		if s.Status != "skipped" {
			t.Errorf("Stage %q status = %q, want %q", s.Stage, s.Status, "skipped")
		}
	}
}

func TestRun_DispatchesAndCollects(t *testing.T) {
	// Create a minimal Go module so go build/test succeed.
	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir)

	cfg := BuildConfig{
		Idea:      "add a simple feature",
		Sandbox:   "none",
		RunID:     "test-run-001",
		OutputDir: filepath.Join(dir, ".sdp", "evidence", "test-run-001"),
		RepoPath:  dir,
	}

	p, err := NewDefaultPipeline(cfg)
	if err != nil {
		t.Fatalf("NewDefaultPipeline: %v", err)
	}

	result, err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Status = %q, want %q", result.Status, "success")
		t.Logf("Summary: %s", result.Summary)
		for _, s := range result.Stages {
			t.Logf("  Stage %q: status=%q error=%q output=%q", s.Stage, s.Status, s.Error, s.Output)
		}
	}
	if len(result.Stages) != 3 {
		t.Errorf("len(Stages) = %d, want 3 (dispatch, sandbox, commit)", len(result.Stages))
	}
	if result.Summary == "" {
		t.Error("Summary should not be empty")
	}

	// Verify evidence was written.
	evidencePath := filepath.Join(cfg.OutputDir, "evidence.json")
	if _, err := os.Stat(evidencePath); os.IsNotExist(err) {
		t.Fatalf("evidence.json not written to %s", evidencePath)
	}

	data, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("read evidence.json: %v", err)
	}

	var ev BuildEvidence
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatalf("unmarshal evidence: %v", err)
	}
	if ev.RunID != "test-run-001" {
		t.Errorf("evidence RunID = %q, want %q", ev.RunID, "test-run-001")
	}
	if ev.Status != "success" {
		t.Errorf("evidence Status = %q, want %q", ev.Status, "success")
	}
}

func TestRun_StrictModeRunsPipeline(t *testing.T) {
	cfg := BuildConfig{
		Idea:     "test",
		Sandbox:  "none",
		Strict:   true,
		RepoPath: t.TempDir(),
	}

	p, err := NewDefaultPipeline(cfg)
	if err != nil {
		t.Fatalf("NewDefaultPipeline: %v", err)
	}

	result, err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("strict mode should no longer error: %v", err)
	}
	if result.Config.Strict != true {
		t.Error("result should reflect strict=true")
	}
}

func TestClassifyIdea(t *testing.T) {
	tests := []struct {
		idea         string
		wantTaskType string
		wantComplex  string
	}{
		{"fix login bug", "bugfix", "medium"},
		{"refactor the database layer", "refactor", "medium"},
		{"research caching strategies", "research", "medium"},
		{"design new API architecture", "architecture", "medium"},
		{"review the security module", "analysis", "medium"},
		{"add simple logging", "feature", "low"},
		{"implement complex multi-step workflow", "feature", "high"},
		{"add user registration", "feature", "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.idea, func(t *testing.T) {
			ic := classifyIdea(tt.idea)
			if ic.TaskType != tt.wantTaskType {
				t.Errorf("TaskType = %q, want %q", ic.TaskType, tt.wantTaskType)
			}
			if ic.Complexity != tt.wantComplex {
				t.Errorf("Complexity = %q, want %q", ic.Complexity, tt.wantComplex)
			}
		})
	}
}

func TestDefaultDispatchDecision(t *testing.T) {
	ic := ideaClassification{TaskType: "feature", Complexity: "medium", RequiredCap: "coding"}

	t.Run("local", func(t *testing.T) {
		cfg := BuildConfig{Local: true}
		dec := defaultDispatchDecision(cfg, ic)
		if dec.Harness != "ollama" {
			t.Errorf("Harness = %q, want %q", dec.Harness, "ollama")
		}
	})

	t.Run("cloud", func(t *testing.T) {
		cfg := BuildConfig{Local: false}
		dec := defaultDispatchDecision(cfg, ic)
		if dec.Harness != "claude-code" {
			t.Errorf("Harness = %q, want %q", dec.Harness, "claude-code")
		}
		if dec.Score <= 0 {
			t.Errorf("Score = %f, want > 0", dec.Score)
		}
	})
}

// --- Test helpers ---

func writeGoMod(t *testing.T, dir string) {
	t.Helper()
	content := "module example.com/test\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func writeGoFile(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := "package pkg\n"
	if err := os.WriteFile(filepath.Join(dir, "pkg", "pkg.go"), []byte(content), 0o644); err != nil {
		t.Fatalf("write pkg.go: %v", err)
	}
}

func TestBuildResult_EndedAtAfterStartedAt(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir)

	cfg := BuildConfig{
		Idea:      "test timing",
		Sandbox:   "none",
		RunID:     "test-timing",
		RepoPath:  dir,
		OutputDir: filepath.Join(dir, ".sdp", "evidence", "test-timing"),
	}

	p, err := NewDefaultPipeline(cfg)
	if err != nil {
		t.Fatalf("NewDefaultPipeline: %v", err)
	}

	result, err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result.EndedAt.Before(result.StartedAt) {
		t.Errorf("EndedAt (%s) before StartedAt (%s)", result.EndedAt, result.StartedAt)
	}
	if result.EndedAt.Sub(result.StartedAt) == 0 {
		t.Error("duration should be non-zero")
	}
}

func TestNoneSandbox_Cleanup(t *testing.T) {
	s := NewNoneSandbox()
	if err := s.Cleanup(); err != nil {
		t.Errorf("Cleanup returned error: %v", err)
	}
}

func TestNoneSandbox_BuildAndTest(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir)

	s := NewNoneSandbox()
	ctx := context.Background()

	buildRes, err := s.Build(ctx, dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !buildRes.Success {
		t.Errorf("Build Success = false, want true; stderr: %s", buildRes.Stderr)
	}

	testRes, err := s.Test(ctx, dir)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !testRes.Success {
		t.Errorf("Test Success = false, want true; stderr: %s", testRes.Stderr)
	}
	if testRes.Duration == 0 {
		t.Error("Test Duration should be non-zero")
	}
}

func TestNoneSandbox_BuildFailure(t *testing.T) {
	dir := t.TempDir()
	// Write an invalid Go file to cause build failure.
	if err := os.MkdirAll(filepath.Join(dir, "bad"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad", "bad.go"), []byte("package bad\nfunc broken(\n"), 0o644); err != nil {
		t.Fatalf("write bad.go: %v", err)
	}
	writeGoMod(t, dir)

	s := NewNoneSandbox()
	ctx := context.Background()

	buildRes, err := s.Build(ctx, dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if buildRes.Success {
		t.Error("Build should fail for invalid Go code")
	}
	if buildRes.ExitCode == 0 {
		t.Error("ExitCode should be non-zero for failed build")
	}
}

func TestNewBuildEvidence(t *testing.T) {
	cfg := BuildConfig{
		RunID: "test-001",
		Idea:  "test idea",
	}
	ev := NewBuildEvidence(cfg)

	if ev.RunID != "test-001" {
		t.Errorf("RunID = %q, want %q", ev.RunID, "test-001")
	}
	if ev.Status != "running" {
		t.Errorf("Status = %q, want %q", ev.Status, "running")
	}
	if ev.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if _, err := time.Parse(time.RFC3339, ev.Timestamp); err != nil {
		t.Errorf("Timestamp not RFC3339: %v", err)
	}
}

func TestBuildEvidence_AddStage(t *testing.T) {
	ev := NewBuildEvidence(BuildConfig{RunID: "test"})
	ev.AddStage(StageEvidence{Name: "dispatch", Status: "success", Duration: "100ms"})
	ev.AddStage(StageEvidence{Name: "sandbox", Status: "success", Duration: "2s"})

	if len(ev.Stages) != 2 {
		t.Fatalf("len(Stages) = %d, want 2", len(ev.Stages))
	}
	if ev.Stages[0].Name != "dispatch" {
		t.Errorf("Stages[0].Name = %q, want %q", ev.Stages[0].Name, "dispatch")
	}
}

func TestBuildEvidence_Write(t *testing.T) {
	dir := t.TempDir()
	ev := NewBuildEvidence(BuildConfig{RunID: "ev-write", Idea: "test"})
	ev.Status = "success"
	ev.AddStage(StageEvidence{Name: "dispatch", Status: "success", Duration: "50ms"})

	if err := ev.Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "evidence.json"))
	if err != nil {
		t.Fatalf("read evidence.json: %v", err)
	}

	var got BuildEvidence
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.RunID != "ev-write" {
		t.Errorf("RunID = %q, want %q", got.RunID, "ev-write")
	}
	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
}

func TestWriteBuildResult(t *testing.T) {
	dir := t.TempDir()
	result := &BuildResult{
		RunID:  "wbr-test",
		Config: BuildConfig{Idea: "write build result test"},
		Stages: []StageResult{
			{
				Stage:    "dispatch",
				Status:   "success",
				Output:   "classified as feature/medium",
				Duration: 10 * time.Millisecond,
				Evidence: map[string]any{
					"decision": dispatchDecision{
						Harness:  "claude-code",
						Provider: "anthropic",
						Model:    "claude-sonnet-4-20250514",
						Score:    0.8,
						Reason:   "auto-classified as feature/medium",
					},
				},
			},
			{
				Stage:    "sandbox",
				Status:   "success",
				Duration: 500 * time.Millisecond,
				Evidence: map[string]any{
					"build_ok":     true,
					"tests_ok":     true,
					"sandbox_type": "none",
				},
			},
			{
				Stage:    "commit",
				Status:   "success",
				Duration: 5 * time.Millisecond,
			},
		},
		Status:    "success",
		Summary:   "all 3 stages passed",
		StartedAt: time.Now().Add(-1 * time.Second).UTC(),
		EndedAt:   time.Now().UTC(),
	}

	if err := WriteBuildResult(result, dir); err != nil {
		t.Fatalf("WriteBuildResult: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "evidence.json"))
	if err != nil {
		t.Fatalf("read evidence.json: %v", err)
	}

	var ev BuildEvidence
	if err := json.Unmarshal(data, &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if ev.Dispatch.Harness != "claude-code" {
		t.Errorf("Dispatch.Harness = %q, want %q", ev.Dispatch.Harness, "claude-code")
	}
	if !ev.Sandbox.BuildOK {
		t.Error("Sandbox.BuildOK should be true")
	}
	if !ev.Sandbox.TestsOK {
		t.Error("Sandbox.TestsOK should be true")
	}
	if ev.Sandbox.Type != "none" {
		t.Errorf("Sandbox.Type = %q, want %q", ev.Sandbox.Type, "none")
	}
	if len(ev.Stages) != 3 {
		t.Errorf("len(Stages) = %d, want 3", len(ev.Stages))
	}
}

func TestNewSandbox(t *testing.T) {
	tests := []struct {
		sandboxType string
		wantErr     bool
		wantType    string
	}{
		{"none", false, "*build.NoneSandbox"},
		{"", false, "*build.NoneSandbox"},
		{"testcontainers", true, ""},
		{"invalid", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.sandboxType, func(t *testing.T) {
			s, err := NewSandbox(tt.sandboxType, false)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := fmt.Sprintf("%T", s); got != tt.wantType {
				t.Errorf("type = %q, want %q", got, tt.wantType)
			}
		})
	}

	// Docker is conditional on availability.
	t.Run("docker", func(t *testing.T) {
		s, err := NewSandbox("docker", false)
		if err != nil {
			// Docker not available — expected on some systems.
			if !dockerUnavailableError(err) {
				t.Errorf("unexpected error: %v", err)
			}
			return
		}
		if got := fmt.Sprintf("%T", s); got != "*build.DockerSandbox" {
			t.Errorf("type = %q, want %q", got, "*build.DockerSandbox")
		}
	})
}

func dockerUnavailableError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "command not found") || strings.Contains(msg, "daemon unavailable")
}

func TestRun_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir)
	writeGoFile(t, dir)

	cfg := BuildConfig{
		Idea:      "test timeout",
		Sandbox:   "none",
		RunID:     "test-cancel",
		OutputDir: filepath.Join(dir, ".sdp", "evidence", "test-cancel"),
		RepoPath:  dir,
	}

	p, err := NewDefaultPipeline(cfg)
	if err != nil {
		t.Fatalf("NewDefaultPipeline: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := p.Run(ctx)
	// Run may return nil error with partial/failed status, or an error.
	if err == nil && result.Status == "success" {
		t.Error("expected non-success status or error from cancelled context")
	}
}

func TestBuildEvidence_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	ev := NewBuildEvidence(BuildConfig{RunID: "atomic-test", Idea: "test"})
	ev.Status = "success"
	ev.AddStage(StageEvidence{Name: "dispatch", Status: "success", Duration: "1ms"})

	if err := ev.Write(dir); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Verify no temp files remain.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("stale temp file found: %s", e.Name())
		}
	}

	// Verify final file exists and is valid.
	data, err := os.ReadFile(filepath.Join(dir, "evidence.json"))
	if err != nil {
		t.Fatalf("read evidence.json: %v", err)
	}
	var got BuildEvidence
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.RunID != "atomic-test" {
		t.Errorf("RunID = %q, want %q", got.RunID, "atomic-test")
	}
}
