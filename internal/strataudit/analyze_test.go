package strataudit

import (
	"context"
	"fmt"
	"testing"

	"sdp_dev/internal/strataudit/model"
)

func TestFindingTemplates_AllClusterTypes(t *testing.T) {
	types := []string{"strategic_gap_cluster", "orphan_cluster", "corpus_quality_cluster", "trace_ambiguity_cluster"}
	for _, ft := range types {
		t.Run(ft, func(t *testing.T) {
			ru := tpl("ru", ft)
			en := tpl("en", ft)
			if ru.Title == "" {
				t.Error("ru title empty")
			}
			if en.Title == "" {
				t.Error("en title empty")
			}
		})
	}
}

func TestFindingTemplates_RussianStrategicGap(t *testing.T) {
	tpl := tpl("ru", "strategic_gap_cluster")
	title := fmt.Sprintf(tpl.Title, 2, "vision.md", "strategy")
	if title == "" {
		t.Fatal("empty title")
	}
	found := false
	for _, r := range title {
		if r >= 0x0400 && r <= 0x04FF {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("title has no Cyrillic: %q", title)
	}
}

func TestFindingTemplates_Fallback(t *testing.T) {
	got := tpl("xx", "strategic_gap_cluster")
	if got.Title == "" {
		t.Error("expected English fallback")
	}
	en := tpl("en", "strategic_gap_cluster")
	if got.Title != en.Title {
		t.Errorf("got %q, want English fallback %q", got.Title, en.Title)
	}
}

func TestConfigDefaultLang(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()
	if cfg.Output.Lang != "ru" {
		t.Errorf("default lang = %q, want 'ru'", cfg.Output.Lang)
	}
}

func TestAnalyze_GroupsRepeatedGapsIntoStableClusters(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	cfg := &Config{
		Output: OutputConfig{Lang: "ru"},
		Thresholds: ThresholdConfig{
			CoverageWarn: 70,
		},
	}

	mustSaveAnalyzeFixture(t, ctx, store)

	result, err := Analyze(ctx, cfg, store)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if result.Findings == 0 {
		t.Fatal("expected grouped findings")
	}

	findings, err := store.AllFindings(ctx, model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("AllFindings: %v", err)
	}

	var gapCluster *model.Finding
	for i := range findings {
		if findings[i].Type == model.FindingStrategicGapCluster {
			gapCluster = &findings[i]
			break
		}
	}
	if gapCluster == nil {
		t.Fatal("expected strategic_gap_cluster finding")
	}
	if len(gapCluster.EntityIDs) != 2 {
		t.Fatalf("len(gapCluster.EntityIDs) = %d, want 2", len(gapCluster.EntityIDs))
	}
	if len(gapCluster.DocumentIDs) != 1 || gapCluster.DocumentIDs[0] != "d_vision" {
		t.Fatalf("unexpected gap cluster document refs: %+v", gapCluster.DocumentIDs)
	}
}

func TestAnalyze_SeparatesCorpusQualityAndCoverageScopes(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	cfg := &Config{
		Output: OutputConfig{Lang: "ru"},
		Thresholds: ThresholdConfig{
			CoverageWarn: 70,
		},
	}

	if err := store.SaveLevels(ctx, []model.Level{
		{ID: "strategy", Name: "Strategy", Rank: 0},
	}); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{ID: "d1", Path: "/tmp/dirty-strategy.md", LevelID: "strategy", ContentHash: "h1", Content: "content"},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}
	if err := store.SaveSections(ctx, []model.Section{
		{ID: "s1", DocumentID: "d1", Ordinal: 0, CharStart: 0, CharEnd: 7, Preview: "content", Content: "content", ContentHash: "hs1", QualityFlags: []string{"mime_header_noise"}},
	}); err != nil {
		t.Fatalf("SaveSections: %v", err)
	}
	if err := store.SaveEntities(ctx, []model.Entity{
		{
			ID:           "e1",
			DocumentID:   "d1",
			SectionID:    "s1",
			LevelID:      "strategy",
			Type:         model.EntityGoal,
			Title:        "Цель",
			SourceQuote:  "content",
			TrustGrade:   model.TrustGradeVerified,
			QualityFlags: []string{"language_mismatch"},
		},
	}); err != nil {
		t.Fatalf("SaveEntities: %v", err)
	}

	if _, err := Analyze(ctx, cfg, store); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	findings, err := store.AllFindings(ctx, model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("AllFindings: %v", err)
	}
	var qualityCluster *model.Finding
	for i := range findings {
		if findings[i].Type == model.FindingCorpusQualityCluster {
			qualityCluster = &findings[i]
			break
		}
	}
	if qualityCluster == nil {
		t.Fatal("expected corpus_quality_cluster finding")
	}
	if len(qualityCluster.DocumentIDs) != 1 || qualityCluster.DocumentIDs[0] != "d1" {
		t.Fatalf("unexpected quality cluster document refs: %+v", qualityCluster.DocumentIDs)
	}

	coverages, err := store.AllCoverage(ctx)
	if err != nil {
		t.Fatalf("AllCoverage: %v", err)
	}
	if !hasCoverageScope(coverages, model.CoverageScopeLevel) || !hasCoverageScope(coverages, model.CoverageScopeDocument) || !hasCoverageScope(coverages, model.CoverageScopeSection) {
		t.Fatalf("expected level/document/section coverage, got %+v", coverages)
	}
}

