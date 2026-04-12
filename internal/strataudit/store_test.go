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

func TestSQLiteStore_SaveEntities_PersistsTrustFields(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_ = store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
	})
	_ = store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "vis.md", LevelID: "vision", ContentHash: "abc", Content: "text"},
	})
	if err := store.SaveSections(ctx, []model.Section{
		{
			ID:           "s1",
			DocumentID:   "d1",
			Ordinal:      0,
			CharStart:    0,
			CharEnd:      4,
			Preview:      "text",
			Content:      "text",
			ContentHash:  "hash",
			QualityFlags: []string{"section_parse_fallback"},
		},
	}); err != nil {
		t.Fatalf("SaveSections: %v", err)
	}

	entities := []model.Entity{
		{
			ID:                  "e1",
			DocumentID:          "d1",
			SectionID:           "s1",
			LevelID:             "vision",
			Type:                model.EntityGoal,
			Title:               "Глобальная экспансия",
			Description:         "Описание для отчёта",
			TitleOriginal:       "Глобальная экспансия",
			DescriptionOriginal: "Описание для отчёта",
			SourceQuote:         "text",
			QuoteStartOffset:    intPtr(0),
			QuoteEndOffset:      intPtr(4),
			Lang:                "ru",
			TrustGrade:          model.TrustGradeVerified,
			QualityFlags:        []string{"quote_verified"},
		},
	}
	if err := store.SaveEntities(ctx, entities); err != nil {
		t.Fatalf("SaveEntities: %v", err)
	}

	got, err := store.EntitiesByLevel(ctx, "vision", model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("EntitiesByLevel: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].TrustGrade != model.TrustGradeVerified {
		t.Fatalf("TrustGrade = %q, want %q", got[0].TrustGrade, model.TrustGradeVerified)
	}
	if got[0].TitleOriginal != "Глобальная экспансия" {
		t.Fatalf("TitleOriginal = %q", got[0].TitleOriginal)
	}
	if got[0].DescriptionOriginal != "Описание для отчёта" {
		t.Fatalf("DescriptionOriginal = %q", got[0].DescriptionOriginal)
	}
	if got[0].Lang != "ru" {
		t.Fatalf("Lang = %q, want ru", got[0].Lang)
	}
	if got[0].SectionID != "s1" {
		t.Fatalf("SectionID = %q, want s1", got[0].SectionID)
	}
	if got[0].QuoteStartOffset == nil || *got[0].QuoteStartOffset != 0 {
		t.Fatalf("QuoteStartOffset = %+v, want 0", got[0].QuoteStartOffset)
	}
	if got[0].QuoteEndOffset == nil || *got[0].QuoteEndOffset != 4 {
		t.Fatalf("QuoteEndOffset = %+v, want 4", got[0].QuoteEndOffset)
	}
	if len(got[0].QualityFlags) != 1 || got[0].QualityFlags[0] != "quote_verified" {
		t.Fatalf("QualityFlags = %+v, want [quote_verified]", got[0].QualityFlags)
	}
}

