package index

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// schemaSQL is the SQLite schema for the index database.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL,
    symbol_name TEXT,
    kind TEXT NOT NULL,
    scope TEXT,
    language TEXT NOT NULL,
    line_start INTEGER NOT NULL,
    line_end INTEGER NOT NULL,
    content TEXT NOT NULL,
    description TEXT,
    pagerank REAL DEFAULT 0,
    hash TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chunks_file ON chunks(file_path);
CREATE INDEX IF NOT EXISTS idx_chunks_kind ON chunks(kind);
CREATE INDEX IF NOT EXISTS idx_chunks_symbol ON chunks(symbol_name);

-- Full-text index for keyword/navigational search
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    file_path, symbol_name, content, scope,
    content='chunks', content_rowid='id'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
    INSERT INTO chunks_fts(rowid, file_path, symbol_name, content, scope)
    VALUES (new.id, new.file_path, new.symbol_name, new.content, new.scope);
END;

CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, file_path, symbol_name, content, scope)
    VALUES ('delete', old.id, old.file_path, old.symbol_name, old.content, old.scope);
END;

CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, file_path, symbol_name, content, scope)
    VALUES ('delete', old.id, old.file_path, old.symbol_name, old.content, old.scope);
    INSERT INTO chunks_fts(rowid, file_path, symbol_name, content, scope)
    VALUES (new.id, new.file_path, new.symbol_name, new.content, new.scope);
END;

-- Structural edges (dependency graph)
CREATE TABLE IF NOT EXISTS edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    target_id INTEGER NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
    relation TEXT NOT NULL,
    weight REAL DEFAULT 1.0
);

CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);

-- File metadata for incremental indexing
CREATE TABLE IF NOT EXISTS files (
    path TEXT PRIMARY KEY,
    hash TEXT NOT NULL,
    last_indexed TEXT NOT NULL,
    language TEXT,
    loc INTEGER,
    is_test BOOLEAN DEFAULT FALSE,
    is_generated BOOLEAN DEFAULT FALSE
);

-- Module metadata (aggregated)
CREATE TABLE IF NOT EXISTS modules (
    name TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    purpose TEXT,
    owner TEXT,
    bus_factor INTEGER DEFAULT 0,
    files_count INTEGER DEFAULT 0,
    loc INTEGER DEFAULT 0,
    is_hotspot BOOLEAN DEFAULT FALSE
);

-- Index metadata (key-value store)
CREATE TABLE IF NOT EXISTS meta (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- Schema version marker
INSERT OR IGNORE INTO meta (key, value) VALUES ('schema_version', '%d');
`

// SQLiteStore wraps a SQLite database for the index.
// It implements the Store interface used by manifest and enrichment.
type SQLiteStore struct {
	db   *sql.DB
	path string
}

// OpenStore creates or opens an index database at the given path.
// It creates the directory and schema if they do not exist.
func OpenStore(dbPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode and foreign keys
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	// Create schema
	finalSQL := fmt.Sprintf(schemaSQL, SchemaVersion)
	if _, err := db.Exec(finalSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &SQLiteStore{db: db, path: dbPath}, nil
}

// Close performs a WAL checkpoint and closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	if _, err := s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		s.db.Close()
		return fmt.Errorf("wal checkpoint: %w", err)
	}
	return s.db.Close()
}

// DBPath returns the filesystem path to the database file.
func (s *SQLiteStore) DBPath() string {
	return s.path
}

// Begin starts a new database transaction.
func (s *SQLiteStore) Begin() (*sql.Tx, error) {
	return s.db.Begin()
}

// CommitTx commits a transaction and returns any error encountered.
func CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx rolls back a transaction. Safe to call in defer blocks even
// after commit (rollback of a committed tx is a no-op in most drivers,
// but the error is silently discarded regardless).
func RollbackTx(tx *sql.Tx) {
	_ = tx.Rollback()
}

// SetMeta stores a key-value pair in the meta table.
func (s *SQLiteStore) SetMeta(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
	return err
}

// GetMeta retrieves a value from the meta table. Returns empty string if not found.
func (s *SQLiteStore) GetMeta(key string) (string, error) {
	var val string
	err := s.db.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// InsertChunk inserts a chunk and returns its ID.
func (s *SQLiteStore) InsertChunk(c Chunk) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO chunks (file_path, symbol_name, kind, scope, language,
			line_start, line_end, content, description, pagerank, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.FilePath, c.SymbolName, c.Kind, c.Scope, c.Language,
		c.LineStart, c.LineEnd, c.Content, c.Description, c.PageRank, c.Hash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetChunk retrieves a chunk by ID.
func (s *SQLiteStore) GetChunk(id int64) (*Chunk, error) {
	c := &Chunk{}
	err := s.db.QueryRow(`
		SELECT id, file_path, symbol_name, kind, scope, language,
			line_start, line_end, content, description, pagerank, hash
		FROM chunks WHERE id = ?`, id,
	).Scan(&c.ID, &c.FilePath, &c.SymbolName, &c.Kind, &c.Scope, &c.Language,
		&c.LineStart, &c.LineEnd, &c.Content, &c.Description, &c.PageRank, &c.Hash)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// InsertEdge inserts a structural edge and returns its ID.
func (s *SQLiteStore) InsertEdge(e Edge) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO edges (source_id, target_id, relation, weight)
		VALUES (?, ?, ?, ?)`, e.SourceID, e.TargetID, e.Relation, e.Weight)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpsertFileMeta inserts or updates file metadata.
