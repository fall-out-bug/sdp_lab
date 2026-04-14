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
