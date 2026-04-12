package strataudit

import (
	"context"
	"testing"

	"sdp_dev/internal/strataudit/model"
)

func TestBuildReport_ExportsDocumentSectionAndEntityProvenance(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	if err := store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
	}); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "/tmp/vision.md", LevelID: "vision", ContentHash: "hash-doc", Content: "Стратегия лидера"},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}
	if err := store.SaveSections(ctx, []model.Section{
		{
			ID:           "s1",
			DocumentID:   "d1",
			Ordinal:      0,
			Heading:      "Введение",
			CharStart:    0,
			CharEnd:      17,
			Preview:      "Стратегия лидера",
			Content:      "Стратегия лидера",
			ContentHash:  "hash-sec",
			QualityFlags: []string{"section_parse_fallback"},
		},
	}); err != nil {
		t.Fatalf("SaveSections: %v", err)
	}

	start, end := 0, 17
	if err := store.SaveEntities(ctx, []model.Entity{
		{
			ID:               "e1",
			DocumentID:       "d1",
			SectionID:        "s1",
			LevelID:          "vision",
			Type:             model.EntityGoal,
			Title:            "Стратегия лидера",
			TitleOriginal:    "Стратегия лидера",
			Description:      "Держать лидерство",
			SourceQuote:      "Стратегия лидера",
			QuoteStartOffset: &start,
			QuoteEndOffset:   &end,
			TrustGrade:       model.TrustGradeVerified,
			QualityFlags:     []string{"quote_verified"},
		},
	}); err != nil {
		t.Fatalf("SaveEntities: %v", err)
	}

	rpt, err := BuildReport(ctx, &Config{Project: ProjectConfig{Name: "test-project"}}, store)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if len(rpt.Documents) != 1 {
		t.Fatalf("len(rpt.Documents) = %d, want 1", len(rpt.Documents))
	}
	if rpt.Documents[0].Path != "/tmp/vision.md" {
		t.Fatalf("document path = %q", rpt.Documents[0].Path)
	}
	if len(rpt.Sections) != 1 {
		t.Fatalf("len(rpt.Sections) = %d, want 1", len(rpt.Sections))
	}
	if rpt.Sections[0].ID != "s1" {
		t.Fatalf("section id = %q, want s1", rpt.Sections[0].ID)
	}
	if len(rpt.Entities) != 1 {
		t.Fatalf("len(rpt.Entities) = %d, want 1", len(rpt.Entities))
	}
	if rpt.Entities[0].SectionID != "s1" {
		t.Fatalf("entity section_id = %q, want s1", rpt.Entities[0].SectionID)
	}
	if rpt.Entities[0].SourceQuote != "Стратегия лидера" {
		t.Fatalf("entity source_quote = %q", rpt.Entities[0].SourceQuote)
	}
	if rpt.Entities[0].QuoteStartOffset == nil || *rpt.Entities[0].QuoteStartOffset != 0 {
		t.Fatalf("entity quote_start_offset = %+v, want 0", rpt.Entities[0].QuoteStartOffset)
	}
}
