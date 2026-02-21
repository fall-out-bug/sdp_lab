package selfimprove

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIngestRuns_MissingDir(t *testing.T) {
	runs, err := IngestRuns(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Errorf("expected 0 runs, got %d", len(runs))
	}
}

func TestIngestRuns_ValidFile(t *testing.T) {
	dir := t.TempDir()
	runsDir := filepath.Join(dir, ".sdp", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(runsDir, "run1.json")
	if err := os.WriteFile(path, []byte(`{"run_id":"r1","issue_id":"i1","events":[],"last_phase":"done","last_state":"ok"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	runs, err := IngestRuns(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].RunID != "r1" || runs[0].IssueID != "i1" {
		t.Errorf("expected 1 run r1/i1, got %+v", runs)
	}
}

func TestIngestIntakeJSONL_MissingFile(t *testing.T) {
	recs, err := IngestIntakeJSONL(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Errorf("expected 0 records, got %d", len(recs))
	}
}

func TestIngestIntakeJSONL_ValidLine(t *testing.T) {
	f := filepath.Join(t.TempDir(), "intake.jsonl")
	line := `{"protocol":{"issue_id":"i1","phase":"run","status":"ok"},"system":{"component":"agent"},"resilience":{"retry_count":1,"fallback_used":false,"escalated":false}}`
	if err := os.WriteFile(f, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	recs, err := IngestIntakeJSONL(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].IssueID != "i1" || recs[0].Phase != "run" || recs[0].RetryCount != 1 {
		t.Errorf("expected 1 record i1/run/ok retry=1, got %+v", recs)
	}
}
