package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCleanupPlans(t *testing.T) {
	checkResult := &CheckResult{
		Issues: []ReferenceIssue{
			{
				Reference: Reference{
					Source:     "test.md",
					LineNumber: 10,
					Target:     "../../workstreams/backlog/00-999-01.md",
					Type:       RefTypeWorkstream,
				},
				Severity: "error",
				Message:  "Workstream file does not exist",
			},
			{
				Reference: Reference{
					Source:     "test.md",
					LineNumber: 20,
					Target:     "F999",
					Type:       RefTypeFeature,
				},
				Severity: "warning",
				Message:  "No workstream file found",
			},
		},
	}

	plans := GenerateCleanupPlans(checkResult)

	if len(plans) != 1 {
		t.Errorf("expected 1 plan, got %d", len(plans))
	}

	if plans[0].File != "test.md" {
		t.Errorf("expected file test.md, got %s", plans[0].File)
	}

	if len(plans[0].Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(plans[0].Issues))
	}

	// Should have fixes for both issues
	if len(plans[0].Fixes) != 2 {
		t.Errorf("expected 2 fixes, got %d", len(plans[0].Fixes))
	}
}

func TestApplyFixesToFileDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	content := "# Test\n\nSee [missing](../../workstreams/backlog/00-999-01.md)\n"
	mustWriteFile(t, testFile, []byte(content))

	plan := CleanupPlan{
		File: testFile,
		Fixes: []ReferenceFix{
			{
				OldContent: "[missing](../../workstreams/backlog/00-999-01.md)",
				NewContent: "[FIXME: broken workstream ref](../../workstreams/backlog/00-999-01.md)",
				AutoFix:    true,
				Reason:     "Test fix",
			},
		},
	}

	opts := CleanupOptions{
		DryRun: true,
	}

	modified, fixesApplied, err := applyFixesToFile(plan, opts)
	if err != nil {
		t.Fatalf("applyFixesToFile failed: %v", err)
	}

	if modified {
		t.Error("file should not be modified in dry-run mode")
	}

	if fixesApplied != 0 {
		t.Errorf("expected 0 fixes applied in dry-run, got %d", fixesApplied)
	}

	// Verify file content is unchanged
	original, _ := os.ReadFile(testFile)
	if string(original) != content {
		t.Error("file content changed in dry-run mode")
	}
}

func TestApplyFixesToFileWithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")
	content := "# Test\n\nSee [missing](../../workstreams/backlog/00-999-01.md)\n"
	mustWriteFile(t, testFile, []byte(content))

	plan := CleanupPlan{
		File: testFile,
		Fixes: []ReferenceFix{
			{
				OldContent: "[missing](../../workstreams/backlog/00-999-01.md)",
				NewContent: "[FIXME](../../workstreams/backlog/00-999-01.md)",
				AutoFix:    true,
			},
		},
	}

	opts := CleanupOptions{
		Backup:    true,
		AutoApply: true,
	}

	modified, fixesApplied, err := applyFixesToFile(plan, opts)
	if err != nil {
		t.Fatalf("applyFixesToFile failed: %v", err)
	}

	if !modified {
		t.Error("file should be modified")
	}

	if fixesApplied != 1 {
		t.Errorf("expected 1 fix applied, got %d", fixesApplied)
	}

	// Verify backup was created
	backupPath := testFile + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}

	// Verify backup content
	backupContent, _ := os.ReadFile(backupPath)
	if string(backupContent) != content {
		t.Error("backup content doesn't match original")
	}

	// Verify file was modified
	newContent, _ := os.ReadFile(testFile)
	if strings.Contains(string(newContent), "[FIXME]") {
		// Fix was applied
	} else {
		t.Error("fix was not applied")
	}
}

func TestQuickFix(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.md")

	tests := []struct {
		name     string
		input    string
		wantFix  bool
		contains string
	}{
		{
			name:     "absolute markdown link",
			input:    "# Test\n\nSee [text](/docs/workstreams/backlog/00-001-01.md)\n",
			wantFix:  true,
			contains: "[text](00-001-01.md)",
		},
		{
			name:     "lowercase feature ID",
			input:    "# Test\n\nThis implements f042.\n",
			wantFix:  true,
			contains: "F042",
		},
		{
			name:     "double slash",
			input:    "# Test\n\nSee [text](../../file.md)\n",
			wantFix:  true,
			contains: "../file.md",
		},
		{
			name:    "no issues",
			input:   "# Test\n\nSee [text](../file.md)\n",
			wantFix: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mustWriteFile(t, testFile, []byte(tt.input))

			err := QuickFix(testFile)
			if err != nil {
				t.Fatalf("QuickFix failed: %v", err)
			}

			content, _ := os.ReadFile(testFile)
			contentStr := string(content)

			if tt.wantFix {
				if !strings.Contains(contentStr, tt.contains) {
					t.Errorf("QuickFix() expected to find %q, got %q", tt.contains, contentStr)
				}
			}
		})
	}
}

