// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/architect"
)

// SQLExtractor implements architect.Extractor for SQL schema analysis.
type SQLExtractor struct{}

// Name returns the extractor name.
func (e *SQLExtractor) Name() string { return "sql" }

// NewSQLExtractor returns a new SQLExtractor.
func NewSQLExtractor() *SQLExtractor {
	return &SQLExtractor{}
}

// Extract walks root and returns a ProfileFragment with SQLAnalysis populated.
func (e *SQLExtractor) Extract(ctx context.Context, root string) (*architect.ProfileFragment, error) {
	var (
		sqlFiles []fileContent
		ormFiles []fileContent
	)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		rel, _ := filepath.Rel(root, path)

		if sqlExtensions[ext] {
			// Skip test fixtures.
			if isTestPath(rel) {
				return nil
			}

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			sqlFiles = append(sqlFiles, fileContent{rel: rel, data: string(data)})
		}
		if ormExtensions[ext] {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			ormFiles = append(ormFiles, fileContent{rel: rel, data: string(data)})
		}
		// .prisma files also need ORM scan
		if ext == ".prisma" && !ormExtensions[ext] {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			ormFiles = append(ormFiles, fileContent{rel: rel, data: string(data)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// If nothing found, return empty fragment.
	if len(sqlFiles) == 0 && len(ormFiles) == 0 {
		return &architect.ProfileFragment{SQLAnalysis: &architect.SQLAnalysis{}}, nil
	}

	analysis := &architect.SQLAnalysis{}

	// Parse SQL files.
	for _, f := range sqlFiles {
		tables, fks := parseTables(f.data, f.rel)
		analysis.Tables = append(analysis.Tables, tables...)
		analysis.ForeignKeys = append(analysis.ForeignKeys, fks...)
		analysis.Indexes = append(analysis.Indexes, parseIndexes(f.data, f.rel)...)
		analysis.Views = append(analysis.Views, parseViews(f.data, f.rel)...)
		analysis.StoredProcs = append(analysis.StoredProcs, parseStoredProcs(f.data, f.rel)...)
	}

	// Migration detection.
	analysis.Migrations = detectMigrations(root)

	// ORM model detection.
	for _, f := range ormFiles {
		analysis.ORMModels = append(analysis.ORMModels, detectORM(f.data, f.rel)...)
	}

	// PII column detection.
	analysis.PIIColumns = detectPII(analysis.Tables)

	// Data domain clustering.
	analysis.DataDomains = clusterDomains(analysis.Tables, analysis.ForeignKeys)

	return &architect.ProfileFragment{SQLAnalysis: analysis}, nil
}

// fileContent is a helper that pairs a relative path with file contents.
type fileContent struct {
	rel  string
	data string
}