func (s *SQLiteStore) UpsertFileMeta(fm FileMeta) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO files (path, hash, last_indexed, language, loc, is_test, is_generated)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fm.Path, fm.Hash, fm.LastIndexed, fm.Language, fm.Loc, fm.IsTest, fm.IsGenerated)
	return err
}

// GetFileMeta retrieves file metadata by path.
func (s *SQLiteStore) GetFileMeta(path string) (*FileMeta, error) {
	fm := &FileMeta{}
	err := s.db.QueryRow(`
		SELECT path, hash, last_indexed, language, loc, is_test, is_generated
		FROM files WHERE path = ?`, path,
	).Scan(&fm.Path, &fm.Hash, &fm.LastIndexed, &fm.Language, &fm.Loc, &fm.IsTest, &fm.IsGenerated)
	if err != nil {
		return nil, err
	}
	return fm, nil
}

// DeleteFileMeta removes file metadata by path.
func (s *SQLiteStore) DeleteFileMeta(path string) error {
	_, err := s.db.Exec("DELETE FROM files WHERE path = ?", path)
	return err
}

// UpsertModuleMeta inserts or updates module metadata.
func (s *SQLiteStore) UpsertModuleMeta(mm ModuleMeta) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO modules (name, path, purpose, owner, bus_factor, files_count, loc, is_hotspot)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		mm.Name, mm.Path, mm.Purpose, mm.Owner, mm.BusFactor, mm.FilesCount, mm.Loc, mm.IsHotspot)
	return err
}

// GetModuleMeta retrieves module metadata by name.
func (s *SQLiteStore) GetModuleMeta(name string) (*ModuleMeta, error) {
	mm := &ModuleMeta{}
	err := s.db.QueryRow(`
		SELECT name, path, purpose, owner, bus_factor, files_count, loc, is_hotspot
		FROM modules WHERE name = ?`, name,
	).Scan(&mm.Name, &mm.Path, &mm.Purpose, &mm.Owner, &mm.BusFactor, &mm.FilesCount, &mm.Loc, &mm.IsHotspot)
	if err != nil {
		return nil, err
	}
	return mm, nil
}

