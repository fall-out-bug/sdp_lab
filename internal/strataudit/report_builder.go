package strataudit

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sdp_dev/internal/strataudit/model"
	"sdp_dev/internal/strataudit/report"
)

type extractCheckpoint struct {
	Verified  int `json:"verified"`
	Suspect   int `json:"suspect"`
	Rejected  int `json:"rejected"`
	Documents int `json:"documents"`
	Saved     int `json:"saved"`
}

// BuildReport gathers data from store and builds a report.AuditReport.
func BuildReport(ctx context.Context, cfg *Config, store *SQLiteStore) (*report.AuditReport, error) {
	generatedAt := time.Now().UTC()
	outputDir := cfg.Output.Dir
	if outputDir == "" {
		outputDir = ".strataudit"
	}
	outputLang := cfg.Output.Lang
	if outputLang == "" {
		outputLang = "ru"
	}

	rpt := &report.AuditReport{
		SchemaVersion: report.SchemaVersion,
		AuditScope: report.AuditScopeReport{
			ProjectName:        cfg.Project.Name,
			ProjectDescription: cfg.Project.Description,
			OutputDir:          outputDir,
			OutputLang:         outputLang,
			GeneratedAt:        generatedAt.Format(time.RFC3339),
			SourceDirs:         append([]string(nil), cfg.Project.SourceDirs...),
			Exclude:            append([]string(nil), cfg.Project.Exclude...),
			Models: report.AuditModelsReport{
				DefaultModel:   cfg.LLM.Model,
				ExtractModel:   cfg.LLM.ExtractModel,
				EmbeddingModel: cfg.LLM.EmbeddingModel,
			},
			Thresholds: report.AuditThresholdsReport{
				Similarity:           cfg.Thresholds.Similarity,
				TraceConfidence:      cfg.Thresholds.TraceConfidence,
				AutoVerifySimilarity: cfg.Thresholds.AutoVerifySimilarity,
				CoverageWarn:         cfg.Thresholds.CoverageWarn,
				LLMVerifyBudget:      cfg.Thresholds.LLMVerifyBudget,
			},
		},
	}

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
	candidates, err := store.AllCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("load candidates: %w", err)
	}
	traces, err := store.AllTraces(ctx)
	if err != nil {
		return nil, fmt.Errorf("load traces: %w", err)
	}
	findings, err := store.AllFindings(ctx, model.Page{Limit: 10000})
	if err != nil {
		return nil, fmt.Errorf("load findings: %w", err)
	}
	coverages, err := store.AllCoverage(ctx)
	if err != nil {
		return nil, fmt.Errorf("load coverage: %w", err)
	}

	levelByID := make(map[string]model.Level, len(levels))
	documentByID := make(map[string]model.Document, len(documents))
	sectionByID := make(map[string]model.Section, len(sections))
	documentCountByLevel := make(map[string]int)
	sectionCountByDocument := make(map[string]int)
	for _, level := range levels {
		levelByID[level.ID] = level
	}
	for _, doc := range documents {
		documentByID[doc.ID] = doc
		documentCountByLevel[doc.LevelID]++
	}
	for _, section := range sections {
		sectionByID[section.ID] = section
		sectionCountByDocument[section.DocumentID]++
	}

	entityCountByLevel := make(map[string]int)
	entityCountByDocument := make(map[string]int)
	entityCountBySection := make(map[string]int)
	var allEntities []model.Entity
	entityByID := make(map[string]model.Entity)
	for _, level := range levels {
		entities, err := store.EntitiesByLevel(ctx, level.ID, model.Page{Limit: 10000})
		if err != nil {
			return nil, fmt.Errorf("entities level %s: %w", level.ID, err)
		}
		entityCountByLevel[level.ID] = len(entities)
		allEntities = append(allEntities, entities...)
		for _, entity := range entities {
			entityByID[entity.ID] = entity
			entityCountByDocument[entity.DocumentID]++
			if entity.SectionID != "" {
				entityCountBySection[entity.SectionID]++
			}
		}
	}

	for _, level := range levels {
		rpt.Levels = append(rpt.Levels, report.LevelReport{
			ID:            level.ID,
			Name:          level.Name,
			Rank:          level.Rank,
			Description:   level.Description,
			Patterns:      append([]string(nil), level.Patterns...),
			EntityCount:   entityCountByLevel[level.ID],
			DocumentCount: documentCountByLevel[level.ID],
		})
	}
	for _, doc := range documents {
		level := levelByID[doc.LevelID]
		rpt.Documents = append(rpt.Documents, report.DocumentReport{
			ID:           doc.ID,
			Path:         doc.Path,
			Name:         filepath.Base(doc.Path),
			LevelID:      doc.LevelID,
			LevelName:    level.Name,
			ContentHash:  doc.ContentHash,
			Version:      doc.Version,
			EntityCount:  entityCountByDocument[doc.ID],
			SectionCount: sectionCountByDocument[doc.ID],
		})
	}
	for _, section := range sections {
		doc := documentByID[section.DocumentID]
		level := levelByID[doc.LevelID]
		rpt.Sections = append(rpt.Sections, report.SectionReport{
			ID:           section.ID,
			DocumentID:   section.DocumentID,
			DocumentPath: doc.Path,
			LevelID:      doc.LevelID,
			LevelName:    level.Name,
			Ordinal:      section.Ordinal,
			Heading:      section.Heading,
			Label:        reportSectionLabel(doc, section),
			CharStart:    section.CharStart,
			CharEnd:      section.CharEnd,
			Preview:      section.Preview,
			EntityCount:  entityCountBySection[section.ID],
			QualityFlags: append([]string(nil), section.QualityFlags...),
		})
	}

	entityCounts := report.EntityTrustCounts{
		Rejected: extractRejectedEntities(ctx, store),
	}
	for _, entity := range allEntities {
		doc := documentByID[entity.DocumentID]
		level := levelByID[entity.LevelID]
		section := sectionByID[entity.SectionID]
		rpt.Entities = append(rpt.Entities, report.EntityReport{
			ID:                  entity.ID,
			Type:                string(entity.Type),
			Title:               entity.Title,
			Description:         entity.Description,
			TitleOriginal:       entity.TitleOriginal,
			DescriptionOriginal: entity.DescriptionOriginal,
			Lang:                entity.Lang,
			LanguageMismatch:    entity.LanguageMismatch,
			LevelID:             entity.LevelID,
			LevelName:           level.Name,
			DocumentID:          entity.DocumentID,
			DocumentPath:        doc.Path,
			SectionID:           entity.SectionID,
			SectionHeading:      section.Heading,
			SourceQuote:         entity.SourceQuote,
			QuoteStartOffset:    entity.QuoteStartOffset,
			QuoteEndOffset:      entity.QuoteEndOffset,
			TrustGrade:          string(entity.TrustGrade),
			QualityFlags:        append([]string(nil), entity.QualityFlags...),
		})
		entityCounts.TotalAdmitted++
		switch entity.TrustGrade {
		case model.TrustGradeVerified, "":
			entityCounts.Verified++
		case model.TrustGradeSuspect:
			entityCounts.Suspect++
		case model.TrustGradeRejected:
			entityCounts.Rejected++
		default:
			entityCounts.Unknown++
		}
	}

	traceModeCounts := make(map[string]int)
	for _, candidate := range candidates {
		rpt.TraceCandidates = append(rpt.TraceCandidates, report.TraceCandidateReport{
			ID:             candidate.ID,
			SourceEntityID: candidate.SourceEntityID,
			TargetEntityID: candidate.TargetEntityID,
			Similarity:     candidate.Similarity,
			Verified:       candidate.Verified,
			TraceID:        candidate.TraceID,
			DiagnosticCode: candidate.DiagnosticCode,
		})
	}
	for _, trace := range traces {
		sourceEntity, ok := entityByID[trace.SourceEntityID]
		if !ok {
			return nil, fmt.Errorf("verified trace %s references missing source entity %s", trace.ID, trace.SourceEntityID)
		}
		targetEntity, ok := entityByID[trace.TargetEntityID]
		if !ok {
			return nil, fmt.Errorf("verified trace %s references missing target entity %s", trace.ID, trace.TargetEntityID)
		}
		sourceDoc, ok := documentByID[sourceEntity.DocumentID]
		if !ok {
			return nil, fmt.Errorf("verified trace %s source document %s missing", trace.ID, sourceEntity.DocumentID)
		}
		targetDoc, ok := documentByID[targetEntity.DocumentID]
		if !ok {
			return nil, fmt.Errorf("verified trace %s target document %s missing", trace.ID, targetEntity.DocumentID)
		}
		sourceSection := sectionByID[sourceEntity.SectionID]
		targetSection := sectionByID[targetEntity.SectionID]
		sourceEvidence := buildEvidenceRef(sourceEntity, sourceDoc, sourceSection)
		targetEvidence := buildEvidenceRef(targetEntity, targetDoc, targetSection)
		if sourceEvidence.DocumentPath == "" || targetEvidence.DocumentPath == "" || sourceEvidence.Quote == "" || targetEvidence.Quote == "" {
			return nil, fmt.Errorf("verified trace %s has incomplete trace_evidence", trace.ID)
		}
		rpt.VerifiedTraces = append(rpt.VerifiedTraces, report.VerifiedTraceReport{
			ID:               trace.ID,
			SourceEntityID:   trace.SourceEntityID,
			TargetEntityID:   trace.TargetEntityID,
			Relation:         string(trace.Relation),
			Confidence:       trace.Confidence,
			SimilarityScore:  trace.SimilarityScore,
			Justification:    trace.Justification,
			VerificationMode: string(trace.VerificationMode),
			TrustGrade:       string(trace.TrustGrade),
			TraceEvidence: report.TraceEvidenceReport{
				Source: sourceEvidence,
				Target: targetEvidence,
			},
		})
		if trace.VerificationMode != "" {
			traceModeCounts[string(trace.VerificationMode)]++
		}
	}

	var criticalFindings, warnFindings, infoFindings int
	corpusQualityFlagCounts := make(map[string]int)
	for _, entity := range allEntities {
		for _, flag := range reportableQualityFlags(entity.QualityFlags) {
			corpusQualityFlagCounts[flag]++
		}
	}
	for _, section := range sections {
		for _, flag := range reportableQualityFlags(section.QualityFlags) {
			corpusQualityFlagCounts[flag]++
		}
	}
	corpusQualityDocs := buildCorpusQualityDocs(findings, sections, allEntities, documentByID, levelByID, sectionByID, corpusQualityFlagCounts)
	rpt.CorpusQuality = report.CorpusQualityReport{
		TotalIssues:       sumMapValues(corpusQualityFlagCounts),
		CriticalDocuments: countCriticalCorpusDocs(corpusQualityDocs),
		FlagCounts:        corpusQualityFlagCounts,
		Documents:         corpusQualityDocs,
	}

	for _, finding := range findings {
		documentPaths := mapDocumentPaths(finding.DocumentIDs, documentByID)
		sectionLabels := mapSectionLabels(finding.SectionIDs, sectionByID, documentByID)
		rpt.FindingsGrouped = append(rpt.FindingsGrouped, report.FindingReport{
			ID:             finding.ID,
			Type:           string(finding.Type),
			Severity:       string(finding.Severity),
			ClusterKey:     finding.ClusterKey,
			Title:          finding.Title,
			Description:    finding.Description,
			Recommendation: finding.Recommendation,
			EntityIDs:      append([]string(nil), finding.EntityIDs...),
			DocumentIDs:    append([]string(nil), finding.DocumentIDs...),
			DocumentPaths:  documentPaths,
			SectionIDs:     append([]string(nil), finding.SectionIDs...),
			SectionLabels:  sectionLabels,
			AffectedCount:  maxAffectedCount(finding),
			Confidence:     finding.ConfidenceScore,
		})
		switch finding.Severity {
		case model.SeverityCritical:
			criticalFindings++
		case model.SeverityWarn:
			warnFindings++
		default:
			infoFindings++
		}
	}

	rpt.Coverage = buildCoverageBlock(coverages, levelByID, documentByID, sectionByID)
	rpt.TrustSummary = report.TrustSummaryReport{
		OverallStatus: deriveOverallTrustStatus(entityCounts, rpt.CorpusQuality),
		Entities:      entityCounts,
		Traces: report.TraceTrustCounts{
			Verified:          len(rpt.VerifiedTraces),
			Candidates:        len(rpt.TraceCandidates),
			VerificationModes: traceModeCounts,
		},
		Findings: report.FindingCountReport{
			Total:    len(rpt.FindingsGrouped),
			Critical: criticalFindings,
			Warn:     warnFindings,
			Info:     infoFindings,
		},
		Disclaimers: buildTrustDisclaimers(outputLang, entityCounts, len(rpt.TraceCandidates), rpt.CorpusQuality.CriticalDocuments),
	}
	rpt.EvidencePack = report.EvidencePackReport{
		Artifacts: []report.ArtifactReport{
			{Kind: "json_v2", Path: filepath.Join(outputDir, "report.v2.json")},
			{Kind: "json_compat", Path: filepath.Join(outputDir, "report.json")},
			{Kind: "html", Path: filepath.Join(outputDir, "report.html")},
			{Kind: "sqlite", Path: filepath.Join(outputDir, "strataudit.db")},
			{Kind: "similarity_distribution", Path: filepath.Join(outputDir, "similarity_distribution.json")},
		},
		DocumentCount:              len(rpt.Documents),
		TraceCandidateCount:        len(rpt.TraceCandidates),
		RejectedEntityCount:        entityCounts.Rejected,
		EntitiesWithSourceQuotes:   countEntitiesWithQuotes(allEntities),
		VerifiedTracesWithEvidence: len(rpt.VerifiedTraces),
	}

	return rpt, nil
}

