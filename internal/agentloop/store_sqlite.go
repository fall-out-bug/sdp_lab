package agentloop

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore implements SessionStore using SQLite with WAL journal mode.
// All tables are created at NewSQLiteStore() via idempotent IF NOT EXISTS migrations.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) a SQLite database at path and runs schema migrations.
// WAL journal mode is set for better concurrent read performance.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	st := &SQLiteStore{db: db}
	if err := st.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return st, nil
}

// Close closes the underlying database connection.
func (st *SQLiteStore) Close() error {
	return st.db.Close()
}

// migrate creates all required tables if they do not exist.
func (st *SQLiteStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id      TEXT PRIMARY KEY,
			branch  TEXT NOT NULL DEFAULT '',
			phase   TEXT NOT NULL DEFAULT '',
			feature_id TEXT NOT NULL DEFAULT '',
			ws_id TEXT NOT NULL DEFAULT '',
			project_root TEXT NOT NULL DEFAULT '',
			claimed_issue_id TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS turn_records (
			id          TEXT NOT NULL,
			session_id  TEXT NOT NULL,
			phase       TEXT NOT NULL DEFAULT '',
			user_role   TEXT NOT NULL DEFAULT 'user',
			user_content TEXT NOT NULL DEFAULT '',
			assistant_text TEXT NOT NULL DEFAULT '',
			tool_calls  TEXT NOT NULL DEFAULT '[]',
			tool_results TEXT NOT NULL DEFAULT '[]',
			created_at  INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (session_id, id)
		)`,
		`CREATE TABLE IF NOT EXISTS phase_records (
			rowid_seq   INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT NOT NULL,
			phase       TEXT NOT NULL DEFAULT '',
			next_phase  TEXT NOT NULL DEFAULT '',
			started_at  INTEGER NOT NULL DEFAULT 0,
			ended_at    INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS decisions (
			session_id   TEXT PRIMARY KEY,
			decision_id  TEXT NOT NULL,
			run_id       INTEGER NOT NULL DEFAULT 0,
			phase        TEXT NOT NULL DEFAULT '',
			payload      TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT NOT NULL,
			type        TEXT NOT NULL DEFAULT '',
			payload     TEXT NOT NULL DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS gate_results (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id  TEXT NOT NULL,
			payload     TEXT NOT NULL DEFAULT '{}'
		)`,
	}
	for _, s := range stmts {
		if _, err := st.db.Exec(s); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	for _, col := range []string{
		"feature_id TEXT NOT NULL DEFAULT ''",
		"ws_id TEXT NOT NULL DEFAULT ''",
		"project_root TEXT NOT NULL DEFAULT ''",
		"claimed_issue_id TEXT NOT NULL DEFAULT ''",
	} {
		if err := st.ensureColumn("sessions", col); err != nil {
			return err
		}
	}
	return nil
}

func (st *SQLiteStore) ensureColumn(table, columnDef string) error {
	colName := strings.Fields(columnDef)[0]
	rows, err := st.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return fmt.Errorf("pragma table_info(%s): %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid       int
			name      string
			typeName  string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table_info(%s): %w", table, err)
		}
		if name == colName {
			return nil
		}
	}
	if rows.Err() != nil {
		return fmt.Errorf("iterate table_info(%s): %w", table, rows.Err())
	}

	if _, err := st.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + columnDef); err != nil {
		return fmt.Errorf("alter table %s add column %s: %w", table, colName, err)
	}
	return nil
}

// ---- toolCallRow / toolResultRow for JSON serialization ----

type toolCallRow struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type toolResultRow struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Args   json.RawMessage `json:"arguments,omitempty"`
	Output string          `json:"output"`
	Err    string          `json:"err,omitempty"` // ToolResult.Err stored as string
}

func encodeToolCalls(calls []ToolCall) (string, error) {
	rows := make([]toolCallRow, len(calls))
	for i, c := range calls {
		rows[i] = toolCallRow{ID: c.ID, Name: c.Name, Arguments: c.Arguments}
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeToolCalls(s string) ([]ToolCall, error) {
	if s == "" || s == "[]" {
		return nil, nil
	}
	var rows []toolCallRow
	if err := json.Unmarshal([]byte(s), &rows); err != nil {
		return nil, err
	}
	out := make([]ToolCall, len(rows))
	for i, r := range rows {
		out[i] = ToolCall{ID: r.ID, Name: r.Name, Arguments: r.Arguments}
	}
	return out, nil
}

func encodeToolResults(results []ToolResult) (string, error) {
	rows := make([]toolResultRow, len(results))
	for i, r := range results {
		errStr := ""
		if r.Err != nil {
			errStr = r.Err.Error()
		}
		rows[i] = toolResultRow{
			ID:     r.ID,
			Name:   r.Name,
			Args:   r.Arguments,
			Output: r.Output,
			Err:    errStr,
		}
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeToolResults(s string) ([]ToolResult, error) {
	if s == "" || s == "[]" {
		return nil, nil
	}
	var rows []toolResultRow
	if err := json.Unmarshal([]byte(s), &rows); err != nil {
		return nil, err
	}
	out := make([]ToolResult, len(rows))
	for i, r := range rows {
		var restoredErr error
		if r.Err != "" {
			restoredErr = errors.New(r.Err)
		}
		out[i] = ToolResult{
			ID:        r.ID,
			Name:      r.Name,
			Arguments: r.Args,
			Output:    r.Output,
			Err:       restoredErr,
		}
	}
	return out, nil
}

// ---- SessionStore implementation ----

func (st *SQLiteStore) Persist(s *Session) error {
	_, err := st.db.Exec(
		`INSERT INTO sessions (id, branch, phase, feature_id, ws_id, project_root, claimed_issue_id) VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET branch=excluded.branch, phase=excluded.phase, feature_id=excluded.feature_id, ws_id=excluded.ws_id, project_root=excluded.project_root, claimed_issue_id=excluded.claimed_issue_id`,
		s.ID, s.Branch, string(s.Phase), s.FeatureID, s.WSID, s.ProjectRoot, s.ClaimedIssueID,
	)
	if err != nil {
		return fmt.Errorf("persist session: %w", err)
	}
	return nil
}

func (st *SQLiteStore) Recover(sessionID string) (*Session, error) {
	row := st.db.QueryRow(`SELECT id, branch, phase, feature_id, ws_id, project_root, claimed_issue_id FROM sessions WHERE id = ?`, sessionID)
	var s Session
	var phase string
	if err := row.Scan(&s.ID, &s.Branch, &phase, &s.FeatureID, &s.WSID, &s.ProjectRoot, &s.ClaimedIssueID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session %q not found", sessionID)
		}
		return nil, fmt.Errorf("recover session: %w", err)
	}
	s.Phase = Role(phase)
	return &s, nil
}

func (st *SQLiteStore) PersistEvent(sessionID string, ev Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = st.db.Exec(
		`INSERT INTO events (session_id, type, payload) VALUES (?, ?, ?)`,
		sessionID, ev.Type, string(payload),
	)
	return err
}

func (st *SQLiteStore) LoadEvents(sessionID string) ([]Event, error) {
	rows, err := st.db.Query(
		`SELECT payload FROM events WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		var ev Event
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			return nil, fmt.Errorf("decode event: %w", err)
		}
		events = append(events, ev)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate events: %w", rows.Err())
	}
	return events, nil
}