func TestRemoveDeadReferences(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectory structure
	docsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	mustMkdirAll(t, docsDir)
	testFile := filepath.Join(docsDir, "test.md")

	// Create a file with both valid and broken references
	content := `# Test

Valid reference to [README](../../README.md).

Broken reference to [MISSING](../../MISSING.md).

More text.
`
	mustWriteFile(t, testFile, []byte(content))

	// Create a valid referenced file (../../README.md from docs/workstreams/backlog)
	// This should be at docs/README.md
	readmePath := filepath.Join(tmpDir, "docs", "README.md")
	mustWriteFile(t, readmePath, []byte("# README"))

	removed, err := RemoveDeadReferences(testFile, tmpDir)
	if err != nil {
		t.Fatalf("RemoveDeadReferences failed: %v", err)
	}

	if removed != 1 {
		t.Errorf("expected 1 reference removed, got %d", removed)
	}

	// Verify the broken reference line was removed
	newContent, _ := os.ReadFile(testFile)
	contentStr := string(newContent)

	if strings.Contains(contentStr, "MISSING.md") {
		t.Error("broken reference was not removed")
	}

	if !strings.Contains(contentStr, "README.md") {
		t.Error("valid reference was removed")
	}
}

func TestFormatCleanupPlan(t *testing.T) {
	plan := CleanupPlan{
		File: "test.md",
		Issues: []ReferenceIssue{
			{
				Reference: Reference{
					LineNumber: 10,
					Target:     "F999",
				},
				Message: "Feature not found",
			},
		},
		Fixes: []ReferenceFix{
			{
				Issue: ReferenceIssue{
					Reference: Reference{LineNumber: 10},
					Message:   "Feature not found",
				},
				Reason:     "Manual review required",
				AutoFix:    false,
				OldContent: "F999",
				NewContent: "F999",
			},
		},
	}

	report := FormatCleanupPlan(plan)

	if !strings.Contains(report, "File: test.md") {
		t.Error("report missing file name")
	}

	if !strings.Contains(report, "Issues found: 1") {
		t.Error("report missing issue count")
	}

	if !strings.Contains(report, "Fixes proposed: 1") {
		t.Error("report missing fix count")
	}

	if !strings.Contains(report, "Manual review required") {
		t.Error("report missing reason")
	}
}

func TestFormatCleanupResult(t *testing.T) {
	result := &CleanupResult{
		FilesScanned:  10,
		FilesModified: 2,
		IssuesFound:   5,
		IssuesFixed:   3,
		IssuesSkipped: 2,
		ModifiedFiles: []string{"file1.md", "file2.md"},
	}

	report := FormatCleanupResult(result)

	if !strings.Contains(report, "Files scanned: 10") {
		t.Error("report missing scanned count")
	}

	if !strings.Contains(report, "Files modified: 2") {
		t.Error("report missing modified count")
	}

	if !strings.Contains(report, "Issues found: 5") {
		t.Error("report missing found count")
	}

	if !strings.Contains(report, "Issues fixed: 3") {
		t.Error("report missing fixed count")
	}

	if !strings.Contains(report, "file1.md") {
		t.Error("report missing modified file")
	}
}

func TestFilepathBase(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			path: "../../workstreams/backlog/00-001-01.md",
			want: "00-001-01.md",
		},
		{
			path: "/absolute/path/to/file.md",
			want: "file.md",
		},
		{
			path: "simple.md",
			want: "simple.md",
		},
		{
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := filepathBase(tt.path)
			if got != tt.want {
				t.Errorf("filepathBase(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBatchApplyCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workstream directory
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	mustMkdirAll(t, wsDir)

	// Create a file with a broken reference
	brokenContent := `---
ws_id: 00-001-01
feature_id: F001
---

# Test

Broken ref: [MISSING](../../MISSING.md)
`
	testFile := filepath.Join(wsDir, "00-001-01.md")
	mustWriteFile(t, testFile, []byte(brokenContent))

	opts := CleanupOptions{
		DryRun:    true,
		AutoApply: false,
		Verbose:   false,
	}

	result, err := BatchApplyCleanup(wsDir, opts)
	if err != nil {
		t.Fatalf("BatchApplyCleanup failed: %v", err)
	}

	if result.FilesScanned == 0 {
		t.Error("no files were scanned")
	}

	if result.IssuesFound == 0 {
		t.Error("no issues were found")
	}
}
