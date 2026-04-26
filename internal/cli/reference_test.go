package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateReferenceDoc(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	// Register some test commands
	mustRegisterCommand(t, &CommandMetadata{
		Name:        "test",
		Category:    "Test commands",
		Description: "A test command",
		Usage:       "sdp test <arg>",
		Examples:    []string{"sdp test hello"},
	})

	doc := GenerateReferenceDoc("v1.0.0")

	if doc.Version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", doc.Version)
	}

	if doc.GeneratedAt == "" {
		t.Error("generated timestamp is empty")
	}

	if len(doc.Categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(doc.Categories))
	}

	stats := doc.Stats
	if stats.TotalCommands != 1 {
		t.Errorf("expected 1 total command, got %d", stats.TotalCommands)
	}
}

func TestGetCategoryDescription(t *testing.T) {
	tests := []struct {
		name     string
		category string
		wantDesc bool
	}{
		{
			name:     "known category",
			category: "Card commands",
			wantDesc: true,
		},
		{
			name:     "unknown category",
			category: "Unknown category",
			wantDesc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := getCategoryDescription(tt.category)
			hasDesc := desc != ""

			if hasDesc != tt.wantDesc {
				t.Errorf("getCategoryDescription() = %v, want %v", hasDesc, tt.wantDesc)
			}
		})
	}
}

func TestWriteMarkdown(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	// Register test command
	mustRegisterCommand(t, &CommandMetadata{
		Name:        "test",
		Category:    "Test commands",
		Description: "Test description",
		Usage:       "sdp test <arg>",
	})

	doc := GenerateReferenceDoc("v1.0.0")

	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "CLI_REFERENCE.md")

	err := doc.WriteMarkdown(outputPath)
	if err != nil {
		t.Fatalf("WriteMarkdown failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("output file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check for expected sections
	if !strings.Contains(contentStr, "# SDP CLI Reference") {
		t.Error("missing title")
	}

	if !strings.Contains(contentStr, "## Statistics") {
		t.Error("missing statistics section")
	}

	if !strings.Contains(contentStr, "## Test commands") {
		t.Error("missing category section")
	}

	if !strings.Contains(contentStr, "### test") {
		t.Error("missing command section")
	}

	if !strings.Contains(contentStr, "Test description") {
		t.Error("missing command description")
	}
}

func TestCheckHarnessParity(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	// Register some commands
	mustRegisterCommand(t, &CommandMetadata{
		Name:     "card",
		Category: "Card commands",
	})
	mustRegisterCommand(t, &CommandMetadata{
		Name:     "board",
		Category: "Board commands",
	})
	mustRegisterCommand(t, &CommandMetadata{
		Name:               "deprecated",
		Category:           "Deprecated",
		Deprecated:         true,
		Hidden:             true, // Hide from regular listing
		DeprecationMessage: "Use 'new' instead",
	})

	tests := []struct {
		name           string
		reference      []string
		wantPassed     bool
		wantMissing    int
		wantExtra      int
		wantDeprecated int
	}{
		{
			name:        "exact match",
			reference:   []string{"card", "board"},
			wantPassed:  true,
			wantMissing: 0,
			wantExtra:   0,
		},
		{
			name:        "missing commands",
			reference:   []string{"card", "board", "dispatch"},
			wantPassed:  false,
			wantMissing: 1,
			wantExtra:   0,
		},
		{
			name:        "extra commands",
			reference:   []string{"card"},
			wantPassed:  true,
			wantMissing: 0,
			wantExtra:   1, // board
		},
		{
			name:           "deprecated in reference",
			reference:      []string{"card", "board", "deprecated"},
			wantPassed:     true, // Deprecated commands in reference don't fail parity
			wantMissing:    0,    // Not counted as missing, just deprecated
			wantExtra:      0,    // All current commands are in reference
			wantDeprecated: 1,    // Should be detected as deprecated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckHarnessParity(tt.reference)

			if result.Passed != tt.wantPassed {
				t.Errorf("CheckHarnessParity() passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if len(result.Missing) != tt.wantMissing {
				t.Errorf("CheckHarnessParity() missing = %d, want %d", len(result.Missing), tt.wantMissing)
			}

			if len(result.Extra) != tt.wantExtra {
				t.Errorf("CheckHarnessParity() extra = %d, want %d", len(result.Extra), tt.wantExtra)
			}

			if len(result.Deprecated) != tt.wantDeprecated {
				t.Errorf("CheckHarnessParity() deprecated = %d, want %d", len(result.Deprecated), tt.wantDeprecated)
			}
		})
	}
}

func TestFormatParityReport(t *testing.T) {
	result := &ParityCheckResult{
		Passed:     false,
		Missing:    []string{"dispatch", "orchestrate"},
		Extra:      []string{"experimental"},
		Deprecated: []string{"old-cmd"},
	}

	result.generateSummary()
	report := result.FormatParityReport()

	if !strings.Contains(report, "Parity check failed") {
		t.Error("report should indicate failure")
	}

	if !strings.Contains(report, "dispatch") {
		t.Error("report should mention missing dispatch command")
	}

	if !strings.Contains(report, "experimental") {
		t.Error("report should mention extra experimental command")
	}

	if !strings.Contains(report, "old-cmd") {
		t.Error("report should mention deprecated command")
	}
}

func TestExitStatusForParity(t *testing.T) {
	tests := []struct {
		name     string
		passed   bool
		wantExit int
	}{
		{
			name:     "passed",
			passed:   true,
			wantExit: 0,
		},
		{
			name:     "failed",
			passed:   false,
			wantExit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ParityCheckResult{Passed: tt.passed}
			exit := result.ExitStatusForParity()

			if exit != tt.wantExit {
				t.Errorf("ExitStatusForParity() = %d, want %d", exit, tt.wantExit)
			}
		})
	}
}

func TestGenerateManPage(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	mustRegisterCommand(t, &CommandMetadata{
		Name:        "test",
		Category:    "Test",
		Description: "Test command for man page",
		Usage:       "sdp test <arg>",
		Subcommands: []string{"sub1", "sub2"},
		Examples:    []string{"sdp test sub1"},
	})

	manPage, err := GenerateManPage("test")
	if err != nil {
		t.Fatalf("GenerateManPage failed: %v", err)
	}

	if !strings.Contains(manPage, ".TH") {
		t.Error("man page missing TH directive")
	}

	if !strings.Contains(manPage, "Test command for man page") {
		t.Error("man page missing description")
	}

	if !strings.Contains(manPage, ".SH SYNOPSIS") {
		t.Error("man page missing SYNOPSIS section")
	}

	if !strings.Contains(manPage, "sub1") {
		t.Error("man page missing subcommands")
	}

	if !strings.Contains(manPage, "sdp test sub1") {
		t.Error("man page missing examples")
	}
}

func TestGenerateManPageNotFound(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	_, err := GenerateManPage("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent command, got nil")
	}
}

func TestGenerateManPageDeprecated(t *testing.T) {
	// Reset global registry for testing
	globalRegistry = NewRegistry()

	mustRegisterCommand(t, &CommandMetadata{
		Name:               "old",
		Category:           "Test",
		Description:        "Old command",
		Deprecated:         true,
		DeprecationMessage: "Use 'new' instead",
	})

	manPage, err := GenerateManPage("old")
	if err != nil {
		t.Fatalf("GenerateManPage failed: %v", err)
	}

	if !strings.Contains(manPage, ".SH DEPRECATED") {
		t.Error("man page missing DEPRECATED section")
	}

	if !strings.Contains(manPage, "Use 'new' instead") {
		t.Error("man page missing deprecation message")
	}
}
