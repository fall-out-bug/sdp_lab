package strataudit

import (
	"context"
	"testing"

	"sdp_dev/internal/strataudit/model"
	reportpkg "sdp_dev/internal/strataudit/report"
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
	if err := store.SaveCandidates(ctx, []model.Candidate{
		{
			ID:             "cand_1",
			SourceEntityID: "e1",
			TargetEntityID: "e1",
			Similarity:     0.88,
			DiagnosticCode: "embedding_similarity_candidate",
		},
	}); err != nil {
		t.Fatalf("SaveCandidates: %v", err)
	}
	if err := store.SaveTraces(ctx, []model.Trace{
		{
			ID:                     "tr_1",
			SourceEntityID:         "e1",
			TargetEntityID:         "e1",
			Relation:               model.RelationContributesTo,
			Confidence:             0.93,
			SimilarityScore:        0.88,
			Justification:          "Evidence-backed relation.",
			Direction:              model.DirectionUp,
			VerificationMode:       model.TraceVerificationModeLLMEvidence,
			TrustGrade:             model.TrustGradeVerified,
			SourceSectionID:        "s1",
			TargetSectionID:        "s1",
			SourceQuoteStartOffset: intPtr(0),
			SourceQuoteEndOffset:   intPtr(17),
			TargetQuoteStartOffset: intPtr(0),
			TargetQuoteEndOffset:   intPtr(17),
		},
	}); err != nil {
		t.Fatalf("SaveTraces: %v", err)
	}
	if err := store.SaveFindings(ctx, []model.Finding{
		{
			ID:              "f1",
			Type:            model.FindingStrategicGapCluster,
			Severity:        model.SeverityWarn,
			EntityIDs:       []string{"e1"},
			DocumentIDs:     []string{"d1"},
			SectionIDs:      []string{"s1"},
			ClusterKey:      "strategic_gap:vision:strategy:d1",
			Title:           "Стратегический разрыв",
			Description:     "Описание кластера",
			Recommendation:  "Добавить поддержку",
			ConfidenceScore: 1,
		},
	}); err != nil {
		t.Fatalf("SaveFindings: %v", err)
	}
	if err := store.SaveCoverage(ctx, []model.Coverage{
		{
			ID:             "cov_vision",
			ScopeType:      model.CoverageScopeLevel,
			ScopeID:        "vision",
			ScopeLabel:     "Vision",
			LevelID:        "vision",
			TotalEntities:  1,
			TracedEntities: 1,
			CoveragePct:    100,
		},
	}); err != nil {
		t.Fatalf("SaveCoverage: %v", err)
	}
	if err := store.SavePipelineState(ctx, model.PipelineState{
		ID:         "ps_extract_1",
		Stage:      "extract",
		Status:     "completed",
		Checkpoint: `{"verified":1,"suspect":0,"rejected":2,"documents":1,"saved":1}`,
	}); err != nil {
		t.Fatalf("SavePipelineState: %v", err)
	}

	rpt, err := BuildReport(ctx, &Config{
		Project: ProjectConfig{
			Name:        "test-project",
			Description: "audit",
			SourceDirs:  []string{"docs/strategy"},
		},
		Output: OutputConfig{
			Dir:  "/tmp/.strataudit",
			Lang: "ru",
		},
	}, store)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	if rpt.SchemaVersion != reportpkg.SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", rpt.SchemaVersion, reportpkg.SchemaVersion)
	}
	if rpt.AuditScope.ProjectName != "test-project" {
		t.Fatalf("audit_scope.project_name = %q", rpt.AuditScope.ProjectName)
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
	if len(rpt.TraceCandidates) != 1 {
		t.Fatalf("len(rpt.TraceCandidates) = %d, want 1", len(rpt.TraceCandidates))
	}
	if rpt.TraceCandidates[0].DiagnosticCode != "embedding_similarity_candidate" {
		t.Fatalf("trace candidate diagnostic = %q", rpt.TraceCandidates[0].DiagnosticCode)
	}
	if len(rpt.VerifiedTraces) != 1 {
		t.Fatalf("len(rpt.VerifiedTraces) = %d, want 1", len(rpt.VerifiedTraces))
	}
	if rpt.VerifiedTraces[0].VerificationMode != string(model.TraceVerificationModeLLMEvidence) {
		t.Fatalf("verification_mode = %q", rpt.VerifiedTraces[0].VerificationMode)
	}
	if rpt.VerifiedTraces[0].TraceEvidence.Source.SectionID != "s1" || rpt.VerifiedTraces[0].TraceEvidence.Target.SectionID != "s1" {
		t.Fatalf("unexpected verified trace evidence refs: %+v", rpt.VerifiedTraces[0].TraceEvidence)
	}
	if rpt.VerifiedTraces[0].TraceEvidence.Source.DocumentPath != "/tmp/vision.md" {
		t.Fatalf("source document path = %q", rpt.VerifiedTraces[0].TraceEvidence.Source.DocumentPath)
	}
	if len(rpt.TraceGraph.Nodes) != 1 {
		t.Fatalf("len(rpt.TraceGraph.Nodes) = %d, want 1", len(rpt.TraceGraph.Nodes))
	}
	if rpt.TraceGraph.Nodes[0].EntityID != "e1" || rpt.TraceGraph.Nodes[0].DocumentPath != "/tmp/vision.md" {
		t.Fatalf("unexpected trace node: %+v", rpt.TraceGraph.Nodes[0])
	}
	if len(rpt.TraceGraph.Edges) != 2 {
		t.Fatalf("len(rpt.TraceGraph.Edges) = %d, want 2", len(rpt.TraceGraph.Edges))
	}
	if rpt.TraceGraph.Edges[0].Status != string(model.TraceEdgeStatusCandidate) {
		t.Fatalf("first edge status = %q, want candidate", rpt.TraceGraph.Edges[0].Status)
	}
	if rpt.TraceGraph.Edges[0].VerificationMode != string(model.TraceVerificationModeCandidateSearch) {
		t.Fatalf("first edge verification_mode = %q, want candidate_search", rpt.TraceGraph.Edges[0].VerificationMode)
	}
	if rpt.TraceGraph.Edges[1].Status != string(model.TraceEdgeStatusVerified) {
		t.Fatalf("second edge status = %q, want verified", rpt.TraceGraph.Edges[1].Status)
	}
	if rpt.TraceGraph.Edges[1].SourceEvidenceRef == nil || rpt.TraceGraph.Edges[1].SourceEvidenceRef.SectionID != "s1" {
		t.Fatalf("verified edge source evidence = %+v", rpt.TraceGraph.Edges[1].SourceEvidenceRef)
	}
	if len(rpt.TraceGraph.Paths) != 2 {
		t.Fatalf("len(rpt.TraceGraph.Paths) = %d, want 2", len(rpt.TraceGraph.Paths))
	}
	if rpt.TraceGraph.Paths[0].EntryNodeID != "e1" || rpt.TraceGraph.Paths[0].TerminalNodeID != "e1" {
		t.Fatalf("unexpected trace path: %+v", rpt.TraceGraph.Paths[0])
	}
	if len(rpt.TraceGaps) != 0 {
		t.Fatalf("len(rpt.TraceGaps) = %d, want 0", len(rpt.TraceGaps))
	}
	if len(rpt.DocumentViews) != 1 {
		t.Fatalf("len(rpt.DocumentViews) = %d, want 1", len(rpt.DocumentViews))
	}
	if rpt.DocumentViews[0].DocumentID != "d1" || rpt.DocumentViews[0].ClaimCount != 1 {
		t.Fatalf("unexpected document view: %+v", rpt.DocumentViews[0])
	}
	if len(rpt.DocumentViews[0].UpstreamDocuments) != 0 || len(rpt.DocumentViews[0].DownstreamDocuments) != 0 {
		t.Fatalf("unexpected single-document correspondence: %+v", rpt.DocumentViews[0])
	}
	if rpt.ReportModes.Default != "analyst" || rpt.ReportModes.DefaultTab != "summary" || rpt.ReportModes.CompareAvailable {
		t.Fatalf("unexpected report modes: %+v", rpt.ReportModes)
	}
	if len(rpt.ReportModes.Tabs) != 5 || rpt.ReportModes.Tabs[0].ID != "summary" || rpt.ReportModes.Tabs[4].ID != "diagnostics" {
		t.Fatalf("unexpected report tabs: %+v", rpt.ReportModes.Tabs)
	}
	if len(rpt.FindingsGrouped) != 1 {
		t.Fatalf("len(rpt.FindingsGrouped) = %d, want 1", len(rpt.FindingsGrouped))
	}
	if len(rpt.FindingsGrouped[0].DocumentPaths) != 1 || rpt.FindingsGrouped[0].DocumentPaths[0] != "/tmp/vision.md" {
		t.Fatalf("finding document_paths = %+v", rpt.FindingsGrouped[0].DocumentPaths)
	}
	if rpt.TrustSummary.Entities.Rejected != 2 {
		t.Fatalf("rejected entities = %d, want 2", rpt.TrustSummary.Entities.Rejected)
	}
	if len(rpt.Coverage.ByLevel) != 1 || rpt.Coverage.ByLevel[0].LevelID != "vision" {
		t.Fatalf("unexpected coverage block: %+v", rpt.Coverage)
	}
	if rpt.EvidencePack.Artifacts[0].Path != "/tmp/.strataudit/report.v2.json" {
		t.Fatalf("artifact path = %q", rpt.EvidencePack.Artifacts[0].Path)
	}
}

