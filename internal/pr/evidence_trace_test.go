package pr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePRURLToEvidence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "evidence.json")
	body := `{"trace":{"pr_url":""},"provenance":{"run_id":"run-1"},"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := WritePRURLToEvidence(path, "https://example/pull/1"); err != nil {
		t.Fatalf("write pr url: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(out), "https://example/pull/1") {
		t.Fatalf("expected updated pr_url, got: %s", string(out))
	}
}

func TestWritePublishTraceToEvidenceWritesContextLinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "evidence.json")
	body := `{"trace":{"pr_url":""},"provenance":{"run_id":"run-2"},"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := WritePublishTraceToEvidence(path, "https://example/pull/2", ".sdp/runs/x.json", ".sdp/evidence/x.json"); err != nil {
		t.Fatalf("write trace: %v", err)
	}

	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	bodyOut := string(out)
	if !strings.Contains(bodyOut, `"run_context_link": ".sdp/runs/x.json"`) {
		t.Fatalf("expected run context link in trace, got: %s", bodyOut)
	}
	if !strings.Contains(bodyOut, `"evidence_context_link": ".sdp/evidence/x.json"`) {
		t.Fatalf("expected evidence context link in trace, got: %s", bodyOut)
	}
}

func TestReadRunIDFromEvidence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "evidence.json")
	body := `{"trace":{},"provenance":{"run_id":"run-3"}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	runID, err := ReadRunIDFromEvidence(path)
	if err != nil {
		t.Fatalf("read run id: %v", err)
	}
	if runID != "run-3" {
		t.Fatalf("unexpected run id: %s", runID)
	}
}
