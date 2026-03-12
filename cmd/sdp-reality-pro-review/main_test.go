package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_WritesReviewArtifacts(t *testing.T) {
	root := t.TempDir()
	seedRepoMemory(t, root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"--project-root", root}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "gaps=") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	for _, rel := range []string{
		".sdp/reality/conflicts-report.json",
		".sdp/reality/intent-gap-report.json",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected artifact %s: %v", rel, err)
		}
	}
}

func TestRun_RequiresRepoMemory(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"--project-root", t.TempDir()}, &stdout, &stderr); err == nil {
		t.Fatal("expected missing repo-memory to fail")
	}
}

func seedRepoMemory(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, ".sdp", "reality", "repo-memory.json"), `{
  "spec_version": "v1.0",
  "generated_at": "2026-03-12T10:00:00Z",
  "repos": [
    {
      "repo_id": "repo:sdp",
      "name": "sdp",
      "root_path": "/tmp/sdp",
      "role": "protocol",
      "summary": "protocol repo"
    },
    {
      "repo_id": "repo:sdp_dev",
      "name": "sdp_dev",
      "root_path": "/tmp/sdp_dev",
      "role": "service",
      "summary": "service repo"
    }
  ],
  "module_summaries": [
    {
      "module_id": "module:repo:sdp_dev:internal",
      "repo_id": "repo:sdp_dev",
      "summary": "internal module"
    }
  ],
  "unresolved_questions": [
    "sdp_dev: how does this repo coordinate versioning with the rest of the reposet?"
  ],
  "hotspots": [
    {
      "hotspot_id": "hotspot:repo:sdp_dev:internal:billing:service.go",
      "repo_id": "repo:sdp_dev",
      "path": "internal/billing/service.go",
      "reason": "line_count=950",
      "severity": "high"
    }
  ]
}`)
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
