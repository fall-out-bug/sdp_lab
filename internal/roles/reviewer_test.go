package roles

import (
	"strings"
	"testing"

	"sdp_dev/internal/beads"
)

func TestParseVerdictFromOutput(t *testing.T) {
	tests := []struct {
		out  string
		want string
	}{
		{"log\n{\"verdict\":\"approve\"}\n", "approve"},
		{"{\"verdict\":\"needs_changes\",\"comments\":[]}", "needs_changes"},
		{"{\"verdict\":\"reject\"}", "reject"},
		{"no json", ""},
		{"{invalid}", ""},
	}
	for _, tt := range tests {
		got := parseVerdictFromOutput(tt.out)
		if got != tt.want {
			t.Errorf("parseVerdictFromOutput(%q) = %q, want %q", tt.out, got, tt.want)
		}
	}
}

func TestExtractComments(t *testing.T) {
	out := `{"verdict":"needs_changes","comments":["fix X","fix Y"]}`
	got := extractComments(out)
	if len(got) != 2 || got[0] != "fix X" || got[1] != "fix Y" {
		t.Errorf("extractComments: %v", got)
	}
	if extractComments("no json") != nil {
		t.Error("extractComments(no json) should return nil")
	}
}

func TestBuildReviewPrompt(t *testing.T) {
	r := &ReviewerStrategy{PersonaID: "security"}
	issue := beads.Issue{ID: "i1", Title: "Fix bug", Description: "Desc", AcceptanceCriteria: "AC"}
	prompt := r.buildReviewPrompt(issue, "evidence text", "security")
	if !strings.Contains(prompt, "security") || !strings.Contains(prompt, "i1") || !strings.Contains(prompt, "Fix bug") {
		t.Errorf("prompt missing fields: %s", prompt)
	}
	if !strings.Contains(prompt, "evidence text") {
		t.Error("prompt should include evidence")
	}
}

func TestBuildReviewPrompt_EmptyEvidence(t *testing.T) {
	r := &ReviewerStrategy{PersonaID: "dx"}
	issue := beads.Issue{ID: "x", Title: "T"}
	prompt := r.buildReviewPrompt(issue, "", "dx")
	if !strings.Contains(prompt, "(no evidence file found)") {
		t.Errorf("empty evidence should show placeholder: %s", prompt)
	}
}
