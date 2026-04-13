package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const SchemaVersion = "2"

// AuditReport is the top-level JSON report structure for report.v2.json.
type AuditReport struct {
	SchemaVersion   string                 `json:"schema_version"`
	AuditScope      AuditScopeReport       `json:"audit_scope"`
	TrustSummary    TrustSummaryReport     `json:"trust_summary"`
	CorpusQuality   CorpusQualityReport    `json:"corpus_quality"`
	Levels          []LevelReport          `json:"levels"`
	Documents       []DocumentReport       `json:"documents,omitempty"`
	Sections        []SectionReport        `json:"sections,omitempty"`
	Entities        []EntityReport         `json:"entities,omitempty"`
	TraceCandidates []TraceCandidateReport `json:"trace_candidates,omitempty"`
	VerifiedTraces  []VerifiedTraceReport  `json:"verified_traces,omitempty"`
	TraceGraph      TraceGraphReport       `json:"trace_graph"`
	FindingsGrouped []FindingReport        `json:"findings_grouped"`
	Coverage        CoverageBlockReport    `json:"coverage"`
	EvidencePack    EvidencePackReport     `json:"evidence_pack"`
}

type AuditScopeReport struct {
	ProjectName        string                `json:"project_name"`
	ProjectDescription string                `json:"project_description,omitempty"`
	OutputDir          string                `json:"output_dir,omitempty"`
	OutputLang         string                `json:"output_lang,omitempty"`
	GeneratedAt        string                `json:"generated_at"`
	SourceDirs         []string              `json:"source_dirs,omitempty"`
	Exclude            []string              `json:"exclude,omitempty"`
	Models             AuditModelsReport     `json:"models"`
	Thresholds         AuditThresholdsReport `json:"thresholds"`
}

type AuditModelsReport struct {
	DefaultModel   string `json:"default_model,omitempty"`
	ExtractModel   string `json:"extract_model,omitempty"`
	EmbeddingModel string `json:"embedding_model,omitempty"`
}

type AuditThresholdsReport struct {
	Similarity           float64 `json:"similarity"`
	TraceConfidence      float64 `json:"trace_confidence"`
	AutoVerifySimilarity float64 `json:"auto_verify_similarity"`
	CoverageWarn         float64 `json:"coverage_warn"`
	LLMVerifyBudget      int     `json:"llm_verify_budget"`
}

type TrustSummaryReport struct {
	OverallStatus string             `json:"overall_status"`
	Entities      EntityTrustCounts  `json:"entities"`
	Traces        TraceTrustCounts   `json:"traces"`
	Findings      FindingCountReport `json:"findings"`
	Disclaimers   []string           `json:"disclaimers,omitempty"`
}

type EntityTrustCounts struct {
	TotalAdmitted int `json:"total_admitted"`
	Verified      int `json:"verified"`
	Suspect       int `json:"suspect"`
	Rejected      int `json:"rejected"`
	Unknown       int `json:"unknown"`
}

type TraceTrustCounts struct {
	Verified          int            `json:"verified"`
	Candidates        int            `json:"candidates"`
	VerificationModes map[string]int `json:"verification_modes,omitempty"`
}

type FindingCountReport struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Warn     int `json:"warn"`
	Info     int `json:"info"`
}

type CorpusQualityReport struct {
	TotalIssues       int                      `json:"total_issues"`
	CriticalDocuments int                      `json:"critical_documents"`
	FlagCounts        map[string]int           `json:"flag_counts,omitempty"`
	Documents         []CorpusQualityDocReport `json:"documents,omitempty"`
}

type CorpusQualityDocReport struct {
	DocumentID   string   `json:"document_id"`
	DocumentPath string   `json:"document_path"`
	LevelID      string   `json:"level_id,omitempty"`
	LevelName    string   `json:"level_name,omitempty"`
	Severity     string   `json:"severity"`
	IssueCount   int      `json:"issue_count"`
	Flags        []string `json:"flags,omitempty"`
	FindingIDs   []string `json:"finding_ids,omitempty"`
	SectionIDs   []string `json:"section_ids,omitempty"`
	EntityIDs    []string `json:"entity_ids,omitempty"`
}

type DocumentReport struct {
	ID           string `json:"id"`
	Path         string `json:"path"`
	Name         string `json:"name"`
	LevelID      string `json:"level_id"`
	LevelName    string `json:"level_name,omitempty"`
	ContentHash  string `json:"content_hash"`
	Version      int    `json:"version"`
	EntityCount  int    `json:"entity_count"`
	SectionCount int    `json:"section_count"`
}

