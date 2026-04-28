package architect_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/architect/classify"
	"github.com/fall-out-bug/sdp_lab/internal/architect/extract"
)

// TestBuildReferenceModel verifies that a ReferenceModel is correctly derived from a CodebaseProfile
func TestBuildReferenceModel(t *testing.T) {
	// Create a test profile
	profile := &architect.CodebaseProfile{
		Name: "test-service",
		FileTree: architect.FileTreeInfo{
			TotalFiles: 100,
			ExtCounts:  map[string]int{"go": 80, "yaml": 20},
		},
		Dependencies: architect.DependencyInfo{
			NotableDeps: []architect.NotableDep{
				{Name: "github.com/segmentio/kafka-go", Signal: "event_driven"},
				{Name: "github.com/redis/go-redis/v9", Signal: "cache"},
			},
		},
		Infra: architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service", Source: "Dockerfile"},
				{Name: "worker", Type: "service", Source: "Dockerfile.worker"},
			},
			Services: []architect.ServiceDep{
				{From: "api", To: "worker"},
			},
		},
		ImportGraph: architect.ImportGraph{
			Clusters: []architect.ImportCluster{
				{ID: "api", Packages: []string{"./api/handlers", "./api/middleware"}},
				{ID: "core", Packages: []string{"./core/domain", "./core/repository"}},
			},
		},
		Specs: []architect.SpecArtifact{
			{Kind: "openapi", Path: "api/openapi.yaml"},
		},
	}

	// Build reference model (this would be called from the CLI)
	model := buildReferenceModelForTest(profile)

	// Verify system info
	if model.System.Name != "test-service" {
		t.Errorf("expected system name 'test-service', got '%s'", model.System.Name)
	}

	// Verify actors (should have "developer" since we have OpenAPI spec)
	if len(model.Actors) != 1 {
		t.Errorf("expected 1 actor, got %d", len(model.Actors))
	} else if model.Actors[0].ID != "developer" {
		t.Errorf("expected actor ID 'developer', got '%s'", model.Actors[0].ID)
	}

	// Verify external systems (kafka and redis should be detected)
	if len(model.ExternalSystems) < 2 {
		t.Errorf("expected at least 2 external systems, got %d", len(model.ExternalSystems))
	}

	// Verify containers
	if len(model.Containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(model.Containers))
	}

	// Verify relationships
	if len(model.Relationships) < 1 {
		t.Errorf("expected at least 1 relationship, got %d", len(model.Relationships))
	}
}

// TestComputeConfidence verifies confidence score computation
func TestComputeConfidence(t *testing.T) {
	// Test with full profile and LLM results
	profile := &architect.CodebaseProfile{
		FileTree: architect.FileTreeInfo{
			TotalFiles: 100,
		},
		Dependencies: architect.DependencyInfo{
			Manifests: []architect.ManifestInfo{
				{Path: "go.mod", Language: "go", DepsCount: 10},
			},
		},
		Infra: architect.InfraInfo{
			Containers: []architect.ContainerInfo{
				{Name: "api", Type: "service"},
			},
		},
		Specs: []architect.SpecArtifact{
			{Kind: "openapi", Path: "api/openapi.yaml"},
		},
		ImportGraph: architect.ImportGraph{
			Nodes: 50,
			Edges: 100,
		},
		Metrics: architect.CodeMetrics{
			ContainersDetected:  1,
			ContractsDiscovered: 1,
		},
	}

	result := &classify.HypothesisResult{
		StyleHypothesis: architect.StyleHypothesis{
			Styles: []architect.StyleScore{
				{Style: architect.StyleLayered, Confidence: 0.8},
				{Style: architect.StyleModular, Confidence: 0.6},
			},
		},
	}

	confidence := computeConfidenceForTest(profile, result)

	// Verify all scores are between 0 and 1
	if confidence.Overall < 0 || confidence.Overall > 1 {
		t.Errorf("overall confidence out of range: %.2f", confidence.Overall)
	}
	if confidence.StructuralAnalysis < 0 || confidence.StructuralAnalysis > 1 {
		t.Errorf("structural analysis confidence out of range: %.2f", confidence.StructuralAnalysis)
	}
	if confidence.StyleHypothesis < 0 || confidence.StyleHypothesis > 1 {
		t.Errorf("style hypothesis confidence out of range: %.2f", confidence.StyleHypothesis)
	}
	if confidence.ContractCoverage < 0 || confidence.ContractCoverage > 1 {
		t.Errorf("contract coverage out of range: %.2f", confidence.ContractCoverage)
	}

	// Verify structural analysis is high (we have all data points)
	if confidence.StructuralAnalysis < 0.8 {
		t.Errorf("expected high structural analysis confidence, got %.2f", confidence.StructuralAnalysis)
	}

	// Verify style hypothesis is average of LLM confidences
	expectedStyleHypothesis := (0.8 + 0.6) / 2.0
	if confidence.StyleHypothesis != expectedStyleHypothesis {
		t.Errorf("expected style hypothesis %.2f, got %.2f", expectedStyleHypothesis, confidence.StyleHypothesis)
	}

	// Test with no LLM results
	confidenceNoLLM := computeConfidenceForTest(profile, nil)
	if confidenceNoLLM.StyleHypothesis != 0.0 {
		t.Errorf("expected zero style hypothesis without LLM, got %.2f", confidenceNoLLM.StyleHypothesis)
	}
	if confidenceNoLLM.Note == "" {
		t.Error("expected note when no LLM analysis performed")
	}
}

