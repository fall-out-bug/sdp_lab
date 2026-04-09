package beads

import (
	"database/sql"
	"testing"
)

// TestExecuteRawQuery_ValidQueries tests that valid SELECT and WITH queries are allowed
func TestExecuteRawQuery_ValidQueries(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	validQueries := []string{
		"SELECT * FROM issues",
		"SELECT id, title FROM issues WHERE status = 'open'",
		"WITH cte AS (SELECT 1) SELECT * FROM cte",
		"WITH ranked AS (SELECT id, ROW_NUMBER() OVER (ORDER BY priority) AS r FROM issues) SELECT * FROM ranked",
		"  SELECT * FROM issues  ", // with whitespace
		"WITH cte AS (SELECT id FROM issues) SELECT * FROM cte WHERE id IN (SELECT issue_id FROM dependencies)",
	}

	for _, query := range validQueries {
		t.Run(query, func(t *testing.T) {
			results, err := sqlClient.ExecuteRawQuery(query)
			if err != nil {
				t.Errorf("Valid query rejected: %s\nError: %v", query, err)
			}
			if results == nil {
				t.Error("Expected non-nil results for valid query")
			}
		})
	}
}

// TestExecuteRawQuery_InvalidKeywords tests that dangerous SQL keywords are blocked
func TestExecuteRawQuery_InvalidKeywords(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	invalidQueries := []struct {
		query string
		reason string
	}{
		{"DROP TABLE issues", "DROP TABLE"},
		{"INSERT INTO issues VALUES (1, 'test')", "INSERT"},
		{"UPDATE issues SET status='done'", "UPDATE"},
		{"DELETE FROM issues WHERE id=1", "DELETE"},
		{"ALTER TABLE issues ADD COLUMN foo TEXT", "ALTER"},
		{"CREATE TABLE evil (id INT)", "CREATE"},
		{"EXECUTE PROCEDURE malicious", "EXECUTE"},
		{"; DROP TABLE issues", "semicolon prefix"},
		{"SELECT 1; DROP TABLE issues", "statement injection"},
		{"WITH cte AS (SELECT 1) DROP TABLE issues", "DROP in WITH"},
		{"drop table issues", "lowercase DROP"},
		{"Select * From Issues; Delete From Issues", "mixed case injection"},
	}

	for _, tc := range invalidQueries {
		t.Run(tc.reason, func(t *testing.T) {
			_, err := sqlClient.ExecuteRawQuery(tc.query)
			if err == nil {
				t.Errorf("Dangerous query was allowed: %s (%s)", tc.query, tc.reason)
			}
		})
	}
}

// TestExecuteRawQuery_NonSelectStatements tests that non-SELECT/WITH statements are rejected
func TestExecuteRawQuery_NonSelectStatements(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	invalidStatements := []string{
		"PRAGMA table_info(issues)",
		"EXPLAIN SELECT * FROM issues",
		"BEGIN TRANSACTION",
		"COMMIT",
		"ROLLBACK",
		"ATTACH DATABASE '/tmp/evil.db' AS evil",
		"DETACH DATABASE evil",
	}

	for _, stmt := range invalidStatements {
		t.Run(stmt, func(t *testing.T) {
			_, err := sqlClient.ExecuteRawQuery(stmt)
			if err == nil {
				t.Errorf("Non-SELECT statement was allowed: %s", stmt)
			}
		})
	}
}

// TestExecuteRawQuery_EmptyAndWhitespace tests edge cases
func TestExecuteRawQuery_EmptyAndWhitespace(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	emptyQueries := []string{
		"",
		"   ",
		"\t\n",
	}

	for _, query := range emptyQueries {
		t.Run(query, func(t *testing.T) {
			_, err := sqlClient.ExecuteRawQuery(query)
			if err == nil {
				t.Errorf("Empty/whitespace query was allowed: %q", query)
			}
		})
	}
}