func TestSQLiteStore_SaveSectionsAndRoundTrip(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_ = store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
	})
	_ = store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "/tmp/vision.md", LevelID: "vision", ContentHash: "abc", Content: "Vision content"},
	})

	sections := []model.Section{
		{
			ID:           "s1",
			DocumentID:   "d1",
			Ordinal:      0,
			Heading:      "Введение",
			CharStart:    0,
			CharEnd:      14,
			Preview:      "Vision content",
			Content:      "Vision content",
			ContentHash:  "hash1",
			QualityFlags: []string{"section_parse_fallback"},
		},
	}
	if err := store.SaveSections(ctx, sections); err != nil {
		t.Fatalf("SaveSections: %v", err)
	}

	got, err := store.SectionsByDocument(ctx, "d1")
	if err != nil {
		t.Fatalf("SectionsByDocument: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].Heading != "Введение" {
		t.Fatalf("Heading = %q, want Введение", got[0].Heading)
	}
	if got[0].Preview != "Vision content" {
		t.Fatalf("Preview = %q", got[0].Preview)
	}
	if len(got[0].QualityFlags) != 1 || got[0].QualityFlags[0] != "section_parse_fallback" {
		t.Fatalf("QualityFlags = %+v", got[0].QualityFlags)
	}

	allDocs, err := store.AllDocuments(ctx)
	if err != nil {
		t.Fatalf("AllDocuments: %v", err)
	}
	if len(allDocs) != 1 || allDocs[0].Path != "/tmp/vision.md" {
		t.Fatalf("AllDocuments = %+v", allDocs)
	}

	allSections, err := store.AllSections(ctx)
	if err != nil {
		t.Fatalf("AllSections: %v", err)
	}
	if len(allSections) != 1 || allSections[0].ID != "s1" {
		t.Fatalf("AllSections = %+v", allSections)
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
	_ = store.SaveSections(ctx, []model.Section{
		{ID: "s1", DocumentID: "d1", Ordinal: 0, CharStart: 0, CharEnd: 1, Preview: "a", Content: "a", ContentHash: "ha"},
		{ID: "s2", DocumentID: "d2", Ordinal: 0, CharStart: 0, CharEnd: 1, Preview: "b", Content: "b", ContentHash: "hb"},
	})
	_ = store.SaveEntities(ctx, []model.Entity{
		{ID: "e1", DocumentID: "d1", LevelID: "l0", Type: model.EntityGoal, Title: "G1"},
		{ID: "e2", DocumentID: "d2", LevelID: "l1", Type: model.EntityTask, Title: "T1"},
	})

	traces := []model.Trace{
		{
			ID:                     "t1",
			SourceEntityID:         "e2",
			TargetEntityID:         "e1",
			Relation:               model.RelationContributesTo,
			Confidence:             0.85,
			SimilarityScore:        0.91,
			Justification:          "Нижняя задача поддерживает верхнюю цель.",
			Direction:              model.DirectionUp,
			VerificationMode:       model.TraceVerificationModeLLMEvidence,
			TrustGrade:             model.TrustGradeVerified,
			SourceSectionID:        "s2",
			TargetSectionID:        "s1",
			SourceQuoteStartOffset: intPtr(3),
			SourceQuoteEndOffset:   intPtr(7),
			TargetQuoteStartOffset: intPtr(0),
			TargetQuoteEndOffset:   intPtr(2),
		},
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
	if got[0].SimilarityScore != 0.91 {
		t.Fatalf("SimilarityScore = %f, want 0.91", got[0].SimilarityScore)
	}
	if got[0].VerificationMode != model.TraceVerificationModeLLMEvidence {
		t.Fatalf("VerificationMode = %q", got[0].VerificationMode)
	}
	if got[0].SourceSectionID != "s2" || got[0].TargetSectionID != "s1" {
		t.Fatalf("unexpected section refs: %+v", got[0])
	}
}

func TestSQLiteStore_SaveCandidatesAndLoadAll(t *testing.T) {
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

	if err := store.SaveCandidates(ctx, []model.Candidate{
		{
			ID:             "cand_1",
			SourceEntityID: "e2",
			TargetEntityID: "e1",
			Similarity:     0.82,
			Verified:       true,
			TraceID:        "tr_1",
			DiagnosticCode: "embedding_similarity_candidate",
		},
	}); err != nil {
		t.Fatalf("SaveCandidates: %v", err)
	}

	got, err := store.AllCandidates(ctx)
	if err != nil {
		t.Fatalf("AllCandidates: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if !got[0].Verified || got[0].TraceID != "tr_1" {
		t.Fatalf("unexpected candidate linkage: %+v", got[0])
	}
	if got[0].DiagnosticCode != "embedding_similarity_candidate" {
		t.Fatalf("DiagnosticCode = %q", got[0].DiagnosticCode)
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
