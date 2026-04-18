// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"regexp"
	"sort"
	"strings"

	"sdp_dev/internal/architect"
)

// ---------------------------------------------------------------------------
// Compiled regexes for DDL parsing
// ---------------------------------------------------------------------------

var (
	// CREATE TABLE [schema.]name ( ... )
	reCreateTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` +
		`(?:["\x60]?(\w+)["\x60]?\.)?["\x60]?(\w+)["\x60]?\s*\(`)

	// Column line inside CREATE TABLE: name TYPE [constraints]
	reColumnDef = regexp.MustCompile(`(?i)^\s*["\x60]?(\w+)["\x60]?\s+(\w+(?:\s*\([^)]*\))?)`)

	// PRIMARY KEY(col, ...)
	rePrimaryKey = regexp.MustCompile(`(?i)PRIMARY\s+KEY\s*\(\s*([^)]+)\)`)

	// FOREIGN KEY(col) REFERENCES [schema.]table(col)
	reForeignKey = regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(\s*["\x60]?(\w+)["\x60]?\s*\)\s*REFERENCES\s+(?:["\x60]?(\w+)["\x60]?\.)?["\x60]?(\w+)["\x60]?\s*\(\s*["\x60]?(\w+)["\x60]?\s*\)`)

	// CREATE [UNIQUE] INDEX name ON [schema.]table(cols)
	reCreateIndex = regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?["\x60]?(\w+)["\x60]?\s+ON\s+(?:["\x60]?(\w+)["\x60]?\.)?["\x60]?(\w+)["\x60]?\s*\(\s*([^)]+)\)`)

	// CREATE [MATERIALIZED] VIEW [schema.]name
	reCreateView = regexp.MustCompile(`(?i)CREATE\s+(MATERIALIZED\s+)?VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:["\x60]?(\w+)["\x60]?\.)?["\x60]?(\w+)["\x60]?`)

	// CREATE [OR REPLACE] FUNCTION/PROCEDURE name
	reCreateProc = regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?(?:FUNCTION|PROCEDURE)\s+(?:["\x60]?(\w+)["\x60]?\.)?["\x60]?(\w+)["\x60]?`)

	// Inline PRIMARY KEY on column def
	reInlinePK = regexp.MustCompile(`(?i)\bPRIMARY\s+KEY\b`)

	// NOT NULL on column def
	reNotNull = regexp.MustCompile(`(?i)\bNOT\s+NULL\b`)
)

// parseTables extracts Table and ForeignKey definitions from SQL text.
func parseTables(sql, file string) ([]architect.Table, []architect.ForeignKey) {
	var tables []architect.Table
	var fks []architect.ForeignKey

	// Find each CREATE TABLE ... ( ... ) block.
	locs := reCreateTable.FindAllStringSubmatchIndex(sql, -1)
	for _, loc := range locs {
		// reCreateTable has two capture groups: [1]=schema (optional), [2]=table name.
		schemaName := ""
		tableName := ""
		// group indices: full(0,1), schema(2,3), table(4,5)
		if loc[4] >= 0 && loc[5] >= 0 {
			tableName = sql[loc[4]:loc[5]]
		} else if loc[2] >= 0 && loc[3] >= 0 {
			tableName = sql[loc[2]:loc[3]]
		}
		if loc[2] >= 0 && loc[3] >= 0 && loc[4] >= 0 {
			schemaName = sql[loc[2]:loc[3]]
		}
		if tableName == "" {
			continue
		}

		// Find the matching closing paren for the column block.
		bodyStart := loc[1] // right after the opening paren
		body := extractParenBody(sql, bodyStart-1)
		if body == "" {
			continue
		}

		// Parse primary key constraints declared at table level.
		pkCols := make(map[string]bool)
		for _, m := range rePrimaryKey.FindAllStringSubmatch(body, -1) {
			for _, c := range strings.Split(m[1], ",") {
				c = strings.TrimSpace(c)
				c = strings.Trim(c, "\"`")
				if c != "" {
					pkCols[strings.ToLower(c)] = true
				}
			}
		}

		// Parse foreign keys.
		// reForeignKey groups: [1]=from_col, [2]=ref_schema(optional), [3]=ref_table, [4]=ref_col
		for _, m := range reForeignKey.FindAllStringSubmatch(body, -1) {
			refTable := m[3]
			if m[2] != "" {
				refTable = m[2] + "." + refTable
			}
			fks = append(fks, architect.ForeignKey{
				FromTable:  tableName,
				FromColumn: m[1],
				ToTable:    refTable,
				ToColumn:   m[4],
				File:       file,
			})
		}

		// Parse columns.
		var columns []architect.Column
		lines := strings.Split(body, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip constraints, blank lines.
			if line == "" {
				continue
			}
			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "PRIMARY") ||
				strings.HasPrefix(upper, "FOREIGN") ||
				strings.HasPrefix(upper, "CONSTRAINT") ||
				strings.HasPrefix(upper, "UNIQUE") ||
				strings.HasPrefix(upper, "CHECK") ||
				strings.HasPrefix(upper, "INDEX") ||
				strings.HasPrefix(upper, ")") {
				continue
			}

			// Split by comma and process each column definition
			colDefs := splitColumnDefs(line)
			for _, colDef := range colDefs {
				colDef = strings.TrimSpace(colDef)
				if colDef == "" {
					continue
				}

				m := reColumnDef.FindStringSubmatch(colDef)
				if m == nil {
					continue
				}
				colName := m[1]
				colType := strings.TrimSpace(m[2])

				col := architect.Column{
					Name: colName,
					Type: colType,
				}
				if reInlinePK.MatchString(colDef) || pkCols[strings.ToLower(colName)] {
					col.PrimaryKey = true
				}
				if reNotNull.MatchString(colDef) || col.PrimaryKey {
					col.NotNull = true
				} else {
					// If no NOT NULL and no PRIMARY KEY, the column is nullable.
					col.Nullable = true
				}
				columns = append(columns, col)
			}
		}

		tbl := architect.Table{
			Name:    tableName,
			Columns: columns,
			File:    file,
		}
		if schemaName != "" {
			tbl.Schema = schemaName
		}
		tables = append(tables, tbl)
	}

	return tables, fks
}

