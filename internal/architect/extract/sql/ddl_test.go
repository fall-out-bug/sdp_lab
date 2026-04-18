// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"testing"

	"sdp_dev/internal/architect"
)

// TestParseTables tests the parseTables function with various SQL DDL statements.
func TestParseTables(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		file     string
		wantLen  int // expected number of tables
		wantFkLen int // expected number of foreign keys
	}{
		{
			name: "simple create table",
			sql:  "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));",
			file: "schema.sql",
			wantLen: 1,
			wantFkLen: 0,
		},
		{
			name: "create table with schema",
			sql:  "CREATE TABLE public.users (id INT PRIMARY KEY, email VARCHAR(255));",
			file: "schema.sql",
			wantLen: 1,
			wantFkLen: 0,
		},
		{
			name: "create table with foreign key",
			sql: `CREATE TABLE posts (
				id INT PRIMARY KEY,
				user_id INT,
				title VARCHAR(255),
				FOREIGN KEY (user_id) REFERENCES users(id)
			);`,
			file: "posts.sql",
			wantLen: 1,
			wantFkLen: 1,
		},
		{
			name: "create table if not exists",
			sql:  "CREATE TABLE IF NOT EXISTS comments (id INT PRIMARY KEY, text TEXT);",
			file: "comments.sql",
			wantLen: 1,
			wantFkLen: 0,
		},
		{
			name: "create table with quoted identifiers",
			sql:  "CREATE TABLE `orders` (id INT PRIMARY KEY, total DECIMAL(10,2));",
			file: "orders.sql",
			wantLen: 1,
			wantFkLen: 0,
		},
		{
			name: "create table with various column types",
			sql: `CREATE TABLE products (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				price DECIMAL(10,2),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				is_active BOOLEAN DEFAULT TRUE
			);`,
			file: "products.sql",
			wantLen: 1,
			wantFkLen: 0,
		},
		{
			name: "multiple tables in one file",
			sql: `CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));
			CREATE TABLE posts (id INT PRIMARY KEY, user_id INT);`,
			file: "multi.sql",
			wantLen: 2,
			wantFkLen: 0,
		},
		{
			name: "table with composite primary key",
			sql: `CREATE TABLE order_items (
				order_id INT,
				item_id INT,
				quantity INT,
				PRIMARY KEY (order_id, item_id)
			);`,
			file: "order_items.sql",
			wantLen: 1,
			wantFkLen: 0,
		},
		{
			name: "table with multiple foreign keys",
			sql: `CREATE TABLE order_items (
				order_id INT,
				product_id INT,
				quantity INT,
				FOREIGN KEY (order_id) REFERENCES orders(id),
				FOREIGN KEY (product_id) REFERENCES products(id)
			);`,
			file: "order_items.sql",
			wantLen: 1,
			wantFkLen: 2,
		},
		{
			name: "table with unique constraint",
			sql: `CREATE TABLE users (
				id INT PRIMARY KEY,
				email VARCHAR(255) UNIQUE NOT NULL,
				username VARCHAR(255) UNIQUE
			);`,
			file: "users.sql",
			wantLen: 1,
			wantFkLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables, fks := parseTables(tt.sql, tt.file)

			if len(tables) != tt.wantLen {
				t.Errorf("parseTables() returned %d tables, want %d", len(tables), tt.wantLen)
			}

			if len(fks) != tt.wantFkLen {
				t.Errorf("parseTables() returned %d foreign keys, want %d", len(fks), tt.wantFkLen)
			}

			// Verify table structure
			for _, table := range tables {
				if table.Name == "" {
					t.Error("parseTables() returned table with empty name")
				}
				if table.File != tt.file {
					t.Errorf("parseTables() table file = %s, want %s", table.File, tt.file)
				}
			}
		})
	}
}

// TestParseColumns tests column parsing with various SQL column definitions.
func TestParseColumns(t *testing.T) {
	sql := `CREATE TABLE users (
		id INT PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		name VARCHAR(100),
		age INT,
		is_active BOOLEAN DEFAULT TRUE
	);`

	tables, _ := parseTables(sql, "users.sql")

	if len(tables) != 1 {
		t.Fatalf("parseTables() returned %d tables, want 1", len(tables))
	}

	table := tables[0]
	if len(table.Columns) != 5 {
		t.Errorf("parseTables() returned %d columns, want 5", len(table.Columns))
	}

	// Check specific columns
	columnsByName := make(map[string]architect.Column)
	for _, col := range table.Columns {
		columnsByName[col.Name] = col
	}

	// Check id column (PRIMARY KEY)
	if col, ok := columnsByName["id"]; ok {
		if !col.PrimaryKey {
			t.Error("id column should have PrimaryKey = true")
		}
	} else {
		t.Error("id column not found")
	}

	// Check email column (NOT NULL)
	if col, ok := columnsByName["email"]; ok {
		if !col.NotNull {
			t.Error("email column should have NotNull = true")
		}
	} else {
		t.Error("email column not found")
	}

	// Check name column (nullable)
	if col, ok := columnsByName["name"]; ok {
		if !col.Nullable {
			t.Error("name column should have Nullable = true")
		}
	} else {
		t.Error("name column not found")
	}
}

