package architect_test

import (
	"context"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/architect/extract"
)

// mockExtractor is a test double that returns a predefined fragment.
type mockExtractor struct {
	name    string
	fragment *architect.ProfileFragment
	fail    bool
}

func (m mockExtractor) Name() string {
	return m.name
}

func (m mockExtractor) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	if m.fail {
		return nil, &testError{msg: "mock extractor failed"}
	}
	return m.fragment, nil
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestAssembler_MergeFragments(t *testing.T) {
	frag1 := &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: 10,
			TotalDirs:  5,
			MaxDepth:   3,
			ExtCounts:  map[string]int{"go": 8, "mod": 2},
		},
	}

	frag2 := &architect.ProfileFragment{
		Dependencies: []architect.DependencyInfo{
			{
				File:     "go.mod",
				Language: "go",
				DepCount: 5,
			},
		},
		Languages: []architect.LanguageInfo{
			{
				Primary: "go",
				All:     []string{"go", "text"},
			},
		},
	}

	frag3 := &architect.ProfileFragment{
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{
					Name:   "api",
					Source: "Dockerfile",
					Type:   "service",
				},
			},
			Deployment: architect.DeploymentInfo{
				Type: "docker",
			},
		},
	}

	assembler := architect.NewProfileAssembler(architect.Tier2,
		[]architect.Extractor{
			mockExtractor{name: "frag1", fragment: frag1},
			mockExtractor{name: "frag2", fragment: frag2},
			mockExtractor{name: "frag3", fragment: frag3},
		},
	)

	ctx := context.Background()
	profile, err := assembler.Assemble(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	// Check FileTree
	if profile.FileTree.TotalFiles != 10 {
		t.Errorf("Expected TotalFiles=10, got %d", profile.FileTree.TotalFiles)
	}
	if profile.FileTree.TotalDirs != 5 {
		t.Errorf("Expected TotalDirs=5, got %d", profile.FileTree.TotalDirs)
	}

	// Check Dependencies
	if len(profile.Dependencies.Manifests) != 1 {
		t.Errorf("Expected 1 manifest, got %d", len(profile.Dependencies.Manifests))
	}
	if len(profile.Dependencies.Manifests) > 0 {
		manifest := profile.Dependencies.Manifests[0]
		if manifest.Path != "go.mod" {
			t.Errorf("Expected manifest path 'go.mod', got %s", manifest.Path)
		}
		if manifest.Language != "go" {
			t.Errorf("Expected language 'go', got %s", manifest.Language)
		}
	}

	// Check Infra
	if len(profile.Infra.Containers) != 1 {
		t.Errorf("Expected 1 container, got %d", len(profile.Infra.Containers))
	}
	if len(profile.Infra.Containers) > 0 {
		container := profile.Infra.Containers[0]
		if container.Name != "api" {
			t.Errorf("Expected container name 'api', got %s", container.Name)
		}
	}

	if profile.Infra.Deployment.Type != "docker" {
		t.Errorf("Expected deployment type 'docker', got %s", profile.Infra.Deployment.Type)
	}
}

func TestAssembler_TokenEstimation(t *testing.T) {
	frag := &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: 10,
		},
	}

	assembler := architect.NewProfileAssembler(architect.Tier2,
		[]architect.Extractor{mockExtractor{name: "test", fragment: frag}},
	)

	ctx := context.Background()
	profile, err := assembler.Assemble(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	tokens := architect.EstimateTokens(profile)
	if tokens < 0 {
		t.Errorf("Expected non-negative token count, got %d", tokens)
	}

	// A minimal profile should have some tokens
	if tokens == 0 {
		t.Error("Expected non-zero token count for minimal profile")
	}
}

func TestAssembler_ContentHashDeterministic(t *testing.T) {
	frag := &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: 10,
			TotalDirs:  5,
			MaxDepth:   3,
		},
	}

	assembler1 := architect.NewProfileAssembler(architect.Tier2,
		[]architect.Extractor{mockExtractor{name: "test", fragment: frag}},
	)
	ctx := context.Background()
	profile1, err := assembler1.Assemble(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	hash1 := assembler1.ContentHash(profile1)

	assembler2 := architect.NewProfileAssembler(architect.Tier2,
		[]architect.Extractor{mockExtractor{name: "test", fragment: frag}},
	)
	profile2, err := assembler2.Assemble(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}
	hash2 := assembler2.ContentHash(profile2)

	if hash1 != hash2 {
		t.Errorf("Content hash should be deterministic: got %s and %s", hash1, hash2)
	}
}

