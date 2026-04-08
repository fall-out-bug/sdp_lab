package main

import (
	"strings"
	"testing"

	"sdp_dev/internal/discovery"
)

func TestBuildFeatureDescription_GOIncludesRequirements(t *testing.T) {
	frame := &discovery.FrameResult{
		ProblemStatement: "developers waste time on manual discovery",
		Appetite:         "medium",
	}
	hyp := &discovery.HypothesisResult{
		WeBelieve: "solo founders need faster validation",
		Requirements: []string{
			"user can upload transcript and receive summary",
			"CLI produces markdown artifact in under 60s",
		},
	}
	val := &discovery.ValidationResult{
		FinalVerdict:  discovery.VerdictGO,
		VerdictReason: "both core assumptions supported",
	}
	desc := buildDiscoveryDescription("automate product discovery", frame, hyp, val, "/tmp/out")

	if !strings.Contains(desc, "GO") {
		t.Error("description missing GO verdict")
	}
	if !strings.Contains(desc, "user can upload transcript") {
		t.Error("description missing requirements")
	}
	if !strings.Contains(desc, "CLI produces markdown artifact") {
		t.Error("description missing second requirement")
	}
}

func TestBuildFeatureDescription_PIVOTOmitsRequirements(t *testing.T) {
	frame := &discovery.FrameResult{
		ProblemStatement: "developers waste time",
		Appetite:         "small",
	}
	hyp := &discovery.HypothesisResult{
		WeBelieve:    "founders need help",
		Requirements: []string{"some requirement"},
	}
	val := &discovery.ValidationResult{
		FinalVerdict:    discovery.VerdictPIVOT,
		VerdictReason:   "evidence mixed",
		PivotSuggestion: "narrow to research repo",
	}
	desc := buildDiscoveryDescription("some idea", frame, hyp, val, "/tmp/out")

	if !strings.Contains(desc, "PIVOT") {
		t.Error("description missing PIVOT verdict")
	}
	if !strings.Contains(desc, "narrow to research repo") {
		t.Error("description missing pivot suggestion")
	}
	if strings.Contains(desc, "some requirement") {
		t.Error("PIVOT description must not include requirements")
	}
	if strings.Contains(desc, "## Requirements") {
		t.Error("PIVOT description must not include Requirements section header")
	}
}
