package strataudit

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sdp_dev/internal/strataudit/model"
)

type findingTpl struct{ Title, Desc, Rec string }

var findingTemplates = map[string]map[string]findingTpl{
	"strategic_gap_cluster": {
		"ru": {
			Title: "Стратегический разрыв: %d сущн. в %q не поддержаны уровнем %s",
			Desc:  "Документ %q содержит %d неподдержанных сущностей уровня %s. Примеры: %s.",
			Rec:   "Добавить доказательные связи от уровня %s или пересмотреть стратегические сущности в документе.",
		},
		"en": {
			Title: "Strategic gap: %d entities in %q lack support from %s",
			Desc:  "Document %q has %d unsupported entities at %s level. Examples: %s.",
			Rec:   "Add evidence-backed links from %s or revise the strategic entities in this document.",
		},
	},
	"orphan_cluster": {
		"ru": {
			Title: "Сироты: %d сущн. в %q не связаны с уровнем %s",
			Desc:  "Документ %q содержит %d операционных сущностей без связи с уровнем %s. Примеры: %s.",
			Rec:   "Подтвердить стратегическую связь или убрать шумовые/локальные сущности из аудита.",
		},
		"en": {
			Title: "Orphans: %d entities in %q have no link to %s",
			Desc:  "Document %q contains %d operational entities with no link to %s. Examples: %s.",
			Rec:   "Confirm the strategic relation or remove noisy/local entities from the audit.",
		},
	},
	"corpus_quality_cluster": {
		"ru": {
			Title: "Качество корпуса: %q загрязнён сигналами %s",
			Desc:  "Документ %q содержит evidence-quality проблемы: %s.",
			Rec:   "Очистить источник или ослабить доверие к выводам из этого документа до исправления качества.",
		},
		"en": {
			Title: "Corpus quality: %q contains %s",
			Desc:  "Document %q contains evidence-quality problems: %s.",
			Rec:   "Clean the source or lower confidence in conclusions from this document until quality issues are fixed.",
		},
	},
	"trace_ambiguity_cluster": {
		"ru": {
			Title: "Неоднозначные трейсы: %d сущн. в %q имеют близкие кандидаты",
			Desc:  "Документ %q содержит %d сущностей с неразличимыми candidate traces. Примеры: %s.",
			Rec:   "Усилить evidence или ввести дополнительные ограничения на link verification для этих сущностей.",
		},
		"en": {
			Title: "Trace ambiguity: %d entities in %q have close candidates",
			Desc:  "Document %q contains %d entities with indistinguishable candidate traces. Examples: %s.",
			Rec:   "Strengthen evidence or add stricter link verification constraints for these entities.",
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

// Analyze builds grouped findings and multi-scope coverage.
func Analyze(ctx context.Context, cfg *Config, store *SQLiteStore) (*AnalyzeResult, error) {
	result := &AnalyzeResult{}
	now := time.Now()

	levels, err := store.LoadLevels(ctx)
	if err != nil {
		return nil, fmt.Errorf("load levels: %w", err)
	}
	if err := store.ClearFindings(ctx); err != nil {
		return nil, fmt.Errorf("clear findings: %w", err)
	}
	if err := store.ClearCoverage(ctx); err != nil {
		return nil, fmt.Errorf("clear coverage: %w", err)
	}
	documents, err := store.AllDocuments(ctx)
	if err != nil {
		return nil, fmt.Errorf("load documents: %w", err)
	}
	sections, err := store.AllSections(ctx)
	if err != nil {
		return nil, fmt.Errorf("load sections: %w", err)
	}
	traces, err := store.AllTraces(ctx)
	if err != nil {
		return nil, fmt.Errorf("load traces: %w", err)
	}
	candidates, err := store.AllCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("load candidates: %w", err)
	}

	documentByID := make(map[string]model.Document, len(documents))
	for _, doc := range documents {
		documentByID[doc.ID] = doc
	}

	var allEntities []model.Entity
	entityByID := make(map[string]model.Entity)
	entitiesByLevel := make(map[string][]model.Entity, len(levels))
	for _, level := range levels {
		entities, err := store.EntitiesByLevel(ctx, level.ID, model.Page{Limit: 10000})
		if err != nil {
			return nil, fmt.Errorf("entities level %s: %w", level.ID, err)
		}
		entitiesByLevel[level.ID] = entities
		allEntities = append(allEntities, entities...)
		for _, entity := range entities {
			entityByID[entity.ID] = entity
		}
	}

	tracesByEntity := groupTracesByEntity(traces)
	inboundSupport, outboundSupport := buildAdjacencySupport(traces, entityByID)
	unverifiedCandidates := groupUnverifiedCandidatesByPair(candidates, entityByID)
	lang := cfg.Output.Lang
	var allFindings []model.Finding

	if len(levels) >= 2 {
		for i := 0; i < len(levels)-1; i++ {
			upper := levels[i]
			lower := levels[i+1]
			pairKey := adjacencyKey(lower.ID, upper.ID)

			allFindings = append(allFindings, buildStrategicGapClusters(lang, upper, lower, entitiesByLevel[upper.ID], inboundSupport[pairKey], documentByID)...)
			allFindings = append(allFindings, buildOrphanClusters(lang, upper, lower, entitiesByLevel[lower.ID], outboundSupport[pairKey], documentByID)...)
			allFindings = append(allFindings, buildTraceAmbiguityClusters(lang, lower, upper, entitiesByLevel[lower.ID], outboundSupport[pairKey], unverifiedCandidates[pairKey], documentByID)...)
		}
	}

	allFindings = append(allFindings, buildCorpusQualityClusters(lang, allEntities, sections, documentByID)...)
	for i := range allFindings {
		allFindings[i].ComputeConfidence()
	}

	coverages := buildCoverageBreakdowns(levels, documents, sections, allEntities, tracesByEntity, documentByID)
	if len(coverages) > 0 {
		for i := range coverages {
			coverages[i].ComputedAt = now
		}
		if err := store.SaveCoverage(ctx, coverages); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save coverage: %w", err))
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

func buildStrategicGapClusters(lang string, upper, lower model.Level, entities []model.Entity, supportedUpper map[string]struct{}, documentByID map[string]model.Document) []model.Finding {
	grouped := make(map[string][]model.Entity)
	for _, entity := range entities {
		if _, ok := supportedUpper[entity.ID]; ok {
			continue
		}
		grouped[entity.DocumentID] = append(grouped[entity.DocumentID], entity)
	}
	keys := sortedKeys(grouped)
	var findings []model.Finding
	for _, documentID := range keys {
		group := grouped[documentID]
		if len(group) == 0 {
			continue
		}
		doc := documentByID[documentID]
		docLabel := docDisplayName(doc)
		t := tpl(lang, string(model.FindingStrategicGapCluster))
		finding := model.Finding{
			ID:             findingClusterID("strategic_gap_cluster", upper.ID, lower.ID, documentID),
			Type:           model.FindingStrategicGapCluster,
			Severity:       strategicGapSeverity(upper.Rank),
			EntityIDs:      entityIDs(group),
			DocumentIDs:    []string{documentID},
			SectionIDs:     uniqueSectionIDs(group),
			ClusterKey:     fmt.Sprintf("strategic_gap:%s:%s:%s", upper.ID, lower.ID, documentID),
			Title:          fmt.Sprintf(t.Title, len(group), docLabel, lower.Name),
			Description:    fmt.Sprintf(t.Desc, docLabel, len(group), upper.Name, sampleEntityTitles(group, 4)),
			Recommendation: fmt.Sprintf(t.Rec, lower.Name),
		}
		findings = append(findings, finding)
	}
	return findings
}

func buildOrphanClusters(lang string, upper, lower model.Level, entities []model.Entity, linkedLower map[string]struct{}, documentByID map[string]model.Document) []model.Finding {
	grouped := make(map[string][]model.Entity)
	for _, entity := range entities {
		if _, ok := linkedLower[entity.ID]; ok {
			continue
		}
		grouped[entity.DocumentID] = append(grouped[entity.DocumentID], entity)
	}
	keys := sortedKeys(grouped)
	var findings []model.Finding
	for _, documentID := range keys {
		group := grouped[documentID]
		if len(group) == 0 {
			continue
		}
		doc := documentByID[documentID]
		docLabel := docDisplayName(doc)
		t := tpl(lang, string(model.FindingOrphanCluster))
		finding := model.Finding{
			ID:             findingClusterID("orphan_cluster", lower.ID, upper.ID, documentID),
			Type:           model.FindingOrphanCluster,
			Severity:       model.SeverityWarn,
			EntityIDs:      entityIDs(group),
			DocumentIDs:    []string{documentID},
			SectionIDs:     uniqueSectionIDs(group),
			ClusterKey:     fmt.Sprintf("orphan:%s:%s:%s", lower.ID, upper.ID, documentID),
			Title:          fmt.Sprintf(t.Title, len(group), docLabel, upper.Name),
			Description:    fmt.Sprintf(t.Desc, docLabel, len(group), upper.Name, sampleEntityTitles(group, 4)),
			Recommendation: t.Rec,
		}
		findings = append(findings, finding)
	}
	return findings
}

func buildTraceAmbiguityClusters(lang string, lower, upper model.Level, entities []model.Entity, linkedLower map[string]struct{}, candidatesBySource map[string][]model.Candidate, documentByID map[string]model.Document) []model.Finding {
	grouped := make(map[string][]model.Entity)
	for _, entity := range entities {
		if _, ok := linkedLower[entity.ID]; ok {
			continue
		}
		candidates := sortCandidatesBySimilarity(candidatesBySource[entity.ID])
		if len(candidates) < 2 {
			continue
		}
		if candidates[0].Similarity-candidates[1].Similarity >= 0.15 {
			continue
		}
		grouped[entity.DocumentID] = append(grouped[entity.DocumentID], entity)
	}
	keys := sortedKeys(grouped)
	var findings []model.Finding
	for _, documentID := range keys {
		group := grouped[documentID]
		if len(group) == 0 {
			continue
		}
		doc := documentByID[documentID]
		docLabel := docDisplayName(doc)
		t := tpl(lang, string(model.FindingTraceAmbiguityCluster))
		findings = append(findings, model.Finding{
			ID:             findingClusterID("trace_ambiguity_cluster", lower.ID, upper.ID, documentID),
			Type:           model.FindingTraceAmbiguityCluster,
			Severity:       model.SeverityWarn,
			EntityIDs:      entityIDs(group),
			DocumentIDs:    []string{documentID},
			SectionIDs:     uniqueSectionIDs(group),
			ClusterKey:     fmt.Sprintf("trace_ambiguity:%s:%s:%s", lower.ID, upper.ID, documentID),
			Title:          fmt.Sprintf(t.Title, len(group), docLabel),
			Description:    fmt.Sprintf(t.Desc, docLabel, len(group), sampleEntityTitles(group, 4)),
			Recommendation: t.Rec,
		})
	}
	return findings
}

func buildCorpusQualityClusters(lang string, entities []model.Entity, sections []model.Section, documentByID map[string]model.Document) []model.Finding {
	buckets := make(map[string]*qualityBucket)
	for _, entity := range entities {
		flags := filterQualityFlags(entity.QualityFlags)
		if len(flags) == 0 {
			continue
		}
		bucket := ensureQualityBucket(buckets, entity.DocumentID)
		bucket.entityIDs = append(bucket.entityIDs, entity.ID)
		if entity.SectionID != "" {
			bucket.sectionIDs = append(bucket.sectionIDs, entity.SectionID)
		}
		bucket.qualityFlags = append(bucket.qualityFlags, flags...)
	}
	for _, section := range sections {
		flags := filterQualityFlags(section.QualityFlags)
		if len(flags) == 0 {
			continue
		}
		bucket := ensureQualityBucket(buckets, section.DocumentID)
		bucket.sectionIDs = append(bucket.sectionIDs, section.ID)
		bucket.qualityFlags = append(bucket.qualityFlags, flags...)
	}

	keys := sortedKeysBuckets(buckets)
	var findings []model.Finding
	for _, documentID := range keys {
		bucket := buckets[documentID]
		flags := dedupeFlags(bucket.qualityFlags)
		if len(flags) == 0 {
			continue
		}
		doc := documentByID[documentID]
		docLabel := docDisplayName(doc)
		t := tpl(lang, string(model.FindingCorpusQualityCluster))
		findings = append(findings, model.Finding{
			ID:             findingClusterID("corpus_quality_cluster", doc.LevelID, "", documentID),
			Type:           model.FindingCorpusQualityCluster,
			Severity:       corpusQualitySeverity(flags),
			EntityIDs:      dedupeStrings(bucket.entityIDs),
			DocumentIDs:    []string{documentID},
			SectionIDs:     dedupeStrings(bucket.sectionIDs),
			ClusterKey:     fmt.Sprintf("corpus_quality:%s:%s", doc.LevelID, documentID),
			Title:          fmt.Sprintf(t.Title, docLabel, strings.Join(flags, ", ")),
			Description:    fmt.Sprintf(t.Desc, docLabel, strings.Join(flags, ", ")),
			Recommendation: t.Rec,
		})
	}
	return findings
}

func buildCoverageBreakdowns(levels []model.Level, documents []model.Document, sections []model.Section, entities []model.Entity, tracesByEntity map[string][]model.Trace, documentByID map[string]model.Document) []model.Coverage {
	entitiesByLevel := make(map[string][]model.Entity)
	entitiesByDocument := make(map[string][]model.Entity)
	entitiesBySection := make(map[string][]model.Entity)
	for _, entity := range entities {
		entitiesByLevel[entity.LevelID] = append(entitiesByLevel[entity.LevelID], entity)
		entitiesByDocument[entity.DocumentID] = append(entitiesByDocument[entity.DocumentID], entity)
		if entity.SectionID != "" {
			entitiesBySection[entity.SectionID] = append(entitiesBySection[entity.SectionID], entity)
		}
	}

	var coverages []model.Coverage
	for _, level := range levels {
		group := entitiesByLevel[level.ID]
		if len(group) == 0 {
			continue
		}
		coverages = append(coverages, newCoverage(model.CoverageScopeLevel, level.ID, level.Name, level.ID, "", "", group, tracesByEntity))
	}
	for _, doc := range documents {
		group := entitiesByDocument[doc.ID]
		if len(group) == 0 {
			continue
		}
		coverages = append(coverages, newCoverage(model.CoverageScopeDocument, doc.ID, docDisplayName(doc), doc.LevelID, doc.ID, "", group, tracesByEntity))
	}
	for _, section := range sections {
		group := entitiesBySection[section.ID]
		if len(group) == 0 {
			continue
		}
		doc := documentByID[section.DocumentID]
		label := sectionDisplayLabel(doc, section)
		coverages = append(coverages, newCoverage(model.CoverageScopeSection, section.ID, label, doc.LevelID, doc.ID, section.ID, group, tracesByEntity))
	}
	return coverages
}

func newCoverage(scopeType model.CoverageScope, scopeID, scopeLabel, levelID, documentID, sectionID string, entities []model.Entity, tracesByEntity map[string][]model.Trace) model.Coverage {
	total := len(entities)
	traced := 0
	for _, entity := range entities {
		if len(tracesByEntity[entity.ID]) > 0 {
			traced++
		}
	}
	pct := 0.0
	if total > 0 {
		pct = float64(traced) / float64(total) * 100
	}
	return model.Coverage{
		ID:             fmt.Sprintf("cov_%s_%s", scopeType, sha256Hash([]byte(scopeID))[:10]),
		ScopeType:      scopeType,
		ScopeID:        scopeID,
		ScopeLabel:     scopeLabel,
		LevelID:        levelID,
		DocumentID:     documentID,
		SectionID:      sectionID,
		TotalEntities:  total,
		TracedEntities: traced,
		CoveragePct:    pct,
	}
}

func buildAdjacencySupport(traces []model.Trace, entityByID map[string]model.Entity) (map[string]map[string]struct{}, map[string]map[string]struct{}) {
	inboundByPair := make(map[string]map[string]struct{})
	outboundByPair := make(map[string]map[string]struct{})
	for _, trace := range traces {
		source, sourceOK := entityByID[trace.SourceEntityID]
		target, targetOK := entityByID[trace.TargetEntityID]
		if !sourceOK || !targetOK {
			continue
		}
		key := adjacencyKey(source.LevelID, target.LevelID)
		if inboundByPair[key] == nil {
			inboundByPair[key] = make(map[string]struct{})
		}
		if outboundByPair[key] == nil {
			outboundByPair[key] = make(map[string]struct{})
		}
		inboundByPair[key][target.ID] = struct{}{}
		outboundByPair[key][source.ID] = struct{}{}
	}
	return inboundByPair, outboundByPair
}

func groupUnverifiedCandidatesByPair(candidates []model.Candidate, entityByID map[string]model.Entity) map[string]map[string][]model.Candidate {
	grouped := make(map[string]map[string][]model.Candidate)
	for _, candidate := range candidates {
		if candidate.Verified {
			continue
		}
		source, sourceOK := entityByID[candidate.SourceEntityID]
		target, targetOK := entityByID[candidate.TargetEntityID]
		if !sourceOK || !targetOK {
			continue
		}
		key := adjacencyKey(source.LevelID, target.LevelID)
		if grouped[key] == nil {
			grouped[key] = make(map[string][]model.Candidate)
		}
		grouped[key][candidate.SourceEntityID] = append(grouped[key][candidate.SourceEntityID], candidate)
	}
	return grouped
}

func adjacencyKey(lowerLevelID, upperLevelID string) string {
	return lowerLevelID + "->" + upperLevelID
}

func groupTracesByEntity(traces []model.Trace) map[string][]model.Trace {
	result := make(map[string][]model.Trace)
	for _, trace := range traces {
		result[trace.SourceEntityID] = append(result[trace.SourceEntityID], trace)
		result[trace.TargetEntityID] = append(result[trace.TargetEntityID], trace)
	}
	return result
}

func strategicGapSeverity(rank int) model.Severity {
	if rank <= 1 {
		return model.SeverityCritical
	}
	return model.SeverityWarn
}

func corpusQualitySeverity(flags []string) model.Severity {
	criticalFlags := map[string]struct{}{
		"prompt_leak":            {},
		"language_mismatch":      {},
		"quote_not_found":        {},
		"boilerplate_repetition": {},
	}
	for _, flag := range flags {
		if _, ok := criticalFlags[flag]; ok {
			return model.SeverityCritical
		}
	}
	return model.SeverityWarn
}

func filterQualityFlags(flags []string) []string {
	var filtered []string
	for _, flag := range flags {
		switch flag {
		case "", "quote_verified", "section_parse_fallback":
			continue
		default:
			filtered = append(filtered, flag)
		}
	}
	return dedupeFlags(filtered)
}

func ensureQualityBucket(buckets map[string]*qualityBucket, documentID string) *qualityBucket {
	bucket := buckets[documentID]
	if bucket == nil {
		bucket = &qualityBucket{}
		buckets[documentID] = bucket
	}
	return bucket
}

type qualityBucket struct {
	entityIDs    []string
	sectionIDs   []string
	qualityFlags []string
}

func docDisplayName(doc model.Document) string {
	if doc.Path == "" {
		return doc.ID
	}
	return filepath.Base(doc.Path)
}

func sectionDisplayLabel(doc model.Document, section model.Section) string {
	base := docDisplayName(doc)
	if section.Heading != "" {
		return fmt.Sprintf("%s#%s", base, section.Heading)
	}
	return fmt.Sprintf("%s#section-%d", base, section.Ordinal)
}

func entityIDs(entities []model.Entity) []string {
	result := make([]string, 0, len(entities))
	for _, entity := range entities {
		result = append(result, entity.ID)
	}
	return dedupeStrings(result)
}

func uniqueSectionIDs(entities []model.Entity) []string {
	var sectionIDs []string
	for _, entity := range entities {
		if entity.SectionID != "" {
			sectionIDs = append(sectionIDs, entity.SectionID)
		}
	}
	return dedupeStrings(sectionIDs)
}

func sampleEntityTitles(entities []model.Entity, limit int) string {
	titles := make([]string, 0, len(entities))
	for _, entity := range entities {
		titles = append(titles, entity.Title)
	}
	titles = dedupeStrings(titles)
	sort.Strings(titles)
	if len(titles) > limit {
		titles = append(titles[:limit], fmt.Sprintf("ещё %d", len(titles)-limit))
	}
	return strings.Join(titles, ", ")
}

func sortedKeys(groups map[string][]model.Entity) []string {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysBuckets(groups map[string]*qualityBucket) []string {
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func findingClusterID(prefix, left, right, scope string) string {
	return fmt.Sprintf("%s_%s", prefix, sha256Hash([]byte(prefix + "|" + left + "|" + right + "|" + scope))[:12])
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func sortTracesByConfidence(traces []model.Trace) []model.Trace {
	sorted := make([]model.Trace, len(traces))
	copy(sorted, traces)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Confidence > sorted[j].Confidence
	})
	return sorted
}

func sortCandidatesBySimilarity(candidates []model.Candidate) []model.Candidate {
	sorted := make([]model.Candidate, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Similarity > sorted[j].Similarity
	})
	return sorted
}
