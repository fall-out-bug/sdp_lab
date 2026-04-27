package decompose

import (
	"fmt"
	"strconv"
)

// TOONColumn describes one column in a TOON table.
type TOONColumn struct {
	// Name is the column header token.
	Name string
	// Type is one of "string", "int", "float", "bool".
	Type string
}

// TOONStitcher serializes tabular data ([]map[string]any) as a TOON table —
// a flat pipe-delimited format that achieves ≥40% token saving vs JSON for
// multi-row data. Nested objects are not supported in v1.
//
// Format:
//   # col1 | col2 | col3
//   val1 | val2 | val3
//
// Single-record fallback:
//   col1=val1, col2=val2
type TOONStitcher struct {
	name    string
	columns []TOONColumn
}

// NewTOONStitcher creates a TOONStitcher with the given column schema.
func NewTOONStitcher(name string, columns []TOONColumn) *TOONStitcher {
	return &TOONStitcher{name: name, columns: columns}
}

func (t *TOONStitcher) Name() string { return t.name }

// Validate checks that out is a []map[string]any with all required columns
// having the correct types. Returns an error for nested objects.
func (t *TOONStitcher) Validate(out any) error {
	rows, err := asTOONRows(out)
	if err != nil {
		return fmt.Errorf("toon stitcher %q: %w", t.name, err)
	}
	for i, row := range rows {
		for _, col := range t.columns {
			v, ok := row[col.Name]
			if !ok {
				return fmt.Errorf("toon stitcher %q: row %d missing column %q", t.name, i, col.Name)
			}
			if v == nil {
				continue // null cell is allowed
			}
			if err := checkTOONType(col.Name, col.Type, v); err != nil {
				return fmt.Errorf("toon stitcher %q: row %d: %w", t.name, i, err)
			}
		}
	}
	return nil
}

// Marshal converts out ([]map[string]any) to a TOON table string.
// Returns an error for nested objects or type mismatches.
func (t *TOONStitcher) Marshal(out any) (string, error) {
	rows, err := asTOONRows(out)
	if err != nil {
		return "", fmt.Errorf("toon stitcher %q: %w", t.name, err)
	}
	if err := t.Validate(out); err != nil {
		return "", err
	}
	return marshalTOON(t.columns, rows)
}

func asTOONRows(out any) ([]map[string]any, error) {
	switch v := out.(type) {
	case []map[string]any:
		return v, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("expected []map[string]any, got %T", out)
	}
}

// checkTOONType validates that v matches colType.
// String values are accepted for numeric/bool columns only if they are parseable
// as the target type — this supports the Marshal→ParseTOON→Validate round-trip
// since ParseTOON returns all cells as strings.
func checkTOONType(colName, colType string, v any) error {
	switch colType {
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("column %q: expected string, got %T", colName, v)
		}
	case "int":
		switch fv := v.(type) {
		case int, int8, int16, int32, int64:
		case float64:
			// json.Unmarshal produces float64; verify it is integral.
			if fv != float64(int64(fv)) {
				return fmt.Errorf("column %q: expected integral int, got fractional float64 %v", colName, fv)
			}
		case string:
			if _, err := strconv.ParseInt(fv, 10, 64); err != nil {
				return fmt.Errorf("column %q: string %q is not a valid int", colName, fv)
			}
		default:
			return fmt.Errorf("column %q: expected int, got %T", colName, v)
		}
	case "float":
		switch fv := v.(type) {
		case float32, float64:
		case string:
			if _, err := strconv.ParseFloat(fv, 64); err != nil {
				return fmt.Errorf("column %q: string %q is not a valid float", colName, fv)
			}
		default:
			return fmt.Errorf("column %q: expected float, got %T", colName, v)
		}
	case "bool":
		switch fv := v.(type) {
		case bool:
		case string:
			if _, err := strconv.ParseBool(fv); err != nil {
				return fmt.Errorf("column %q: string %q is not a valid bool", colName, fv)
			}
		default:
			return fmt.Errorf("column %q: expected bool, got %T", colName, v)
		}
	default:
		return fmt.Errorf("column %q: unknown type %q", colName, colType)
	}
	return nil
}
