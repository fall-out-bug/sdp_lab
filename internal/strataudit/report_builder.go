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
	rpt.TraceGraph = buildTraceGraph(allEntities, candidates, traces, documentByID, levelByID, sectionByID)
	rpt.TraceGaps = buildTraceGaps(levels, allEntities, candidates, traces, documentByID, levelByID, sectionByID)

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
	rpt.DocumentViews = buildDocumentViews(rpt.Documents, rpt.Levels, rpt.TraceGraph, rpt.TraceGaps, rpt.CorpusQuality.Documents)
	rpt.ReportModes = defaultReportModes()

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
			{Kind: "llm_diagnostics", Path: filepath.Join(outputDir, "llm_diagnostics.json")},
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

func defaultReportModes() report.ReportModesReport {
	return report.ReportModesReport{
		Default:          "analyst",
		DefaultTab:       "summary",
		CompareAvailable: false,
		Tabs: []report.ReportTabReport{
			{ID: "summary", Label: "Сводка"},
			{ID: "documents", Label: "Документы"},
			{ID: "trace", Label: "Трассировка"},
			{ID: "gaps", Label: "Разрывы"},
			{ID: "diagnostics", Label: "Диагностика"},
		},
	}
}

type documentViewAccumulator struct {
	view         *report.DocumentViewReport
	upstream     map[string]*documentCorrespondenceAccumulator
	downstream   map[string]*documentCorrespondenceAccumulator
	blockers     map[string]*documentBlockerAccumulator
	claimScores  map[string]int
	localClaims  map[string]struct{}
	brokenClaims map[string]struct{}
}

type documentCorrespondenceAccumulator struct {
	item     *report.DocumentCorrespondenceReport
	claimIDs map[string]struct{}
	edgeIDs  map[string]struct{}
}

type documentBlockerAccumulator struct {
	item     *report.DocumentBlockerReport
	claimIDs map[string]struct{}
}