func (st *SQLiteStore) PersistGateResult(sessionID string, r GateResult) error {
	payload, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal gate result: %w", err)
	}
	_, err = st.db.Exec(
		`INSERT INTO gate_results (session_id, payload) VALUES (?, ?)`,
		sessionID, string(payload),
	)
	return err
}

func (st *SQLiteStore) PersistTurnRecord(sessionID string, r TurnRecord) error {
	tcJSON, err := encodeToolCalls(r.ToolCalls)
	if err != nil {
		return fmt.Errorf("encode tool calls: %w", err)
	}
	trJSON, err := encodeToolResults(r.ToolResults)
	if err != nil {
		return fmt.Errorf("encode tool results: %w", err)
	}
	// Fix R5: plain INSERT — no ON CONFLICT. turn_records is an append-only canonical log.
	// Duplicate IDs (same session_id + id) are a bug in runID generation and must surface
	// as a UNIQUE constraint error, not be silently swallowed by an upsert.
	_, err = st.db.Exec(
		`INSERT INTO turn_records
		 (id, session_id, phase, user_role, user_content, assistant_text, tool_calls, tool_results, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, sessionID, string(r.Phase),
		r.UserMsg.Role, r.UserMsg.Content,
		r.AssistantText, tcJSON, trJSON,
		r.CreatedAt.UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("persist turn record: %w", err)
	}
	return nil
}

func (st *SQLiteStore) LoadTurnRecords(sessionID string) ([]TurnRecord, error) {
	rows, err := st.db.Query(
		`SELECT id, phase, user_role, user_content, assistant_text, tool_calls, tool_results, created_at
		 FROM turn_records WHERE session_id = ? ORDER BY created_at ASC, id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query turn records: %w", err)
	}
	defer rows.Close()

	var out []TurnRecord
	for rows.Next() {
		var tr TurnRecord
		var phase, userRole, tcJSON, trJSON string
		var createdAtUnix int64
		if err := rows.Scan(
			&tr.ID, &phase, &userRole, &tr.UserMsg.Content,
			&tr.AssistantText, &tcJSON, &trJSON, &createdAtUnix,
		); err != nil {
			return nil, fmt.Errorf("scan turn record: %w", err)
		}
		tr.Phase = Role(phase)
		tr.UserMsg.Role = userRole
		tr.CreatedAt = time.Unix(createdAtUnix, 0).UTC()

		if tc, err := decodeToolCalls(tcJSON); err != nil {
			return nil, fmt.Errorf("decode tool calls: %w", err)
		} else {
			tr.ToolCalls = tc
		}
		if trs, err := decodeToolResults(trJSON); err != nil {
			return nil, fmt.Errorf("decode tool results: %w", err)
		} else {
			tr.ToolResults = trs
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

func (st *SQLiteStore) PersistPhaseRecord(sessionID string, r PhaseRecord) error {
	_, err := st.db.Exec(
		`INSERT INTO phase_records (session_id, phase, next_phase, started_at, ended_at)
		 VALUES (?, ?, ?, ?, ?)`,
		sessionID, string(r.Phase), string(r.NextPhase),
		r.StartedAt.UTC().Unix(), r.EndedAt.UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("persist phase record: %w", err)
	}
	return nil
}

func (st *SQLiteStore) LoadPhaseRecords(sessionID string) ([]PhaseRecord, error) {
	rows, err := st.db.Query(
		`SELECT phase, next_phase, started_at, ended_at
		 FROM phase_records WHERE session_id = ? ORDER BY rowid_seq ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query phase records: %w", err)
	}
	defer rows.Close()

	var out []PhaseRecord
	for rows.Next() {
		var pr PhaseRecord
		var phase, nextPhase string
		var startedAtUnix, endedAtUnix int64
		if err := rows.Scan(&phase, &nextPhase, &startedAtUnix, &endedAtUnix); err != nil {
			return nil, fmt.Errorf("scan phase record: %w", err)
		}
		pr.Phase = Role(phase)
		pr.NextPhase = Role(nextPhase)
		pr.StartedAt = time.Unix(startedAtUnix, 0).UTC()
		pr.EndedAt = time.Unix(endedAtUnix, 0).UTC()
		out = append(out, pr)
	}
	return out, rows.Err()
}

func (st *SQLiteStore) PersistDecision(sessionID string, d PendingDecision) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("marshal decision: %w", err)
	}
	_, err = st.db.Exec(
		`INSERT INTO decisions (session_id, decision_id, run_id, phase, payload) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(session_id) DO UPDATE SET
		   decision_id=excluded.decision_id,
		   run_id=excluded.run_id,
		   phase=excluded.phase,
		   payload=excluded.payload`,
		sessionID, d.DecisionID, int64(d.RunID), string(d.Phase), string(payload),
	)
	if err != nil {
		return fmt.Errorf("persist decision: %w", err)
	}
	return nil
}

func (st *SQLiteStore) ValidateDecision(sessionID, decisionID string) error {
	var stored string
	err := st.db.QueryRow(
		`SELECT decision_id FROM decisions WHERE session_id = ?`, sessionID,
	).Scan(&stored)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("no pending decision for session %q", sessionID)
		}
		return fmt.Errorf("validate decision: %w", err)
	}
	if stored != decisionID {
		return fmt.Errorf("decision ID mismatch: want %q got %q", stored, decisionID)
	}
	return nil
}

func (st *SQLiteStore) ClearDecision(sessionID, decisionID string) error {
	if err := st.ValidateDecision(sessionID, decisionID); err != nil {
		return err
	}
	_, err := st.db.Exec(`DELETE FROM decisions WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("clear decision: %w", err)
	}
	return nil
}

func (st *SQLiteStore) LoadDecision(sessionID string) (*PendingDecision, error) {
	var payload string
	err := st.db.QueryRow(
		`SELECT payload FROM decisions WHERE session_id = ?`, sessionID,
	).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // no pending decision
		}
		return nil, fmt.Errorf("load decision: %w", err)
	}
	var d PendingDecision
	if err := json.Unmarshal([]byte(payload), &d); err != nil {
		return nil, fmt.Errorf("unmarshal decision: %w", err)
	}
	return &d, nil
}
