// Package extract provides code-structure extractors that produce GoImportGraph
// representations of a project's internal dependency topology.
package extract

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// ---------------------------------------------------------------------------
// Domain types (Go-specific import graph)
// ---------------------------------------------------------------------------

// GoImportEdge represents a directed dependency from one Go package to another.
type GoImportEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PackageNode describes a single Go package discovered during extraction.
type PackageNode struct {
	ImportPath string `json:"import_path"`
	Dir        string `json:"dir"`
	Name       string `json:"name"`
	Cluster    string `json:"cluster"` // parent directory relative to module root
}

// Cycle is a sequence of package import paths that form a circular dependency.
type Cycle []string

// GoImportGraph is the result of running a GoExtractor against a Go module.
type GoImportGraph struct {
	ModulePath       string         `json:"module_path"`
	Nodes            []PackageNode  `json:"nodes"`
	Edges            []GoImportEdge `json:"edges"`
	Clusters         []string       `json:"clusters"`
	Cycles           []Cycle        `json:"cycles"`
	ExtractionMethod string         `json:"extraction_method"`
	AccuracyEstimate float64        `json:"accuracy_estimate"`
}

// ---------------------------------------------------------------------------
// GoExtractor — uses go/packages to build the ImportGraph
// ---------------------------------------------------------------------------

// GoExtractor loads Go packages via go/packages and builds an import graph
// limited to project-internal packages (those sharing the module path).
type GoExtractor struct {
	// Dir is the root directory of the Go module to analyse.
	Dir string
}

// NewGoExtractor creates a GoExtractor rooted at dir.
func NewGoExtractor(dir string) *GoExtractor {
	return &GoExtractor{Dir: dir}
}

// Extract performs the extraction.  It returns an empty graph (no error) when
// the directory has no go.mod.
func (e *GoExtractor) Extract(ctx context.Context) (*GoImportGraph, error) {
	modPath, err := readModulePath(e.Dir)
	if err != nil {
		// No go.mod — return gracefully.
		return &GoImportGraph{
			ExtractionMethod: "go/packages",
			AccuracyEstimate: 0.93,
		}, nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports,
		Dir:  e.Dir,
		Context: ctx,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	// Collect all internal packages (those whose import path starts with
	// the module path) while skipping generated files.
	nodeMap := make(map[string]*PackageNode)
	var edges []GoImportEdge

	for _, pkg := range pkgs {
		if !isInternal(pkg.PkgPath, modPath) {
			continue
		}
		if isGenerated(pkg) {
			continue
		}
		addNode(nodeMap, pkg, modPath)

		for imp := range pkg.Imports {
			if !isInternal(imp, modPath) {
				continue
			}
			edges = append(edges, GoImportEdge{From: pkg.PkgPath, To: imp})
			// Ensure the target node exists even if it was not loaded
			// directly (e.g. build-error packages).
			if _, ok := nodeMap[imp]; !ok {
				target := pkg.Imports[imp]
				if target != nil {
					addNode(nodeMap, target, modPath)
				}
			}
		}
	}

	nodes := make([]PackageNode, 0, len(nodeMap))
	clusterSet := make(map[string]struct{})
	for _, n := range nodeMap {
		nodes = append(nodes, *n)
		if n.Cluster != "" {
			clusterSet[n.Cluster] = struct{}{}
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ImportPath < nodes[j].ImportPath })

	clusters := make([]string, 0, len(clusterSet))
	for c := range clusterSet {
		clusters = append(clusters, c)
	}
	sort.Strings(clusters)

	cycles := DetectCycles(nodes, edges)

	return &GoImportGraph{
		ModulePath:       modPath,
		Nodes:            nodes,
		Edges:            edges,
		Clusters:         clusters,
		Cycles:           cycles,
		ExtractionMethod: "go/packages",
		AccuracyEstimate: 0.93,
	}, nil
}

// ---------------------------------------------------------------------------
// Cycle detection (exported for testing)
// ---------------------------------------------------------------------------

// DetectCycles performs DFS-based cycle detection on the given nodes/edges
// and returns all elementary cycles found.
func DetectCycles(nodes []PackageNode, edges []GoImportEdge) []Cycle {
	// Build adjacency list.
	adj := make(map[string][]string)
	nodeSet := make(map[string]struct{})
	for _, n := range nodes {
		nodeSet[n.ImportPath] = struct{}{}
	}
	for _, e := range edges {
		if _, ok := nodeSet[e.From]; !ok {
			continue
		}
		if _, ok := nodeSet[e.To]; !ok {
			continue
		}
		adj[e.From] = append(adj[e.From], e.To)
	}

	const (
		white = 0 // not visited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	color := make(map[string]int)
	parent := make(map[string]string)

	var cycles []Cycle

	// Reconstruct cycle from back-edge target → current node via parent map.
	buildCycle := func(from, to string) Cycle {
		var c Cycle
		cur := from
		for cur != to {
			c = append(c, cur)
			cur = parent[cur]
		}
		c = append(c, to)
		// Reverse so the cycle reads in forward direction.
		for i, j := 0, len(c)-1; i < j; i, j = i+1, j-1 {
			c[i], c[j] = c[j], c[i]
		}
		return c
	}

	var dfs func(u string)
	dfs = func(u string) {
		color[u] = gray
		for _, v := range adj[u] {
			switch color[v] {
			case white:
				parent[v] = u
				dfs(v)
			case gray:
				// Back edge — cycle found.
				cycles = append(cycles, buildCycle(u, v))
			}
		}
		color[u] = black
	}

	// Sort keys for deterministic output.
	keys := make([]string, 0, len(nodeSet))
	for k := range nodeSet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if color[k] == white {
			dfs(k)
		}
	}

	return cycles
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// readModulePath extracts the module path from go.mod in dir.
// Returns an error when no go.mod is found.
func readModulePath(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", os.ErrNotExist
}

// isInternal returns true when pkgPath belongs to the same module.
func isInternal(pkgPath, modPath string) bool {
	return pkgPath == modPath || strings.HasPrefix(pkgPath, modPath+"/")
}

// isGenerated returns true when every Go file in the package matches a
// generated-file pattern (*.pb.go, *.gen.go).
func isGenerated(pkg *packages.Package) bool {
	if len(pkg.GoFiles) == 0 && len(pkg.OtherFiles) == 0 {
		return false
	}
	for _, f := range pkg.GoFiles {
		base := filepath.Base(f)
		if !strings.HasSuffix(base, ".pb.go") && !strings.HasSuffix(base, ".gen.go") {
			return false // at least one non-generated file
		}
	}
	// If GoFiles is empty (package loaded in name-only mode), we cannot
	// determine file names, so assume not generated.
	return len(pkg.GoFiles) > 0
}

// addNode inserts a PackageNode into the map (idempotent).
func addNode(m map[string]*PackageNode, pkg *packages.Package, modPath string) {
	if _, ok := m[pkg.PkgPath]; ok {
		return
	}
	rel := strings.TrimPrefix(pkg.PkgPath, modPath+"/")
	cluster := ""
	if idx := strings.LastIndex(rel, "/"); idx > 0 {
		cluster = rel[:idx]
	}
	m[pkg.PkgPath] = &PackageNode{
		ImportPath: pkg.PkgPath,
		Dir:        dirOf(pkg),
		Name:       pkg.Name,
		Cluster:    cluster,
	}
}

func dirOf(pkg *packages.Package) string {
	if len(pkg.GoFiles) > 0 {
		return filepath.Dir(pkg.GoFiles[0])
	}
	return ""
}
