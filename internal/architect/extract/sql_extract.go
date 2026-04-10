// Package extract provides extractors for the AI Architect module.
package extract

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"sdp_dev/internal/architect"
)

// sqlExtensions lists file extensions scanned for SQL content.
var sqlExtensions = map[string]bool{
	".sql": true,
}

// ormExtensions lists file extensions scanned for ORM models.
var ormExtensions = map[string]bool{
	".go":      true,
	".py":      true,
	".java":    true,
	".prisma":  true,
	".graphql": true,
}

// migrationDirs lists well-known migration directory basenames.
var migrationDirs = []string{
	"migrations",
	"migrate",
	"alembic/versions",
	"flyway",
}

// migrationRoots are top-level directories that may contain migration sub-dirs.
var migrationRoots = []string{
	"migrations",
	"db/migrate",
	"alembic/versions",
	"flyway",
}

// piiPatterns maps PII indicator patterns to their exact/partial confidence.
// Exact match means the column name equals the pattern; partial means it contains it.
var piiExactPatterns = []string{
	"email",
	"phone",
	"ssn",
	"birth_date",
	"address",
	"first_name",
	"last_name",
	"ip_address",
	"credit_card",
}

// ---------------------------------------------------------------------------
// Compiled regexes for DDL parsing
// ---------------------------------------------------------------------------

var (
	// CREATE TABLE name ( ... )
	reCreateTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` +
		`["\x60]?(\w+)["\x60]?\s*\(`)

	// Column line inside CREATE TABLE: name TYPE [constraints]
	reColumnDef = regexp.MustCompile(`(?i)^\s*["\x60]?(\w+)["\x60]?\s+(\w[\w() ,]*?)` +
		`(\s+PRIMARY\s+KEY|\s+NOT\s+NULL|\s+UNIQUE|\s+DEFAULT\s+\S+|\s+REFERENCES\s+\S+|\s+CHECK\s*\([^)]*\))*\s*,?\s*$`)

	// PRIMARY KEY(col, ...)
	rePrimaryKey = regexp.MustCompile(`(?i)PRIMARY\s+KEY\s*\(\s*([^)]+)\)`)

	// FOREIGN KEY(col) REFERENCES table(col)
	reForeignKey = regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(\s*["\x60]?(\w+)["\x60]?\s*\)\s*REFERENCES\s+["\x60]?(\w+)["\x60]?\s*\(\s*["\x60]?(\w+)["\x60]?\s*\)`)

	// CREATE [UNIQUE] INDEX name ON table(cols)
	reCreateIndex = regexp.MustCompile(`(?i)CREATE\s+(UNIQUE\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?["\x60]?(\w+)["\x60]?\s+ON\s+["\x60]?(\w+)["\x60]?\s*\(\s*([^)]+)\)`)

	// CREATE [MATERIALIZED] VIEW name
	reCreateView = regexp.MustCompile(`(?i)CREATE\s+(MATERIALIZED\s+)?VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?["\x60]?(\w+)["\x60]?`)

	// Inline PRIMARY KEY on column def
	reInlinePK = regexp.MustCompile(`(?i)\bPRIMARY\s+KEY\b`)

	// NOT NULL on column def
	reNotNull = regexp.MustCompile(`(?i)\bNOT\s+NULL\b`)
)

// ---------------------------------------------------------------------------
// ORM detection regexes (file-level)
// ---------------------------------------------------------------------------

var (
	reGORM       = regexp.MustCompile(`gorm\.Model`)
	reDjango     = regexp.MustCompile(`models\.Model`)
	reSQLAlchemy = regexp.MustCompile(`(?:Column\s*\(|declarative_base)`)
	rePrisma     = regexp.MustCompile(`(?m)^\s*model\s+\w+\s*\{`)
	reJPA        = regexp.MustCompile(`@(?:Entity|Table)`)
)

// SQLExtractor implements architect.Extractor for SQL schema analysis.
type SQLExtractor struct{}

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

// ---------------------------------------------------------------------------
// DDL parsing helpers
// ---------------------------------------------------------------------------