// TestParseIndexes tests the parseIndexes function.
func TestParseIndexes(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		file    string
		wantLen int
	}{
		{
			name:    "simple index",
			sql:     "CREATE INDEX idx_user_email ON users(email);",
			file:    "indexes.sql",
			wantLen: 1,
		},
		{
			name:    "unique index",
			sql:     "CREATE UNIQUE INDEX idx_user_email ON users(email);",
			file:    "indexes.sql",
			wantLen: 1,
		},
		{
			name:    "composite index",
			sql:     "CREATE INDEX idx_post_user ON posts(user_id, created_at);",
			file:    "indexes.sql",
			wantLen: 1,
		},
		{
			name:    "index with schema",
			sql:     "CREATE INDEX idx_user_email ON public.users(email);",
			file:    "indexes.sql",
			wantLen: 1,
		},
		{
			name:    "create if not exists",
			sql:     "CREATE INDEX IF NOT EXISTS idx_user_email ON users(email);",
			file:    "indexes.sql",
			wantLen: 1,
		},
		{
			name:    "multiple indexes",
			sql:     "CREATE INDEX idx1 ON users(email); CREATE INDEX idx2 ON posts(user_id);",
			file:    "indexes.sql",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexes := parseIndexes(tt.sql, tt.file)

			if len(indexes) != tt.wantLen {
				t.Errorf("parseIndexes() returned %d indexes, want %d", len(indexes), tt.wantLen)
			}

			// Verify index structure
			for _, idx := range indexes {
				if idx.Name == "" {
					t.Error("parseIndexes() returned index with empty name")
				}
				if idx.Table == "" {
					t.Error("parseIndexes() returned index with empty table")
				}
				if idx.File != tt.file {
					t.Errorf("parseIndexes() index file = %s, want %s", idx.File, tt.file)
				}
			}
		})
	}
}

// TestParseViews tests the parseViews function.
func TestParseViews(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		file    string
		wantLen int
	}{
		{
			name:    "simple view",
			sql:     "CREATE VIEW user_emails AS SELECT email FROM users;",
			file:    "views.sql",
			wantLen: 1,
		},
		{
			name:    "materialized view",
			sql:     "CREATE MATERIALIZED VIEW user_summary AS SELECT COUNT(*) FROM users;",
			file:    "views.sql",
			wantLen: 1,
		},
		{
			name:    "view with schema",
			sql:     "CREATE VIEW public.active_users AS SELECT * FROM users WHERE active = true;",
			file:    "views.sql",
			wantLen: 1,
		},
		{
			name:    "create or replace view",
			sql:     "CREATE OR REPLACE VIEW user_emails AS SELECT email FROM users;",
			file:    "views.sql",
			wantLen: 1,
		},
		{
			name:    "multiple views",
			sql:     "CREATE VIEW v1 AS SELECT * FROM t1; CREATE VIEW v2 AS SELECT * FROM t2;",
			file:    "views.sql",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			views := parseViews(tt.sql, tt.file)

			if len(views) != tt.wantLen {
				t.Errorf("parseViews() returned %d views, want %d", len(views), tt.wantLen)
			}

			// Verify view structure
			for _, view := range views {
				if view.Name == "" {
					t.Error("parseViews() returned view with empty name")
				}
				if view.File != tt.file {
					t.Errorf("parseViews() view file = %s, want %s", view.File, tt.file)
				}
			}
		})
	}
}