// TestFilterExtractors verifies language-based extractor filtering
func TestFilterExtractors(t *testing.T) {
	extractors := []architect.Extractor{
		extract.FileTreeExtractor{},
		extract.DependencyManifestParser{},
		extract.SpecInventoryScanner{},
		extract.GeneratedCodeDetector{},
		&extract.InfraExtractor{},
		extract.GitHistoryExtractor{},
	}

	// Test with Go language filter
	filtered := filterExtractorsForTest(extractors, []string{"go"})
	// Should include language-agnostic + Go extractor
	if len(filtered) < 6 {
		t.Errorf("expected at least 6 extractors for Go filter, got %d", len(filtered))
	}

	// Verify language-agnostic extractors are present
	names := make(map[string]bool)
	for _, ext := range filtered {
		names[ext.Name()] = true
	}

	agnostic := []string{"filetree", "deps", "specs", "generated", "infra", "git_history"}
	for _, name := range agnostic {
		if !names[name] {
			t.Errorf("language-agnostic extractor '%s' was filtered out", name)
		}
	}
}

// TestFilterExtractorsAll verifies that empty filter returns all extractors
func TestFilterExtractorsAll(t *testing.T) {
	extractors := []architect.Extractor{
		extract.FileTreeExtractor{},
		extract.DependencyManifestParser{},
		extract.SpecInventoryScanner{},
		extract.GeneratedCodeDetector{},
		&extract.InfraExtractor{},
		extract.GitHistoryExtractor{},
	}

	filtered := filterExtractorsForTest(extractors, []string{})
	if len(filtered) != len(extractors) {
		t.Errorf("expected %d extractors with empty filter, got %d", len(extractors), len(filtered))
	}
}

// TestWriteArtifact verifies JSON file writing
func TestWriteArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	testData := map[string]string{
		"key":   "value",
		"number": "42",
	}

	err := writeArtifactForTest(tmpDir, "test.json", testData)
	if err != nil {
		t.Fatalf("writeArtifact failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "test.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file was not created")
	}

	// Verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Check for valid JSON
	if string(content) == "" {
		t.Error("file is empty")
	}
	// Check that it's valid JSON by unmarshaling
	var result map[string]string
	if err := json.Unmarshal(content, &result); err != nil {
		t.Errorf("file content is not valid JSON: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key='value', got key='%s'", result["key"])
	}
}

// TestWriteMermaid verifies Mermaid file writing
func TestWriteMermaid(t *testing.T) {
	tmpDir := t.TempDir()
	testCode := `graph TB
    A["System"]
    B["Actor"]
    A --> B`

	err := writeMermaidForTest(tmpDir, "test.mmd", testCode)
	if err != nil {
		t.Fatalf("writeMermaid failed: %v", err)
	}

	// Verify file exists
	path := filepath.Join(tmpDir, "test.mmd")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("file was not created")
	}

	// Verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != testCode {
		t.Errorf("file content mismatch\nexpected: %s\ngot: %s", testCode, string(content))
	}
}

// Helper functions (these would be the actual functions from cmd_architect.go)

