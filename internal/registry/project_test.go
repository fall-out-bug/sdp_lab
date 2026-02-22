package registry

import (
	"testing"
)

func TestBeadsPrefixFromRepo(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/org/repo", "repo"},
		{"git@github.com:org/repo.git", "repo"},
		{"https://github.com/org/repo.git", "repo"},
		{"", "."},
	}
	for _, tt := range tests {
		got := BeadsPrefixFromRepo(tt.url)
		if got != tt.want {
			t.Errorf("BeadsPrefixFromRepo(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestProject_EnsureBeadsPrefix(t *testing.T) {
	p := &Project{ID: "p1", RepoURL: "https://github.com/org/my-repo"}
	p.EnsureBeadsPrefix()
	if p.BeadsPrefix != "my-repo" {
		t.Errorf("EnsureBeadsPrefix: got %q", p.BeadsPrefix)
	}
	p.BeadsPrefix = "custom"
	p.EnsureBeadsPrefix()
	if p.BeadsPrefix != "custom" {
		t.Error("EnsureBeadsPrefix should not overwrite existing")
	}
}

func TestRepoURLToSlug(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/org/repo", "org/repo"},
		{"https://github.com/org/repo.git", "org/repo"},
		{"git@github.com:org/repo.git", "org/repo"},
		{"", ""},
		{".", ""},
	}
	for _, tt := range tests {
		got := RepoURLToSlug(tt.url)
		if got != tt.want {
			t.Errorf("RepoURLToSlug(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestProject_RepoSlug(t *testing.T) {
	p := &Project{RepoURL: "https://github.com/fork/opencode", Fork: false}
	if got := p.RepoSlug(); got != "fork/opencode" {
		t.Errorf("RepoSlug (no fork) = %q, want fork/opencode", got)
	}
	p.Fork = true
	p.UpstreamURL = "https://github.com/upstream/opencode"
	if got := p.RepoSlug(); got != "upstream/opencode" {
		t.Errorf("RepoSlug (fork) = %q, want upstream/opencode", got)
	}
}
