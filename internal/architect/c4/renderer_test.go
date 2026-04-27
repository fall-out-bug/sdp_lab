package c4

import (
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

func sampleModel() *architect.ReferenceModel {
	return &architect.ReferenceModel{
		Version: "1.0.0",
		System: architect.SystemInfo{
			Name:        "E-Commerce",
			Description: "Sample e-commerce system",
		},
		Actors: []architect.Actor{
			{ID: "end_user", Description: "Customer"},
			{ID: "admin", Description: "Administrator"},
		},
		ExternalSystems: []architect.ExternalSystem{
			{ID: "stripe", Description: "Payment provider", Technology: "REST API"},
			{ID: "email.service", Description: "Email service", Technology: "SMTP"},
		},
		Containers: []architect.C4Container{
			{
				ID:         "orders",
				Name:       "Orders Service",
				Technology: "Go",
				Components: []architect.C4Component{
					{ID: "domain", Path: "domain/", Description: "Domain logic"},
					{ID: "handler", Path: "handler/", Description: "HTTP handlers"},
					{ID: "repo", Path: "repository/", Description: "Data access"},
				},
			},
			{ID: "api", Name: "API Gateway", Technology: "Node.js"},
			{ID: "frontend", Name: "Web UI", Technology: "React"},
		},
		Relationships: []architect.C4Relationship{
			// L1 relationships (actor -> system, system -> external)
			{From: "end_user", To: "system", Description: "Uses system", Type: "sync"},
			{From: "system", To: "stripe", Description: "Process payments", Type: "sync"},
			// L2 relationships (actor -> container, container -> container, container -> external)
			{From: "end_user", To: "frontend", Description: "Places orders", Type: "sync"},
			{From: "frontend", To: "api", Description: "API calls", Type: "sync"},
			{From: "api", To: "orders", Description: "Routes requests", Type: "sync"},
			{From: "orders", To: "stripe", Description: "Process payments", Type: "sync"},
			{From: "orders", To: "email.service", Description: "Send notifications", Type: "async"},
			// L3 relationships (component -> component)
			{From: "domain", To: "repo", Description: "Persists data", Type: "sync"},
		},
	}
}

func TestRenderL1(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{Direction: "TB", Theme: "default"}

	result, err := RenderL1(model, opts)
	if err != nil {
		t.Fatalf("RenderL1 failed: %v", err)
	}

	if result.Level != Level1 {
		t.Errorf("expected Level1, got %v", result.Level)
	}

	if result.NodeCount == 0 {
		t.Error("expected non-zero node count")
	}

	code := result.MermaidCode

	// Check for system node
	if !contains(code, "E_Commerce") && !contains(code, "E-Commerce") {
		t.Error("expected system name in diagram")
	}

	// Check for actor nodes
	if !contains(code, "actor_end_user") {
		t.Error("expected actor_end_user node")
	}

	// Check for external system nodes
	if !contains(code, "ext_stripe") {
		t.Error("expected ext_stripe node")
	}

	// Check for edges
	if result.EdgeCount == 0 {
		t.Error("expected non-zero edge count")
	}

	// Check for class definitions
	if !contains(code, "classDef actorStyle") {
		t.Error("expected actorStyle class definition")
	}
	if !contains(code, "classDef systemStyle") {
		t.Error("expected systemStyle class definition")
	}
	if !contains(code, "classDef externalStyle") {
		t.Error("expected externalStyle class definition")
	}

	// Check for graph direction
	if !contains(code, "graph TB") {
		t.Error("expected 'graph TB' directive")
	}
}

func TestRenderL1Empty(t *testing.T) {
	model := &architect.ReferenceModel{
		System: architect.SystemInfo{Name: "Empty System"},
	}
	opts := RenderOptions{}

	result, err := RenderL1(model, opts)
	if err != nil {
		t.Fatalf("RenderL1 with empty model failed: %v", err)
	}

	if result.MermaidCode == "" {
		t.Error("expected non-empty mermaid code")
	}

	if !contains(result.MermaidCode, "graph") {
		t.Error("expected 'graph' in mermaid code")
	}
}

func TestRenderL2(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{Direction: "LR"}

	result, err := RenderL2(model, opts)
	if err != nil {
		t.Fatalf("RenderL2 failed: %v", err)
	}

	if result.Level != Level2 {
		t.Errorf("expected Level2, got %v", result.Level)
	}

	code := result.MermaidCode

	// Check for subgraph (system boundary)
	if !contains(code, "subgraph") {
		t.Error("expected subgraph for system boundary")
	}

	// Check for container nodes
	if !contains(code, "container_orders") {
		t.Error("expected container_orders node")
	}
	if !contains(code, "container_api") {
		t.Error("expected container_api node")
	}
	if !contains(code, "container_frontend") {
		t.Error("expected container_frontend node")
	}

	// Check for actor nodes
	if !contains(code, "actor_end_user") {
		t.Error("expected actor_end_user node")
	}

	// Check for external system nodes
	if !contains(code, "ext_stripe") {
		t.Error("expected ext_stripe node")
	}

	// Check for graph direction
	if !contains(code, "graph LR") {
		t.Error("expected 'graph LR' directive")
	}

	// Check for container style
	if !contains(code, "classDef containerStyle") {
		t.Error("expected containerStyle class definition")
	}
}

func TestRenderL2WithRelationships(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{}

	result, err := RenderL2(model, opts)
	if err != nil {
		t.Fatalf("RenderL2 failed: %v", err)
	}

	code := result.MermaidCode

	// Check for sync relationship
	if !contains(code, "sync") {
		t.Error("expected 'sync' in edge labels")
	}

	// Check for async relationship
	if !contains(code, "async") {
		t.Error("expected 'async' in edge labels")
	}

	// Check for edges between containers
	if !contains(code, "-->") {
		t.Error("expected edge declarations")
	}
}

func TestRenderL3(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{}

	result, err := RenderL3(model, "orders", opts)
	if err != nil {
		t.Fatalf("RenderL3 failed: %v", err)
	}

	if result.Level != Level3 {
		t.Errorf("expected Level3, got %v", result.Level)
	}

	code := result.MermaidCode

	// Check for container subgraph
	if !contains(code, "subgraph container_orders") {
		t.Error("expected container_orders subgraph")
	}

	// Check for component nodes
	if !contains(code, "comp_orders_domain") {
		t.Error("expected comp_orders_domain node")
	}
	if !contains(code, "comp_orders_handler") {
		t.Error("expected comp_orders_handler node")
	}
	if !contains(code, "comp_orders_repo") {
		t.Error("expected comp_orders_repo node")
	}

	// Check for component descriptions
	if !contains(code, "Domain logic") {
		t.Error("expected component description in diagram")
	}

	// Check for component style
	if !contains(code, "classDef componentStyle") {
		t.Error("expected componentStyle class definition")
	}
}

func TestRenderL3NoContainerID(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{}

	_, err := RenderL3(model, "", opts)
	if err == nil {
		t.Error("expected error for empty containerID")
	}

	if !contains(err.Error(), "containerID cannot be empty") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

func TestRenderL3ContainerNotFound(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{}

	_, err := RenderL3(model, "nonexistent", opts)
	if err == nil {
		t.Error("expected error for nonexistent container")
	}

	if !contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error message, got: %v", err)
	}
}

func TestRenderAll(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{}

	results, err := RenderAll(model, opts)
	if err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	// Should have L1 + L2 + 3 L3s (one per container)
	expectedCount := 1 + 1 + 3 // L1 + L2 + 3 containers
	if len(results) != expectedCount {
		t.Errorf("expected %d diagrams, got %d", expectedCount, len(results))
	}

	// Check L1
	if results[0].Level != Level1 {
		t.Errorf("expected first result to be L1, got %v", results[0].Level)
	}

	// Check L2
	if results[1].Level != Level2 {
		t.Errorf("expected second result to be L2, got %v", results[1].Level)
	}

	// Check L3s
	l3Count := 0
	for _, r := range results {
		if r.Level == Level3 {
			l3Count++
		}
	}
	if l3Count != 3 {
		t.Errorf("expected 3 L3 diagrams, got %d", l3Count)
	}
}

func TestToJSON(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{}

	results, err := RenderAll(model, opts)
	if err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	jsonStr, err := ToJSON(results)
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if jsonStr == "" {
		t.Error("expected non-empty JSON string")
	}

	// Check for JSON fields
	if !contains(jsonStr, `"Level"`) {
		t.Error("expected 'Level' field in JSON")
	}
	if !contains(jsonStr, `"MermaidCode"`) {
		t.Error("expected 'MermaidCode' field in JSON")
	}
	if !contains(jsonStr, `"NodeCount"`) {
		t.Error("expected 'NodeCount' field in JSON")
	}
	if !contains(jsonStr, `"EdgeCount"`) {
		t.Error("expected 'EdgeCount' field in JSON")
	}
}

func TestMaxNodesTruncation(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{MaxNodes: 2}

	result, err := RenderL1(model, opts)
	if err != nil {
		t.Fatalf("RenderL1 with MaxNodes failed: %v", err)
	}

	if !result.Truncated {
		t.Error("expected Truncated to be true")
	}

	// With MaxNodes=2, we should have at most 2 nodes
	if result.NodeCount > 2 {
		t.Errorf("expected at most 2 nodes, got %d", result.NodeCount)
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "with_space"},
		{"with-hyphen", "with_hyphen"},
		{"with.dot", "with_dot"},
		{"with@special#chars", "with_special_chars"},
		{"MixedCase-With-Special.chars", "MixedCase_With_Special_chars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeID(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{Level1, "L1"},
		{Level2, "L2"},
		{Level3, "L3"},
		{Level(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestRenderL2LeftRight(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{Direction: "LR"}

	result, err := RenderL2(model, opts)
	if err != nil {
		t.Fatalf("RenderL2 with LR direction failed: %v", err)
	}

	if !contains(result.MermaidCode, "graph LR") {
		t.Error("expected 'graph LR' directive")
	}
}

func TestRenderL3WithComponents(t *testing.T) {
	model := sampleModel()
	opts := RenderOptions{}

	result, err := RenderL3(model, "orders", opts)
	if err != nil {
		t.Fatalf("RenderL3 failed: %v", err)
	}

	code := result.MermaidCode

	// Check that all components are present
	components := []string{"domain", "handler", "repo"}
	for _, comp := range components {
		if !contains(code, "comp_orders_"+comp) {
			t.Errorf("expected component node for %s", comp)
		}
	}

	// Check for internal relationships (domain -> repo)
	if !contains(code, "comp_orders_domain") || !contains(code, "comp_orders_repo") {
		t.Error("expected component nodes to be present for internal relationships")
	}
}

func TestEdgeLabel(t *testing.T) {
	tests := []struct {
		rel      architect.C4Relationship
		expected string
	}{
		{
			architect.C4Relationship{Type: "sync", Description: "calls"},
			"\"sync: calls\"",
		},
		{
			architect.C4Relationship{Type: "async", Description: "sends"},
			"\"async: sends\"",
		},
		{
			architect.C4Relationship{Description: "uses"},
			"\"uses\"",
		},
		{
			architect.C4Relationship{Type: "sync"},
			"\"sync\"",
		},
		{
			architect.C4Relationship{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := edgeLabel(tt.rel)
			if result != tt.expected {
				t.Errorf("edgeLabel(%+v) = %q, want %q", tt.rel, result, tt.expected)
			}
		})
	}
}

func TestCountNodes(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "empty",
			code:     "",
			expected: 0,
		},
		{
			name:     "single node",
			code:     `node1["label"]`,
			expected: 1,
		},
		{
			name:     "multiple nodes",
			code:     `node1["label1"] node2["label2"] node3["label3"]`,
			expected: 3,
		},
		{
			name:     "with edges",
			code:     `node1["label1"] --> node2["label2"]`,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countNodes(tt.code)
			if result != tt.expected {
				t.Errorf("countNodes(%q) = %d, want %d", tt.code, result, tt.expected)
			}
		})
	}
}

func TestCountEdges(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name:     "empty",
			code:     "",
			expected: 0,
		},
		{
			name:     "single edge",
			code:     `node1 --> node2`,
			expected: 1,
		},
		{
			name:     "multiple edges",
			code:     `node1 --> node2 node3 --> node4 node5 --> node6`,
			expected: 3,
		},
		{
			name:     "with labels",
			code:     `node1 -->|"label"| node2`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countEdges(tt.code)
			if result != tt.expected {
				t.Errorf("countEdges(%q) = %d, want %d", tt.code, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
