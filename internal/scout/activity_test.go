package scout

import (
	"testing"
)

func TestActivityFromGitRepo(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"main.go": "package main\nfunc main() {}\n",
	}, true)

	addAndCommit(t, dir, "util.go", "package main\nfunc util() {}\n", "2026-04-10T10:00:00Z")

	activity := detectActivity(dir)

	if activity.TotalCommits < 2 {
		t.Errorf("TotalCommits = %d, want >= 2", activity.TotalCommits)
	}
	if activity.Contributors < 1 {
		t.Errorf("Contributors = %d, want >= 1", activity.Contributors)
	}
	if activity.FirstCommit == nil {
		t.Error("FirstCommit should not be nil")
	}
	if activity.LastCommit == nil {
		t.Error("LastCommit should not be nil")
	}
}

func TestActivityEmptyDir(t *testing.T) {
	activity := detectActivity(t.TempDir())

	if activity.TotalCommits != 0 {
		t.Errorf("TotalCommits = %d, want 0 for non-git dir", activity.TotalCommits)
	}
	if activity.FirstCommit != nil {
		t.Error("FirstCommit should be nil for non-git dir")
	}
}

func TestStripCredentials(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no credentials", "https://github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"https with token", "https://ghp_TOKEN@github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"https with user:pass", "https://user:pass123@github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"scp-like with token", "token123@github.com:org/repo.git", "github.com:org/repo.git"},
		{"scp-like with user", "git@github.com:org/repo.git", "github.com:org/repo.git"},
		{"ssh full", "ssh://git@github.com/org/repo.git", "ssh://github.com/org/repo.git"},
		{"no at sign", "https://github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCredentials(tt.input)
			if got != tt.want {
				t.Errorf("stripCredentials(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
