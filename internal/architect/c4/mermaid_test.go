package c4

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/architect"
)

func sampleModelForMermaid() *architect.ReferenceModel {
	return &architect.ReferenceModel{
		Version: "1.0.0",
		System: architect.SystemInfo{
			Name: "TestSystem",
		},
		Actors: []architect.Actor{
			{ID: "user", Description: "End User"},
		},
		ExternalSystems: []architect.ExternalSystem{
			{ID: "stripe", Description: "Stripe API"},
		},
		Containers: []architect.C4Container{
			{
				ID:         "api",
				Name:       "API Service",
				Technology: "Go",
				Components: []architect.C4Component{
					{ID: "handlers", Description: "HTTP Handlers"},
				},
			},
		},
		Relationships: []architect.C4Relationship{
			{From: "user", To: "system", Description: "uses", Type: "sync"},
		},
	}
}

func TestRenderLevel1(t *testing.T) {
	model := sampleModelForMermaid()
	opts := RenderOptions{Direction: "TB"}

	output, err := RenderLevel1(model, opts)
	if err != nil {
		t.Fatalf("RenderLevel1 failed: %v", err)
	}

	if output.Level != Level1 {
		t.Errorf("expected Level1, got %v", output.Level)
	}

	if output.MermaidCode == "" {
		t.Error("expected non-empty mermaid code")
	}

	if output.NodeCount == 0 {
		t.Error("expected non-zero node count")
	}

	// Should not have export data for small diagrams
	if output.ExportData != nil {
		t.Error("expected nil ExportData for small diagram")
	}
}

func TestRenderLevel2(t *testing.T) {
	model := sampleModelForMermaid()
	opts := RenderOptions{Direction: "LR"}

	output, err := RenderLevel2(model, opts)
	if err != nil {
		t.Fatalf("RenderLevel2 failed: %v", err)
	}

	if output.Level != Level2 {
		t.Errorf("expected Level2, got %v", output.Level)
	}

	if !strings.Contains(output.MermaidCode, "graph LR") {
		t.Error("expected 'graph LR' directive")
	}
}

func TestRenderLevel3(t *testing.T) {
	model := sampleModelForMermaid()
	opts := RenderOptions{}

	output, err := RenderLevel3(model, "api", opts)
	if err != nil {
		t.Fatalf("RenderLevel3 failed: %v", err)
	}

	if output.Level != Level3 {
		t.Errorf("expected Level3, got %v", output.Level)
	}

	if !strings.Contains(output.MermaidCode, "subgraph") {
		t.Error("expected subgraph in L3 diagram")
	}
}

func TestRenderLevel3_ContainerNotFound(t *testing.T) {
	model := sampleModelForMermaid()
	opts := RenderOptions{}

	_, err := RenderLevel3(model, "nonexistent", opts)
	if err == nil {
		t.Error("expected error for nonexistent container")
	}
}

func TestWriteMermaidDiagram(t *testing.T) {
	output := &MermaidOutput{
		Level:       Level1,
		MermaidCode: "graph TB\n    A[B]",
		NodeCount:   1,
		EdgeCount:   0,
	}

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "output")
	filename := "test.mmd"

	err := WriteMermaidDiagram(output, targetDir, filename)
	if err != nil {
		t.Fatalf("WriteMermaidDiagram failed: %v", err)
	}

	// Check file was created
	fullPath := filepath.Join(targetDir, filename)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != output.MermaidCode {
		t.Error("file content doesn't match expected mermaid code")
	}

	// Check directory was created
	info, err := os.Stat(targetDir)
	if err != nil {
		t.Fatalf("failed to stat target dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("target path is not a directory")
	}
}

func TestWriteMermaidDiagram_WithExportData(t *testing.T) {
	output := &MermaidOutput{
		Level:       Level1,
		MermaidCode: "graph TB\n    A[B]",
		NodeCount:   20, // Above threshold
		ExportData: &ExportData{
			Level:  "L1",
			System: "test",
			Nodes:  []ExportNode{{ID: "A", Type: "System", Label: "Test"}},
		},
	}

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "output")
	filename := "test.mmd"

	err := WriteMermaidDiagram(output, targetDir, filename)
	if err != nil {
		t.Fatalf("WriteMermaidDiagram failed: %v", err)
	}

	// Check both .mmd and .mmd.json files exist
	mmdPath := filepath.Join(targetDir, filename)
	jsonPath := mmdPath + ".json"

	if _, err := os.Stat(mmdPath); os.IsNotExist(err) {
		t.Error("mermaid file was not created")
	}

	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("export data JSON file was not created")
	}

	// Check JSON content
	jsonContent, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	if !strings.Contains(string(jsonContent), `"level": "L1"`) {
		t.Error("JSON export data doesn't contain expected fields")
	}
}

