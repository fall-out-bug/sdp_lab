package decompose_test

import (
	"encoding/json"
	"testing"

	"sdp_dev/internal/inference/decompose"
)

var benchRows = func() []map[string]any {
	rows := make([]map[string]any, 20)
	for i := range rows {
		rows[i] = map[string]any{
			"file":   "internal/something/module/longpath_file.go",
			"status": "pass",
			"score":  float64(0.9),
		}
	}
	return rows
}()

var benchCols = []decompose.TOONColumn{
	{Name: "file", Type: "string"},
	{Name: "status", Type: "string"},
	{Name: "score", Type: "float"},
}

func BenchmarkStitcher_Marshal_JSON(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, _ := json.MarshalIndent(benchRows, "", "  ")
		_ = data
	}
}

func BenchmarkStitcher_Marshal_TOON(b *testing.B) {
	s := decompose.NewTOONStitcher("bench", benchCols)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		out, _ := s.Marshal(benchRows)
		_ = out
	}
}
