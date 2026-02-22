package registry

import (
	"path"
	"regexp"
	"strings"
)

var githubSlugRE = regexp.MustCompile(`(?:https?://github\.com/|git@github\.com:)([^/]+)/([^/]+?)(?:\.git)?$`)

// Project represents a registered SDP project.
type Project struct {
	ID             string            `yaml:"id" json:"id"`
	RepoURL        string            `yaml:"repo_url" json:"repo_url"`
	RepoBranch     string            `yaml:"repo_branch" json:"repo_branch"`
	BeadsPrefix    string            `yaml:"beads_prefix,omitempty" json:"beads_prefix,omitempty"`
	Language       string            `yaml:"language" json:"language"`
	Workstreams    []string          `yaml:"workstreams" json:"workstreams"`
	ModelPolicy    string            `yaml:"model_policy,omitempty" json:"model_policy,omitempty"`
	Config         map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
	Fork           bool   `yaml:"fork,omitempty" json:"fork,omitempty"`
	UpstreamRemote string `yaml:"upstream_remote,omitempty" json:"upstream_remote,omitempty"`
	UpstreamURL    string `yaml:"upstream_url,omitempty" json:"upstream_url,omitempty"`
}

// BeadsPrefixFromRepo derives a default prefix from repo URL.
// e.g. git@github.com:org/repo.git -> repo, https://github.com/org/repo -> repo
func BeadsPrefixFromRepo(repoURL string) string {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	base := path.Base(repoURL)
	if idx := strings.Index(base, ":"); idx >= 0 {
		base = base[idx+1:]
	}
	return strings.ReplaceAll(base, " ", "-")
}

// EnsureBeadsPrefix sets BeadsPrefix from RepoURL if empty.
func (p *Project) EnsureBeadsPrefix() {
	if p.BeadsPrefix == "" && p.RepoURL != "" {
		p.BeadsPrefix = BeadsPrefixFromRepo(p.RepoURL)
	}
}

// RepoSlug returns owner/repo for GitHub URLs, or empty if not a GitHub URL.
// Fork projects: use UpstreamURL for PR target when Fork is true.
func (p *Project) RepoSlug() string {
	url := p.RepoURL
	if p.Fork && strings.TrimSpace(p.UpstreamURL) != "" {
		url = p.UpstreamURL
	}
	return RepoURLToSlug(url)
}

// RepoURLToSlug converts a repo URL to owner/repo (e.g. for gh pr create --repo).
func RepoURLToSlug(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" || repoURL == "." {
		return ""
	}
	m := githubSlugRE.FindStringSubmatch(repoURL)
	if len(m) >= 3 {
		return m[1] + "/" + strings.TrimSuffix(m[2], ".git")
	}
	return ""
}