func TestWriteAllDiagrams(t *testing.T) {
	model := sampleModelForMermaid()

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "output")

	outputs, err := WriteAllDiagrams(model, targetDir, RenderOptions{})
	if err != nil {
		t.Fatalf("WriteAllDiagrams failed: %v", err)
	}

	// Should have L1 + L2 + 1 L3
	expectedCount := 1 + 1 + len(model.Containers)
	if len(outputs) != expectedCount {
		t.Errorf("expected %d outputs, got %d", expectedCount, len(outputs))
	}

	// Check files were created
	expectedFiles := []string{
		"level-1-system-context.mmd",
		"level-2-containers.mmd",
		"level-3-components-api.mmd",
	}

	for _, filename := range expectedFiles {
		fullPath := filepath.Join(targetDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file %q was not created", filename)
		}
	}
}

func TestWriteAllDiagrams_LargeModel(t *testing.T) {
	// Create a model with many containers to trigger export data
	containers := make([]architect.C4Container, 20)
	for i := 0; i < 20; i++ {
		containers[i] = architect.C4Container{
			ID:   string(rune('a' + i)),
			Name: string(rune('A' + i)),
		}
	}

	model := &architect.ReferenceModel{
		System:     architect.SystemInfo{Name: "Large"},
		Containers: containers,
	}

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "output")

	_, err := WriteAllDiagrams(model, targetDir, RenderOptions{})
	if err != nil {
		t.Fatalf("WriteAllDiagrams failed: %v", err)
	}

	// For large models, some diagrams should have export data
	foundExport := false
	_ = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			foundExport = true
		}
		return nil
	})

	if !foundExport {
		t.Error("expected at least one export data JSON file for large model")
	}
}

func TestGetLayoutSuggestion(t *testing.T) {
	tests := []struct {
		level    Level
		contains string
	}{
		{Level1, "actors"},
		{Level2, "databases"},
		{Level3, "layer"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			suggestion := GetLayoutSuggestion(tt.level)
			if suggestion == "" {
				t.Error("expected non-empty suggestion")
			}
			if !strings.Contains(strings.ToLower(suggestion), tt.contains) {
				t.Errorf("expected suggestion to contain %q, got %q", tt.contains, suggestion)
			}
		})
	}
}

func TestRenderLevel1_WithExportData(t *testing.T) {
	// Create a model with many actors/externals to exceed threshold
	actors := make([]architect.Actor, 10)
	for i := 0; i < 10; i++ {
		actors[i] = architect.Actor{
			ID:          string(rune('a' + i)),
			Description: string(rune('A' + i)),
		}
	}

	externals := make([]architect.ExternalSystem, 10)
	for i := 0; i < 10; i++ {
		externals[i] = architect.ExternalSystem{
			ID:          string(rune('0' + i)),
			Description: string(rune('0' + i)),
		}
	}

	model := &architect.ReferenceModel{
		System:          architect.SystemInfo{Name: "Large"},
		Actors:          actors,
		ExternalSystems: externals,
	}

	output, err := RenderLevel1(model, RenderOptions{})
	if err != nil {
		t.Fatalf("RenderLevel1 failed: %v", err)
	}

	// With >15 nodes, should have export data
	if output.ExportData == nil {
		t.Error("expected ExportData for large diagram")
	}

	if output.ExportData.Level != "L1" {
		t.Errorf("expected export level L1, got %q", output.ExportData.Level)
	}
}

func TestMermaidOutput_Truncated(t *testing.T) {
	model := sampleModelForMermaid()
	opts := RenderOptions{MaxNodes: 1}

	output, err := RenderLevel1(model, opts)
	if err != nil {
		t.Fatalf("RenderLevel1 failed: %v", err)
	}

	if !output.Truncated {
		t.Error("expected Truncated to be true with MaxNodes=1")
	}
}