func extractRejectedEntities(ctx context.Context, store *SQLiteStore) int {
	state, err := store.LoadPipelineState(ctx, "extract")
	if err != nil || state == nil || state.Checkpoint == "" {
		return 0
	}
	var checkpoint extractCheckpoint
	if err := json.Unmarshal([]byte(state.Checkpoint), &checkpoint); err != nil {
		return 0
	}
	return checkpoint.Rejected
}

func reportSectionLabel(doc model.Document, section model.Section) string {
	base := filepath.Base(doc.Path)
	if section.Heading != "" {
		return fmt.Sprintf("%s#%s", base, section.Heading)
	}
	return fmt.Sprintf("%s#section-%d", base, section.Ordinal)
}

func buildEvidenceRef(entity model.Entity, doc model.Document, section model.Section) report.EvidenceRefReport {
	return report.EvidenceRefReport{
		DocumentID:       entity.DocumentID,
		DocumentPath:     doc.Path,
		SectionID:        entity.SectionID,
		SectionHeading:   section.Heading,
		Quote:            entity.SourceQuote,
		QuoteStartOffset: entity.QuoteStartOffset,
		QuoteEndOffset:   entity.QuoteEndOffset,
		TrustGrade:       string(entity.TrustGrade),
	}
}

func reportableQualityFlags(flags []string) []string {
	var filtered []string
	for _, flag := range flags {
		switch flag {
		case "", "quote_verified", "section_parse_fallback":
			continue
		default:
			filtered = append(filtered, flag)
		}
	}
	return dedupeStrings(filtered)
}

