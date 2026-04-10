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

		// Get traces between these levels
		upperTraced := tracedEntityIDs(ctx, store, upper.ID)
		lowerTraced := tracedEntityIDs(ctx, store, lower.ID)

		// Detect gaps: upper-level entities with no traces to lower level
		for _, e := range upperEntities {
			if !upperTraced[e.ID] {
				findingIdx++
				allFindings = append(allFindings, model.Finding{
					ID:          findingID("gap", findingIdx),
					Type:        model.FindingGap,
					Severity:    model.SeverityCritical,
					EntityIDs:   []string{e.ID},
					Title:       fmt.Sprintf("Gap: %q has no support from %s", e.Title, lower.Name),
					Description: fmt.Sprintf("Entity %q at level %s has no traced contributions from level %s.", e.Title, upper.Name, lower.Name),
					Recommendation: fmt.Sprintf("Add operational entities at %s level that contribute to this goal.", lower.Name),
				})
			} else {
				// Check for strong traces (high confidence)
				traces, _ := store.TracesForEntity(ctx, e.ID)
				for _, tr := range traces {
					if tr.Confidence >= 0.8 && tr.Relation == model.RelationContributesTo {
						findingIdx++
						allFindings = append(allFindings, model.Finding{
							ID:       findingID("alignment", findingIdx),
							Type:     model.FindingAlignment,
							Severity: model.SeverityInfo,
							EntityIDs: []string{tr.SourceEntityID, tr.TargetEntityID},
							Title:    fmt.Sprintf("Strong alignment: trace to %q", e.Title),
							Description: fmt.Sprintf("Entity at %s level contributes to %q with %.0f%% confidence.", lower.Name, e.Title, tr.Confidence*100),
							LLMScore: model.LLMScoreHigh,
							ConfidenceScore: tr.Confidence,
						})
						break // one strong trace per entity
					}
				}
			}
		}

		// Detect orphans: lower-level entities with no traces to upper level
		for _, e := range lowerEntities {
			if !lowerTraced[e.ID] {
				findingIdx++
				allFindings = append(allFindings, model.Finding{
					ID:          findingID("orphan", findingIdx),
					Type:        model.FindingOrphan,
					Severity:    model.SeverityWarn,
					EntityIDs:   []string{e.ID},
					Title:       fmt.Sprintf("Orphan: %q has no link to %s", e.Title, upper.Name),
					Description: fmt.Sprintf("Entity %q at level %s is not traced to any entity at level %s.", e.Title, lower.Name, upper.Name),
					Recommendation: "Verify if this entity supports a strategic goal or remove/deprioritize it.",
				})
			}
		}
	}

	// Detect unknown rationale: orphans without any source quote or description
	for i := len(allFindings) - 1; i >= 0; i-- {
		f := allFindings[i]
		if f.Type == model.FindingOrphan && len(f.EntityIDs) > 0 {
			entities, _ := store.EntitiesByLevel(ctx, "", model.Page{Limit: 1})
			_ = entities
			// Check entity source
			for _, eid := range f.EntityIDs {
				traces, _ := store.TracesForEntity(ctx, eid)
				if len(traces) == 0 {
					findingIdx++
					allFindings = append(allFindings, model.Finding{
						ID:        findingID("unknown_rationale", findingIdx),
						Type:      model.FindingUnknownRationale,
						Severity:  model.SeverityWarn,
						EntityIDs: []string{eid},
						Title:     fmt.Sprintf("Unknown rationale: purpose unclear"),
						Description: "This entity has no traces to any level and its strategic purpose cannot be determined.",
						Recommendation: "Document why this work is being done or remove it.",
					})
				}
			}
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
			ID:       findingID("coverage", int(total)),
			Type:     model.FindingCoverage,
			Severity: coverageSeverity(pct, cfg.Thresholds.CoverageWarn),
			Title:    fmt.Sprintf("%s coverage: %.0f%% (%d/%d)", level.Name, pct, tracedCount, total),
			Description: fmt.Sprintf("Level %s has %d entities, %d traced (%.1f%% coverage).", level.Name, total, tracedCount, pct),
			ConfidenceScore: 1.0,
		})

		// Save coverage record
		store.SaveCoverage(ctx, []model.Coverage{{
			ID:             fmt.Sprintf("cov_%s", level.ID),
			LevelID:        level.ID,
			TotalEntities:  int(total),
			TracedEntities: int(tracedCount),
			CoveragePct:    pct,
			ComputedAt:     now,
		}})
	}

	// Save all findings
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
