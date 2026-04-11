package strataudit

import (
	"context"
	"fmt"
	"time"

	"sdp_dev/internal/strataudit/model"
)

type findingTpl struct{ Title, Desc, Rec string }

var findingTemplates = map[string]map[string]findingTpl{
	"gap": {
		"ru": {
			Title: "Разрыв: %q не имеет поддержки от уровня %s",
			Desc:  "Сущность %q на уровне %s (ранг %d) не имеет трассированных связей с уровнем %s.",
			Rec:   "Добавить операционные сущности на уровне %s, которые поддерживают эту цель.",
		},
		"en": {
			Title: "Gap: %q has no support from %s",
			Desc:  "Entity %q at level %s (rank %d) has no traced contributions from level %s.",
			Rec:   "Add operational entities at %s level that contribute to this goal.",
		},
	},
	"orphan": {
		"ru": {
			Title: "Сирота: %q не связана с уровнем %s",
			Desc:  "Сущность %q на уровне %s не трассирована ни к одной сущности уровня %s.",
			Rec:   "Проверить, поддерживает ли эта сущность стратегическую цель, или удалить/снизить приоритет.",
		},
		"en": {
			Title: "Orphan: %q has no link to %s",
			Desc:  "Entity %q at level %s is not traced to any entity at level %s.",
			Rec:   "Verify if this entity supports a strategic goal or remove/deprioritize it.",
		},
	},
	"strong_trace": {
		"ru": {
			Title: "Сильная связь: %.0f%% уверенность к %q",
			Desc:  "Сущность вносит вклад в %q с уверенностью %.2f.",
		},
		"en": {
			Title: "Strong trace: %.0f%% confidence to %q",
			Desc:  "Entity contributes to %q with %.2f confidence.",
		},
	},
	"alignment": {
		"ru": {
			Title: "Выравнивание: связь с %q",
			Desc:  "Сущность уровня %s вносит вклад в %q с уверенностью %.0f%%.",
		},
		"en": {
			Title: "Alignment: trace to %q",
			Desc:  "Entity at %s level contributes to %q with %.0f%% confidence.",
		},
	},
	"weak_link": {
		"ru": {
			Title: "Слабая связь: %.0f%% уверенность к %q",
			Desc:  "Связь от %s к %q имеет низкую уверенность (%.2f). Требуется ручная проверка.",
		},
		"en": {
			Title: "Weak link: %.0f%% confidence to %q",
			Desc:  "Trace from %s to %q has low confidence (%.2f). Verify manually.",
		},
	},
	"ambiguous": {
		"ru": {
			Title: "Неоднозначная связь: %q имеет несколько близких кандидатов",
			Desc:  "Разница Top-2 уверенности %.2f (< 0.15). Невозможно определить основную связь.",
		},
		"en": {
			Title: "Ambiguous trace: %q has multiple close candidates",
			Desc:  "Top-2 confidence delta is %.2f (< 0.15). Cannot determine primary trace.",
		},
	},
	"coverage": {
		"ru": {
			Title: "Покрытие уровня %s: %.0f%% (%d/%d)",
			Desc:  "Уровень %s: %d сущностей, %d трассировано (%.1f%% покрытия).",
		},
		"en": {
			Title: "%s coverage: %.0f%% (%d/%d)",
			Desc:  "Level %s has %d entities, %d traced (%.1f%% coverage).",
		},
	},
}

