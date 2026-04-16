package spec

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

// ParseSQLFile reads a SQL file and extracts table constraints.
func ParseSQLFile(filePath string) ([]SQLConstraint, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("spec: resolve path: %w", err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, nil
	}
	return parseSQL(string(data), filepath.Base(filePath)), nil
}

func parseSQL(content, rel string) []SQLConstraint {
	var cs []SQLConstraint
	var table string
	lineNum := 0
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		lineNum++
		line := strings.TrimSpace(sc.Text())
		low := strings.ToLower(line)
		if low == "" || strings.HasPrefix(low, "--") {
			continue
		}
		if strings.HasPrefix(low, "create table") {
			table = tableName(low)
			continue
		}
		if table == "" {
			continue
		}
		cs = append(cs, colConstraints(low, table, rel, lineNum)...)
	}
	return cs
}

func tableName(low string) string {
	a := strings.TrimSpace(strings.TrimPrefix(low, "create table "))
	a = strings.TrimSpace(strings.TrimPrefix(a, "if not exists "))
	var name string
	for _, ch := range a {
		if ch == '(' || unicode.IsSpace(ch) {
			break
		}
		name += string(ch)
	}
	return name
}

func colConstraints(low, table, rel string, ln int) []SQLConstraint {
	col := colName(low)
	if col == "" || isTableKeyword(col) {
		return tblConstraints(low, table, rel, ln)
	}
	var cs []SQLConstraint
	add := func(t, v, ref string) {
		cs = append(cs, SQLConstraint{Table: table, Column: col, Type: t,
			Value: v, References: ref, SourceFile: rel, SourceLine: ln})
	}
	if strings.Contains(low, " not null") {
		add("NOT NULL", "", "")
	}
	if strings.Contains(low, " unique") && !strings.Contains(low, "unique index") {
		add("UNIQUE", "", "")
	}
	if idx := strings.Index(low, " check ("); idx >= 0 || strings.HasPrefix(low, "check (") {
		add("CHECK", parenContent(low, idx+7), "")
	}
	if idx := strings.Index(low, " default "); idx >= 0 {
		add("DEFAULT", afterKW(low, "default "), "")
	}
	if strings.Contains(low, " references ") {
		add("FOREIGN KEY", "", afterKW(low, "references "))
	}
	return cs
}

func colName(low string) string {
	line := strings.TrimLeft(low, " \t,")
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

var tableKeywords = []string{"constraint", "primary", "unique", "check", "foreign", "index", "create", ")"}

func isTableKeyword(s string) bool {
	return slices.Contains(tableKeywords, s)
}

func tblConstraints(low, table, rel string, ln int) []SQLConstraint {
	var cs []SQLConstraint
	if strings.Contains(low, "check (") {
		cs = append(cs, SQLConstraint{Table: table, Type: "CHECK",
			Value: parenContent(low, strings.Index(low, "check (")+6),
			SourceFile: rel, SourceLine: ln})
	}
	if strings.Contains(low, "foreign key") {
		fkCol := parenContent(low, strings.Index(low, "foreign key")+11)
		if strings.Contains(low, " references ") {
			cs = append(cs, SQLConstraint{Table: table, Column: fkCol, Type: "FOREIGN KEY",
				References: afterKW(low, "references "), SourceFile: rel, SourceLine: ln})
		}
	}
	return cs
}

func parenContent(s string, start int) string {
	if start < 0 || start >= len(s) {
		return ""
	}
	depth := 0
	var buf strings.Builder
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(buf.String())
			}
		default:
			if depth > 0 {
				buf.WriteByte(s[i])
			}
		}
	}
	return strings.TrimSpace(buf.String())
}

func afterKW(s, kw string) string {
	i := strings.Index(s, kw)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(strings.TrimRight(s[i+len(kw):], ",;"))
}