func buildDocumentViews(
	documents []report.DocumentReport,
	levels []report.LevelReport,
	graph report.TraceGraphReport,
	gaps []report.TraceGapReport,
	corpusDocs []report.CorpusQualityDocReport,
) []report.DocumentViewReport {
	views := make([]report.DocumentViewReport, len(documents))
	if len(documents) == 0 {
		return views
	}

	levelRank := make(map[string]int, len(levels))
	for _, level := range levels {
		levelRank[level.ID] = level.Rank
	}

	accByDocID := make(map[string]*documentViewAccumulator, len(documents))
	for i, doc := range documents {
		views[i] = report.DocumentViewReport{
			DocumentID:           doc.ID,
			DocumentPath:         doc.Path,
			DocumentName:         doc.Name,
			LevelID:              doc.LevelID,
			LevelName:            doc.LevelName,
			UpstreamDocuments:    []report.DocumentCorrespondenceReport{},
			DownstreamDocuments:  []report.DocumentCorrespondenceReport{},
			Blockers:             []report.DocumentBlockerReport{},
			CriticalQualityFlags: []string{},
			KeyClaimIDs:          []string{},
		}
		accByDocID[doc.ID] = &documentViewAccumulator{
			view:         &views[i],
			upstream:     make(map[string]*documentCorrespondenceAccumulator),
			downstream:   make(map[string]*documentCorrespondenceAccumulator),
			blockers:     make(map[string]*documentBlockerAccumulator),
			claimScores:  make(map[string]int),
			localClaims:  make(map[string]struct{}),
			brokenClaims: make(map[string]struct{}),
		}
	}

	nodeByID := make(map[string]report.TraceNodeReport, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
		acc, ok := accByDocID[node.DocumentID]
		if !ok {
			continue
		}
		acc.view.ClaimCount++
		acc.localClaims[node.ID] = struct{}{}
	}

	for _, corpusDoc := range corpusDocs {
		acc, ok := accByDocID[corpusDoc.DocumentID]
		if !ok {
			continue
		}
		acc.view.CriticalQualityFlags = documentCriticalQualityFlags(corpusDoc)
	}

	for _, edge := range graph.Edges {
		sourceNode, sourceOK := nodeByID[edge.SourceNodeID]
		targetNode, targetOK := nodeByID[edge.TargetNodeID]
		if !sourceOK || !targetOK {
			continue
		}
		if sourceNode.DocumentID == "" || targetNode.DocumentID == "" || sourceNode.DocumentID == targetNode.DocumentID {
			continue
		}
		addDocumentEdge(accByDocID[sourceNode.DocumentID], sourceNode, targetNode, edge, levelRank, true)
		addDocumentEdge(accByDocID[targetNode.DocumentID], targetNode, sourceNode, edge, levelRank, false)
	}

	for _, gap := range gaps {
		acc, ok := accByDocID[gap.DocumentID]
		if !ok {
			continue
		}
		claimID := firstNonEmptyNonReport(gap.NodeID, gap.EntityID)
		blockerKey := gap.Stage + "|" + gap.GapType
		blocker, ok := acc.blockers[blockerKey]
		if !ok {
			item := &report.DocumentBlockerReport{
				Stage:    gap.Stage,
				GapType:  gap.GapType,
				ClaimIDs: []string{},
			}
			blocker = &documentBlockerAccumulator{
				item:     item,
				claimIDs: make(map[string]struct{}),
			}
			acc.blockers[blockerKey] = blocker
		}
		blocker.item.Count++
		if claimID != "" {
			blocker.claimIDs[claimID] = struct{}{}
			acc.claimScores[claimID]++
			acc.localClaims[claimID] = struct{}{}
			acc.brokenClaims[claimID] = struct{}{}
		}
		acc.view.BlockerCount++
	}

	for i := range views {
		acc := accByDocID[views[i].DocumentID]
		if acc == nil {
			continue
		}
		acc.view.UpstreamDocuments = finalizeDocumentCorrespondences(acc.upstream)
		acc.view.DownstreamDocuments = finalizeDocumentCorrespondences(acc.downstream)
		acc.view.Blockers = finalizeDocumentBlockers(acc.blockers)
		acc.view.BrokenLinkCount = len(acc.brokenClaims)
		acc.view.KeyClaimIDs = topDocumentClaimIDs(acc.claimScores, acc.localClaims, 5)
		if acc.view.ClaimCount == 0 && documents[i].EntityCount > 0 {
			acc.view.ClaimCount = documents[i].EntityCount
		}
	}

	return views
}

func addDocumentEdge(
	acc *documentViewAccumulator,
	currentNode report.TraceNodeReport,
	otherNode report.TraceNodeReport,
	edge report.TraceEdgeReport,
	levelRank map[string]int,
	currentIsSource bool,
) {
	if acc == nil {
		return
	}

	claimID := currentNode.ID
	if claimID != "" {
		acc.claimScores[claimID]++
		acc.localClaims[claimID] = struct{}{}
	}

	bucket := acc.downstream
	if documentRelationDirection(currentNode.LevelID, otherNode.LevelID, currentIsSource, levelRank) == "upstream" {
		bucket = acc.upstream
	}
	correspondence, ok := bucket[otherNode.DocumentID]
	if !ok {
		documentName := firstNonEmptyNonReport(otherNode.DocumentPath, otherNode.DocumentID)
		if otherNode.DocumentPath != "" {
			documentName = filepath.Base(otherNode.DocumentPath)
		}
		item := &report.DocumentCorrespondenceReport{
			DocumentID:   otherNode.DocumentID,
			DocumentPath: otherNode.DocumentPath,
			DocumentName: documentName,
			LevelID:      otherNode.LevelID,
			LevelName:    otherNode.LevelName,
			ClaimIDs:     []string{},
			EdgeIDs:      []string{},
		}
		correspondence = &documentCorrespondenceAccumulator{
			item:     item,
			claimIDs: make(map[string]struct{}),
			edgeIDs:  make(map[string]struct{}),
		}
		bucket[otherNode.DocumentID] = correspondence
	}

	switch edge.Status {
	case string(model.TraceEdgeStatusVerified):
		acc.view.VerifiedLinkCount++
		correspondence.item.VerifiedEdgeCount++
	case string(model.TraceEdgeStatusCandidate):
		acc.view.CandidateLinkCount++
		correspondence.item.CandidateEdgeCount++
	default:
		acc.view.RejectedLinkCount++
		correspondence.item.RejectedEdgeCount++
		if claimID != "" {
			acc.brokenClaims[claimID] = struct{}{}
		}
	}

	if claimID != "" {
		correspondence.claimIDs[claimID] = struct{}{}
	}
	if edge.ID != "" {
		correspondence.edgeIDs[edge.ID] = struct{}{}
	}
}

