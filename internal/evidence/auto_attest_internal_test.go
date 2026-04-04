package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectDeclaredScopePrefixes_ExtractsScopeFiles(t *testing.T) {
	repo := t.TempDir()
	backlogDir := filepath.Join(repo, "docs", "workstreams", "backlog")
	if err := os.MkdirAll(backlogDir, 0o755); err != nil {
		t.Fatalf("mkdir backlog: %v", err)
	}

	ws1 := "# WS\n\n## Scope Files\n- `internal/evidence/auto_attest.go`\n- docs/workstreams/INDEX.md\n\n## Acceptance Criteria\n- [ ] x\n"
	if err := os.WriteFile(filepath.Join(backlogDir, "00-001-01.md"), []byte(ws1), 0o644); err != nil {
		t.Fatalf("write workstream 1: %v", err)
	}

	ws2 := "# WS\n\n## Scope Files\n- docs/workstreams/INDEX.md\n- internal/evidence/discrepancy.go\n"
	if err := os.WriteFile(filepath.Join(backlogDir, "00-001-02.md"), []byte(ws2), 0o644); err != nil {
		t.Fatalf("write workstream 2: %v", err)
	}

	prefixes := collectDeclaredScopePrefixes(repo, []string{"00-001-01"})
	if len(prefixes) != 2 {
		t.Fatalf("prefix count = %d, want 2 (%v)", len(prefixes), prefixes)
	}

	joined := strings.Join(prefixes, ",")
	if !strings.Contains(joined, "internal/evidence/auto_attest.go") {
		t.Fatalf("missing auto_attest path in %v", prefixes)
	}
	if !strings.Contains(joined, "docs/workstreams/INDEX.md") {
		t.Fatalf("missing index path in %v", prefixes)
	}
	if strings.Contains(joined, "internal/evidence/discrepancy.go") {
		t.Fatalf("unexpected unrelated workstream path in %v", prefixes)
	}
}

func TestCollectDeclaredScopePrefixes_NoLinkedWorkstreams(t *testing.T) {
	repo := t.TempDir()
	if got := collectDeclaredScopePrefixes(repo, nil); got != nil {
		t.Fatalf("collectDeclaredScopePrefixes() = %v, want nil", got)
	}
}

func TestCollectTestResults_CommandFailure(t *testing.T) {
	repo := t.TempDir()
	goMod := "module example.com/failmod\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	failingTest := "package failmod\n\nimport \"testing\"\n\nfunc TestFail(t *testing.T) { t.Fatal(\"boom\") }\n"
	if err := os.WriteFile(filepath.Join(repo, "fail_test.go"), []byte(failingTest), 0o644); err != nil {
		t.Fatalf("write fail_test.go: %v", err)
	}

	results, coverage := collectTestResults(repo)
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0].Name != "go-test" {
		t.Fatalf("result name = %q, want go-test", results[0].Name)
	}
	if !strings.HasPrefix(results[0].Status, "fail") {
		t.Fatalf("status = %q, want fail prefix", results[0].Status)
	}
	if coverage != -1 {
		t.Fatalf("coverage = %v, want -1 on command error", coverage)
	}
}

func TestExtractBeadsIDs_SupportsCurrentAndLegacyPrefixes(t *testing.T) {
	text := strings.Join([]string{
		"first sdplab-n8w",
		"legacy sdp_dev-abcd",
		"short sdplab-7",
		"repeat sdplab-n8w",
	}, "\n")

	got := extractBeadsIDs(text)
	want := []string{"sdplab-n8w", "sdp_dev-abcd", "sdplab-7"}

	if len(got) != len(want) {
		t.Fatalf("len(extractBeadsIDs()) = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("extractBeadsIDs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
