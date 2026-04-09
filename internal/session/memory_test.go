package session

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestLoadMemoryContextDisabled(t *testing.T) {
	root := t.TempDir()
	if err := initGitRepo(t, root, "git@github.com:org/repo.git"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SDP_MEMORY_ENABLED", "false")

	ctx, err := LoadMemoryContext(root)
	if err != nil {
		t.Fatalf("LoadMemoryContext: %v", err)
	}
	if ctx != "" {
		t.Fatalf("expected empty context when disabled, got %q", ctx)
	}
}

func TestLoadMemoryContextNoData(t *testing.T) {
	root := t.TempDir()
	if err := initGitRepo(t, root, "git@github.com:org/repo.git"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SDP_MEMORY_ENABLED", "true")

	ctx, err := LoadMemoryContext(root)
	if err != nil {
		t.Fatalf("LoadMemoryContext: %v", err)
	}
	if ctx != "" {
		t.Fatalf("expected empty context with no storage, got %q", ctx)
	}
}

func TestLoadMemoryContextWithScopedData(t *testing.T) {
	root := t.TempDir()
	repoURL := "git@github.com:org/repo.git"
	if err := initGitRepo(t, root, repoURL); err != nil {
		t.Fatal(err)
	}
	storage := filepath.Join(root, ".sdp", "mem")
	if err := os.MkdirAll(filepath.Join(storage, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := seedMetadataDB(filepath.Join(storage, "metadata.db")); err != nil {
		t.Fatal(err)
	}
	if err := seedProjectShardDB(filepath.Join(storage, "projects", "project_abc_shard_0.db"), repoURL); err != nil {
		t.Fatal(err)
	}
	if err := seedUserProfileDB(filepath.Join(storage, "user-profiles.db")); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SDP_MEMORY_ENABLED", "true")

	ctx, err := LoadMemoryContext(root)
	if err != nil {
		t.Fatalf("LoadMemoryContext: %v", err)
	}
	if !strings.Contains(ctx, "[MEMORY]") {
		t.Fatalf("expected memory header, got %q", ctx)
	}
	if !strings.Contains(ctx, "Project Memories:") {
		t.Fatalf("expected project memories section, got %q", ctx)
	}
	if !strings.Contains(ctx, "relevant scoped memory") {
		t.Fatalf("expected scoped memory content, got %q", ctx)
	}
	if strings.Contains(ctx, "other repo memory") {
		t.Fatalf("unexpected memory from other repo, got %q", ctx)
	}
	if !strings.Contains(ctx, "User Profile:") {
		t.Fatalf("expected user profile section, got %q", ctx)
	}
	if !strings.Contains(ctx, "prefers concise commit messages") {
		t.Fatalf("expected preference item, got %q", ctx)
	}
}

func initGitRepo(t *testing.T, dir, remote string) error {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapErr("git init", out, err)
	}
	cmd = exec.Command("git", "remote", "add", "origin", remote)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		return wrapErr("git remote add", out, err)
	}
	return nil
}

func seedMetadataDB(path string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope TEXT NOT NULL,
			scope_hash TEXT NOT NULL,
			shard_index INTEGER NOT NULL,
			db_path TEXT NOT NULL,
			vector_count INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			created_at INTEGER NOT NULL
		)
	`); err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO shards(scope, scope_hash, shard_index, db_path, vector_count, is_active, created_at)
		VALUES('project', 'abc', 0, 'projects/project_abc_shard_0.db', 2, 1, 1700000000000)
	`)
	return err
}

func seedProjectShardDB(path, repoURL string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			vector BLOB,
			tags_vector BLOB,
			container_tag TEXT,
			tags TEXT,
			type TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			metadata TEXT,
			display_name TEXT,
			user_name TEXT,
			user_email TEXT,
			project_path TEXT,
			project_name TEXT,
			git_repo_url TEXT,
			is_pinned INTEGER DEFAULT 0
		)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO memories(id, content, created_at, updated_at, git_repo_url)
		VALUES('m1', 'relevant scoped memory', 1700000001000, 1700000001000, ?)
	`, repoURL); err != nil {
		return err
	}
	_, err = db.Exec(`
		INSERT INTO memories(id, content, created_at, updated_at, git_repo_url)
		VALUES('m2', 'other repo memory', 1700000002000, 1700000002000, 'git@github.com:other/repo.git')
	`)
	return err
}

func seedUserProfileDB(path string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_profiles (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			user_name TEXT NOT NULL,
			user_email TEXT NOT NULL,
			profile_data TEXT NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			last_analyzed_at INTEGER NOT NULL,
			total_prompts_analyzed INTEGER NOT NULL DEFAULT 0,
			is_active BOOLEAN NOT NULL DEFAULT 1
		)
	`); err != nil {
		return err
	}

	profileJSON := `{"preferences":[{"category":"style","description":"prefers concise commit messages","confidence":0.91}]}`
	_, err = db.Exec(`
		INSERT INTO user_profiles(id, user_id, display_name, user_name, user_email, profile_data, version, created_at, last_analyzed_at, total_prompts_analyzed, is_active)
		VALUES('p1', 'u1', 'Dev User', 'dev', 'dev@example.com', ?, 1, 1700000000000, 1700000000001, 10, 1)
	`, profileJSON)
	return err
}

func wrapErr(step string, out []byte, err error) error {
	if len(out) == 0 {
		return err
	}
	return fmt.Errorf("%s: %s: %w", step, strings.TrimSpace(string(out)), err)
}
