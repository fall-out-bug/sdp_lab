// Package gotest tests for Go import graph extraction.
package gotest

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"sdp_dev/internal/assert"
)

// TestSimpleCLI tests extraction from a simple CLI project.
func TestSimpleCLI(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("testdata", "simple_cli")
	extractor := NewExtractor(projectDir)

	ctx := context.Background()
	graph, err := extractor.Extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)
	assert.Equal(t, "github.com/example/simplecli", graph.ModulePath)
	assert.True(t, len(graph.Nodes) > 0, "expected at least one node")

	// Check that cmd/simplecli is detected
	var simplecliUnit *DeployUnit
	for _, unit := range graph.DeployUnits {
		if unit.Name == "simplecli" {
			simplecliUnit = &unit
			break
		}
	}
	assert.NotNil(t, simplecliUnit, "expected to find simplecli deploy unit")
	assert.True(t, simplecliUnit.HasMain, "expected simplecli to have main")

	// Verify no cycles for a simple project
	assert.Equal(t, 0, len(graph.Cycles), "expected no circular dependencies")
}

// TestMultiModuleMonorepo tests extraction from a multi-module monorepo.
func TestMultiModuleMonorepo(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("testdata", "monorepo")
	extractor := NewExtractor(projectDir)

	ctx := context.Background()
	graph, err := extractor.Extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	// Should detect multiple modules
	assert.True(t, len(graph.DeployUnits) > 0, "expected at least one deploy unit")

	// Verify module info is parsed
	assert.NotNil(t, graph.ModuleInfo)
	assert.NotEqual(t, "", graph.ModuleInfo.GoVersion, "expected Go version to be set")
}

// TestGRPCService tests extraction from a gRPC service project.
func TestGRPCService(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("testdata", "grpc_service")
	extractor := NewExtractor(projectDir)

	ctx := context.Background()
	graph, err := extractor.Extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	// Should detect gRPC framework
	hasGRPC := false
	for _, fw := range graph.Frameworks {
		if fw.Name == "gRPC" {
			hasGRPC = true
			assert.Greater(t, fw.Confidence, 0.8, "expected high confidence for gRPC detection")
			break
		}
	}
	assert.True(t, hasGRPC, "expected to detect gRPC framework")
}

// TestGinRestAPI tests extraction from a Gin REST API project.
func TestGinRestAPI(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("testdata", "gin_api")
	extractor := NewExtractor(projectDir)

	ctx := context.Background()
	graph, err := extractor.Extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	// Should detect Gin framework
	hasGin := false
	for _, fw := range graph.Frameworks {
		if fw.Name == "Gin" {
			hasGin = true
			assert.Greater(t, fw.Confidence, 0.9, "expected high confidence for Gin detection")
			break
		}
	}
	assert.True(t, hasGin, "expected to detect Gin framework")

	// Check for handler clustering
	hasHandlerCluster := false
	for _, cluster := range graph.Clusters {
		if strings.Contains(cluster, "handler") || strings.Contains(cluster, "api") {
			hasHandlerCluster = true
			break
		}
	}
	assert.True(t, hasHandlerCluster, "expected to detect handler/api cluster")
}

// TestCircularDependencies tests cycle detection with Tarjan's algorithm.
func TestCircularDependencies(t *testing.T) {
	t.Parallel()

	projectDir := filepath.Join("testdata", "circular_deps")
	extractor := NewExtractor(projectDir)

	ctx := context.Background()
	graph, err := extractor.Extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	// Should detect at least one cycle
	assert.Greater(t, len(graph.Cycles), 0, "expected to detect circular dependencies")

	// Verify cycle format
	for _, cycle := range graph.Cycles {
		assert.Greater(t, len(cycle), 1, "cycle should have at least 2 nodes")
		// First and last should connect (cycle)
		if len(cycle) > 0 {
			assert.NotEqual(t, cycle[0], cycle[len(cycle)-1], "cycle should not repeat start node")
		}
	}
}

