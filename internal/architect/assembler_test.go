package architect

import (
	"context"
	"errors"
	"testing"
)

// failingMockExtractor is a test implementation that can fail.
type failingMockExtractor struct {
	name     string
	fragment *ProfileFragment
	fail     bool
}

func (m *failingMockExtractor) Name() string {
	return m.name
}

func (m *failingMockExtractor) Extract(ctx context.Context, repoRoot string) (*ProfileFragment, error) {
	if m.fail {
		return nil, errors.New("mock extractor failure")
	}
	return m.fragment, nil
}

// TestMergeModulesByIDD verifies that modules are merged by ID with higher priority winning.
func TestMergeModulesByIDD(t *testing.T) {
	frag1 := &ProfileFragment{
		Modules: []Module{
			{ID: "go\x00internal/arch\x00arch", Name: "arch", Language: "go", Path: "internal/arch"},
			{ID: "go\x00internal/api\x00api", Name: "api", Language: "go", Path: "internal/api"},
		},
	}
	frag2 := &ProfileFragment{
		Modules: []Module{
			{ID: "go\x00internal/arch\x00arch", Name: "arch-updated", Language: "go", Path: "internal/arch-new"}, // Same ID, higher priority should win
			{ID: "go\x00internal/auth\x00auth", Name: "auth", Language: "go", Path: "internal/auth"},
		},
	}

	assembler := NewProfileAssembler(Tier2, []Extractor{
		&failingMockExtractor{name: "filetree", fragment: frag1}, // priority 1
		&failingMockExtractor{name: "specs", fragment: frag2},    // priority 3
	})

	// Simulate merge
	priorityFrags := []priorityFragment{
		{fragment: frag1, priority: 1, name: "filetree"},
		{fragment: frag2, priority: 3, name: "specs"},
	}

	profile := assembler.mergeFragments(priorityFrags)

	// Should have 3 unique modules
	if len(profile.Modules) != 3 {
		t.Errorf("expected 3 modules, got %d", len(profile.Modules))
	}

	// Find the arch module (should be from higher priority specs extractor)
	var archModule *Module
	for i := range profile.Modules {
		if profile.Modules[i].ID == "go\x00internal/arch\x00arch" {
			archModule = &profile.Modules[i]
			break
		}
	}

	if archModule == nil {
		t.Fatal("arch module not found")
	}

	if archModule.Name != "arch-updated" {
		t.Errorf("expected arch module name 'arch-updated' from higher priority, got '%s'", archModule.Name)
	}
	if archModule.Path != "internal/arch-new" {
		t.Errorf("expected arch module path 'internal/arch-new' from higher priority, got '%s'", archModule.Path)
	}
}

// TestMergeEdgesByTuple verifies that edges are merged by (source, target, kind, protocol) tuple.
func TestMergeEdgesByTuple(t *testing.T) {
	frag1 := &ProfileFragment{
		Edges: []StructuralEdge{
			{Source: "mod1", Target: "mod2", Kind: EdgeImport, Weight: 1, Confidence: 0.8},
			{Source: "mod2", Target: "mod3", Kind: EdgeCall, Weight: 1, Confidence: 0.7},
		},
	}
	frag2 := &ProfileFragment{
		Edges: []StructuralEdge{
			{Source: "mod1", Target: "mod2", Kind: EdgeImport, Weight: 2, Confidence: 0.9}, // Same tuple, should merge
			{Source: "mod3", Target: "mod4", Kind: EdgeCall, Weight: 1, Confidence: 0.6},
		},
	}

	assembler := NewProfileAssembler(Tier2, []Extractor{})

	priorityFrags := []priorityFragment{
		{fragment: frag1, priority: 1, name: "filetree"},
		{fragment: frag2, priority: 2, name: "deps"},
	}

	profile := assembler.mergeFragments(priorityFrags)

	// Should have 3 unique edges
	if len(profile.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(profile.Edges))
	}

	// Find the merged edge
	var mergedEdge *StructuralEdge
	for i := range profile.Edges {
		if profile.Edges[i].Source == "mod1" && profile.Edges[i].Target == "mod2" && profile.Edges[i].Kind == EdgeImport {
			mergedEdge = &profile.Edges[i]
			break
		}
	}

	if mergedEdge == nil {
		t.Fatal("merged edge not found")
	}

	// Weight should be summed
	if mergedEdge.Weight != 3 {
		t.Errorf("expected weight 3 (1+2), got %d", mergedEdge.Weight)
	}

	// Confidence should be higher of the two
	if mergedEdge.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9 (higher), got %f", mergedEdge.Confidence)
	}
}

