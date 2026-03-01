// Package beads provides dependency graph query capabilities for Beads issues.
package beads

import (
	"fmt"
	"time"
)

// DependencyType represents the type of relationship between issues.
type DependencyType string

const (
	DependencyTypeBlocks      DependencyType = "blocks"          // A blocks B (B cannot start until A is done)
	DependencyTypeParentChild DependencyType = "parent-child"    // A is parent of B (B is subtask of A)
	DependencyTypeDiscovered  DependencyType = "discovered-from" // B was discovered from A
	DependencyTypeRelated     DependencyType = "related"         // A and B are related (no blocking)
)

// Dependency represents a relationship between two issues.
type Dependency struct {
	FromIssueID    string         `json:"from_issue_id"`
	ToIssueID      string         `json:"to_issue_id"`
	DependencyType DependencyType `json:"type"`
	CreatedAt      time.Time      `json:"created_at"`
}

// DependencyQuery provides methods to query dependency graph.
type DependencyQuery struct {
	client *Client
}

// NewDependencyQuery creates a new dependency query client.
func NewDependencyQuery(client *Client) *DependencyQuery {
	return &DependencyQuery{client: client}
}

// GetDependencies returns all dependencies of a specific type.
func (dq *DependencyQuery) GetDependencies(depType DependencyType) ([]Dependency, error) {
	query := `
		SELECT issue_id, depends_on_id, type, created_at
		FROM dependencies
		WHERE type = ?
		ORDER BY created_at DESC
	`

	rows, err := dq.client.db.Query(query, string(depType))
	if err != nil {
		return nil, fmt.Errorf("query dependencies: %w", err)
	}
	defer rows.Close()

	var deps []Dependency
	for rows.Next() {
		var dep Dependency
		var createdAt string
		err := rows.Scan(&dep.FromIssueID, &dep.ToIssueID, &dep.DependencyType, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan dependency: %w", err)
		}
		dep.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		deps = append(deps, dep)
	}

	return deps, nil
}

// GetBlockingDependencies returns all blocking dependencies for a given issue.
func (dq *DependencyQuery) GetBlockingDependencies(issueID string) ([]Dependency, error) {
	query := `
		SELECT issue_id, depends_on_id, type, created_at
		FROM dependencies
		WHERE issue_id = ? AND type = 'blocks'
	`

	rows, err := dq.client.db.Query(query, issueID)
	if err != nil {
		return nil, fmt.Errorf("query blocking dependencies: %w", err)
	}
	defer rows.Close()

	var deps []Dependency
	for rows.Next() {
		var dep Dependency
		var createdAt string
		err := rows.Scan(&dep.FromIssueID, &dep.ToIssueID, &dep.DependencyType, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan blocking dependency: %w", err)
		}
		dep.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		deps = append(deps, dep)
	}

	return deps, nil
}

// GetReadyIssuesWithDeps returns issues ready to work, considering all dependency types.
// An issue is ready if:
// - Status is 'open'
// - No blocking dependencies (type = 'blocks')
// - Parent-child, discovered-from, and related dependencies do NOT block
func (dq *DependencyQuery) GetReadyIssuesWithDeps() ([]ReadyIssue, error) {
	// Query for open issues with no blocking dependencies
	// Only 'blocks' type prevents ready status
	query := `
		SELECT i.id, i.title, i.status, i.priority, i.created_at, i.updated_at
		FROM issues i
		WHERE i.status = 'open'
		AND i.id NOT IN (
			SELECT DISTINCT d.issue_id 
			FROM dependencies d 
			WHERE d.type = 'blocks'
			AND EXISTS (
				SELECT 1 FROM issues bi WHERE bi.id = d.depends_on_id AND bi.status != 'done'
			)
		)
		ORDER BY i.priority ASC, i.created_at ASC
	`

	rows, err := dq.client.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query ready issues with deps: %w", err)
	}
	defer rows.Close()

	var issues []ReadyIssue
	for rows.Next() {
		var issue ReadyIssue
		var createdAt, updatedAt string
		err := rows.Scan(&issue.ID, &issue.Title, &issue.Status, &issue.Priority, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan ready issue: %w", err)
		}
		issue.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		issue.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		issues = append(issues, issue)
	}

	return issues, nil
}

