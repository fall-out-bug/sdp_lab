// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// detectPII scans table columns for PII indicators.
// Exact match (column == pattern) yields 0.95 confidence.
// Partial match (column contains pattern) yields 0.75 confidence.
// Exact is checked first across all patterns to avoid a partial match
// shadowing a later exact match.
func detectPII(tables []architect.Table) []architect.PIIColumn {
	var results []architect.PIIColumn
	for _, t := range tables {
		for _, c := range t.Columns {
			colLower := strings.ToLower(c.Name)

			// Pass 1: exact match (highest priority).
			matched := false
			for _, pat := range piiExactPatterns {
				if colLower == pat {
					results = append(results, architect.PIIColumn{
						Table:      t.Name,
						Column:     c.Name,
						PIIType:    piiTypeMap[pat],
						Pattern:    pat,
						Confidence: 0.95,
					})
					matched = true
					break
				}
			}
			if matched {
				continue
			}

			// Pass 2: partial (contains) match.
			for _, pat := range piiExactPatterns {
				if strings.Contains(colLower, pat) {
					results = append(results, architect.PIIColumn{
						Table:      t.Name,
						Column:     c.Name,
						PIIType:    piiTypeMap[pat],
						Pattern:    pat,
						Confidence: 0.75,
					})
					break
				}
			}
		}
	}
	return results
}

// GroupPIIByType groups PII columns by their PII type.
func GroupPIIByType(piiCols []architect.PIIColumn) map[string][]architect.PIIColumn {
	groups := make(map[string][]architect.PIIColumn)
	for _, p := range piiCols {
		groups[p.PIIType] = append(groups[p.PIIType], p)
	}
	return groups
}

// GroupPIIByTable groups PII columns by their table.
func GroupPIIByTable(piiCols []architect.PIIColumn) map[string][]architect.PIIColumn {
	groups := make(map[string][]architect.PIIColumn)
	for _, p := range piiCols {
		groups[p.Table] = append(groups[p.Table], p)
	}
	return groups
}

// SortPIIColumns sorts PII columns by table then column for deterministic output.
func SortPIIColumns(piiCols []architect.PIIColumn) {
	sort.Slice(piiCols, func(i, j int) bool {
		if piiCols[i].Table == piiCols[j].Table {
			return piiCols[i].Column < piiCols[j].Column
		}
		return piiCols[i].Table < piiCols[j].Table
	})
}

// GetPIIStats returns statistics about detected PII columns.
func GetPIIStats(piiCols []architect.PIIColumn) map[string]int {
	stats := map[string]int{
		"total":       len(piiCols),
		"high_conf":   0, // confidence >= 0.9
		"medium_conf": 0, // confidence >= 0.7 and < 0.9
		"low_conf":    0, // confidence < 0.7
	}

	for _, p := range piiCols {
		switch {
		case p.Confidence >= 0.9:
			stats["high_conf"]++
		case p.Confidence >= 0.7:
			stats["medium_conf"]++
		default:
			stats["low_conf"]++
		}
	}

	return stats
}

// FilterPIIByConfidence filters PII columns by minimum confidence level.
func FilterPIIByConfidence(piiCols []architect.PIIColumn, minConfidence float64) []architect.PIIColumn {
	var filtered []architect.PIIColumn
	for _, p := range piiCols {
		if p.Confidence >= minConfidence {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterPIIByType filters PII columns by PII type.
func FilterPIIByType(piiCols []architect.PIIColumn, piiType string) []architect.PIIColumn {
	var filtered []architect.PIIColumn
	for _, p := range piiCols {
		if p.PIIType == piiType {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterPIIByTable filters PII columns by table name.
func FilterPIIByTable(piiCols []architect.PIIColumn, table string) []architect.PIIColumn {
	var filtered []architect.PIIColumn
	for _, p := range piiCols {
		if p.Table == table {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// GeneratePIIReport generates a human-readable PII report.
func GeneratePIIReport(piiCols []architect.PIIColumn) string {
	if len(piiCols) == 0 {
		return "No PII columns detected."
	}

	var report strings.Builder
	report.WriteString("PII Detection Report\n")
	report.WriteString("===================\n\n")

	// Group by table
	byTable := GroupPIIByTable(piiCols)

	// Sort tables for deterministic output
	tables := make([]string, 0, len(byTable))
	for table := range byTable {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	for _, table := range tables {
		cols := byTable[table]
		report.WriteString(fmt.Sprintf("Table: %s (%d columns)\n", table, len(cols)))

		for _, col := range cols {
			report.WriteString(fmt.Sprintf("  - %s (%s, confidence: %.2f)\n",
				col.Column, col.PIIType, col.Confidence))
		}
		report.WriteString("\n")
	}

	// Summary stats
	stats := GetPIIStats(piiCols)
	report.WriteString(fmt.Sprintf("Summary: %d total PII columns (%d high confidence)\n",
		stats["total"], stats["high_conf"]))

	return report.String()
}

// CheckPIIEncryption attempts to detect if PII columns might be encrypted.
// Checks for common encryption indicators in column names and types.
func CheckPIIEncryption(column architect.Column) bool {
	colLower := strings.ToLower(column.Name)
	typeLower := strings.ToLower(column.Type)

	// Check for encryption indicators in column name
	encryptedPatterns := []string{
		"encrypted", "hashed", "bcrypt", "sha256", "sha512", "md5",
		"cipher", "crypt", "salt", "digest", "hash",
	}

	for _, pattern := range encryptedPatterns {
		if strings.Contains(colLower, pattern) {
			return true
		}
	}

	// Check for binary/varbinary types which might store encrypted data
	encryptedTypes := []string{
		"varbinary", "binary", "blob", "bytea",
	}

	for _, encType := range encryptedTypes {
		if strings.Contains(typeLower, encType) {
			return true
		}
	}

	return false
}

// EnrichPIIWithEncryptionDetection adds encryption detection to PII columns.
func EnrichPIIWithEncryptionDetection(piiCols []architect.PIIColumn, tables []architect.Table) []architect.PIIColumn {
	// Create a map of table columns
	tableCols := make(map[string]map[string]architect.Column)
	for _, t := range tables {
		tableCols[t.Name] = make(map[string]architect.Column)
		for _, c := range t.Columns {
			tableCols[t.Name][c.Name] = c
		}
	}

	// Check encryption for each PII column
	for i := range piiCols {
		if cols, ok := tableCols[piiCols[i].Table]; ok {
			if col, ok := cols[piiCols[i].Column]; ok {
				piiCols[i].EncryptionDetected = CheckPIIEncryption(col)
			}
		}
	}

	return piiCols
}