func documentRelationDirection(currentLevelID, otherLevelID string, currentIsSource bool, levelRank map[string]int) string {
	currentRank, currentOK := levelRank[currentLevelID]
	otherRank, otherOK := levelRank[otherLevelID]
	if currentOK && otherOK {
		switch {
		case otherRank < currentRank:
			return "upstream"
		case otherRank > currentRank:
			return "downstream"
		}
	}
	if currentIsSource {
		return "upstream"
	}
	return "downstream"
}

func finalizeDocumentCorrespondences(values map[string]*documentCorrespondenceAccumulator) []report.DocumentCorrespondenceReport {
	if len(values) == 0 {
		return []report.DocumentCorrespondenceReport{}
	}
	items := make([]report.DocumentCorrespondenceReport, 0, len(values))
	for _, value := range values {
		value.item.ClaimIDs = sortedStringKeys(value.claimIDs)
		value.item.EdgeIDs = sortedStringKeys(value.edgeIDs)
		items = append(items, *value.item)
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.VerifiedEdgeCount != right.VerifiedEdgeCount {
			return left.VerifiedEdgeCount > right.VerifiedEdgeCount
		}
		if left.CandidateEdgeCount != right.CandidateEdgeCount {
			return left.CandidateEdgeCount > right.CandidateEdgeCount
		}
		if left.RejectedEdgeCount != right.RejectedEdgeCount {
			return left.RejectedEdgeCount > right.RejectedEdgeCount
		}
		if left.LevelID != right.LevelID {
			return left.LevelID < right.LevelID
		}
		return left.DocumentPath < right.DocumentPath
	})
	return items
}

func finalizeDocumentBlockers(values map[string]*documentBlockerAccumulator) []report.DocumentBlockerReport {
	if len(values) == 0 {
		return []report.DocumentBlockerReport{}
	}
	items := make([]report.DocumentBlockerReport, 0, len(values))
	for _, value := range values {
		value.item.ClaimIDs = sortedStringKeys(value.claimIDs)
		items = append(items, *value.item)
	}
	sort.Slice(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.Count != right.Count {
			return left.Count > right.Count
		}
		if left.Stage != right.Stage {
			return left.Stage < right.Stage
		}
		return left.GapType < right.GapType
	})
	return items
}

func topDocumentClaimIDs(scores map[string]int, localClaims map[string]struct{}, limit int) []string {
	if len(localClaims) == 0 || limit <= 0 {
		return []string{}
	}
	type claimScore struct {
		id    string
		score int
	}
	ranked := make([]claimScore, 0, len(localClaims))
	for claimID := range localClaims {
		ranked = append(ranked, claimScore{id: claimID, score: scores[claimID]})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].id < ranked[j].id
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, 0, len(ranked))
	for _, item := range ranked {
		ids = append(ids, item.id)
	}
	return ids
}

func documentCriticalQualityFlags(doc report.CorpusQualityDocReport) []string {
	filtered := make([]string, 0, len(doc.Flags))
	for _, flag := range dedupeStrings(doc.Flags) {
		if isCriticalQualityFlag(flag) {
			filtered = append(filtered, flag)
		}
	}
	if len(filtered) == 0 && doc.Severity == string(model.SeverityCritical) {
		filtered = append(filtered, dedupeStrings(doc.Flags)...)
	}
	sort.Strings(filtered)
	return filtered
}

func isCriticalQualityFlag(flag string) bool {
	switch flag {
	case "prompt_leak", "quote_not_found", "boilerplate_repetition", "language_mismatch":
		return true
	default:
		return false
	}
}

func sortedStringKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}

