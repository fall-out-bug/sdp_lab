package strataudit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"sdp_dev/internal/strataudit/model"
)

func TestEndToEnd_StoreAndQuery(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	// 1. Save levels
	_ = store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
		{ID: "strategy", Name: "Strategy", Rank: 1},
		{ID: "task", Name: "Task", Rank: 2},
	})

	// 2. Save documents
	_ = store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "vision.md", LevelID: "vision", ContentHash: "h1", Content: "Be the global leader"},
		{ID: "d2", Path: "strategy.md", LevelID: "strategy", ContentHash: "h2", Content: "Expand to SEA"},
		{ID: "d3", Path: "tasks.md", LevelID: "task", ContentHash: "h3", Content: "Hire country manager"},
	})

	// 3. Save entities
	_ = store.SaveEntities(ctx, []model.Entity{
		{ID: "e1", DocumentID: "d1", LevelID: "vision", Type: model.EntityGoal, Title: "Global leadership"},
		{ID: "e2", DocumentID: "d2", LevelID: "strategy", Type: model.EntityObjective, Title: "SEA expansion"},
		{ID: "e3", DocumentID: "d3", LevelID: "task", Type: model.EntityTask, Title: "Hire SG manager"},
	})

	// 4. Save traces
	_ = store.SaveTraces(ctx, []model.Trace{
		{ID: "t1", SourceEntityID: "e2", TargetEntityID: "e1", Relation: model.RelationContributesTo, Confidence: 0.9, Direction: model.DirectionUp},
		{ID: "t2", SourceEntityID: "e3", TargetEntityID: "e2", Relation: model.RelationContributesTo, Confidence: 0.85, Direction: model.DirectionUp},
	})

	// 5. Save findings
	_ = store.SaveFindings(ctx, []model.Finding{
		{ID: "f1", Type: model.FindingAlignment, Severity: model.SeverityInfo, Title: "SEA fully traced", ConfidenceScore: 0.88, LLMScore: model.LLMScoreHigh},
	})

	// Verify queries
	entities, _ := store.EntitiesByLevel(ctx, "task", model.Page{Limit: 100})
	if len(entities) != 1 {
		t.Fatalf("task entities: got %d, want 1", len(entities))
	}

	traces, _ := store.TracesForEntity(ctx, "e3")
	if len(traces) != 1 {
		t.Fatalf("traces for e3: got %d, want 1", len(traces))
	}

	findings, _ := store.FindingsByType(ctx, model.FindingAlignment, model.Page{Limit: 100})
	if len(findings) != 1 {
		t.Fatalf("alignment findings: got %d, want 1", len(findings))
	}

	// Verify coverage computation
	for _, levelID := range []string{"vision", "strategy", "task"} {
		count, _ := store.CountEntitiesByLevel(ctx, levelID)
		t.Logf("Level %s: %d entities", levelID, count)
	}

	// Verify pipeline state persistence
	_ = store.SavePipelineState(ctx, model.PipelineState{ID: "ps1", Stage: "ingest", Status: "completed", Checkpoint: `{"last": "d3"}`})
	ps, _ := store.LoadPipelineState(ctx, "ingest")
	if ps == nil || ps.Status != "completed" {
		t.Fatal("pipeline state not persisted correctly")
	}

	// Verify WAL mode
	var mode string
	_ = store.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode)
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func newMockLLMClient(t *testing.T, response string) *LLMClient {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		defer func() { _ = r.Body.Close() }()
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		call++
		payload := fmt.Sprintf(`{"choices":[{"message":{"role":"assistant","content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":4,"cost":0.0001}}`, response)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(func() {
		srv.Close()
	})
	client := NewLLMClient("test-key", srv.URL)
	client.SetRateLimit(1200)
	client.SetRetryConfig(0, 0)
	t.Cleanup(func() {
		if call == 0 {
			t.Error("mock llm was never called")
		}
	})
	return client
}

func TestExtractEntities_RejectsSourceQuoteMismatch(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	cfg := &Config{
		Levels: []LevelConfig{
			{Name: "vision", Rank: 0},
		},
		EntityTypes: []string{"goal"},
		LLM: LLMConfig{
			ExtractModel: "test-model",
		},
		Thresholds: ThresholdConfig{
			ChunkTokenLimit:    3000,
			ChunkOverlapTokens: 0,
		},
	}
	cfg.setDefaults()

	if err := store.SaveLevels(ctx, cfgToLevels(cfg)); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{
			ID:          "d1",
			Path:        "vision.md",
			LevelID:     "vision",
			ContentHash: "h1",
			Content:     "Стратегия: выйти на 30% рынка цифровых платежей в 2027.",
		},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}

	llm := newMockLLMClient(t, `{"entities":[{"type":"goal","title":"Financial target","description":"Increase share","source_quote":"This phrase does not exist in the document"}]}`)
	result, err := ExtractEntities(ctx, cfg, store, llm)
	if err != nil {
		t.Fatalf("ExtractEntities: %v", err)
	}

	entities, err := store.EntitiesByLevel(ctx, "vision", model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("EntitiesByLevel: %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("entities: got %d, want 0 (quote mismatch must be rejected as suspect path)", len(entities))
	}
	if result.EntitiesExtracted != 0 {
		t.Fatalf("result.EntitiesExtracted = %d, want 0", result.EntitiesExtracted)
	}
	if result.RejectedEntities != 1 {
		t.Fatalf("result.RejectedEntities = %d, want 1", result.RejectedEntities)
	}
}

func TestExtractEntities_RejectsBoilerplateRepetition(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()

	cfg := &Config{
		Levels: []LevelConfig{
			{Name: "vision", Rank: 0},
		},
		EntityTypes: []string{"goal"},
		LLM: LLMConfig{
			ExtractModel: "test-model",
		},
		Thresholds: ThresholdConfig{
			ChunkTokenLimit:    2,
			ChunkOverlapTokens: 0,
		},
	}
	cfg.setDefaults()

	if err := store.SaveLevels(ctx, cfgToLevels(cfg)); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{
			ID:          "d1",
			Path:        "vision.md",
			LevelID:     "vision",
			ContentHash: "h1",
			Content:     "This document is confidential. This document is confidential. This document is confidential.",
		},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}

	llm := newMockLLMClient(t, `{"entities":[{"type":"goal","title":"Confidentiality rule","description":"Template boilerplate","source_quote":"This document is confidential."}]}`)
	result, err := ExtractEntities(ctx, cfg, store, llm)
	if err != nil {
		t.Fatalf("ExtractEntities: %v", err)
	}

	entities, err := store.EntitiesByLevel(ctx, "vision", model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("EntitiesByLevel: %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("entities: got %d, want 0 (boilerplate repetition must not be verified truth)", len(entities))
	}
	if result.EntitiesExtracted != 0 {
		t.Fatalf("result.EntitiesExtracted = %d, want 0", result.EntitiesExtracted)
	}
	if result.RejectedEntities != 1 {
		t.Fatalf("result.RejectedEntities = %d, want 1", result.RejectedEntities)
	}
}
