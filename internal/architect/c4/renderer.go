package c4

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// Level controls the C4 diagram depth.
type Level int

const (
	Level1 Level = iota + 1 // System Context
	Level2                   // Container
	Level3                   // Component
)

func (l Level) String() string {
	switch l {
	case Level1:
		return "L1"
	case Level2:
		return "L2"
	case Level3:
		return "L3"
	default:
		return "Unknown"
	}
}

// RenderOptions controls diagram rendering behavior.
type RenderOptions struct {
	Direction string // "TB" (top-bottom) or "LR" (left-right), default "TB"
	MaxNodes  int    // 0 = unlimited; if >0, truncate to this many nodes
	Theme     string // "default", "dark", "forest" — applied as comment hint
}

// DiagramResult holds the rendered diagram and metadata.
type DiagramResult struct {
	Level       Level
	MermaidCode string
	NodeCount   int
	EdgeCount   int
	Truncated   bool // true if MaxNodes caused truncation
}

// SECURITY: Per spec Section 4, the rendered Mermaid diagram MUST be placed inside
// a sandboxed <iframe sandbox="allow-scripts"> (no allow-same-origin) when
// displayed in a browser. This function produces raw Mermaid code — the caller
// is responsible for iframe sandboxing at the UI layer.
//
// RenderL1 renders a System Context diagram (Level 1).
func RenderL1(model *architect.ReferenceModel, opts RenderOptions) (*DiagramResult, error) {
	if opts.Direction == "" {
		opts.Direction = "TB"
	}

	var builder strings.Builder

	// Collect all node declarations
	var nodes []string
	var edges []string

	// Add the main system node
	systemID := sanitizeID(model.System.Name)
	nodes = append(nodes, fmt.Sprintf("    %s[\"%s\\nSystem\"]", systemID, model.System.Name))

	// Add actor nodes
	actorIDs := make(map[string]string)
	for _, actor := range model.Actors {
		actorID := sanitizeID("actor_" + actor.ID)
		actorIDs[actor.ID] = actorID
		label := actor.Description
		if label == "" {
			label = actor.ID
		}
		nodes = append(nodes, fmt.Sprintf("    %s[\"%s\"]", actorID, label))
	}

	// Add external system nodes
	extSystemIDs := make(map[string]string)
	for _, ext := range model.ExternalSystems {
		extID := sanitizeID("ext_" + ext.ID)
		extSystemIDs[ext.ID] = extID
		label := ext.Description
		if label == "" {
			label = ext.ID
		}
		if ext.Technology != "" {
			label += fmt.Sprintf("\\n%s", ext.Technology)
		}
		nodes = append(nodes, fmt.Sprintf("    %s[\"%s\\nExternal System\"]", extID, label))
	}

	// Check node threshold for >15 node fallback
	totalNodes := len(nodes)
	if totalNodes > nodeThreshold {
		builder.WriteString(fmt.Sprintf("%% WARNING: %d nodes, layout may be suboptimal\n", totalNodes))
	}

	// Header with securityLevel: 'strict'
	theme := opts.Theme
	if theme == "" {
		theme = "default"
	}
	builder.WriteString(fmt.Sprintf("%%{init: {'theme':'%s', 'securityLevel':'strict'}}%%\n", theme))
	builder.WriteString(fmt.Sprintf("graph %s\n", opts.Direction))

	// Apply truncation if needed
	truncated := false
	if opts.MaxNodes > 0 && len(nodes) > opts.MaxNodes {
		nodes, truncated = truncateToMaxNodes(nodes, opts.MaxNodes)
	}

	// Write nodes
	for _, node := range nodes {
		builder.WriteString(node + "\n")
	}

	// Add edges for relationships
	for _, rel := range model.Relationships {
		fromID := getL1NodeID(rel.From, systemID, actorIDs, extSystemIDs)
		toID := getL1NodeID(rel.To, systemID, actorIDs, extSystemIDs)

		// Only include edges where both endpoints are in L1
		if fromID == "" || toID == "" {
			continue
		}

		// Skip if either endpoint was truncated
		if !nodeExists(nodes, fromID) || !nodeExists(nodes, toID) {
			continue
		}

		edgeLabel := edgeLabel(rel)
		edges = append(edges, fmt.Sprintf("    %s -->%s %s", fromID, edgeLabel, toID))
	}

	// Write edges
	for _, edge := range edges {
		builder.WriteString(edge + "\n")
	}

	// Add styling
	builder.WriteString("\n")
	builder.WriteString("    classDef actorStyle fill:#e1f5fe,stroke:#01579b,stroke-width:2px,rx:10,ry:10\n")
	builder.WriteString("    classDef systemStyle fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px\n")
	builder.WriteString("    classDef externalStyle fill:#fff3e0,stroke:#ef6c00,stroke-width:2px,stroke-dasharray: 5 5\n")

	// Apply styles to nodes
	var actorClasses []string
	for _, actorID := range actorIDs {
		if nodeExists(nodes, actorID) {
			actorClasses = append(actorClasses, actorID)
		}
	}
	if len(actorClasses) > 0 {
		builder.WriteString(fmt.Sprintf("    class %s actorStyle\n", strings.Join(actorClasses, ",")))
	}

	if nodeExists(nodes, systemID) {
		builder.WriteString(fmt.Sprintf("    class %s systemStyle\n", systemID))
	}

	var extClasses []string
	for _, extID := range extSystemIDs {
		if nodeExists(nodes, extID) {
			extClasses = append(extClasses, extID)
		}
	}
	if len(extClasses) > 0 {
		builder.WriteString(fmt.Sprintf("    class %s externalStyle\n", strings.Join(extClasses, ",")))
	}

	code := builder.String()

	return &DiagramResult{
		Level:       Level1,
		MermaidCode: code,
		NodeCount:   countNodes(code),
		EdgeCount:   countEdges(code),
		Truncated:   truncated,
	}, nil
}

