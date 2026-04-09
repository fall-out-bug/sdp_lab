package beads


import (
	"database/sql"
	"testing"
	_ "github.com/mattn/go-sqlite3"
)

func TestDependencyType(t *testing.T) {
	tests := []struct {
		name string
		dep  DependencyType
	}{
		{"blocks", DependencyTypeBlocks},
		{"parent-child", DependencyTypeParentChild},
		{"discovered-from", DependencyTypeDiscovered},
		{"related", DependencyTypeRelated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.dep) != tt.name {
				t.Errorf("DependencyType mismatch: got %s, want %s", string(tt.dep), tt.name)
			}
        })
	 }
}

func TestDependencyQuery_GetDependencies(t *testing.T) {
	// Create in-memory database
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to create in-memory db: %v", err)
    }
    defer func() { _ = db.Close() }()

    // Create schema
    _, err = db.Exec(`
        CREATE TABLE issues (
            id TEXT PRIMARY KEY,
            title TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'open',
            priority INTEGER NOT NULL DEFAULT 2,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        CREATE TABLE dependencies (
            issue_id TEXT NOT NULL,
            depends_on_id TEXT NOT NULL,
            type TEXT NOT NULL DEFAULT 'blocks',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            created_by TEXT NOT NULL,
            metadata TEXT,
            thread_id TEXT,
            PRIMARY KEY (issue_id, depends_on_id, type)
            FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
        );
    `)
    if err != nil {
        t.Fatalf("Failed to create schema: %v", err)
    }

    // Insert test issues
    _, err = db.Exec(`
        INSERT INTO issues (id, title, status, priority) VALUES
        ('issue-1', 'First issue', 'open', 1),
        ('issue-2', 'Second issue', 'open', 2),
        ('issue-3', 'Third issue', 'done', 1)
    `)
    if err != nil {
        t.Fatalf("Failed to insert issues: %v", err)
    }

    // Insert test dependencies
    _, err = db.Exec(`
        INSERT INTO dependencies (issue_id, depends_on_id, type, created_by) VALUES
        ('issue-1', 'issue-2', 'blocks', 'test'),
        ('issue-1', 'issue-3', 'parent-child', 'test'),
        ('issue-2', 'issue-3', 'discovered-from', 'test')
    `)
    if err != nil {
        t.Fatalf("Failed to insert dependencies: %v", err)
    }

    // Create client and query
    client := &Client{db: db}
    depQuery := NewDependencyQuery(client)

    // Test GetDependencies for blocks
    t.Run("blocks", func(t *testing.T) {
        deps, err := depQuery.GetDependencies(DependencyTypeBlocks)
        if err != nil {
            t.Fatalf("GetDependencies failed: %v", err)
        }
        if len(deps) != 1 {
            t.Errorf("Expected 1 blocks dependency, got %d", len(deps))
        }
        if deps[0].FromIssueID != "issue-1" || deps[0].ToIssueID != "issue-2" {
            t.Errorf("Unexpected dependency: %+v", deps[0])
        }
    })

    // Test GetDependencies for parent-child
    t.Run("parent-child", func(t *testing.T) {
        deps, err := depQuery.GetDependencies(DependencyTypeParentChild)
        if err != nil {
            t.Fatalf("GetDependencies failed: %v", err)
        }
        if len(deps) != 1 {
            t.Errorf("Expected 1 parent-child dependency, got %d", len(deps))
        }
    })

    // Test GetDependencies for discovered-from
    t.Run("discovered-from", func(t *testing.T) {
        deps, err := depQuery.GetDependencies(DependencyTypeDiscovered)
        if err != nil {
            t.Fatalf("GetDependencies failed: %v", err)
        }
        if len(deps) != 1 {
            t.Errorf("Expected 1 discovered-from dependency, got %d", len(deps))
        }
    })
}

func TestDependencyQuery_GetBlockingDependencies(t *testing.T) {
    // Create in-memory database
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatalf("Failed to create in-memory db: %v", err)
    }
    defer func() { _ = db.Close() }()

    // Create schema
    _, err = db.Exec(`
        CREATE TABLE issues (
            id TEXT PRIMARY KEY,
            title TEXT NOT NULL,
            status TEXT NOT NULL DEFAULT 'open',
            priority INTEGER NOT NULL DEFAULT 2,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        CREATE TABLE dependencies (
            issue_id TEXT NOT NULL,
            depends_on_id TEXT NOT NULL,
            type TEXT NOT NULL DEFAULT 'blocks',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            created_by TEXT NOT NULL,
            metadata TEXT,
            thread_id TEXT,
            PRIMARY KEY (issue_id, depends_on_id, type)
            FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
        );
    `)
    if err != nil {
        t.Fatalf("Failed to create schema: %v", err)
    }

    // Insert test issues
    _, err = db.Exec(`
        INSERT INTO issues (id, title, status, priority) VALUES
        ('ready-1', 'Ready issue', 'open', 1),
        ('blocked-1', 'Blocked issue', 'open', 2),
        ('blocker-1', 'Blocker issue', 'open', 1),
        ('done-blocker', 'Done blocker', 'done', 1)
    `)
    if err != nil {
        t.Fatalf("Failed to insert issues: %v", err)
    }

    // Insert blocking dependencies
    _, err = db.Exec(`
        INSERT INTO dependencies (issue_id, depends_on_id, type, created_by) VALUES
        ('blocked-1', 'blocker-1', 'blocks', 'test'),
        ('blocked-1', 'done-blocker', 'blocks', 'test')
    `)
    if err != nil {
        t.Fatalf("Failed to insert dependencies: %v", err)
    }

    // Create client and query
    client := &Client{db: db}
    depQuery := NewDependencyQuery(client)

    // Test HasOpenBlockers for ready issue
    hasBlockers, err := depQuery.HasOpenBlockers("ready-1")
    if err != nil {
        t.Fatalf("HasOpenBlockers failed: %v", err)
    }
    if hasBlockers {
        t.Error("ready-1 should not have open blockers")
    }

    // Test HasOpenBlockers for blocked issue
    hasBlockers, err = depQuery.HasOpenBlockers("blocked-1")
    if err != nil {
        t.Fatalf("HasOpenBlockers failed: %v", err)
    }
    if !hasBlockers {
        t.Error("blocked-1 should have open blockers")
    }
}

