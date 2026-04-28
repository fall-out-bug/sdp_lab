package backlog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/backlog"
)

// writeWSFile creates a ws file at wsDir/name with optional frontmatter status.
func writeWSFile(t *testing.T, wsDir, name, status string) {
	t.Helper()
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		t.Fatalf("mkdirAll %s: %v", wsDir, err)
	}
	var content string
	if status != "" {
		content = "---\nws_id: " + name + "\nstatus: " + status + "\n---\n\n# " + name + "\n"
	} else {
		content = "# " + name + "\n"
	}
	dest := filepath.Join(wsDir, name)
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile %s: %v", dest, err)
	}
}

// TestAudit_Clean verifies that features with ws files OR children produce 0 findings.
func TestAudit_Clean(t *testing.T) {
	root := t.TempDir()
	wsDir := filepath.Join(root, "docs", "workstreams", "backlog")

	// F141 has a ws file.
	writeWSFile(t, wsDir, "00-141-01.md", "open")
	// F100-02 has a ws file.
	writeWSFile(t, wsDir, "00-100-02.md", "")
	// F999 has children (DepCount > 0), no ws file.

	features := []backlog.Feature{
		{BeadID: "sdplab-aaa1", FID: "F141", Title: "F141: Multi-harness", Status: "open", IssueType: "epic", DepCount: 0},
		{BeadID: "sdplab-aaa2", FID: "F100-02", Title: "F100-02: Some feature", Status: "open", IssueType: "feature", DepCount: 0},
		{BeadID: "sdplab-aaa3", FID: "F999", Title: "F999: Has children", Status: "open", IssueType: "feature", DepCount: 3},
	}

	opts := backlog.AuditOpts{RepoRoot: root, IncludeStatus: []string{"open"}}
	result := backlog.Audit(opts, features)

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d: %v", len(result.Findings), result.Findings)
	}
	if result.Checked != 3 {
		t.Errorf("expected 3 checked, got %d", result.Checked)
	}
}

// TestAudit_FlagsLeaflessFeature verifies that a feature with ws_count=0 and
// DepCount=0 produces exactly 1 finding ("picker bait").
func TestAudit_FlagsLeaflessFeature(t *testing.T) {
	root := t.TempDir()
	// No ws files created — wsDir doesn't even exist.

	features := []backlog.Feature{
		{BeadID: "sdplab-bbb1", FID: "F200", Title: "F200: Bare feature", Status: "open", IssueType: "feature", DepCount: 0},
	}

	opts := backlog.AuditOpts{RepoRoot: root, IncludeStatus: []string{"open"}}
	result := backlog.Audit(opts, features)

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %v", len(result.Findings), result.Findings)
	}
	f := result.Findings[0]
	if f.BeadID != "sdplab-bbb1" {
		t.Errorf("expected BeadID sdplab-bbb1, got %s", f.BeadID)
	}
	if f.FID != "F200" {
		t.Errorf("expected FID F200, got %s", f.FID)
	}
	// Reason should mention "picker bait"
	if !containsSub(f.Reason, "picker bait") {
		t.Errorf("expected reason to mention 'picker bait', got: %q", f.Reason)
	}
}

// TestAudit_AcceptsFeatureWithChildren verifies that a feature with ws_count=0
// but DepCount>0 is NOT flagged.
func TestAudit_AcceptsFeatureWithChildren(t *testing.T) {
	root := t.TempDir()
	// No ws files — but the feature has children.

	features := []backlog.Feature{
		{BeadID: "sdplab-ccc1", FID: "F300", Title: "F300: Epic with children", Status: "open", IssueType: "epic", DepCount: 2},
	}

	opts := backlog.AuditOpts{RepoRoot: root, IncludeStatus: []string{"open"}}
	result := backlog.Audit(opts, features)

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings (has children), got %d: %v", len(result.Findings), result.Findings)
	}
}

