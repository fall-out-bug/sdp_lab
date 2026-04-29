package pireview

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTestCommand_Explicit(t *testing.T) {
	cfg := Config{TestCommand: "go test -v ./pkg/..."}
	cmd, err := resolveTestCommand(cfg)
	if err != nil {
		t.Fatalf("resolveTestCommand() error: %v", err)
	}
	if len(cmd) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(cmd), cmd)
	}
	if cmd[0] != "go" || cmd[1] != "test" {
		t.Errorf("expected go test, got %v", cmd)
	}
}

func TestResolveTestCommand_GoProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{ProjectRoot: dir}
	cmd, err := resolveTestCommand(cfg)
	if err != nil {
		t.Fatalf("resolveTestCommand() error: %v", err)
	}
	if cmd[0] != "go" || cmd[1] != "test" {
		t.Errorf("expected go test, got %v", cmd)
	}
}

func TestResolveTestCommand_NpmProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{ProjectRoot: dir}
	cmd, err := resolveTestCommand(cfg)
	if err != nil {
		t.Fatalf("resolveTestCommand() error: %v", err)
	}
	if cmd[0] != "npm" || cmd[1] != "test" {
		t.Errorf("expected npm test, got %v", cmd)
	}
}

func TestResolveTestCommand_NoProject(t *testing.T) {
	cfg := Config{ProjectRoot: t.TempDir()}
	_, err := resolveTestCommand(cfg)
	if err == nil {
		t.Fatal("expected error for unknown project type")
	}
}

func TestCollectTestEvidence_Skipped(t *testing.T) {
	cfg := Config{ProjectRoot: t.TempDir()}
	evidence, err := CollectTestEvidence(context.Background(), cfg)
	if err != nil {
		t.Fatalf("CollectTestEvidence() error: %v", err)
	}
	if evidence.Status != "skipped" {
		t.Errorf("Status = %q, want %q", evidence.Status, "skipped")
	}
	if evidence.SkipReason == "" {
		t.Error("SkipReason should not be empty")
	}
}

func TestCollectTestEvidence_ExplicitCommand(t *testing.T) {
	dir := t.TempDir()
	artifactDir := filepath.Join(dir, ".sdp", "runs", "pi-review")

	cfg := Config{
		ProjectRoot: dir,
		TestCommand: "echo hello",
	}

	evidence, err := CollectTestEvidence(context.Background(), cfg)
	if err != nil {
		t.Fatalf("CollectTestEvidence() error: %v", err)
	}
	if evidence.Status != "passed" {
		t.Errorf("Status = %q, want %q", evidence.Status, "passed")
	}
	if evidence.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", evidence.ExitCode)
	}
	if evidence.Command != "echo hello" {
		t.Errorf("Command = %q, want %q", evidence.Command, "echo hello")
	}

	expectedPath := filepath.Join(artifactDir, "test-output.txt")
	if evidence.ArtifactPath != expectedPath {
		t.Errorf("ArtifactPath = %q, want %q", evidence.ArtifactPath, expectedPath)
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Errorf("artifact content should contain 'hello', got %q", string(data))
	}
}
