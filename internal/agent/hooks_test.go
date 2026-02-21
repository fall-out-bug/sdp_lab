package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHookRegistry(t *testing.T) {
	r := NewHookRegistry("coder")
	if r == nil {
		t.Fatal("NewHookRegistry returned nil")
	}
}

func TestHookRegistry_RunPreExecute(t *testing.T) {
	r := NewHookRegistry("coder")
	ctx := context.Background()
	data := HookData{IssueID: "1", Role: "coder", WorkDir: t.TempDir()}
	if err := r.RunPreExecute(ctx, data); err != nil {
		t.Errorf("RunPreExecute: %v", err)
	}
}

func TestHookRegistry_RunPostExecute(t *testing.T) {
	r := NewHookRegistry("analyst")
	ctx := context.Background()
	data := HookData{IssueID: "2", Role: "analyst"}
	if err := r.RunPostExecute(ctx, data); err != nil {
		t.Errorf("RunPostExecute: %v", err)
	}
}

func TestHookRegistry_Register(t *testing.T) {
	r := NewHookRegistry("coder")
	ctx := context.Background()
	called := false
	r.Register(HookPreExecute, func(_ context.Context, _ HookData) error {
		called = true
		return nil
	})
	if err := r.RunPreExecute(ctx, HookData{}); err != nil {
		t.Fatalf("RunPreExecute: %v", err)
	}
	if !called {
		t.Error("custom hook was not called")
	}
}

func TestHookRegistry_LoadFromFile_missing(t *testing.T) {
	r := NewHookRegistry("coder")
	if err := r.LoadFromFile(t.TempDir()); err != nil {
		t.Errorf("LoadFromFile (missing): %v", err)
	}
}

func TestHookRegistry_LoadFromFile_exists(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `roles:
  coder:
    pre_execute: [boundary-check]
`
	if err := os.WriteFile(filepath.Join(specsDir, "agent-hooks.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewHookRegistry("coder")
	if err := r.LoadFromFile(dir); err != nil {
		t.Errorf("LoadFromFile: %v", err)
	}
}