// TestAudit_AcceptsFeatureWithWS verifies that a feature with ws_count>0 and
// DepCount=0 is NOT flagged in non-strict mode.
func TestAudit_AcceptsFeatureWithWS(t *testing.T) {
	root := t.TempDir()
	wsDir := filepath.Join(root, "docs", "workstreams", "backlog")
	writeWSFile(t, wsDir, "00-400-01.md", "open")

	features := []backlog.Feature{
		{BeadID: "sdplab-ddd1", FID: "F400", Title: "F400: Has ws", Status: "open", IssueType: "feature", DepCount: 0},
	}

	opts := backlog.AuditOpts{RepoRoot: root, IncludeStatus: []string{"open"}, Strict: false}
	result := backlog.Audit(opts, features)

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings (has ws file), got %d: %v", len(result.Findings), result.Findings)
	}
}

// TestAudit_StrictFlagsDesignPending verifies that with Strict=true, a feature
// whose ws status==design-pending and DepCount=0 gets flagged.
func TestAudit_StrictFlagsDesignPending(t *testing.T) {
	root := t.TempDir()
	wsDir := filepath.Join(root, "docs", "workstreams", "backlog")
	// F500-01 has ws file with status design-pending.
	writeWSFile(t, wsDir, "00-500-01.md", "design-pending")

	features := []backlog.Feature{
		{BeadID: "sdplab-eee1", FID: "F500-01", Title: "F500-01: Design pending ws", Status: "open", IssueType: "feature", DepCount: 0},
	}

	opts := backlog.AuditOpts{RepoRoot: root, IncludeStatus: []string{"open"}, Strict: true}
	result := backlog.Audit(opts, features)

	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding (strict+design-pending), got %d: %v", len(result.Findings), result.Findings)
	}
	if !containsSub(result.Findings[0].Reason, "design-pending") {
		t.Errorf("expected reason to mention 'design-pending', got: %q", result.Findings[0].Reason)
	}
}

// TestAudit_NonStrictIgnoresDesignPending verifies that Strict=false does NOT
// flag design-pending ws features.
func TestAudit_NonStrictIgnoresDesignPending(t *testing.T) {
	root := t.TempDir()
	wsDir := filepath.Join(root, "docs", "workstreams", "backlog")
	writeWSFile(t, wsDir, "00-600-01.md", "design-pending")

	features := []backlog.Feature{
		{BeadID: "sdplab-fff1", FID: "F600-01", Title: "F600-01: Design pending", Status: "open", IssueType: "feature", DepCount: 0},
	}

	opts := backlog.AuditOpts{RepoRoot: root, IncludeStatus: []string{"open"}, Strict: false}
	result := backlog.Audit(opts, features)

	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings (non-strict ignores design-pending), got %d: %v", len(result.Findings), result.Findings)
	}
}

// TestFormatReport_Clean verifies that a clean result renders "ok:" message.
func TestFormatReport_Clean(t *testing.T) {
	r := backlog.Result{Findings: nil, Checked: 5}
	report := backlog.FormatReport(r)
	if !containsSub(report, "ok:") {
		t.Errorf("expected 'ok:' in clean report, got: %q", report)
	}
	if !containsSub(report, "5") {
		t.Errorf("expected checked count '5' in report, got: %q", report)
	}
}

// TestFormatReport_WithFindings verifies that findings are rendered.
func TestFormatReport_WithFindings(t *testing.T) {
	r := backlog.Result{
		Findings: []backlog.Finding{
			{BeadID: "sdplab-x1", FID: "F999", Title: "F999: Test", Reason: "no ws file and no children — picker bait"},
		},
		Checked: 1,
	}
	report := backlog.FormatReport(r)
	if !containsSub(report, "BACKLOG DRIFT") {
		t.Errorf("expected 'BACKLOG DRIFT' in report, got: %q", report)
	}
	if !containsSub(report, "sdplab-x1") {
		t.Errorf("expected bead id in report, got: %q", report)
	}
}

func containsSub(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