func buildCorpusQualityDocs(findings []model.Finding, sections []model.Section, entities []model.Entity, documentByID map[string]model.Document, levelByID map[string]model.Level, sectionByID map[string]model.Section, flagCounts map[string]int) []report.CorpusQualityDocReport {
	byDocument := make(map[string]*report.CorpusQualityDocReport)

	ensureDoc := func(documentID string) *report.CorpusQualityDocReport {
		entry, ok := byDocument[documentID]
		if ok {
			return entry
		}
		doc := documentByID[documentID]
		level := levelByID[doc.LevelID]
		entry = &report.CorpusQualityDocReport{
			DocumentID:   documentID,
			DocumentPath: doc.Path,
			LevelID:      doc.LevelID,
			LevelName:    level.Name,
			Severity:     string(model.SeverityWarn),
		}
		byDocument[documentID] = entry
		return entry
	}

	for _, finding := range findings {
		if finding.Type != model.FindingCorpusQualityCluster {
			continue
		}
		for _, documentID := range finding.DocumentIDs {
			entry := ensureDoc(documentID)
			flags := extractFlagsFromCluster(finding.ClusterKey, finding.Title, finding.Description, flagCounts)
			entry.Severity = higherCorpusSeverity(entry.Severity, string(finding.Severity))
			entry.Flags = dedupeStrings(append(entry.Flags, flags...))
			entry.FindingIDs = dedupeStrings(append(entry.FindingIDs, finding.ID))
			entry.SectionIDs = dedupeStrings(append(entry.SectionIDs, finding.SectionIDs...))
			entry.EntityIDs = dedupeStrings(append(entry.EntityIDs, finding.EntityIDs...))
		}
	}

	for _, section := range sections {
		flags := reportableQualityFlags(section.QualityFlags)
		if len(flags) == 0 {
			continue
		}
		entry := ensureDoc(section.DocumentID)
		entry.Severity = higherCorpusSeverity(entry.Severity, corpusSeverityFromFlags(flags))
		entry.Flags = dedupeStrings(append(entry.Flags, flags...))
		entry.SectionIDs = dedupeStrings(append(entry.SectionIDs, section.ID))
	}

	for _, entity := range entities {
		flags := reportableQualityFlags(entity.QualityFlags)
		if len(flags) == 0 {
			continue
		}
		entry := ensureDoc(entity.DocumentID)
		entry.Severity = higherCorpusSeverity(entry.Severity, corpusSeverityFromFlags(flags))
		entry.Flags = dedupeStrings(append(entry.Flags, flags...))
		entry.EntityIDs = dedupeStrings(append(entry.EntityIDs, entity.ID))
		if entity.SectionID != "" {
			entry.SectionIDs = dedupeStrings(append(entry.SectionIDs, entity.SectionID))
		}
	}

	docs := make([]report.CorpusQualityDocReport, 0, len(byDocument))
	for _, entry := range byDocument {
		entry.IssueCount = max(1, len(entry.Flags)+len(entry.FindingIDs))
		docs = append(docs, *entry)
	}

	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Severity != docs[j].Severity {
			return corpusSeverityRank(docs[i].Severity) < corpusSeverityRank(docs[j].Severity)
		}
		if docs[i].IssueCount != docs[j].IssueCount {
			return docs[i].IssueCount > docs[j].IssueCount
		}
		return docs[i].DocumentPath < docs[j].DocumentPath
	})
	return docs
}

