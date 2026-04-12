package strataudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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
	CREATE TABLE IF NOT EXISTS sections (
		id TEXT PRIMARY KEY, document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL, heading TEXT, char_start INTEGER NOT NULL, char_end INTEGER NOT NULL,
		preview TEXT NOT NULL, content TEXT NOT NULL, content_hash TEXT NOT NULL, quality_flags TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(document_id, ordinal)
	);
	CREATE TABLE IF NOT EXISTS entities (
		id TEXT PRIMARY KEY, document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
		level_id TEXT NOT NULL REFERENCES levels(id), section_id TEXT, type TEXT NOT NULL, title TEXT NOT NULL,
		description TEXT, title_original TEXT, description_original TEXT, source_quote TEXT,
		quote_start_offset INTEGER, quote_end_offset INTEGER,
		lang TEXT, language_mismatch BOOLEAN DEFAULT FALSE, trust_grade TEXT NOT NULL DEFAULT 'verified', quality_flags TEXT, page_number INTEGER,
		embedding BLOB, embedding_model TEXT, embedding_dims INTEGER,
		extraction_model TEXT, metadata TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		CHECK (embedding IS NULL OR embedding_dims IS NOT NULL)
	);
	CREATE TABLE IF NOT EXISTS traces (
		id TEXT PRIMARY KEY, source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		relation TEXT NOT NULL, confidence REAL NOT NULL DEFAULT 0, similarity_score REAL NOT NULL DEFAULT 0,
		justification TEXT, direction TEXT NOT NULL CHECK (direction IN ('up','down','bidirectional')),
		verification_mode TEXT, trust_grade TEXT NOT NULL DEFAULT 'verified',
		source_section_id TEXT, target_section_id TEXT,
		source_quote_start_offset INTEGER, source_quote_end_offset INTEGER,
		target_quote_start_offset INTEGER, target_quote_end_offset INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS trace_candidates (
		id TEXT PRIMARY KEY, source_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		target_entity_id TEXT NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
		similarity REAL NOT NULL, verified BOOLEAN DEFAULT FALSE, trace_id TEXT, diagnostic_code TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS findings (
		id TEXT PRIMARY KEY, type TEXT NOT NULL CHECK (type IN ('alignment','strong_trace','coverage','gap','orphan','strategic_gap_cluster','orphan_cluster','corpus_quality_cluster','trace_ambiguity_cluster','unknown_rationale','ambiguous_trace','conflict','weak_link','stale','inferred_strategy','shadow_strategy')),
		severity TEXT NOT NULL CHECK (severity IN ('info','warn','critical')),
		entity_ids TEXT, document_ids TEXT, section_ids TEXT, cluster_key TEXT, title TEXT NOT NULL, description TEXT, recommendation TEXT,
		suppressed BOOLEAN DEFAULT FALSE, llm_score TEXT,
		evidence_quotes TEXT, evidence_verified BOOLEAN DEFAULT FALSE, evidence_count INTEGER DEFAULT 0,
		support_ratio REAL DEFAULT 0, cross_model_status TEXT,
		verification_passed BOOLEAN, confidence_score REAL DEFAULT 0,
		ephemeral BOOLEAN DEFAULT FALSE, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS trace_coverage (
		id TEXT PRIMARY KEY, scope_type TEXT NOT NULL DEFAULT 'level', scope_id TEXT, scope_label TEXT,
		level_id TEXT NOT NULL REFERENCES levels(id), document_id TEXT, section_id TEXT,
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
	CREATE INDEX IF NOT EXISTS idx_entities_section ON entities(section_id);
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
	CREATE INDEX IF NOT EXISTS idx_sections_document ON sections(document_id);
	CREATE INDEX IF NOT EXISTS idx_sections_document_ordinal ON sections(document_id, ordinal);
	CREATE INDEX IF NOT EXISTS idx_trace_candidates_source ON trace_candidates(source_entity_id);
	CREATE INDEX IF NOT EXISTS idx_trace_candidates_verified ON trace_candidates(verified) WHERE verified = FALSE;
	CREATE INDEX IF NOT EXISTS idx_llm_invocations_stage ON llm_invocations(stage);
	CREATE INDEX IF NOT EXISTS idx_llm_cache_hash ON llm_cache(prompt_hash);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	if err := s.ensureEntityTrustColumns(); err != nil {
		return err
	}
	if err := s.ensureEntityLanguageColumns(); err != nil {
		return err
	}
	if err := s.ensureSectionsTable(); err != nil {
		return err
	}
	if err := s.ensureEntityProvenanceColumns(); err != nil {
		return err
	}
	if err := s.ensureTraceColumns(); err != nil {
		return err
	}
	if err := s.ensureCandidateColumns(); err != nil {
		return err
	}
	if err := s.ensureCoverageColumns(); err != nil {
		return err
	}
	if err := s.ensureFindingsSchema(); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) ensureSectionsTable() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS sections (
		id TEXT PRIMARY KEY, document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
		ordinal INTEGER NOT NULL, heading TEXT, char_start INTEGER NOT NULL, char_end INTEGER NOT NULL,
		preview TEXT NOT NULL, content TEXT NOT NULL, content_hash TEXT NOT NULL, quality_flags TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(document_id, ordinal)
	);
	CREATE INDEX IF NOT EXISTS idx_sections_document ON sections(document_id);
	CREATE INDEX IF NOT EXISTS idx_sections_document_ordinal ON sections(document_id, ordinal);
	`)
	if err != nil {
		return fmt.Errorf("ensure sections table: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ensureEntityTrustColumns() error {
	hasTrustGrade := false
	hasQualityFlags := false

	rows, err := s.db.Query(`PRAGMA table_info(entities)`)
	if err != nil {
		return fmt.Errorf("pragma table_info(entities): %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table_info(entities): %w", err)
		}
		switch name {
		case "trust_grade":
			hasTrustGrade = true
		case "quality_flags":
			hasQualityFlags = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info(entities): %w", err)
	}

	if !hasTrustGrade {
		if _, err := s.db.Exec(`ALTER TABLE entities ADD COLUMN trust_grade TEXT NOT NULL DEFAULT 'verified'`); err != nil {
			return fmt.Errorf("add entities.trust_grade: %w", err)
		}
	}
	if !hasQualityFlags {
		if _, err := s.db.Exec(`ALTER TABLE entities ADD COLUMN quality_flags TEXT`); err != nil {
			return fmt.Errorf("add entities.quality_flags: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) ensureEntityLanguageColumns() error {
	requiredColumns := map[string]string{
		"title_original":       `ALTER TABLE entities ADD COLUMN title_original TEXT`,
		"description_original": `ALTER TABLE entities ADD COLUMN description_original TEXT`,
		"lang":                 `ALTER TABLE entities ADD COLUMN lang TEXT`,
		"language_mismatch":    `ALTER TABLE entities ADD COLUMN language_mismatch BOOLEAN DEFAULT FALSE`,
	}

	present := make(map[string]bool, len(requiredColumns))
	rows, err := s.db.Query(`PRAGMA table_info(entities)`)
	if err != nil {
		return fmt.Errorf("pragma table_info(entities): %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table_info(entities): %w", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info(entities): %w", err)
	}

	for column, stmt := range requiredColumns {
		if present[column] {
			continue
		}
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("add entities.%s: %w", column, err)
		}
	}
	return nil
}

func (s *SQLiteStore) ensureEntityProvenanceColumns() error {
	requiredColumns := map[string]string{
		"section_id":         `ALTER TABLE entities ADD COLUMN section_id TEXT`,
		"quote_start_offset": `ALTER TABLE entities ADD COLUMN quote_start_offset INTEGER`,
		"quote_end_offset":   `ALTER TABLE entities ADD COLUMN quote_end_offset INTEGER`,
	}

	present := make(map[string]bool, len(requiredColumns))
	rows, err := s.db.Query(`PRAGMA table_info(entities)`)
	if err != nil {
		return fmt.Errorf("pragma table_info(entities): %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table_info(entities): %w", err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info(entities): %w", err)
	}

	for column, stmt := range requiredColumns {
		if present[column] {
			continue
		}
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("add entities.%s: %w", column, err)
		}
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_entities_section ON entities(section_id)`); err != nil {
		return fmt.Errorf("ensure idx_entities_section: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ensureTraceColumns() error {
	requiredColumns := map[string]string{
		"similarity_score":          `ALTER TABLE traces ADD COLUMN similarity_score REAL NOT NULL DEFAULT 0`,
		"verification_mode":         `ALTER TABLE traces ADD COLUMN verification_mode TEXT`,
		"trust_grade":               `ALTER TABLE traces ADD COLUMN trust_grade TEXT NOT NULL DEFAULT 'verified'`,
		"source_section_id":         `ALTER TABLE traces ADD COLUMN source_section_id TEXT`,
		"target_section_id":         `ALTER TABLE traces ADD COLUMN target_section_id TEXT`,
		"source_quote_start_offset": `ALTER TABLE traces ADD COLUMN source_quote_start_offset INTEGER`,
		"source_quote_end_offset":   `ALTER TABLE traces ADD COLUMN source_quote_end_offset INTEGER`,
		"target_quote_start_offset": `ALTER TABLE traces ADD COLUMN target_quote_start_offset INTEGER`,
		"target_quote_end_offset":   `ALTER TABLE traces ADD COLUMN target_quote_end_offset INTEGER`,
	}
	return s.ensureTableColumns("traces", requiredColumns)
}

func (s *SQLiteStore) ensureCandidateColumns() error {
	requiredColumns := map[string]string{
		"diagnostic_code": `ALTER TABLE trace_candidates ADD COLUMN diagnostic_code TEXT`,
	}
	return s.ensureTableColumns("trace_candidates", requiredColumns)
}

func (s *SQLiteStore) ensureCoverageColumns() error {
	requiredColumns := map[string]string{
		"scope_type":  `ALTER TABLE trace_coverage ADD COLUMN scope_type TEXT NOT NULL DEFAULT 'level'`,
		"scope_id":    `ALTER TABLE trace_coverage ADD COLUMN scope_id TEXT`,
		"scope_label": `ALTER TABLE trace_coverage ADD COLUMN scope_label TEXT`,
		"document_id": `ALTER TABLE trace_coverage ADD COLUMN document_id TEXT`,
		"section_id":  `ALTER TABLE trace_coverage ADD COLUMN section_id TEXT`,
	}
	return s.ensureTableColumns("trace_coverage", requiredColumns)
}

func (s *SQLiteStore) ensureFindingsSchema() error {
	var createSQL string
	err := s.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'findings'`).Scan(&createSQL)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load findings schema: %w", err)
	}

	if strings.Contains(createSQL, "strategic_gap_cluster") &&
		strings.Contains(createSQL, "document_ids") &&
		strings.Contains(createSQL, "section_ids") &&
		strings.Contains(createSQL, "cluster_key") {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin findings schema migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		CREATE TABLE findings_new (
			id TEXT PRIMARY KEY, type TEXT NOT NULL CHECK (type IN ('alignment','strong_trace','coverage','gap','orphan','strategic_gap_cluster','orphan_cluster','corpus_quality_cluster','trace_ambiguity_cluster','unknown_rationale','ambiguous_trace','conflict','weak_link','stale','inferred_strategy','shadow_strategy')),
			severity TEXT NOT NULL CHECK (severity IN ('info','warn','critical')),
			entity_ids TEXT, document_ids TEXT, section_ids TEXT, cluster_key TEXT, title TEXT NOT NULL, description TEXT, recommendation TEXT,
			suppressed BOOLEAN DEFAULT FALSE, llm_score TEXT,
			evidence_quotes TEXT, evidence_verified BOOLEAN DEFAULT FALSE, evidence_count INTEGER DEFAULT 0,
			support_ratio REAL DEFAULT 0, cross_model_status TEXT,
			verification_passed BOOLEAN, confidence_score REAL DEFAULT 0,
			ephemeral BOOLEAN DEFAULT FALSE, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return fmt.Errorf("create findings_new: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO findings_new (
			id, type, severity, entity_ids, document_ids, section_ids, cluster_key, title, description, recommendation,
			suppressed, llm_score, evidence_quotes, evidence_verified, evidence_count, support_ratio,
			cross_model_status, verification_passed, confidence_score, ephemeral, created_at
		)
		SELECT
			id, type, severity, entity_ids, NULL, NULL, NULL, title, description, recommendation,
			suppressed, llm_score, evidence_quotes, evidence_verified, evidence_count, support_ratio,
			cross_model_status, verification_passed, confidence_score, ephemeral, created_at
		FROM findings
	`); err != nil {
		return fmt.Errorf("copy findings to findings_new: %w", err)
	}
	if _, err := tx.Exec(`DROP TABLE findings`); err != nil {
		return fmt.Errorf("drop findings: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE findings_new RENAME TO findings`); err != nil {
		return fmt.Errorf("rename findings_new: %w", err)
	}
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_findings_type ON findings(type);
		CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);
		CREATE INDEX IF NOT EXISTS idx_findings_suppressed ON findings(suppressed) WHERE suppressed = FALSE;
	`); err != nil {
		return fmt.Errorf("recreate findings indexes: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit findings schema migration: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ensureTableColumns(table string, requiredColumns map[string]string) error {
	present := make(map[string]bool, len(requiredColumns))
	rows, err := s.db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return fmt.Errorf("pragma table_info(%s): %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table_info(%s): %w", table, err)
		}
		present[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table_info(%s): %w", table, err)
	}

	for column, stmt := range requiredColumns {
		if present[column] {
			continue
		}
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("add %s.%s: %w", table, column, err)
		}
	}
	return nil
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

func (s *SQLiteStore) SaveSections(ctx context.Context, sections []model.Section) error {
	for _, section := range sections {
		qualityFlags, _ := json.Marshal(section.QualityFlags)
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO sections (id, document_id, ordinal, heading, char_start, char_end, preview, content, content_hash, quality_flags)
			VALUES (?,?,?,?,?,?,?,?,?,?)`,
			section.ID, section.DocumentID, section.Ordinal, nullableString(section.Heading), section.CharStart, section.CharEnd, section.Preview, section.Content, section.ContentHash, string(qualityFlags))
		if err != nil {
			return fmt.Errorf("save section %s: %w", section.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) SaveEntities(ctx context.Context, entities []model.Entity) error {
	for _, e := range entities {
		meta, _ := json.Marshal(e.Metadata)
		qualityFlags, _ := json.Marshal(e.QualityFlags)
		var embBlob interface{}
		if len(e.Embedding) > 0 {
			embData, _ := json.Marshal(e.Embedding)
			embBlob = embData
		}
		trustGrade := string(e.TrustGrade)
		if trustGrade == "" {
			trustGrade = string(model.TrustGradeVerified)
		}
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO entities (id, document_id, level_id, section_id, type, title, description, title_original, description_original, source_quote, quote_start_offset, quote_end_offset, lang, language_mismatch, trust_grade, quality_flags, page_number, embedding, embedding_model, embedding_dims, extraction_model, metadata)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			e.ID, e.DocumentID, e.LevelID, nullableString(e.SectionID), string(e.Type), e.Title, e.Description, nullableString(e.TitleOriginal), nullableString(e.DescriptionOriginal), e.SourceQuote, nullableIntPtr(e.QuoteStartOffset), nullableIntPtr(e.QuoteEndOffset), nullableString(e.Lang), e.LanguageMismatch, trustGrade, string(qualityFlags), nilIfZero(e.PageNumber), embBlob, e.EmbeddingModel, nilIfZero(e.EmbeddingDims), e.ExtractionModel, string(meta))
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

func (s *SQLiteStore) DeleteSectionsForDocument(ctx context.Context, docID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sections WHERE document_id = ?`, docID)
	return err
}

func (s *SQLiteStore) EntitiesByLevel(ctx context.Context, levelID string, page model.Page) ([]model.Entity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, document_id, level_id, section_id, type, title, description, title_original, description_original, source_quote, quote_start_offset, quote_end_offset, lang, language_mismatch, trust_grade, quality_flags, extraction_model,
		embedding, embedding_model, embedding_dims
		FROM entities WHERE level_id = ? ORDER BY title LIMIT ? OFFSET ?`,
		levelID, page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanEntities(rows)
}

func (s *SQLiteStore) SectionsByDocument(ctx context.Context, documentID string) ([]model.Section, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, document_id, ordinal, heading, char_start, char_end, preview, content, content_hash, quality_flags
		FROM sections WHERE document_id = ? ORDER BY ordinal`, documentID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSections(rows)
}

func (s *SQLiteStore) TracesForEntity(ctx context.Context, entityID string) ([]model.Trace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source_entity_id, target_entity_id, relation, confidence, similarity_score, justification, direction, verification_mode, trust_grade,
		source_section_id, target_section_id, source_quote_start_offset, source_quote_end_offset, target_quote_start_offset, target_quote_end_offset
		FROM traces WHERE source_entity_id = ? OR target_entity_id = ?`,
		entityID, entityID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanTraces(rows)
}

func (s *SQLiteStore) AllTraces(ctx context.Context) ([]model.Trace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source_entity_id, target_entity_id, relation, confidence, similarity_score, justification, direction, verification_mode, trust_grade,
		source_section_id, target_section_id, source_quote_start_offset, source_quote_end_offset, target_quote_start_offset, target_quote_end_offset
		FROM traces ORDER BY confidence DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanTraces(rows)
}

func (s *SQLiteStore) AllCandidates(ctx context.Context) ([]model.Candidate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source_entity_id, target_entity_id, similarity, verified, trace_id, diagnostic_code
		FROM trace_candidates ORDER BY similarity DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanCandidates(rows)
}

func (s *SQLiteStore) SaveTraces(ctx context.Context, traces []model.Trace) error {
	for _, t := range traces {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO traces (id, source_entity_id, target_entity_id, relation, confidence, similarity_score, justification, direction, verification_mode, trust_grade,
			source_section_id, target_section_id, source_quote_start_offset, source_quote_end_offset, target_quote_start_offset, target_quote_end_offset)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			t.ID, t.SourceEntityID, t.TargetEntityID, string(t.Relation), t.Confidence, t.SimilarityScore, t.Justification, string(t.Direction), nullableString(string(t.VerificationMode)), nullableString(string(t.TrustGrade)),
			nullableString(t.SourceSectionID), nullableString(t.TargetSectionID), nullableIntPtr(t.SourceQuoteStartOffset), nullableIntPtr(t.SourceQuoteEndOffset), nullableIntPtr(t.TargetQuoteStartOffset), nullableIntPtr(t.TargetQuoteEndOffset))
		if err != nil {
			return fmt.Errorf("save trace %s: %w", t.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) SaveFindings(ctx context.Context, findings []model.Finding) error {
	for _, f := range findings {
		entityIDs, _ := json.Marshal(f.EntityIDs)
		documentIDs, _ := json.Marshal(f.DocumentIDs)
		sectionIDs, _ := json.Marshal(f.SectionIDs)
		evidenceQuotes, _ := json.Marshal(f.EvidenceQuotes)
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO findings (id, type, severity, entity_ids, document_ids, section_ids, cluster_key, title, description, recommendation,
			suppressed, llm_score, evidence_quotes, evidence_verified, evidence_count, support_ratio,
			cross_model_status, verification_passed, confidence_score, ephemeral)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			f.ID, string(f.Type), string(f.Severity), string(entityIDs), string(documentIDs), string(sectionIDs), nullableString(f.ClusterKey), f.Title, f.Description, f.Recommendation,
			f.Suppressed, string(f.LLMScore), string(evidenceQuotes), f.EvidenceVerified, f.EvidenceCount, f.SupportRatio,
			string(f.CrossModelStatus), f.VerificationPassed, f.ConfidenceScore, f.Ephemeral)
		if err != nil {
			return fmt.Errorf("save finding %s: %w", f.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) ClearFindings(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM findings`)
	return err
}

func (s *SQLiteStore) FindingsByType(ctx context.Context, ft model.FindingType, page model.Page) ([]model.Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, severity, entity_ids, document_ids, section_ids, cluster_key, title, description, confidence_score, evidence_verified
		FROM findings WHERE type = ? LIMIT ? OFFSET ?`,
		string(ft), page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanFindings(rows)
}

func (s *SQLiteStore) AllFindings(ctx context.Context, page model.Page) ([]model.Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, severity, entity_ids, document_ids, section_ids, cluster_key, title, description, confidence_score, evidence_verified
		FROM findings ORDER BY
			CASE severity WHEN 'critical' THEN 0 WHEN 'warn' THEN 1 ELSE 2 END,
			confidence_score DESC,
			title ASC
		LIMIT ? OFFSET ?`,
		page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanFindings(rows)
}

func (s *SQLiteStore) SaveCoverage(ctx context.Context, coverages []model.Coverage) error {
	for _, c := range coverages {
		scopeType := c.ScopeType
		if scopeType == "" {
			scopeType = model.CoverageScopeLevel
		}
		scopeID := c.ScopeID
		if scopeID == "" {
			scopeID = firstNonEmpty(c.SectionID, c.DocumentID, c.LevelID)
		}
		scopeLabel := c.ScopeLabel
		if scopeLabel == "" {
			scopeLabel = scopeID
		}
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO trace_coverage (id, scope_type, scope_id, scope_label, level_id, document_id, section_id, total_entities, traced_entities, coverage_pct, computed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,datetime('now'))`, c.ID, string(scopeType), scopeID, scopeLabel, c.LevelID, nullableString(c.DocumentID), nullableString(c.SectionID), c.TotalEntities, c.TracedEntities, c.CoveragePct)
		if err != nil {
			return fmt.Errorf("save coverage %s: %w", c.ID, err)
		}
	}
	return nil
}

func (s *SQLiteStore) ClearCoverage(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM trace_coverage`)
	return err
}

func (s *SQLiteStore) CoverageByLevel(ctx context.Context) ([]model.Coverage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scope_type, scope_id, scope_label, level_id, document_id, section_id, total_entities, traced_entities, coverage_pct
		FROM trace_coverage WHERE scope_type = 'level' ORDER BY level_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanCoverage(rows)
}

func (s *SQLiteStore) AllCoverage(ctx context.Context) ([]model.Coverage, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scope_type, scope_id, scope_label, level_id, document_id, section_id, total_entities, traced_entities, coverage_pct
		FROM trace_coverage ORDER BY
			CASE scope_type WHEN 'level' THEN 0 WHEN 'document' THEN 1 ELSE 2 END,
			coverage_pct ASC,
			scope_label ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanCoverage(rows)
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

func (s *SQLiteStore) AllDocuments(ctx context.Context) ([]model.Document, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path, level_id, content_hash, version, metadata FROM documents ORDER BY path`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var docs []model.Document
	for rows.Next() {
		var doc model.Document
		var meta sql.NullString
		if err := rows.Scan(&doc.ID, &doc.Path, &doc.LevelID, &doc.ContentHash, &doc.Version, &meta); err != nil {
			return nil, err
		}
		if meta.Valid {
			_ = json.Unmarshal([]byte(meta.String), &doc.Metadata)
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (s *SQLiteStore) AllSections(ctx context.Context) ([]model.Section, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, document_id, ordinal, heading, char_start, char_end, preview, content, content_hash, quality_flags
		FROM sections ORDER BY document_id, ordinal`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanSections(rows)
}

func nilIfZero(n int) interface{} {
	if n == 0 {
		return nil
	}
	return n
}

func nullableIntPtr(n *int) interface{} {
	if n == nil {
		return nil
	}
	return *n
}

func nullableString(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func scanEntities(rows *sql.Rows) ([]model.Entity, error) {
	var entities []model.Entity
	for rows.Next() {
		var e model.Entity
		var entityType string
		var sectionID sql.NullString
		var titleOriginal sql.NullString
		var descriptionOriginal sql.NullString
		var quoteStart sql.NullInt64
		var quoteEnd sql.NullInt64
		var lang sql.NullString
		var languageMismatch bool
		var trustGrade sql.NullString
		var qualityFlagsJSON sql.NullString
		var embBlob []byte
		var embModel sql.NullString
		var embDims sql.NullInt64
		if err := rows.Scan(&e.ID, &e.DocumentID, &e.LevelID, &sectionID, &entityType, &e.Title, &e.Description, &titleOriginal, &descriptionOriginal, &e.SourceQuote, &quoteStart, &quoteEnd, &lang, &languageMismatch, &trustGrade, &qualityFlagsJSON, &e.ExtractionModel, &embBlob, &embModel, &embDims); err != nil {
			return nil, err
		}
		e.Type = model.EntityType(entityType)
		if sectionID.Valid {
			e.SectionID = sectionID.String
		}
		if titleOriginal.Valid {
			e.TitleOriginal = titleOriginal.String
		}
		if descriptionOriginal.Valid {
			e.DescriptionOriginal = descriptionOriginal.String
		}
		if quoteStart.Valid {
			offset := int(quoteStart.Int64)
			e.QuoteStartOffset = &offset
		}
		if quoteEnd.Valid {
			offset := int(quoteEnd.Int64)
			e.QuoteEndOffset = &offset
		}
		if lang.Valid {
			e.Lang = lang.String
		}
		e.LanguageMismatch = languageMismatch
		if trustGrade.Valid {
			e.TrustGrade = model.TrustGrade(trustGrade.String)
		} else {
			e.TrustGrade = model.TrustGradeVerified
		}
		if qualityFlagsJSON.Valid && qualityFlagsJSON.String != "" {
			_ = json.Unmarshal([]byte(qualityFlagsJSON.String), &e.QualityFlags)
		}
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

func scanSections(rows *sql.Rows) ([]model.Section, error) {
	var sections []model.Section
	for rows.Next() {
		var section model.Section
		var heading sql.NullString
		var qualityFlagsJSON sql.NullString
		if err := rows.Scan(&section.ID, &section.DocumentID, &section.Ordinal, &heading, &section.CharStart, &section.CharEnd, &section.Preview, &section.Content, &section.ContentHash, &qualityFlagsJSON); err != nil {
			return nil, err
		}
		if heading.Valid {
			section.Heading = heading.String
		}
		if qualityFlagsJSON.Valid && qualityFlagsJSON.String != "" {
			_ = json.Unmarshal([]byte(qualityFlagsJSON.String), &section.QualityFlags)
		}
		sections = append(sections, section)
	}
	return sections, rows.Err()
}

func scanTraces(rows *sql.Rows) ([]model.Trace, error) {
	var traces []model.Trace
	for rows.Next() {
		var trace model.Trace
		var relation string
		var direction string
		var verificationMode sql.NullString
		var trustGrade sql.NullString
		var sourceSectionID sql.NullString
		var targetSectionID sql.NullString
		var sourceQuoteStart sql.NullInt64
		var sourceQuoteEnd sql.NullInt64
		var targetQuoteStart sql.NullInt64
		var targetQuoteEnd sql.NullInt64
		if err := rows.Scan(
			&trace.ID,
			&trace.SourceEntityID,
			&trace.TargetEntityID,
			&relation,
			&trace.Confidence,
			&trace.SimilarityScore,
			&trace.Justification,
			&direction,
			&verificationMode,
			&trustGrade,
			&sourceSectionID,
			&targetSectionID,
			&sourceQuoteStart,
			&sourceQuoteEnd,
			&targetQuoteStart,
			&targetQuoteEnd,
		); err != nil {
			return nil, err
		}
		trace.Relation = model.TraceRelation(relation)
		trace.Direction = model.TraceDirection(direction)
		if verificationMode.Valid {
			trace.VerificationMode = model.TraceVerificationMode(verificationMode.String)
		}
		if trustGrade.Valid {
			trace.TrustGrade = model.TrustGrade(trustGrade.String)
		} else {
			trace.TrustGrade = model.TrustGradeVerified
		}
		if sourceSectionID.Valid {
			trace.SourceSectionID = sourceSectionID.String
		}
		if targetSectionID.Valid {
			trace.TargetSectionID = targetSectionID.String
		}
		if sourceQuoteStart.Valid {
			offset := int(sourceQuoteStart.Int64)
			trace.SourceQuoteStartOffset = &offset
		}
		if sourceQuoteEnd.Valid {
			offset := int(sourceQuoteEnd.Int64)
			trace.SourceQuoteEndOffset = &offset
		}
		if targetQuoteStart.Valid {
			offset := int(targetQuoteStart.Int64)
			trace.TargetQuoteStartOffset = &offset
		}
		if targetQuoteEnd.Valid {
			offset := int(targetQuoteEnd.Int64)
			trace.TargetQuoteEndOffset = &offset
		}
		traces = append(traces, trace)
	}
	return traces, rows.Err()
}

func scanCandidates(rows *sql.Rows) ([]model.Candidate, error) {
	var candidates []model.Candidate
	for rows.Next() {
		var candidate model.Candidate
		var traceID sql.NullString
		var diagnosticCode sql.NullString
		if err := rows.Scan(&candidate.ID, &candidate.SourceEntityID, &candidate.TargetEntityID, &candidate.Similarity, &candidate.Verified, &traceID, &diagnosticCode); err != nil {
			return nil, err
		}
		if traceID.Valid {
			candidate.TraceID = traceID.String
		}
		if diagnosticCode.Valid {
			candidate.DiagnosticCode = diagnosticCode.String
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func scanFindings(rows *sql.Rows) ([]model.Finding, error) {
	var findings []model.Finding
	for rows.Next() {
		var f model.Finding
		var ftype, severity string
		var entityIDsJSON sql.NullString
		var documentIDsJSON sql.NullString
		var sectionIDsJSON sql.NullString
		var clusterKey sql.NullString
		if err := rows.Scan(&f.ID, &ftype, &severity, &entityIDsJSON, &documentIDsJSON, &sectionIDsJSON, &clusterKey, &f.Title, &f.Description, &f.ConfidenceScore, &f.EvidenceVerified); err != nil {
			return nil, err
		}
		f.Type = model.FindingType(ftype)
		f.Severity = model.Severity(severity)
		if entityIDsJSON.Valid {
			_ = json.Unmarshal([]byte(entityIDsJSON.String), &f.EntityIDs)
		}
		if documentIDsJSON.Valid {
			_ = json.Unmarshal([]byte(documentIDsJSON.String), &f.DocumentIDs)
		}
		if sectionIDsJSON.Valid {
			_ = json.Unmarshal([]byte(sectionIDsJSON.String), &f.SectionIDs)
		}
		if clusterKey.Valid {
			f.ClusterKey = clusterKey.String
		}
		findings = append(findings, f)
	}
	return findings, rows.Err()
}

func scanCoverage(rows *sql.Rows) ([]model.Coverage, error) {
	var result []model.Coverage
	for rows.Next() {
		var c model.Coverage
		var scopeType string
		var scopeID sql.NullString
		var scopeLabel sql.NullString
		var documentID sql.NullString
		var sectionID sql.NullString
		if err := rows.Scan(&c.ID, &scopeType, &scopeID, &scopeLabel, &c.LevelID, &documentID, &sectionID, &c.TotalEntities, &c.TracedEntities, &c.CoveragePct); err != nil {
			return nil, err
		}
		c.ScopeType = model.CoverageScope(scopeType)
		if scopeID.Valid {
			c.ScopeID = scopeID.String
		}
		if scopeLabel.Valid {
			c.ScopeLabel = scopeLabel.String
		}
		if documentID.Valid {
			c.DocumentID = documentID.String
		}
		if sectionID.Valid {
			c.SectionID = sectionID.String
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
