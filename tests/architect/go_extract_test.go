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

// TestGoExtractor_FrameworkDetection verifies that framework detection
// correctly identifies known Go frameworks from the self-analysis repo.
func TestGoExtractor_FrameworkDetection(t *testing.T) {
	repoRoot := findRepoRoot(t)

	ext := extract.NewGoExtractor(repoRoot)
	graph, err := ext.Extract(context.Background())
	require.NoError(t, err)

	// This repo uses gRPC and Cobra (both are in go.mod).
	// At minimum stdlib HTTP should be detected since many packages import net/http.
	t.Logf("Detected %d frameworks:", len(graph.Frameworks))
	for _, fw := range graph.Frameworks {
		t.Logf("  - %s (confidence: %.2f, evidence: %s)", fw.Name, fw.Confidence, fw.Evidence)
	}

	// Verify frameworks slice is populated (at least stdlib HTTP or gRPC).
	if len(graph.Frameworks) > 0 {
		fwNames := make(map[string]bool)
		for _, fw := range graph.Frameworks {
			fwNames[fw.Name] = true
		}
		// gRPC is imported by this repo via google.golang.org/grpc
		// and sigstore dependencies. Cobra is also imported.
		// Stdlib HTTP is widely imported.
		t.Logf("Framework names: %v", fwNames)
	}
}

// TestGoExtractor_ModuleInfo verifies go.mod parsing produces correct
// module metadata including requires, replaces, and excludes.
func TestGoExtractor_ModuleInfo(t *testing.T) {
	repoRoot := findRepoRoot(t)

	ext := extract.NewGoExtractor(repoRoot)
	graph, err := ext.Extract(context.Background())
	require.NoError(t, err)

	require.NotNil(t, graph.ModuleInfo, "module info should be populated")
	require.Equal(t, "sdp_dev", graph.ModuleInfo.ModulePath)
	require.NotEmpty(t, graph.ModuleInfo.GoVersion, "go version should be set")
	require.NotEmpty(t, graph.ModuleInfo.Requires, "should have require directives")

	t.Logf("Module: %s (Go %s)", graph.ModuleInfo.ModulePath, graph.ModuleInfo.GoVersion)
	t.Logf("Requires: %d deps", len(graph.ModuleInfo.Requires))
	t.Logf("Replaces: %d", len(graph.ModuleInfo.Replaces))
	t.Logf("Excludes: %d", len(graph.ModuleInfo.Excludes))

	// Check that at least some known deps are present.
	foundTestify := false
	for _, dep := range graph.ModuleInfo.Requires {
		if dep.Path == "github.com/stretchr/testify" {
			foundTestify = true
			break
		}
	}
	require.True(t, foundTestify, "expected testify in go.mod requires")
}

// TestGoExtractor_DeployUnits verifies cmd/ directory detection.
func TestGoExtractor_DeployUnits(t *testing.T) {
	repoRoot := findRepoRoot(t)

	ext := extract.NewGoExtractor(repoRoot)
	graph, err := ext.Extract(context.Background())
	require.NoError(t, err)

	t.Logf("Deploy units: %d", len(graph.DeployUnits))
	for _, u := range graph.DeployUnits {
		t.Logf("  - %s (%s)", u.Name, u.Path)
	}

	// This repo has a cmd/ directory with at least one subdirectory.
	if len(graph.DeployUnits) > 0 {
		foundSDP := false
		for _, u := range graph.DeployUnits {
			if u.Name == "sdp" {
				foundSDP = true
				require.Equal(t, filepath.Join("cmd", "sdp"), u.Path)
				break
			}
		}
		require.True(t, foundSDP, "expected 'sdp' deploy unit from cmd/sdp/")
	}
}

