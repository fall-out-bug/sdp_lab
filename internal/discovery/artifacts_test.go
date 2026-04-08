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

func TestArtifacts_WritesHypothesisFile(t *testing.T) {
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
		Hypothesis: &discovery.HypothesisResult{
			WeBelieve:    "developers need automated discovery",
			ToVerify:     "run a landing page test",
			WeMeasure:    "signups in 14 days",
			WeAreRightIf: ">50 signups",
			Assumptions: []discovery.Assumption{
				{Statement: "gap exists", RiskLevel: "high", Uncertainty: "high", RATScore: 9, RATRank: 1},
			},
			Requirements: []string{"CLI entry point", "markdown output"},
			RawIdea:      "test idea",
		},
	}
	if err := discovery.WriteArtifacts(dir, session); err != nil {
		t.Fatalf("write: %v", err)
	}
	hypothesisFile := filepath.Join(dir, "2026-04-08-test-idea-hypothesis.md")
	if _, err := os.Stat(hypothesisFile); err != nil {
		t.Errorf("hypothesis file not created: %v", err)
	}
	content, _ := os.ReadFile(hypothesisFile)
	if !strings.Contains(string(content), "developers need automated discovery") {
		t.Error("hypothesis file missing we_believe")
	}
	if !strings.Contains(string(content), "gap exists") {
		t.Error("hypothesis file missing assumption")
	}
}
