// Package bridge provides synchronization between GitHub CI findings and Beads tasks.
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ProtocolFindings represents findings from sdp-protocol-check.
type ProtocolFindings struct {
	SpecVersion   string                 `json:"spec_version"`
	FindingsID    string                 `json:"findings_id"`
	Timestamp     string                 `json:"timestamp"`
	Source        FindingsSource         `json:"source"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Findings      []ProtocolFinding      `json:"findings"`
	Summary       FindingsSummary        `json:"summary"`
}

// DocsFindings represents findings from sdp-doc-sync.
type DocsFindings struct {
	SpecVersion   string                 `json:"spec_version"`
	FindingsID    string                 `json:"findings_id"`
	Timestamp     string                 `json:"timestamp"`
	Source        FindingsSource         `json:"source"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Findings      []DocsFinding          `json:"findings"`
	Summary       FindingsSummary        `json:"summary"`
}

// FindingsSource contains metadata about where findings came from.
type FindingsSource struct {
	CheckName  string `json:"check_name"`
	Workflow   string `json:"workflow"`
	RunID      int64  `json:"run_id"`
	RunNumber  int    `json:"run_number,omitempty"`
	Repository string `json:"repository,omitempty"`
	Branch     string `json:"branch,omitempty"`
	CommitSHA  string `json:"commit_sha,omitempty"`
}

// ProtocolFinding is a single protocol violation.
type ProtocolFinding struct {
	FindingKey  string          `json:"finding_key"`
	Severity    string          `json:"severity"`
	Category    string          `json:"category"`
	Code        string          `json:"code,omitempty"`
	File        string          `json:"file"`
	Line        int             `json:"line,omitempty"`
	Message     string          `json:"message"`
	Remediation *Remediation    `json:"remediation,omitempty"`
	Context     ProtocolContext `json:"context,omitempty"`
}

// DocsFinding is a single documentation violation.
type DocsFinding struct {
	FindingKey  string       `json:"finding_key"`
	Severity    string       `json:"severity"`
	Category    string       `json:"category"`
	Code        string       `json:"code,omitempty"`
	File        string       `json:"file"`
	Line        int          `json:"line,omitempty"`
	Column      int          `json:"column,omitempty"`
	EndLine     int          `json:"end_line,omitempty"`
	Message     string       `json:"message"`
	Remediation *Remediation `json:"remediation,omitempty"`
	Context     DocsContext  `json:"context,omitempty"`
}