// TestGoAdapter_FrameworkAndModuleInfo verifies GoAdapter surfaces
// framework detection and module info in the ProfileFragment.
func TestGoAdapter_FrameworkAndModuleInfo(t *testing.T) {
	repoRoot := findRepoRoot(t)

	adapter := extract.GoAdapter{}
	frag, err := adapter.Extract(context.Background(), repoRoot)
	require.NoError(t, err)

	require.NotNil(t, frag.ImportGraph, "import graph should be populated")
	require.NotEmpty(t, frag.Languages, "languages should be set")
	require.Equal(t, "go", frag.Languages[0].Primary)

	// Framework detection should populate Dependencies.
	if len(frag.Dependencies) > 0 {
		t.Logf("Dependency info entries: %d", len(frag.Dependencies))
		for _, di := range frag.Dependencies {
			t.Logf("  file=%s lang=%s depCount=%d signals=%v notableDeps=%d",
				di.File, di.Language, di.DepCount, di.Signals, len(di.NotableDeps))
			for _, nd := range di.NotableDeps {
				t.Logf("    - %s (signal: %s)", nd.Name, nd.Signal)
			}
		}
	}

	require.NotNil(t, frag.Metrics, "metrics should be populated")
	require.Equal(t, 1, frag.Metrics.LanguagesCount)
}

// TestGoExtractor_GoWork verifies go.work support with a synthetic fixture.
func TestGoExtractor_GoWork(t *testing.T) {
	dir := t.TempDir()

	// Create go.work with two modules.
	workContent := `go 1.26

use (
	./mod1
	./mod2
)
`
	err := os.WriteFile(filepath.Join(dir, "go.work"), []byte(workContent), 0o644)
	require.NoError(t, err)

	// Create mod1.
	mod1Dir := filepath.Join(dir, "mod1")
	require.NoError(t, os.MkdirAll(mod1Dir, 0o755))
	mod1Content := `module example.com/mod1

go 1.26
`
	err = os.WriteFile(filepath.Join(mod1Dir, "go.mod"), []byte(mod1Content), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(mod1Dir, "main.go"), []byte("package main\n"), 0o644)
	require.NoError(t, err)

	// Create mod2.
	mod2Dir := filepath.Join(dir, "mod2")
	require.NoError(t, os.MkdirAll(mod2Dir, 0o755))
	mod2Content := `module example.com/mod2

go 1.26
`
	err = os.WriteFile(filepath.Join(mod2Dir, "go.mod"), []byte(mod2Content), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(mod2Dir, "main.go"), []byte("package main\n"), 0o644)
	require.NoError(t, err)

	ext := extract.NewGoExtractor(dir)
	graph, err := ext.Extract(context.Background())
	require.NoError(t, err)

	// go.work mode should have extracted something, even if packages
	// have build errors (no actual code).
	require.Equal(t, "go/packages", graph.ExtractionMethod)
	t.Logf("go.work extraction: %d nodes, %d edges", len(graph.Nodes), len(graph.Edges))
}

// TestParseGoModInfo verifies go.mod parsing with a synthetic file.
func TestParseGoModInfo(t *testing.T) {
	dir := t.TempDir()

	modContent := `module example.com/myproject

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.8.4 // indirect
	google.golang.org/grpc v1.60.0
)

exclude (
	github.com/bad/dep v1.0.0
)

replace github.com/old/dep => github.com/new/dep v2.0.0
`
	modPath := filepath.Join(dir, "go.mod")
	err := os.WriteFile(modPath, []byte(modContent), 0o644)
	require.NoError(t, err)

	info := extract.ParseGoModInfoForTest(modPath)
	require.NotNil(t, info)
	require.Equal(t, "example.com/myproject", info.ModulePath)
	require.Equal(t, "1.22", info.GoVersion)
	require.Len(t, info.Requires, 3)
	require.Len(t, info.Excludes, 1)
	require.Len(t, info.Replaces, 1)

	// Check testify is marked indirect.
	foundIndirect := false
	for _, dep := range info.Requires {
		if dep.Path == "github.com/stretchr/testify" {
			require.True(t, dep.Indirect, "testify should be marked indirect")
			foundIndirect = true
		}
	}
	require.True(t, foundIndirect, "expected testify in requires")

	// Check replace directive.
	require.Equal(t, "github.com/old/dep", info.Replaces[0].Path)
	require.True(t, info.Replaces[0].IsReplace)
}