func TestAssembler_TierFiltering(t *testing.T) {
	frag := &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: 10,
		},
		ImportGraph: &architect.ImportGraph{
			Nodes: 5,
			Edges: 10,
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Source: "Dockerfile", Type: "service"},
			},
		},
	}

	// Test Tier1 (summary only)
	assembler1 := architect.NewProfileAssembler(architect.Tier1,
		[]architect.Extractor{mockExtractor{name: "test", fragment: frag}},
	)
	ctx := context.Background()
	profile1, err := assembler1.Assemble(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	if profile1.ImportGraph.Nodes != 5 {
		t.Errorf("Tier1 should keep ImportGraph counts, got Nodes=%d", profile1.ImportGraph.Nodes)
	}
	if len(profile1.ImportGraph.Clusters) != 0 {
		t.Errorf("Tier1 should strip ImportGraph clusters, got %d", len(profile1.ImportGraph.Clusters))
	}

	// Test Tier2 (full detail)
	assembler2 := architect.NewProfileAssembler(architect.Tier2,
		[]architect.Extractor{mockExtractor{name: "test", fragment: frag}},
	)
	profile2, err := assembler2.Assemble(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	if profile2.ImportGraph.Nodes != 5 {
		t.Errorf("Tier2 should preserve ImportGraph details, got Nodes=%d", profile2.ImportGraph.Nodes)
	}
}

func TestAssembler_ErrorCollection(t *testing.T) {
	frag := &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: 10,
		},
	}

	assembler := architect.NewProfileAssembler(architect.Tier2,
		[]architect.Extractor{
			mockExtractor{name: "good", fragment: frag},
			mockExtractor{name: "bad", fragment: nil, fail: true},
			mockExtractor{name: "also-good", fragment: frag},
		},
	)

	ctx := context.Background()
	profile, err := assembler.Assemble(ctx, "/tmp/test")

	if err != nil {
		t.Fatalf("Assemble should not fail on non-fatal errors: %v", err)
	}

	if profile == nil {
		t.Fatal("Expected non-nil profile")
	}

	errors := assembler.Errors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	if len(errors) > 0 && errors[0].Extractor != "bad" {
		t.Errorf("Expected error from 'bad' extractor, got %s", errors[0].Extractor)
	}
}

func TestAssembler_TierSummary(t *testing.T) {
	frag := &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: 10,
			TotalDirs:  5,
			MaxDepth:   3,
			ExtCounts:  map[string]int{"go": 8, "mod": 2},
			Patterns:   []string{"service", "repository"},
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Source: "Dockerfile", Type: "service"},
			},
			Deployment: architect.DeploymentInfo{
				Type: "docker",
			},
		},
		Specs: []architect.SpecArtifact{
			{Kind: "openapi", Path: "api/openapi.yaml"},
		},
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		Metrics: &architect.CodeMetrics{
			TotalLOC:  1000,
			TestRatio: 0.2,
		},
	}

	assembler := architect.NewProfileAssembler(architect.Tier1,
		[]architect.Extractor{mockExtractor{name: "test", fragment: frag}},
	)
	ctx := context.Background()
	profile, err := assembler.Assemble(ctx, "/tmp/test")
	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	summary := assembler.TierSummary(profile)

	// Check that summary contains key information
	if len(summary) == 0 {
		t.Error("Expected non-empty summary")
	}

	// Should contain section headers
	if !contains(summary, "Codebase Profile Summary") {
		t.Error("Summary should contain title")
	}

	// Should mention files
	if !contains(summary, "10 files") {
		t.Error("Summary should mention file count")
	}

	// Should mention containers
	if !contains(summary, "1 detected") {
		t.Error("Summary should mention containers")
	}
}

func TestDefaultExtractors(t *testing.T) {
	extractors := extract.DefaultExtractors()

	if len(extractors) != 11 {
		t.Errorf("Expected 11 extractors, got %d", len(extractors))
	}

	names := make(map[string]bool)
	for _, ext := range extractors {
		names[ext.Name()] = true
	}

	expectedNames := []string{
		"filetree",
		"deps",
		"specs",
		"generated",
		"infra",
		"git_history",
		"go",
		"python",
		"java",
		"typescript",
		"sql",
	}

	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected extractor '%s' not found", name)
		}
	}
}

func TestAssembler_Assemble(t *testing.T) {
	frag := &architect.ProfileFragment{
		FileTree: &architect.FileTreeInfo{
			TotalFiles: 10,
			TotalDirs:  5,
			MaxDepth:   3,
			ExtCounts:  map[string]int{"go": 10},
		},
		Languages: []architect.LanguageInfo{
			{Primary: "go", All: []string{"go"}},
		},
		Infra: &architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Source: "Dockerfile", Type: "service"},
			},
		},
	}

	assembler := architect.NewProfileAssembler(architect.Tier2,
		[]architect.Extractor{mockExtractor{name: "test", fragment: frag}},
	)

	ctx := context.Background()
	profile, err := assembler.Assemble(ctx, "/tmp/test")

	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	if profile == nil {
		t.Fatal("Expected non-nil profile")
	}

	// Check that metrics were computed
	if profile.Metrics.TotalFiles != 10 {
		t.Errorf("Expected TotalFiles=10, got %d", profile.Metrics.TotalFiles)
	}

	if profile.Metrics.LanguagesCount != 1 {
		t.Errorf("Expected LanguagesCount=1, got %d", profile.Metrics.LanguagesCount)
	}

	if profile.Metrics.ContainersDetected != 1 {
		t.Errorf("Expected ContainersDetected=1, got %d", profile.Metrics.ContainersDetected)
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