// SECURITY: Per spec Section 4, the rendered Mermaid diagram MUST be placed inside
// a sandboxed <iframe sandbox="allow-scripts"> (no allow-same-origin) when
// displayed in a browser. This function produces raw Mermaid code — the caller
// is responsible for iframe sandboxing at the UI layer.
//
// RenderL2 renders a Container diagram (Level 2).
func RenderL2(model *architect.ReferenceModel, opts RenderOptions) (*DiagramResult, error) {
	if opts.Direction == "" {
		opts.Direction = "TB"
	}

	var builder strings.Builder

	// Collect all node declarations
	var nodes []string
	var edges []string

	// Map to track node IDs for truncation check
	nodeMap := make(map[string]bool)

	// Add the main system boundary
	systemID := sanitizeID(model.System.Name)

	// Add actor nodes
	actorIDs := make(map[string]string)
	for _, actor := range model.Actors {
		actorID := sanitizeID("actor_" + actor.ID)
		actorIDs[actor.ID] = actorID
		label := actor.Description
		if label == "" {
			label = actor.ID
		}
		nodes = append(nodes, fmt.Sprintf("    %s[\"%s\"]", actorID, label))
		nodeMap[actorID] = true
	}

	// Add external system nodes
	extSystemIDs := make(map[string]string)
	for _, ext := range model.ExternalSystems {
		extID := sanitizeID("ext_" + ext.ID)
		extSystemIDs[ext.ID] = extID
		label := ext.Description
		if label == "" {
			label = ext.ID
		}
		if ext.Technology != "" {
			label += fmt.Sprintf("\\n%s", ext.Technology)
		}
		nodes = append(nodes, fmt.Sprintf("    %s[\"%s\\nExternal System\"]", extID, label))
		nodeMap[extID] = true
	}

	// Add container nodes (inside subgraph)
	containerIDs := make(map[string]string)
	for _, container := range model.Containers {
		containerID := sanitizeID("container_" + container.ID)
		containerIDs[container.ID] = containerID
		label := container.Name
		if container.Confidence < confidenceHigh {
			label = confidenceMarker(container.Confidence) + label
		}
		if container.Technology != "" {
			label += fmt.Sprintf("\\n%s", container.Technology)
		}
		nodes = append(nodes, fmt.Sprintf("        %s[\"%s\"]", containerID, label))
		nodeMap[containerID] = true
	}

	// Check node threshold for >15 node fallback
	totalNodes := len(nodes)
	if totalNodes > nodeThreshold {
		builder.WriteString(fmt.Sprintf("%% WARNING: %d nodes, layout may be suboptimal\n", totalNodes))
	}

	// Header with securityLevel: 'strict'
	theme := opts.Theme
	if theme == "" {
		theme = "default"
	}
	builder.WriteString(fmt.Sprintf("%%{init: {'theme':'%s', 'securityLevel':'strict'}}%%\n", theme))
	builder.WriteString(fmt.Sprintf("graph %s\n", opts.Direction))

	// Apply truncation if needed
	truncated := false
	if opts.MaxNodes > 0 && len(nodes) > opts.MaxNodes {
		nodes, truncated = truncateToMaxNodes(nodes, opts.MaxNodes)
	}

	// Write actor nodes (outside subgraph)
	for _, actorID := range actorIDs {
		if nodeExists(nodes, actorID) {
			builder.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", actorID, getActorLabel(model, actorID)))
		}
	}

	// Write external system nodes (outside subgraph)
	for _, ext := range model.ExternalSystems {
		extID := extSystemIDs[ext.ID]
		if nodeExists(nodes, extID) {
			label := ext.Description
			if label == "" {
				label = ext.ID
			}
			if ext.Technology != "" {
				label += fmt.Sprintf("\\n%s", ext.Technology)
			}
			builder.WriteString(fmt.Sprintf("    %s[\"%s\\nExternal System\"]\n", extID, label))
		}
	}

	// Write system subgraph with containers
	builder.WriteString(fmt.Sprintf("    subgraph %s [%s]\n", systemID, model.System.Name))
	for _, node := range nodes {
		// Only write container nodes (they have extra indentation)
		if strings.HasPrefix(node, "        container_") {
			builder.WriteString(node + "\n")
		}
	}
	builder.WriteString("    end\n")

	// Add edges for relationships
	for _, rel := range model.Relationships {
		fromID := getL2NodeID(rel.From, actorIDs, extSystemIDs, containerIDs)
		toID := getL2NodeID(rel.To, actorIDs, extSystemIDs, containerIDs)

		// Only include edges where both endpoints exist
		if fromID == "" || toID == "" {
			continue
		}

		// Skip if either endpoint was truncated
		if !nodeExists(nodes, fromID) && !strings.HasPrefix(fromID, "container_") {
			continue
		}
		if !nodeExists(nodes, toID) && !strings.HasPrefix(toID, "container_") {
			continue
		}

		edgeLabel := edgeLabel(rel)
		edges = append(edges, fmt.Sprintf("    %s -->%s %s", fromID, edgeLabel, toID))
	}

	// Write edges
	for _, edge := range edges {
		builder.WriteString(edge + "\n")
	}

	// Add styling
	builder.WriteString("\n")
	builder.WriteString("    classDef actorStyle fill:#e1f5fe,stroke:#01579b,stroke-width:2px,rx:10,ry:10\n")
	builder.WriteString("    classDef containerStyle fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px\n")
	builder.WriteString("    classDef externalStyle fill:#fff3e0,stroke:#ef6c00,stroke-width:2px,stroke-dasharray: 5 5\n")

	// Apply styles to nodes
	var actorClasses []string
	for _, actorID := range actorIDs {
		if nodeExists(nodes, actorID) {
			actorClasses = append(actorClasses, actorID)
		}
	}
	if len(actorClasses) > 0 {
		builder.WriteString(fmt.Sprintf("    class %s actorStyle\n", strings.Join(actorClasses, ",")))
	}

	var containerClasses []string
	for _, containerID := range containerIDs {
		if nodeExists(nodes, containerID) {
			containerClasses = append(containerClasses, containerID)
		}
	}
	if len(containerClasses) > 0 {
		builder.WriteString(fmt.Sprintf("    class %s containerStyle\n", strings.Join(containerClasses, ",")))
	}

	var extClasses []string
	for _, extID := range extSystemIDs {
		if nodeExists(nodes, extID) {
			extClasses = append(extClasses, extID)
		}
	}
	if len(extClasses) > 0 {
		builder.WriteString(fmt.Sprintf("    class %s externalStyle\n", strings.Join(extClasses, ",")))
	}

	code := builder.String()

	return &DiagramResult{
		Level:       Level2,
		MermaidCode: code,
		NodeCount:   countNodes(code),
		EdgeCount:   countEdges(code),
		Truncated:   truncated,
	}, nil
}

