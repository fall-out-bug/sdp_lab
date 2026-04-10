package strataudit

import (
	"context"
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
