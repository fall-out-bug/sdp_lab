package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrPublish_dryRun_withProjectAndBase(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := wd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(root, "specs", "project-registry.yaml")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Skip("specs/project-registry.yaml not found")
		}
		root = parent
	}

	cmd := exec.Command("go", "run", "./cmd/pr-publish", "--issue", "sdp_dev-5l9", "--title", "Test", "--head", "worker/test", "--project", "sdp_dev", "--base", "master", "--dry-run")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pr-publish --dry-run: %v: %s", err, string(out))
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("parse output: %v", err)
	}
	cmdLine, _ := result["command"].([]any)
	if cmdLine == nil {
		t.Fatal("expected command in output")
	}
	cmdStr := ""
	for _, c := range cmdLine {
		if s, ok := c.(string); ok {
			cmdStr += " " + s
		}
	}
	if !strings.Contains(cmdStr, "pr create") {
		t.Error("expected gh pr create in command")
	}
	if !strings.Contains(cmdStr, "--base") {
		t.Error("expected --base in command")
	}
}
