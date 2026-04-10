package c4

import (
	"encoding/json"
	"fmt"

	"sdp_dev/internal/architect"
)

// nodeThreshold controls when the system emits a graph data fallback file
// alongside the Mermaid diagram.  Per spec Section 5.4, diagrams with >15
// nodes produce suboptimal auto-layout.
const nodeThreshold = 15

// ExportData holds structured graph data for external editors (Excalidraw,
// draw.io, etc.).
type ExportData struct {
	Level           string        `json:"level"`
	System          string        `json:"system"`
	Nodes           []ExportNode  `json:"nodes"`
	Edges           []ExportEdge  `json:"edges"`
	LayoutSuggestion string       `json:"layout_suggestion"`
}

// ExportNode is a single node in the exported graph.
type ExportNode struct {
	ID    string   `json:"id"`
	Type  string   `json:"type"`
	Label string   `json:"label"`
	Tech  []string `json:"tech,omitempty"`
}

// ExportEdge is a single edge in the exported graph.
type ExportEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Label  string `json:"label,omitempty"`
}

// ShouldExport returns true when a diagram exceeds the node threshold and
// should produce a graph data file alongside Mermaid.
func ShouldExport(nodeCount int) bool {
	return nodeCount > nodeThreshold
}

// ExportLevel1 creates structured graph data for an L1 diagram.
func ExportLevel1(model *architect.ReferenceModel) *ExportData {
	data := &ExportData{
		Level:  "L1",
		System: model.System.Name,
		LayoutSuggestion: "Place actors on the left, system in the center, external systems on the right",
	}

	// System node.
	data.Nodes = append(data.Nodes, ExportNode{
		ID:    "system",
		Type:  "System",
		Label: model.System.Name,
	})

	// Actor nodes.
	for _, a := range model.Actors {
		data.Nodes = append(data.Nodes, ExportNode{
			ID:    a.ID,
			Type:  "Actor",
			Label: a.Description,
		})
	}

	// External system nodes.
	for _, ext := range model.ExternalSystems {
		tech := []string{}
		if ext.Technology != "" {
			tech = append(tech, ext.Technology)
		}
		data.Nodes = append(data.Nodes, ExportNode{
			ID:    ext.ID,
			Type:  "ExternalSystem",
			Label: ext.Description,
			Tech:  tech,
		})
	}

	// Edges.
	for _, r := range model.Relationships {
		data.Edges = append(data.Edges, ExportEdge{
			Source: r.From,
			Target: r.To,
			Type:   r.Type,
			Label:  r.Description,
		})
	}

	return data
}

// ExportLevel2 creates structured graph data for an L2 diagram.
func ExportLevel2(model *architect.ReferenceModel) *ExportData {
	data := &ExportData{
		Level:  "L2",
		System: model.System.Name,
		LayoutSuggestion: "Arrange databases on the right, services in center, external actors on left",
	}

	// Actor nodes.
	for _, a := range model.Actors {
		data.Nodes = append(data.Nodes, ExportNode{
			ID:    a.ID,
			Type:  "Actor",
			Label: a.Description,
		})
	}

	// External system nodes.
	for _, ext := range model.ExternalSystems {
		tech := []string{}
		if ext.Technology != "" {
			tech = append(tech, ext.Technology)
		}
		data.Nodes = append(data.Nodes, ExportNode{
			ID:    ext.ID,
			Type:  "ExternalSystem",
			Label: ext.Description,
			Tech:  tech,
		})
	}

	// Container nodes.
	for _, c := range model.Containers {
		tech := []string{}
		if c.Technology != "" {
			tech = append(tech, c.Technology)
		}
		nodeType := "Container"
		if isDatabaseContainer(c) {
			nodeType = "Database"
		}
		data.Nodes = append(data.Nodes, ExportNode{
			ID:    c.ID,
			Type:  nodeType,
			Label: c.Name,
			Tech:  tech,
		})
	}

	// Edges.
	for _, r := range model.Relationships {
		data.Edges = append(data.Edges, ExportEdge{
			Source: r.From,
			Target: r.To,
			Type:   r.Type,
			Label:  r.Description,
		})
	}

	return data
}

// ExportLevel3 creates structured graph data for an L3 diagram.
func ExportLevel3(model *architect.ReferenceModel, containerID string) (*ExportData, error) {
	var container *architect.C4Container
	for i := range model.Containers {
		if model.Containers[i].ID == containerID {
			container = &model.Containers[i]
			break
		}
	}
	if container == nil {
		return nil, fmt.Errorf("container %q not found", containerID)
	}

	data := &ExportData{
		Level:  "L3",
		System: model.System.Name + " / " + container.Name,
		LayoutSuggestion: "Group by layer (handlers at top, domain in middle, data at bottom)",
	}

	// Component nodes.
	for _, comp := range container.Components {
		data.Nodes = append(data.Nodes, ExportNode{
			ID:    comp.ID,
			Type:  "Component",
			Label: comp.Description,
		})
	}

	// Internal edges.
	for _, r := range model.Relationships {
		fromInContainer := false
		toInContainer := false
		for _, comp := range container.Components {
			if r.From == comp.ID {
				fromInContainer = true
			}
			if r.To == comp.ID {
				toInContainer = true
			}
		}
		if fromInContainer && toInContainer {
			data.Edges = append(data.Edges, ExportEdge{
				Source: r.From,
				Target: r.To,
				Type:   r.Type,
				Label:  r.Description,
			})
		}
	}

	return data, nil
}

// MarshalExportData serializes export data to JSON.
func MarshalExportData(data *ExportData) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal export data: %w", err)
	}
	return string(b), nil
}

// countModelNodesL1 counts the number of L1-relevant nodes.
func countModelNodesL1(model *architect.ReferenceModel) int {
	count := 1 // system node
	count += len(model.Actors)
	count += len(model.ExternalSystems)
	return count
}

// countModelNodesL2 counts the number of L2-relevant nodes.
func countModelNodesL2(model *architect.ReferenceModel) int {
	count := len(model.Actors)
	count += len(model.ExternalSystems)
	count += len(model.Containers)
	return count
}

// countModelNodesL3 counts the number of L3-relevant nodes for a container.
func countModelNodesL3(container *architect.C4Container) int {
	return len(container.Components)
}