func TestBuildTraceGaps_ClassifiesWaterfallReasons(t *testing.T) {
	levels := []model.Level{
		{ID: "vision", Name: "Vision", Rank: 0},
		{ID: "strategy", Name: "Strategy", Rank: 1},
		{ID: "design", Name: "Design", Rank: 2},
		{ID: "implementation", Name: "Implementation", Rank: 3},
	}
	levelByID := map[string]model.Level{
		"vision":         levels[0],
		"strategy":       levels[1],
		"design":         levels[2],
		"implementation": levels[3],
	}
	documentByID := map[string]model.Document{
		"d_v": {ID: "d_v", Path: "/tmp/vision.md", LevelID: "vision"},
		"d_s": {ID: "d_s", Path: "/tmp/strategy.md", LevelID: "strategy"},
		"d_i": {ID: "d_i", Path: "/tmp/implementation.md", LevelID: "implementation"},
	}
	sectionByID := map[string]model.Section{
		"s_v": {ID: "s_v", DocumentID: "d_v", Heading: "Vision"},
		"s_s": {ID: "s_s", DocumentID: "d_s", Heading: "Strategy"},
		"s_i": {ID: "s_i", DocumentID: "d_i", Heading: "Implementation"},
	}
	entities := []model.Entity{
		{
			ID:               "v1",
			DocumentID:       "d_v",
			SectionID:        "s_v",
			LevelID:          "vision",
			Type:             model.EntityGoal,
			Title:            "North Star",
			SourceQuote:      "North Star",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(10),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "s_no",
			DocumentID:       "d_s",
			SectionID:        "s_s",
			LevelID:          "strategy",
			Type:             model.EntityObjective,
			Title:            "No Candidates",
			SourceQuote:      "No Candidates",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(13),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "s_quote",
			DocumentID:       "d_s",
			SectionID:        "s_s",
			LevelID:          "strategy",
			Type:             model.EntityObjective,
			Title:            "Quote Missing",
			SourceQuote:      "Quote Missing",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(13),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "s_reject",
			DocumentID:       "d_s",
			SectionID:        "s_s",
			LevelID:          "strategy",
			Type:             model.EntityObjective,
			Title:            "Rejected",
			SourceQuote:      "Rejected",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(8),
			TrustGrade:       model.TrustGradeVerified,
		},
		{
			ID:               "impl_missing",
			DocumentID:       "d_i",
			SectionID:        "s_i",
			LevelID:          "implementation",
			Type:             model.EntityTask,
			Title:            "Missing Upstream",
			SourceQuote:      "Missing Upstream",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(16),
			TrustGrade:       model.TrustGradeVerified,
		},
	}
	candidates := []model.Candidate{
		{
			ID:             "cand_quote",
			SourceEntityID: "s_quote",
			TargetEntityID: "v1",
			Similarity:     0.91,
			DiagnosticCode: string(model.TraceCandidateDiagnosticQuoteEvidenceMissing),
		},
		{
			ID:             "cand_reject",
			SourceEntityID: "s_reject",
			TargetEntityID: "v1",
			Similarity:     0.89,
			DiagnosticCode: string(model.TraceCandidateDiagnosticLLMVerificationRejected),
		},
	}

	gaps := buildTraceGaps(levels, entities, candidates, nil, documentByID, levelByID, sectionByID)
	if len(gaps) != 4 {
		t.Fatalf("len(gaps) = %d, want 4", len(gaps))
	}

	byEntity := make(map[string]reportpkg.TraceGapReport, len(gaps))
	for _, gap := range gaps {
		byEntity[gap.EntityID] = gap
	}

	if byEntity["s_no"].GapType != string(model.TraceGapTypeNoCandidates) || byEntity["s_no"].Stage != string(model.TraceGapStageCandidateSearch) {
		t.Fatalf("unexpected no-candidates gap: %+v", byEntity["s_no"])
	}
	if byEntity["s_quote"].GapType != string(model.TraceGapTypeQuoteEvidenceMissing) || byEntity["s_quote"].CandidateCount != 1 {
		t.Fatalf("unexpected quote-missing gap: %+v", byEntity["s_quote"])
	}
	if len(byEntity["s_quote"].TopCandidateIDs) != 1 || byEntity["s_quote"].TopCandidateIDs[0] != "cand_quote" {
		t.Fatalf("unexpected quote-missing top candidates: %+v", byEntity["s_quote"].TopCandidateIDs)
	}
	if byEntity["s_reject"].GapType != string(model.TraceGapTypeAllCandidatesRejected) || byEntity["s_reject"].Reason != string(model.TraceCandidateDiagnosticLLMVerificationRejected) {
		t.Fatalf("unexpected rejected gap: %+v", byEntity["s_reject"])
	}
	if byEntity["impl_missing"].GapType != string(model.TraceGapTypeMissingUpstreamEntities) || byEntity["impl_missing"].ExpectedToLevelID != "design" {
		t.Fatalf("unexpected missing-upstream gap: %+v", byEntity["impl_missing"])
	}
}

