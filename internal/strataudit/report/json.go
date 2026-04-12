package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AuditReport is the top-level JSON report structure.
type AuditReport struct {
	Project         string                 `json:"project"`
	Levels          []LevelReport          `json:"levels"`
	Documents       []DocumentReport       `json:"documents,omitempty"`
	Sections        []SectionReport        `json:"sections,omitempty"`
	Entities        []EntityReport         `json:"entities,omitempty"`
	TraceCandidates []TraceCandidateReport `json:"trace_candidates,omitempty"`
	VerifiedTraces  []VerifiedTraceReport  `json:"verified_traces,omitempty"`
	Findings        []FindingReport        `json:"findings"`
	Coverage        []CoverageReport       `json:"coverage"`
	Summary         SummaryReport          `json:"summary"`
}

type DocumentReport struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	LevelID     string `json:"level_id"`
	ContentHash string `json:"content_hash"`
	Version     int    `json:"version"`
}

type SectionReport struct {
	ID           string   `json:"id"`
	DocumentID   string   `json:"document_id"`
	Ordinal      int      `json:"ordinal"`
	Heading      string   `json:"heading,omitempty"`
	CharStart    int      `json:"char_start"`
	CharEnd      int      `json:"char_end"`
	Preview      string   `json:"preview"`
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
	DocumentID          string   `json:"document_id"`
	SectionID           string   `json:"section_id,omitempty"`
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
	ID                     string  `json:"id"`
	SourceEntityID         string  `json:"source_entity_id"`
	TargetEntityID         string  `json:"target_entity_id"`
	Relation               string  `json:"relation"`
	Confidence             float64 `json:"confidence"`
	SimilarityScore        float64 `json:"similarity_score"`
	Justification          string  `json:"justification"`
	VerificationMode       string  `json:"verification_mode"`
	TrustGrade             string  `json:"trust_grade,omitempty"`
	SourceSectionID        string  `json:"source_section_id,omitempty"`
	TargetSectionID        string  `json:"target_section_id,omitempty"`
	SourceQuoteStartOffset *int    `json:"source_quote_start_offset,omitempty"`
	SourceQuoteEndOffset   *int    `json:"source_quote_end_offset,omitempty"`
	TargetQuoteStartOffset *int    `json:"target_quote_start_offset,omitempty"`
	TargetQuoteEndOffset   *int    `json:"target_quote_end_offset,omitempty"`
}

type LevelReport struct {
	Name     string `json:"name"`
	Rank     int    `json:"rank"`
	Entities int    `json:"entities"`
}

type FindingReport struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	EntityIDs   []string `json:"entity_ids"`
	Confidence  float64  `json:"confidence"`
}

type CoverageReport struct {
	Level  string  `json:"level"`
	Total  int     `json:"total"`
	Traced int     `json:"traced"`
	Pct    float64 `json:"pct"`
}

type SummaryReport struct {
	TotalEntities int     `json:"total_entities"`
	TotalFindings int     `json:"total_findings"`
	CriticalCount int     `json:"critical_count"`
	WarnCount     int     `json:"warn_count"`
	AvgCoverage   float64 `json:"avg_coverage"`
}

// WriteJSON writes a JSON audit report to disk.
func WriteJSON(rpt *AuditReport, outputDir string) error {
	data, err := json.MarshalIndent(rpt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	return os.WriteFile(filepath.Join(outputDir, "report.json"), data, 0644)
}

// WriteHTML writes an HTML audit report to disk.
func WriteHTML(rpt *AuditReport, outputDir string) error {
	html := buildHTML(rpt)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	return os.WriteFile(filepath.Join(outputDir, "report.html"), []byte(html), 0644)
}