// SECURITY: Per spec Section 4, the rendered Mermaid diagram MUST be placed inside
// a sandboxed <iframe sandbox="allow-scripts"> (no allow-same-origin) when
// displayed in a browser. This function produces raw Mermaid code — the caller
// is responsible for iframe sandboxing at the UI layer.
//
// RenderL3 renders a Component diagram for a specific container (Level 3).
func RenderL3(model *architect.ReferenceModel, containerID string, opts RenderOptions) (*DiagramResult, error) {
	if containerID == "" {
		return nil, fmt.Errorf("containerID cannot be empty")
	}

	// Find the container
	var container *architect.C4Container
	for i := range model.Containers {
		if model.Containers[i].ID == containerID {
			container = &model.Containers[i]
			break
		}
	}

	if container == nil {
		return nil, fmt.Errorf("container with ID %q not found", containerID)
	}

	if opts.Direction == "" {
		opts.Direction = "TB"
	}

	var builder strings.Builder

	// Collect all node declarations
	var nodes []string
	var edges []string

	containerSanitizedID := sanitizeID("container_" + container.ID)

	// Add component nodes
	componentIDs := make(map[string]string)
	for _, comp := range container.Components {
		compID := sanitizeID("comp_" + container.ID + "_" + comp.ID)
		componentIDs[comp.ID] = compID
		label := comp.Description
		if label == "" {
			label = comp.ID
		}
		if comp.Path != "" {
			label += fmt.Sprintf("\\n%s", comp.Path)
		}
		nodes = append(nodes, fmt.Sprintf("        %s[\"%s\"]", compID, label))
	}

	// Check node threshold for >15 node fallback
	totalNodes := len(nodes)
	if totalNodes > nodeThreshold {
		builder.WriteString(fmt.Sprintf("%% WARNING: %d nodes, layout may be suboptimal\n", totalNodes))
	}

	// Header with securityLevel: 'strict'
	theme := opts.Theme
	if theme == "" {
		theme = "default"
	}
	builder.WriteString(fmt.Sprintf("%%{init: {'theme':'%s', 'securityLevel':'strict'}}%%\n", theme))
	builder.WriteString(fmt.Sprintf("graph %s\n", opts.Direction))

	// Apply truncation if needed
	truncated := false
	if opts.MaxNodes > 0 && len(nodes) > opts.MaxNodes {
		nodes, truncated = truncateToMaxNodes(nodes, opts.MaxNodes)
	}

	// Write container subgraph with components
	containerLabel := container.Name
	if container.Confidence < confidenceHigh {
		containerLabel = confidenceMarker(container.Confidence) + containerLabel
	}
	builder.WriteString(fmt.Sprintf("    subgraph %s [%s", containerSanitizedID, containerLabel))
	if container.Technology != "" {
		builder.WriteString(fmt.Sprintf("\\n%s", container.Technology))
	}
	builder.WriteString("]\n")

	for _, node := range nodes {
		builder.WriteString(node + "\n")
	}

	builder.WriteString("    end\n")

	// Add internal edges (component to component relationships)
	for _, rel := range model.Relationships {
		// Check if both from and to are components within this container
		fromIsComponent := false
		toIsComponent := false
		var fromCompID, toCompID string

		for _, comp := range container.Components {
			if rel.From == comp.ID {
				fromIsComponent = true
				fromCompID = componentIDs[comp.ID]
			}
			if rel.To == comp.ID {
				toIsComponent = true
				toCompID = componentIDs[comp.ID]
			}
		}

		if !fromIsComponent || !toIsComponent {
			continue
		}

		// Skip if either endpoint was truncated
		if !nodeExists(nodes, "        "+fromCompID) || !nodeExists(nodes, "        "+toCompID) {
			continue
		}

		edgeLabel := edgeLabel(rel)
		edges = append(edges, fmt.Sprintf("    %s -->%s %s", fromCompID, edgeLabel, toCompID))
	}

	// Write edges
	for _, edge := range edges {
		builder.WriteString(edge + "\n")
	}

	// Add styling
	builder.WriteString("\n")
	builder.WriteString("    classDef componentStyle fill:#e0f2f1,stroke:#00695c,stroke-width:2px\n")

	// Apply styles to component nodes
	var componentClasses []string
	for _, compID := range componentIDs {
		if nodeExists(nodes, "        "+compID) {
			componentClasses = append(componentClasses, compID)
		}
	}
	if len(componentClasses) > 0 {
		builder.WriteString(fmt.Sprintf("    class %s componentStyle\n", strings.Join(componentClasses, ",")))
	}

	code := builder.String()

	return &DiagramResult{
		Level:       Level3,
		MermaidCode: code,
		NodeCount:   countNodes(code),
		EdgeCount:   countEdges(code),
		Truncated:   truncated,
	}, nil
}

