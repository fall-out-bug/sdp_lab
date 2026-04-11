package strataudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"sdp_dev/internal/strataudit/model"
)

type SQLiteStore struct {
	dbPath string
	db     *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &SQLiteStore{dbPath: dbPath, db: db}
	if err := s.pragma(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma: %w", err)
	}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) pragma() error {
	_, err := s.db.Exec(`PRAGMA foreign_keys = ON`)
	return err
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS levels (
		id TEXT PRIMARY KEY, name TEXT NOT NULL, rank INTEGER NOT NULL UNIQUE,
		description TEXT, patterns TEXT, config TEXT
	);
	CREATE TABLE IF NOT EXISTS documents (
		id TEXT PRIMARY KEY, path TEXT NOT NULL UNIQUE, level_id TEXT NOT NULL REFERENCES levels(id),
		content_hash TEXT NOT NULL, content TEXT NOT NULL, version INTEGER NOT NULL DEFAULT 1,
		file_modified_at DATETIME, metadata TEXT, ingested_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS entities (
		id TEXT PRIMARY KEY, document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
		level_id TEXT NOT NULL REFERENCES levels(id), type TEXT NOT NULL, title TEXT NOT NULL,
		description TEXT, source_quote TEXT, page_number INTEGER,
		embedding BLOB, embedding_model TEXT, embedding_dims INTEGER,
		extraction_model TEXT, metadata TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		CHECK (embedding IS NULL OR embedding_dims IS NOT NULL)
	);
	CREATE TABLE IF NOT EXISTS traces (
		id TEXT PRIMARY KEY, source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		relation TEXT NOT NULL, confidence REAL NOT NULL DEFAULT 0,
		justification TEXT, direction TEXT NOT NULL CHECK (direction IN ('up','down','bidirectional')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS trace_candidates (
		id TEXT PRIMARY KEY, source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		similarity REAL NOT NULL, verified BOOLEAN DEFAULT FALSE, trace_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS findings (
		id TEXT PRIMARY KEY, type TEXT NOT NULL CHECK (type IN ('alignment','strong_trace','coverage','gap','orphan','unknown_rationale','ambiguous_trace','conflict','weak_link','stale','inferred_strategy','shadow_strategy')),
		severity TEXT NOT NULL CHECK (severity IN ('info','warn','critical')),
		entity_ids TEXT, title TEXT NOT NULL, description TEXT, recommendation TEXT,
		suppressed BOOLEAN DEFAULT FALSE, llm_score TEXT,
		evidence_quotes TEXT, evidence_verified BOOLEAN DEFAULT FALSE, evidence_count INTEGER DEFAULT 0,
		support_ratio REAL DEFAULT 0, cross_model_status TEXT,
		verification_passed BOOLEAN, confidence_score REAL DEFAULT 0,
		ephemeral BOOLEAN DEFAULT FALSE, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS trace_coverage (
		id TEXT PRIMARY KEY, level_id TEXT NOT NULL REFERENCES levels(id),
		total_entities INTEGER DEFAULT 0, traced_entities INTEGER DEFAULT 0,
		coverage_pct REAL DEFAULT 0, computed_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS pipeline_state (
		id TEXT PRIMARY KEY, stage TEXT NOT NULL, status TEXT NOT NULL,
		checkpoint TEXT, error TEXT, started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS llm_invocations (
		id TEXT PRIMARY KEY, stage TEXT NOT NULL, model TEXT NOT NULL,
		prompt_hash TEXT NOT NULL, tokens_in INTEGER, tokens_out INTEGER,
		cost_usd REAL, duration_ms INTEGER, cached BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS llm_cache (
		prompt_hash TEXT PRIMARY KEY, model TEXT NOT NULL, response TEXT NOT NULL,
		tokens_in INTEGER, tokens_out INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_entities_level ON entities(level_id);
	CREATE INDEX IF NOT EXISTS idx_entities_document ON entities(document_id);
	CREATE INDEX IF NOT EXISTS idx_entities_type_level ON entities(type, level_id);
	CREATE INDEX IF NOT EXISTS idx_traces_source ON traces(source_entity_id);
	CREATE INDEX IF NOT EXISTS idx_traces_target ON traces(target_entity_id);
	CREATE INDEX IF NOT EXISTS idx_traces_direction ON traces(direction);
	CREATE INDEX IF NOT EXISTS idx_traces_relation_confidence ON traces(relation, confidence);
	CREATE INDEX IF NOT EXISTS idx_traces_confidence ON traces(confidence);
	CREATE INDEX IF NOT EXISTS idx_findings_type ON findings(type);
	CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);
	CREATE INDEX IF NOT EXISTS idx_findings_suppressed ON findings(suppressed) WHERE suppressed = FALSE;
	CREATE INDEX IF NOT EXISTS idx_documents_ingested ON documents(ingested_at);
	CREATE INDEX IF NOT EXISTS idx_trace_candidates_source ON trace_candidates(source_entity_id);
	CREATE INDEX IF NOT EXISTS idx_trace_candidates_verified ON trace_candidates(verified) WHERE verified = FALSE;
	CREATE INDEX IF NOT EXISTS idx_llm_invocations_stage ON llm_invocations(stage);
	CREATE INDEX IF NOT EXISTS idx_llm_cache_hash ON llm_cache(prompt_hash);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) SaveLevels(ctx context.Context, levels []model.Level) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, l := range levels {
		patterns, _ := json.Marshal(l.Patterns)
		_, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO levels (id, name, rank, description, patterns) VALUES (?,?,?,?,?)`,
			l.ID, l.Name, l.Rank, l.Description, string(patterns))
		if err != nil {
			return fmt.Errorf("save level %s: %w", l.ID, err)
		}
	}
	return tx.Commit()
}

func (s *SQLiteStore) LoadLevels(ctx context.Context) ([]model.Level, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, rank, description, patterns FROM levels ORDER BY rank`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var levels []model.Level
	for rows.Next() {
		var l model.Level
		var patternsJSON sql.NullString
		if err := rows.Scan(&l.ID, &l.Name, &l.Rank, &l.Description, &patternsJSON); err != nil {
			return nil, err
		}
		if patternsJSON.Valid {
			_ = json.Unmarshal([]byte(patternsJSON.String), &l.Patterns)
		}
		levels = append(levels, l)
	}
	return levels, rows.Err()
}

func (s *SQLiteStore) SaveDocuments(ctx context.Context, docs []model.Document) error {
	for _, d := range docs {
		meta, _ := json.Marshal(d.Metadata)
		ver := d.Version
		if ver == 0 {
			ver = 1
		}
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO documents (id, path, level_id, content_hash, content, version, file_modified_at, metadata, ingested_at)
			VALUES (?,?,?,?,?,?,?,?,datetime('now'))`,
			d.ID, d.Path, d.LevelID, d.ContentHash, d.Content, ver, d.FileModifiedAt, string(meta))
		if err != nil {
			return fmt.Errorf("save document %s: %w", d.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) SaveEntities(ctx context.Context, entities []model.Entity) error {
	for _, e := range entities {
		meta, _ := json.Marshal(e.Metadata)
		var embBlob interface{}
		if len(e.Embedding) > 0 {
			embData, _ := json.Marshal(e.Embedding)
			embBlob = embData
		}
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO entities (id, document_id, level_id, type, title, description, source_quote, page_number, embedding, embedding_model, embedding_dims, extraction_model, metadata)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			e.ID, e.DocumentID, e.LevelID, string(e.Type), e.Title, e.Description, e.SourceQuote, nilIfZero(e.PageNumber), embBlob, e.EmbeddingModel, nilIfZero(e.EmbeddingDims), e.ExtractionModel, string(meta))
		if err != nil {
			return fmt.Errorf("save entity %s: %w", e.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) DeleteEntitiesForDocument(ctx context.Context, docID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM entities WHERE document_id = ?`, docID)
	return err
}

func (s *SQLiteStore) EntitiesByLevel(ctx context.Context, levelID string, page model.Page) ([]model.Entity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, document_id, level_id, type, title, description, source_quote, extraction_model,
		embedding, embedding_model, embedding_dims
		FROM entities WHERE level_id = ? ORDER BY title LIMIT ? OFFSET ?`,
		levelID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanEntities(rows)
}

func (s *SQLiteStore) TracesForEntity(ctx context.Context, entityID string) ([]model.Trace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source_entity_id, target_entity_id, relation, confidence, justification, direction
		FROM traces WHERE source_entity_id = ? OR target_entity_id = ?`,
		entityID, entityID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var traces []model.Trace
	for rows.Next() {
		var t model.Trace
		if err := rows.Scan(&t.ID, &t.SourceEntityID, &t.TargetEntityID, &t.Relation, &t.Confidence, &t.Justification, &t.Direction); err != nil {
			return nil, err
		}
		traces = append(traces, t)
	}
	return traces, rows.Err()
}

func (s *SQLiteStore) AllTraces(ctx context.Context) ([]model.Trace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source_entity_id, target_entity_id, relation, confidence, justification, direction
		FROM traces ORDER BY confidence DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var traces []model.Trace
	for rows.Next() {
		var t model.Trace
		if err := rows.Scan(&t.ID, &t.SourceEntityID, &t.TargetEntityID, &t.Relation, &t.Confidence, &t.Justification, &t.Direction); err != nil {
			return nil, err
		}
		traces = append(traces, t)
	}
	return traces, rows.Err()
}

func (s *SQLiteStore) SaveTraces(ctx context.Context, traces []model.Trace) error {
	for _, t := range traces {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO traces (id, source_entity_id, target_entity_id, relation, confidence, justification, direction)
			VALUES (?,?,?,?,?,?,?)`,
			t.ID, t.SourceEntityID, t.TargetEntityID, string(t.Relation), t.Confidence, t.Justification, string(t.Direction))
		if err != nil {
			return fmt.Errorf("save trace %s: %w", t.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) SaveFindings(ctx context.Context, findings []model.Finding) error {
	for _, f := range findings {
		entityIDs, _ := json.Marshal(f.EntityIDs)
		evidenceQuotes, _ := json.Marshal(f.EvidenceQuotes)
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO findings (id, type, severity, entity_ids, title, description, recommendation,
			suppressed, llm_score, evidence_quotes, evidence_verified, evidence_count, support_ratio,
			cross_model_status, verification_passed, confidence_score, ephemeral)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			f.ID, string(f.Type), string(f.Severity), string(entityIDs), f.Title, f.Description, f.Recommendation,
			f.Suppressed, string(f.LLMScore), string(evidenceQuotes), f.EvidenceVerified, f.EvidenceCount, f.SupportRatio,
			string(f.CrossModelStatus), f.VerificationPassed, f.ConfidenceScore, f.Ephemeral)
		if err != nil {
			return fmt.Errorf("save finding %s: %w", f.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) FindingsByType(ctx context.Context, ft model.FindingType, page model.Page) ([]model.Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, severity, entity_ids, title, description, confidence_score, evidence_verified
		FROM findings WHERE type = ? LIMIT ? OFFSET ?`,
		string(ft), page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanFindings(rows)
}

func (s *SQLiteStore) SaveCoverage(ctx context.Context, coverages []model.Coverage) error {
	for _, c := range coverages {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO trace_coverage (id, level_id, total_entities, traced_entities, coverage_pct, computed_at)
			VALUES (?,?,?,?,?,datetime('now'))`, c.ID, c.LevelID, c.TotalEntities, c.TracedEntities, c.CoveragePct)
		if err != nil {
			return fmt.Errorf("save coverage %s: %w", c.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) CoverageByLevel(ctx context.Context) ([]model.Coverage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, level_id, total_entities, traced_entities, coverage_pct FROM trace_coverage`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var result []model.Coverage
	for rows.Next() {
		var c model.Coverage
		if err := rows.Scan(&c.ID, &c.LevelID, &c.TotalEntities, &c.TracedEntities, &c.CoveragePct); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) SavePipelineState(ctx context.Context, state model.PipelineState) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO pipeline_state (id, stage, status, checkpoint, error, started_at, completed_at)
		VALUES (?,?,?,?,?,?,?)`,
		state.ID, state.Stage, state.Status, state.Checkpoint, state.Error, state.StartedAt, state.CompletedAt)
	return err
}

func (s *SQLiteStore) LoadPipelineState(ctx context.Context, stage string) (*model.PipelineState, error) {
	var ps model.PipelineState
	err := s.db.QueryRowContext(ctx,
		`SELECT id, stage, status, checkpoint, error FROM pipeline_state WHERE stage = ? ORDER BY started_at DESC LIMIT 1`, stage).
		Scan(&ps.ID, &ps.Stage, &ps.Status, &ps.Checkpoint, &ps.Error)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ps, nil
}

func (s *SQLiteStore) CountEntitiesByLevel(ctx context.Context, levelID string) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entities WHERE level_id = ?`, levelID).Scan(&count)
	return count, err
}

func (s *SQLiteStore) DocumentByPath(ctx context.Context, path string) (*model.Document, error) {
	var d model.Document
	var meta sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT id, path, level_id, content_hash, content, version, metadata FROM documents WHERE path = ?`, path).
		Scan(&d.ID, &d.Path, &d.LevelID, &d.ContentHash, &d.Content, &d.Version, &meta)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if meta.Valid {
		_ = json.Unmarshal([]byte(meta.String), &d.Metadata)
	}
	return &d, nil
}

func nilIfZero(n int) interface{} {
	if n == 0 {
		return nil
	}
	return n
}

func scanEntities(rows *sql.Rows) ([]model.Entity, error) {
	var entities []model.Entity
	for rows.Next() {
		var e model.Entity
		var entityType string
		var embBlob []byte
		var embModel sql.NullString
		var embDims sql.NullInt64
		if err := rows.Scan(&e.ID, &e.DocumentID, &e.LevelID, &entityType, &e.Title, &e.Description, &e.SourceQuote, &e.ExtractionModel, &embBlob, &embModel, &embDims); err != nil {
			return nil, err
		}
		e.Type = model.EntityType(entityType)
		if len(embBlob) > 0 {
			_ = json.Unmarshal(embBlob, &e.Embedding)
		}
		if embModel.Valid {
			e.EmbeddingModel = embModel.String
		}
		if embDims.Valid {
			e.EmbeddingDims = int(embDims.Int64)
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

func scanFindings(rows *sql.Rows) ([]model.Finding, error) {
	var findings []model.Finding
	for rows.Next() {
		var f model.Finding
		var ftype, severity string
		var entityIDsJSON sql.NullString
		if err := rows.Scan(&f.ID, &ftype, &severity, &entityIDsJSON, &f.Title, &f.Description, &f.ConfidenceScore, &f.EvidenceVerified); err != nil {
			return nil, err
		}
		f.Type = model.FindingType(ftype)
		f.Severity = model.Severity(severity)
		if entityIDsJSON.Valid {
			_ = json.Unmarshal([]byte(entityIDsJSON.String), &f.EntityIDs)
		}
		findings = append(findings, f)
	}
	return findings, rows.Err()
}