// extractParenBody returns the text inside the outermost parentheses starting
// at position openIdx (which must point to '(').
func extractParenBody(s string, openIdx int) string {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '(' {
		return ""
	}
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[openIdx+1 : i]
			}
		}
	}
	return ""
}

// parseIndexes extracts Index definitions from SQL text.
func parseIndexes(sql, file string) []architect.Index {
	var indexes []architect.Index
	for _, m := range reCreateIndex.FindAllStringSubmatch(sql, -1) {
		unique := strings.TrimSpace(m[1]) != ""
		// groups: [1]=unique, [2]=name, [3]=schema(optional), [4]=table, [5]=cols
		cols := splitTrimmed(m[5])
		indexes = append(indexes, architect.Index{
			Name:    m[2],
			Table:   m[4],
			Columns: cols,
			Unique:  unique,
			File:    file,
		})
	}
	return indexes
}

// parseViews extracts View definitions from SQL text.
func parseViews(sql, file string) []architect.View {
	var views []architect.View
	for _, m := range reCreateView.FindAllStringSubmatch(sql, -1) {
		mat := strings.TrimSpace(m[1]) != ""
		// groups: [1]=materialized, [2]=schema(optional), [3]=name
		views = append(views, architect.View{
			Name:         m[3],
			Materialized: mat,
			File:         file,
		})
	}
	return views
}

// parseStoredProcs extracts CREATE FUNCTION / CREATE PROCEDURE definitions.
func parseStoredProcs(sql, file string) []architect.StoredProc {
	var procs []architect.StoredProc
	for _, m := range reCreateProc.FindAllStringSubmatch(sql, -1) {
		// groups: [1]=schema(optional), [2]=name
		procName := m[2]
		if m[1] != "" {
			procName = m[1] + "." + procName
		}
		procs = append(procs, architect.StoredProc{
			Name: procName,
			Path: file,
		})
	}
	return procs
}

// GetDDLStats returns statistics about parsed DDL elements.
func GetDDLStats(tables []architect.Table, indexes []architect.Index, views []architect.View, procs []architect.StoredProc) map[string]int {
	return map[string]int{
		"tables":        len(tables),
		"indexes":       len(indexes),
		"views":         len(views),
		"stored_procs":  len(procs),
		"unique_index":  countUniqueIndexes(indexes),
		"materialized":  countMaterializedViews(views),
	}
}

// countUniqueIndexes returns the number of unique indexes.
func countUniqueIndexes(indexes []architect.Index) int {
	count := 0
	for _, idx := range indexes {
		if idx.Unique {
			count++
		}
	}
	return count
}

// countMaterializedViews returns the number of materialized views.
func countMaterializedViews(views []architect.View) int {
	count := 0
	for _, v := range views {
		if v.Materialized {
			count++
		}
	}
	return count
}

// SortTables sorts tables by name for deterministic output.
func SortTables(tables []architect.Table) {
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})
}

// SortIndexes sorts indexes by table then name for deterministic output.
func SortIndexes(indexes []architect.Index) {
	sort.Slice(indexes, func(i, j int) bool {
		if indexes[i].Table == indexes[j].Table {
			return indexes[i].Name < indexes[j].Name
		}
		return indexes[i].Table < indexes[j].Table
	})
}

// SortViews sorts views by name for deterministic output.
func SortViews(views []architect.View) {
	sort.Slice(views, func(i, j int) bool {
		return views[i].Name < views[j].Name
	})
}

// splitColumnDefs splits a line by commas, respecting parentheses.
func splitColumnDefs(line string) []string {
	var parts []string
	var current strings.Builder
	depth := 0

	for _, r := range line {
		switch r {
		case '(':
			depth++
			current.WriteRune(r)
		case ')':
			depth--
			current.WriteRune(r)
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(current.String())
				if part != "" {
					parts = append(parts, part)
				}
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	// Add the last part
	part := strings.TrimSpace(current.String())
	if part != "" {
		parts = append(parts, part)
	}

	return parts
}
