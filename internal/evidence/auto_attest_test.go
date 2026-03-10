package evidence

import (
	"os"
	"path/filepath"
	"testing"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
)

func TestCheckScopeCompliance_NoDeclaredScope(t *testing.T) {
	dir := t.TempDir()
	boundary, ok := checkScopeCompliance(dir, []string{"internal/foo.go"})
	if !ok {
		t.Error("no declared scope should be compliant")
	}
	if boundary.Compliance.Reason == "" {
		t.Error("reason should be set")
	}
}

func TestParseCoverageLine(t *testing.T) {
	tests := []struct {
		line string
		want float64
	}{
		{"ok  \tsdp_dev/internal/evidence\t2.481s\tcoverage: 85.3% of statements", 85.3},
		{"coverage: 42.1%", 42.1},
		{"no coverage here", -1},
		{"", -1},
	}
	for _, tt := range tests {
		got := parseCoverageLine(tt.line)
		if got != tt.want {
			t.Errorf("parseCoverageLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestExtractWorkstreamsFromBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   []string
	}{
		{"feature/F031-something", nil},
		{"feature/F053-x", nil},
		{"ws/00-053-01", []string{"00-053-01"}},
		{"feature/F014-00-014-01-fix", []string{"00-014-01"}},
		{"branch-00-053-16-00-053-17", []string{"00-053-16", "00-053-17"}},
	}
	for _, tt := range tests {
		got := extractWorkstreamsFromBranch(tt.branch)
		if len(got) != len(tt.want) {
			t.Errorf("extractWorkstreamsFromBranch(%q) = %v, want %v", tt.branch, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractWorkstreamsFromBranch(%q)[%d] = %q, want %q", tt.branch, i, got[i], tt.want[i])
			}
		}
	}
}

func TestExtractBeadsIDsFromText(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "current and legacy prefixes",
			text: "feat: sdplab-8rx\nbody sdp_dev-2aq.7.3 and sdplab-8rx again",
			want: []string{"sdplab-8rx", "sdp_dev-2aq.7.3"},
		},
		{
			name: "no ids",
			text: "feat: no beads id here",
			want: nil,
		},
	}

	for _, tt := range tests {
		got := extractBeadsIDsFromText(tt.text)
		if len(got) != len(tt.want) {
			t.Fatalf("%s: len = %d, want %d (%v)", tt.name, len(got), len(tt.want), got)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("%s: got[%d] = %q, want %q", tt.name, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMatchesAnyPrefix(t *testing.T) {
	prefixes := []string{"internal/", "cmd/", "docs/"}
	if !matchesAnyPrefix("internal/evidence/foo.go", prefixes) {
		t.Error("internal/ should match")
	}
	if !matchesAnyPrefix("cmd/auto-attest/main.go", prefixes) {
		t.Error("cmd/ should match")
	}
	if matchesAnyPrefix("other/file.go", prefixes) {
		t.Error("other/ should not match")
	}
	if !matchesAnyPrefix("internal", prefixes) {
		t.Error("exact match internal should match")
	}
}

func TestWriteAutoAttestationReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	stmt := NewStatement(
		[]intoto.Subject{{Name: "test", Digest: common.DigestSet{"sha256": "abc"}}}, //nolint:staticcheck
		CodingWorkflowPredicate{
			Intent: Intent{IssueID: "x"},
			Trace:  Trace{Branch: "main"},
			Verification: Verification{
				Tests:    []GateResult{{Name: "go-test", Status: "pass (1 passed, 0 failed)"}},
				Lint:     []GateResult{{Name: "go-vet", Status: "pass"}},
				Coverage: &Coverage{Value: 85, Threshold: 80},
			},
			Boundary:   Boundary{Compliance: BoundaryCompliance{OK: true, Reason: "ok"}},
			Provenance: Provenance{RunID: "run-1", CapturedAt: "2026-01-01T00:00:00Z"},
		},
	)
	if err := WriteAutoAttestationReport(path, stmt); err != nil {
		t.Fatalf("WriteAutoAttestationReport: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("report should not be empty")
	}
}

// TestAutoAttest_Integration runs AutoAttest in repo (git, go test). Run with: go test -run TestAutoAttest_Integration -count=1
