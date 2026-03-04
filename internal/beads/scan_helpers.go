package beads

import (
	"database/sql"
	"fmt"
)

func scanIssueRow(rows *sql.Rows, issue *Issue, errContext string) error {
	var createdAt string
	var updatedAt string
	if err := rows.Scan(&issue.ID, &issue.Title, &issue.Status, &issue.Priority, &createdAt, &updatedAt); err != nil {
		return fmt.Errorf("%s: %w", errContext, err)
	}
	issue.CreatedAt = parseTime(createdAt)
	issue.UpdatedAt = parseTime(updatedAt)
	return nil
}

func scanDependencyRow(rows *sql.Rows, dep *Dependency, errContext string) error {
	var createdAt string
	if err := rows.Scan(&dep.FromIssueID, &dep.ToIssueID, &dep.DependencyType, &createdAt); err != nil {
		return fmt.Errorf("%s: %w", errContext, err)
	}
	dep.CreatedAt = parseTime(createdAt)
	return nil
}