func corpusSeverityFromFlags(flags []string) string {
	for _, flag := range flags {
		switch flag {
		case "prompt_leak", "quote_not_found", "boilerplate_repetition", "language_mismatch":
			return string(model.SeverityCritical)
		}
	}
	return string(model.SeverityWarn)
}

func higherCorpusSeverity(current, candidate string) string {
	if corpusSeverityRank(candidate) < corpusSeverityRank(current) {
		return candidate
	}
	return current
}

func extractFlagsFromCluster(clusterKey, title, description string, knownFlags map[string]int) []string {
	var found []string
	searchSpace := strings.Join([]string{clusterKey, title, description}, " ")
	for flag := range knownFlags {
		if strings.Contains(searchSpace, flag) {
			found = append(found, flag)
		}
	}
	sort.Strings(found)
	return found
}

func buildCoverageBlock(coverages []model.Coverage, levelByID map[string]model.Level, documentByID map[string]model.Document, sectionByID map[string]model.Section) report.CoverageBlockReport {
	var block report.CoverageBlockReport
	for _, coverage := range coverages {
		doc := documentByID[coverage.DocumentID]
		level := levelByID[coverage.LevelID]
		section := sectionByID[coverage.SectionID]
		entry := report.CoverageScopeReport{
			ScopeType:      string(coverage.ScopeType),
			ScopeID:        coverage.ScopeID,
			ScopeLabel:     coverage.ScopeLabel,
			LevelID:        coverage.LevelID,
			LevelName:      level.Name,
			DocumentID:     coverage.DocumentID,
			DocumentPath:   doc.Path,
			SectionID:      coverage.SectionID,
			SectionHeading: section.Heading,
			Total:          coverage.TotalEntities,
			Traced:         coverage.TracedEntities,
			Pct:            coverage.CoveragePct,
		}
		switch coverage.ScopeType {
		case model.CoverageScopeLevel:
			block.ByLevel = append(block.ByLevel, entry)
			block.AverageLevelPct += coverage.CoveragePct
		case model.CoverageScopeDocument:
			block.ByDocument = append(block.ByDocument, entry)
		case model.CoverageScopeSection:
			block.BySection = append(block.BySection, entry)
		}
	}
	if len(block.ByLevel) > 0 {
		block.AverageLevelPct /= float64(len(block.ByLevel))
	}
	block.LowestDocuments = topCoverageSlice(block.ByDocument, 5)
	block.LowestSections = topCoverageSlice(block.BySection, 5)
	return block
}