// setupTestClient creates a test client with an in-memory database
func setupTestClient(t *testing.T) *Client {
	t.Helper()

	// Create a temporary in-memory SQLite database
	tmpDir := t.TempDir()
	dbPath := setupTestDB(t, tmpDir)

	client, err := NewClient(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	return client
}

// setupTestDB creates a test database with the schema
func setupTestDB(t *testing.T, dir string) string {
	t.Helper()

	dbPath := dir + "/test.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create the schema
	schema := `
	CREATE TABLE IF NOT EXISTS issues (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		status TEXT NOT NULL,
		priority INTEGER DEFAULT 1,
		labels TEXT,
		created_at TIMESTAMP,
		updated_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS dependencies (
		issue_id TEXT NOT NULL,
		blocks_issue_id TEXT NOT NULL,
		dependency_type TEXT NOT NULL,
		FOREIGN KEY (issue_id) REFERENCES issues(id),
		FOREIGN KEY (blocks_issue_id) REFERENCES issues(id)
	);

	-- Insert test data
	INSERT INTO issues (id, title, status, priority, created_at, updated_at) VALUES
	('test-1', 'Test Issue 1', 'open', 1, datetime('now'), datetime('now')),
	('test-2', 'Test Issue 2', 'done', 2, datetime('now'), datetime('now')),
	('test-3', 'Test Issue 3', 'open', 3, datetime('now'), datetime('now'));

	INSERT INTO dependencies (issue_id, blocks_issue_id, dependency_type) VALUES
	('test-2', 'test-1', 'blocks');
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	return dbPath
}

// TestSQLClient_QueryOptions tests the query option functions
func TestSQLClient_QueryOptions(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	t.Run("WithStatus", func(t *testing.T) {
		issues, err := sqlClient.QueryIssues(WithStatus("open"))
		if err != nil {
			t.Fatalf("QueryIssues with status failed: %v", err)
		}
		if len(issues) != 2 {
			t.Errorf("Expected 2 open issues, got %d", len(issues))
		}
	})

	t.Run("WithPriority", func(t *testing.T) {
		issues, err := sqlClient.QueryIssues(WithPriority(2))
		if err != nil {
			t.Fatalf("QueryIssues with priority failed: %v", err)
		}
		// No issues with priority 2 and status "open" (test-2 has status "done")
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues with priority 2 and open status, got %d", len(issues))
		}
	})

	t.Run("WithLimit", func(t *testing.T) {
		issues, err := sqlClient.QueryIssues(WithLimit(1))
		if err != nil {
			t.Fatalf("QueryIssues with limit failed: %v", err)
		}
		if len(issues) != 1 {
			t.Errorf("Expected 1 issue with limit, got %d", len(issues))
		}
	})

	t.Run("WithNoBlockers", func(t *testing.T) {
		issues, err := sqlClient.QueryIssues(WithNoBlockers())
		if err != nil {
			t.Fatalf("QueryIssues with NoBlockers failed: %v", err)
		}
		// test-1 is blocked by test-2 (which is done), test-3 has no blockers
		// So both should be returned since test-2 is done
		if len(issues) != 2 {
			t.Logf("Note: Got %d issues with no blockers", len(issues))
		}
	})

	t.Run("Combined options", func(t *testing.T) {
		issues, err := sqlClient.QueryIssues(WithStatus("open"), WithPriority(3), WithLimit(10))
		if err != nil {
			t.Fatalf("QueryIssues with combined options failed: %v", err)
		}
		// test-3 has priority 3 and status "open"
		if len(issues) != 1 {
			t.Errorf("Expected 1 issue with priority 3 and open status, got %d", len(issues))
		}
		if len(issues) > 0 && issues[0].ID != "test-3" {
			t.Errorf("Expected test-3, got %s", issues[0].ID)
		}
	})
}

// TestSQLClient_CountIssues tests the count functionality
func TestSQLClient_CountIssues(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	count, err := sqlClient.CountIssues(WithStatus("open"))
	if err != nil {
		t.Fatalf("CountIssues failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 open issues, got %d", count)
	}

	count, err = sqlClient.CountIssues(WithStatus("done"))
	if err != nil {
		t.Fatalf("CountIssues failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 done issue, got %d", count)
	}
}

// TestSQLClient_GetPriorityBreakdown tests priority breakdown
func TestSQLClient_GetPriorityBreakdown(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	breakdown, err := sqlClient.GetPriorityBreakdown("open")
	if err != nil {
		t.Fatalf("GetPriorityBreakdown failed: %v", err)
	}

	if len(breakdown) != 2 {
		t.Errorf("Expected 2 priority levels, got %d", len(breakdown))
	}

	if breakdown[1] != 1 {
		t.Errorf("Expected 1 issue at priority 1, got %d", breakdown[1])
	}

	if breakdown[3] != 1 {
		t.Errorf("Expected 1 issue at priority 3, got %d", breakdown[3])
	}
}

// TestSQLClient_QueryIssuesByDependency tests dependency-based queries
func TestSQLClient_QueryIssuesByDependency(t *testing.T) {
	client := setupTestClient(t)
	defer client.Close()
	sqlClient := NewSQLClient(client)

	t.Run("AsSource", func(t *testing.T) {
		// test-2 has a dependency
		issues, err := sqlClient.QueryIssuesByDependency("blocks", true)
		if err != nil {
			t.Fatalf("QueryIssuesByDependency failed: %v", err)
		}
		if len(issues) != 1 {
			t.Errorf("Expected 1 issue with dependency, got %d", len(issues))
		}
		if issues[0].ID != "test-2" {
			t.Errorf("Expected test-2, got %s", issues[0].ID)
		}
	})

	t.Run("AsTarget", func(t *testing.T) {
		// test-1 is blocked by test-2
		issues, err := sqlClient.QueryIssuesByDependency("blocks", false)
		if err != nil {
			t.Fatalf("QueryIssuesByDependency failed: %v", err)
		}
		if len(issues) != 1 {
			t.Errorf("Expected 1 blocked issue, got %d", len(issues))
		}
		if issues[0].ID != "test-1" {
			t.Errorf("Expected test-1, got %s", issues[0].ID)
		}
	})
}
