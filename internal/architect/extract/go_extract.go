// Package extract provides code-structure extractors that produce GoImportGraph
// representations of a project's internal dependency topology.
package extract

import (
	"context"
	"fmt"
	"os"
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

// DetectedFramework records a Go framework or library detected from imports.
type DetectedFramework struct {
	Name       string  `json:"name"`
	ImportPath string  `json:"import_path"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// GoModuleDep represents a single dependency from go.mod.
type GoModuleDep struct {
	Path      string `json:"path"`
	Version   string `json:"version"`
	Indirect  bool   `json:"indirect"`
	IsReplace bool   `json:"is_replace,omitempty"`
}

// GoModuleInfo holds parsed go.mod metadata.
type GoModuleInfo struct {
	ModulePath string         `json:"module_path"`
	GoVersion  string         `json:"go_version"`
	Requires   []GoModuleDep  `json:"requires"`
	Replaces   []GoModuleDep  `json:"replaces"`
	Excludes   []string       `json:"excludes"`
}

// DeployUnit represents a deployable binary detected from cmd/ directories.
type DeployUnit struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GoImportGraph is the result of running a GoExtractor against a Go module.
type GoImportGraph struct {
	ModulePath       string             `json:"module_path"`
	Nodes            []PackageNode      `json:"nodes"`
	Edges            []GoImportEdge     `json:"edges"`
	Clusters         []string           `json:"clusters"`
	Cycles           []Cycle            `json:"cycles"`
	Frameworks       []DetectedFramework `json:"frameworks,omitempty"`
	ModuleInfo       *GoModuleInfo      `json:"module_info,omitempty"`
	DeployUnits      []DeployUnit       `json:"deploy_units,omitempty"`
	ExtractionMethod string             `json:"extraction_method"`
	AccuracyEstimate float64            `json:"accuracy_estimate"`
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
	// Check for go.work first — if present, extract from each workspace module.
	if workMods := detectGoWorkModules(e.Dir); len(workMods) > 0 {
		return e.extractFromWork(ctx, workMods)
	}

	modPath, err := readModulePath(e.Dir)
	if err != nil {
		// No go.mod — return gracefully.
		return &GoImportGraph{
			ExtractionMethod: "go/packages",
			AccuracyEstimate: 0.93,
		}, nil
	}

	graph, err := e.extractModule(ctx, e.Dir, modPath)
	if err != nil {
		return nil, err
	}
	return graph, nil
}

// extractFromWork runs extraction across all modules listed in go.work and
// merges the results into a single GoImportGraph.
func (e *GoExtractor) extractFromWork(ctx context.Context, modules []goWorkModule) (*GoImportGraph, error) {
	merged := &GoImportGraph{
		ExtractionMethod: "go/packages",
		AccuracyEstimate: 0.93,
	}
	nodeSet := make(map[string]struct{})
	edgeSet := make(map[GoImportEdge]struct{})
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
		for _, e := range graph.Edges {
			if _, ok := edgeSet[e]; !ok {
				edgeSet[e] = struct{}{}
				merged.Edges = append(merged.Edges, e)
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

	merged.Cycles = DetectCycles(merged.Nodes, merged.Edges)

	return merged, nil
}

// extractModule extracts the import graph from a single Go module.
func (e *GoExtractor) extractModule(ctx context.Context, dir, modPath string) (*GoImportGraph, error) {
	cfg := &packages.Config{
		Mode:    packages.NeedName | packages.NeedImports,
		Dir:     dir,
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
	externalImports := make(map[string]struct{})

	for _, pkg := range pkgs {
		if !isInternal(pkg.PkgPath, modPath) {
			continue
		}
		if isGenerated(pkg) {
			continue
		}
		addNode(nodeMap, pkg, modPath)

		for imp := range pkg.Imports {
			if isInternal(imp, modPath) {
				edges = append(edges, GoImportEdge{From: pkg.PkgPath, To: imp})
				// Ensure the target node exists even if it was not loaded
				// directly (e.g. build-error packages).
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

	cycles := DetectCycles(nodes, edges)

	// Framework detection from external imports.
	frameworks := detectFrameworks(externalImports)

	// Parse go.mod for module graph analysis.
	moduleInfo := parseGoModInfo(filepath.Join(dir, "go.mod"))

	// Detect cmd/ deploy units.
	deployUnits := detectDeployUnits(dir)

	return &GoImportGraph{
		ModulePath:       modPath,
		Nodes:            nodes,
		Edges:            edges,
		Clusters:         clusters,
		Cycles:           cycles,
		Frameworks:       frameworks,
		ModuleInfo:       moduleInfo,
		DeployUnits:      deployUnits,
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
// Framework detection
// ---------------------------------------------------------------------------

// goFrameworkSignals maps import path prefixes to framework metadata.
var goFrameworkSignals = map[string]DetectedFramework{
	"github.com/gin-gonic/gin":         {Name: "Gin", Confidence: 0.95, Evidence: "gin import detected"},
	"github.com/labstack/echo/v4":      {Name: "Echo", Confidence: 0.95, Evidence: "echo import detected"},
	"github.com/labstack/echo":         {Name: "Echo", Confidence: 0.90, Evidence: "echo import detected (unversioned)"},
	"github.com/go-chi/chi/v5":         {Name: "Chi", Confidence: 0.95, Evidence: "chi import detected"},
	"github.com/go-chi/chi":            {Name: "Chi", Confidence: 0.90, Evidence: "chi import detected"},
	"google.golang.org/grpc":           {Name: "gRPC", Confidence: 0.90, Evidence: "grpc import detected"},
	"github.com/gorilla/mux":           {Name: "Gorilla Mux", Confidence: 0.90, Evidence: "gorilla/mux import detected"},
	"github.com/go-kratos/kratos/v2":   {Name: "Kratos", Confidence: 0.90, Evidence: "kratos import detected"},
	"github.com/gofiber/fiber/v2":      {Name: "Fiber", Confidence: 0.90, Evidence: "fiber import detected"},
	"net/http":                         {Name: "stdlib HTTP", Confidence: 0.70, Evidence: "net/http import detected"},
	"github.com/spf13/cobra":           {Name: "Cobra CLI", Confidence: 0.90, Evidence: "cobra import detected"},
	"github.com/urfave/cli":            {Name: "urfave/cli", Confidence: 0.90, Evidence: "urfave/cli import detected"},
	"github.com/go-telegram-bot-api/telegram-bot-api": {Name: "Telegram Bot", Confidence: 0.85, Evidence: "telegram bot api import detected"},
}

// detectFrameworks scans external imports for known Go framework signals.
func detectFrameworks(externalImports map[string]struct{}) []DetectedFramework {
	var frameworks []DetectedFramework
	seen := make(map[string]bool)

	for imp := range externalImports {
		for prefix, fw := range goFrameworkSignals {
			if seen[fw.Name] {
				continue
			}
			if imp == prefix || strings.HasPrefix(imp, prefix+"/") {
				fw.ImportPath = imp
				frameworks = append(frameworks, fw)
				seen[fw.Name] = true
			}
		}
	}

	sort.Slice(frameworks, func(i, j int) bool {
		return frameworks[i].Name < frameworks[j].Name
	})
	return frameworks
}

// ---------------------------------------------------------------------------
// go.mod parsing (module graph analysis)
// ---------------------------------------------------------------------------

// parseGoModInfo reads a go.mod file and extracts module metadata including
// require, replace, and exclude directives.
func parseGoModInfo(path string) *GoModuleInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	info := &GoModuleInfo{}
	var inRequire bool
	var inExclude bool

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		// Module path.
		if strings.HasPrefix(line, "module ") {
			info.ModulePath = strings.TrimSpace(strings.TrimPrefix(line, "module"))
			continue
		}

		// Go version.
		if strings.HasPrefix(line, "go ") {
			info.GoVersion = strings.TrimSpace(strings.TrimPrefix(line, "go"))
			continue
		}

		// Require block start.
		if line == "require (" || line == "require(" {
			inRequire = true
			continue
		}

		// Require block end.
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		// Single-line require.
		if strings.HasPrefix(line, "require ") && !strings.HasPrefix(line, "require (") {
		 dep := parseGoModDep(strings.TrimPrefix(line, "require "))
		 info.Requires = append(info.Requires, dep)
		 continue
		}

		// Inside require block.
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			info.Requires = append(info.Requires, parseGoModDep(line))
			continue
		}

		// Exclude block start.
		if line == "exclude (" || line == "exclude(" {
			inExclude = true
			continue
		}

		// Exclude block end.
		if inExclude && line == ")" {
			inExclude = false
			continue
		}

		// Single-line exclude.
		if strings.HasPrefix(line, "exclude ") && !strings.HasPrefix(line, "exclude (") {
			info.Excludes = append(info.Excludes, strings.Fields(strings.TrimPrefix(line, "exclude"))[0])
			continue
		}

		// Inside exclude block.
		if inExclude && line != "" && !strings.HasPrefix(line, "//") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				info.Excludes = append(info.Excludes, fields[0])
			}
			continue
		}

		// Replace directive (single-line only for simplicity).
		if strings.HasPrefix(line, "replace ") && !strings.HasPrefix(line, "replace (") {
			// replace old => new or replace old v1 => new v2
			content := strings.TrimSpace(strings.TrimPrefix(line, "replace"))
			if arrowIdx := strings.Index(content, " => "); arrowIdx >= 0 {
				oldPart := strings.TrimSpace(content[:arrowIdx])
				newPart := strings.TrimSpace(content[arrowIdx+4:])
				info.Replaces = append(info.Replaces, GoModuleDep{
					Path:      oldPart,
					Version:   newPart,
					IsReplace: true,
				})
			}
		}
	}

	return info
}

// parseGoModDep parses a single dependency line from go.mod.
// Format: "path version" or "path version // indirect"
func parseGoModDep(line string) GoModuleDep {
	line = strings.TrimSpace(line)
	indirect := false
	// Remove trailing comments.
	if idx := strings.Index(line, "//"); idx >= 0 {
		comment := strings.TrimSpace(line[idx+2:])
		line = strings.TrimSpace(line[:idx])
		indirect = strings.Contains(comment, "indirect")
	}
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return GoModuleDep{
			Path:      fields[0],
			Version:   fields[1],
			Indirect:  indirect,
		}
	}
	if len(fields) == 1 {
		return GoModuleDep{
			Path:     fields[0],
			Indirect: indirect,
		}
	}
	return GoModuleDep{}
}

// ---------------------------------------------------------------------------
// go.work support
// ---------------------------------------------------------------------------

// goWorkModule represents a single module in a go.work file.
type goWorkModule struct {
	Dir string
}

// detectGoWorkModules checks for a go.work file and returns the listed modules.
// Returns nil if no go.work exists.
func detectGoWorkModules(rootDir string) []goWorkModule {
	data, err := os.ReadFile(filepath.Join(rootDir, "go.work"))
	if err != nil {
		return nil
	}

	var modules []goWorkModule
	var inUse bool

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		if line == "use (" || line == "use(" {
			inUse = true
			continue
		}
		if inUse && line == ")" {
			inUse = false
			continue
		}
		if inUse && line != "" && !strings.HasPrefix(line, "//") {
			dir := strings.Trim(line, "\"")
			absDir := dir
			if !filepath.IsAbs(dir) {
				absDir = filepath.Join(rootDir, dir)
			}
			modules = append(modules, goWorkModule{Dir: absDir})
			continue
		}

		// Single-line use: use ./foo or use ("./foo")
		if strings.HasPrefix(line, "use ") && !strings.HasPrefix(line, "use (") {
			dir := strings.TrimSpace(strings.TrimPrefix(line, "use"))
			dir = strings.Trim(dir, "\"()")
			absDir := dir
			if !filepath.IsAbs(dir) {
				absDir = filepath.Join(rootDir, dir)
			}
			modules = append(modules, goWorkModule{Dir: absDir})
		}
	}

	return modules
}

// ---------------------------------------------------------------------------
// cmd/ deploy unit detection
// ---------------------------------------------------------------------------

// detectDeployUnits scans the cmd/ directory for subdirectories, each of
// which represents a deployable Go binary.
func detectDeployUnits(rootDir string) []DeployUnit {
	cmdDir := filepath.Join(rootDir, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil
	}

	var units []DeployUnit
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Verify the directory has at least one .go file.
		if !hasGoFiles(filepath.Join(cmdDir, e.Name())) {
			continue
		}
		units = append(units, DeployUnit{
			Name: e.Name(),
			Path: filepath.Join("cmd", e.Name()),
		})
	}

	sort.Slice(units, func(i, j int) bool { return units[i].Name < units[j].Name })
	return units
}

// hasGoFiles returns true if the directory contains at least one .go file.
func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
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

// formatCyclePath returns a human-readable representation of a cycle.
func formatCyclePath(c Cycle) string {
	return strings.Join(c, " -> ") + " -> " + c[0]
}

// FormatGoModuleInfo returns a human-readable summary of GoModuleInfo.
func FormatGoModuleInfo(info *GoModuleInfo) string {
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

// ---------------------------------------------------------------------------
// Test-only exported wrappers
// ---------------------------------------------------------------------------

// ParseGoModInfoForTest exports parseGoModInfo for testing.
func ParseGoModInfoForTest(path string) *GoModuleInfo {
	return parseGoModInfo(path)
}

// DetectDeployUnitsForTest exports detectDeployUnits for testing.
func DetectDeployUnitsForTest(rootDir string) []DeployUnit {
	return detectDeployUnits(rootDir)
}

// DetectFrameworksForTest exports detectFrameworks for testing.
func DetectFrameworksForTest(externalImports map[string]struct{}) []DetectedFramework {
	return detectFrameworks(externalImports)
}