// DeleteChunksByFile removes all chunks for a given file path.
// Returns the number of chunks deleted.
func (s *SQLiteStore) DeleteChunksByFile(filePath string) (int, error) {
	res, err := s.db.Exec("DELETE FROM chunks WHERE file_path = ?", filePath)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Stats returns summary statistics about the index.
func (s *SQLiteStore) Stats() (*IndexStats, error) {
	stats := &IndexStats{}

	err := s.db.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&stats.TotalChunks)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&stats.TotalFiles)
	if err != nil {
		return nil, err
	}
	err = s.db.QueryRow("SELECT COUNT(*) FROM edges").Scan(&stats.TotalEdges)
	if err != nil {
		return nil, err
	}

	// Get distinct languages
	rows, err := s.db.Query("SELECT DISTINCT language FROM chunks WHERE language != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	langSet := map[string]bool{}
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err != nil {
			return nil, err
		}
		langSet[lang] = true
	}
	for lang := range langSet {
		stats.Languages = append(stats.Languages, lang)
	}
	sort.Strings(stats.Languages)

	return stats, nil
}

// EnsureSdpDir creates the .sdp directory under repoPath and returns the
// path to index.db inside it.
func EnsureSdpDir(repoPath string) (string, error) {
	sdpDir := filepath.Join(repoPath, ".sdp")
	if err := os.MkdirAll(sdpDir, 0o755); err != nil {
		return "", fmt.Errorf("create .sdp dir: %w", err)
	}
	return filepath.Join(sdpDir, "index.db"), nil
}

// IsBinaryFile returns true if the file content appears to be binary.
// Uses the same heuristic as git: a NUL byte in the first 8KB means binary.
func IsBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true
	}
	defer f.Close()

	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil {
		return true
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}

// IsSecretFile returns true if the file path looks like a secrets file.
// Uses exact extension matching and targeted patterns to avoid false positives
// on legitimate source files (e.g., tokenizer.py, token_handler.go).
func IsSecretFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))

	// Exact extension match for certificate/key files
	secretExts := map[string]bool{
		".pem":        true,
		".key":        true,
		".p12":        true,
		".pfx":        true,
		".jks":        true,
		".keystore":   true,
		".netrc":      true,
		".kubeconfig": true,
	}

	// Check exact extension match
	ext := filepath.Ext(base)
	if secretExts[ext] {
		return true
	}

	// Special files without extensions (exact basename match)
	secretFiles := map[string]bool{
		"id_rsa":     true,
		"id_rsa.pub": true,
		"id_ed25519": true,
		"id_ed25519.pub": true,
		"id_dsa":     true,
		"id_ecdsa":   true,
		"id_ecdsa_sk": true,
	}
	if secretFiles[base] {
		return true
	}

	// Exact match for .env files (including .env.local, .env.production, etc.)
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}

	// Targeted patterns: must match specific formats, not substrings
	// These patterns are more specific to reduce false positives
	secretPatterns := []struct {
		prefix        string
		suffix        string
		requireExt    string // if set, require this specific extension
	}{
		{"credentials.", "", ""},  // credentials.json, credentials.yaml
		{"", ".credentials", ""},  // service.credentials
		{"secret.", "", ""},       // secret.env, secret.config (but NOT secret.go)
		{"", ".secret", ""},       // config.secret, db.secret
		{"password.", "", ""},     // password.txt, password.file (but NOT password.go)
		{"", ".password", ""},     // config.password
		{"private.", "", ""},      // private.key (already covered by .key but for completeness)
		{"", ".private", ""},      // config.private
	}

	// Extensions that are commonly used for secrets/config files
	// Source code extensions (.go, .py, .js, .rs, etc.) should NOT be treated as secrets
	secretConfigExtensions := map[string]bool{
		".env":        true,
		".config":     true,
		".conf":       true,
		".txt":        true,
		".json":       true,
		".yaml":       true,
		".yml":        true,
		".xml":        true,
		".ini":        true,
		".cfg":        true,
		".properties": true,
		".file":       true, // password.file
	}

	for _, pat := range secretPatterns {
		if (pat.prefix == "" || strings.HasPrefix(base, pat.prefix)) &&
		   (pat.suffix == "" || strings.HasSuffix(base, pat.suffix)) {
			// If both prefix and suffix specified, that's a strong signal
			if pat.prefix != "" && pat.suffix != "" {
				return true
			}
			// If only prefix specified, require a secret/config extension (not source code)
			if pat.prefix != "" && len(base) > len(pat.prefix) {
				ext := filepath.Ext(base)
				if secretConfigExtensions[ext] {
					return true
				}
			}
			// If only suffix specified, that's a strong signal
			if pat.suffix != "" && len(base) > len(pat.suffix) {
				return true
			}
		}
	}

	return false
}

