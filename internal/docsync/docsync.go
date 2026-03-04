package docsync

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"sdp_dev/internal/workstream"
)

type Issue struct {
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
}

type ConsistencyReport struct {
	Issues []Issue `json:"issues"`
}

func (r ConsistencyReport) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == "error" {
			return true
		}
	}
	return false
}

func CheckConsistency(projectRoot string, strict bool) (ConsistencyReport, error) {
	report := ConsistencyReport{Issues: []Issue{}}

	protocol, err := workstream.ValidateProtocol(projectRoot, false, strict)
	if err != nil {
		return report, err
	}
	for _, p := range protocol.Issues {
		report.Issues = append(report.Issues, Issue{Severity: p.Severity, File: p.File, Message: p.Message})
	}

	linkIssues, err := checkMarkdownLinks(projectRoot, strict)
	if err != nil {
		return report, err
	}
	report.Issues = append(report.Issues, linkIssues...)

	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Severity == report.Issues[j].Severity {
			if report.Issues[i].File == report.Issues[j].File {
				return report.Issues[i].Message < report.Issues[j].Message
			}
			return report.Issues[i].File < report.Issues[j].File
		}
		return report.Issues[i].Severity < report.Issues[j].Severity
	})

	return report, nil
}

func UpdateChangelog(projectRoot, sinceRange string) (string, error) {
	if sinceRange == "" {
		sinceRange = "HEAD~1..HEAD"
	}

	commitLines, err := gitOutput(projectRoot, "log", sinceRange, "--pretty=format:%h\t%s\t%ad", "--date=short")
	if err != nil {
		return "", fmt.Errorf("git log: %w", err)
	}
	if strings.TrimSpace(commitLines) == "" {
		return "", nil
	}

	changedFiles, err := gitOutput(projectRoot, "diff", "--name-only", sinceRange)
	if err != nil {
		return "", fmt.Errorf("git diff --name-only: %w", err)
	}

	changelogPath := filepath.Join(projectRoot, "docs", "CHANGELOG.md")
	if err := os.MkdirAll(filepath.Dir(changelogPath), 0o755); err != nil {
		return "", err
	}

	existing := "# Changelog\n\n"
	if b, err := os.ReadFile(changelogPath); err == nil {
		existing = string(b)
	}

	date := time.Now().Format("2006-01-02")
	entry := &strings.Builder{}
	fmt.Fprintf(entry, "## %s\n\n", date)
	fmt.Fprintln(entry, "### Commits")
	for _, ln := range splitNonEmpty(commitLines) {
		parts := strings.SplitN(ln, "\t", 3)
		if len(parts) == 3 {
			fmt.Fprintf(entry, "- `%s` %s (%s)\n", parts[0], parts[1], parts[2])
		} else {
			fmt.Fprintf(entry, "- %s\n", ln)
		}
	}
	fmt.Fprintln(entry)
	fmt.Fprintln(entry, "### Changed Files")
	for _, f := range splitNonEmpty(changedFiles) {
		fmt.Fprintf(entry, "- `%s`\n", f)
	}
	fmt.Fprintln(entry)

	newContent := existing
	if strings.Contains(existing, "## "+date+"\n") {
		newContent = strings.Replace(existing, "## "+date+"\n", entry.String()+"\n", 1)
	} else {
		newContent = strings.TrimRight(existing, "\n") + "\n\n" + entry.String()
	}

	if err := os.WriteFile(changelogPath, []byte(newContent), 0o644); err != nil {
		return "", err
	}
	return changelogPath, nil
}

func checkMarkdownLinks(projectRoot string, strict bool) ([]Issue, error) {
	issues := []Issue{}
	docsRoot := filepath.Join(projectRoot, "docs")

	var files []string
	err := filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	for _, path := range files {
		relPath := rel(projectRoot, path)
		if skipLinkCheck(relPath) {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, Issue{Severity: "warning", File: relPath, Message: fmt.Sprintf("read file: %v", err)})
			continue
		}
		matches := re.FindAllStringSubmatch(string(b), -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			target := strings.TrimSpace(m[1])
			if target == "" || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") {
				continue
			}
			if i := strings.Index(target, "#"); i >= 0 {
				target = target[:i]
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), target))
			if _, err := os.Stat(resolved); err != nil {
				sev := "warning"
				if strict {
					sev = "error"
				}
				issues = append(issues, Issue{Severity: sev, File: relPath, Message: fmt.Sprintf("broken local link: %s", target)})
			}
		}
	}

	return issues, nil
}

func gitOutput(projectRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return string(out), nil
}

func splitNonEmpty(s string) []string {
	result := []string{}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func rel(projectRoot, path string) string {
	if p, err := filepath.Rel(projectRoot, path); err == nil {
		return p
	}
	return path
}

func skipLinkCheck(relPath string) bool {
	legacyPrefixes := []string{
		"docs/reference/",
		"docs/decisions/",
		"docs/design/",
		"docs/attestation/",
		"docs/beads-integration/",
		"docs/integrations/",
		"docs/specs/",
		"docs/vision/",
	}
	for _, p := range legacyPrefixes {
		if strings.HasPrefix(relPath, p) {
			return true
		}
	}
	if relPath == "docs/INCIDENT_RESPONSE.md" {
		return true
	}
	if strings.HasPrefix(relPath, "docs/workstreams/backlog/") {
		base := filepath.Base(relPath)
		var prefix, feature, seq int
		if _, err := fmt.Sscanf(strings.TrimSuffix(base, filepath.Ext(base)), "%d-%d-%d", &prefix, &feature, &seq); err == nil {
			if feature < 59 {
				return true
			}
		}
	}
	if relPath == "docs/plans/2026-02-25-beads-remediation-plan.md" {
		return true
	}
	return false
}