func tpl(lang, findingType string) findingTpl {
	if t, ok := findingTemplates[findingType][lang]; ok {
		return t
	}
	return findingTemplates[findingType]["en"]
}

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
	lang := cfg.Output.Lang

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
				sev := model.SeverityCritical
				if upper.Rank > 1 {
					sev = model.SeverityWarn
				}
				findingIdx++
				t := tpl(lang, "gap")
				allFindings = append(allFindings, model.Finding{
					ID:             findingID("gap", findingIdx),
					Type:           model.FindingGap,
					Severity:       sev,
					EntityIDs:      []string{e.ID},
					Title:          fmt.Sprintf(t.Title, e.Title, lower.Name),
					Description:    fmt.Sprintf(t.Desc, e.Title, upper.Name, upper.Rank, lower.Name),
					Recommendation: fmt.Sprintf(t.Rec, lower.Name),
				})
			} else {
				traces, _ := store.TracesForEntity(ctx, e.ID)
				for _, tr := range traces {
					if tr.Relation != model.RelationContributesTo {
						continue
					}
					findingIdx++
					if tr.Confidence >= 0.85 {
						t := tpl(lang, "strong_trace")
						allFindings = append(allFindings, model.Finding{
							ID:             findingID("strong_trace", findingIdx),
							Type:           model.FindingStrongTrace,
							Severity:       model.SeverityInfo,
							EntityIDs:      []string{tr.SourceEntityID, tr.TargetEntityID},
							Title:          fmt.Sprintf(t.Title, tr.Confidence*100, e.Title),
							Description:    fmt.Sprintf(t.Desc, e.Title, tr.Confidence),
							LLMScore:       model.LLMScoreHigh,
							ConfidenceScore: tr.Confidence,
						})
					} else if tr.Confidence >= 0.8 {
						t := tpl(lang, "alignment")
						allFindings = append(allFindings, model.Finding{
							ID:             findingID("alignment", findingIdx),
							Type:           model.FindingAlignment,
							Severity:       model.SeverityInfo,
							EntityIDs:      []string{tr.SourceEntityID, tr.TargetEntityID},
							Title:          fmt.Sprintf(t.Title, e.Title),
							Description:    fmt.Sprintf(t.Desc, lower.Name, e.Title, tr.Confidence*100),
							LLMScore:       model.LLMScoreHigh,
							ConfidenceScore: tr.Confidence,
						})
					} else if tr.Confidence < cfg.Thresholds.TraceConfidence+0.1 {
						t := tpl(lang, "weak_link")
						allFindings = append(allFindings, model.Finding{
							ID:             findingID("weak_link", findingIdx),
							Type:           model.FindingWeakLink,
							Severity:       model.SeverityWarn,
							EntityIDs:      []string{tr.SourceEntityID, tr.TargetEntityID},
							Title:          fmt.Sprintf(t.Title, tr.Confidence*100, e.Title),
							Description:    fmt.Sprintf(t.Desc, lower.Name, e.Title, tr.Confidence),
							LLMScore:       model.LLMScoreLow,
							ConfidenceScore: tr.Confidence,
						})
					}
					break
				}

				// Check for ambiguous traces
				if len(traces) >= 2 {
					sorted := sortTracesByConfidence(traces)
					if sorted[0].Confidence-sorted[1].Confidence < 0.15 {
						findingIdx++
						t := tpl(lang, "ambiguous")
						allFindings = append(allFindings, model.Finding{
							ID:          findingID("ambiguous", findingIdx),
							Type:        model.FindingAmbiguousTrace,
							Severity:    model.SeverityWarn,
							EntityIDs:   []string{sorted[0].TargetEntityID, sorted[1].TargetEntityID},
							Title:       fmt.Sprintf(t.Title, e.Title),
							Description: fmt.Sprintf(t.Desc, sorted[0].Confidence-sorted[1].Confidence),
						})
					}
				}
			}
		}

		// Detect orphans
		for _, e := range lowerEntities {
			if !lowerTraced[e.ID] {
				findingIdx++
				t := tpl(lang, "orphan")
				allFindings = append(allFindings, model.Finding{
					ID:             findingID("orphan", findingIdx),
					Type:           model.FindingOrphan,
					Severity:       model.SeverityWarn,
					EntityIDs:      []string{e.ID},
					Title:          fmt.Sprintf(t.Title, e.Title, upper.Name),
					Description:    fmt.Sprintf(t.Desc, e.Title, lower.Name, upper.Name),
					Recommendation: t.Rec,
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
			Title:          fmt.Sprintf(tpl(lang, "coverage").Title, level.Name, pct, tracedCount, total),
			Description:    fmt.Sprintf(tpl(lang, "coverage").Desc, level.Name, total, tracedCount, pct),
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