// RenderAll renders L1 + L2 + one L3 per container.
func RenderAll(model *architect.ReferenceModel, opts RenderOptions) ([]*DiagramResult, error) {
	var results []*DiagramResult

	// Render L1
	l1, err := RenderL1(model, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to render L1: %w", err)
	}
	results = append(results, l1)

	// Render L2
	l2, err := RenderL2(model, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to render L2: %w", err)
	}
	results = append(results, l2)

	// Render L3 for each container
	for _, container := range model.Containers {
		l3, err := RenderL3(model, container.ID, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to render L3 for container %s: %w", container.ID, err)
		}
		results = append(results, l3)
	}

	return results, nil
}

// ToJSON marshals diagram results to JSON for external editors.
func ToJSON(results []*DiagramResult) (string, error) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results to JSON: %w", err)
	}
	return string(data), nil
}

// Helper functions

// sanitizeID replaces spaces, dots, hyphens with underscores.
func sanitizeID(id string) string {
	// Replace spaces, dots, hyphens, and other special chars with underscores
	re := regexp.MustCompile(`[^\w]`)
	return re.ReplaceAllString(id, "_")
}

// truncateToMaxNodes applies the MaxNodes limit.
func truncateToMaxNodes(nodes []string, max int) ([]string, bool) {
	if len(nodes) <= max {
		return nodes, false
	}
	return nodes[:max], true
}

