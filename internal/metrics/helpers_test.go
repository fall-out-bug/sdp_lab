package metrics

import (
	"testing"
)

// ── F121-6l0: isFixCommit helper ───────────────────────────────────

func TestIsFixCommit(t *testing.T) {
	tests := []struct {
		subject string
		want    bool
	}{
		{"fix: urgent bug", true},
		{"Fix: uppercase", true},
		{"bugfix: null pointer", true},
		{"BugFix: mixed case", true},
		{"patch: security update", true},
		{"hotfix: production issue", true},
		{"feat: new feature", false},
		{"chore: cleanup", false},
		{"refactor: extract helper", false},
		{"document the fix process", true}, // contains "fix"
		{"fixing the broken test", true},   // contains "fix"
		{"affix the label", true},          // contains "fix" — acceptable false positive
	}

	for _, tt := range tests {
		got := isFixCommit(tt.subject)
		if got != tt.want {
			t.Errorf("isFixCommit(%q) = %v, want %v", tt.subject, got, tt.want)
		}
	}
}