// TestFrameworkDetection tests framework detection accuracy.
func TestFrameworkDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		imports       map[string]bool
		expectedFw    []string
		unexpectedFw []string
	}{
		{
			name: "Gin only",
			imports: map[string]bool{
				"github.com/gin-gonic/gin": true,
				"net/http":                true,
			},
			expectedFw: []string{"Gin", "stdlib HTTP"},
		},
		{
			name: "Echo only",
			imports: map[string]bool{
				"github.com/labstack/echo/v4": true,
				"net/http":                    true,
			},
			expectedFw: []string{"Echo", "stdlib HTTP"},
		},
		{
			name: "Chi only",
			imports: map[string]bool{
				"github.com/go-chi/chi/v5": true,
				"net/http":                 true,
			},
			expectedFw: []string{"Chi", "stdlib HTTP"},
		},
		{
			name: "gRPC only",
			imports: map[string]bool{
				"google.golang.org/grpc": true,
			},
			expectedFw: []string{"gRPC"},
		},
		{
			name: "Cobra CLI",
			imports: map[string]bool{
				"github.com/spf13/cobra": true,
			},
			expectedFw: []string{"Cobra"},
		},
		{
			name: "Multiple frameworks",
			imports: map[string]bool{
				"github.com/gin-gonic/gin":          true,
				"google.golang.org/grpc":            true,
				"github.com/spf13/cobra":            true,
				"github.com/golang/mock":            true,
				"github.com/stretchr/testify/assert": true,
			},
			expectedFw: []string{"Gin", "gRPC", "Cobra", "testify"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importSet := make(map[string]struct{})
			for imp := range tt.imports {
				importSet[imp] = struct{}{}
			}

			frameworks := detectFrameworks(importSet)

			// Check expected frameworks
			for _, expected := range tt.expectedFw {
				found := false
				for _, fw := range frameworks {
					if fw.Name == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "expected to find framework %s", expected)
			}

			// Check unexpected frameworks
			for _, unexpected := range tt.unexpectedFw {
				found := false
				for _, fw := range frameworks {
					if fw.Name == unexpected {
						found = true
						break
					}
				}
				assert.False(t, found, "did not expect to find framework %s", unexpected)
			}
		})
	}
}

// TestGoModParsing tests go.mod parsing with various directives.
func TestGoModParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		goModContent string
		expectedInfo ModuleInfo
	}{
		{
			name: "basic go.mod",
			goModContent: `module github.com/example/test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`,
			expectedInfo: ModuleInfo{
				ModulePath: "github.com/example/test",
				GoVersion:  "1.21",
				Requires: []ModuleDep{
					{Path: "github.com/gin-gonic/gin", Version: "v1.9.1", Indirect: false},
				},
			},
		},
		{
			name: "with indirect dependency",
			goModContent: `module github.com/example/test

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.8.4 // indirect
)
`,
			expectedInfo: ModuleInfo{
				ModulePath: "github.com/example/test",
				GoVersion:  "1.21",
				Requires: []ModuleDep{
					{Path: "github.com/gin-gonic/gin", Version: "v1.9.1", Indirect: false},
					{Path: "github.com/stretchr/testify", Version: "v1.8.4", Indirect: true},
				},
			},
		},
		{
			name: "with replace directive",
			goModContent: `module github.com/example/test

go 1.21

require github.com/gin-gonic/gin v1.9.1

replace github.com/gin-gonic/gin => ../gin
`,
			expectedInfo: ModuleInfo{
				ModulePath: "github.com/example/test",
				GoVersion:  "1.21",
				Requires: []ModuleDep{
					{Path: "github.com/gin-gonic/gin", Version: "v1.9.1", Indirect: false},
				},
				Replaces: []ModuleDep{
					{Path: "github.com/gin-gonic/gin", Version: "../gin", IsReplace: true},
				},
			},
		},
		{
			name: "with exclude directive",
			goModContent: `module github.com/example/test

go 1.21

require github.com/gin-gonic/gin v1.9.1

exclude github.com/gin-gonic/gin v1.9.0
`,
			expectedInfo: ModuleInfo{
				ModulePath: "github.com/example/test",
				GoVersion:  "1.21",
				Requires: []ModuleDep{
					{Path: "github.com/gin-gonic/gin", Version: "v1.9.1", Indirect: false},
				},
				Excludes: []string{"github.com/gin-gonic/gin"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary go.mod file
			tmpDir := t.TempDir()
			goModPath := filepath.Join(tmpDir, "go.mod")
			err := os.WriteFile(goModPath, []byte(tt.goModContent), 0644)
			assert.NoError(t, err)

			info := parseModuleInfo(goModPath)
			assert.NotNil(t, info)
			assert.Equal(t, tt.expectedInfo.ModulePath, info.ModulePath)
			assert.Equal(t, tt.expectedInfo.GoVersion, info.GoVersion)

			// Check requires
			assert.Equal(t, len(tt.expectedInfo.Requires), len(info.Requires))
			for i, req := range tt.expectedInfo.Requires {
				assert.Equal(t, req.Path, info.Requires[i].Path)
				assert.Equal(t, req.Version, info.Requires[i].Version)
				assert.Equal(t, req.Indirect, info.Requires[i].Indirect)
			}

			// Check replaces
			assert.Equal(t, len(tt.expectedInfo.Replaces), len(info.Replaces))
			for i, rep := range tt.expectedInfo.Replaces {
				assert.Equal(t, rep.Path, info.Replaces[i].Path)
				assert.Equal(t, rep.Version, info.Replaces[i].Version)
				assert.Equal(t, rep.IsReplace, info.Replaces[i].IsReplace)
			}

			// Check excludes
			assert.Equal(t, len(tt.expectedInfo.Excludes), len(info.Excludes))
			sort.Strings(info.Excludes)
			for i, exc := range tt.expectedInfo.Excludes {
				assert.Equal(t, exc, info.Excludes[i])
			}
		})
	}
}

