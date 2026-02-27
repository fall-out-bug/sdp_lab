package orchestrate_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/orchestrate"
)

func TestLoadHookConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := orchestrate.LoadHookConfig(dir)
	if err != nil {
		t.Fatalf("LoadHookConfig: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config when file missing")
	}
}

func TestLoadHookConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: build
    when: post
    executable: echo
    args: ["post-build"]
    on_fail: halt
  - phase: review
    when: pre
    executable: echo
    args: ["pre-review"]
    on_fail: warn
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := orchestrate.LoadHookConfig(dir)
	if err != nil {
		t.Fatalf("LoadHookConfig: %v", err)
	}
	if cfg == nil || len(cfg.Hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %v", cfg)
	}
	if cfg.Hooks[0].Phase != "build" || cfg.Hooks[0].When != "post" || cfg.Hooks[0].OnFail != "halt" {
		t.Errorf("hook 0: %+v", cfg.Hooks[0])
	}
	if cfg.Hooks[1].Phase != "review" || cfg.Hooks[1].When != "pre" || cfg.Hooks[1].OnFail != "warn" {
		t.Errorf("hook 1: %+v", cfg.Hooks[1])
	}
}

func TestRunHooks_PreBuildHalt(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: build
    when: pre
    executable: false
    args: []
    on_fail: halt
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	env := orchestrate.HookEnv{WSID: "00-024-01", FeatureID: "F024", Phase: "build"}
	err := orchestrate.RunHooks(ctx, dir, "build", "pre", env, nil)
	if err == nil {
		t.Error("expected error from halt hook")
	}
}

func TestRunHooks_PostBuildWarn(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: build
    when: post
    executable: false
    args: []
    on_fail: warn
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	env := orchestrate.HookEnv{WSID: "00-024-01", FeatureID: "F024", Phase: "build"}
	err := orchestrate.RunHooks(ctx, dir, "build", "post", env, nil)
	if err != nil {
		t.Errorf("warn should not fail: %v", err)
	}
}

func TestRunHooks_Ignore(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: ci
    when: post
    executable: false
    args: []
    on_fail: ignore
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err := orchestrate.RunHooks(ctx, dir, "ci", "post", orchestrate.HookEnv{}, nil)
	if err != nil {
		t.Errorf("ignore should not fail: %v", err)
	}
}

func TestRunHooks_MissingConfig(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	err := orchestrate.RunHooks(ctx, dir, "build", "pre", orchestrate.HookEnv{}, nil)
	if err != nil {
		t.Errorf("missing config should not fail: %v", err)
	}
}

// --- 00-053-16: executable+args (no sh -c) ---

func TestLoadHookConfig_RejectsLegacyCommand(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: build
    when: pre
    command: "echo legacy"
    on_fail: halt
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := orchestrate.LoadHookConfig(dir)
	if err == nil {
		t.Error("expected error when legacy command field present")
	}
	if !strings.Contains(err.Error(), "command") {
		t.Errorf("error should mention command: %v", err)
	}
}

func TestLoadHookConfig_ExecutableAndArgs(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: build
    when: post
    executable: echo
    args: ["post-build"]
    on_fail: halt
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := orchestrate.LoadHookConfig(dir)
	if err != nil {
		t.Fatalf("LoadHookConfig: %v", err)
	}
	if cfg == nil || len(cfg.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %v", cfg)
	}
	h := cfg.Hooks[0]
	if h.Executable != "echo" || len(h.Args) != 1 || h.Args[0] != "post-build" {
		t.Errorf("hook: executable=%q args=%v", h.Executable, h.Args)
	}
}

func TestRunHooks_ExecutableArgsNoShell(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: build
    when: post
    executable: echo
    args: ["hello", "world"]
    on_fail: halt
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	env := orchestrate.HookEnv{WSID: "00-053-16", FeatureID: "F053", Phase: "build"}
	err := orchestrate.RunHooks(ctx, dir, "build", "post", env, nil)
	if err != nil {
		t.Errorf("RunHooks: %v", err)
	}
}

func TestRunHooks_RejectsShellC(t *testing.T) {
	dir := t.TempDir()
	sdp := filepath.Join(dir, ".sdp")
	if err := os.MkdirAll(sdp, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sdp, "pipeline-hooks.yaml")
	content := `
hooks:
  - phase: build
    when: pre
    executable: sh
    args: ["-c", "echo injected"]
    on_fail: halt
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err := orchestrate.RunHooks(ctx, dir, "build", "pre", orchestrate.HookEnv{}, nil)
	if err == nil {
		t.Error("expected error when sh -c used (shell injection)")
	}
}
