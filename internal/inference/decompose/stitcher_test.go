package decompose_test

import (
	"encoding/json"
	"strings"
	"testing"

	"sdp_dev/internal/inference/decompose"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- EnumStitcher ----

func TestEnumStitcher_ValidateOK(t *testing.T) {
	s := decompose.NewEnumStitcher("verdict", []string{"pass", "warn", "fail"})
	assert.NoError(t, s.Validate("pass"))
	assert.NoError(t, s.Validate("warn"))
	assert.NoError(t, s.Validate("fail"))
}

func TestEnumStitcher_ValidateNotInSet(t *testing.T) {
	s := decompose.NewEnumStitcher("verdict", []string{"pass", "fail"})
	err := s.Validate("unknown")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in allowed set")
}

func TestEnumStitcher_ValidateWrongType(t *testing.T) {
	s := decompose.NewEnumStitcher("verdict", []string{"pass"})
	err := s.Validate(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected string")
}

func TestEnumStitcher_MarshalOK(t *testing.T) {
	s := decompose.NewEnumStitcher("verdict", []string{"pass", "fail"})
	out, err := s.Marshal("pass")
	require.NoError(t, err)
	assert.Equal(t, "pass", out)
}

func TestEnumStitcher_MarshalInvalid(t *testing.T) {
	s := decompose.NewEnumStitcher("verdict", []string{"pass"})
	_, err := s.Marshal("other")
	require.Error(t, err)
}

func TestEnumStitcher_CaseSensitive(t *testing.T) {
	s := decompose.NewEnumStitcher("v", []string{"Pass"})
	assert.Error(t, s.Validate("pass"))
	assert.NoError(t, s.Validate("Pass"))
}

// ---- JSONStitcher ----

var simpleSchema = []byte(`{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["name", "score"],
  "properties": {
    "name": {"type": "string"},
    "score": {"type": "number", "minimum": 0, "maximum": 1}
  }
}`)

func TestJSONStitcher_ValidateOK(t *testing.T) {
	s, err := decompose.NewJSONStitcherFromBytes("extract", simpleSchema)
	require.NoError(t, err)
	assert.NoError(t, s.Validate(map[string]any{"name": "file.go", "score": 0.9}))
}

func TestJSONStitcher_ValidateMissingField(t *testing.T) {
	s, err := decompose.NewJSONStitcherFromBytes("extract", simpleSchema)
	require.NoError(t, err)
	err = s.Validate(map[string]any{"name": "file.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema violation")
}

func TestJSONStitcher_ValidateWrongType(t *testing.T) {
	s, err := decompose.NewJSONStitcherFromBytes("extract", simpleSchema)
	require.NoError(t, err)
	err = s.Validate(map[string]any{"name": 123, "score": 0.5})
	require.Error(t, err)
}

func TestJSONStitcher_MarshalRoundTrip(t *testing.T) {
	s, err := decompose.NewJSONStitcherFromBytes("extract", simpleSchema)
	require.NoError(t, err)
	input := map[string]any{"name": "main.go", "score": 0.8}
	marshalled, err := s.Marshal(input)
	require.NoError(t, err)

	var parsed any
	require.NoError(t, json.Unmarshal([]byte(marshalled), &parsed))
	assert.NoError(t, s.Validate(parsed))
}

func TestJSONStitcher_InvalidSchema(t *testing.T) {
	_, err := decompose.NewJSONStitcherFromBytes("bad", []byte(`{bad json`))
	require.Error(t, err)
}

// ---- TOONStitcher ----

var toonCols = []decompose.TOONColumn{
	{Name: "file", Type: "string"},
	{Name: "status", Type: "string"},
	{Name: "score", Type: "float"},
}

func TestTOONStitcher_ValidateOK(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	rows := []map[string]any{
		{"file": "main.go", "status": "pass", "score": 0.9},
		{"file": "util.go", "status": "warn", "score": 0.5},
	}
	assert.NoError(t, s.Validate(rows))
}

func TestTOONStitcher_ValidateMissingColumn(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	rows := []map[string]any{{"file": "main.go", "status": "pass"}} // missing score
	require.Error(t, s.Validate(rows))
}

func TestTOONStitcher_ValidateWrongType(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	rows := []map[string]any{{"file": "main.go", "status": "pass", "score": "bad"}}
	err := s.Validate(rows)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "score")
}

func TestTOONStitcher_ValidateNullCell(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	rows := []map[string]any{{"file": nil, "status": "pass", "score": 0.9}}
	assert.NoError(t, s.Validate(rows)) // null is allowed
}

func TestTOONStitcher_ValidateNestedObject(t *testing.T) {
	// Nested objects cause wrong type error (map[string]any is not a basic type).
	s := decompose.NewTOONStitcher("agg", []decompose.TOONColumn{{Name: "meta", Type: "string"}})
	rows := []map[string]any{{"meta": map[string]any{"k": "v"}}}
	err := s.Validate(rows)
	require.Error(t, err) // expected string, got map
	assert.Contains(t, err.Error(), "meta")
}

func TestTOONStitcher_MarshalMultiRow(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	rows := []map[string]any{
		{"file": "main.go", "status": "pass", "score": float64(0.9)},
		{"file": "util.go", "status": "warn", "score": float64(0.5)},
	}
	out, err := s.Marshal(rows)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "# file | status | score"), "expected TOON header, got: %s", out)
	assert.Contains(t, out, "main.go | pass | 0.9")
}

func TestTOONStitcher_MarshalSingleRow(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	rows := []map[string]any{
		{"file": "main.go", "status": "pass", "score": float64(1.0)},
	}
	out, err := s.Marshal(rows)
	require.NoError(t, err)
	// Single-record inline: "file=main.go, status=pass, score=1"
	assert.Contains(t, out, "file=")
	assert.NotContains(t, out, "#") // no header in inline form
}

