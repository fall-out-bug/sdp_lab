// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// TestDDLParser_SimpleTable tests parsing a simple CREATE TABLE statement.
func TestDDLParser_SimpleTable(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255));"
	sqlPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.Tables) != 1 {
		t.Errorf("Extract() found %d tables, want 1", len(fragment.SQLAnalysis.Tables))
	}

	table := fragment.SQLAnalysis.Tables[0]
	if table.Name != "users" {
		t.Errorf("Table name = %s, want 'users'", table.Name)
	}

	if len(table.Columns) != 2 {
		t.Errorf("Table has %d columns, want 2", len(table.Columns))
	}
}

// TestDDLParser_TableWithSchema tests parsing CREATE TABLE with schema prefix.
func TestDDLParser_TableWithSchema(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := "CREATE TABLE public.users (id INT PRIMARY KEY, email VARCHAR(255));"
	sqlPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.Tables) != 1 {
		t.Errorf("Extract() found %d tables, want 1", len(fragment.SQLAnalysis.Tables))
	}

	table := fragment.SQLAnalysis.Tables[0]
	if table.Schema != "public" {
		t.Errorf("Table schema = %s, want 'public'", table.Schema)
	}
}

// TestDDLParser_TableWithForeignKey tests parsing CREATE TABLE with FOREIGN KEY.
func TestDDLParser_TableWithForeignKey(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := `CREATE TABLE posts (
		id INT PRIMARY KEY,
		user_id INT,
		title VARCHAR(255),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	sqlPath := filepath.Join(tmpDir, "posts.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.ForeignKeys) != 1 {
		t.Errorf("Extract() found %d foreign keys, want 1", len(fragment.SQLAnalysis.ForeignKeys))
	}

	fk := fragment.SQLAnalysis.ForeignKeys[0]
	if fk.FromTable != "posts" {
		t.Errorf("FK from table = %s, want 'posts'", fk.FromTable)
	}
	if fk.FromColumn != "user_id" {
		t.Errorf("FK from column = %s, want 'user_id'", fk.FromColumn)
	}
	if fk.ToTable != "users" {
		t.Errorf("FK to table = %s, want 'users'", fk.ToTable)
	}
	if fk.ToColumn != "id" {
		t.Errorf("FK to column = %s, want 'id'", fk.ToColumn)
	}
}

// TestDDLParser_ColumnAttributes tests parsing of column attributes.
func TestDDLParser_ColumnAttributes(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := `CREATE TABLE users (
		id INT PRIMARY KEY,
		email VARCHAR(255) NOT NULL,
		name VARCHAR(100),
		age INT
	);`

	sqlPath := filepath.Join(tmpDir, "users.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	table := fragment.SQLAnalysis.Tables[0]

	// Find columns by name
	columns := make(map[string]architect.Column)
	for _, col := range table.Columns {
		columns[col.Name] = col
	}

	// Check id column (PRIMARY KEY)
	idCol, ok := columns["id"]
	if !ok {
		t.Fatal("id column not found")
	}
	if !idCol.PrimaryKey {
		t.Error("id column should have PrimaryKey = true")
	}
	if !idCol.NotNull {
		t.Error("id column should have NotNull = true")
	}

	// Check email column (NOT NULL)
	emailCol, ok := columns["email"]
	if !ok {
		t.Fatal("email column not found")
	}
	if !emailCol.NotNull {
		t.Error("email column should have NotNull = true")
	}

	// Check name column (nullable)
	nameCol, ok := columns["name"]
	if !ok {
		t.Fatal("name column not found")
	}
	if !nameCol.Nullable {
		t.Error("name column should have Nullable = true")
	}
}

// TestDDLParser_Indexes tests parsing of CREATE INDEX statements.
func TestDDLParser_Indexes(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := `
		CREATE TABLE users (id INT PRIMARY KEY, email VARCHAR(255));
		CREATE INDEX idx_user_email ON users(email);
		CREATE UNIQUE INDEX idx_user_id ON users(id);
	`

	sqlPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.Indexes) != 2 {
		t.Errorf("Extract() found %d indexes, want 2", len(fragment.SQLAnalysis.Indexes))
	}

	// Check for unique index
	var uniqueIndexFound bool
	for _, idx := range fragment.SQLAnalysis.Indexes {
		if idx.Unique {
			uniqueIndexFound = true
			break
		}
	}

	if !uniqueIndexFound {
		t.Error("Extract() did not find unique index")
	}
}

// TestDDLParser_Views tests parsing of CREATE VIEW statements.
func TestDDLParser_Views(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := `
		CREATE TABLE users (id INT PRIMARY KEY, email VARCHAR(255));
		CREATE VIEW user_emails AS SELECT email FROM users;
		CREATE MATERIALIZED VIEW user_stats AS SELECT COUNT(*) FROM users;
	`

	sqlPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.Views) != 2 {
		t.Errorf("Extract() found %d views, want 2", len(fragment.SQLAnalysis.Views))
	}

	// Check for materialized view
	var materializedViewFound bool
	for _, view := range fragment.SQLAnalysis.Views {
		if view.Materialized {
			materializedViewFound = true
			break
		}
	}

	if !materializedViewFound {
		t.Error("Extract() did not find materialized view")
	}
}

// TestDDLParser_StoredProcs tests parsing of CREATE FUNCTION/PROCEDURE statements.
func TestDDLParser_StoredProcs(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := `
		CREATE FUNCTION get_user_count() RETURNS INT AS $$ SELECT COUNT(*) FROM users; $$ LANGUAGE SQL;
		CREATE PROCEDURE update_user_stats() AS BEGIN UPDATE user_stats SET count = (SELECT COUNT(*) FROM users); END;
	`

	sqlPath := filepath.Join(tmpDir, "procs.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.StoredProcs) != 2 {
		t.Errorf("Extract() found %d stored procedures, want 2", len(fragment.SQLAnalysis.StoredProcs))
	}
}

// TestDDLParser_CompositePrimaryKey tests parsing of composite primary keys.
func TestDDLParser_CompositePrimaryKey(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := `CREATE TABLE order_items (
		order_id INT,
		item_id INT,
		quantity INT,
		PRIMARY KEY (order_id, item_id)
	);`

	sqlPath := filepath.Join(tmpDir, "order_items.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	table := fragment.SQLAnalysis.Tables[0]

	// Find primary key columns
	var pkColumns []string
	for _, col := range table.Columns {
		if col.PrimaryKey {
			pkColumns = append(pkColumns, col.Name)
		}
	}

	if len(pkColumns) != 2 {
		t.Errorf("Table has %d primary key columns, want 2", len(pkColumns))
	}
}

// TestDDLParser_MultipleForeignKeys tests parsing of multiple foreign keys.
func TestDDLParser_MultipleForeignKeys(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := `CREATE TABLE order_items (
		order_id INT,
		product_id INT,
		quantity INT,
		FOREIGN KEY (order_id) REFERENCES orders(id),
		FOREIGN KEY (product_id) REFERENCES products(id)
	);`

	sqlPath := filepath.Join(tmpDir, "order_items.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.ForeignKeys) != 2 {
		t.Errorf("Extract() found %d foreign keys, want 2", len(fragment.SQLAnalysis.ForeignKeys))
	}
}

// TestDDLParser_QuotedIdentifiers tests parsing of quoted identifiers.
func TestDDLParser_QuotedIdentifiers(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := "CREATE TABLE `orders` (id INT PRIMARY KEY, total DECIMAL(10,2));"
	sqlPath := filepath.Join(tmpDir, "orders.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.Tables) != 1 {
		t.Errorf("Extract() found %d tables, want 1", len(fragment.SQLAnalysis.Tables))
	}

	table := fragment.SQLAnalysis.Tables[0]
	if table.Name != "orders" {
		t.Errorf("Table name = %s, want 'orders'", table.Name)
	}
}

// TestDDLParser_IfNotExists tests parsing of CREATE TABLE IF NOT EXISTS.
func TestDDLParser_IfNotExists(t *testing.T) {
	extractor := NewSQLExtractor()
	tmpDir := t.TempDir()

	sqlContent := "CREATE TABLE IF NOT EXISTS comments (id INT PRIMARY KEY, text TEXT);"
	sqlPath := filepath.Join(tmpDir, "comments.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if len(fragment.SQLAnalysis.Tables) != 1 {
		t.Errorf("Extract() found %d tables, want 1", len(fragment.SQLAnalysis.Tables))
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
