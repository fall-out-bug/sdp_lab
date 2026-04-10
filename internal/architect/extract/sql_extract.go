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

// migrationRoots are top-level directories that may contain migration sub-dirs.
// Ordered from most specific to least specific; first match wins.
var migrationRoots = []string{
	"migrations",
	"db/migrate",
	"db/migration",
	"alembic/versions",
	"database/migrations",
	"flyway",
	"prisma/migrations",
	"supabase/migrations",
	"drizzle",
}

// piiExactPatterns lists column name substrings that indicate PII.
// Exact match (column name equals the pattern) yields 0.95 confidence.
// Partial match (column name contains the pattern) yields 0.75 confidence.
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
	"passport",
	"mobile",
	"telephone",
	"zip_code",
	"postal_code",
	"date_of_birth",
	"maiden_name",
	"national_id",
	"tax_id",
	"driver_license",
	"social_security",
}

// piiTypeMap maps each pattern to a human-readable PII category.
var piiTypeMap = map[string]string{
	"email":           "email_address",
	"phone":           "phone_number",
	"ssn":             "social_security_number",
	"birth_date":      "date_of_birth",
	"address":         "physical_address",
	"first_name":      "personal_name",
	"last_name":       "personal_name",
	"ip_address":      "network_identifier",
	"credit_card":     "financial_identifier",
	"passport":        "government_identifier",
	"mobile":          "phone_number",
	"telephone":       "phone_number",
	"zip_code":        "location_identifier",
	"postal_code":     "location_identifier",
	"date_of_birth":   "date_of_birth",
	"maiden_name":     "personal_name",
	"national_id":     "government_identifier",
	"tax_id":          "government_identifier",
	"driver_license":  "government_identifier",
	"social_security": "social_security_number",
}

// ---------------------------------------------------------------------------
// Compiled regexes for DDL parsing
// ---------------------------------------------------------------------------

var (
	// CREATE TABLE [schema.]name ( ... )
	reCreateTable = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` +
		`(?:["\x60]?(\w+)["\x60]?\.)?["\x60]?(\w+)["\x60]?\s*\(`)

	// Column line inside CREATE TABLE: name TYPE [constraints]
	reColumnDef = regexp.MustCompile(`(?i)^\s*["\x60]?(\w+)["\x60]?\s+(\w[\w() ,]*?)` +
		`(\s+PRIMARY\s+KEY|\s+NOT\s+NULL|\s+UNIQUE|\s+DEFAULT\s+\S+|\s+REFERENCES\s+\S+|\s+CHECK\s*\([^)]*\))*\s*,?\s*$`)

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

// ---------------------------------------------------------------------------
// ORM detection regexes (file-level)
// ---------------------------------------------------------------------------

var (
	reGORM            = regexp.MustCompile(`gorm\.Model`)
	reGORMTag         = regexp.MustCompile(`gorm:"`)
	reGORMStruct      = regexp.MustCompile(`(?m)type\s+(\w+)\s+struct\b`)
	reDjangoORM       = regexp.MustCompile(`models\.Model`)
	reDjangoORMModel  = regexp.MustCompile(`(?m)class\s+(\w+)\s*\(\s*models\.Model\s*\)`)
	reSQLAlchemy      = regexp.MustCompile(`(?:Column\s*\(|declarative_base)`)
	reSAModelClass    = regexp.MustCompile(`(?m)class\s+(\w+)\s*\(\s*Base\s*\)`)
	rePrismaModel     = regexp.MustCompile(`(?m)^\s*model\s+(\w+)\s*\{`)
	reJPA             = regexp.MustCompile(`@(?:Entity|Table)`)
	reJPAClass        = regexp.MustCompile(`(?m)public\s+class\s+(\w+)`)
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
			} else {
				// If no NOT NULL and no PRIMARY KEY, the column is nullable.
				col.Nullable = true
			}
			columns = append(columns, col)
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
		if reGORM.MatchString(content) || reGORMTag.MatchString(content) {
			// Try to extract struct names that use GORM.
			structNames := reGORMStruct.FindAllStringSubmatch(content, -1)
			if len(structNames) > 0 {
				for _, m := range structNames {
					models = append(models, architect.ORMModel{
						Framework: "gorm",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "gorm",
					File:      file,
				})
			}
		}
	case ".py":
		// Django models: class Foo(models.Model)
		if reDjangoORM.MatchString(content) {
			djangoMatches := reDjangoORMModel.FindAllStringSubmatch(content, -1)
			if len(djangoMatches) > 0 {
				for _, m := range djangoMatches {
					models = append(models, architect.ORMModel{
						Framework: "django",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "django",
					File:      file,
				})
			}
		}
		// SQLAlchemy models: class Foo(Base) or Column(...) usage
		if reSQLAlchemy.MatchString(content) {
			saMatches := reSAModelClass.FindAllStringSubmatch(content, -1)
			if len(saMatches) > 0 {
				for _, m := range saMatches {
					models = append(models, architect.ORMModel{
						Framework: "sqlalchemy",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "sqlalchemy",
					File:      file,
				})
			}
		}
	case ".prisma":
		// Prisma model names extracted directly from regex.
		for _, m := range rePrismaModel.FindAllStringSubmatch(content, -1) {
			models = append(models, architect.ORMModel{
				Framework: "prisma",
				File:      file,
				Model:     m[1],
			})
		}
	case ".java":
		if reJPA.MatchString(content) {
			// Try to extract the class name annotated with @Entity/@Table.
			classMatches := reJPAClass.FindAllStringSubmatch(content, -1)
			if len(classMatches) > 0 {
				for _, m := range classMatches {
					models = append(models, architect.ORMModel{
						Framework: "jpa",
						File:      file,
						Model:     m[1],
					})
				}
			} else {
				models = append(models, architect.ORMModel{
					Framework: "jpa",
					File:      file,
				})
			}
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