// TestMergeAPISurfaces verifies that API surfaces are merged by (path, method) tuple.
func TestMergeAPISurfaces(t *testing.T) {
	frag1 := &ProfileFragment{
		APISurfaces: []APISurface{
			{Path: "/api/users", Method: "GET", Handler: "GetUsers"},
			{Path: "/api/posts", Method: "POST", Handler: "CreatePost"},
		},
	}
	frag2 := &ProfileFragment{
		APISurfaces: []APISurface{
			{Path: "/api/users", Method: "GET", Handler: "GetUsersV2"}, // Same path/method, higher priority wins
			{Path: "/api/comments", Method: "GET", Handler: "GetComments"},
		},
	}

	assembler := NewProfileAssembler(Tier2, []Extractor{})

	priorityFrags := []priorityFragment{
		{fragment: frag1, priority: 1, name: "filetree"},
		{fragment: frag2, priority: 5, name: "go"}, // higher priority
	}

	profile := assembler.mergeFragments(priorityFrags)

	// Should have 3 unique API surfaces
	if len(profile.APISurfaces) != 3 {
		t.Errorf("expected 3 API surfaces, got %d", len(profile.APISurfaces))
	}

	// Find the GET /api/users endpoint
	var usersAPI *APISurface
	for i := range profile.APISurfaces {
		if profile.APISurfaces[i].Path == "/api/users" && profile.APISurfaces[i].Method == "GET" {
			usersAPI = &profile.APISurfaces[i]
			break
		}
	}

	if usersAPI == nil {
		t.Fatal("GET /api/users not found")
	}

	// Should be from higher priority extractor
	if usersAPI.Handler != "GetUsersV2" {
		t.Errorf("expected handler 'GetUsersV2' from higher priority, got '%s'", usersAPI.Handler)
	}
}

// TestMergeBoundaries verifies that boundaries are merged by name with entry_files and public_interfaces unioned.
func TestMergeBoundaries(t *testing.T) {
	frag1 := &ProfileFragment{
		Boundaries: []ModuleBoundary{
			{Name: "auth", Pattern: "internal/auth/*", EntryFiles: []string{"internal/auth/main.go"}, PublicInterfaces: []string{"Login"}},
		},
	}
	frag2 := &ProfileFragment{
		Boundaries: []ModuleBoundary{
			{Name: "auth", Pattern: "internal/auth/**", EntryFiles: []string{"internal/auth/handler.go"}, PublicInterfaces: []string{"Logout"}},
			{Name: "billing", Pattern: "internal/billing/*", EntryFiles: []string{"internal/billing/service.go"}, PublicInterfaces: []string{"Charge"}},
		},
	}

	assembler := NewProfileAssembler(Tier2, []Extractor{})

	priorityFrags := []priorityFragment{
		{fragment: frag1, priority: 1, name: "filetree"},
		{fragment: frag2, priority: 4, name: "infra"},
	}

	profile := assembler.mergeFragments(priorityFrags)

	// Should have 2 unique boundaries
	if len(profile.Boundaries) != 2 {
		t.Errorf("expected 2 boundaries, got %d", len(profile.Boundaries))
	}

	// Find the auth boundary
	var authBoundary *ModuleBoundary
	for i := range profile.Boundaries {
		if profile.Boundaries[i].Name == "auth" {
			authBoundary = &profile.Boundaries[i]
			break
		}
	}

	if authBoundary == nil {
		t.Fatal("auth boundary not found")
	}

	// Pattern should be from higher priority extractor
	if authBoundary.Pattern != "internal/auth/**" {
		t.Errorf("expected pattern 'internal/auth/**' from higher priority, got '%s'", authBoundary.Pattern)
	}

	// EntryFiles should be unioned
	if len(authBoundary.EntryFiles) != 2 {
		t.Errorf("expected 2 entry files (union), got %d", len(authBoundary.EntryFiles))
	}

	// PublicInterfaces should be unioned
	if len(authBoundary.PublicInterfaces) != 2 {
		t.Errorf("expected 2 public interfaces (union), got %d", len(authBoundary.PublicInterfaces))
	}
}