// --- ManifestStore interface methods ---

// LoadModules returns all modules from the modules table.
func (s *SQLiteStore) LoadModules() ([]ModuleMeta, error) {
	rows, err := s.db.Query(`
		SELECT name, path, purpose, owner, bus_factor, files_count, loc, is_hotspot
		FROM modules ORDER BY loc DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var modules []ModuleMeta
	for rows.Next() {
		var m ModuleMeta
		if err := rows.Scan(&m.Name, &m.Path, &m.Purpose, &m.Owner,
			&m.BusFactor, &m.FilesCount, &m.Loc, &m.IsHotspot); err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}
	return modules, nil
}

// LoadMeta retrieves multiple meta values by key. Returns a map with found keys.
func (s *SQLiteStore) LoadMeta(keys ...string) (map[string]string, error) {
	result := make(map[string]string)
	for _, k := range keys {
		val, err := s.GetMeta(k)
		if err != nil {
			return nil, err
		}
		if val != "" {
			result[k] = val
		}
	}
	return result, nil
}

// LoadMetaPrefix retrieves all meta entries whose key starts with prefix.
func (s *SQLiteStore) LoadMetaPrefix(prefix string) (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM meta WHERE key LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, nil
}

// UpdateModules replaces all modules with the given slice.
func (s *SQLiteStore) UpdateModules(modules []ModuleMeta) error {
	if _, err := s.db.Exec("DELETE FROM modules"); err != nil {
		return err
	}
	for _, m := range modules {
		if err := s.UpsertModuleMeta(m); err != nil {
			return err
		}
	}
	return nil
}

// LoadEntryPoints returns file paths containing main functions or CLI entry points.
func (s *SQLiteStore) LoadEntryPoints() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT file_path FROM chunks
		WHERE symbol_name = 'main' AND kind = 'function'
		UNION
		SELECT DISTINCT path FROM files
		WHERE path LIKE 'cmd/%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// LoadStats returns index statistics for manifest generation.
func (s *SQLiteStore) LoadStats() (*IndexStats, error) {
	stats, err := s.Stats()
	if err != nil {
		return nil, err
	}
	repoName, _ := s.GetMeta("repo_name")
	stats.RepoName = repoName
	return stats, nil
}

// ListIndexedFilePaths returns all file paths currently tracked in the files table.
func (s *SQLiteStore) ListIndexedFilePaths() ([]string, error) {
	rows, err := s.db.Query("SELECT path FROM files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// LoadFileHashMap returns a map of file_path -> hash for all indexed files.
// This replaces separate ListIndexedFilePaths + GetFileMeta calls to avoid N+1 queries.
func (s *SQLiteStore) LoadFileHashMap() (map[string]string, error) {
	rows, err := s.db.Query("SELECT path, hash FROM files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var p, h string
		if err := rows.Scan(&p, &h); err != nil {
			continue
		}
		result[p] = h
	}
	return result, nil
}

// LoadChunksByIDs retrieves multiple chunks by their IDs in a single query.
// Returns a map of chunk ID -> Chunk. Missing IDs are silently omitted.
func (s *SQLiteStore) LoadChunksByIDs(ids []int64) (map[int64]*Chunk, error) {
	if len(ids) == 0 {
		return map[int64]*Chunk{}, nil
	}

	// Build parameterized IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT id, file_path, symbol_name, kind, scope, language,
			line_start, line_end, content, description, pagerank, hash
		FROM chunks WHERE id IN (%s)`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]*Chunk, len(ids))
	for rows.Next() {
		c := &Chunk{}
		if err := rows.Scan(&c.ID, &c.FilePath, &c.SymbolName, &c.Kind, &c.Scope,
			&c.Language, &c.LineStart, &c.LineEnd, &c.Content,
			&c.Description, &c.PageRank, &c.Hash); err != nil {
			continue
		}
		result[c.ID] = c
	}
	return result, nil
}