func firstNonEmptyNonReport(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

func buildTraceGraph(
	entities []model.Entity,
	candidates []model.Candidate,
	traces []model.Trace,
	documentByID map[string]model.Document,
	levelByID map[string]model.Level,
	sectionByID map[string]model.Section,
) report.TraceGraphReport {
	graph := report.TraceGraphReport{
		Nodes: []report.TraceNodeReport{},
		Edges: []report.TraceEdgeReport{},
		Paths: []report.TracePathReport{},
	}
	nodeIDs := make(map[string]struct{}, len(entities))
	entityByID := make(map[string]model.Entity, len(entities))

	for _, entity := range entities {
		doc, ok := documentByID[entity.DocumentID]
		if !ok {
			continue
		}
		entityByID[entity.ID] = entity
		level := levelByID[entity.LevelID]
		section := sectionByID[entity.SectionID]
		graph.Nodes = append(graph.Nodes, report.TraceNodeReport{
			ID:             entity.ID,
			EntityID:       entity.ID,
			Type:           string(entity.Type),
			Title:          entity.Title,
			LevelID:        entity.LevelID,
			LevelName:      level.Name,
			DocumentID:     entity.DocumentID,
			DocumentPath:   doc.Path,
			SectionID:      entity.SectionID,
			SectionHeading: section.Heading,
			SourceQuote:    entity.SourceQuote,
			TrustGrade:     string(entity.TrustGrade),
			Lang:           entity.Lang,
		})
		nodeIDs[entity.ID] = struct{}{}
	}

	for _, trace := range traces {
		if !hasTraceNodes(nodeIDs, trace.SourceEntityID, trace.TargetEntityID) {
			continue
		}
		sourceEntity, sourceOK := entityByID[trace.SourceEntityID]
		targetEntity, targetOK := entityByID[trace.TargetEntityID]
		if !sourceOK || !targetOK {
			continue
		}
		edge := report.TraceEdgeReport{
			ID:                trace.ID,
			SourceNodeID:      trace.SourceEntityID,
			TargetNodeID:      trace.TargetEntityID,
			SourceEntityID:    trace.SourceEntityID,
			TargetEntityID:    trace.TargetEntityID,
			Relation:          string(trace.Relation),
			Direction:         string(trace.Direction),
			Status:            string(model.TraceEdgeStatusVerified),
			VerificationMode:  string(trace.VerificationMode),
			Confidence:        trace.Confidence,
			Similarity:        trace.SimilarityScore,
			Reason:            trace.Justification,
			TrustGrade:        string(trace.TrustGrade),
			SourceEvidenceRef: buildTraceEdgeEvidenceRef(sourceEntity, documentByID, sectionByID, trace.SourceSectionID, trace.SourceQuoteStartOffset, trace.SourceQuoteEndOffset),
			TargetEvidenceRef: buildTraceEdgeEvidenceRef(targetEntity, documentByID, sectionByID, trace.TargetSectionID, trace.TargetQuoteStartOffset, trace.TargetQuoteEndOffset),
		}
		graph.Edges = append(graph.Edges, edge)
		graph.Paths = append(graph.Paths, edgeAsPath(edge))
	}

	for _, candidate := range candidates {
		if candidate.Verified || candidate.TraceID != "" {
			continue
		}
		if !hasTraceNodes(nodeIDs, candidate.SourceEntityID, candidate.TargetEntityID) {
			continue
		}
		sourceEntity, sourceOK := entityByID[candidate.SourceEntityID]
		targetEntity, targetOK := entityByID[candidate.TargetEntityID]
		if !sourceOK || !targetOK {
			continue
		}
		edge := report.TraceEdgeReport{
			ID:                candidate.ID,
			SourceNodeID:      candidate.SourceEntityID,
			TargetNodeID:      candidate.TargetEntityID,
			SourceEntityID:    candidate.SourceEntityID,
			TargetEntityID:    candidate.TargetEntityID,
			Status:            string(traceEdgeStatusForCandidate(candidate)),
			VerificationMode:  traceVerificationModeForCandidate(candidate),
			Similarity:        candidate.Similarity,
			Reason:            candidate.DiagnosticCode,
			SourceEvidenceRef: buildTraceEdgeEvidenceRef(sourceEntity, documentByID, sectionByID, sourceEntity.SectionID, sourceEntity.QuoteStartOffset, sourceEntity.QuoteEndOffset),
			TargetEvidenceRef: buildTraceEdgeEvidenceRef(targetEntity, documentByID, sectionByID, targetEntity.SectionID, targetEntity.QuoteStartOffset, targetEntity.QuoteEndOffset),
		}
		graph.Edges = append(graph.Edges, edge)
		graph.Paths = append(graph.Paths, edgeAsPath(edge))
	}

	sort.Slice(graph.Nodes, func(i, j int) bool {
		left := graph.Nodes[i]
		right := graph.Nodes[j]
		if levelByID[left.LevelID].Rank != levelByID[right.LevelID].Rank {
			return levelByID[left.LevelID].Rank < levelByID[right.LevelID].Rank
		}
		if left.DocumentPath != right.DocumentPath {
			return left.DocumentPath < right.DocumentPath
		}
		return left.ID < right.ID
	})
	sort.Slice(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].Status != graph.Edges[j].Status {
			return graph.Edges[i].Status < graph.Edges[j].Status
		}
		if graph.Edges[i].SourceNodeID != graph.Edges[j].SourceNodeID {
			return graph.Edges[i].SourceNodeID < graph.Edges[j].SourceNodeID
		}
		return graph.Edges[i].ID < graph.Edges[j].ID
	})
	sort.Slice(graph.Paths, func(i, j int) bool {
		if graph.Paths[i].Status != graph.Paths[j].Status {
			return graph.Paths[i].Status < graph.Paths[j].Status
		}
		if graph.Paths[i].EntryNodeID != graph.Paths[j].EntryNodeID {
			return graph.Paths[i].EntryNodeID < graph.Paths[j].EntryNodeID
		}
		return graph.Paths[i].ID < graph.Paths[j].ID
	})

	return graph
}

