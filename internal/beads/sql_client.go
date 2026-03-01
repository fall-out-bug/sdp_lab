// Package beads provides SQL client extensions for advanced Beads queries.
package beads

import (
	"database/sql"
	"fmt"
	"time"
)

// SQLClient provides advanced SQL query capabilities for Beads.
type SQLClient struct {
	db *sql.DB
}

// NewSQLClient creates a new SQL client from an existing Beads client.
func NewSQLClient(client *Client) *SQLClient {
	return &SQLClient{db: client.db}
}

// QueryOption represents a query filter option.
type QueryOption func(*QueryOptions)

// QueryOptions contains filter options for issue queries.
type QueryOptions struct {
	Status         string
	Priority       int
	LabelPattern   string
	NoBlockers     bool
	DependencyType DependencyType
	Limit          int
}

// WithStatus filters by issue status.
func WithStatus(status string) QueryOption {
	return func(opts *QueryOptions) {
		opts.Status = status
	}
}

// WithPriority filters by priority level.
func WithPriority(priority int) QueryOption {
	return func(opts *QueryOptions) {
		opts.Priority = priority
	}
}

// WithLabelPattern filters by label glob pattern.
func WithLabelPattern(pattern string) QueryOption {
	return func(opts *QueryOptions) {
		opts.LabelPattern = pattern
	}
}

// WithNoBlockers filters for issues with no blocking dependencies.
func WithNoBlockers() QueryOption {
	return func(opts *QueryOptions) {
		opts.NoBlockers = true
	}
}

// WithLimit limits the number of results.
func WithLimit(limit int) QueryOption {
	return func(opts *QueryOptions) {
		opts.Limit = limit
	}
}

// QueryIssues executes a flexible SQL query with the given options.
func (sc *SQLClient) QueryIssues(opts ...QueryOption) ([]Issue, error) {
	options := &QueryOptions{
		Status: "open",
		Limit:  50,
	}
	for _, opt := range opts {
		opt(options)
	}

	query := `
		SELECT DISTINCT i.id, i.title, i.status, i.priority, i.created_at, i.updated_at
		FROM issues i
		WHERE 1=1
	`
	args := []interface{}{}

	if options.Status != "" {
		query += " AND i.status = ?"
		args = append(args, options.Status)
	}

	if options.Priority > 0 {
		query += " AND i.priority = ?"
		args = append(args, options.Priority)
	}

	if options.NoBlockers {
		query += `
			AND i.id NOT IN (
				SELECT DISTINCT d.issue_id 
				FROM dependencies d 
				WHERE d.dependency_type = 'blocks'
				AND EXISTS (
					SELECT 1 FROM issues bi 
					WHERE bi.id = d.blocks_issue_id 
					AND bi.status != 'done'
				)
			)
		`
	}

	query += " ORDER BY i.priority ASC, i.created_at ASC"

	if options.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, options.Limit)
	}

	rows, err := sc.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query issues: %w", err)
	}
	defer rows.Close()

	var issues []Issue
	for rows.Next() {
		var issue Issue
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

// QueryIssuesByDependency returns issues that have a specific dependency type.
func (sc *SQLClient) QueryIssuesByDependency(depType DependencyType, asSource bool) ([]Issue, error) {
	var query string
	if asSource {
		// Issues that have dependencies of this type (issue_id side)
		query = `
			SELECT DISTINCT i.id, i.title, i.status, i.priority, i.created_at, i.updated_at
			FROM issues i
			JOIN dependencies d ON i.id = d.issue_id
			WHERE d.dependency_type = ?
			ORDER BY i.priority ASC, i.created_at ASC
		`
	} else {
		// Issues that are the target of dependencies (blocks_issue_id side)
		query = `
			SELECT DISTINCT i.id, i.title, i.status, i.priority, i.created_at, i.updated_at
			FROM issues i
			JOIN dependencies d ON i.id = d.blocks_issue_id
			WHERE d.dependency_type = ?
			ORDER BY i.priority ASC, i.created_at ASC
		`
	}

	rows, err := sc.db.Query(query, string(depType))
	if err != nil {
		return nil, fmt.Errorf("query issues by dependency: %w", err)
	}
	defer rows.Close()

	var issues []Issue
	for rows.Next() {
		var issue Issue
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

// CountIssues counts issues matching the given options.
func (sc *SQLClient) CountIssues(opts ...QueryOption) (int, error) {
	options := &QueryOptions{
		Status: "open",
	}
	for _, opt := range opts {
		opt(options)
	}

	query := "SELECT COUNT(DISTINCT i.id) FROM issues i WHERE 1=1"
	args := []interface{}{}

	if options.Status != "" {
		query += " AND i.status = ?"
		args = append(args, options.Status)
	}

	if options.Priority > 0 {
		query += " AND i.priority = ?"
		args = append(args, options.Priority)
	}

	if options.NoBlockers {
		query += `
			AND i.id NOT IN (
				SELECT DISTINCT d.issue_id 
				FROM dependencies d 
				WHERE d.dependency_type = 'blocks'
				AND EXISTS (
					SELECT 1 FROM issues bi 
					WHERE bi.id = d.blocks_issue_id 
					AND bi.status != 'done'
				)
			)
		`
	}

	var count int
	err := sc.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count issues: %w", err)
	}

	return count, nil
}

// GetPriorityBreakdown returns issue counts grouped by priority.
func (sc *SQLClient) GetPriorityBreakdown(status string) (map[int]int, error) {
	query := `
		SELECT priority, COUNT(*)
		FROM issues
		WHERE status = ?
		GROUP BY priority
		ORDER BY priority
	`

	rows, err := sc.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("query priority breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := make(map[int]int)
	for rows.Next() {
		var priority, count int
		err := rows.Scan(&priority, &count)
		if err != nil {
			return nil, fmt.Errorf("scan priority breakdown: %w", err)
		}
		breakdown[priority] = count
	}

	return breakdown, nil
}

// ExecuteRawQuery executes a raw SQL query and returns results as maps.
// This is for advanced users who need custom queries.
func (sc *SQLClient) ExecuteRawQuery(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := sc.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("execute raw query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("get columns: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}
		results = append(results, result)
	}

	return results, nil
}
