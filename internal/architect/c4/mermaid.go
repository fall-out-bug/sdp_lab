package c4

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// MermaidOutput holds the rendered Mermaid diagram for a specific level.
type MermaidOutput struct {
	Level       Level
	MermaidCode string
	NodeCount   int
	EdgeCount   int
	Truncated   bool
	ExportData  *ExportData // Populated when nodeCount > nodeThreshold
}

// RenderLevel1 renders a Level 1 (System Context) Mermaid diagram.
func RenderLevel1(model *architect.ReferenceModel, opts RenderOptions) (*MermaidOutput, error) {
	result, err := RenderL1(model, opts)
	if err != nil {
		return nil, err
	}

	output := &MermaidOutput{
		Level:       result.Level,
		MermaidCode: result.MermaidCode,
		NodeCount:   result.NodeCount,
		EdgeCount:   result.EdgeCount,
		Truncated:   result.Truncated,
	}

	// Add export data if node threshold exceeded
	if ShouldExport(result.NodeCount) {
		output.ExportData = ExportLevel1(model)
	}

	return output, nil
}

// RenderLevel2 renders a Level 2 (Container) Mermaid diagram.
func RenderLevel2(model *architect.ReferenceModel, opts RenderOptions) (*MermaidOutput, error) {
	result, err := RenderL2(model, opts)
	if err != nil {
		return nil, err
	}

	output := &MermaidOutput{
		Level:       result.Level,
		MermaidCode: result.MermaidCode,
		NodeCount:   result.NodeCount,
		EdgeCount:   result.EdgeCount,
		Truncated:   result.Truncated,
	}

	// Add export data if node threshold exceeded
	if ShouldExport(result.NodeCount) {
		output.ExportData = ExportLevel2(model)
	}

	return output, nil
}

// RenderLevel3 renders a Level 3 (Component) Mermaid diagram for a specific container.
func RenderLevel3(model *architect.ReferenceModel, containerID string, opts RenderOptions) (*MermaidOutput, error) {
	result, err := RenderL3(model, containerID, opts)
	if err != nil {
		return nil, err
	}

	output := &MermaidOutput{
		Level:       result.Level,
		MermaidCode: result.MermaidCode,
		NodeCount:   result.NodeCount,
		EdgeCount:   result.EdgeCount,
		Truncated:   result.Truncated,
	}

	// Add export data if node threshold exceeded
	if ShouldExport(result.NodeCount) {
		exportData, err := ExportLevel3(model, containerID)
		if err == nil {
			output.ExportData = exportData
		}
	}

	return output, nil
}

// WriteMermaidDiagram writes a Mermaid diagram to a file at the specified path.
// It creates parent directories as needed.
func WriteMermaidDiagram(output *MermaidOutput, targetDir, filename string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	fullPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(fullPath, []byte(output.MermaidCode), 0644); err != nil {
		return fmt.Errorf("failed to write mermaid file: %w", err)
	}

	// If export data exists, write it alongside the diagram
	if output.ExportData != nil {
		exportPath := fullPath + ".json"
		exportJSON, err := MarshalExportData(output.ExportData)
		if err != nil {
			return fmt.Errorf("failed to marshal export data: %w", err)
		}
		if err := os.WriteFile(exportPath, []byte(exportJSON), 0644); err != nil {
			return fmt.Errorf("failed to write export data: %w", err)
		}
	}

	return nil
}

// WriteAllDiagrams renders all levels and writes them to the target directory.
// Output files are named: level-1-system-context.mmd, level-2-containers.mmd,
// level-3-components-<container-id>.mmd
func WriteAllDiagrams(model *architect.ReferenceModel, targetDir string, opts RenderOptions) ([]*MermaidOutput, error) {
	var outputs []*MermaidOutput

	// Render L1
	l1, err := RenderLevel1(model, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to render L1: %w", err)
	}
	outputs = append(outputs, l1)
	if err := WriteMermaidDiagram(l1, targetDir, "level-1-system-context.mmd"); err != nil {
		return nil, err
	}

	// Render L2
	l2, err := RenderLevel2(model, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to render L2: %w", err)
	}
	outputs = append(outputs, l2)
	if err := WriteMermaidDiagram(l2, targetDir, "level-2-containers.mmd"); err != nil {
		return nil, err
	}

	// Render L3 for each container
	for _, container := range model.Containers {
		l3, err := RenderLevel3(model, container.ID, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to render L3 for container %s: %w", container.ID, err)
		}
		outputs = append(outputs, l3)
		filename := fmt.Sprintf("level-3-components-%s.mmd", container.ID)
		if err := WriteMermaidDiagram(l3, targetDir, filename); err != nil {
			return nil, err
		}
	}

	return outputs, nil
}

// GetLayoutSuggestion returns a human-readable layout suggestion for the diagram.
func GetLayoutSuggestion(level Level) string {
	switch level {
	case Level1:
		return "Place actors on the left, system in the center, external systems on the right"
	case Level2:
		return "Arrange databases on the right, services in center, external actors on left"
	case Level3:
		return "Group by layer (handlers at top, domain in middle, data at bottom)"
	default:
		return ""
	}
}