func buildTraceGaps(
	levels []model.Level,
	entities []model.Entity,
	candidates []model.Candidate,
	traces []model.Trace,
	documentByID map[string]model.Document,
	levelByID map[string]model.Level,
	sectionByID map[string]model.Section,
) []report.TraceGapReport {
	gaps := []report.TraceGapReport{}
	if len(levels) < 2 || len(entities) == 0 {
		return gaps
	}

	entityByID := make(map[string]model.Entity, len(entities))
	for _, entity := range entities {
		entityByID[entity.ID] = entity
	}

	entitiesByLevel := groupByLevel(entities)
	_, outboundSupport := buildAdjacencySupport(traces, entityByID)
	candidatesByPair := groupUnverifiedCandidatesByPair(candidates, entityByID)

	for i := 1; i < len(levels); i++ {
		lower := levels[i]
		upper := levels[i-1]
		lowerEntities := entitiesByLevel[lower.ID]
		if len(lowerEntities) == 0 {
			continue
		}

		pairKey := adjacencyKey(lower.ID, upper.ID)
		verifiedBySource := outboundSupport[pairKey]
		candidatesBySource := candidatesByPair[pairKey]
		upperEntitiesExist := len(entitiesByLevel[upper.ID]) > 0

		for _, entity := range lowerEntities {
			if _, ok := verifiedBySource[entity.ID]; ok {
				continue
			}
			gap, ok := buildTraceGap(entity, upper, upperEntitiesExist, candidatesBySource[entity.ID], documentByID, levelByID, sectionByID)
			if !ok {
				continue
			}
			gaps = append(gaps, gap)
		}
	}

	sort.Slice(gaps, func(i, j int) bool {
		left := gaps[i]
		right := gaps[j]
		if levelByID[left.LevelID].Rank != levelByID[right.LevelID].Rank {
			return levelByID[left.LevelID].Rank < levelByID[right.LevelID].Rank
		}
		if left.DocumentPath != right.DocumentPath {
			return left.DocumentPath < right.DocumentPath
		}
		if left.ExpectedToLevelID != right.ExpectedToLevelID {
			return left.ExpectedToLevelID < right.ExpectedToLevelID
		}
		return left.EntityID < right.EntityID
	})

	return gaps
}

