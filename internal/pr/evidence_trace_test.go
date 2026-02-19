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
	body := `{"trace":{"pr_url":""},"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{}}`
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
