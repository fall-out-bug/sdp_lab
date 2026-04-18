// Package golang implements code-structure extraction for Go projects using go/packages.
package golang

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Known blind spots (documented per WS-05 acceptance criteria):
//   - Build tags (//go:build) may cause packages to be skipped when tags are
//     not satisfied during analysis.
//   - CGo imports (import "C") are filtered out by the isInternal check.
//   - Dot imports (import . "pkg") are recorded but the implicit namespace
//     pollution is not tracked.
//   - go generate output is partially handled by isGenerated() but only
//     checks file suffixes (.pb.go, .gen.go).

// ImportEdge represents a directed dependency from one Go package to another.
type ImportEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PackageNode describes a single Go package discovered during extraction.
type PackageNode struct {
	ImportPath  string   `json:"import_path"`
	Dir         string   `json:"dir"`
	Name        string   `json:"name"`
	Cluster     string   `json:"cluster"` // parent directory relative to module root
	IsGenerated bool     `json:"is_generated"`
	Interfaces  []string `json:"interfaces,omitempty"`  // interfaces defined in this package
	Implements  []string `json:"implements,omitempty"`  // interfaces implemented by this package
}

// Cycle is a sequence of package import paths that form a circular dependency.
type Cycle []string

// DetectedFramework records a Go framework or library detected from imports.
type DetectedFramework struct {
	Name       string  `json:"name"`
	ImportPath string  `json:"import_path"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// ModuleDep represents a single dependency from go.mod.
type ModuleDep struct {
	Path      string `json:"path"`
	Version   string `json:"version"`
	Indirect  bool   `json:"indirect"`
	IsReplace bool   `json:"is_replace,omitempty"`
}

// ModuleInfo holds parsed go.mod metadata.
type ModuleInfo struct {
	ModulePath string      `json:"module_path"`
	GoVersion  string      `json:"go_version"`
	Requires   []ModuleDep `json:"requires"`
	Replaces   []ModuleDep `json:"replaces"`
	Excludes   []string    `json:"excludes"`
}

// DeployUnit represents a deployable binary detected from cmd/ directories.
type DeployUnit struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	HasMain     bool   `json:"has_main"`
	PackageName string `json:"package_name"`
}

// ImportGraph is the result of running an Extractor against a Go module.
type ImportGraph struct {
	ModulePath       string             `json:"module_path"`
	Nodes            []PackageNode      `json:"nodes"`
	Edges            []ImportEdge       `json:"edges"`
	Clusters         []string           `json:"clusters"`
	Cycles           []Cycle            `json:"cycles"`
	Frameworks       []DetectedFramework `json:"frameworks,omitempty"`
	ModuleInfo       *ModuleInfo        `json:"module_info,omitempty"`
	DeployUnits      []DeployUnit       `json:"deploy_units,omitempty"`
	ExtractionMethod string             `json:"extraction_method"`
	AccuracyEstimate float64            `json:"accuracy_estimate"`
}

// Extractor loads Go packages via go/packages and builds an import graph
// limited to project-internal packages (those sharing the module path).
type Extractor struct {
	// Dir is the root directory of the Go module to analyse.
	Dir string

	// ModulePath is the Go module path (e.g., "github.com/user/project").
	// If empty, it will be read from go.mod.
	ModulePath string

	// IncludeTests enables extraction of test packages.
	IncludeTests bool

	// EnableInterfaceDetection enables interface implementation mapping.
	EnableInterfaceDetection bool
}

// NewExtractor creates an Extractor rooted at dir.
func NewExtractor(dir string) *Extractor {
	return &Extractor{Dir: dir}
}

// Extract performs the extraction. It returns an empty graph (no error) when
// the directory has no go.mod.
func (e *Extractor) Extract(ctx context.Context) (*ImportGraph, error) {
	// Check for go.work first — if present, extract from each workspace module.
	if workMods := detectGoWorkModules(e.Dir); len(workMods) > 0 {
		return e.extractFromWork(ctx, workMods)
	}

	modPath, err := readModulePath(e.Dir)
	if err != nil {
		// No go.mod — return gracefully.
		return &ImportGraph{
			ExtractionMethod: "go/packages",
			AccuracyEstimate: 0.93,
		}, nil
	}

	if e.ModulePath != "" {
		modPath = e.ModulePath
	}

	graph, err := e.extractModule(ctx, e.Dir, modPath)
	if err != nil {
		return nil, err
	}
	return graph, nil
}

// extractFromWork runs extraction across all modules listed in go.work and
// merges the results into a single ImportGraph.
func (e *Extractor) extractFromWork(ctx context.Context, modules []goWorkModule) (*ImportGraph, error) {
	merged := &ImportGraph{
		ExtractionMethod: "go/packages",
		AccuracyEstimate: 0.93,
	}
	nodeSet := make(map[string]struct{})
	edgeSet := make(map[ImportEdge]struct{})
	clusterSet := make(map[string]struct{})
	fwSet := make(map[string]DetectedFramework)

	for _, m := range modules {
		modPath, err := readModulePath(m.Dir)
		if err != nil {
			continue // skip modules without go.mod
		}
		graph, err := e.extractModule(ctx, m.Dir, modPath)
		if err != nil {
			continue // non-fatal for individual modules
		}
		if merged.ModulePath == "" {
			merged.ModulePath = modPath
		}
		for _, n := range graph.Nodes {
			if _, ok := nodeSet[n.ImportPath]; !ok {
				nodeSet[n.ImportPath] = struct{}{}
				merged.Nodes = append(merged.Nodes, n)
			}
		}
		for _, edge := range graph.Edges {
			if _, ok := edgeSet[edge]; !ok {
				edgeSet[edge] = struct{}{}
				merged.Edges = append(merged.Edges, edge)
			}
		}
		for _, c := range graph.Clusters {
			clusterSet[c] = struct{}{}
		}
		for _, fw := range graph.Frameworks {
			if _, ok := fwSet[fw.ImportPath]; !ok {
				fwSet[fw.ImportPath] = fw
			}
		}
		for _, u := range graph.DeployUnits {
			merged.DeployUnits = append(merged.DeployUnits, u)
		}
		if graph.ModuleInfo != nil {
			if merged.ModuleInfo == nil {
				merged.ModuleInfo = graph.ModuleInfo
			} else {
				merged.ModuleInfo.Requires = append(merged.ModuleInfo.Requires, graph.ModuleInfo.Requires...)
				merged.ModuleInfo.Replaces = append(merged.ModuleInfo.Replaces, graph.ModuleInfo.Replaces...)
				merged.ModuleInfo.Excludes = append(merged.ModuleInfo.Excludes, graph.ModuleInfo.Excludes...)
			}
		}
	}

	clusters := make([]string, 0, len(clusterSet))
	for c := range clusterSet {
		clusters = append(clusters, c)
	}
	sort.Strings(clusters)
	merged.Clusters = clusters

	merged.Cycles = DetectCyclesTarjan(merged.Nodes, merged.Edges)

	return merged, nil
}

// extractModule extracts the import graph from a single Go module.
func (e *Extractor) extractModule(ctx context.Context, dir, modPath string) (*ImportGraph, error) {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax,
		Dir:   dir,
		Tests: e.IncludeTests,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	// Collect all internal packages
	nodeMap := make(map[string]*PackageNode)
	var edges []ImportEdge
	externalImports := make(map[string]struct{})

	for _, pkg := range pkgs {
		if !isInternal(pkg.PkgPath, modPath) {
			continue
		}
		if isGenerated(pkg) {
			continue
		}
		addNode(nodeMap, pkg, modPath)

		// Build edges from import graph
		for imp := range pkg.Imports {
			if isInternal(imp, modPath) {
				edges = append(edges, ImportEdge{From: pkg.PkgPath, To: imp})
				// Ensure the target node exists
				if _, ok := nodeMap[imp]; !ok {
					target := pkg.Imports[imp]
					if target != nil {
						addNode(nodeMap, target, modPath)
					}
				}
			} else {
				externalImports[imp] = struct{}{}
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

	// Use Tarjan's algorithm for cycle detection
	cycles := DetectCyclesTarjan(nodes, edges)

	// Framework detection from external imports
	frameworks := detectFrameworks(externalImports)

	// Parse go.mod for module graph analysis
	moduleInfo := parseModuleInfo(filepath.Join(dir, "go.mod"))

	// Detect cmd/ deploy units with enhanced information
	deployUnits := detectDeployUnits(dir)

	// Apply clustering heuristics
	enhancedClusters := applyClusteringHeuristics(nodes, clusters)
	sort.Strings(enhancedClusters)

	return &ImportGraph{
		ModulePath:       modPath,
		Nodes:            nodes,
		Edges:            edges,
		Clusters:         enhancedClusters,
		Cycles:           cycles,
		Frameworks:       frameworks,
		ModuleInfo:       moduleInfo,
		DeployUnits:      deployUnits,
		ExtractionMethod: "go/packages",
		AccuracyEstimate: 0.93,
	}, nil
}

// formatCyclePath returns a human-readable representation of a cycle.
func formatCyclePath(c Cycle) string {
	return strings.Join(c, " -> ") + " -> " + c[0]
}

// FormatModuleInfo returns a human-readable summary of ModuleInfo.
func FormatModuleInfo(info *ModuleInfo) string {
	if info == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Module: %s", info.ModulePath)
	if info.GoVersion != "" {
		fmt.Fprintf(&b, " (Go %s)", info.GoVersion)
	}
	fmt.Fprintf(&b, "\nRequires: %d", len(info.Requires))
	if len(info.Replaces) > 0 {
		fmt.Fprintf(&b, "\nReplaces: %d", len(info.Replaces))
	}
	if len(info.Excludes) > 0 {
		fmt.Fprintf(&b, "\nExcludes: %d", len(info.Excludes))
	}
	return b.String()
}