func TestBuildDocumentViews_AggregatesCorrespondenceAndBlockers(t *testing.T) {
	views := buildDocumentViews(
		[]reportpkg.DocumentReport{
			{ID: "d_v", Path: "/tmp/vision.md", Name: "vision.md", LevelID: "vision", LevelName: "Vision", EntityCount: 1},
			{ID: "d_s", Path: "/tmp/strategy.md", Name: "strategy.md", LevelID: "strategy", LevelName: "Strategy", EntityCount: 2},
			{ID: "d_d", Path: "/tmp/design.md", Name: "design.md", LevelID: "design", LevelName: "Design", EntityCount: 1},
		},
		[]reportpkg.LevelReport{
			{ID: "vision", Name: "Vision", Rank: 0},
			{ID: "strategy", Name: "Strategy", Rank: 1},
			{ID: "design", Name: "Design", Rank: 2},
		},
		reportpkg.TraceGraphReport{
			Nodes: []reportpkg.TraceNodeReport{
				{ID: "v1", EntityID: "v1", Title: "Vision Claim", LevelID: "vision", LevelName: "Vision", DocumentID: "d_v", DocumentPath: "/tmp/vision.md"},
				{ID: "s1", EntityID: "s1", Title: "Strategy Link", LevelID: "strategy", LevelName: "Strategy", DocumentID: "d_s", DocumentPath: "/tmp/strategy.md"},
				{ID: "s2", EntityID: "s2", Title: "Strategy Gap", LevelID: "strategy", LevelName: "Strategy", DocumentID: "d_s", DocumentPath: "/tmp/strategy.md"},
				{ID: "d1", EntityID: "d1", Title: "Design Claim", LevelID: "design", LevelName: "Design", DocumentID: "d_d", DocumentPath: "/tmp/design.md"},
			},
			Edges: []reportpkg.TraceEdgeReport{
				{ID: "tr_vs", SourceNodeID: "s1", TargetNodeID: "v1", SourceEntityID: "s1", TargetEntityID: "v1", Status: "verified"},
				{ID: "cand_ds", SourceNodeID: "d1", TargetNodeID: "s1", SourceEntityID: "d1", TargetEntityID: "s1", Status: "candidate"},
				{ID: "rej_ds", SourceNodeID: "d1", TargetNodeID: "s2", SourceEntityID: "d1", TargetEntityID: "s2", Status: "rejected"},
			},
		},
		[]reportpkg.TraceGapReport{
			{
				ID:                "gap_s2",
				NodeID:            "s2",
				EntityID:          "s2",
				DocumentID:        "d_s",
				DocumentPath:      "/tmp/strategy.md",
				LevelID:           "strategy",
				ExpectedToLevelID: "vision",
				Stage:             "verification",
				GapType:           "all_candidates_rejected",
			},
			{
				ID:                "gap_d1",
				NodeID:            "d1",
				EntityID:          "d1",
				DocumentID:        "d_d",
				DocumentPath:      "/tmp/design.md",
				LevelID:           "design",
				ExpectedToLevelID: "strategy",
				Stage:             "verification",
				GapType:           "low_confidence",
			},
		},
		[]reportpkg.CorpusQualityDocReport{
			{
				DocumentID: "d_s",
				Severity:   "critical",
				Flags:      []string{"language_mismatch", "html_export_noise"},
			},
		},
	)

	if len(views) != 3 {
		t.Fatalf("len(views) = %d, want 3", len(views))
	}

	byDocID := make(map[string]reportpkg.DocumentViewReport, len(views))
	for _, view := range views {
		byDocID[view.DocumentID] = view
	}

	strategy := byDocID["d_s"]
	if strategy.ClaimCount != 2 {
		t.Fatalf("strategy claim_count = %d, want 2", strategy.ClaimCount)
	}
	if strategy.VerifiedLinkCount != 1 || strategy.CandidateLinkCount != 1 || strategy.RejectedLinkCount != 1 {
		t.Fatalf("unexpected strategy link counts: %+v", strategy)
	}
	if strategy.BlockerCount != 1 || strategy.BrokenLinkCount != 1 {
		t.Fatalf("unexpected strategy blocker counts: %+v", strategy)
	}
	if len(strategy.UpstreamDocuments) != 1 || strategy.UpstreamDocuments[0].DocumentID != "d_v" || strategy.UpstreamDocuments[0].VerifiedEdgeCount != 1 {
		t.Fatalf("unexpected strategy upstream docs: %+v", strategy.UpstreamDocuments)
	}
	if len(strategy.DownstreamDocuments) != 1 || strategy.DownstreamDocuments[0].DocumentID != "d_d" {
		t.Fatalf("unexpected strategy downstream docs: %+v", strategy.DownstreamDocuments)
	}
	if strategy.DownstreamDocuments[0].CandidateEdgeCount != 1 || strategy.DownstreamDocuments[0].RejectedEdgeCount != 1 {
		t.Fatalf("unexpected strategy downstream counts: %+v", strategy.DownstreamDocuments[0])
	}
	if len(strategy.Blockers) != 1 || strategy.Blockers[0].GapType != "all_candidates_rejected" || strategy.Blockers[0].Count != 1 {
		t.Fatalf("unexpected strategy blockers: %+v", strategy.Blockers)
	}
	if len(strategy.CriticalQualityFlags) != 1 || strategy.CriticalQualityFlags[0] != "language_mismatch" {
		t.Fatalf("unexpected strategy critical quality flags: %+v", strategy.CriticalQualityFlags)
	}

	design := byDocID["d_d"]
	if design.CandidateLinkCount != 1 || design.RejectedLinkCount != 1 || design.BlockerCount != 1 {
		t.Fatalf("unexpected design counts: %+v", design)
	}
	if len(design.UpstreamDocuments) != 1 || design.UpstreamDocuments[0].DocumentID != "d_s" {
		t.Fatalf("unexpected design upstream docs: %+v", design.UpstreamDocuments)
	}
	if len(design.DownstreamDocuments) != 0 {
		t.Fatalf("unexpected design downstream docs: %+v", design.DownstreamDocuments)
	}

	vision := byDocID["d_v"]
	if vision.VerifiedLinkCount != 1 || len(vision.DownstreamDocuments) != 1 || vision.DownstreamDocuments[0].DocumentID != "d_s" {
		t.Fatalf("unexpected vision correspondence: %+v", vision)
	}
	if len(vision.UpstreamDocuments) != 0 {
		t.Fatalf("unexpected vision upstream docs: %+v", vision.UpstreamDocuments)
	}
}
