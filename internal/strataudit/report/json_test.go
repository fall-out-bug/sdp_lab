package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteJSON_WritesV2AndCompatAlias(t *testing.T) {
	dir := t.TempDir()
	rpt := &AuditReport{
		SchemaVersion: "2",
		AuditScope: AuditScopeReport{
			ProjectName:        "sample-project",
			ProjectDescription: "sample audit",
			OutputDir:          ".strataudit",
			OutputLang:         "ru",
			GeneratedAt:        "2026-04-13T00:00:00Z",
			SourceDirs:         []string{"docs/strategy"},
			Exclude:            []string{".DS_Store"},
			Models: AuditModelsReport{
				DefaultModel:   "deepseek/deepseek-v3.2",
				ExtractModel:   "deepseek/deepseek-v3.2",
				EmbeddingModel: "openai/text-embedding-3-small",
			},
			Thresholds: AuditThresholdsReport{
				Similarity:           0.5,
				TraceConfidence:      0.6,
				AutoVerifySimilarity: 0.85,
				CoverageWarn:         70,
				LLMVerifyBudget:      50,
			},
		},
		TrustSummary: TrustSummaryReport{
			OverallStatus: "warning",
			Entities: EntityTrustCounts{
				TotalAdmitted: 3,
				Verified:      2,
				Suspect:       1,
				Rejected:      4,
			},
			Traces: TraceTrustCounts{
				Verified:   1,
				Candidates: 2,
				VerificationModes: map[string]int{
					"llm_evidence": 1,
				},
			},
			Findings: FindingCountReport{
				Total:    2,
				Critical: 1,
				Warn:     1,
			},
			Disclaimers: []string{"Есть suspect-сущн.; часть выводов требует ручной проверки."},
		},
		CorpusQuality: CorpusQualityReport{
			TotalIssues:       2,
			CriticalDocuments: 1,
			FlagCounts: map[string]int{
				"language_mismatch": 1,
				"prompt_leak":       1,
			},
			Documents: []CorpusQualityDocReport{
				{
					DocumentID:   "d1",
					DocumentPath: "/tmp/vision.md",
					LevelID:      "vision",
					LevelName:    "Vision",
					Severity:     "critical",
					IssueCount:   2,
					Flags:        []string{"language_mismatch", "prompt_leak"},
					FindingIDs:   []string{"f1"},
					SectionIDs:   []string{"s1"},
					EntityIDs:    []string{"e1"},
				},
			},
		},
		Levels: []LevelReport{
			{
				ID:            "vision",
				Name:          "Vision",
				Rank:          0,
				Description:   "Vision layer",
				Patterns:      []string{"*vision*"},
				EntityCount:   1,
				DocumentCount: 1,
			},
		},
		Documents: []DocumentReport{
			{
				ID:           "d1",
				Path:         "/tmp/vision.md",
				Name:         "vision.md",
				LevelID:      "vision",
				LevelName:    "Vision",
				ContentHash:  "hash-doc",
				Version:      1,
				EntityCount:  1,
				SectionCount: 1,
			},
		},
		Sections: []SectionReport{
			{
				ID:           "s1",
				DocumentID:   "d1",
				DocumentPath: "/tmp/vision.md",
				LevelID:      "vision",
				LevelName:    "Vision",
				Ordinal:      0,
				Heading:      "Введение",
				Label:        "vision.md#Введение",
				CharStart:    0,
				CharEnd:      17,
				Preview:      "Стратегия лидера",
				EntityCount:  1,
				QualityFlags: []string{"prompt_leak"},
			},
		},
		Entities: []EntityReport{
			{
				ID:             "e1",
				Type:           "goal",
				Title:          "Стратегия лидера",
				TitleOriginal:  "Стратегия лидера",
				Lang:           "ru",
				LevelID:        "vision",
				LevelName:      "Vision",
				DocumentID:     "d1",
				DocumentPath:   "/tmp/vision.md",
				SectionID:      "s1",
				SectionHeading: "Введение",
				SourceQuote:    "Стратегия лидера",
				TrustGrade:     "verified",
				QualityFlags:   []string{"quote_verified"},
			},
		},
		TraceCandidates: []TraceCandidateReport{
			{
				ID:             "cand1",
				SourceEntityID: "e1",
				TargetEntityID: "e1",
				Similarity:     0.81,
				DiagnosticCode: "embedding_similarity_candidate",
			},
		},
		VerifiedTraces: []VerifiedTraceReport{
			{
				ID:               "tr1",
				SourceEntityID:   "e1",
				TargetEntityID:   "e1",
				Relation:         "contributes_to",
				Confidence:       0.92,
				SimilarityScore:  0.81,
				Justification:    "Evidence-backed relation.",
				VerificationMode: "llm_evidence",
				TrustGrade:       "verified",
				TraceEvidence: TraceEvidenceReport{
					Source: EvidenceRefReport{
						DocumentID:     "d1",
						DocumentPath:   "/tmp/vision.md",
						SectionID:      "s1",
						SectionHeading: "Введение",
						Quote:          "Стратегия лидера",
						TrustGrade:     "verified",
					},
					Target: EvidenceRefReport{
						DocumentID:     "d1",
						DocumentPath:   "/tmp/vision.md",
						SectionID:      "s1",
						SectionHeading: "Введение",
						Quote:          "Стратегия лидера",
						TrustGrade:     "verified",
					},
				},
			},
		},
		TraceGraph: TraceGraphReport{
			Nodes: []TraceNodeReport{
				{
					ID:             "e1",
					EntityID:       "e1",
					Type:           "goal",
					Title:          "Стратегия лидера",
					LevelID:        "vision",
					LevelName:      "Vision",
					DocumentID:     "d1",
					DocumentPath:   "/tmp/vision.md",
					SectionID:      "s1",
					SectionHeading: "Введение",
					SourceQuote:    "Стратегия лидера",
					TrustGrade:     "verified",
					Lang:           "ru",
				},
			},
			Edges: []TraceEdgeReport{
				{
					ID:               "cand1",
					SourceNodeID:     "e1",
					TargetNodeID:     "e1",
					SourceEntityID:   "e1",
					TargetEntityID:   "e1",
					Status:           "candidate",
					VerificationMode: "candidate_search",
					Confidence:       0,
					Similarity:       0.81,
					Reason:           "embedding_similarity_candidate",
					SourceEvidenceRef: &EvidenceRefReport{
						DocumentID:     "d1",
						DocumentPath:   "/tmp/vision.md",
						SectionID:      "s1",
						SectionHeading: "Введение",
						Quote:          "Стратегия лидера",
						TrustGrade:     "verified",
					},
					TargetEvidenceRef: &EvidenceRefReport{
						DocumentID:     "d1",
						DocumentPath:   "/tmp/vision.md",
						SectionID:      "s1",
						SectionHeading: "Введение",
						Quote:          "Стратегия лидера",
						TrustGrade:     "verified",
					},
				},
				{
					ID:               "tr1",
					SourceNodeID:     "e1",
					TargetNodeID:     "e1",
					SourceEntityID:   "e1",
					TargetEntityID:   "e1",
					Relation:         "contributes_to",
					Direction:        "up",
					Status:           "verified",
					VerificationMode: "llm_evidence",
					Confidence:       0.92,
					Similarity:       0.81,
					Reason:           "Evidence-backed relation.",
					TrustGrade:       "verified",
					SourceEvidenceRef: &EvidenceRefReport{
						DocumentID:     "d1",
						DocumentPath:   "/tmp/vision.md",
						SectionID:      "s1",
						SectionHeading: "Введение",
						Quote:          "Стратегия лидера",
						TrustGrade:     "verified",
					},
					TargetEvidenceRef: &EvidenceRefReport{
						DocumentID:     "d1",
						DocumentPath:   "/tmp/vision.md",
						SectionID:      "s1",
						SectionHeading: "Введение",
						Quote:          "Стратегия лидера",
						TrustGrade:     "verified",
					},
				},
			},
			Paths: []TracePathReport{
				{
					ID:             "cand1",
					EntryNodeID:    "e1",
					TerminalNodeID: "e1",
					NodeIDs:        []string{"e1", "e1"},
					EdgeIDs:        []string{"cand1"},
					Status:         "candidate",
					HopCount:       1,
				},
				{
					ID:             "tr1",
					EntryNodeID:    "e1",
					TerminalNodeID: "e1",
					NodeIDs:        []string{"e1", "e1"},
					EdgeIDs:        []string{"tr1"},
					Status:         "verified",
					HopCount:       1,
				},
			},
		},
		TraceGaps: []TraceGapReport{
			{
				ID:                  "gap1",
				NodeID:              "e1",
				EntityID:            "e1",
				Title:               "Стратегия лидера",
				LevelID:             "vision",
				LevelName:           "Vision",
				DocumentID:          "d1",
				DocumentPath:        "/tmp/vision.md",
				SectionID:           "s1",
				SectionHeading:      "Введение",
				SourceQuote:         "Стратегия лидера",
				ExpectedToLevelID:   "strategy",
				ExpectedToLevelName: "Strategy",
				Stage:               "verification",
				GapType:             "all_candidates_rejected",
				Reason:              "llm_verification_rejected",
				CandidateCount:      1,
				TopCandidateIDs:     []string{"cand1"},
			},
		},
		DocumentViews: []DocumentViewReport{
			{
				DocumentID:          "d1",
				DocumentPath:        "/tmp/vision.md",
				DocumentName:        "vision.md",
				LevelID:             "vision",
				LevelName:           "Vision",
				ClaimCount:          1,
				VerifiedLinkCount:   0,
				CandidateLinkCount:  0,
				RejectedLinkCount:   0,
				BrokenLinkCount:     1,
				BlockerCount:        1,
				UpstreamDocuments:   []DocumentCorrespondenceReport{},
				DownstreamDocuments: []DocumentCorrespondenceReport{},
				Blockers: []DocumentBlockerReport{
					{
						Stage:    "verification",
						GapType:  "all_candidates_rejected",
						Count:    1,
						ClaimIDs: []string{"e1"},
					},
				},
				CriticalQualityFlags: []string{"language_mismatch", "prompt_leak"},
				KeyClaimIDs:          []string{"e1"},
			},
		},
		ReportModes: ReportModesReport{
			Default:          "analyst",
			DefaultTab:       "summary",
			CompareAvailable: false,
			Tabs: []ReportTabReport{
				{ID: "summary", Label: "Сводка"},
				{ID: "documents", Label: "Документы"},
				{ID: "trace", Label: "Трассировка"},
				{ID: "gaps", Label: "Разрывы"},
				{ID: "diagnostics", Label: "Диагностика"},
			},
		},
		FindingsGrouped: []FindingReport{
			{
				ID:             "f1",
				Type:           "corpus_quality_cluster",
				Severity:       "critical",
				ClusterKey:     "corpus_quality:vision:d1",
				Title:          "Качество корпуса",
				Description:    "Документ содержит проблемы качества.",
				Recommendation: "Очистить источник.",
				EntityIDs:      []string{"e1"},
				DocumentIDs:    []string{"d1"},
				DocumentPaths:  []string{"/tmp/vision.md"},
				SectionIDs:     []string{"s1"},
				SectionLabels:  []string{"vision.md#Введение"},
				AffectedCount:  1,
				Confidence:     1,
			},
		},
		Coverage: CoverageBlockReport{
			AverageLevelPct: 75,
			ByLevel: []CoverageScopeReport{
				{
					ScopeType:  "level",
					ScopeID:    "vision",
					ScopeLabel: "Vision",
					LevelID:    "vision",
					LevelName:  "Vision",
					Total:      4,
					Traced:     3,
					Pct:        75,
				},
			},
			ByDocument: []CoverageScopeReport{
				{
					ScopeType:    "document",
					ScopeID:      "d1",
					ScopeLabel:   "vision.md",
					LevelID:      "vision",
					LevelName:    "Vision",
					DocumentID:   "d1",
					DocumentPath: "/tmp/vision.md",
					Total:        2,
					Traced:       1,
					Pct:          50,
				},
			},
			BySection: []CoverageScopeReport{
				{
					ScopeType:      "section",
					ScopeID:        "s1",
					ScopeLabel:     "vision.md#Введение",
					LevelID:        "vision",
					LevelName:      "Vision",
					DocumentID:     "d1",
					DocumentPath:   "/tmp/vision.md",
					SectionID:      "s1",
					SectionHeading: "Введение",
					Total:          1,
					Traced:         1,
					Pct:            100,
				},
			},
			LowestDocuments: []CoverageScopeReport{
				{
					ScopeType:    "document",
					ScopeID:      "d1",
					ScopeLabel:   "vision.md",
					LevelID:      "vision",
					LevelName:    "Vision",
					DocumentID:   "d1",
					DocumentPath: "/tmp/vision.md",
					Total:        2,
					Traced:       1,
					Pct:          50,
				},
			},
			LowestSections: []CoverageScopeReport{
				{
					ScopeType:      "section",
					ScopeID:        "s1",
					ScopeLabel:     "vision.md#Введение",
					LevelID:        "vision",
					LevelName:      "Vision",
					DocumentID:     "d1",
					DocumentPath:   "/tmp/vision.md",
					SectionID:      "s1",
					SectionHeading: "Введение",
					Total:          1,
					Traced:         1,
					Pct:            100,
				},
			},
		},
		EvidencePack: EvidencePackReport{
			Artifacts: []ArtifactReport{
				{Kind: "json_v2", Path: "/tmp/report.v2.json"},
				{Kind: "html", Path: "/tmp/report.html"},
			},
			DocumentCount:              1,
			TraceCandidateCount:        1,
			RejectedEntityCount:        4,
			EntitiesWithSourceQuotes:   1,
			VerifiedTracesWithEvidence: 1,
		},
	}

	if err := WriteJSON(rpt, dir); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	v2Bytes, err := os.ReadFile(filepath.Join(dir, "report.v2.json"))
	if err != nil {
		t.Fatalf("ReadFile(report.v2.json): %v", err)
	}
	compatBytes, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatalf("ReadFile(report.json): %v", err)
	}
	goldenBytes, err := os.ReadFile(filepath.Join("testdata", "report_v2.golden.json"))
	if err != nil {
		t.Fatalf("ReadFile(golden): %v", err)
	}
	if strings.TrimSpace(string(v2Bytes)) != strings.TrimSpace(string(goldenBytes)) {
		t.Fatalf("report.v2.json mismatch\n--- got ---\n%s\n--- want ---\n%s", string(v2Bytes), string(goldenBytes))
	}
	if strings.TrimSpace(string(compatBytes)) != strings.TrimSpace(string(v2Bytes)) {
		t.Fatalf("report.json compatibility alias mismatch\n--- compat ---\n%s\n--- v2 ---\n%s", string(compatBytes), string(v2Bytes))
	}

	var got AuditReport
	if err := json.Unmarshal(v2Bytes, &got); err != nil {
		t.Fatalf("Unmarshal(report.v2.json): %v", err)
	}
	if got.ReportModes.Default != "analyst" || got.ReportModes.DefaultTab != "summary" || got.ReportModes.CompareAvailable {
		t.Fatalf("unexpected report_modes: %+v", got.ReportModes)
	}
	if len(got.ReportModes.Tabs) != 5 {
		t.Fatalf("len(report_modes.tabs) = %d, want 5", len(got.ReportModes.Tabs))
	}
	if len(got.TraceGaps) != 1 || got.TraceGaps[0].Reason == "" || len(got.TraceGaps[0].TopCandidateIDs) != 1 {
		t.Fatalf("trace_gaps lost explainability: %+v", got.TraceGaps)
	}
	if len(got.DocumentViews) != 1 || got.DocumentViews[0].BlockerCount != 1 || len(got.DocumentViews[0].KeyClaimIDs) != 1 {
		t.Fatalf("document_views lost blocker/key-claim contract: %+v", got.DocumentViews)
	}
}
