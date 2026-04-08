package discovery_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/discovery"
)

func TestArtifacts_WritesFiles(t *testing.T) {
	dir := t.TempDir()
	session := &discovery.Session{
		Slug: "test-idea",
		Date: "2026-04-08",
		Frame: &discovery.FrameResult{
			RawIdea:          "test idea",
			ProblemStatement: "test problem",
			Jobs:             []string{"job 1"},
			Appetite:         "small",
		},
		Scan: &discovery.ScanResult{
			Items:      []discovery.ScanItem{{Name: "ToolA", Disposition: discovery.DispositionInspire}},
			Whitespace: "gap description",
		},
	}
	if err := discovery.WriteArtifacts(dir, session); err != nil {
		t.Fatalf("write: %v", err)
	}
	frameFile := filepath.Join(dir, "2026-04-08-test-idea-frame.md")
	if _, err := os.Stat(frameFile); err != nil {
		t.Errorf("frame file not created: %v", err)
	}
	content, _ := os.ReadFile(frameFile)
	if !strings.Contains(string(content), "test problem") {
		t.Error("frame file missing problem statement")
	}
}
