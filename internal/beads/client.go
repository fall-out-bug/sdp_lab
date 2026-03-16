// Package beads provides integration with the Beads issue tracking system.
package beads

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// parseTime parses a time string in RFC3339 format, returning a default time on error.
// This ensures issues are still returned even if timestamps are malformed.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Log to stderr for debugging, but don't fail the operation
		fmt.Fprintf(os.Stderr, "beads: failed to parse time %q: %v\n", s, err)
		return time.Time{}
	}
	return t
}

// Issue represents a Beads issue.
type Issue struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Priority  int       `json:"priority"`
	Labels    []string  `json:"labels,omitempty"`
	Blocks    []string  `json:"blocks,omitempty"`
	BlockedBy []string  `json:"blocked_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ReadyIssue is an issue ready to be worked on (no blockers).
type ReadyIssue struct {
	Issue
	WSID string `json:"ws_id,omitempty"` // SDP workstream ID if mapped
}

type ListedIssue struct {
	Issue
	DependencyCount int `json:"dependency_count,omitempty"`
	DependentCount  int `json:"dependent_count,omitempty"`
}

type DependencyIssue struct {
	Issue
	DependencyType string `json:"dependency_type,omitempty"`
}

// Client provides access to Beads data.
type Client struct {
	dbPath string
	db     *sql.DB
}

// NewClient creates a new Beads client.
// If dbPath is empty, it attempts to find the Beads database automatically.
func NewClient(dbPath string) (*Client, error) {
	if dbPath == "" {
		var err error
		dbPath, err = findBeadsDB()
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open beads db: %w", err)
	}

	return &Client{dbPath: dbPath, db: db}, nil
}

// Close closes the database connection.
func (c *Client) Close() error {
	return c.db.Close()
}

// findBeadsDB locates the Beads database file.
func findBeadsDB() (string, error) {
	// Check common locations
	locations := []string{
		".beads/beads.db",
		".beads/issues.db",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	cmd := exec.Command("bd", "where")
	output, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "database:") {
				path := strings.TrimSpace(strings.TrimPrefix(line, "database:"))
				if path != "" {
					if _, err := os.Stat(path); err == nil {
						if strings.HasSuffix(path, "/dolt") {
							return "", fmt.Errorf("beads database is Dolt-backed; direct SQLite access is not supported by internal/beads client")
						}
						return path, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("beads database not found")
}

// QueryReadyIssues returns issues that are ready to work on (status open, no blockers).
func (c *Client) QueryReadyIssues() ([]ReadyIssue, error) {
	// Query for open issues with no blockers
	query := `
		SELECT id, title, status, priority, created_at, updated_at
		FROM issues
		WHERE status = 'open'
		AND id NOT IN (
			SELECT DISTINCT issue_id FROM dependencies WHERE type = 'blocks'
		)
		ORDER BY priority ASC, created_at ASC
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query ready issues: %w", err)
	}
	defer rows.Close()

	var issues []ReadyIssue
	for rows.Next() {
		var issue ReadyIssue
		if err := scanIssueRow(rows, &issue.Issue, "scan issue"); err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// GetBlockingIssues returns all issues that block the given issue (transitively).
func (c *Client) GetBlockingIssues(issueID string) ([]Issue, error) {
	// Use recursive CTE for transitive blocking
	query := `
		WITH RECURSIVE blockers AS (
			SELECT id, title, status, priority, created_at, updated_at
			FROM issues
			WHERE id IN (
				SELECT blocks_issue_id FROM dependencies WHERE issue_id = ?
			)
			UNION ALL
			SELECT i.id, i.title, i.status, i.priority, i.created_at, i.updated_at
			FROM issues i
			JOIN dependencies d ON i.id = d.blocks_issue_id
			JOIN blockers b ON d.issue_id = b.id
		)
		SELECT * FROM blockers
	`

	rows, err := c.db.Query(query, issueID)
	if err != nil {
		return nil, fmt.Errorf("query blocking issues: %w", err)
	}
	defer rows.Close()

	var issues []Issue
	for rows.Next() {
		var issue Issue
		if err := scanIssueRow(rows, &issue, "scan blocking issue"); err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// MappingFile represents the .beads-sdp-mapping.jsonl file.
type MappingFile struct {
	path    string
	entries map[string]string // beads_id -> sdp_id
}

// LoadMapping loads the Beads to SDP mapping file.
func LoadMapping(projectRoot string) (*MappingFile, error) {
	path := filepath.Join(projectRoot, ".beads-sdp-mapping.jsonl")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mapping file: %w", err)
	}

	mf := &MappingFile{
		path:    path,
		entries: make(map[string]string),
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry struct {
			SDPID   string `json:"sdp_id"`
			BeadsID string `json:"beads_id"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}
		mf.entries[entry.BeadsID] = entry.SDPID
	}

	return mf, nil
}

// GetSDPID returns the SDP workstream ID for a Beads issue ID.
func (mf *MappingFile) GetSDPID(beadsID string) string {
	return mf.entries[beadsID]
}

// EmptyMapping returns an empty mapping file.
func EmptyMapping() *MappingFile {
	return &MappingFile{entries: make(map[string]string)}
}

// ReadyCommand wraps the `bd ready` command and parses output.
func ReadyCommand() ([]ReadyIssue, error) {
	cmd := exec.Command("bd", "ready", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd ready: %w", err)
	}

	var issues []ReadyIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("parse bd ready output: %w", err)
	}

	return issues, nil
}

func ListIssuesCommand(all bool) ([]ListedIssue, error) {
	args := []string{"list", "--json", "-n", "0"}
	if all {
		args = append(args, "--all")
	}
	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list: %w", err)
	}
	var issues []ListedIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("parse bd list output: %w", err)
	}
	return issues, nil
}

func DependencyListCommand(issueID, direction, depType string) ([]DependencyIssue, error) {
	args := []string{"dep", "list", issueID, "--json"}
	if strings.TrimSpace(direction) != "" {
		args = append(args, "--direction", direction)
	}
	if strings.TrimSpace(depType) != "" {
		args = append(args, "--type", depType)
	}
	cmd := exec.Command("bd", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd dep list: %w", err)
	}
	var issues []DependencyIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("parse bd dep list output: %w", err)
	}
	return issues, nil
}

func DependencyTreeCommand(issueID string) ([]Issue, error) {
	cmd := exec.Command("bd", "dep", "tree", issueID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd dep tree: %w", err)
	}
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("parse bd dep tree output: %w", err)
	}
	return issues, nil
}

// ReadyWithBlockersCommand gets ready issues with blocker information.
// This calls bd ready first, then enriches with blocker data from bd show.
func ReadyWithBlockersCommand() ([]ReadyIssue, error) {
	// Get base ready issues
	issues, err := ReadyCommand()
	if err != nil {
		return nil, err
	}

	// Enrich each issue with blocker information
	for i := range issues {
		blockers, err := getBlockersForIssue(issues[i].ID)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "warning: get blockers for %s: %v\n", issues[i].ID, err)
			continue
		}
		issues[i].BlockedBy = blockers
	}

	return issues, nil
}

// getBlockersForIssue fetches blocker information for a single issue.
func getBlockersForIssue(issueID string) ([]string, error) {
	cmd := exec.Command("bd", "show", issueID, "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show: %w", err)
	}

	// Parse issue details
	var issueData []struct {
		Dependencies []struct {
			ID             string `json:"id"`
			Status         string `json:"status"`
			DependencyType string `json:"dependency_type"`
		} `json:"dependencies"`
	}

	if err := json.Unmarshal(output, &issueData); err != nil {
		return nil, fmt.Errorf("parse bd show output: %w", err)
	}

	if len(issueData) == 0 {
		return nil, nil
	}

	// Extract blockers (dependencies with type='blocks' and status != 'done')
	var blockers []string
	for _, dep := range issueData[0].Dependencies {
		if dep.DependencyType == "blocks" && dep.Status != "done" && dep.Status != "closed" {
			blockers = append(blockers, dep.ID)
		}
	}

	return blockers, nil
}
