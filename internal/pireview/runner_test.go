package pireview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultSlots(t *testing.T) {
	slots := DefaultSlots()
	if len(slots) != 3 {
		t.Fatalf("expected 3 default slots, got %d", len(slots))
	}
	if slots[0].Slot != "zai" {
		t.Errorf("slot[0] = %q, want %q", slots[0].Slot, "zai")
	}
	if slots[1].Slot != "kimi" {
		t.Errorf("slot[1] = %q, want %q", slots[1].Slot, "kimi")
	}
	if slots[2].Slot != "minimax" {
		t.Errorf("slot[2] = %q, want %q", slots[2].Slot, "minimax")
	}
}

func TestNewRunner_ValidatesConfig(t *testing.T) {
	_, err := NewRunner(Config{}, nil)
	if err == nil {
		t.Fatal("expected error for empty config")
	}
}

func TestNewRunner_DefaultSlots(t *testing.T) {
	runner := &fakeRunner{}
	r, err := NewRunner(Config{
		ProjectRoot: "/tmp/test",
		Scope:       ScopeAuto,
		BaseRef:     "main",
		Runner:      runner,
	}, nil)
	if err != nil {
		t.Fatalf("NewRunner() error: %v", err)
	}
	if len(r.slots) != 3 {
		t.Errorf("expected 3 default slots, got %d", len(r.slots))
	}
}

func TestParseFindingsFromOutput(t *testing.T) {
	output := `Here are the findings:
[{"priority":"P1","title":"missing error check","file":"main.go","start_line":42,"end_line":45,"rationale":"error not checked","suggested_fix":"add if err != nil"}]
End of review.`

	findings := parseFindingsFromOutput(output, "zai")
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Priority != "P1" {
		t.Errorf("Priority = %q, want P1", f.Priority)
	}
	if f.Title != "missing error check" {
		t.Errorf("Title = %q", f.Title)
	}
	if f.File != "main.go" {
		t.Errorf("File = %q", f.File)
	}
	if f.Reviewer != "zai" {
		t.Errorf("Reviewer = %q, want zai", f.Reviewer)
	}
}

func TestParseFindingsFromOutput_NoJSON(t *testing.T) {
	output := "No JSON here"
	findings := parseFindingsFromOutput(output, "kimi")
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(findings))
	}
}

func TestDedupeFindings(t *testing.T) {
	findings := []Finding{
		{Priority: "P1", Title: "issue A", File: "a.go", DedupeKey: "P1:a.go:issue A"},
		{Priority: "P1", Title: "issue A", File: "a.go", DedupeKey: "P1:a.go:issue A"},
		{Priority: "P2", Title: "issue B", File: "b.go", DedupeKey: "P2:b.go:issue B"},
	}
	deduped := dedupeFindings(findings)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 deduped findings, got %d", len(deduped))
	}
}

func TestBuildVerdict_Approved(t *testing.T) {
	findings := []Finding{
		{Priority: "P2", Title: "polish", File: "a.go", DedupeKey: "P2:a.go:polish"},
	}
	verdict := buildVerdict("F161", 1, findings, nil, &ContextPacket{}, 2, 2)
	if verdict.Verdict != "APPROVED" {
		t.Errorf("Verdict = %q, want APPROVED", verdict.Verdict)
	}
	if verdict.P0Count != 0 || verdict.P1Count != 0 {
		t.Errorf("P0=%d P1=%d, want both 0", verdict.P0Count, verdict.P1Count)
	}
}

func TestBuildVerdict_ChangesRequested(t *testing.T) {
	findings := []Finding{
		{Priority: "P1", Title: "bug", File: "a.go", DedupeKey: "P1:a.go:bug"},
		{Priority: "P2", Title: "polish", File: "b.go", DedupeKey: "P2:b.go:polish"},
	}
	verdict := buildVerdict("F161", 1, findings, nil, &ContextPacket{}, 2, 2)
	if verdict.Verdict != "CHANGES_REQUESTED" {
		t.Errorf("Verdict = %q, want CHANGES_REQUESTED", verdict.Verdict)
	}
	if verdict.P1Count != 1 {
		t.Errorf("P1Count = %d, want 1", verdict.P1Count)
	}
}