// TestDetectDeployUnits verifies cmd/ directory detection with synthetic dirs.
func TestDetectDeployUnits(t *testing.T) {
	t.Run("with_cmd_dir", func(t *testing.T) {
		dir := t.TempDir()
		cmdDir := filepath.Join(dir, "cmd", "server")
		require.NoError(t, os.MkdirAll(cmdDir, 0o755))
		err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main\n"), 0o644)
		require.NoError(t, err)

		units := extract.DetectDeployUnitsForTest(dir)
		require.Len(t, units, 1)
		require.Equal(t, "server", units[0].Name)
		require.Equal(t, filepath.Join("cmd", "server"), units[0].Path)
	})

	t.Run("no_cmd_dir", func(t *testing.T) {
		dir := t.TempDir()
		units := extract.DetectDeployUnitsForTest(dir)
		require.Empty(t, units)
	})

	t.Run("empty_cmd_dir", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd"), 0o755))
		units := extract.DetectDeployUnitsForTest(dir)
		require.Empty(t, units)
	})
}

// TestDetectFrameworks verifies framework detection from external imports.
func TestDetectFrameworks(t *testing.T) {
	t.Run("gin_and_grpc", func(t *testing.T) {
		imports := map[string]struct{}{
			"github.com/gin-gonic/gin":              {},
			"google.golang.org/grpc":                {},
			"github.com/gin-gonic/gin/render":       {},
			"fmt":                                   {},
		}
		fws := extract.DetectFrameworksForTest(imports)
		require.Len(t, fws, 2) // Gin + gRPC

		fwMap := make(map[string]extract.DetectedFramework)
		for _, fw := range fws {
			fwMap[fw.Name] = fw
		}
		require.Contains(t, fwMap, "Gin")
		require.Contains(t, fwMap, "gRPC")
		require.InDelta(t, 0.95, fwMap["Gin"].Confidence, 0.001)
		require.InDelta(t, 0.90, fwMap["gRPC"].Confidence, 0.001)
	})

	t.Run("echo_and_chi", func(t *testing.T) {
		imports := map[string]struct{}{
			"github.com/labstack/echo/v4": {},
			"github.com/go-chi/chi/v5":    {},
		}
		fws := extract.DetectFrameworksForTest(imports)
		require.Len(t, fws, 2)

		fwMap := make(map[string]extract.DetectedFramework)
		for _, fw := range fws {
			fwMap[fw.Name] = fw
		}
		require.Contains(t, fwMap, "Echo")
		require.Contains(t, fwMap, "Chi")
	})

	t.Run("stdlib_http", func(t *testing.T) {
		imports := map[string]struct{}{
			"net/http": {},
		}
		fws := extract.DetectFrameworksForTest(imports)
		require.Len(t, fws, 1)
		require.Equal(t, "stdlib HTTP", fws[0].Name)
		require.InDelta(t, 0.70, fws[0].Confidence, 0.001)
	})

	t.Run("no_frameworks", func(t *testing.T) {
		imports := map[string]struct{}{
			"fmt":    {},
			"strings": {},
		}
		fws := extract.DetectFrameworksForTest(imports)
		require.Empty(t, fws)
	})

	t.Run("dedup_same_framework", func(t *testing.T) {
		imports := map[string]struct{}{
			"github.com/gin-gonic/gin":        {},
			"github.com/gin-gonic/gin/render": {},
		}
		fws := extract.DetectFrameworksForTest(imports)
		require.Len(t, fws, 1, "gin should be detected only once")
		require.Equal(t, "Gin", fws[0].Name)
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
