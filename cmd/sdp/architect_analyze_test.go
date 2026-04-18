package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sdp_dev/internal/architect"
	"sdp_dev/internal/architect/c4"
	"sdp_dev/internal/architect/extract"
)

// TestArchitectAnalyzeIntegration tests the full analyze pipeline end-to-end.
func TestArchitectAnalyzeIntegration(t *testing.T) {
	// Create a temporary test repository
	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	// Test basic analyze with default settings
	config := architect.PipelineConfig{
		RepoRoot: tmpDir,
		Tier:     architect.Tier2,
		NoLLM:    true,
		Timeout:  30 * time.Second,
		Format:   "json",
	}

	extractors := extract.DefaultExtractors()
	reporter := architect.NewProgressReporter(false)
	pipeline := architect.NewPipeline(config, extractors)
	pipeline.SetProgressCallback(reporter.Callback())

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Profile == nil {
		t.Fatal("Profile is nil")
	}
	if result.Report == nil {
		t.Fatal("Report is nil")
	}
	if result.ReferenceModel == nil {
		t.Fatal("ReferenceModel is nil")
	}
	if result.Duration == 0 {
		t.Error("Duration should be > 0")
	}

	// Verify profile content
	if result.Profile.Name == "" {
		t.Error("Profile name should not be empty")
	}
	if result.Profile.Metrics.TotalFiles == 0 {
		t.Error("Should have detected some files")
	}

	// Test JSON output
	jsonData, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("JSON output should not be empty")
	}

	// Test text output
	textData := result.ToText()
	if len(textData) == 0 {
		t.Error("Text output should not be empty")
	}
}

// TestArchitectAnalyzeWithC4Diagrams tests C4 diagram generation.
func TestArchitectAnalyzeWithC4Diagrams(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	config := architect.PipelineConfig{
		RepoRoot: tmpDir,
		Tier:     architect.Tier2,
		NoLLM:    true,
		Timeout:  30 * time.Second,
	}

	extractors := extract.DefaultExtractors()
	pipeline := architect.NewPipeline(config, extractors)

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	if result.ReferenceModel == nil {
		t.Fatal("ReferenceModel is nil")
	}

	// Generate C4 diagrams
	opts := c4.RenderOptions{
		Direction: "TB",
		Theme:     "default",
	}

	// Test L1 diagram
	l1, err := c4.RenderL1(result.ReferenceModel, opts)
	if err != nil {
		t.Fatalf("RenderL1 failed: %v", err)
	}
	if l1 == nil {
		t.Fatal("L1 diagram is nil")
	}
	if len(l1.MermaidCode) == 0 {
		t.Error("L1 diagram should have Mermaid code")
	}

	// Test L2 diagram
	l2, err := c4.RenderL2(result.ReferenceModel, opts)
	if err != nil {
		t.Fatalf("RenderL2 failed: %v", err)
	}
	if l2 == nil {
		t.Fatal("L2 diagram is nil")
	}
	if len(l2.MermaidCode) == 0 {
		t.Error("L2 diagram should have Mermaid code")
	}
}

// TestArchitectAnalyzeTierLevels tests different tier levels.
func TestArchitectAnalyzeTierLevels(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	tiers := []architect.TierLevel{architect.Tier1, architect.Tier2, architect.Tier3}

	for _, tier := range tiers {
		t.Run("", func(t *testing.T) {
			config := architect.PipelineConfig{
				RepoRoot: tmpDir,
				Tier:     tier,
				NoLLM:    true,
				Timeout:  30 * time.Second,
			}

			extractors := extract.DefaultExtractors()
			pipeline := architect.NewPipeline(config, extractors)

			ctx := context.Background()
			result, err := pipeline.Run(ctx)
			if err != nil {
				t.Fatalf("Pipeline.Run failed for tier %d: %v", tier, err)
			}

			if result.Profile == nil {
				t.Errorf("Profile is nil for tier %d", tier)
			}
			if result.ReferenceModel == nil {
				t.Errorf("ReferenceModel is nil for tier %d", tier)
			}
		})
	}
}

// TestArchitectAnalyzeOutputFormats tests different output formats.
func TestArchitectAnalyzeOutputFormats(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	config := architect.PipelineConfig{
		RepoRoot: tmpDir,
		Tier:     architect.Tier1,
		NoLLM:    true,
		Timeout:  30 * time.Second,
	}

	extractors := extract.DefaultExtractors()
	pipeline := architect.NewPipeline(config, extractors)

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	// Test JSON format
	jsonData, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("JSON output should not be empty")
	}

	// Test text format
	textData := result.ToText()
	if len(textData) == 0 {
		t.Error("Text output should not be empty")
	}

	// Verify text format contains key information
	// Note: profile name might be inferred from directory, so we just check for basic content
	if !contains(textData, "Architecture Analysis") {
		t.Error("Text output should contain analysis header")
	}
}

// TestArchitectAnalyzeWithFiltering tests extractor filtering.
func TestArchitectAnalyzeWithFiltering(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	// Test with only specific extractors
	config := architect.PipelineConfig{
		RepoRoot:         tmpDir,
		Tier:             architect.Tier1,
		NoLLM:            true,
		Timeout:          30 * time.Second,
		ExtractorNames:   []string{"filetree", "deps"},
		AllowExternalLLM: false,
	}

	extractors := filterExtractorsByName(extract.DefaultExtractors(), config.ExtractorNames)
	pipeline := architect.NewPipeline(config, extractors)

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	if result.Profile == nil {
		t.Fatal("Profile is nil")
	}
}

// TestArchitectAnalyzeProgressReporting tests progress reporting.
func TestArchitectAnalyzeProgressReporting(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRepo(t, tmpDir)

	config := architect.PipelineConfig{
		RepoRoot: tmpDir,
		Tier:     architect.Tier1,
		NoLLM:    true,
		Timeout:  30 * time.Second,
		Verbose:  true,
	}

	extractors := extract.DefaultExtractors()
	reporter := architect.NewProgressReporter(true)
	pipeline := architect.NewPipeline(config, extractors)
	pipeline.SetProgressCallback(reporter.Callback())

	ctx := context.Background()
	result, err := pipeline.Run(ctx)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}

	// Summary should not panic
	reporter.Summary()
}

// createTestRepo creates a minimal test repository with known structure.
func createTestRepo(t *testing.T, dir string) {
	t.Helper()

	// Create go.mod
	goMod := `module test-repo

go 1.22
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create main.go
	mainGo := `package main

func main() {
	println("Hello, World!")
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}

	// Create README.md
	readme := `# Test Repository

This is a test repository for architect analysis.
`
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0644); err != nil {
		t.Fatalf("failed to create README.md: %v", err)
	}

	// Create internal directory
	internalDir := filepath.Join(dir, "internal")
	if err := os.MkdirAll(internalDir, 0755); err != nil {
		t.Fatalf("failed to create internal dir: %v", err)
	}

	// Create app.go
	appGo := `package app

func Run() {
	println("Running app")
}
`
	if err := os.WriteFile(filepath.Join(internalDir, "app.go"), []byte(appGo), 0644); err != nil {
		t.Fatalf("failed to create app.go: %v", err)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring searches for a substring within a string.
func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
