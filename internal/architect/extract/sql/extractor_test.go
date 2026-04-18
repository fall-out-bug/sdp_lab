// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/architect"
)

// TestSQLExtractor_Name tests the Name method.
func TestSQLExtractor_Name(t *testing.T) {
	extractor := NewSQLExtractor()
	if extractor.Name() != "sql" {
		t.Errorf("SQLExtractor.Name() = %s, want 'sql'", extractor.Name())
	}
}

// TestSQLExtractor_Extract_EmptyDir tests extraction from an empty directory.
func TestSQLExtractor_Extract_EmptyDir(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory
	tmpDir := t.TempDir()

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	if fragment == nil {
		t.Fatal("Extract() returned nil fragment")
	}

	if fragment.SQLAnalysis == nil {
		t.Fatal("Extract() returned nil SQLAnalysis")
	}
}

// TestSQLExtractor_Extract_SimpleSQL tests extraction from a simple SQL file.
func TestSQLExtractor_Extract_SimpleSQL(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with SQL file
	tmpDir := t.TempDir()
	sqlContent := `
		CREATE TABLE users (
			id INT PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			name VARCHAR(100)
		);

		CREATE TABLE posts (
			id INT PRIMARY KEY,
			user_id INT,
			title VARCHAR(255),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
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

	if fragment == nil || fragment.SQLAnalysis == nil {
		t.Fatal("Extract() returned nil SQLAnalysis")
	}

	analysis := fragment.SQLAnalysis

	// Check tables
	if len(analysis.Tables) != 2 {
		t.Errorf("Extract() found %d tables, want 2", len(analysis.Tables))
	}

	// Check foreign keys
	if len(analysis.ForeignKeys) != 1 {
		t.Errorf("Extract() found %d foreign keys, want 1", len(analysis.ForeignKeys))
	}

	// Check table names
	tableNames := make(map[string]bool)
	for _, table := range analysis.Tables {
		tableNames[table.Name] = true
	}

	if !tableNames["users"] {
		t.Error("Extract() did not find 'users' table")
	}

	if !tableNames["posts"] {
		t.Error("Extract() did not find 'posts' table")
	}
}

// TestSQLExtractor_Extract_IndexesViews tests extraction of indexes and views.
func TestSQLExtractor_Extract_IndexesViews(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with SQL file
	tmpDir := t.TempDir()
	sqlContent := `
		CREATE TABLE users (id INT PRIMARY KEY, email VARCHAR(255));

		CREATE INDEX idx_user_email ON users(email);
		CREATE UNIQUE INDEX idx_user_id ON users(id);

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

	analysis := fragment.SQLAnalysis

	// Check indexes
	if len(analysis.Indexes) != 2 {
		t.Errorf("Extract() found %d indexes, want 2", len(analysis.Indexes))
	}

	// Check views
	if len(analysis.Views) != 2 {
		t.Errorf("Extract() found %d views, want 2", len(analysis.Views))
	}

	// Check for materialized view
	var foundMaterialized bool
	for _, view := range analysis.Views {
		if view.Materialized {
			foundMaterialized = true
			break
		}
	}

	if !foundMaterialized {
		t.Error("Extract() did not find materialized view")
	}
}

// TestSQLExtractor_Extract_PII tests PII detection.
func TestSQLExtractor_Extract_PII(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with SQL file containing PII
	tmpDir := t.TempDir()
	sqlContent := `
		CREATE TABLE users (
			id INT PRIMARY KEY,
			email VARCHAR(255),
			phone VARCHAR(20),
			name VARCHAR(100)
		);

		CREATE TABLE customers (
			id INT PRIMARY KEY,
			shipping_address TEXT,
			birth_date DATE
		);
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

	analysis := fragment.SQLAnalysis

	// Check PII detection
	if len(analysis.PIIColumns) == 0 {
		t.Error("Extract() did not detect any PII columns")
	}

	// Check that specific PII types were detected
	piiTypes := make(map[string]bool)
	for _, pii := range analysis.PIIColumns {
		piiTypes[pii.PIIType] = true
	}

	expectedTypes := []string{"email_address", "phone_number", "physical_address", "date_of_birth"}
	for _, expectedType := range expectedTypes {
		if !piiTypes[expectedType] {
			t.Errorf("Extract() did not detect PII type: %s", expectedType)
		}
	}
}

// TestSQLExtractor_Extract_DataDomains tests domain clustering.
func TestSQLExtractor_Extract_DataDomains(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with SQL file
	tmpDir := t.TempDir()
	sqlContent := `
		CREATE TABLE users (id INT PRIMARY KEY, email VARCHAR(255));
		CREATE TABLE user_profiles (id INT PRIMARY KEY, user_id INT);
		CREATE TABLE posts (id INT PRIMARY KEY, user_id INT);
		CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(255));
		CREATE TABLE orders (id INT PRIMARY KEY, product_id INT);
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

	analysis := fragment.SQLAnalysis

	// Check domain detection
	if len(analysis.DataDomains) == 0 {
		t.Error("Extract() did not detect any data domains")
	}

	// Verify domains are non-empty
	for _, domain := range analysis.DataDomains {
		if len(domain.Tables) == 0 {
			t.Errorf("Data domain %s has no tables", domain.Name)
		}
		if domain.Name == "" {
			t.Error("Data domain has empty name")
		}
	}
}