// Remediation provides hints for fixing an issue.
type Remediation struct {
	Hint         string `json:"hint,omitempty"`
	Action       string `json:"action,omitempty"`
	Template     string `json:"template,omitempty"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
	DocURL       string `json:"doc_url,omitempty"`
}

// ProtocolContext is additional context for protocol findings.
type ProtocolContext struct {
	FeatureID    string   `json:"feature_id,omitempty"`
	WSID         string   `json:"ws_id,omitempty"`
	BeadsID      string   `json:"beads_id,omitempty"`
	RelatedFiles []string `json:"related_files,omitempty"`
}

// DocsContext is additional context for docs findings.
type DocsContext struct {
	LinkTarget   string   `json:"link_target,omitempty"`
	LinkText     string   `json:"link_text,omitempty"`
	Section      string   `json:"section,omitempty"`
	Expected     string   `json:"expected,omitempty"`
	Actual       string   `json:"actual,omitempty"`
	RelatedFiles []string `json:"related_files,omitempty"`
}

// FindingsSummary contains summary statistics.
type FindingsSummary struct {
	Total        int            `json:"total"`
	BySeverity   map[string]int `json:"by_severity,omitempty"`
	ByCategory   map[string]int `json:"by_category,omitempty"`
	LinksChecked int            `json:"links_checked,omitempty"`
	FilesChecked int            `json:"files_checked,omitempty"`
}

// GitHubClient fetches findings from GitHub.
type GitHubClient struct {
	repo   string // owner/repo format
	token  string // GitHub token (optional, uses gh CLI if empty)
	client *http.Client
}

// NewGitHubClient creates a new GitHub client.
func NewGitHubClient(repo string) *GitHubClient {
	return &GitHubClient{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetToken sets the GitHub API token.
func (c *GitHubClient) SetToken(token string) {
	c.token = token
}

// FetchArtifacts downloads findings artifacts from a GitHub Actions run.
func (c *GitHubClient) FetchArtifacts(ctx context.Context, runID int64, destDir string) ([]string, error) {
	// Use gh CLI to download artifacts
	// gh run download <run-id> -R <repo> -n <artifact-name> -D <dest-dir>
	cmd := exec.CommandContext(ctx, "gh", "run", "download", fmt.Sprintf("%d", runID),
		"-R", c.repo,
		"-n", "sdp-findings",
		"-D", destDir,
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		// Artifact may not exist, try to get all artifacts
		cmd = exec.CommandContext(ctx, "gh", "run", "download", fmt.Sprintf("%d", runID),
			"-R", c.repo,
			"-D", destDir,
		)
		fallbackOutput, fallbackErr := cmd.CombinedOutput()
		if fallbackErr != nil {
			return nil, fmt.Errorf("gh run download failed: %w\n%s", fallbackErr, string(fallbackOutput))
		}
		_ = output
	}

	// Find all JSON files in destDir
	var files []string
	walkErr := filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			files = append(files, path)
		}
		return nil
	})

	return files, walkErr
}

// GetLatestWorkflowRuns fetches the latest workflow runs.
func (c *GitHubClient) GetLatestWorkflowRuns(ctx context.Context, branch string, limit int) ([]WorkflowRun, error) {
	// Use gh CLI to list runs
	args := []string{"run", "list", "-R", c.repo, "--json", "id,name,headBranch,status,conclusion,createdAt", "-L", fmt.Sprintf("%d", limit)}
	if branch != "" {
		args = append(args, "-b", branch)
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh run list failed: %w", err)
	}

	var runs []WorkflowRun
	if err := json.Unmarshal(output, &runs); err != nil {
		return nil, fmt.Errorf("parse workflow runs: %w", err)
	}

	return runs, nil
}

// WorkflowRun represents a GitHub Actions workflow run.
type WorkflowRun struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	HeadBranch string `json:"headBranch"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	CreatedAt  string `json:"createdAt"`
}

// ParseFindingsFile parses a findings JSON file.
func ParseFindingsFile(path string) (interface{}, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}

	// Try to detect type by looking at check_name or structure
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, "", fmt.Errorf("parse JSON: %w", err)
	}

	source, ok := raw["source"].(map[string]interface{})
	if !ok {
		return nil, "", fmt.Errorf("missing source field")
	}

	checkName, _ := source["check_name"].(string)

	switch {
	case strings.Contains(checkName, "protocol"):
		var pf ProtocolFindings
		if err := json.Unmarshal(data, &pf); err != nil {
			return nil, "", fmt.Errorf("parse protocol findings: %w", err)
		}
		return &pf, "protocol", nil
	case strings.Contains(checkName, "doc"):
		var df DocsFindings
		if err := json.Unmarshal(data, &df); err != nil {
			return nil, "", fmt.Errorf("parse docs findings: %w", err)
		}
		return &df, "docs", nil
	default:
		// Try protocol first, then docs
		var pf ProtocolFindings
		if err := json.Unmarshal(data, &pf); err == nil && len(pf.Findings) > 0 {
			return &pf, "protocol", nil
		}

		var df DocsFindings
		if err := json.Unmarshal(data, &df); err == nil && len(df.Findings) > 0 {
			return &df, "docs", nil
		}

		return nil, "", fmt.Errorf("unknown findings format")
	}
}

// LoadLocalFindings loads findings from a local directory.
func LoadLocalFindings(dir string) ([]interface{}, []string, error) {
	var findings []interface{}
	var types []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		f, t, err := ParseFindingsFile(path)
		if err != nil {
			continue // Skip unparseable files
		}

		findings = append(findings, f)
		types = append(types, t)
	}

	return findings, types, nil
}
