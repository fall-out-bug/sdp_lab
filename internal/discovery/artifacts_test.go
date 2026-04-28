package discovery_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestArtifacts_WritesExperimentFile(t *testing.T) {
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
		Experiment: &discovery.ExperimentBrief{
			Format:        discovery.ExperimentCustomerInterview,
			Objective:     "validate that founders trust LLM-generated insights",
			Hypothesis:    "if we interview 10 founders, 7 will rate LLM insights as trustworthy",
			SuccessMetric: "7/10 founders rate insights as trustworthy within 7 days",
			TimeBoxDays:   7,
			SetupSteps:    []string{"write interview script", "recruit 10 participants", "run interviews"},
			RequiredTools: []string{"Calendly", "Zoom"},
			RawClaim:      "LLM-generated validation is trusted by founders",
			CostUSD:       0.00080,
		},
	}
	if err := discovery.WriteArtifacts(dir, session); err != nil {
		t.Fatalf("write: %v", err)
	}
	expFile := filepath.Join(dir, "2026-04-08-test-idea-experiment.md")
	if _, err := os.Stat(expFile); err != nil {
		t.Errorf("experiment file not created: %v", err)
	}
	content, _ := os.ReadFile(expFile)
	s := string(content)
	for _, want := range []string{"customer_interview", "validate that founders", "7 days", "Calendly"} {
		if !strings.Contains(s, want) {
			t.Errorf("experiment file missing %q", want)
		}
	}
}

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

func TestArtifacts_WritesValidationFile(t *testing.T) {
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
		Validation: &discovery.ValidationResult{
			FinalVerdict:  discovery.VerdictGO,
			VerdictReason: "evidence supports both core assumptions",
			Claims: []discovery.ClaimValidation{
				{
					Claim:   "founders lack time for discovery",
					RATRank: 1,
					Verdict: discovery.VerdictSupported,
					Evidence: []discovery.Evidence{
						{Direction: "for", Statement: "62% of indie hackers skip validation", IsEstimate: true},
						{Direction: "against", Statement: "some use customer interviews", IsEstimate: true},
					},
					Confidence: 0.8,
					Notes:      "strong signal from survey data",
				},
			},
			NeedsExperiment: false,
			CostUSD:         0.00123,
		},
	}
	if err := discovery.WriteArtifacts(dir, session); err != nil {
		t.Fatalf("write: %v", err)
	}
	valFile := filepath.Join(dir, "2026-04-08-test-idea-validation.md")
	if _, err := os.Stat(valFile); err != nil {
		t.Errorf("validation file not created: %v", err)
	}
	content, _ := os.ReadFile(valFile)
	s := string(content)
	for _, want := range []string{"GO", "founders lack time", "supported", "evidence supports"} {
		if !strings.Contains(s, want) {
			t.Errorf("validation file missing %q", want)
		}
	}
}