// TestSQLExtractor_Extract_TestFixtureSkipping tests that test fixtures are skipped.
func TestSQLExtractor_Extract_TestFixtureSkipping(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with test fixtures
	tmpDir := t.TempDir()

	// Create test fixture directories
	fixturesDir := filepath.Join(tmpDir, "fixtures")
	testDataDir := filepath.Join(tmpDir, "testdata")

	if err := os.MkdirAll(fixturesDir, 0755); err != nil {
		t.Fatalf("Failed to create fixtures directory: %v", err)
	}
	if err := os.MkdirAll(testDataDir, 0755); err != nil {
		t.Fatalf("Failed to create testdata directory: %v", err)
	}

	// Write SQL files in various locations
	sqlContent := "CREATE TABLE test_data (id INT PRIMARY KEY);"

	// Normal SQL file (should be processed)
	normalPath := filepath.Join(tmpDir, "schema.sql")
	if err := os.WriteFile(normalPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write normal SQL file: %v", err)
	}

	// Fixture SQL file (should be skipped)
	fixturePath := filepath.Join(fixturesDir, "test.sql")
	if err := os.WriteFile(fixturePath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write fixture SQL file: %v", err)
	}

	// Testdata SQL file (should be skipped)
	testDataPath := filepath.Join(testDataDir, "test.sql")
	if err := os.WriteFile(testDataPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write testdata SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	analysis := fragment.SQLAnalysis

	// Should only find the normal SQL file
	if len(analysis.Tables) != 1 {
		t.Errorf("Extract() found %d tables, want 1 (test fixtures should be skipped)", len(analysis.Tables))
	}
}

// TestSQLExtractor_Extract_MigrationDetection tests migration directory detection.
func TestSQLExtractor_Extract_MigrationDetection(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with migrations
	tmpDir := t.TempDir()

	// Create migrations directory
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create migrations directory: %v", err)
	}

	// Create migration files
	migration1 := "20230101_initial.up.sql"
	migration2 := "20230102_add_users.up.sql"
	migration3 := "20230103_add_posts.up.sql"

	if err := os.WriteFile(filepath.Join(migrationsDir, migration1), []byte("-- migration"), 0644); err != nil {
		t.Fatalf("Failed to write migration file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, migration2), []byte("-- migration"), 0644); err != nil {
		t.Fatalf("Failed to write migration file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, migration3), []byte("-- migration"), 0644); err != nil {
		t.Fatalf("Failed to write migration file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	analysis := fragment.SQLAnalysis

	// Check migration detection
	if analysis.Migrations == nil {
		t.Error("Extract() did not detect migrations")
	} else {
		if analysis.Migrations.Count != 3 {
			t.Errorf("Extract() found %d migrations, want 3", analysis.Migrations.Count)
		}
		if analysis.Migrations.Dir != "migrations" {
			t.Errorf("Extract() migration dir = %s, want 'migrations'", analysis.Migrations.Dir)
		}
		if analysis.Migrations.Latest != migration3 {
			t.Errorf("Extract() latest migration = %s, want %s", analysis.Migrations.Latest, migration3)
		}
	}
}

// TestSQLExtractor_Extract_ORMDetection tests ORM framework detection.
func TestSQLExtractor_Extract_ORMDetection(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with ORM files
	tmpDir := t.TempDir()

	// Create GORM model file
	gormContent := `
package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email string ` + "`" + `gorm:"type:varchar(255)"` + "`" + `
	Name  string
}
`

	gormPath := filepath.Join(tmpDir, "user.go")
	if err := os.WriteFile(gormPath, []byte(gormContent), 0644); err != nil {
		t.Fatalf("Failed to write GORM file: %v", err)
	}

	// Create Prisma schema file
	prismaContent := `
model User {
  id    Int     @id
  email String
  posts Post[]
}

model Post {
  id       Int  @id
  title    String
  authorId Int
}
`

	prismaPath := filepath.Join(tmpDir, "schema.prisma")
	if err := os.WriteFile(prismaPath, []byte(prismaContent), 0644); err != nil {
		t.Fatalf("Failed to write Prisma file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	analysis := fragment.SQLAnalysis

	// Check ORM detection
	if len(analysis.ORMModels) == 0 {
		t.Error("Extract() did not detect any ORM models")
	}

	// Verify frameworks detected
	frameworks := make(map[string]bool)
	for _, model := range analysis.ORMModels {
		frameworks[model.Framework] = true
	}

	if !frameworks["gorm"] {
		t.Error("Extract() did not detect GORM framework")
	}

	if !frameworks["prisma"] {
		t.Error("Extract() did not detect Prisma framework")
	}

	// Check that model names were extracted
	var userModelFound bool
	for _, model := range analysis.ORMModels {
		if model.Framework == "prisma" && model.Model == "User" {
			userModelFound = true
			break
		}
	}

	if !userModelFound {
		t.Error("Extract() did not extract Prisma User model name")
	}
}

// TestSQLExtractor_Extract_MultipleSQLFiles tests extraction from multiple SQL files.
func TestSQLExtractor_Extract_MultipleSQLFiles(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with multiple SQL files
	tmpDir := t.TempDir()

	// Create multiple SQL files
	sqlFiles := map[string]string{
		"users.sql":     "CREATE TABLE users (id INT PRIMARY KEY, email VARCHAR(255));",
		"posts.sql":     "CREATE TABLE posts (id INT PRIMARY KEY, title VARCHAR(255));",
		"comments.sql":  "CREATE TABLE comments (id INT PRIMARY KEY, text TEXT);",
	}

	for filename, content := range sqlFiles {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write SQL file %s: %v", filename, err)
		}
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	analysis := fragment.SQLAnalysis

	// Check that all tables were found
	if len(analysis.Tables) != 3 {
		t.Errorf("Extract() found %d tables, want 3", len(analysis.Tables))
	}

	// Check table names
	tableNames := make(map[string]bool)
	for _, table := range analysis.Tables {
		tableNames[table.Name] = true
	}

	expectedTables := []string{"users", "posts", "comments"}
	for _, expected := range expectedTables {
		if !tableNames[expected] {
			t.Errorf("Extract() did not find table: %s", expected)
		}
	}
}

// TestSQLExtractor_Extract_ComplexSchema tests extraction from a complex schema.
func TestSQLExtractor_Extract_ComplexSchema(t *testing.T) {
	extractor := NewSQLExtractor()

	// Create temporary directory with complex SQL schema
	tmpDir := t.TempDir()
	sqlContent := `
-- Users and authentication
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    birth_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User profiles
CREATE TABLE user_profiles (
    id INT PRIMARY KEY,
    user_id INT NOT NULL,
    bio TEXT,
    avatar_url VARCHAR(500),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Posts and content
CREATE TABLE posts (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT,
    published BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Comments
CREATE TABLE comments (
    id INT PRIMARY KEY AUTO_INCREMENT,
    post_id INT NOT NULL,
    user_id INT NOT NULL,
    parent_id INT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE UNIQUE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_posts_user ON posts(user_id);
CREATE INDEX idx_posts_published ON posts(user_id, published);
CREATE INDEX idx_comments_post ON comments(post_id);
CREATE INDEX idx_comments_user ON comments(user_id);

-- Views
CREATE VIEW user_post_counts AS
    SELECT u.id, u.email, COUNT(p.id) as post_count
    FROM users u
    LEFT JOIN posts p ON u.id = p.user_id
    GROUP BY u.id;

CREATE MATERIALIZED VIEW popular_posts AS
    SELECT p.*, COUNT(c.id) as comment_count
    FROM posts p
    LEFT JOIN comments c ON p.id = c.post_id
    GROUP BY p.id
    HAVING COUNT(c.id) > 10;

-- Functions
CREATE FUNCTION get_user_post_count(user_id INT) RETURNS INT
DETERMINISTIC
READS SQL DATA
BEGIN
    DECLARE post_count INT;
    SELECT COUNT(*) INTO post_count FROM posts WHERE user_id = user_id;
    RETURN post_count;
END;
`

	sqlPath := filepath.Join(tmpDir, "complex_schema.sql")
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0644); err != nil {
		t.Fatalf("Failed to write SQL file: %v", err)
	}

	ctx := context.Background()
	fragment, err := extractor.Extract(ctx, tmpDir)

	if err != nil {
		t.Fatalf("Extract() returned error: %v", err)
	}

	analysis := fragment.SQLAnalysis

	// Verify comprehensive extraction
	if len(analysis.Tables) != 4 {
		t.Errorf("Extract() found %d tables, want 4", len(analysis.Tables))
	}

	if len(analysis.ForeignKeys) != 4 {
		t.Errorf("Extract() found %d foreign keys, want 4", len(analysis.ForeignKeys))
	}

	if len(analysis.Indexes) != 5 {
		t.Errorf("Extract() found %d indexes, want 5", len(analysis.Indexes))
	}

	if len(analysis.Views) != 2 {
		t.Errorf("Extract() found %d views, want 2", len(analysis.Views))
	}

	if len(analysis.StoredProcs) != 1 {
		t.Errorf("Extract() found %d stored procedures, want 1", len(analysis.StoredProcs))
	}

	// Check PII detection
	if len(analysis.PIIColumns) == 0 {
		t.Error("Extract() did not detect PII columns in complex schema")
	}

	// Check data domain clustering
	if len(analysis.DataDomains) == 0 {
		t.Error("Extract() did not cluster tables into data domains")
	}
}