// parseTables extracts Table and ForeignKey definitions from SQL text.
func parseTables(sql, file string) ([]architect.Table, []architect.ForeignKey) {
	var tables []architect.Table
	var fks []architect.ForeignKey

	// Find each CREATE TABLE ... ( ... ) block.
	locs := reCreateTable.FindAllStringSubmatchIndex(sql, -1)
	for _, loc := range locs {
		tableName := sql[loc[2]:loc[3]]
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
		for _, m := range reForeignKey.FindAllStringSubmatch(body, -1) {
			fks = append(fks, architect.ForeignKey{
				FromTable:  tableName,
				FromColumn: m[1],
				ToTable:    m[2],
				ToColumn:   m[3],
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

			m := reColumnDef.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			colName := m[1]
			colType := strings.TrimSpace(m[2])
			// Clean trailing commas from type.
			colType = strings.TrimRight(colType, ", ")

			col := architect.Column{
				Name: colName,
				Type: colType,
			}
			if reInlinePK.MatchString(line) || pkCols[strings.ToLower(colName)] {
				col.PrimaryKey = true
			}
			if reNotNull.MatchString(line) || col.PrimaryKey {
				col.NotNull = true
			}
			columns = append(columns, col)
		}

		tables = append(tables, architect.Table{
			Name:    tableName,
			Columns: columns,
			File:    file,
		})
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
		cols := splitTrimmed(m[4])
		indexes = append(indexes, architect.Index{
			Name:    m[2],
			Table:   m[3],
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
		views = append(views, architect.View{
			Name:         m[2],
			Materialized: mat,
			File:         file,
		})
	}
	return views
}

// ---------------------------------------------------------------------------
// Migration detection
// ---------------------------------------------------------------------------

// detectMigrations scans well-known directories for migration files.
func detectMigrations(root string) *architect.MigrationInfo {
	for _, dir := range migrationRoots {
		full := filepath.Join(root, dir)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			continue
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			names = append(names, e.Name())
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)
		return &architect.MigrationInfo{
			Dir:    dir,
			Count:  len(names),
			Latest: names[len(names)-1],
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// ORM model detection (file-level)
// ---------------------------------------------------------------------------

// detectORM detects ORM frameworks used in a source file.
func detectORM(content, file string) []architect.ORMModel {
	var models []architect.ORMModel

	ext := strings.ToLower(filepath.Ext(file))

	switch ext {
	case ".go":
		if reGORM.MatchString(content) {
			models = append(models, architect.ORMModel{
				Framework: "gorm",
				File:      file,
			})
		}
	case ".py":
		if reDjango.MatchString(content) {
			models = append(models, architect.ORMModel{
				Framework: "django",
				File:      file,
			})
		}
		if reSQLAlchemy.MatchString(content) {
			models = append(models, architect.ORMModel{
				Framework: "sqlalchemy",
				File:      file,
			})
		}
	case ".prisma":
		if rePrisma.MatchString(content) {
			// Extract model names.
			for _, m := range rePrisma.FindAllString(content, -1) {
				parts := strings.Fields(m)
				name := ""
				if len(parts) >= 2 {
					name = parts[1]
				}
				models = append(models, architect.ORMModel{
					Framework: "prisma",
					File:      file,
					Model:     name,
				})
			}
		}
	case ".java":
		if reJPA.MatchString(content) {
			models = append(models, architect.ORMModel{
				Framework: "jpa",
				File:      file,
			})
		}
	}

	return models
}

// ---------------------------------------------------------------------------
// PII column detection
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Data domain clustering (BFS connected components via FK edges)
// ---------------------------------------------------------------------------

// clusterDomains groups tables into connected components using FK relationships.
func clusterDomains(tables []architect.Table, fks []architect.ForeignKey) []architect.DataDomain {
	// Build adjacency list.
	adj := make(map[string]map[string]bool)
	allTables := make(map[string]bool)
	for _, t := range tables {
		allTables[t.Name] = true
		if adj[t.Name] == nil {
			adj[t.Name] = make(map[string]bool)
		}
	}
	for _, fk := range fks {
		allTables[fk.FromTable] = true
		allTables[fk.ToTable] = true
		if adj[fk.FromTable] == nil {
			adj[fk.FromTable] = make(map[string]bool)
		}
		if adj[fk.ToTable] == nil {
			adj[fk.ToTable] = make(map[string]bool)
		}
		adj[fk.FromTable][fk.ToTable] = true
		adj[fk.ToTable][fk.FromTable] = true
	}

	visited := make(map[string]bool)
	var domains []architect.DataDomain

	// Sorted iteration for deterministic output.
	sortedTables := make([]string, 0, len(allTables))
	for t := range allTables {
		sortedTables = append(sortedTables, t)
	}
	sort.Strings(sortedTables)

	for _, start := range sortedTables {
		if visited[start] {
			continue
		}
		// BFS
		var component []string
		queue := []string{start}
		visited[start] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			component = append(component, cur)
			neighbors := adj[cur]
			// Sort neighbors for determinism.
			sorted := make([]string, 0, len(neighbors))
			for n := range neighbors {
				sorted = append(sorted, n)
			}
			sort.Strings(sorted)
			for _, n := range sorted {
				if !visited[n] {
					visited[n] = true
					queue = append(queue, n)
				}
			}
		}
		sort.Strings(component)
		domains = append(domains, architect.DataDomain{
			Name:   domainName(component),
			Tables: component,
		})
	}

	return domains
}

// domainName picks a representative name for a domain.
// It uses the longest common prefix of table names, falling back to the first table name.
func domainName(tables []string) string {
	if len(tables) == 0 {
		return "unknown"
	}
	if len(tables) == 1 {
		return tables[0]
	}

	// Try to find a common prefix (before the first underscore).
	prefixes := make(map[string]int)
	for _, t := range tables {
		parts := strings.SplitN(t, "_", 2)
		if len(parts) >= 1 && parts[0] != "" {
			prefixes[parts[0]]++
		}
	}
	// Pick the prefix that covers the most tables.
	bestPrefix := ""
	bestCount := 0
	for p, c := range prefixes {
		if c > bestCount || (c == bestCount && p < bestPrefix) {
			bestPrefix = p
			bestCount = c
		}
	}
	// Use prefix only if it covers a majority.
	if bestCount > 1 && bestCount >= len(tables)/2 {
		return bestPrefix
	}
	return tables[0]
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

// splitTrimmed splits on commas and trims whitespace and quotes from each part.
func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "\"`")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
