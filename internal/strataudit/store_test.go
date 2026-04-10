package strataudit

import (
	"context"
	"path/filepath"
	"testing"

	"sdp_dev/internal/strataudit/model"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewSQLiteStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSQLiteStore_SaveAndLoadLevels(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	levels := []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0, Patterns: []string{"*vision*"}},
		{ID: "strategy", Name: "Strategy", Rank: 1, Patterns: []string{"*strat*"}},
	}
	if err := store.SaveLevels(ctx, levels); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}

	got, err := store.LoadLevels(ctx)
	if err != nil {
		t.Fatalf("LoadLevels: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Name != "Vision" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "Vision")
	}
}

func TestSQLiteStore_SaveEntitiesAndGetByLevel(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_ = store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
	})
	_ = store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "vis.md", LevelID: "vision", ContentHash: "abc", Content: "text"},
	})

	entities := []model.Entity{
		{ID: "e1", DocumentID: "d1", LevelID: "vision", Type: model.EntityGoal, Title: "Global expansion"},
		{ID: "e2", DocumentID: "d1", LevelID: "vision", Type: model.EntityGoal, Title: "AI-first"},
	}
	if err := store.SaveEntities(ctx, entities); err != nil {
		t.Fatalf("SaveEntities: %v", err)
	}

	got, err := store.EntitiesByLevel(ctx, "vision", model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("EntitiesByLevel: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
}

func TestSQLiteStore_SaveTracesAndGetForEntity(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_ = store.SaveLevels(ctx, []model.Level{{ID: "l0", Name: "L0", Rank: 0}, {ID: "l1", Name: "L1", Rank: 1}})
	_ = store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "a.md", LevelID: "l0", ContentHash: "a", Content: "a"},
		{ID: "d2", Path: "b.md", LevelID: "l1", ContentHash: "b", Content: "b"},
	})
	_ = store.SaveEntities(ctx, []model.Entity{
		{ID: "e1", DocumentID: "d1", LevelID: "l0", Type: model.EntityGoal, Title: "G1"},
		{ID: "e2", DocumentID: "d2", LevelID: "l1", Type: model.EntityTask, Title: "T1"},
	})

	traces := []model.Trace{
		{ID: "t1", SourceEntityID: "e2", TargetEntityID: "e1", Relation: model.RelationContributesTo, Confidence: 0.85, Direction: model.DirectionUp},
	}
	if err := store.SaveTraces(ctx, traces); err != nil {
		t.Fatalf("SaveTraces: %v", err)
	}

	got, err := store.TracesForEntity(ctx, "e2")
	if err != nil {
		t.Fatalf("TracesForEntity: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Relation != model.RelationContributesTo {
		t.Errorf("Relation = %q, want %q", got[0].Relation, model.RelationContributesTo)
	}
}

func TestSQLiteStore_SaveFindings(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	findings := []model.Finding{
		{ID: "f1", Type: model.FindingGap, Severity: model.SeverityCritical, Title: "No support", ConfidenceScore: 1.0},
	}
	if err := store.SaveFindings(ctx, findings); err != nil {
		t.Fatalf("SaveFindings: %v", err)
	}

	got, err := store.FindingsByType(ctx, model.FindingGap, model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("FindingsByType: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
}

func TestSQLiteStore_CascadeDelete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_ = store.SaveLevels(ctx, []model.Level{{ID: "l0", Name: "L0", Rank: 0}})
	_ = store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "vis.md", LevelID: "l0", ContentHash: "abc", Content: "text"},
	})
	_ = store.SaveEntities(ctx, []model.Entity{
		{ID: "e1", DocumentID: "d1", LevelID: "l0", Type: model.EntityGoal, Title: "G1"},
	})

	if err := store.DeleteEntitiesForDocument(ctx, "d1"); err != nil {
		t.Fatalf("DeleteEntitiesForDocument: %v", err)
	}

	got, _ := store.EntitiesByLevel(ctx, "l0", model.Page{Limit: 100})
	if len(got) != 0 {
		t.Fatalf("after delete, len(got) = %d, want 0", len(got))
	}
}

func TestSQLiteStore_Coverage(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_ = store.SaveLevels(ctx, []model.Level{{ID: "l0", Name: "L0", Rank: 0}})

	coverages := []model.Coverage{
		{ID: "c1", LevelID: "l0", TotalEntities: 10, TracedEntities: 7, CoveragePct: 70},
	}
	_ = store.SaveCoverage(ctx, coverages)

	got, err := store.CoverageByLevel(ctx)
	if err != nil {
		t.Fatalf("CoverageByLevel: %v", err)
	}
	if len(got) != 1 || got[0].CoveragePct != 70 {
		t.Fatalf("unexpected coverage: %+v", got)
	}
}

func TestSQLiteStore_PipelineState(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	state := model.PipelineState{ID: "ps1", Stage: "extract", Status: "completed", Checkpoint: `{"last_doc": "d5"}`}
	if err := store.SavePipelineState(ctx, state); err != nil {
		t.Fatalf("SavePipelineState: %v", err)
	}

	got, err := store.LoadPipelineState(ctx, "extract")
	if err != nil {
		t.Fatalf("LoadPipelineState: %v", err)
	}
	if got.Checkpoint != state.Checkpoint {
		t.Errorf("Checkpoint = %q, want %q", got.Checkpoint, state.Checkpoint)
	}
}
