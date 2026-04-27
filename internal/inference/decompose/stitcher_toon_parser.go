package decompose

import (
	"fmt"
	"strings"
)

// marshalTOON converts tabular data to TOON wire format.
// Multi-row: "# col1 | col2\nv1 | v2\n..."
// Single-row: "col1=v1, col2=v2"
// Empty table: header only.
func marshalTOON(columns []TOONColumn, rows []map[string]any) (string, error) {
	names := make([]string, len(columns))
	for i, c := range columns {
		names[i] = c.Name
	}

	if len(rows) == 1 {
		// Single-record inline form.
		return marshalTOONInline(names, rows[0])
	}

	var b strings.Builder
	// Header.
	b.WriteString("# ")
	b.WriteString(strings.Join(names, " | "))
	if len(rows) == 0 {
		return b.String(), nil
	}
	// Rows.
	for _, row := range rows {
		b.WriteByte('\n')
		cells := make([]string, len(columns))
		for i, col := range columns {
			cells[i] = toonCell(row[col.Name])
		}
		b.WriteString(strings.Join(cells, " | "))
	}
	return b.String(), nil
}

func marshalTOONInline(names []string, row map[string]any) (string, error) {
	pairs := make([]string, len(names))
	for i, name := range names {
		pairs[i] = name + "=" + toonCell(row[name])
	}
	return strings.Join(pairs, ", "), nil
}

// toonCell converts a cell value to its TOON string form.
// nil/null → empty string (two adjacent separators: "v1 |  | v3").
func toonCell(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// ParseTOON parses a TOON table string back to []map[string]any.
// Supports both multi-row (# header\nrow...) and single-record inline (k=v, k=v).
// Values are always string-typed; callers must re-cast if needed.
func ParseTOON(s string) ([]map[string]any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	// Inline form: no leading '#'.
	if !strings.HasPrefix(s, "#") {
		return parseTOONInline(s)
	}

	lines := strings.Split(s, "\n")
	header := strings.TrimPrefix(lines[0], "# ")
	cols := splitPipe(header)
	if len(cols) == 0 {
		return nil, fmt.Errorf("toon: empty header")
	}

	if len(lines) == 1 {
		return []map[string]any{}, nil // header-only, 0 rows
	}

	rows := make([]map[string]any, 0, len(lines)-1)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cells := splitPipe(line)
		if len(cells) != len(cols) {
			return nil, fmt.Errorf("toon: row has %d cells, header has %d columns", len(cells), len(cols))
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			if cells[i] == "" {
				row[col] = nil
			} else {
				row[col] = cells[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseTOONInline(s string) ([]map[string]any, error) {
	parts := strings.Split(s, ", ")
	row := make(map[string]any, len(parts))
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("toon inline: invalid pair %q", part)
		}
		row[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return []map[string]any{row}, nil
}

func splitPipe(s string) []string {
	parts := strings.Split(s, "|")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}
