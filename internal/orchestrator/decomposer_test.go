package orchestrator

import (
	"strings"
	"testing"

	"sdp_dev/internal/beads"
)

func TestBuildDecomposePrompt(t *testing.T) {
	feature := beads.Issue{
		ID:          "feat-1",
		Title:       "Add feature",
		Description: "Do something",
		AcceptanceCriteria: "AC: done",
	}
	prompt := buildDecomposePrompt(feature, "/tmp/out.json")
	if !strings.Contains(prompt, "feat-1") || !strings.Contains(prompt, "Add feature") {
		t.Errorf("prompt missing feature info: %s", prompt)
	}
	if !strings.Contains(prompt, "Do something") || !strings.Contains(prompt, "AC: done") {
		t.Errorf("prompt missing description/acceptance")
	}
	if !strings.Contains(prompt, "/tmp/out.json") {
		t.Errorf("prompt missing output path")
	}
	if !strings.Contains(prompt, "depends_on_index") {
		t.Errorf("prompt missing depends_on_index")
	}
}

func TestBuildDecomposePrompt_EmptyOptional(t *testing.T) {
	feature := beads.Issue{ID: "f1", Title: "T"}
	prompt := buildDecomposePrompt(feature, "/x.json")
	if !strings.Contains(prompt, "f1") || !strings.Contains(prompt, "T") {
		t.Errorf("minimal prompt: %s", prompt)
	}
}
