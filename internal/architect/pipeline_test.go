package architect

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPipelineConfigDefaults(t *testing.T) {
	config := PipelineConfig{
		RepoRoot: ".",
		Tier:     Tier2,
		NoLLM:    true,
	}

	extractors := []Extractor{&mockExtractor{name: "test_ext"}}
	pipeline := NewPipeline(config, extractors)

	if pipeline == nil {
		t.Fatal("NewPipeline returned nil")
	}
	if pipeline.secFilter == nil {
		t.Fatal("security filter not initialized")
	}
	if pipeline.secFilter.AllowExternalLLM {
		t.Error("AllowExternalLLM should be false when NoLLM is true")
	}
}

func TestPipelineBasicRun(t *testing.T) {
	// Create a temp directory to act as a repo root
	tmpDir := t.TempDir()
	createMinimalRepo(t, tmpDir)

	config := PipelineConfig{
		RepoRoot: tmpDir,
		Tier:     Tier1,
		NoLLM:    true,
		Timeout:  30 * time.Second,
	}

	extractors := []Extractor{&mockExtractor{name: "test_ext"}}
	pipeline := NewPipeline(config, extractors)

	var progressCalls []string
	pipeline.SetProgressCallback(func(stage PipelineStage, msg string, timing *ExtractorTiming) {
		progressCalls = append(progressCalls, string(stage))
	})

	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	if result == nil {
		t.Fatal("Pipeline.Run returned nil result")
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

	// Verify progress callbacks were invoked
	if len(progressCalls) == 0 {
		t.Error("No progress callbacks were invoked")
	}

	expectedStages := []string{"extract", "assemble", "filter", "enrich", "model", "output"}
	for _, stage := range expectedStages {
		found := false
		for _, call := range progressCalls {
			if call == stage {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected stage %q in progress calls, not found", stage)
		}
	}
}

func TestPipelineWithTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	createMinimalRepo(t, tmpDir)

	config := PipelineConfig{
		RepoRoot: tmpDir,
		Tier:     Tier1,
		NoLLM:    true,
		Timeout:  10 * time.Second,
	}

	extractors := []Extractor{&mockExtractor{name: "fast_ext"}}
	pipeline := NewPipeline(config, extractors)

	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Pipeline.Run with timeout failed: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestPipelineExtractorErrors(t *testing.T) {
	tmpDir := t.TempDir()

	config := PipelineConfig{
		RepoRoot: tmpDir,
		Tier:     Tier1,
		NoLLM:    true,
	}

	// Use a failing extractor
	extractors := []Extractor{&failingExtractor{name: "fail_ext"}}
	pipeline := NewPipeline(config, extractors)

	result, err := pipeline.Run(context.Background())
	// Should not fail hard, but should have errors in result
	if err != nil {
		t.Fatalf("Pipeline should handle extractor errors gracefully: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil even with failing extractors")
	}
}

func TestBuildReferenceModelFromProfile(t *testing.T) {
	profile := &CodebaseProfile{
		Name: "test-repo",
		Infra: InfraInfo{
			Containers: []ContainerInfo{
				{Name: "web", Type: "service", Source: "Dockerfile"},
				{Name: "db", Type: "database", Image: "postgres:15"},
			},
			Services: []ServiceDep{
				{From: "web", To: "db"},
			},
		},
		ImportGraph: ImportGraph{
			Clusters: []ImportCluster{
				{ID: "api", Packages: []string{"api/handlers", "api/routes"}},
			},
		},
		Dependencies: DependencyInfo{
			NotableDeps: []NotableDep{
				{Name: "redis-client", Signal: "cache"},
			},
		},
		Specs: []SpecArtifact{
			{Kind: "openapi", Path: "api/openapi.yaml"},
		},
	}

	model := BuildReferenceModelFromProfile(profile)

	if model == nil {
		t.Fatal("model is nil")
	}
	if model.System.Name != "test-repo" {
		t.Errorf("System.Name = %q, want %q", model.System.Name, "test-repo")
	}
	if model.State != ModelObserved {
		t.Errorf("State = %q, want %q", model.State, ModelObserved)
	}

	// Should have containers from infra
	if len(model.Containers) < 2 {
		t.Errorf("expected at least 2 containers, got %d", len(model.Containers))
	}

	// Should have cluster containers
	if len(model.Containers) < 3 {
		t.Error("expected at least 3 containers (2 infra + 1 cluster)")
	}

	// Should have relationships from service deps
	if len(model.Relationships) < 1 {
		t.Error("expected at least 1 relationship")
	}

	// Should have external system from notable deps
	if len(model.ExternalSystems) < 1 {
		t.Error("expected at least 1 external system (redis)")
	}

	// Should have actor from spec
	if len(model.Actors) < 1 {
		t.Error("expected at least 1 actor (developer from openapi spec)")
	}
}

func TestPipelineResultToJSON(t *testing.T) {
	result := &PipelineResult{
		Profile: &CodebaseProfile{
			Name: "test-repo",
			Metrics: CodeMetrics{
				TotalFiles: 10,
				TotalLOC:   100,
			},
		},
		Report: &ArchitectureReport{
			Version:   "1.0.0",
			RepoRoot:  "/tmp/test",
			AnalyzedAt: time.Now(),
		},
		ReferenceModel: &ReferenceModel{
			Version: "1.0.0",
			System:  SystemInfo{Name: "test"},
		},
		Duration: 100 * time.Millisecond,
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("ToJSON output is not valid JSON: %v", err)
	}

	// Verify key fields exist
	if _, ok := parsed["profile"]; !ok {
		t.Error("profile field missing from JSON output")
	}
	if _, ok := parsed["report"]; !ok {
		t.Error("report field missing from JSON output")
	}
	if _, ok := parsed["reference_model"]; !ok {
		t.Error("reference_model field missing from JSON output")
	}
}

func TestPipelineResultToText(t *testing.T) {
	result := &PipelineResult{
		Profile: &CodebaseProfile{
			Metrics: CodeMetrics{
				TotalFiles:          42,
				TotalLOC:            4200,
				LanguagesCount:      3,
				ContainersDetected:  2,
				ComponentsDetected:  5,
				ContractsDiscovered: 1,
			},
		},
		Report: &ArchitectureReport{
			RepoRoot: "/tmp/test-repo",
		},
		ReferenceModel: &ReferenceModel{
			System: SystemInfo{Name: "test"},
			Containers: []C4Container{
				{Name: "web", Technology: "docker"},
				{Name: "api", Technology: "go"},
			},
			Relationships: []C4Relationship{
				{From: "web", To: "api", Description: "calls"},
			},
		},
		Duration: 250 * time.Millisecond,
	}

	text := result.ToText()

	if text == "" {
		t.Fatal("ToText returned empty string")
	}
	// Check key info appears in text output
	checks := []string{"42", "4200", "2 containers", "web", "api"}
	for _, check := range checks {
		if !containsSubstring(text, check) {
			t.Errorf("ToText output missing expected substring %q", check)
		}
	}
}

func TestPipelineResultToOutput(t *testing.T) {
	result := &PipelineResult{
		Profile: &CodebaseProfile{Name: "test"},
		Report:  &ArchitectureReport{RepoRoot: "/test", AnalyzedAt: time.Now()},
		Duration: 50 * time.Millisecond,
		Errors: []ExtractorError{
			{Extractor: "ext1", Err: fmt.Errorf("test error")},
		},
	}

	output := result.ToOutput()
	if output == nil {
		t.Fatal("ToOutput returned nil")
	}
	if output.RepoRoot != "/test" {
		t.Errorf("RepoRoot = %q, want %q", output.RepoRoot, "/test")
	}
	if output.DurationMs != 50 {
		t.Errorf("DurationMs = %d, want %d", output.DurationMs, 50)
	}
	if len(output.Errors) != 1 {
		t.Errorf("Errors = %d, want 1", len(output.Errors))
	}
}

func TestPipelineNoLLMOverride(t *testing.T) {
	config := PipelineConfig{
		RepoRoot:         ".",
		Tier:             Tier1,
		AllowExternalLLM: true,
		NoLLM:            true, // should override AllowExternalLLM
	}

	pipeline := NewPipeline(config, nil)
	if pipeline.secFilter.AllowExternalLLM {
		t.Error("NoLLM=true should override AllowExternalLLM=true")
	}
}

func TestBuildReferenceModelEmptyProfile(t *testing.T) {
	profile := &CodebaseProfile{}
	model := BuildReferenceModelFromProfile(profile)

	if model == nil {
		t.Fatal("model should not be nil for empty profile")
	}
	if model.System.Name != "unknown-system" {
		t.Errorf("System.Name for empty profile = %q, want %q", model.System.Name, "unknown-system")
	}
}

func TestBuildReferenceModelWithServiceDeps(t *testing.T) {
	profile := &CodebaseProfile{
		Name: "my-service",
		Infra: InfraInfo{
			Containers: []ContainerInfo{
				{Name: "frontend", Type: "service", Source: "frontend/Dockerfile"},
				{Name: "backend", Type: "service", Source: "backend/Dockerfile"},
				{Name: "cache", Type: "cache", Image: "redis:7"},
			},
			Services: []ServiceDep{
				{From: "frontend", To: "backend"},
				{From: "backend", To: "cache"},
			},
		},
	}

	model := BuildReferenceModelFromProfile(profile)

	// Should have 3 containers
	if len(model.Containers) != 3 {
		t.Errorf("expected 3 containers, got %d", len(model.Containers))
	}

	// Should have 2 relationships (frontend->backend, backend->cache)
	if len(model.Relationships) != 2 {
		t.Errorf("expected 2 relationships, got %d", len(model.Relationships))
	}
}

func TestBuildReferenceModelWithImportClusters(t *testing.T) {
	profile := &CodebaseProfile{
		Name: "clustered-repo",
		ImportGraph: ImportGraph{
			Clusters: []ImportCluster{
				{ID: "auth", Packages: []string{"auth/jwt", "auth/oauth"}, InternalEdges: 5, ExternalEdges: 2},
				{ID: "api", Packages: []string{"api/handlers", "api/middleware"}, InternalEdges: 8, ExternalEdges: 3},
			},
		},
	}

	model := BuildReferenceModelFromProfile(profile)

	// Should have cluster containers
	found := 0
	for _, c := range model.Containers {
		if c.ID == "cluster_auth" || c.ID == "cluster_api" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("expected 2 cluster containers, found %d", found)
	}
}

func TestProgressReporterCallback(t *testing.T) {
	reporter := NewProgressReporter(false)

	var receivedStage PipelineStage
	var receivedMsg string
	cb := reporter.Callback()

	// The callback should not panic
	cb(StageExtract, "testing extractors", nil)

	// Test with timing
	timing := &ExtractorTiming{
		Name:     "test_ext",
		Duration: 100 * time.Millisecond,
		Success:  true,
	}
	cb(StageExtract, "extractor done", timing)
	_ = receivedStage
	_ = receivedMsg
}

func TestProgressReporterVerboseMode(t *testing.T) {
	reporter := NewProgressReporter(true)

	// Should not panic in verbose mode
	cb := reporter.Callback()
	cb(StageAssemble, "assembling profile", &ExtractorTiming{
		Name:     "ext1",
		Duration: 50 * time.Millisecond,
		Success:  true,
	})

	// Summary should not panic
	reporter.Summary()
}

// --- Test helpers ---

type mockExtractor struct {
	name string
}

func (m *mockExtractor) Name() string { return m.name }
func (m *mockExtractor) Extract(ctx context.Context, repoRoot string) (*ProfileFragment, error) {
	return &ProfileFragment{
		FileTree: &FileTreeInfo{
			TotalFiles: 10,
			TotalDirs:  3,
			TopLevel:   []string{"cmd", "internal"},
			ExtCounts:  map[string]int{"go": 8, "md": 2},
		},
		Metrics: &CodeMetrics{
			TotalFiles:     10,
			TotalLOC:       500,
			LanguagesCount: 1,
		},
	}, nil
}

type failingExtractor struct {
	name string
}

func (f *failingExtractor) Name() string { return f.name }
func (f *failingExtractor) Extract(ctx context.Context, repoRoot string) (*ProfileFragment, error) {
	return nil, fmt.Errorf("extractor %s failed intentionally", f.name)
}

func createMinimalRepo(t *testing.T, dir string) {
	t.Helper()
	// Create a minimal go.mod
	goMod := `module test-repo

go 1.22
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}
	// Create a minimal main.go
	mainGo := `package main
func main() {}
`
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to create main.go: %v", err)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && len(sub) > 0 && findSubstring(s, sub)))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