type SectionReport struct {
	ID           string   `json:"id"`
	DocumentID   string   `json:"document_id"`
	DocumentPath string   `json:"document_path"`
	LevelID      string   `json:"level_id,omitempty"`
	LevelName    string   `json:"level_name,omitempty"`
	Ordinal      int      `json:"ordinal"`
	Heading      string   `json:"heading,omitempty"`
	Label        string   `json:"label"`
	CharStart    int      `json:"char_start"`
	CharEnd      int      `json:"char_end"`
	Preview      string   `json:"preview"`
	EntityCount  int      `json:"entity_count"`
	QualityFlags []string `json:"quality_flags,omitempty"`
}

type EntityReport struct {
	ID                  string   `json:"id"`
	Type                string   `json:"type"`
	Title               string   `json:"title"`
	Description         string   `json:"description,omitempty"`
	TitleOriginal       string   `json:"title_original,omitempty"`
	DescriptionOriginal string   `json:"description_original,omitempty"`
	Lang                string   `json:"lang,omitempty"`
	LanguageMismatch    bool     `json:"language_mismatch,omitempty"`
	LevelID             string   `json:"level_id"`
	LevelName           string   `json:"level_name,omitempty"`
	DocumentID          string   `json:"document_id"`
	DocumentPath        string   `json:"document_path"`
	SectionID           string   `json:"section_id,omitempty"`
	SectionHeading      string   `json:"section_heading,omitempty"`
	SourceQuote         string   `json:"source_quote,omitempty"`
	QuoteStartOffset    *int     `json:"quote_start_offset,omitempty"`
	QuoteEndOffset      *int     `json:"quote_end_offset,omitempty"`
	TrustGrade          string   `json:"trust_grade,omitempty"`
	QualityFlags        []string `json:"quality_flags,omitempty"`
}

type TraceCandidateReport struct {
	ID             string  `json:"id"`
	SourceEntityID string  `json:"source_entity_id"`
	TargetEntityID string  `json:"target_entity_id"`
	Similarity     float64 `json:"similarity"`
	Verified       bool    `json:"verified,omitempty"`
	TraceID        string  `json:"trace_id,omitempty"`
	DiagnosticCode string  `json:"diagnostic_code,omitempty"`
}

type VerifiedTraceReport struct {
	ID               string              `json:"id"`
	SourceEntityID   string              `json:"source_entity_id"`
	TargetEntityID   string              `json:"target_entity_id"`
	Relation         string              `json:"relation"`
	Confidence       float64             `json:"confidence"`
	SimilarityScore  float64             `json:"similarity_score"`
	Justification    string              `json:"justification"`
	VerificationMode string              `json:"verification_mode"`
	TrustGrade       string              `json:"trust_grade,omitempty"`
	TraceEvidence    TraceEvidenceReport `json:"trace_evidence"`
}

type TraceGraphReport struct {
	Nodes []TraceNodeReport `json:"nodes"`
	Edges []TraceEdgeReport `json:"edges"`
	Paths []TracePathReport `json:"paths"`
}

type TraceNodeReport struct {
	ID             string `json:"id"`
	EntityID       string `json:"entity_id"`
	Type           string `json:"type,omitempty"`
	Title          string `json:"title"`
	LevelID        string `json:"level_id"`
	LevelName      string `json:"level_name,omitempty"`
	DocumentID     string `json:"document_id"`
	DocumentPath   string `json:"document_path"`
	SectionID      string `json:"section_id,omitempty"`
	SectionHeading string `json:"section_heading,omitempty"`
	SourceQuote    string `json:"source_quote,omitempty"`
	TrustGrade     string `json:"trust_grade,omitempty"`
	Lang           string `json:"lang,omitempty"`
}

type TraceEdgeReport struct {
	ID                string             `json:"id"`
	SourceNodeID      string             `json:"source_node_id"`
	TargetNodeID      string             `json:"target_node_id"`
	SourceEntityID    string             `json:"source_entity_id"`
	TargetEntityID    string             `json:"target_entity_id"`
	Relation          string             `json:"relation,omitempty"`
	Direction         string             `json:"direction,omitempty"`
	Status            string             `json:"status"`
	VerificationMode  string             `json:"verification_mode"`
	Confidence        float64            `json:"confidence"`
	Similarity        float64            `json:"similarity,omitempty"`
	Reason            string             `json:"reason,omitempty"`
	TrustGrade        string             `json:"trust_grade,omitempty"`
	SourceEvidenceRef *EvidenceRefReport `json:"source_evidence_ref,omitempty"`
	TargetEvidenceRef *EvidenceRefReport `json:"target_evidence_ref,omitempty"`
}

