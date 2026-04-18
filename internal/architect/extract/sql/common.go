// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"path/filepath"
	"strings"
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

// isTestPath returns true if the path looks like a test fixture or test directory.
// It uses a two-pass approach: first check explicit test directories, then check
// filename patterns. Paths under /src/test/resources/ are test fixtures and skipped,
// but paths under /src/test/ that contain actual SQL DDL are kept for analysis.
func isTestPath(rel string) bool {
	lower := strings.ToLower(rel)

	// Skip generated output and binary-like artifacts in any path.
	skipExts := []string{".out", ".explain", ".crc", ".bin", ".zip", ".jar"}
	for _, ext := range skipExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}

	// Hard skip: known fixture/resource directories.
	fixturePatterns := []string{
		"/fixtures/", "/fixture/", "fixtures/", "fixture/",
		"/testdata/", "/test_data/", "testdata/", "test_data/",
		"/mock/", "/mocks/", "mock/", "mocks/",
		"/__tests__/", "__tests__/",
		"/src/test/resources/",
	}
	for _, p := range fixturePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	// For /test/ and /tests/: only skip if it's a top-level test directory
	// (e.g. "project/test/") but not if it's "sql/core/src/test/resources/".
	// We already handled /src/test/resources/ above.
	// Keep files under /src/test/ that may contain DDL (common in Java/Scala projects).

	// Filename patterns.
	base := strings.ToLower(filepath.Base(rel))
	if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.sql") {
		return true
	}

	return false
}

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
