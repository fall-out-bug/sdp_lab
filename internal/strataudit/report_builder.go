package strataudit

import (
	"context"
	"fmt"

	"sdp_dev/internal/strataudit/model"
	"sdp_dev/internal/strataudit/report"
)

// BuildReport gathers data from store and builds a report.AuditReport.
func BuildReport(ctx context.Context, cfg *Config, store *SQLiteStore) (*report.AuditReport, error) {
	rpt := &report.AuditReport{Project: cfg.Project.Name}

	levels, err := store.LoadLevels(ctx)
	if err != nil {
		return nil, fmt.Errorf("load levels: %w", err)
	}
	documents, err := store.AllDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("load documents: %w", err)
	}
	sections, err := store.AllSections(ctx)
	if err != nil {
		return nil, fmt.Errorf("load sections: %w", err)
	}

	var totalEntities int
	for _, l := range levels {
		count, _ := store.CountEntitiesByLevel(ctx, l.ID)
		rpt.Levels = append(rpt.Levels, report.LevelReport{Name: l.Name, Rank: l.Rank, Entities: int(count)})
		totalEntities += int(count)
	}
	for _, doc := range documents {
		rpt.Documents = append(rpt.Documents, report.DocumentReport{
			ID: doc.ID, Path: doc.Path, LevelID: doc.LevelID, ContentHash: doc.ContentHash, Version: doc.Version,
		})
	}
	for _, section := range sections {
		rpt.Sections = append(rpt.Sections, report.SectionReport{
			ID: section.ID, DocumentID: section.DocumentID, Ordinal: section.Ordinal, Heading: section.Heading,
			CharStart: section.CharStart, CharEnd: section.CharEnd, Preview: section.Preview, QualityFlags: section.QualityFlags,
		})
	}

	coverages, _ := store.AllCoverage(ctx)
	var avgCov float64
	var levelCoverageCount int
	for _, c := range coverages {
		rpt.Coverage = append(rpt.Coverage, report.CoverageReport{
			ScopeType: string(c.ScopeType), ScopeID: c.ScopeID, ScopeLabel: c.ScopeLabel,
			LevelID: c.LevelID, DocumentID: c.DocumentID, SectionID: c.SectionID,
			Level: c.LevelID, Total: c.TotalEntities, Traced: c.TracedEntities, Pct: c.CoveragePct,
		})
		if c.ScopeType == model.CoverageScopeLevel {
			avgCov += c.CoveragePct
			levelCoverageCount++
		}
	}
	if levelCoverageCount > 0 {
		avgCov /= float64(levelCoverageCount)
	}

	var crit, warn int
	findings, _ := store.AllFindings(ctx, model.Page{Limit: 10000})
	for _, f := range findings {
		rpt.Findings = append(rpt.Findings, report.FindingReport{
			ID: f.ID, Type: string(f.Type), Severity: string(f.Severity), ClusterKey: f.ClusterKey,
			Title: f.Title, Description: f.Description,
			EntityIDs: f.EntityIDs, DocumentIDs: f.DocumentIDs, SectionIDs: f.SectionIDs, Confidence: f.ConfidenceScore,
		})
		switch f.Severity {
		case model.SeverityCritical:
			crit++
		case model.SeverityWarn:
			warn++
		}
	}

	rpt.Summary = report.SummaryReport{
		TotalEntities: totalEntities, TotalFindings: len(rpt.Findings),
		CriticalCount: crit, WarnCount: warn, AvgCoverage: avgCov,
	}

	// Load entities per level
	for _, l := range levels {
		entities, _ := store.EntitiesByLevel(ctx, l.ID, model.Page{Limit: 10000})
		for _, e := range entities {
			rpt.Entities = append(rpt.Entities, report.EntityReport{
				ID: e.ID, Type: string(e.Type), Title: e.Title,
				Description: e.Description, TitleOriginal: e.TitleOriginal,
				DescriptionOriginal: e.DescriptionOriginal, Lang: e.Lang,
				LanguageMismatch: e.LanguageMismatch, LevelID: e.LevelID, DocumentID: e.DocumentID,
				SectionID: e.SectionID, SourceQuote: e.SourceQuote, QuoteStartOffset: e.QuoteStartOffset,
				QuoteEndOffset: e.QuoteEndOffset, TrustGrade: string(e.TrustGrade), QualityFlags: e.QualityFlags,
			})
		}
	}

	candidates, _ := store.AllCandidates(ctx)
	for _, candidate := range candidates {
		rpt.TraceCandidates = append(rpt.TraceCandidates, report.TraceCandidateReport{
			ID: candidate.ID, SourceEntityID: candidate.SourceEntityID, TargetEntityID: candidate.TargetEntityID,
			Similarity: candidate.Similarity, Verified: candidate.Verified, TraceID: candidate.TraceID, DiagnosticCode: candidate.DiagnosticCode,
		})
	}

	// Load all verified traces
	traces, _ := store.AllTraces(ctx)
	for _, t := range traces {
		rpt.VerifiedTraces = append(rpt.VerifiedTraces, report.VerifiedTraceReport{
			ID: t.ID, SourceEntityID: t.SourceEntityID, TargetEntityID: t.TargetEntityID,
			Relation: string(t.Relation), Confidence: t.Confidence, SimilarityScore: t.SimilarityScore,
			Justification: t.Justification, VerificationMode: string(t.VerificationMode), TrustGrade: string(t.TrustGrade),
			SourceSectionID: t.SourceSectionID, TargetSectionID: t.TargetSectionID,
			SourceQuoteStartOffset: t.SourceQuoteStartOffset, SourceQuoteEndOffset: t.SourceQuoteEndOffset,
			TargetQuoteStartOffset: t.TargetQuoteStartOffset, TargetQuoteEndOffset: t.TargetQuoteEndOffset,
		})
	}

	return rpt, nil
}