// TestParseStoredProcs tests the parseStoredProcs function.
func TestParseStoredProcs(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		file    string
		wantLen int
	}{
		{
			name:    "simple function",
			sql:     "CREATE FUNCTION get_user_count() RETURNS INT AS $$ SELECT COUNT(*) FROM users; $$ LANGUAGE SQL;",
			file:    "functions.sql",
			wantLen: 1,
		},
		{
			name:    "procedure",
			sql:     "CREATE PROCEDURE update_user_stats() AS BEGIN UPDATE user_stats SET count = (SELECT COUNT(*) FROM users); END;",
			file:    "procedures.sql",
			wantLen: 1,
		},
		{
			name:    "function with schema",
			sql:     "CREATE FUNCTION public.calculate_age(birthdate DATE) RETURNS INT AS $$ ... $$;",
			file:    "functions.sql",
			wantLen: 1,
		},
		{
			name:    "create or replace function",
			sql:     "CREATE OR REPLACE FUNCTION get_user_email(user_id INT) RETURNS VARCHAR AS $$ ... $$;",
			file:    "functions.sql",
			wantLen: 1,
		},
		{
			name:    "multiple functions",
			sql:     "CREATE FUNCTION f1() RETURNS INT AS $$ SELECT 1; $$; CREATE FUNCTION f2() RETURNS INT AS $$ SELECT 2; $$;",
			file:    "functions.sql",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			procs := parseStoredProcs(tt.sql, tt.file)

			if len(procs) != tt.wantLen {
				t.Errorf("parseStoredProcs() returned %d procedures, want %d", len(procs), tt.wantLen)
			}

			// Verify procedure structure
			for _, proc := range procs {
				if proc.Name == "" {
					t.Error("parseStoredProcs() returned procedure with empty name")
				}
				if proc.Path != tt.file {
					t.Errorf("parseStoredProcs() procedure path = %s, want %s", proc.Path, tt.file)
				}
			}
		})
	}
}

// TestExtractParenBody tests the extractParenBody helper function.
func TestExtractParenBody(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		openIdx   int
		wantEmpty bool
		wantSub   string // if wantEmpty is false, check if result contains this substring
	}{
		{
			name:      "simple parentheses",
			input:     "(id INT, name VARCHAR)",
			openIdx:   0,
			wantEmpty: false,
			wantSub:   "id INT, name VARCHAR",
		},
		{
			name:      "nested parentheses",
			input:     "(id INT, name VARCHAR(255))",
			openIdx:   0,
			wantEmpty: false,
			wantSub:   "id INT, name VARCHAR(255)",
		},
		{
			name:      "multiple parentheses",
			input:     "(first) (second)",
			openIdx:   0,
			wantEmpty: false,
			wantSub:   "first",
		},
		{
			name:      "invalid open index",
			input:     "(id INT)",
			openIdx:   -1,
			wantEmpty: true,
		},
		{
			name:      "not a parenthesis",
			input:     "id INT",
			openIdx:   0,
			wantEmpty: true,
		},
		{
			name:      "unclosed parenthesis",
			input:     "(id INT, name VARCHAR",
			openIdx:   0,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractParenBody(tt.input, tt.openIdx)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("extractParenBody() = %s, want empty string", result)
				}
			} else {
				if result == "" {
					t.Error("extractParenBody() returned empty string, want non-empty")
				}
				if tt.wantSub != "" && !contains(result, tt.wantSub) {
					t.Errorf("extractParenBody() = %s, want result containing %s", result, tt.wantSub)
				}
			}
		})
	}
}

// TestGetDDLStats tests the GetDDLStats function.
func TestGetDDLStats(t *testing.T) {
	tables := []architect.Table{
		{Name: "users"},
		{Name: "posts"},
	}
	indexes := []architect.Index{
		{Name: "idx_users_email", Unique: false},
		{Name: "idx_posts_user_id", Unique: true},
	}
	views := []architect.View{
		{Name: "user_emails", Materialized: false},
		{Name: "post_stats", Materialized: true},
	}
	procs := []architect.StoredProc{
		{Name: "get_user_count"},
	}

	stats := GetDDLStats(tables, indexes, views, procs)

	if stats["tables"] != 2 {
		t.Errorf("GetDDLStats() tables = %d, want 2", stats["tables"])
	}
	if stats["indexes"] != 2 {
		t.Errorf("GetDDLStats() indexes = %d, want 2", stats["indexes"])
	}
	if stats["views"] != 2 {
		t.Errorf("GetDDLStats() views = %d, want 2", stats["views"])
	}
	if stats["stored_procs"] != 1 {
		t.Errorf("GetDDLStats() stored_procs = %d, want 1", stats["stored_procs"])
	}
	if stats["unique_index"] != 1 {
		t.Errorf("GetDDLStats() unique_index = %d, want 1", stats["unique_index"])
	}
	if stats["materialized"] != 1 {
		t.Errorf("GetDDLStats() materialized = %d, want 1", stats["materialized"])
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
