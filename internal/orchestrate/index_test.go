package orchestrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateIndexTable(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a couple of workstream files
	for _, name := range []string{"00-053-01.md", "00-053-02.md"} {
		content := `---
ws_id: ` + name[:len(name)-3] + `
feature_id: F053
status: done
---
# ` + name[:len(name)-3] + `: Test Title ` + name[7:9] + `
`
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := GenerateIndexTable(root, "F053", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0].WS != "00-053-01" || rows[0].Title != "Test Title 01" || rows[0].Status != "Done" {
		t.Errorf("row0: got WS=%q Title=%q Status=%q", rows[0].WS, rows[0].Title, rows[0].Status)
	}
	if rows[1].WS != "00-053-02" || rows[1].Title != "Test Title 02" || rows[1].Status != "Done" {
		t.Errorf("row1: got WS=%q Title=%q Status=%q", rows[1].WS, rows[1].Title, rows[1].Status)
	}

	out := FormatIndexTable(rows)
	if !strings.Contains(out, "| WS | Feature | Title | Status |") {
		t.Errorf("missing header: %s", out)
	}
	if !strings.Contains(out, "| 00-053-01 | F053 | Test Title 01 | Done |") {
		t.Errorf("missing row: %s", out)
	}
}

func TestGenerateIndexTable_CheckpointOverride(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
ws_id: 00-053-99
feature_id: F053
status: pending
---
# 00-053-99: Pending WS
`
	if err := os.WriteFile(filepath.Join(dir, "00-053-99.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cp := &Checkpoint{Workstreams: []WSStatus{{ID: "00-053-99", Status: "done"}}}
	rows, err := GenerateIndexTable(root, "F053", cp)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].Status != "Done" {
		t.Errorf("checkpoint override: want Done, got %s", rows[0].Status)
	}
}
