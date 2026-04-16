package spec

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSQL_NotNull(t *testing.T) {
	constraints := parseSQLFromTestdata(t, "001_create_users.sql")
	var notNull []SQLConstraint
	for _, c := range constraints {
		if c.Type == "NOT NULL" {
			notNull = append(notNull, c)
		}
	}
	assert.NotEmpty(t, notNull, "should find NOT NULL constraints")
}

func TestParseSQL_Unique(t *testing.T) {
	constraints := parseSQLFromTestdata(t, "001_create_users.sql")
	var unique []SQLConstraint
	for _, c := range constraints {
		if c.Type == "UNIQUE" {
			unique = append(unique, c)
		}
	}
	assert.NotEmpty(t, unique, "should find UNIQUE constraints")
}

func TestParseSQL_Check(t *testing.T) {
	constraints := parseSQLFromTestdata(t, "001_create_users.sql")
	var checks []SQLConstraint
	for _, c := range constraints {
		if c.Type == "CHECK" {
			checks = append(checks, c)
		}
	}
	assert.NotEmpty(t, checks, "should find CHECK constraints")
}

func TestParseSQL_ForeignKey(t *testing.T) {
	constraints := parseSQLFromTestdata(t, "001_create_users.sql")
	var fks []SQLConstraint
	for _, c := range constraints {
		if c.Type == "FOREIGN KEY" {
			fks = append(fks, c)
		}
	}
	assert.NotEmpty(t, fks, "should find FOREIGN KEY constraints")
	assert.Equal(t, "users(id)", fks[0].References)
}

func TestParseSQL_Default(t *testing.T) {
	constraints := parseSQLFromTestdata(t, "001_create_users.sql")
	var defaults []SQLConstraint
	for _, c := range constraints {
		if c.Type == "DEFAULT" {
			defaults = append(defaults, c)
		}
	}
	assert.NotEmpty(t, defaults, "should find DEFAULT constraints")
}

func TestParseSQL_Tables(t *testing.T) {
	constraints := parseSQLFromTestdata(t, "001_create_users.sql")
	tables := map[string]bool{}
	for _, c := range constraints {
		tables[c.Table] = true
	}
	assert.True(t, tables["users"], "should find users table")
	assert.True(t, tables["orders"], "should find orders table")
}

func TestParseSQL_NonexistentFile(t *testing.T) {
	constraints, err := ParseSQLFile(filepath.Join("testdata", "migrations", "nonexistent.sql"))
	assert.NoError(t, err)
	assert.Empty(t, constraints)
}

func parseSQLFromTestdata(t *testing.T, filename string) []SQLConstraint {
	t.Helper()
	path := filepath.Join("testdata", "migrations", filename)
	constraints, err := ParseSQLFile(path)
	require.NoError(t, err)
	return constraints
}