func TestBuildVerdict_P0Blocks(t *testing.T) {
	findings := []Finding{
		{Priority: "P0", Title: "security", File: "a.go", DedupeKey: "P0:a.go:security"},
	}
	verdict := buildVerdict("F161", 1, findings, nil, &ContextPacket{}, 2, 2)
	if verdict.Verdict != "CHANGES_REQUESTED" {
		t.Errorf("Verdict = %q, want CHANGES_REQUESTED", verdict.Verdict)
	}
	if verdict.P0Count != 1 {
		t.Errorf("P0Count = %d, want 1", verdict.P0Count)
	}
}

func TestBuildVerdict_QuorumFailure_Escalated(t *testing.T) {
	verdict := buildVerdict("F161", 1, nil, nil, &ContextPacket{}, 0, 2)
	if verdict.Verdict != "ESCALATED" {
		t.Errorf("Verdict = %q, want ESCALATED when 0/2 required reviewers succeed", verdict.Verdict)
	}
}

func TestBuildVerdict_PartialQuorum_Escalated(t *testing.T) {
	verdict := buildVerdict("F161", 1, nil, nil, &ContextPacket{}, 1, 2)
	if verdict.Verdict != "ESCALATED" {
		t.Errorf("Verdict = %q, want ESCALATED when 1/2 required reviewers succeed", verdict.Verdict)
	}
}

func TestBuildVerdict_MajorityQuorumApproved(t *testing.T) {
	verdict := buildVerdict("F161", 1, nil, nil, &ContextPacket{}, 2, 3)
	if verdict.Verdict != "APPROVED" {
		t.Errorf("Verdict = %q, want APPROVED when 2/3 required reviewers succeed and no P0/P1 findings", verdict.Verdict)
	}
}

func TestBuildVerdict_SevenRoles(t *testing.T) {
	verdict := buildVerdict("F161", 1, nil, nil, &ContextPacket{}, 2, 2)
	roles := []string{"qa", "security", "devops", "sre", "techlead", "docs", "promptops"}
	for _, role := range roles {
		if _, ok := verdict.Reviewers[role]; !ok {
			t.Errorf("missing reviewer role: %s", role)
		}
	}
}

