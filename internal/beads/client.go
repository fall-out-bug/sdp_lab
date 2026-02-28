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

	// Use bd command to find database
	cmd := exec.Command("bd", "db", "path")
	output, err := cmd.Output()
	if err == nil {
		path := strings.TrimSpace(string(output))
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return path, nil
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
			SELECT DISTINCT issue_id FROM dependencies WHERE dependency_type = 'blocks'
		)
		ORDER BY priority DESC, created_at ASC
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query ready issues: %w", err)
	}
	defer rows.Close()

	var issues []ReadyIssue
	for rows.Next() {
		var issue ReadyIssue
		var createdAt, updatedAt string
		err := rows.Scan(&issue.ID, &issue.Title, &issue.Status, &issue.Priority, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan issue: %w", err)
		}
		issue.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		issue.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
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
		var createdAt, updatedAt string
		err := rows.Scan(&issue.ID, &issue.Title, &issue.Status, &issue.Priority, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan blocking issue: %w", err)
		}
		issue.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		issue.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
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
	cmd := exec.Command("bd", "ready", "--format", "json")
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
