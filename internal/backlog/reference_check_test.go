package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func TestExtractReferences(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		source    string
		lineNum   int
		wantCount int
		wantTypes []ReferenceType
	}{
		{
			name:      "workstream reference",
			line:      "See [F042](../../workstreams/backlog/00-042-01.md) for details.",
			source:    "docs/workstreams/backlog/00-001-01.md",
			lineNum:   10,
			wantCount: 2, // Workstream link + feature ID in link text
			wantTypes: []ReferenceType{RefTypeWorkstream, RefTypeFeature},
		},
		{
			name:      "feature reference",
			line:      "This implements F042 and F101-02.",
			source:    "docs/workstreams/backlog/00-001-01.md",
			lineNum:   5,
			wantCount: 2,
			wantTypes: []ReferenceType{RefTypeFeature, RefTypeFeature},
		},
		{
			name:      "bead reference",
			line:      "Bead sdplab-abc123 implements this.",
			source:    "docs/workstreams/backlog/00-001-01.md",
			lineNum:   3,
			wantCount: 1,
			wantTypes: []ReferenceType{RefTypeBead},
		},
		{
			name:      "file reference",
			line:      "See [README](../../README.md) for more info.",
			source:    "docs/workstreams/backlog/00-001-01.md",
			lineNum:   7,
			wantCount: 1,
			wantTypes: []ReferenceType{RefTypeFile},
		},
		{
			name:      "URL reference",
			line:      "Visit https://example.com for details.",
			source:    "docs/workstreams/backlog/00-001-01.md",
			lineNum:   8,
			wantCount: 1,
			wantTypes: []ReferenceType{RefTypeExternal},
		},
		{
			name:      "mixed references",
			line:      "This implements F042. See [docs](../../README.md) and https://example.com",
			source:    "docs/workstreams/backlog/00-001-01.md",
			lineNum:   10,
			wantCount: 3,
			wantTypes: []ReferenceType{RefTypeFeature, RefTypeFile, RefTypeExternal},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := extractReferences(tt.line, tt.source, tt.lineNum)

			if len(refs) != tt.wantCount {
				t.Errorf("extractReferences() count = %d, want %d", len(refs), tt.wantCount)
			}

			if len(refs) >= len(tt.wantTypes) {
				for i, wantType := range tt.wantTypes {
					if refs[i].Type != wantType {
						t.Errorf("extractReferences()[%d].Type = %v, want %v", i, refs[i].Type, wantType)
					}
				}
			}
		})
	}
}

func TestValidateWorkstreamReference(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a workstream file
	wsPath := filepath.Join(tmpDir, "docs", "workstreams", "backlog", "00-042-01.md")
	mustMkdirAll(t, filepath.Dir(wsPath))
	mustWriteFile(t, wsPath, []byte("# Test"))

	opts := DefaultCheckOptions(tmpDir)

	tests := []struct {
		name      string
		target    string
		source    string
		wantIssue bool
		wantSev   string
	}{
		{
			name:      "valid workstream",
			target:    "../../workstreams/backlog/00-042-01.md",
			source:    "docs/workstreams/backlog/00-001-01.md",
			wantIssue: false,
		},
		{
			name:      "invalid workstream",
			target:    "../../workstreams/backlog/00-999-01.md",
			source:    "docs/workstreams/backlog/00-001-01.md",
			wantIssue: true,
			wantSev:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := Reference{
				Type:   RefTypeWorkstream,
				Target: tt.target,
				Source: tt.source,
			}

			issue := validateWorkstreamReference(ref, opts)

			hasIssue := issue != nil
			if hasIssue != tt.wantIssue {
				t.Errorf("validateWorkstreamReference() issue = %v, want %v", hasIssue, tt.wantIssue)
			}

			if issue != nil && tt.wantSev != "" && issue.Severity != tt.wantSev {
				t.Errorf("validateWorkstreamReference() severity = %q, want %q", issue.Severity, tt.wantSev)
			}
		})
	}
}