func buildTraceGap(
	entity model.Entity,
	expectedLevel model.Level,
	upperEntitiesExist bool,
	candidates []model.Candidate,
	documentByID map[string]model.Document,
	levelByID map[string]model.Level,
	sectionByID map[string]model.Section,
) (report.TraceGapReport, bool) {
	doc, ok := documentByID[entity.DocumentID]
	if !ok {
		return report.TraceGapReport{}, false
	}
	stage, gapType, reason := classifyTraceGap(candidates, upperEntitiesExist)
	section := sectionByID[entity.SectionID]
	return report.TraceGapReport{
		ID:                  traceGapID(entity.ID, expectedLevel.ID),
		NodeID:              entity.ID,
		EntityID:            entity.ID,
		Title:               entity.Title,
		LevelID:             entity.LevelID,
		LevelName:           levelByID[entity.LevelID].Name,
		DocumentID:          entity.DocumentID,
		DocumentPath:        doc.Path,
		SectionID:           entity.SectionID,
		SectionHeading:      section.Heading,
		SourceQuote:         entity.SourceQuote,
		ExpectedToLevelID:   expectedLevel.ID,
		ExpectedToLevelName: expectedLevel.Name,
		Stage:               string(stage),
		GapType:             string(gapType),
		Reason:              reason,
		CandidateCount:      len(candidates),
		TopCandidateIDs:     topCandidateIDs(candidates, 3),
	}, true
}

func classifyTraceGap(candidates []model.Candidate, upperEntitiesExist bool) (model.TraceGapStage, model.TraceGapType, string) {
	if !upperEntitiesExist {
		return model.TraceGapStageUpstreamMissing, model.TraceGapTypeMissingUpstreamEntities, "no_entities_in_expected_level"
	}
	if len(candidates) == 0 {
		return model.TraceGapStageCandidateSearch, model.TraceGapTypeNoCandidates, "no_similarity_candidates_above_threshold"
	}
	switch {
	case allCandidatesHaveDiagnostic(candidates, string(model.TraceCandidateDiagnosticQuoteEvidenceMissing)):
		return model.TraceGapStageVerification, model.TraceGapTypeQuoteEvidenceMissing, string(model.TraceCandidateDiagnosticQuoteEvidenceMissing)
	case allCandidatesHaveDiagnostic(candidates, string(model.TraceCandidateDiagnosticBelowTraceConfidence)):
		return model.TraceGapStageVerification, model.TraceGapTypeLowConfidence, string(model.TraceCandidateDiagnosticBelowTraceConfidence)
	case hasCandidateDiagnostic(candidates, string(model.TraceCandidateDiagnosticVerificationBudgetExhausted)):
		return model.TraceGapStageVerification, model.TraceGapTypeVerificationBudgetExhausted, string(model.TraceCandidateDiagnosticVerificationBudgetExhausted)
	case hasCandidateDiagnostic(candidates, string(model.TraceCandidateDiagnosticVerificationUnavailable)):
		return model.TraceGapStageVerification, model.TraceGapTypeVerificationUnavailable, string(model.TraceCandidateDiagnosticVerificationUnavailable)
	case hasCandidateDiagnostic(candidates, string(model.TraceCandidateDiagnosticLLMVerificationRejected)):
		return model.TraceGapStageVerification, model.TraceGapTypeAllCandidatesRejected, string(model.TraceCandidateDiagnosticLLMVerificationRejected)
	case hasCandidateDiagnostic(candidates, string(model.TraceCandidateDiagnosticQuoteEvidenceMissing)):
		return model.TraceGapStageVerification, model.TraceGapTypeQuoteEvidenceMissing, dominantCandidateDiagnostic(candidates)
	case hasCandidateDiagnostic(candidates, string(model.TraceCandidateDiagnosticBelowTraceConfidence)):
		return model.TraceGapStageVerification, model.TraceGapTypeLowConfidence, dominantCandidateDiagnostic(candidates)
	default:
		return model.TraceGapStageVerification, model.TraceGapTypeAllCandidatesRejected, dominantCandidateDiagnostic(candidates)
	}
}

func allCandidatesHaveDiagnostic(candidates []model.Candidate, diagnostic string) bool {
	if len(candidates) == 0 {
		return false
	}
	for _, candidate := range candidates {
		if candidate.DiagnosticCode != diagnostic {
			return false
		}
	}
	return true
}