func TestTOONStitcher_MarshalEmptyTable(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	out, err := s.Marshal([]map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "# file | status | score", out)
}

func TestTOONStitcher_MarshalWrongType(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", toonCols)
	_, err := s.Marshal("not a slice")
	require.Error(t, err)
}

// ---- Name() via interface ----

func TestStitcher_Names(t *testing.T) {
	var s decompose.Stitcher

	s = decompose.NewEnumStitcher("my-enum", nil)
	assert.Equal(t, "my-enum", s.Name())

	js, err := decompose.NewJSONStitcherFromBytes("my-json", simpleSchema)
	require.NoError(t, err)
	s = js
	assert.Equal(t, "my-json", s.Name())

	s = decompose.NewTOONStitcher("my-toon", nil)
	assert.Equal(t, "my-toon", s.Name())
}

// ---- TOON type coverage (bool, unknown) ----

func TestTOONStitcher_BoolColumn(t *testing.T) {
	s := decompose.NewTOONStitcher("flags", []decompose.TOONColumn{{Name: "ok", Type: "bool"}})
	assert.NoError(t, s.Validate([]map[string]any{{"ok": true}}))
	assert.Error(t, s.Validate([]map[string]any{{"ok": "true"}}))
}

func TestTOONStitcher_FloatColumn(t *testing.T) {
	s := decompose.NewTOONStitcher("scores", []decompose.TOONColumn{{Name: "v", Type: "float"}})
	assert.NoError(t, s.Validate([]map[string]any{{"v": float64(0.5)}}))
	assert.Error(t, s.Validate([]map[string]any{{"v": "bad"}}))
}

func TestTOONStitcher_UnknownColumnType(t *testing.T) {
	s := decompose.NewTOONStitcher("bad", []decompose.TOONColumn{{Name: "x", Type: "bytes"}})
	err := s.Validate([]map[string]any{{"x": "value"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

// ---- Cross-stitcher property: Marshal → parseback → Validate ----

func TestCrossStitcher_RoundTrip_Enum(t *testing.T) {
	s := decompose.NewEnumStitcher("v", []string{"ok", "fail"})
	marshalled, err := s.Marshal("ok")
	require.NoError(t, err)
	assert.NoError(t, s.Validate(marshalled))
}

func TestCrossStitcher_RoundTrip_JSON(t *testing.T) {
	s, err := decompose.NewJSONStitcherFromBytes("x", simpleSchema)
	require.NoError(t, err)
	input := map[string]any{"name": "a.go", "score": 0.5}
	marshalled, err := s.Marshal(input)
	require.NoError(t, err)
	var parsed any
	require.NoError(t, json.Unmarshal([]byte(marshalled), &parsed))
	assert.NoError(t, s.Validate(parsed))
}

func TestCrossStitcher_RoundTrip_TOON(t *testing.T) {
	s := decompose.NewTOONStitcher("agg", []decompose.TOONColumn{
		{Name: "file", Type: "string"},
		{Name: "status", Type: "string"},
	})
	input := []map[string]any{
		{"file": "a.go", "status": "pass"},
		{"file": "b.go", "status": "fail"},
	}
	marshalled, err := s.Marshal(input)
	require.NoError(t, err)

	parsed, err := decompose.ParseTOON(marshalled)
	require.NoError(t, err)
	// Validate parsed (values are strings after ParseTOON).
	assert.NoError(t, s.Validate(parsed))
}

// ---- ParseTOON edge cases ----

func TestParseTOON_MultiRow(t *testing.T) {
	toon := "# name | score\nmain.go | 0.9\nutil.go | 0.5"
	rows, err := decompose.ParseTOON(toon)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "main.go", rows[0]["name"])
	assert.Equal(t, "0.9", rows[0]["score"])
}

func TestParseTOON_Inline(t *testing.T) {
	rows, err := decompose.ParseTOON("file=main.go, status=pass")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "main.go", rows[0]["file"])
}

func TestParseTOON_EmptyHeaderOnly(t *testing.T) {
	rows, err := decompose.ParseTOON("# col1 | col2")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParseTOON_NullCell(t *testing.T) {
	toon := "# a | b\nval |  "
	rows, err := decompose.ParseTOON(toon)
	require.NoError(t, err)
	assert.Nil(t, rows[0]["b"])
}

func TestParseTOON_Empty(t *testing.T) {
	rows, err := decompose.ParseTOON("")
	require.NoError(t, err)
	assert.Nil(t, rows)
}

// ---- Token saving ----

func TestTOONStitcher_TokenSaving(t *testing.T) {
	// Build a 10-row fixture and compare marshalled length (proxy for token count).
	cols := []decompose.TOONColumn{
		{Name: "file", Type: "string"},
		{Name: "status", Type: "string"},
		{Name: "score", Type: "float"},
	}
	rows := make([]map[string]any, 10)
	for i := range rows {
		rows[i] = map[string]any{
			"file":   "internal/something/longpath_file.go",
			"status": "pass",
			"score":  float64(0.9),
		}
	}

	toonStitcher := decompose.NewTOONStitcher("bench", cols)
	toonOut, err := toonStitcher.Marshal(rows)
	require.NoError(t, err)

	jsonOut, err := json.MarshalIndent(rows, "", "  ")
	require.NoError(t, err)

	ratio := float64(len(toonOut)) / float64(len(jsonOut))
	assert.Less(t, ratio, 0.60, "TOON should use <60%% of JSON chars; got ratio %.2f (toon=%d json=%d)",
		ratio, len(toonOut), len(jsonOut))
}