func TestValidateFeatureReference(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workstream files
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	mustMkdirAll(t, wsDir)
	mustWriteFile(t, filepath.Join(wsDir, "00-042-01.md"), []byte("# Test"))
	mustWriteFile(t, filepath.Join(wsDir, "00-101-02.md"), []byte("# Test"))

	opts := DefaultCheckOptions(tmpDir)

	tests := []struct {
		name      string
		target    string
		wantIssue bool
		wantSev   string
	}{
		{
			name:      "valid feature with sub",
			target:    "F042-01",
			wantIssue: false,
		},
		{
			name:      "valid feature epic",
			target:    "F042",
			wantIssue: false,
		},
		{
			name:      "feature without ws file",
			target:    "F999",
			wantIssue: true,
			wantSev:   "warning",
		},
		{
			name:      "invalid feature format",
			target:    "F99",
			wantIssue: true,
			wantSev:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := Reference{
				Type:   RefTypeFeature,
				Target: tt.target,
				Source: "docs/workstreams/backlog/00-001-01.md",
			}

			issue := validateFeatureReference(ref, opts)

			hasIssue := issue != nil
			if hasIssue != tt.wantIssue {
				t.Errorf("validateFeatureReference() issue = %v, want %v", hasIssue, tt.wantIssue)
			}

			if issue != nil && tt.wantSev != "" && issue.Severity != tt.wantSev {
				t.Errorf("validateFeatureReference() severity = %q, want %q", issue.Severity, tt.wantSev)
			}
		})
	}
}

func TestValidateFileReference(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a reference file at the correct location
	// The source path is "docs/workstreams/backlog/00-001-01.md"
	// So "../../README.md" would be at "docs/README.md"
	docsDir := filepath.Join(tmpDir, "docs")
	mustMkdirAll(t, docsDir)
	readmePath := filepath.Join(docsDir, "README.md")
	mustWriteFile(t, readmePath, []byte("# Test"))

	opts := DefaultCheckOptions(tmpDir)

	tests := []struct {
		name      string
		target    string
		source    string
		wantIssue bool
		wantSev   string
	}{
		{
			name:      "valid file",
			target:    "../../README.md",
			source:    "docs/workstreams/backlog/00-001-01.md",
			wantIssue: false,
		},
		{
			name:      "invalid file",
			target:    "../../NONEXISTENT.md",
			source:    "docs/workstreams/backlog/00-001-01.md",
			wantIssue: true,
			wantSev:   "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := Reference{
				Type:   RefTypeFile,
				Target: tt.target,
				Source: tt.source,
			}

			issue := validateFileReference(ref, opts)

			hasIssue := issue != nil
			if hasIssue != tt.wantIssue {
				t.Errorf("validateFileReference() issue = %v, want %v", hasIssue, tt.wantIssue)
			}

			if issue != nil && tt.wantSev != "" && issue.Severity != tt.wantSev {
				t.Errorf("validateFileReference() severity = %q, want %q", issue.Severity, tt.wantSev)
			}
		})
	}
}

func TestCheckReferenceIntegrity(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workstream directory structure
	wsDir := filepath.Join(tmpDir, "docs", "workstreams", "backlog")
	mustMkdirAll(t, wsDir)

	// Create a workstream file with valid references
	validWs := `---
ws_id: 00-001-01
feature_id: F001
---

# Test Feature

This implements F001.

See [F001](../../workstreams/backlog/00-001-01.md) for details.
`
	mustWriteFile(t, filepath.Join(wsDir, "00-001-01.md"), []byte(validWs))

	// Create a workstream file with broken references
	brokenWs := `---
ws_id: 00-002-01
feature_id: F002
---

# Broken Feature

This implements F999.

See [Missing](../../workstreams/backlog/00-999-01.md) for details.
`
	mustWriteFile(t, filepath.Join(wsDir, "00-002-01.md"), []byte(brokenWs))

	// Create a referenced file
	readmePath := filepath.Join(tmpDir, "README.md")
	mustWriteFile(t, readmePath, []byte("# Test"))

	opts := DefaultCheckOptions(tmpDir)

	result, err := CheckReferenceIntegrity(opts)
	if err != nil {
		t.Fatalf("CheckReferenceIntegrity failed: %v", err)
	}

	if result.CheckedFiles != 2 {
		t.Errorf("expected 2 checked files, got %d", result.CheckedFiles)
	}

	if result.TotalReferences < 4 {
		t.Errorf("expected at least 4 total references, got %d", result.TotalReferences)
	}

	if len(result.Issues) == 0 {
		t.Error("expected issues to be found")
	}
}