func TestAnalyze_DirectionalSupportDoesNotHideStrategicGap(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	cfg := &Config{
		Output: OutputConfig{Lang: "ru"},
		Thresholds: ThresholdConfig{
			CoverageWarn: 70,
		},
	}

	if err := store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
		{ID: "strategy", Name: "Strategy", Rank: 1},
		{ID: "task", Name: "Task", Rank: 2},
	}); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{ID: "d_v", Path: "/tmp/vision.md", LevelID: "vision", ContentHash: "hv", Content: "vision"},
		{ID: "d_s", Path: "/tmp/strategy.md", LevelID: "strategy", ContentHash: "hs", Content: "strategy"},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}
	if err := store.SaveSections(ctx, []model.Section{
		{ID: "s_v", DocumentID: "d_v", Ordinal: 0, CharStart: 0, CharEnd: 6, Preview: "vision", Content: "vision", ContentHash: "sv"},
		{ID: "s_s", DocumentID: "d_s", Ordinal: 0, CharStart: 0, CharEnd: 8, Preview: "strategy", Content: "strategy", ContentHash: "ss"},
	}); err != nil {
		t.Fatalf("SaveSections: %v", err)
	}
	if err := store.SaveEntities(ctx, []model.Entity{
		{
			ID:               "e_v",
			DocumentID:       "d_v",
			SectionID:        "s_v",
			LevelID:          "vision",
			Type:             model.EntityGoal,
			Title:            "Видение",
			SourceQuote:      "vision",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(6),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "e_s",
			DocumentID:       "d_s",
			SectionID:        "s_s",
			LevelID:          "strategy",
			Type:             model.EntityObjective,
			Title:            "Стратегия",
			SourceQuote:      "strategy",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(8),
			TrustGrade:       model.TrustGradeVerified,
		},
	}); err != nil {
		t.Fatalf("SaveEntities: %v", err)
	}
	if err := store.SaveTraces(ctx, []model.Trace{
		{
			ID:                     "tr_s_v",
			SourceEntityID:         "e_s",
			TargetEntityID:         "e_v",
			Relation:               model.RelationContributesTo,
			Direction:              model.DirectionUp,
			Confidence:             0.92,
			SimilarityScore:        0.95,
			VerificationMode:       model.TraceVerificationModeLLMEvidence,
			TrustGrade:             model.TrustGradeVerified,
			SourceSectionID:        "s_s",
			TargetSectionID:        "s_v",
			SourceQuoteStartOffset: intPtr(0),
			SourceQuoteEndOffset:   intPtr(8),
			TargetQuoteStartOffset: intPtr(0),
			TargetQuoteEndOffset:   intPtr(6),
		},
	}); err != nil {
		t.Fatalf("SaveTraces: %v", err)
	}

	if _, err := Analyze(ctx, cfg, store); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	findings, err := store.AllFindings(ctx, model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("AllFindings: %v", err)
	}
	for _, finding := range findings {
		if finding.Type != model.FindingStrategicGapCluster {
			continue
		}
		if len(finding.DocumentIDs) == 1 && finding.DocumentIDs[0] == "d_s" {
			return
		}
	}
	t.Fatal("expected strategy document to remain a strategic gap without lower-level support")
}

