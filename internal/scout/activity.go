package scout

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const defaultGitTimeout = 30 * time.Second

// detectActivity runs Phase 3: git history analysis.
// Returns zero-value Activity for non-git directories.
func detectActivity(root string) Activity {
	return detectActivityWithContext(context.Background(), root)
}

func detectActivityWithContext(ctx context.Context, root string) Activity {
	var a Activity
	if !isGitRepo(root) {
		return a
	}

	logData := gitCmdWithContext(ctx, root, "log", "--format=%aI|%aN", "--no-merges", "--since=2 years ago")
	if logData == "" {
		logData = gitCmdWithContext(ctx, root, "log", "--format=%aI|%aN", "--no-merges")
	}

	commits := parseCommitLog(logData)
	a.TotalCommits = len(commits)
	if a.TotalCommits == 0 {
		return a
	}

	first := commits[len(commits)-1].date
	last := commits[0].date
	fs := first.Format("2006-01-02")
	ls := last.Format("2006-01-02")
	a.FirstCommit = &fs
	a.LastCommit = &ls
	a.AgeMonths = int(last.Sub(first).Hours() / (24 * 30.44))

	now := time.Now()
	cutoff90 := now.AddDate(0, -3, 0)
	cutoff30 := now.AddDate(0, 0, -30)
	authors := make(map[string]int)
	active90 := make(map[string]bool)
	for _, c := range commits {
		authors[c.author]++
		if c.date.After(cutoff90) {
			active90[c.author] = true
			a.Commits90d++
		}
		if c.date.After(cutoff30) {
			a.Commits30d++
		}
	}
	a.Contributors = len(authors)
	a.ActiveContributors90d = len(active90)

	// B2 fix: branch detection with fallback (main → master → HEAD)
	defaultBranch := detectDefaultBranch(root)
	branches := gitCmdWithContext(ctx, root, "branch", "-r", "--no-merged", defaultBranch)
	a.ActiveBranches = countNonEmptyLines(branches)
	return a
}

// detectDefaultBranch tries main, master, then falls back to HEAD.
func detectDefaultBranch(dir string) string {
	for _, branch := range []string{"main", "master"} {
		result := gitCmdWithContext(context.Background(), dir, "rev-parse", "--verify", branch)
		if result != "" {
			return branch
		}
	}
	return "HEAD"
}

type commitInfo struct {
	date   time.Time
	author string
}

func parseCommitLog(data string) []commitInfo {
	var commits []commitInfo
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		before, after, ok := strings.Cut(line, "|")
		if !ok {
			continue
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(before))
		if err != nil {
			continue
		}
		commits = append(commits, commitInfo{date: t, author: strings.TrimSpace(after)})
	}
	return commits
}

func isGitRepo(dir string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultGitTimeout)
	defer cancel()
	_, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--git-dir").CombinedOutput()
	return err == nil
}

// gitCmdWithContext runs a git command with context-based cancellation.
func gitCmdWithContext(ctx context.Context, dir string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, defaultGitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}

// detectMaturityFromGitWithContext extends Maturity with tag/release signals.
func detectMaturityFromGitWithContext(ctx context.Context, root string, mat *Maturity) {
	if !isGitRepo(root) {
		return
	}
	tags := gitCmdWithContext(ctx, root, "tag", "--sort=-creatordate")
	var releases []string
	for _, tag := range strings.Split(strings.TrimSpace(tags), "\n") {
		tag = strings.TrimSpace(tag)
		if len(tag) > 1 && tag[0] == 'v' && tag[1] >= '0' && tag[1] <= '9' {
			releases = append(releases, tag)
		}
	}
	mat.ReleaseCount = len(releases)
	mat.HasReleases = len(releases) > 0
	if len(releases) > 0 {
		v := releases[0]
		mat.LatestRelease = &v
	}
}

// detectRepoURL returns the origin remote URL with credentials stripped, or nil if none.
func detectRepoURL(root string) *string {
	raw := gitCmdWithContext(context.Background(), root, "remote", "get-url", "origin")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	clean := stripCredentials(raw)
	return &clean
}

// stripCredentials removes embedded userinfo (user:pass@) from a git remote URL.
// Handles both HTTPS (https://user:pass@host/path) and scp-like (user@host:path) formats.
func stripCredentials(rawURL string) string {
	atIdx := strings.LastIndex(rawURL, "@")
	if atIdx < 0 {
		return rawURL
	}

	// HTTPS format: scheme://userinfo@host/path
	if schemeIdx := strings.Index(rawURL, "://"); schemeIdx >= 0 {
		hostStart := atIdx + 1
		scheme := rawURL[:schemeIdx+3]
		rest := rawURL[hostStart:]
		return scheme + rest
	}

	// scp-like format: userinfo@host:path.git
	// Pattern: something@host:path — strip everything before @
	hostStart := atIdx + 1
	if hostStart < len(rawURL) {
		return rawURL[hostStart:]
	}

	return rawURL
}

// detectBuildEntries finds entry points (files with func main()).
func detectBuildEntries(root string, _ *string) []string {
	var entries []string
	seen := make(map[string]bool)
	for _, pat := range []string{
		filepath.Join(root, "cmd", "*", "main.go"),
		filepath.Join(root, "cmd", "main.go"),
		filepath.Join(root, "main.go"),
	} {
		matches, _ := filepath.Glob(pat)
		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil || !strings.Contains(string(data), "func main()") {
				continue
			}
			rel, _ := filepath.Rel(root, m)
			if !seen[rel] {
				seen[rel] = true
				entries = append(entries, rel)
			}
		}
	}
	return entries
}