func TestFormatCheckReport(t *testing.T) {
	result := &CheckResult{
		CheckedFiles:    2,
		SkippedFiles:    0,
		TotalReferences: 10,
		ValidReferences: 7,
		Issues: []ReferenceIssue{
			{
				Reference: Reference{
					Source:     "test.md",
					LineNumber: 10,
				},
				Severity:   "error",
				Message:    "File not found",
				Suggestion: "Create the file",
			},
			{
				Reference: Reference{
					Source:     "test.md",
					LineNumber: 20,
				},
				Severity: "warning",
				Message:  "Feature may not exist",
			},
		},
	}

	report := FormatCheckReport(result)

	if !strings.Contains(report, "Reference Integrity Check") {
		t.Error("report missing title")
	}

	if !strings.Contains(report, "Files checked: 2") {
		t.Error("report missing file count")
	}

	if !strings.Contains(report, "Issues found: 2") {
		t.Error("report missing issue count")
	}

	if !strings.Contains(report, "Errors (1)") {
		t.Error("report missing error section")
	}

	if !strings.Contains(report, "Warnings (1)") {
		t.Error("report missing warning section")
	}
}

func TestExitStatusForCheck(t *testing.T) {
	tests := []struct {
		name       string
		result     *CheckResult
		strictMode bool
		wantExit   int
	}{
		{
			name: "no issues",
			result: &CheckResult{
				Issues: []ReferenceIssue{},
			},
			strictMode: false,
			wantExit:   0,
		},
		{
			name: "warnings only, not strict",
			result: &CheckResult{
				Issues: []ReferenceIssue{
					{Severity: "warning"},
				},
			},
			strictMode: false,
			wantExit:   0,
		},
		{
			name: "errors, not strict",
			result: &CheckResult{
				Issues: []ReferenceIssue{
					{Severity: "error"},
				},
			},
			strictMode: false,
			wantExit:   1,
		},
		{
			name: "warnings only, strict",
			result: &CheckResult{
				Issues: []ReferenceIssue{
					{Severity: "warning"},
				},
			},
			strictMode: true,
			wantExit:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exit := ExitStatusForCheck(tt.result, tt.strictMode)
			if exit != tt.wantExit {
				t.Errorf("ExitStatusForCheck() = %d, want %d", exit, tt.wantExit)
			}
		})
	}
}

func TestGetContext(t *testing.T) {
	line := "This is a long line with REFERENCE in the middle and more text after"
	ref := "REFERENCE"

	context := getContext(line, ref)

	// Should include surrounding context
	if !strings.Contains(context, "long line") {
		t.Error("context should include text before reference")
	}

	if !strings.Contains(context, "in the middle") {
		t.Error("context should include text after reference")
	}

	// Should be truncated with ... if needed
	if len(context) < len(ref) {
		t.Error("context should be at least as long as reference")
	}
}

func TestShouldExclude(t *testing.T) {
	patterns := []string{"node_modules", ".git", "vendor"}

	tests := []struct {
		path     string
		excluded bool
	}{
		{
			path:     "docs/workstreams/backlog/00-001-01.md",
			excluded: false,
		},
		{
			path:     "node_modules/package/file.js",
			excluded: true,
		},
		{
			path:     ".git/config",
			excluded: true,
		},
		{
			path:     "vendor/github.com/lib/file.go",
			excluded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			excluded := shouldExclude(tt.path, patterns)
			if excluded != tt.excluded {
				t.Errorf("shouldExclude(%q) = %v, want %v", tt.path, excluded, tt.excluded)
			}
		})
	}
}