func hasCandidateDiagnostic(candidates []model.Candidate, diagnostic string) bool {
	for _, candidate := range candidates {
		if candidate.DiagnosticCode == diagnostic {
			return true
		}
	}
	return false
}

func dominantCandidateDiagnostic(candidates []model.Candidate) string {
	if len(candidates) == 0 {
		return ""
	}
	counts := make(map[string]int, len(candidates))
	bestCode := ""
	bestCount := -1
	for _, candidate := range candidates {
		code := candidate.DiagnosticCode
		if code == "" {
			code = string(model.TraceCandidateDiagnosticEmbeddingSimilarityCandidate)
		}
		counts[code]++
		if counts[code] > bestCount || (counts[code] == bestCount && code < bestCode) {
			bestCode = code
			bestCount = counts[code]
		}
	}
	return bestCode
}

func topCandidateIDs(candidates []model.Candidate, limit int) []string {
	if len(candidates) == 0 || limit <= 0 {
		return nil
	}
	sorted := sortCandidatesBySimilarity(candidates)
	if len(sorted) > limit {
		sorted = sorted[:limit]
	}
	ids := make([]string, 0, len(sorted))
	for _, candidate := range sorted {
		ids = append(ids, candidate.ID)
	}
	return ids
}

func traceGapID(entityID, expectedLevelID string) string {
	return fmt.Sprintf("gap_%s", sha256Hash([]byte(entityID + "|" + expectedLevelID))[:12])
}

func edgeAsPath(edge report.TraceEdgeReport) report.TracePathReport {
	return report.TracePathReport{
		ID:             edge.ID,
		EntryNodeID:    edge.SourceNodeID,
		TerminalNodeID: edge.TargetNodeID,
		NodeIDs:        []string{edge.SourceNodeID, edge.TargetNodeID},
		EdgeIDs:        []string{edge.ID},
		Status:         edge.Status,
		HopCount:       1,
	}
}

func hasTraceNodes(nodeIDs map[string]struct{}, sourceID, targetID string) bool {
	_, sourceOK := nodeIDs[sourceID]
	_, targetOK := nodeIDs[targetID]
	return sourceOK && targetOK
}

func traceEdgeStatusForCandidate(candidate model.Candidate) model.TraceEdgeStatus {
	switch candidate.DiagnosticCode {
	case string(model.TraceCandidateDiagnosticEmbeddingSimilarityCandidate), string(model.TraceCandidateDiagnosticVerificationUnavailable):
		return model.TraceEdgeStatusCandidate
	default:
		return model.TraceEdgeStatusRejected
	}
}

func traceVerificationModeForCandidate(candidate model.Candidate) string {
	switch candidate.DiagnosticCode {
	case string(model.TraceCandidateDiagnosticBelowTraceConfidence), string(model.TraceCandidateDiagnosticEmbeddingSimilarityCandidate):
		return string(model.TraceVerificationModeCandidateSearch)
	default:
		return string(model.TraceVerificationModeLLMEvidence)
	}
}

func buildTraceEdgeEvidenceRef(
	entity model.Entity,
	documentByID map[string]model.Document,
	sectionByID map[string]model.Section,
	sectionID string,
	quoteStartOffset *int,
	quoteEndOffset *int,
) *report.EvidenceRefReport {
	doc, ok := documentByID[entity.DocumentID]
	if !ok {
		return nil
	}
	if sectionID == "" {
		sectionID = entity.SectionID
	}
	if quoteStartOffset == nil {
		quoteStartOffset = entity.QuoteStartOffset
	}
	if quoteEndOffset == nil {
		quoteEndOffset = entity.QuoteEndOffset
	}

	ref := &report.EvidenceRefReport{
		DocumentID:       entity.DocumentID,
		DocumentPath:     doc.Path,
		Quote:            entity.SourceQuote,
		QuoteStartOffset: quoteStartOffset,
		QuoteEndOffset:   quoteEndOffset,
		TrustGrade:       string(entity.TrustGrade),
	}
	if sectionID != "" {
		ref.SectionID = sectionID
		ref.SectionHeading = sectionByID[sectionID].Heading
	}
	return ref
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