func topCoverageSlice(values []report.CoverageScopeReport, limit int) []report.CoverageScopeReport {
	if len(values) == 0 {
		return nil
	}
	if len(values) < limit {
		limit = len(values)
	}
	out := make([]report.CoverageScopeReport, limit)
	copy(out, values[:limit])
	return out
}

func deriveOverallTrustStatus(entityCounts report.EntityTrustCounts, corpus report.CorpusQualityReport) string {
	if entityCounts.Rejected > 0 || corpus.CriticalDocuments > 0 {
		return "critical"
	}
	if entityCounts.Suspect > 0 || len(corpus.Documents) > 0 {
		return "warning"
	}
	return "ok"
}

func buildTrustDisclaimers(lang string, entityCounts report.EntityTrustCounts, candidateCount int, criticalCorpusDocs int) []string {
	var disclaimers []string
	switch lang {
	case "ru":
		if entityCounts.Rejected > 0 {
			disclaimers = append(disclaimers, fmt.Sprintf("Отклонено %d сущн.; отчёт нельзя читать как полную картину корпуса.", entityCounts.Rejected))
		}
		if entityCounts.Suspect > 0 {
			disclaimers = append(disclaimers, fmt.Sprintf("Есть %d suspect-сущн.; часть выводов требует ручной проверки.", entityCounts.Suspect))
		}
		if candidateCount > 0 {
			disclaimers = append(disclaimers, "Similarity candidates не считаются доказательством без verified trace.")
		}
		if criticalCorpusDocs > 0 {
			disclaimers = append(disclaimers, fmt.Sprintf("Есть %d документов с критическими проблемами качества корпуса.", criticalCorpusDocs))
		}
	default:
		if entityCounts.Rejected > 0 {
			disclaimers = append(disclaimers, fmt.Sprintf("%d entities were rejected; treat the audit as incomplete until source quality is fixed.", entityCounts.Rejected))
		}
		if entityCounts.Suspect > 0 {
			disclaimers = append(disclaimers, fmt.Sprintf("%d suspect entities remain and require manual review.", entityCounts.Suspect))
		}
		if candidateCount > 0 {
			disclaimers = append(disclaimers, "Similarity candidates are diagnostics only and do not count as proof.")
		}
		if criticalCorpusDocs > 0 {
			disclaimers = append(disclaimers, fmt.Sprintf("%d documents have critical corpus-quality problems.", criticalCorpusDocs))
		}
	}
	return disclaimers
}