// TestMergeLayers verifies that layers are merged by layer name with highest confidence kept.
func TestMergeLayers(t *testing.T) {
	frag1 := &ProfileFragment{
		Layers: []LayerAssignment{
			{Layer: "presentation", Directories: []string{"api/", "handlers/"}, Confidence: 0.7},
			{Layer: "business", Directories: []string{"services/"}, Confidence: 0.8},
		},
	}
	frag2 := &ProfileFragment{
		Layers: []LayerAssignment{
			{Layer: "presentation", Directories: []string{"web/"}, Confidence: 0.9}, // Same layer, higher confidence wins
			{Layer: "data", Directories: []string{"repositories/"}, Confidence: 0.6},
		},
	}

	assembler := NewProfileAssembler(Tier2, []Extractor{})

	priorityFrags := []priorityFragment{
		{fragment: frag1, priority: 1, name: "filetree"},
		{fragment: frag2, priority: 2, name: "deps"},
	}

	profile := assembler.mergeFragments(priorityFrags)

	// Should have 3 unique layers
	if len(profile.Layers) != 3 {
		t.Errorf("expected 3 layers, got %d", len(profile.Layers))
	}

	// Find the presentation layer
	var presLayer *LayerAssignment
	for i := range profile.Layers {
		if profile.Layers[i].Layer == "presentation" {
			presLayer = &profile.Layers[i]
			break
		}
	}

	if presLayer == nil {
		t.Fatal("presentation layer not found")
	}

	// Should have higher confidence
	if presLayer.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", presLayer.Confidence)
	}

	// Should have directories from the higher confidence entry
	if len(presLayer.Directories) != 1 || presLayer.Directories[0] != "web/" {
		t.Errorf("expected directories [\"web/\"], got %v", presLayer.Directories)
	}
}

// TestExtractorPrecedence verifies the extractor priority order.
func TestExtractorPrecedence(t *testing.T) {
	expected := map[string]int{
		"filetree":    1,
		"deps":        2,
		"specs":       3,
		"infra":       4,
		"go":          5,
		"python":      6,
		"java":        7,
		"typescript":  8,
		"git_history": 9,
		"sql":         10,
		"generated":   11,
	}

	for name, expectedPrio := range expected {
		actualPrio := extractorPriorityOf(name)
		if actualPrio != expectedPrio {
			t.Errorf("extractor %s: expected priority %d, got %d", name, expectedPrio, actualPrio)
		}
	}

	// Unknown extractor should return default (0)
	unknownPrio := extractorPriorityOf("unknown")
	if unknownPrio != 0 {
		t.Errorf("unknown extractor: expected priority 0, got %d", unknownPrio)
	}
}

// TestAssembleWithFailedExtractor verifies that non-fatal extractor errors don't stop assembly.
func TestAssembleWithFailedExtractor(t *testing.T) {
	successFrag := &ProfileFragment{
		Modules: []Module{
			{ID: "go\x00main\x00main", Name: "main", Language: "go", Path: "."},
		},
	}

	assembler := NewProfileAssembler(Tier2, []Extractor{
		&failingMockExtractor{name: "filetree", fragment: successFrag, fail: false},
		&failingMockExtractor{name: "specs", fragment: nil, fail: true}, // This one will fail
	})

	ctx := context.Background()
	profile, err := assembler.Assemble(ctx, "/tmp/fake-repo")

	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	if profile == nil {
		t.Fatal("expected non-nil profile")
	}

	// Should have modules from the successful extractor
	if len(profile.Modules) != 1 {
		t.Errorf("expected 1 module from successful extractor, got %d", len(profile.Modules))
	}

	// Should have recorded the error
	errors := assembler.Errors()
	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}
}

