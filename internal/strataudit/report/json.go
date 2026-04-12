package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AuditReport is the top-level JSON report structure.
type AuditReport struct {
	Project  string           `json:"project"`
	Levels   []LevelReport    `json:"levels"`
	Entities []EntityReport   `json:"entities,omitempty"`
	Traces   []TraceReport    `json:"traces,omitempty"`
	Findings []FindingReport  `json:"findings"`
	Coverage []CoverageReport `json:"coverage"`
	Summary  SummaryReport    `json:"summary"`
}

type EntityReport struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	LevelID     string `json:"level_id"`
	DocumentID  string `json:"document_id"`
}

type TraceReport struct {
	ID             string  `json:"id"`
	SourceEntityID string  `json:"source_entity_id"`
	TargetEntityID string  `json:"target_entity_id"`
	Relation       string  `json:"relation"`
	Confidence     float64 `json:"confidence"`
	Justification  string  `json:"justification"`
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