func countEntitiesWithQuotes(entities []model.Entity) int {
	var count int
	for _, entity := range entities {
		if strings.TrimSpace(entity.SourceQuote) != "" {
			count++
		}
	}
	return count
}

func mapDocumentPaths(documentIDs []string, documentByID map[string]model.Document) []string {
	paths := make([]string, 0, len(documentIDs))
	for _, documentID := range documentIDs {
		doc, ok := documentByID[documentID]
		if !ok || doc.Path == "" {
			continue
		}
		paths = append(paths, doc.Path)
	}
	return dedupeStrings(paths)
}

func mapSectionLabels(sectionIDs []string, sectionByID map[string]model.Section, documentByID map[string]model.Document) []string {
	labels := make([]string, 0, len(sectionIDs))
	for _, sectionID := range sectionIDs {
		section, ok := sectionByID[sectionID]
		if !ok {
			continue
		}
		doc := documentByID[section.DocumentID]
		labels = append(labels, reportSectionLabel(doc, section))
	}
	return dedupeStrings(labels)
}

func maxAffectedCount(finding model.Finding) int {
	counts := []int{len(finding.EntityIDs), len(finding.DocumentIDs), len(finding.SectionIDs)}
	maxValue := 0
	for _, count := range counts {
		if count > maxValue {
			maxValue = count
		}
	}
	return maxValue
}

func corpusSeverityRank(severity string) int {
	switch severity {
	case string(model.SeverityCritical):
		return 0
	case string(model.SeverityWarn):
		return 1
	default:
		return 2
	}
}

func countCriticalCorpusDocs(docs []report.CorpusQualityDocReport) int {
	var count int
	for _, doc := range docs {
		if doc.Severity == string(model.SeverityCritical) {
			count++
		}
	}
	return count
}

func sumMapValues(values map[string]int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
