package strataudit

import (
	"context"
	"fmt"
	"time"

	"sdp_dev/internal/strataudit/model"
)

// AnalyzeResult holds analysis statistics.
type AnalyzeResult struct {
	Findings int
	Errors   []error
}

// Analyze runs all finding detectors and produces findings.
func Analyze(ctx context.Context, cfg *Config, store *SQLiteStore) (*AnalyzeResult, error) {
	result := &AnalyzeResult{}
	now := time.Now()

	levels, err := store.LoadLevels(ctx)
	if err != nil {
		return nil, fmt.Errorf("load levels: %w", err)
	}
	if len(levels) < 2 {
		return result, nil
	}

	var allFindings []model.Finding
	findingIdx := 0

	for i := 0; i < len(levels)-1; i++ {
		upper := levels[i]
		lower := levels[i+1]

		upperEntities, _ := store.EntitiesByLevel(ctx, upper.ID, model.Page{Limit: 10000})
		lowerEntities, _ := store.EntitiesByLevel(ctx, lower.ID, model.Page{Limit: 10000})

		upperTraced := tracedEntityIDs(ctx, store, upper.ID)
		lowerTraced := tracedEntityIDs(ctx, store, lower.ID)

		// Detect gaps: upper-level entities with no traces to lower level
		for _, e := range upperEntities {
			if !upperTraced[e.ID] {
				// Severity by rank: critical if rank <= 1, warn otherwise
				sev := model.SeverityCritical
				if upper.Rank > 1 {
					sev = model.SeverityWarn
				}
				findingIdx++
				allFindings = append(allFindings, model.Finding{
					ID:            findingID("gap", findingIdx),
					Type:          model.FindingGap,
					Severity:      sev,
					EntityIDs:     []string{e.ID},
					Title:         fmt.Sprintf("Gap: %q has no support from %s", e.Title, lower.Name),
					Description:   fmt.Sprintf("Entity %q at level %s (rank %d) has no traced contributions from level %s.", e.Title, upper.Name, upper.Rank, lower.Name),
					Recommendation: fmt.Sprintf("Add operational entities at %s level that contribute to this goal.", lower.Name),
				})
			} else {
				// Check for strong traces and weak links
				traces, _ := store.TracesForEntity(ctx, e.ID)
				for _, tr := range traces {
					if tr.Relation != model.RelationContributesTo {
						continue
					}
					findingIdx++
					if tr.Confidence >= 0.85 {
						allFindings = append(allFindings, model.Finding{
							ID:            findingID("strong_trace", findingIdx),
							Type:          model.FindingStrongTrace,
							Severity:      model.SeverityInfo,
							EntityIDs:     []string{tr.SourceEntityID, tr.TargetEntityID},
							Title:         fmt.Sprintf("Strong trace: %.0f%% confidence to %q", tr.Confidence*100, e.Title),
							Description:   fmt.Sprintf("Entity contributes to %q with %.2f confidence.", e.Title, tr.Confidence),
							LLMScore:      model.LLMScoreHigh,
							ConfidenceScore: tr.Confidence,
						})
					} else if tr.Confidence >= 0.8 {
						allFindings = append(allFindings, model.Finding{
							ID:            findingID("alignment", findingIdx),
							Type:          model.FindingAlignment,
							Severity:      model.SeverityInfo,
							EntityIDs:     []string{tr.SourceEntityID, tr.TargetEntityID},
							Title:         fmt.Sprintf("Alignment: trace to %q", e.Title),
							Description:   fmt.Sprintf("Entity at %s level contributes to %q with %.0f%% confidence.", lower.Name, e.Title, tr.Confidence*100),
							LLMScore:      model.LLMScoreHigh,
							ConfidenceScore: tr.Confidence,
						})
					} else if tr.Confidence < cfg.Thresholds.TraceConfidence+0.1 {
						allFindings = append(allFindings, model.Finding{
							ID:            findingID("weak_link", findingIdx),
							Type:          model.FindingWeakLink,
							Severity:      model.SeverityWarn,
							EntityIDs:     []string{tr.SourceEntityID, tr.TargetEntityID},
							Title:         fmt.Sprintf("Weak link: %.0f%% confidence to %q", tr.Confidence*100, e.Title),
							Description:   fmt.Sprintf("Trace from %s to %q has low confidence (%.2f). Verify manually.", lower.Name, e.Title, tr.Confidence),
							LLMScore:      model.LLMScoreLow,
							ConfidenceScore: tr.Confidence,
						})
					}
					break // one trace summary per entity
				}

				// Check for ambiguous traces (multiple traces with close confidence)
				if len(traces) >= 2 {
					sorted := sortTracesByConfidence(traces)
					if sorted[0].Confidence-sorted[1].Confidence < 0.15 {
						findingIdx++
						allFindings = append(allFindings, model.Finding{
							ID:        findingID("ambiguous", findingIdx),
							Type:      model.FindingAmbiguousTrace,
							Severity:  model.SeverityWarn,
							EntityIDs: []string{sorted[0].TargetEntityID, sorted[1].TargetEntityID},
							Title:     fmt.Sprintf("Ambiguous trace: %q has multiple close candidates", e.Title),
							Description: fmt.Sprintf("Top-2 confidence delta is %.2f (< 0.15). Cannot determine primary trace.", sorted[0].Confidence-sorted[1].Confidence),
						})
					}
				}
			}
		}

		// Detect orphans: lower-level entities with no traces to upper level
		for _, e := range lowerEntities {
			if !lowerTraced[e.ID] {
				findingIdx++
				allFindings = append(allFindings, model.Finding{
					ID:            findingID("orphan", findingIdx),
					Type:          model.FindingOrphan,
					Severity:      model.SeverityWarn,
					EntityIDs:     []string{e.ID},
					Title:         fmt.Sprintf("Orphan: %q has no link to %s", e.Title, upper.Name),
					Description:   fmt.Sprintf("Entity %q at level %s is not traced to any entity at level %s.", e.Title, lower.Name, upper.Name),
					Recommendation: "Verify if this entity supports a strategic goal or remove/deprioritize it.",
				})
			}
		}
	}

	// Compute confidence for all findings that need it
	for i := range allFindings {
		f := &allFindings[i]
		if f.ConfidenceScore == 0 {
			f.ComputeConfidence()
		}
	}

	// Compute coverage per level
	for _, level := range levels {
		total, _ := store.CountEntitiesByLevel(ctx, level.ID)
		if total == 0 {
			continue
		}
		traced := tracedEntityIDs(ctx, store, level.ID)
		tracedCount := int64(len(traced))
		pct := float64(tracedCount) / float64(total) * 100

		allFindings = append(allFindings, model.Finding{
			ID:             findingID("coverage_"+level.ID, 0),
			Type:           model.FindingCoverage,
			Severity:       coverageSeverity(pct, cfg.Thresholds.CoverageWarn),
			Title:          fmt.Sprintf("%s coverage: %.0f%% (%d/%d)", level.Name, pct, tracedCount, total),
			Description:    fmt.Sprintf("Level %s has %d entities, %d traced (%.1f%% coverage).", level.Name, total, tracedCount, pct),
			ConfidenceScore: 1.0,
		})

		if err := store.SaveCoverage(ctx, []model.Coverage{{
			ID:             fmt.Sprintf("cov_%s", level.ID),
			LevelID:        level.ID,
			TotalEntities:  int(total),
			TracedEntities: int(tracedCount),
			CoveragePct:    pct,
			ComputedAt:     now,
		}}); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save coverage %s: %w", level.ID, err))
		}
	}

	if len(allFindings) > 0 {
		if err := store.SaveFindings(ctx, allFindings); err != nil {
			return nil, fmt.Errorf("save findings: %w", err)
		}
	}

	result.Findings = len(allFindings)
	return result, nil
}

func tracedEntityIDs(ctx context.Context, store *SQLiteStore, levelID string) map[string]bool {
	entities, _ := store.EntitiesByLevel(ctx, levelID, model.Page{Limit: 10000})
	traced := make(map[string]bool)
	for _, e := range entities {
		traces, err := store.TracesForEntity(ctx, e.ID)
		if err == nil && len(traces) > 0 {
			traced[e.ID] = true
		}
	}
	return traced
}

func coverageSeverity(pct, warnThreshold float64) model.Severity {
	if pct >= warnThreshold {
		return model.SeverityInfo
	}
	if pct >= warnThreshold/2 {
		return model.SeverityWarn
	}
	return model.SeverityCritical
}

func findingID(prefix string, idx int) string {
	return fmt.Sprintf("%s_%d_%s", prefix, idx, sha256Hash([]byte(fmt.Sprintf("%s%d", prefix, idx)))[:6])
}

func sortTracesByConfidence(traces []model.Trace) []model.Trace {
	sorted := make([]model.Trace, len(traces))
	copy(sorted, traces)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Confidence > sorted[i].Confidence {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}
