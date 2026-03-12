package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sdp_dev/internal/realitypro"
)

func TestRun_WritesReportArtifacts(t *testing.T) {
	root := t.TempDir()
	seedRepo(t, root)
	submoduleRoot := filepath.Join(root, "sdp")
	seedProtocolRepo(t, submoduleRoot)

	if _, err := realitypro.Ingest(realitypro.Options{
		ProjectRoot: root,
		Repos:       []string{root, submoduleRoot},
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 10, 10, 0, 0, time.UTC)
		},
	}); err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if _, err := realitypro.Review(realitypro.ReviewOptions{
		ProjectRoot: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 12, 10, 11, 0, 0, time.UTC)
		},
	}); err != nil {
		t.Fatalf("Review failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"--project-root", root}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "backlog=") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	for _, rel := range []string{
		".sdp/reality/c4-system-context.json",
		".sdp/reality/c4-container.json",
		".sdp/reality/c4-component.json",
		".sdp/reality/bootstrap-backlog.json",
		".sdp/reality/agent-readiness-plan.json",
		"docs/reality/c4-system-context.md",
		"docs/reality/c4-containers.md",
		"docs/reality/c4-components.md",
		"docs/reality/intent-gap.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected artifact %s: %v", rel, err)
		}
	}
}

func TestRun_RequiresInputs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"--project-root", t.TempDir()}, &stdout, &stderr); err == nil {
		t.Fatal("expected missing inputs to fail")
	}
}

func seedProtocolRepo(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/protocol\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "prompts", "skills", "reality", "SKILL.md"), "# skill\n")
	writeFile(t, filepath.Join(root, "schema", "reality", "placeholder.json"), "{}\n")
}

func seedRepo(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/app\n\ngo 1.26\n")
	writeFile(t, filepath.Join(root, "cmd", "app", "main.go"), "package main\n\nfunc main() {}\n")
	writeFile(t, filepath.Join(root, "internal", "billing", "service.go"), "package billing\n\nfunc Enabled() bool { return true }\n")
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
