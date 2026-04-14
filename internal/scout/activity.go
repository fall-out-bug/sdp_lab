package scout

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// detectActivity runs Phase 3: git history analysis.
// Returns zero-value Activity for non-git directories.
func detectActivity(root string) Activity {
	var a Activity
	if !isGitRepo(root) {
		return a
	}

	logData := gitCmd(root, "log", "--format=%aI|%aN", "--no-merges", "--since=2 years ago")
	if logData == "" {
		logData = gitCmd(root, "log", "--format=%aI|%aN", "--no-merges")
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

	branches := gitCmd(root, "branch", "-r", "--no-merged", "main")
	a.ActiveBranches = countNonEmptyLines(branches)
	return a
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
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		commits = append(commits, commitInfo{date: t, author: strings.TrimSpace(parts[1])})
	}
	return commits
}

func isGitRepo(dir string) bool {
	_, err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").CombinedOutput()
	return err == nil
}

func gitCmd(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
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

// detectMaturityFromGit extends Maturity with tag/release signals.
func detectMaturityFromGit(root string, mat *Maturity) {
	if !isGitRepo(root) {
		return
	}
	tags := gitCmd(root, "tag", "--sort=-creatordate")
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
