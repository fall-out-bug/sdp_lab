package roles

import (
	"testing"
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