func TestAnalyze_TraceAmbiguityUsesUnverifiedCandidates(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	cfg := &Config{
		Output: OutputConfig{Lang: "ru"},
		Thresholds: ThresholdConfig{
			CoverageWarn: 70,
		},
	}

	if err := store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
		{ID: "strategy", Name: "Strategy", Rank: 1},
	}); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{ID: "d_v", Path: "/tmp/vision.md", LevelID: "vision", ContentHash: "hv", Content: "vision"},
		{ID: "d_s", Path: "/tmp/strategy.md", LevelID: "strategy", ContentHash: "hs", Content: "strategy"},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}
	if err := store.SaveSections(ctx, []model.Section{
		{ID: "s_v", DocumentID: "d_v", Ordinal: 0, CharStart: 0, CharEnd: 6, Preview: "vision", Content: "vision", ContentHash: "sv"},
		{ID: "s_s", DocumentID: "d_s", Ordinal: 0, CharStart: 0, CharEnd: 8, Preview: "strategy", Content: "strategy", ContentHash: "ss"},
	}); err != nil {
		t.Fatalf("SaveSections: %v", err)
	}
	if err := store.SaveEntities(ctx, []model.Entity{
		{
			ID:               "e_v1",
			DocumentID:       "d_v",
			SectionID:        "s_v",
			LevelID:          "vision",
			Type:             model.EntityGoal,
			Title:            "Рост",
			SourceQuote:      "vision",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(6),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "e_v2",
			DocumentID:       "d_v",
			SectionID:        "s_v",
			LevelID:          "vision",
			Type:             model.EntityGoal,
			Title:            "Лидерство",
			SourceQuote:      "vision",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(6),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "e_s",
			DocumentID:       "d_s",
			SectionID:        "s_s",
			LevelID:          "strategy",
			Type:             model.EntityObjective,
			Title:            "Выход на рынок",
			SourceQuote:      "strategy",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(8),
			TrustGrade:       model.TrustGradeVerified,
		},
	}); err != nil {
		t.Fatalf("SaveEntities: %v", err)
	}
	if err := store.SaveCandidates(ctx, []model.Candidate{
		{ID: "cand1", SourceEntityID: "e_s", TargetEntityID: "e_v1", Similarity: 0.91, Verified: false},
		{ID: "cand2", SourceEntityID: "e_s", TargetEntityID: "e_v2", Similarity: 0.83, Verified: false},
	}); err != nil {
		t.Fatalf("SaveCandidates: %v", err)
	}

	if _, err := Analyze(ctx, cfg, store); err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	findings, err := store.AllFindings(ctx, model.Page{Limit: 100})
	if err != nil {
		t.Fatalf("AllFindings: %v", err)
	}
	for _, finding := range findings {
		if finding.Type != model.FindingTraceAmbiguityCluster {
			continue
		}
		if len(finding.DocumentIDs) == 1 && finding.DocumentIDs[0] == "d_s" {
			return
		}
	}
	t.Fatal("expected ambiguity cluster from close unverified candidates")
}

func mustSaveAnalyzeFixture(t *testing.T, ctx context.Context, store *SQLiteStore) {
	t.Helper()

	if err := store.SaveLevels(ctx, []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
		{ID: "strategy", Name: "Strategy", Rank: 1},
	}); err != nil {
		t.Fatalf("SaveLevels: %v", err)
	}
	if err := store.SaveDocuments(ctx, []model.Document{
		{ID: "d_vision", Path: "/tmp/vision.md", LevelID: "vision", ContentHash: "hv", Content: "vision"},
		{ID: "d_strategy", Path: "/tmp/strategy.md", LevelID: "strategy", ContentHash: "hs", Content: "strategy"},
	}); err != nil {
		t.Fatalf("SaveDocuments: %v", err)
	}
	if err := store.SaveSections(ctx, []model.Section{
		{ID: "s_v1", DocumentID: "d_vision", Ordinal: 0, CharStart: 0, CharEnd: 10, Preview: "vision", Content: "vision", ContentHash: "sv1"},
		{ID: "s_s1", DocumentID: "d_strategy", Ordinal: 0, CharStart: 0, CharEnd: 10, Preview: "strategy", Content: "strategy", ContentHash: "ss1"},
	}); err != nil {
		t.Fatalf("SaveSections: %v", err)
	}
	if err := store.SaveEntities(ctx, []model.Entity{
		{
			ID:               "e_v1",
			DocumentID:       "d_vision",
			SectionID:        "s_v1",
			LevelID:          "vision",
			Type:             model.EntityGoal,
			Title:            "Рост рынка",
			SourceQuote:      "vision",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(6),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "e_v2",
			DocumentID:       "d_vision",
			SectionID:        "s_v1",
			LevelID:          "vision",
			Type:             model.EntityGoal,
			Title:            "Лидерство",
			SourceQuote:      "vision",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(6),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "e_s1",
			DocumentID:       "d_strategy",
			SectionID:        "s_s1",
			LevelID:          "strategy",
			Type:             model.EntityObjective,
			Title:            "Операционная цель",
			SourceQuote:      "strategy",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(8),
			TrustGrade:       model.TrustGradeVerified,
		},
	}); err != nil {
		t.Fatalf("SaveEntities: %v", err)
	}
}

func hasCoverageScope(coverages []model.Coverage, scope model.CoverageScope) bool {
	for _, coverage := range coverages {
		if coverage.ScopeType == scope {
			return true
		}
	}
	return false
}