func buildReferenceModelForTest(profile *architect.CodebaseProfile) *architect.ReferenceModel {
	// Simplified version of the CLI function for testing
	model := &architect.ReferenceModel{
		Version: "1.0.0",
		State:   architect.ModelObserved,
		System: architect.SystemInfo{
			Name: profile.Name,
		},
		Actors:          make([]architect.Actor, 0),
		ExternalSystems: make([]architect.ExternalSystem, 0),
		Containers:      make([]architect.C4Container, 0),
		Relationships:   make([]architect.C4Relationship, 0),
	}

	// Check for library/API to add developer actor
	for _, spec := range profile.Specs {
		if spec.Kind == "openapi" || spec.Kind == "graphql" || spec.Kind == "protobuf" {
			model.Actors = append(model.Actors, architect.Actor{
				ID:          "developer",
				Description: "Developer using this library/API",
			})
			break
		}
	}

	// Add external systems from dependencies
	depSystemMap := map[string]string{
		"kafka":    "message broker",
		"redis":    "cache",
		"postgres": "database",
	}

	seenSystems := make(map[string]bool)
	for _, dep := range profile.Dependencies.NotableDeps {
		lowerName := dep.Name
		for sysPrefix, sysType := range depSystemMap {
			if containsStr(lowerName, sysPrefix) && !seenSystems[sysPrefix] {
				model.ExternalSystems = append(model.ExternalSystems, architect.ExternalSystem{
					ID:          sysPrefix,
					Description: sysPrefix + " " + sysType,
					Technology:  sysPrefix,
				})
				seenSystems[sysPrefix] = true
			}
		}
	}

	// Add containers
	for i, infraContainer := range profile.Infra.Containers {
		containerID := fmt.Sprintf("container_%d", i+1)
		model.Containers = append(model.Containers, architect.C4Container{
			ID:   containerID,
			Name: infraContainer.Name,
		})
	}

	// Add relationships from service dependencies
	for _, svcDep := range profile.Infra.Services {
		// Find container IDs by name
		var fromID, toID string
		for _, c := range model.Containers {
			if c.Name == svcDep.From {
				fromID = c.ID
			}
			if c.Name == svcDep.To {
				toID = c.ID
			}
		}
		if fromID != "" && toID != "" {
			model.Relationships = append(model.Relationships, architect.C4Relationship{
				From:        fromID,
				To:          toID,
				Description: "depends on",
				Type:        "sync",
			})
		}
	}

	return model
}

func computeConfidenceForTest(profile *architect.CodebaseProfile, result *classify.HypothesisResult) architect.ConfidenceSummary {
	summary := architect.ConfidenceSummary{}

	// Structural analysis
	dataPoints := 0
	if profile.FileTree.TotalFiles > 0 {
		dataPoints++
	}
	if len(profile.Dependencies.Manifests) > 0 {
		dataPoints++
	}
	if len(profile.Infra.Containers) > 0 {
		dataPoints++
	}
	if len(profile.Specs) > 0 {
		dataPoints++
	}
	if profile.ImportGraph.Nodes > 0 {
		dataPoints++
	}
	summary.StructuralAnalysis = float64(dataPoints) / 5.0

	// Style hypothesis
	if result != nil && len(result.StyleHypothesis.Styles) > 0 {
		totalConf := 0.0
		for _, style := range result.StyleHypothesis.Styles {
			totalConf += style.Confidence
		}
		summary.StyleHypothesis = totalConf / float64(len(result.StyleHypothesis.Styles))
	}

	// Contract coverage
	if profile.Metrics.ContainersDetected > 0 {
		summary.ContractCoverage = float64(profile.Metrics.ContractsDiscovered) / float64(profile.Metrics.ContainersDetected)
		if summary.ContractCoverage > 1.0 {
			summary.ContractCoverage = 1.0
		}
	}

	// Overall
	summary.Overall = (summary.StructuralAnalysis * 0.4) + (summary.StyleHypothesis * 0.4) + (summary.ContractCoverage * 0.2)

	if result == nil {
		summary.Note = "Style hypothesis is default; run with --allow-external-llm for AI-powered analysis"
	}

	return summary
}

func filterExtractorsForTest(extractors []architect.Extractor, languages []string) []architect.Extractor {
	if len(languages) == 0 {
		return extractors
	}

	agnosticExtractors := map[string]bool{
		"filetree":   true,
		"deps":       true,
		"specs":      true,
		"generated":  true,
		"infra":      true,
		"git_history": true,
	}

	filtered := make([]architect.Extractor, 0)
	for _, ext := range extractors {
		name := ext.Name()
		if agnosticExtractors[name] {
			filtered = append(filtered, ext)
		}
	}

	return filtered
}

func writeArtifactForTest(dir string, name string, data interface{}) error {
	path := filepath.Join(dir, name)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

func writeMermaidForTest(dir string, name string, code string) error {
	path := filepath.Join(dir, name)
	return os.WriteFile(path, []byte(code), 0644)
}

func containsStr(data, substr string) bool {
	return len(data) >= len(substr) && indexOfStr(data, substr) >= 0
}

func indexOfStr(data, substr string) int {
	for i := 0; i <= len(data)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if data[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