// TestAssembleTierFiltering verifies that tier filtering works correctly.
func TestAssembleTierFiltering(t *testing.T) {
	frag := &ProfileFragment{
		FileTree: &FileTreeInfo{
			TotalFiles: 100,
			ExtCounts:  map[string]int{".go": 80, ".md": 20},
		},
		ImportGraph: &ImportGraph{
			ExtractionMethod: "tree-sitter",
			AccuracyEstimate: 0.95,
			Nodes:            50,
			Edges:            120,
			Clusters:         []ImportCluster{{ID: "cluster1"}},
		},
		Infra: &InfraInfo{
			Containers: []ContainerInfo{{Name: "api", Type: "service", Source: "Dockerfile"}},
			Resources:  []ResourceInfo{{Type: "aws_s3_bucket", Name: "data"}},
		},
		Metrics: &CodeMetrics{
			TotalFiles: 100,
			TotalLOC:   5000,
		},
		Modules: []Module{
			{ID: "go\x00main\x00main", Name: "main", Language: "go", Path: "."},
		},
	}

	assembler := NewProfileAssembler(Tier1, []Extractor{
		&failingMockExtractor{name: "filetree", fragment: frag},
	})

	ctx := context.Background()
	profile, err := assembler.Assemble(ctx, "/tmp/fake-repo")

	if err != nil {
		t.Fatalf("Assemble failed: %v", err)
	}

	// Tier1 should keep containers but strip resources
	if len(profile.Infra.Containers) != 1 {
		t.Errorf("Tier1: expected 1 container, got %d", len(profile.Infra.Containers))
	}
	if len(profile.Infra.Resources) != 0 {
		t.Errorf("Tier1: expected 0 resources (stripped), got %d", len(profile.Infra.Resources))
	}

	// Tier1 should keep import graph summary but strip clusters
	if profile.ImportGraph.Nodes != 50 {
		t.Errorf("Tier1: expected 50 nodes, got %d", profile.ImportGraph.Nodes)
	}
	if len(profile.ImportGraph.Clusters) != 0 {
		t.Errorf("Tier1: expected 0 clusters (stripped), got %d", len(profile.ImportGraph.Clusters))
	}

	// Tier1 should generate summary
	if profile.Summary == "" {
		t.Error("Tier1: expected non-empty summary")
	}
}

// TestExtractoPrecedenceHigherWins verifies that once a higher-precedence extractor
// populates a key, no lower-precedence extractor may overwrite it.
func TestExtractorPrecedenceHigherWins(t *testing.T) {
	// Lower priority extractor populates module
	fragLow := &ProfileFragment{
		Modules: []Module{
			{ID: "go\x00main\x00main", Name: "main-low", Language: "go", Path: "."},
		},
	}

	// Higher priority extractor populates same module ID
	fragHigh := &ProfileFragment{
		Modules: []Module{
			{ID: "go\x00main\x00main", Name: "main-high", Language: "go", Path: "./cmd"},
		},
	}

	assembler := NewProfileAssembler(Tier2, []Extractor{})

	// Sort by priority (low first)
	priorityFrags := []priorityFragment{
		{fragment: fragLow, priority: 1, name: "filetree"},
		{fragment: fragHigh, priority: 5, name: "go"},
	}

	profile := assembler.mergeFragments(priorityFrags)

	// Should have 1 module with higher priority values
	if len(profile.Modules) != 1 {
		t.Errorf("expected 1 module, got %d", len(profile.Modules))
	}

	if profile.Modules[0].Name != "main-high" {
		t.Errorf("expected module name 'main-high' from higher priority, got '%s'", profile.Modules[0].Name)
	}

	if profile.Modules[0].Path != "./cmd" {
		t.Errorf("expected module path './cmd' from higher priority, got '%s'", profile.Modules[0].Path)
	}
}

// TestMergeStringSlices verifies the string slice union deduplication helper.
func TestMergeStringSlices(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"b", "c", "d"}

	result := mergeStringSlices(a, b)

	// Should have 4 unique elements
	if len(result) != 4 {
		t.Errorf("expected 4 elements, got %d", len(result))
	}

	// Check for uniqueness
	seen := make(map[string]bool)
	for _, s := range result {
		if seen[s] {
			t.Errorf("duplicate element: %s", s)
		}
		seen[s] = true
	}

	// Check all expected elements are present
	expected := map[string]bool{"a": true, "b": true, "c": true, "d": true}
	for _, s := range result {
		if !expected[s] {
			t.Errorf("unexpected element: %s", s)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