// countNodes counts node declarations in Mermaid code.
func countNodes(code string) int {
	// Count patterns like: id["label"] or id["label"]
	// This regex matches node declarations
	re := regexp.MustCompile(`\w+\["`)
	matches := re.FindAllString(code, -1)
	return len(matches)
}

// countEdges counts edge declarations in Mermaid code.
func countEdges(code string) int {
	// Count patterns like: --> or --|
	re := regexp.MustCompile(`-->`)
	matches := re.FindAllString(code, -1)
	return len(matches)
}

// edgeLabel formats an edge label with type and description.
func edgeLabel(r architect.C4Relationship) string {
	label := r.Description
	if label == "" {
		label = "uses"
	}

	parts := []string{}
	if r.Type != "" {
		parts = append(parts, r.Type)
	}
	if r.Description != "" {
		parts = append(parts, r.Description)
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("\"%s\"", strings.Join(parts, ": "))
}

// getL1NodeID returns the Mermaid node ID for a C4 element in L1 context.
func getL1NodeID(elementID string, systemID string, actorIDs, extSystemIDs map[string]string) string {
	if elementID == "system" {
		return systemID
	}
	if id, ok := actorIDs[elementID]; ok {
		return id
	}
	if id, ok := extSystemIDs[elementID]; ok {
		return id
	}
	return ""
}

// getL2NodeID returns the Mermaid node ID for a C4 element in L2 context.
func getL2NodeID(elementID string, actorIDs, extSystemIDs, containerIDs map[string]string) string {
	if id, ok := actorIDs[elementID]; ok {
		return id
	}
	if id, ok := extSystemIDs[elementID]; ok {
		return id
	}
	if id, ok := containerIDs[elementID]; ok {
		return id
	}
	return ""
}

// nodeExists checks if a node ID exists in the nodes slice.
func nodeExists(nodes []string, nodeID string) bool {
	for _, node := range nodes {
		if strings.HasPrefix(strings.TrimSpace(node), nodeID+"[") {
			return true
		}
	}
	return false
}

// getActorLabel returns the label for an actor node.
func getActorLabel(model *architect.ReferenceModel, actorID string) string {
	for _, actor := range model.Actors {
		if sanitizeID("actor_"+actor.ID) == actorID {
			if actor.Description != "" {
				return actor.Description
			}
			return actor.ID
		}
	}
	return actorID
}
