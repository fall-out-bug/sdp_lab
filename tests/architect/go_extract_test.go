package architect_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/architect/extract"

	"github.com/stretchr/testify/require"
)

// TestGoExtractor_SelfAnalysis runs the extractor against the SDP repo's
// internal/architect/ subtree and verifies that it discovers packages.
func TestGoExtractor_SelfAnalysis(t *testing.T) {
	// Find the repo root by walking up from the test file location.
	repoRoot := findRepoRoot(t)

	ext := extract.NewGoExtractor(repoRoot)
	graph, err := ext.Extract(context.Background())
	require.NoError(t, err)

	require.Equal(t, "go/packages", graph.ExtractionMethod)
	require.InDelta(t, 0.93, graph.AccuracyEstimate, 0.001)
	require.Equal(t, "sdp_dev", graph.ModulePath)

	// The repo itself has packages — we should find at least the
	// internal/architect/extract package.
	require.NotEmpty(t, graph.Nodes, "expected at least one node")

	found := false
	for _, n := range graph.Nodes {
		if n.ImportPath == "sdp_dev/internal/architect/extract" {
			found = true
			require.Equal(t, "extract", n.Name)
			require.Equal(t, "internal/architect", n.Cluster)
			break
		}
	}
	require.True(t, found, "expected to find sdp_dev/internal/architect/extract package")

	// There must be clusters (parent directories).
	require.NotEmpty(t, graph.Clusters, "expected at least one cluster")

	t.Logf("Discovered %d nodes, %d edges, %d clusters, %d cycles",
		len(graph.Nodes), len(graph.Edges), len(graph.Clusters), len(graph.Cycles))
}

// TestGoExtractor_NoGoMod verifies graceful handling of a directory that
// has no go.mod file.
func TestGoExtractor_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	// Write a plain .go file (no go.mod).
	err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	require.NoError(t, err)

	ext := extract.NewGoExtractor(dir)
	graph, err := ext.Extract(context.Background())
	require.NoError(t, err, "should not error on missing go.mod")

	require.Empty(t, graph.Nodes, "no nodes expected without go.mod")
	require.Empty(t, graph.Edges, "no edges expected without go.mod")
	require.Empty(t, graph.ModulePath, "module path should be empty")
	require.Equal(t, "go/packages", graph.ExtractionMethod)
	require.InDelta(t, 0.93, graph.AccuracyEstimate, 0.001)
}

// TestGoExtractor_CircularDetection is a unit test for DetectCycles using
// a synthetic import graph that contains known cycles.
func TestGoExtractor_CircularDetection(t *testing.T) {
	t.Run("no_cycles", func(t *testing.T) {
		nodes := []extract.PackageNode{
			{ImportPath: "m/a"},
			{ImportPath: "m/b"},
			{ImportPath: "m/c"},
		}
		edges := []extract.GoImportEdge{
			{From: "m/a", To: "m/b"},
			{From: "m/b", To: "m/c"},
		}
		cycles := extract.DetectCycles(nodes, edges)
		require.Empty(t, cycles, "DAG should have no cycles")
	})

	t.Run("simple_cycle", func(t *testing.T) {
		nodes := []extract.PackageNode{
			{ImportPath: "m/a"},
			{ImportPath: "m/b"},
		}
		edges := []extract.GoImportEdge{
			{From: "m/a", To: "m/b"},
			{From: "m/b", To: "m/a"},
		}
		cycles := extract.DetectCycles(nodes, edges)
		require.Len(t, cycles, 1, "expected exactly one cycle")
		require.Contains(t, cycles[0], "m/a")
		require.Contains(t, cycles[0], "m/b")
	})

	t.Run("triangle_cycle", func(t *testing.T) {
		nodes := []extract.PackageNode{
			{ImportPath: "m/a"},
			{ImportPath: "m/b"},
			{ImportPath: "m/c"},
		}
		edges := []extract.GoImportEdge{
			{From: "m/a", To: "m/b"},
			{From: "m/b", To: "m/c"},
			{From: "m/c", To: "m/a"},
		}
		cycles := extract.DetectCycles(nodes, edges)
		require.NotEmpty(t, cycles, "expected at least one cycle in triangle")
		// The cycle should contain all three nodes.
		allPaths := make(map[string]bool)
		for _, c := range cycles {
			for _, p := range c {
				allPaths[p] = true
			}
		}
		require.True(t, allPaths["m/a"], "cycle must contain m/a")
		require.True(t, allPaths["m/b"], "cycle must contain m/b")
		require.True(t, allPaths["m/c"], "cycle must contain m/c")
	})

	t.Run("self_loop", func(t *testing.T) {
		nodes := []extract.PackageNode{
			{ImportPath: "m/x"},
		}
		edges := []extract.GoImportEdge{
			{From: "m/x", To: "m/x"},
		}
		cycles := extract.DetectCycles(nodes, edges)
		require.NotEmpty(t, cycles, "self-loop should be detected as a cycle")
	})

	t.Run("disconnected_graph_with_cycle", func(t *testing.T) {
		nodes := []extract.PackageNode{
			{ImportPath: "m/a"},
			{ImportPath: "m/b"},
			{ImportPath: "m/c"},
			{ImportPath: "m/d"},
		}
		edges := []extract.GoImportEdge{
			{From: "m/a", To: "m/b"},
			// c <-> d cycle, disconnected from a->b
			{From: "m/c", To: "m/d"},
			{From: "m/d", To: "m/c"},
		}
		cycles := extract.DetectCycles(nodes, edges)
		require.Len(t, cycles, 1, "only the c<->d cycle should be detected")
		require.Contains(t, cycles[0], "m/c")
		require.Contains(t, cycles[0], "m/d")
	})
}

// findRepoRoot walks up from the working directory looking for go.mod.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root (no go.mod found in parents)")
		}
		dir = parent
	}
}
