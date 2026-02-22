package llm

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	issue := IssueInput{
		ID:                 "sdp_dev-4pg",
		Title:              "QA: Test coverage",
		Description:        "Raise coverage to 80%",
		AcceptanceCriteria: "go test ./... passes",
		SpecID:             "qa-001",
	}
	boundary := BoundarySpec{
		AllowedPathPrefixes:   []string{"internal/", "cmd/"},
		ForbiddenPathPrefixes: []string{".git/"},
		ControlPathPrefixes:   []string{".beads/"},
	}
	prompt := BuildPrompt(issue, boundary)
	if !strings.Contains(prompt, "sdp_dev-4pg") {
		t.Error("prompt should contain issue ID")
	}
	if !strings.Contains(prompt, "QA: Test coverage") {
		t.Error("prompt should contain title")
	}
	if !strings.Contains(prompt, "internal/") {
		t.Error("prompt should contain allowed paths")
	}
	if !strings.Contains(prompt, ".git/") {
		t.Error("prompt should contain forbidden paths")
	}
}

func TestBuildPromptShort(t *testing.T) {
	issue := IssueInput{ID: "x", Title: "Fix bug", Description: "The bug is..."}
	got := BuildPromptShort(issue)
	if !strings.Contains(got, "x") || !strings.Contains(got, "Fix bug") {
		t.Errorf("BuildPromptShort: %q", got)
	}
}