// GetDependencyGraph returns the full dependency graph for visualization.
func (dq *DependencyQuery) GetDependencyGraph() (map[string][]Dependency, error) {
	query := `
		SELECT issue_id, depends_on_id, type, created_at
		FROM dependencies
		ORDER BY type, created_at DESC
	`

	rows, err := dq.client.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query dependency graph: %w", err)
	}
	defer rows.Close()

	graph := make(map[string][]Dependency)
	for rows.Next() {
		var dep Dependency
		var createdAt string
		err := rows.Scan(&dep.FromIssueID, &dep.ToIssueID, &dep.DependencyType, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan dependency graph: %w", err)
		}
		dep.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		graph[dep.FromIssueID] = append(graph[dep.FromIssueID], dep)
	}

	return graph, nil
}

// GetRelatedIssues returns all issues related to the given issue (any dependency type).
func (dq *DependencyQuery) GetRelatedIssues(issueID string) ([]Dependency, error) {
	query := `
		SELECT issue_id, depends_on_id, type, created_at
		FROM dependencies
		WHERE issue_id = ? OR depends_on_id = ?
		ORDER BY type, created_at DESC
	`

	rows, err := dq.client.db.Query(query, issueID, issueID)
	if err != nil {
		return nil, fmt.Errorf("query related issues: %w", err)
	}
	defer rows.Close()

	var deps []Dependency
	for rows.Next() {
		var dep Dependency
		var createdAt string
		err := rows.Scan(&dep.FromIssueID, &dep.ToIssueID, &dep.DependencyType, &createdAt)
		if err != nil {
			return nil, fmt.Errorf("scan related issue: %w", err)
		}
		dep.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		deps = append(deps, dep)
	}

	return deps, nil
}

// HasOpenBlockers checks if an issue has any open blocking dependencies.
func (dq *DependencyQuery) HasOpenBlockers(issueID string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM dependencies d
		JOIN issues i ON i.id = d.depends_on_id
		WHERE d.issue_id = ? 
		AND d.type = 'blocks'
		AND i.status != 'done'
	`

	var count int
	err := dq.client.db.QueryRow(query, issueID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check open blockers: %w", err)
	}

	return count > 0, nil
}

// GetTransitiveBlockers returns all issues that transitively block the given issue.
func (dq *DependencyQuery) GetTransitiveBlockers(issueID string) ([]Issue, error) {
	// Use recursive CTE for transitive blocking
	query := `
		WITH RECURSIVE blockers AS (
			SELECT i.id, i.title, i.status, i.priority, i.created_at, i.updated_at, 1 as depth
			FROM issues i
			WHERE i.id IN (
				SELECT d.depends_on_id 
				FROM dependencies d 
				WHERE d.issue_id = ? AND d.type = 'blocks'
			)
			UNION ALL
			SELECT i.id, i.title, i.status, i.priority, i.created_at, i.updated_at, b.depth + 1
			FROM issues i
			JOIN dependencies d ON i.id = d.depends_on_id AND d.type = 'blocks'
			JOIN blockers b ON d.issue_id = b.id
			WHERE b.depth < 10  -- Prevent infinite loops
		)
		SELECT DISTINCT id, title, status, priority, created_at, updated_at
		FROM blockers
		ORDER BY depth
	`

	rows, err := dq.client.db.Query(query, issueID)
	if err != nil {
		return nil, fmt.Errorf("query transitive blockers: %w", err)
	}
	defer rows.Close()

	var issues []Issue
	for rows.Next() {
		var issue Issue
		var createdAt, updatedAt string
		err := rows.Scan(&issue.ID, &issue.Title, &issue.Status, &issue.Priority, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan transitive blocker: %w", err)
		}
		issue.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		issue.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		issues = append(issues, issue)
	}

	return issues, nil
}

// GetDependencyStats returns statistics about dependencies in the system.
func (dq *DependencyQuery) GetDependencyStats() (map[DependencyType]int, error) {
	query := `
		SELECT type, COUNT(*)
		FROM dependencies
		GROUP BY type
	`

	rows, err := dq.client.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query dependency stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[DependencyType]int)
	for rows.Next() {
		var depType string
		var count int
		err := rows.Scan(&depType, &count)
		if err != nil {
			return nil, fmt.Errorf("scan dependency stat: %w", err)
		}
		stats[DependencyType(depType)] = count
	}

	return stats, nil
}
