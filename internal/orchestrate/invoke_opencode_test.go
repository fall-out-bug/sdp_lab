package orchestrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/kernel"
)

func TestRunBuildPhase_WithFakeInvoker(t *testing.T) {
	dir := t.TempDir()
	// Minimal project layout for RunBuildPhase
	sdpDir := filepath.Join(dir, ".sdp", "checkpoints")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cpPath := filepath.Join(sdpDir, "F053.json")
	if err := os.WriteFile(cpPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	wsDir := filepath.Join(dir, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, "00-053-34.md"), []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}

	fake := &fakeLLMInvoker{
		output:   "abc123def4567890123456789012345678901234",
		exitCode: 0,
	}
	commit, err := RunBuildPhase(context.Background(), dir, "F053", "00-053-34", fake)
	if err != nil {
		t.Fatalf("RunBuildPhase: %v", err)
	}
	if commit != "abc123def4567890123456789012345678901234" {
		t.Errorf("commit = %q, want %q", commit, "abc123def4567890123456789012345678901234")
	}
	if !fake.invoked {
		t.Error("fake invoker was not invoked")
	}
}

func TestRunBuildPhase_WithFakeInvoker_NonZeroExit(t *testing.T) {
	dir := t.TempDir()
	sdpDir := filepath.Join(dir, ".sdp", "checkpoints")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(sdpDir, "F053.json"), []byte("{}"), 0o644)
	wsDir := filepath.Join(dir, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(wsDir, "00-053-34.md"), []byte("# test"), 0o644)

	fake := &fakeLLMInvoker{output: "build failed", exitCode: 1}
	_, err := RunBuildPhase(context.Background(), dir, "F053", "00-053-34", fake)
	if err == nil {
		t.Fatal("expected error from non-zero exit")
	}
}

func TestRunReviewPhaseDetailedParsesExplicitVerdicts(t *testing.T) {
	cases := []struct {
		name         string
		output       string
		exitCode     int
		wantApproved bool
		wantVerdict  string
	}{
		{
			name:         "explicit approved line",
			output:       "review complete\nVERDICT: APPROVED\n",
			exitCode:     0,
			wantApproved: true,
			wantVerdict:  "APPROVED",
		},
		{
			name:         "negative approved phrase is not approval",
			output:       "Review is not approved because blockers remain.\nVERDICT: CHANGES_REQUESTED\n",
			exitCode:     0,
			wantApproved: false,
			wantVerdict:  "CHANGES_REQUESTED",
		},
		{
			name:         "json partial verdict",
			output:       `{"feature":"F104","verdict":"PARTIALLY_APPROVED","partial_failing_roles":["docs"]}`,
			exitCode:     0,
			wantApproved: false,
			wantVerdict:  "PARTIALLY_APPROVED",
		},
		{
			name:         "json escalated verdict",
			output:       `{"feature":"F104","verdict":"ESCALATED","escalation_issue":"sdplab-123"}`,
			exitCode:     0,
			wantApproved: false,
			wantVerdict:  "ESCALATED",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := RunReviewPhaseDetailed(context.Background(), t.TempDir(), "F104", &fakeLLMInvoker{
				output:   tt.output,
				exitCode: tt.exitCode,
			})
			if err != nil {
				t.Fatalf("RunReviewPhaseDetailed: %v", err)
			}
			if res.Approved != tt.wantApproved {
				t.Fatalf("Approved = %v, want %v", res.Approved, tt.wantApproved)
			}
			if res.Verdict != tt.wantVerdict {
				t.Fatalf("Verdict = %q, want %q", res.Verdict, tt.wantVerdict)
			}
		})
	}
}

type fakeLLMInvoker struct {
	output   string
	exitCode int
	invoked  bool
}

func (f *fakeLLMInvoker) Invoke(ctx context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	f.invoked = true
	return kernel.RuntimeResult{Output: f.output, ExitCode: f.exitCode}, nil
}

func TestComputePromptHash(t *testing.T) {
	// Empty string has known SHA-256
	got := ComputePromptHash("")
	if len(got) != 64 {
		t.Errorf("hash length = %d, want 64", len(got))
	}
	// Deterministic
	if got != ComputePromptHash("") {
		t.Error("hash should be deterministic")
	}
}

func TestBuildContextSources(t *testing.T) {
	dir := t.TempDir()
	// Create minimal files
	wsDir := filepath.Join(dir, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wsPath := filepath.Join(wsDir, "00-026-01.md")
	if err := os.WriteFile(wsPath, []byte("# test"), 0o644); err != nil {
		t.Fatal(err)
	}
	sdpDir := filepath.Join(dir, ".sdp", "checkpoints")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cpPath := filepath.Join(sdpDir, "F026.json")
	if err := os.WriteFile(cpPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	sources := BuildContextSources(dir, "F026", "00-026-01", nil)
	if len(sources) == 0 {
		t.Error("expected at least workstream_spec and checkpoint")
	}
	for _, s := range sources {
		if s.Type == "" || s.Path == "" || s.Hash == "" {
			t.Errorf("invalid source: %+v", s)
		}
		if len(s.Hash) != 64 {
			t.Errorf("hash length = %d for %s", len(s.Hash), s.Type)
		}
	}
}

func TestWritePromptProvenance(t *testing.T) {
	dir := t.TempDir()
	sources := []ContextSource{
		{Type: "workstream_spec", Path: "docs/ws.md", Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	}
	if err := WritePromptProvenance(dir, "abc123", sources); err != nil {
		t.Fatalf("WritePromptProvenance: %v", err)
	}
	path := filepath.Join(dir, ".sdp", "prompt-provenance.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(b) == 0 {
		t.Error("expected non-empty file")
	}
}