func TestWriteModelArtifact(t *testing.T) {
	dir := t.TempDir()
	output := `{"findings": []}`
	path := writeModelArtifact(dir, "test-run", "zai", output)

	if path != filepath.Join(dir, ".sdp", "runs", "pi-review", "test-run", "models", "zai.json") {
		t.Errorf("unexpected path: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(data) != output {
		t.Errorf("artifact content = %q, want %q", string(data), output)
	}
}

func TestRunner_Run_WithFakes(t *testing.T) {
	dir := t.TempDir()

	modelOutput := `[{"priority":"P2","title":"minor style","file":"main.go","start_line":10,"rationale":"naming"}]`

	fr := &fakeRunner{
		responses: map[string][]byte{
			"git rev-parse --abbrev-ref HEAD":              []byte("feature/F161\n"),
			"git rev-parse HEAD":                           []byte("abc123\n"),
			"git status --porcelain --untracked-files=all": []byte(" M main.go\n"),
			"git diff HEAD":                                []byte("+new code\n"),
		},
	}

	slots := []ReviewerSlot{
		{Slot: "zai", Provider: "zai", Model: "glm", Role: "reviewer", Required: true},
	}

	cfg := Config{
		ProjectRoot: dir,
		Scope:       ScopeWorkingTree,
		Runner:      fr,
		Feature:     "F161",
	}

	r, err := NewRunner(cfg, slots)
	if err != nil {
		t.Fatalf("NewRunner() error: %v", err)
	}

	// fakeRunner returns nil,nil for unmapped pi keys,
	// so pi "succeeds" with empty output, quorum passes.
	run, verdict, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if run.RunID == "" {
		t.Error("RunID should not be empty")
	}
	if verdict == nil {
		t.Fatal("verdict should not be nil")
	}
	// With empty model output, no findings, quorum passes → APPROVED
	if verdict.Verdict != "APPROVED" {
		t.Errorf("Verdict = %q, want APPROVED", verdict.Verdict)
	}

	// Verify artifacts were written
	if _, err := os.Stat(filepath.Join(dir, ".sdp", "runs", "pi-review", run.RunID, "context.json")); err != nil {
		t.Errorf("context.json not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".sdp", "runs", "pi-review", run.RunID, "test-evidence.json")); err != nil {
		t.Errorf("test-evidence.json not written: %v", err)
	}
	_ = modelOutput
}

type blockingRunner struct{}

func (blockingRunner) Output(context.Context, string, string, ...string) ([]byte, error) {
	return nil, nil
}

func (blockingRunner) Run(context.Context, string, string, ...string) error {
	return nil
}

func (blockingRunner) CombinedOutput(ctx context.Context, _ string, _ string, _ ...string) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestInvokePiHonorsModelTimeout(t *testing.T) {
	r := &Runner{
		cfg: Config{
			ProjectRoot:  t.TempDir(),
			Scope:        ScopeWorkingTree,
			ModelTimeout: 10 * time.Millisecond,
			Runner:       blockingRunner{},
		},
		runner: blockingRunner{},
	}

	start := time.Now()
	_, err := r.invokePi(context.Background(), ReviewerSlot{
		Slot:     "zai",
		Provider: "zai",
		Model:    "glm",
		Role:     "reviewer",
	}, &ContextPacket{}, &TestEvidence{})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("model timeout was not bounded; elapsed=%s", elapsed)
	}
}

func TestInvokePiUsesPiPrintContract(t *testing.T) {
	fr := &fakeRunner{}
	r := &Runner{
		cfg: Config{
			ProjectRoot:  t.TempDir(),
			Scope:        ScopeWorkingTree,
			ModelTimeout: time.Second,
			Runner:       fr,
		},
		runner: fr,
	}

	_, err := r.invokePi(context.Background(), ReviewerSlot{
		Slot:     "kimi",
		Provider: "kimi-coding",
		Model:    "k2p6",
		Role:     "reviewer",
	}, &ContextPacket{}, &TestEvidence{})
	if err != nil {
		t.Fatalf("invokePi: %v", err)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fr.calls))
	}
	call := fr.calls[0]
	if call.name != "pi" {
		t.Fatalf("command = %q, want pi", call.name)
	}
	got := strings.Join(call.args, " ")
	for _, want := range []string{"--provider kimi-coding", "--model k2p6", "--no-tools", "--no-context-files", "--no-session", "-p"} {
		if !strings.Contains(got, want) {
			t.Fatalf("pi args missing %q: %v", want, call.args)
		}
	}
	if gotPromptArg := call.args[len(call.args)-1]; !strings.HasPrefix(gotPromptArg, "@/") {
		t.Fatalf("pi prompt should be passed as @file, got %q", gotPromptArg)
	}
	if len(call.args) > 0 && call.args[0] == "run" {
		t.Fatalf("pi invocation must not use removed run subcommand: %v", call.args)
	}
}

func TestHashString(t *testing.T) {
	h1 := hashString("hello")
	h2 := hashString("hello")
	h3 := hashString("world")

	if h1 != h2 {
		t.Errorf("same input should produce same hash")
	}
	if h1 == h3 {
		t.Errorf("different input should produce different hash")
	}
	if len(h1) != 64 {
		t.Errorf("SHA-256 hex should be 64 chars, got %d", len(h1))
	}
}