// TestClustering tests package clustering heuristics.
func TestClustering(t *testing.T) {
	t.Parallel()

	nodes := []PackageNode{
		{ImportPath: "test.com/api/handler", Cluster: "api/handler"},
		{ImportPath: "test.com/api/middleware", Cluster: "api/middleware"},
		{ImportPath: "test.com/service/user", Cluster: "service/user"},
		{ImportPath: "test.com/service/product", Cluster: "service/product"},
		{ImportPath: "test.com/repository/db", Cluster: "repository/db"},
		{ImportPath: "test.com/repository/cache", Cluster: "repository/cache"},
		{ImportPath: "test.com/cmd/server", Cluster: "cmd"},
	}

	baseClusters := []string{"api/handler", "api/middleware", "service/user", "service/product", "repository/db", "repository/cache", "cmd"}

	clusters := applyClusteringHeuristics(nodes, baseClusters)

	assert.True(t, len(clusters) > 0, "expected at least one cluster")

	// Check that layer-based clustering is detected
	hasPresentationLayer := false
	hasBusinessLayer := false
	hasDataLayer := false

	for _, cluster := range clusters {
		if strings.Contains(cluster, "api") || strings.Contains(cluster, "cmd") {
			hasPresentationLayer = true
		}
		if strings.Contains(cluster, "service") {
			hasBusinessLayer = true
		}
		if strings.Contains(cluster, "repository") {
			hasDataLayer = true
		}
	}

	assert.True(t, hasPresentationLayer, "expected to detect presentation layer")
	assert.True(t, hasBusinessLayer, "expected to detect business layer")
	assert.True(t, hasDataLayer, "expected to detect data layer")
}

// TestTarjanAlgorithm tests Tarjan's algorithm for cycle detection.
func TestTarjanAlgorithm(t *testing.T) {
	t.Parallel()

	// Create a simple graph with a cycle: A -> B -> C -> A
	nodes := []PackageNode{
		{ImportPath: "test.com/a"},
		{ImportPath: "test.com/b"},
		{ImportPath: "test.com/c"},
	}
	edges := []ImportEdge{
		{From: "test.com/a", To: "test.com/b"},
		{From: "test.com/b", To: "test.com/c"},
		{From: "test.com/c", To: "test.com/a"},
	}

	cycles := DetectCyclesTarjan(nodes, edges)

	assert.Greater(t, len(cycles), 0, "expected to detect cycle")
	assert.Equal(t, 3, len(cycles[0]), "cycle should have 3 nodes")
}

// TestGeneratedFileDetection tests detection of generated files.
func TestGeneratedFileDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"protobuf", "test.pb.go", true},
		{"stringer", "string.go", false},
		{"gen tool", "generated.go", false}, // Not in our patterns
		{"wire", "wire_gen.go", true},
		{"mock", "mock_user.go", true},
		{"sqlc", "db.go", false}, // Not specific enough
		{"normal", "handler.go", false},
		{"sqlboiler", "tbls.go", true},
		{"deepcopy", "deepcopy.go", true},
		{"facade", "facade.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeneratedFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// BenchmarkCycleDetection benchmarks cycle detection algorithms.
func BenchmarkCycleDetection(b *testing.B) {
	// Create a large graph for benchmarking
	nodes := make([]PackageNode, 1000)
	for i := 0; i < 1000; i++ {
		nodes[i] = PackageNode{ImportPath: fmt.Sprintf("test.com/pkg%d", i)}
	}

	edges := make([]ImportEdge, 2000)
	for i := 0; i < 1000; i++ {
		edges[i] = ImportEdge{
			From: fmt.Sprintf("test.com/pkg%d", i),
			To:   fmt.Sprintf("test.com/pkg%d", (i+1)%1000),
		}
	}

	b.Run("Tarjan", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = DetectCyclesTarjan(nodes, edges)
		}
	})

	b.Run("DFS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = DetectCyclesDFS(nodes, edges)
		}
	})
}
