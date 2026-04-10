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

	var totalEntities int
	for _, l := range levels {
		count, _ := store.CountEntitiesByLevel(ctx, l.ID)
		rpt.Levels = append(rpt.Levels, report.LevelReport{Name: l.Name, Rank: l.Rank, Entities: int(count)})
		totalEntities += int(count)
	}

	coverages, _ := store.CoverageByLevel(ctx)
	var avgCov float64
	for _, c := range coverages {
		rpt.Coverage = append(rpt.Coverage, report.CoverageReport{
			Level: c.LevelID, Total: c.TotalEntities, Traced: c.TracedEntities, Pct: c.CoveragePct,
		})
		avgCov += c.CoveragePct
	}
	if len(coverages) > 0 {
		avgCov /= float64(len(coverages))
	}

	allTypes := []model.FindingType{
		model.FindingGap, model.FindingOrphan, model.FindingAlignment,
		model.FindingCoverage, model.FindingConflict, model.FindingUnknownRationale,
		model.FindingWeakLink, model.FindingStale, model.FindingInferredStrategy,
		model.FindingShadowStrategy, model.FindingStrongTrace, model.FindingAmbiguousTrace,
	}

	var crit, warn int
	for _, ft := range allTypes {
		findings, _ := store.FindingsByType(ctx, ft, model.Page{Limit: 10000})
		for _, f := range findings {
			rpt.Findings = append(rpt.Findings, report.FindingReport{
				ID: f.ID, Type: string(f.Type), Severity: string(f.Severity),
				Title: f.Title, Description: f.Description,
				EntityIDs: f.EntityIDs, Confidence: f.ConfidenceScore,
			})
			if f.Severity == model.SeverityCritical {
				crit++
			} else if f.Severity == model.SeverityWarn {
				warn++
			}
		}
	}

	rpt.Summary = report.SummaryReport{
		TotalEntities: totalEntities, TotalFindings: len(rpt.Findings),
		CriticalCount: crit, WarnCount: warn, AvgCoverage: avgCov,
	}

	return rpt, nil
}