type TracePathReport struct {
	ID             string   `json:"id"`
	EntryNodeID    string   `json:"entry_node_id"`
	TerminalNodeID string   `json:"terminal_node_id"`
	NodeIDs        []string `json:"node_ids"`
	EdgeIDs        []string `json:"edge_ids"`
	Status         string   `json:"status"`
	HopCount       int      `json:"hop_count"`
}

type TraceEvidenceReport struct {
	Source EvidenceRefReport `json:"source"`
	Target EvidenceRefReport `json:"target"`
}

type EvidenceRefReport struct {
	DocumentID       string `json:"document_id"`
	DocumentPath     string `json:"document_path"`
	SectionID        string `json:"section_id,omitempty"`
	SectionHeading   string `json:"section_heading,omitempty"`
	Quote            string `json:"quote,omitempty"`
	QuoteStartOffset *int   `json:"quote_start_offset,omitempty"`
	QuoteEndOffset   *int   `json:"quote_end_offset,omitempty"`
	TrustGrade       string `json:"trust_grade,omitempty"`
}

type LevelReport struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Rank          int      `json:"rank"`
	Description   string   `json:"description,omitempty"`
	Patterns      []string `json:"patterns,omitempty"`
	EntityCount   int      `json:"entity_count"`
	DocumentCount int      `json:"document_count"`
}

type FindingReport struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Severity       string   `json:"severity"`
	ClusterKey     string   `json:"cluster_key,omitempty"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
	EntityIDs      []string `json:"entity_ids,omitempty"`
	DocumentIDs    []string `json:"document_ids,omitempty"`
	DocumentPaths  []string `json:"document_paths,omitempty"`
	SectionIDs     []string `json:"section_ids,omitempty"`
	SectionLabels  []string `json:"section_labels,omitempty"`
	AffectedCount  int      `json:"affected_count"`
	Confidence     float64  `json:"confidence"`
}

type CoverageBlockReport struct {
	AverageLevelPct float64               `json:"average_level_pct"`
	ByLevel         []CoverageScopeReport `json:"by_level,omitempty"`
	ByDocument      []CoverageScopeReport `json:"by_document,omitempty"`
	BySection       []CoverageScopeReport `json:"by_section,omitempty"`
	LowestDocuments []CoverageScopeReport `json:"lowest_documents,omitempty"`
	LowestSections  []CoverageScopeReport `json:"lowest_sections,omitempty"`
}

type CoverageScopeReport struct {
	ScopeType      string  `json:"scope_type"`
	ScopeID        string  `json:"scope_id"`
	ScopeLabel     string  `json:"scope_label"`
	LevelID        string  `json:"level_id,omitempty"`
	LevelName      string  `json:"level_name,omitempty"`
	DocumentID     string  `json:"document_id,omitempty"`
	DocumentPath   string  `json:"document_path,omitempty"`
	SectionID      string  `json:"section_id,omitempty"`
	SectionHeading string  `json:"section_heading,omitempty"`
	Total          int     `json:"total"`
	Traced         int     `json:"traced"`
	Pct            float64 `json:"pct"`
}

type EvidencePackReport struct {
	Artifacts                  []ArtifactReport `json:"artifacts,omitempty"`
	DocumentCount              int              `json:"document_count"`
	TraceCandidateCount        int              `json:"trace_candidate_count"`
	RejectedEntityCount        int              `json:"rejected_entity_count"`
	EntitiesWithSourceQuotes   int              `json:"entities_with_source_quotes"`
	VerifiedTracesWithEvidence int              `json:"verified_traces_with_evidence"`
}

type ArtifactReport struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

// WriteJSON writes the canonical report.v2.json and a compatibility alias report.json.
func WriteJSON(rpt *AuditReport, outputDir string) error {
	data, err := json.MarshalIndent(rpt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	v2Path := filepath.Join(outputDir, "report.v2.json")
	if err := os.WriteFile(v2Path, data, 0644); err != nil {
		return fmt.Errorf("write report.v2.json: %w", err)
	}
	compatPath := filepath.Join(outputDir, "report.json")
	if err := os.WriteFile(compatPath, data, 0644); err != nil {
		return fmt.Errorf("write report.json compatibility alias: %w", err)
	}
	return nil
}

// WriteHTML writes an HTML audit report to disk.
func WriteHTML(rpt *AuditReport, outputDir string) error {
	html := buildHTML(rpt)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	return os.WriteFile(filepath.Join(outputDir, "report.html"), []byte(html), 0644)
}