// CountChunks returns the total number of chunks in the index.
func (s *SQLiteStore) CountChunks() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM chunks").Scan(&count)
	return count, err
}

// CountFiles returns the total number of files in the files table.
func (s *SQLiteStore) CountFiles() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM files").Scan(&count)
	return count, err
}

// InsertChunkTx inserts a chunk within an existing transaction.
func InsertChunkTx(tx *sql.Tx, c Chunk) (int64, error) {
	res, err := tx.Exec(`
		INSERT INTO chunks (file_path, symbol_name, kind, scope, language,
			line_start, line_end, content, description, pagerank, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.FilePath, c.SymbolName, c.Kind, c.Scope, c.Language,
		c.LineStart, c.LineEnd, c.Content, c.Description, c.PageRank, c.Hash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertEdgeTx inserts an edge within an existing transaction.
func InsertEdgeTx(tx *sql.Tx, e Edge) (int64, error) {
	res, err := tx.Exec(`
		INSERT INTO edges (source_id, target_id, relation, weight)
		VALUES (?, ?, ?, ?)`, e.SourceID, e.TargetID, e.Relation, e.Weight)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpsertFileMetaTx inserts or updates file metadata within an existing transaction.
func UpsertFileMetaTx(tx *sql.Tx, fm FileMeta) error {
	_, err := tx.Exec(`
		INSERT OR REPLACE INTO files (path, hash, last_indexed, language, loc, is_test, is_generated)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fm.Path, fm.Hash, fm.LastIndexed, fm.Language, fm.Loc, fm.IsTest, fm.IsGenerated)
	return err
}

// SetMetaTx stores a key-value pair in the meta table within an existing transaction.
func SetMetaTx(tx *sql.Tx, key, value string) error {
	_, err := tx.Exec("INSERT OR REPLACE INTO meta (key, value) VALUES (?, ?)", key, value)
	return err
}

// DeleteChunksByFileTx removes all chunks for a given file path within a transaction.
func DeleteChunksByFileTx(tx *sql.Tx, filePath string) (int, error) {
	res, err := tx.Exec("DELETE FROM chunks WHERE file_path = ?", filePath)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// DeleteFileMetaTx removes file metadata by path within a transaction.
func DeleteFileMetaTx(tx *sql.Tx, path string) error {
	_, err := tx.Exec("DELETE FROM files WHERE path = ?", path)
	return err
}

// BuildSymbolIDMap queries all chunks and returns a map from "file_path:symbol_name" to chunk ID.
// This is used for resolving symbolic edges to ID-based edges after all chunks are inserted.
func BuildSymbolIDMap(store *SQLiteStore) (map[string]int64, error) {
	rows, err := store.db.Query("SELECT id, file_path, symbol_name FROM chunks WHERE symbol_name IS NOT NULL AND symbol_name != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]int64)
	for rows.Next() {
		var id int64
		var fp, sym string
		if err := rows.Scan(&id, &fp, &sym); err != nil {
			continue
		}
		key := fp + ":" + sym
		m[key] = id
	}
	return m, nil
}

// buildSymbolIDMapTx queries all chunks within a transaction and returns a map from "file_path:symbol_name" to chunk ID.
// This is the transactional version used during ColdBuild for consistency.
func buildSymbolIDMapTx(tx *sql.Tx) (map[string]int64, error) {
	rows, err := tx.Query("SELECT id, file_path, symbol_name FROM chunks WHERE symbol_name IS NOT NULL AND symbol_name != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[string]int64)
	for rows.Next() {
		var id int64
		var fp, sym string
		if err := rows.Scan(&id, &fp, &sym); err != nil {
			continue
		}
		key := fp + ":" + sym
		m[key] = id
	}
	return m, nil
}
