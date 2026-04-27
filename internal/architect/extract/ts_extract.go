package extract

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// ---------------------------------------------------------------------------
// Known blind spots (documented per WS-08 acceptance criteria):
//   - Path aliases (@/...) are collected from tsconfig.json but not resolved
//     during import graph construction.
//   - Barrel re-export chains (index.ts -> index.ts) are detected as re-exports
//     but the transitive closure is not computed.
//   - Dynamic import() with variable arguments is not handled.
//   - CommonJS require() with computed paths (require(variable)) is not handled.
//   - Webpack module federation is out of scope.
//   - Multi-line import statements spanned across lines may be missed by
//     line-based regex parsing.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Domain types (TS/JS-specific import graph)
// ---------------------------------------------------------------------------

// TSImportKind classifies the type of import statement.
type TSImportKind string

const (
	TSImportESModule   TSImportKind = "es_module"
	TSImportCommonJS   TSImportKind = "commonjs"
	TSImportSideEffect TSImportKind = "side_effect"
	TSImportReExport   TSImportKind = "re_export"
	TSImportDynamic    TSImportKind = "dynamic"
)

// TSImportEdge represents a directed import from one file to another.
type TSImportEdge struct {
	From     string       `json:"from"`
	To       string       `json:"to"`
	Kind     TSImportKind `json:"kind"`
	Line     int          `json:"line,omitempty"`
	Resolved bool         `json:"resolved"` // true if specifier was resolved to a local path
}

// TSPackageNode describes a single TS/JS module discovered during extraction.
type TSPackageNode struct {
	Path        string `json:"path"`
	RelPath     string `json:"rel_path"`
	IsBarrel    bool   `json:"is_barrel"`
	IsGenerated bool   `json:"is_generated"`
	Cluster     string `json:"cluster"`
}

// TSDetectedFramework records a TS/JS framework detected from imports and config.
type TSDetectedFramework struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
}

// TSDependencyEntry represents a single dependency from package.json.
type TSDependencyEntry struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Dev     bool   `json:"dev,omitempty"`
}

// TSWorkspaceInfo describes a detected monorepo workspace package.
type TSWorkspaceInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// TSPathAlias maps a tsconfig path alias prefix to its resolved directory.
type TSPathAlias struct {
	Alias  string `json:"alias"`
	Target string `json:"target"`
}

// TSImportGraph is the result of running TSExtractor against a TS/JS project.
type TSImportGraph struct {
	Nodes            []TSPackageNode       `json:"nodes"`
	Edges            []TSImportEdge        `json:"edges"`
	Clusters         []string              `json:"clusters"`
	BarrelFiles      []string              `json:"barrel_files"`
	Frameworks       []TSDetectedFramework `json:"frameworks,omitempty"`
	Dependencies     []TSDependencyEntry   `json:"dependencies,omitempty"`
	Workspaces       []TSWorkspaceInfo     `json:"workspaces,omitempty"`
	PathAliases      []TSPathAlias         `json:"path_aliases,omitempty"`
	IsMonorepo       bool                  `json:"is_monorepo"`
	MonorepoTool     string                `json:"monorepo_tool,omitempty"`
	ExtractionMethod string                `json:"extraction_method"`
	AccuracyEstimate float64               `json:"accuracy_estimate"`
}

// ---------------------------------------------------------------------------
// TSExtractor — regex-based TS/JS import graph extractor
// ---------------------------------------------------------------------------

// TSExtractor builds a TSImportGraph from a TypeScript/JavaScript project root.
// It implements the architect.Extractor interface.
type TSExtractor struct{}

// NewTSExtractor creates a new TSExtractor.
func NewTSExtractor() *TSExtractor {
	return &TSExtractor{}
}

// Name returns the extractor identifier.
func (TSExtractor) Name() string { return "typescript" }

// Extract implements architect.Extractor.  Returns an empty fragment (no error)
// when the directory has no TS/JS markers.
func (e *TSExtractor) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	if !e.detect(repoRoot) {
		return &architect.ProfileFragment{}, nil
	}

	graph, err := e.extractGraph(ctx, repoRoot)
	if err != nil {
		return nil, err
	}

	return convertTSImportGraph(graph, repoRoot), nil
}

// detect returns true if rootDir appears to contain a TypeScript/JavaScript project.
func (e *TSExtractor) detect(rootDir string) bool {
	for _, f := range []string{"tsconfig.json", "package.json", "jsconfig.json"} {
		if fileExists(filepath.Join(rootDir, f)) {
			return true
		}
	}
	// Fallback: scan for TS/JS source files.
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if tsExtensions[filepath.Ext(path)] {
			found = true
		}
		return nil
	})
	return found
}

// ---------------------------------------------------------------------------
// Internal extraction
// ---------------------------------------------------------------------------

// extractGraph performs the full extraction pipeline.
func (e *TSExtractor) extractGraph(ctx context.Context, rootDir string) (*TSImportGraph, error) {
	graph := &TSImportGraph{
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.65,
	}

	// 1. Parse tsconfig.json for path aliases.
	graph.PathAliases = parseTSConfigAliases(rootDir)

	// 2. Parse package.json for dependencies and workspaces.
	graph.Dependencies = parsePackageJSONDeps(rootDir)
	graph.Workspaces, graph.IsMonorepo, graph.MonorepoTool = detectMonorepo(rootDir)

	// 3. Walk source files: extract imports, detect barrel files, build clusters.
	nodeMap := make(map[string]*TSPackageNode)
	var edges []TSImportEdge
	externalSpecifiers := make(map[string]struct{})
	barrelSet := make(map[string]bool)

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if !tsExtensions[ext] {
			return nil
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			rel = path
		}

		// Detect barrel file.
		isBarrel := isBarrelFile(path, rel)

		// Detect generated file.
		isGen := isTSGenerated(path)

		// Compute cluster (parent directory relative to root).
		cluster := computeTSCluster(rel)

		nodeMap[rel] = &TSPackageNode{
			Path:        path,
			RelPath:     rel,
			IsBarrel:    isBarrel,
			IsGenerated: isGen,
			Cluster:     cluster,
		}

		if isBarrel {
			barrelSet[rel] = true
		}

		// Extract imports from the file.
		fileEdges, exts := extractFileImports(path, rel, rootDir)
		edges = append(edges, fileEdges...)
		for s := range exts {
			externalSpecifiers[s] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build sorted node list.
	nodes := make([]TSPackageNode, 0, len(nodeMap))
	clusterSet := make(map[string]struct{})
	for _, n := range nodeMap {
		nodes = append(nodes, *n)
		if n.Cluster != "" {
			clusterSet[n.Cluster] = struct{}{}
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].RelPath < nodes[j].RelPath })

	clusters := sortedKeys(clusterSet)

	// Barrel files list.
	barrels := sortedKeys(barrelSet)

	// Framework detection.
	graph.Nodes = nodes
	graph.Edges = edges
	graph.Clusters = clusters
	graph.BarrelFiles = barrels
	graph.Frameworks = detectTSFrameworksV2(rootDir, graph.Dependencies, nodeMap)

	return graph, nil
}

// ---------------------------------------------------------------------------
// Import extraction
// ---------------------------------------------------------------------------

// Import regexes — compiled once.
var (
	reESImport     = regexp.MustCompile(`import\s+(?:.*?)\s+from\s+['"]([^'"]+)['"]`)
	reSideEffect   = regexp.MustCompile(`^import\s+['"]([^'"]+)['"]\s*;?\s*$`)
	reCommonJS     = regexp.MustCompile(`(?:const|let|var)\s+\w+\s*=\s*require\(\s*['"]([^'"]+)['"]\s*\)`)
	reReExport     = regexp.MustCompile(`export\s+(?:.*?)\s+from\s+['"]([^'"]+)['"]`)
	reDynamicImport = regexp.MustCompile(`import\(\s*['"]([^'"]+)['"]\s*\)`)
)

// extractFileImports reads a single source file and returns import edges and
// a set of external specifiers.
func extractFileImports(path, relPath, rootDir string) ([]TSImportEdge, map[string]struct{}) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	var edges []TSImportEdge
	externals := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments.
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			continue
		}

		type match struct {
			specifier string
			kind      TSImportKind
		}

		var matches []match

		// Re-export: export ... from "X"
		if m := reReExport.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportReExport})
		}

		// ES module: import X from "Y"
		if m := reESImport.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportESModule})
		}

		// Side-effect: import "X"
		if m := reSideEffect.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportSideEffect})
		}

		// Dynamic import: import("X")
		if m := reDynamicImport.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportDynamic})
		}

		// CommonJS: const X = require("Y")
		if m := reCommonJS.FindStringSubmatch(line); m != nil {
			matches = append(matches, match{m[1], TSImportCommonJS})
		}

		for _, mt := range matches {
			spec := mt.specifier
			resolved := resolveSpecifier(spec, relPath, rootDir)
			isExternal := !isLocalSpecifier(spec)

			edge := TSImportEdge{
				From:     relPath,
				To:       resolved,
				Kind:     mt.kind,
				Line:     lineNo,
				Resolved: !isExternal,
			}

			if isExternal {
				externals[spec] = struct{}{}
				// For external packages, use the specifier as-is for the target.
				edge.To = spec
			}

			edges = append(edges, edge)
		}
	}

	return edges, externals
}

// isLocalSpecifier returns true if the import specifier refers to a local file.
func isLocalSpecifier(spec string) bool {
	return strings.HasPrefix(spec, "./") ||
		strings.HasPrefix(spec, "../") ||
		strings.HasPrefix(spec, "/")
}

// resolveSpecifier attempts to resolve an import specifier to a relative path.
// For local specifiers it resolves relative to the importing file's directory.
// For package specifiers it returns the package name (first path segment).
func resolveSpecifier(spec, fromRelPath, rootDir string) string {
	if !isLocalSpecifier(spec) {
		// Package specifier: return the package name (first segment, possibly scoped).
		parts := strings.SplitN(spec, "/", -1)
		if strings.HasPrefix(spec, "@") && len(parts) >= 2 {
			return parts[0] + "/" + parts[1] // @scope/package
		}
		return parts[0]
	}

	// Local import: resolve relative to the importing file's directory.
	fromDir := filepath.Dir(fromRelPath)
	resolved := filepath.Join(fromDir, spec)
	// Clean the path.
	resolved = filepath.Clean(resolved)
	return resolved
}

// ---------------------------------------------------------------------------
// Barrel file detection
// ---------------------------------------------------------------------------

// barrelFileNames are filenames treated as barrel (re-export) files.
var barrelFileNames = map[string]bool{
	"index.ts":  true,
	"index.tsx": true,
	"index.js":  true,
	"index.jsx": true,
}

// isBarrelFile checks if a file is a barrel file (index.ts/js) that contains
// re-export statements.
func isBarrelFile(absPath, relPath string) bool {
	base := filepath.Base(relPath)
	if !barrelFileNames[base] {
		return false
	}
	// Verify that the file contains at least one re-export.
	data, err := os.ReadFile(absPath)
	if err != nil {
		return false
	}
	return reReExport.Match(data)
}

// ---------------------------------------------------------------------------
// Monorepo detection
// ---------------------------------------------------------------------------

// detectMonorepo checks for monorepo markers and returns workspace packages.
func detectMonorepo(rootDir string) ([]TSWorkspaceInfo, bool, string) {
	var workspaces []TSWorkspaceInfo
	isMonorepo := false
	tool := ""

	// Check package.json workspaces.
	pkgWorkspaces := parsePackageJSONWorkspaces(filepath.Join(rootDir, "package.json"))
	if len(pkgWorkspaces) > 0 {
		isMonorepo = true
		tool = "npm"
		workspaces = expandWorkspacePatterns(rootDir, pkgWorkspaces)
	}

	// Check yarn.lock — indicates yarn workspaces.
	if fileExists(filepath.Join(rootDir, "yarn.lock")) && len(pkgWorkspaces) > 0 {
		tool = "yarn"
	}

	// Check pnpm-lock.yaml + pnpm-workspace.yaml.
	if fileExists(filepath.Join(rootDir, "pnpm-lock.yaml")) {
		pnpmWorkspaces := parsePnpmWorkspace(filepath.Join(rootDir, "pnpm-workspace.yaml"))
		if len(pnpmWorkspaces) > 0 {
			isMonorepo = true
			tool = "pnpm"
			ws := expandWorkspacePatterns(rootDir, pnpmWorkspaces)
			workspaces = mergeWorkspaces(workspaces, ws)
		}
	}

	// Check lerna.json.
	if fileExists(filepath.Join(rootDir, "lerna.json")) {
		isMonorepo = true
		if tool == "" {
			tool = "lerna"
		}
	}

	// Check turborepo (turbo.json).
	if fileExists(filepath.Join(rootDir, "turbo.json")) {
		isMonorepo = true
		if tool == "" || tool == "npm" {
			tool = "turborepo"
		}
	}

	return workspaces, isMonorepo, tool
}

// parsePackageJSONWorkspaces reads package.json and returns workspace glob patterns.
func parsePackageJSONWorkspaces(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pkg struct {
		Workspaces interface{} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	return parseWorkspaces(pkg.Workspaces)
}

// parsePnpmWorkspace reads a pnpm-workspace.yaml file for workspace packages.
// Simple line-based parsing (no full YAML parser).
func parsePnpmWorkspace(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var patterns []string
	inPackages := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "packages:" {
			inPackages = true
			continue
		}
		if inPackages {
			if line == "" || (!strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "\"")) {
				if len(patterns) > 0 {
					break
				}
				continue
			}
			pattern := strings.TrimPrefix(line, "- ")
			pattern = strings.Trim(pattern, "\"' ")
			if pattern != "" {
				patterns = append(patterns, pattern)
			}
		}
	}
	return patterns
}

// expandWorkspacePatterns expands workspace glob patterns (e.g. "packages/*")
// into actual directory paths by listing the parent directory.
func expandWorkspacePatterns(rootDir string, patterns []string) []TSWorkspaceInfo {
	var ws []TSWorkspaceInfo
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		// Only handle simple star patterns like "packages/*".
		if !strings.Contains(pattern, "*") {
			// Direct path reference.
			absPath := filepath.Join(rootDir, pattern)
			pkgJSON := filepath.Join(absPath, "package.json")
			if fileExists(pkgJSON) {
				name := readPkgName(pkgJSON)
				rel := pattern
				if !seen[rel] {
					seen[rel] = true
					ws = append(ws, TSWorkspaceInfo{Name: name, Path: rel})
				}
			}
			continue
		}

		// Glob pattern: split on "/*" to find parent directory.
		parts := strings.SplitN(pattern, "*", 2)
		parentDir := filepath.Join(rootDir, parts[0])
		entries, err := os.ReadDir(parentDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			relPath := filepath.Join(parts[0], e.Name())
			pkgJSON := filepath.Join(rootDir, relPath, "package.json")
			if fileExists(pkgJSON) {
				name := readPkgName(pkgJSON)
				if !seen[relPath] {
					seen[relPath] = true
					ws = append(ws, TSWorkspaceInfo{Name: name, Path: relPath})
				}
			}
		}
	}
	sort.Slice(ws, func(i, j int) bool { return ws[i].Path < ws[j].Path })
	return ws
}

// readPkgName reads the "name" field from a package.json.
func readPkgName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return pkg.Name
}

// mergeWorkspaces merges two workspace slices, deduplicating by path.
func mergeWorkspaces(a, b []TSWorkspaceInfo) []TSWorkspaceInfo {
	seen := make(map[string]bool)
	for _, w := range a {
		seen[w.Path] = true
	}
	for _, w := range b {
		if !seen[w.Path] {
			seen[w.Path] = true
			a = append(a, w)
		}
	}
	return a
}

// ---------------------------------------------------------------------------
// tsconfig.json path alias parsing
// ---------------------------------------------------------------------------

// parseTSConfigAliases reads tsconfig.json and extracts path aliases.
func parseTSConfigAliases(rootDir string) []TSPathAlias {
	path := filepath.Join(rootDir, "tsconfig.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var cfg struct {
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	var aliases []TSPathAlias
	for alias, targets := range cfg.CompilerOptions.Paths {
		if len(targets) == 0 {
			continue
		}
		cleanAlias := strings.TrimSuffix(alias, "*")
		cleanTarget := strings.TrimSuffix(targets[0], "*")
		aliases = append(aliases, TSPathAlias{
			Alias:  cleanAlias,
			Target: cleanTarget,
		})
	}
	sort.Slice(aliases, func(i, j int) bool { return aliases[i].Alias < aliases[j].Alias })
	return aliases
}

// ---------------------------------------------------------------------------
// package.json dependency parsing
// ---------------------------------------------------------------------------

// parsePackageJSONDeps reads package.json and returns all dependencies.
func parsePackageJSONDeps(rootDir string) []TSDependencyEntry {
	path := filepath.Join(rootDir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	var deps []TSDependencyEntry
	for name, version := range pkg.Dependencies {
		deps = append(deps, TSDependencyEntry{Name: name, Version: version, Dev: false})
	}
	for name, version := range pkg.DevDependencies {
		deps = append(deps, TSDependencyEntry{Name: name, Version: version, Dev: true})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	return deps
}

// ---------------------------------------------------------------------------
// Framework detection (v2)
// ---------------------------------------------------------------------------

// tsFrameworkSignals maps dependency names to framework metadata.
var tsFrameworkSignals = []struct {
	Name       string
	DepName    string
	Confidence float64
	FileCheck  string // optional config file basename
}{
	{Name: "Next.js", DepName: "next", Confidence: 0.90, FileCheck: "next.config.js"},
	{Name: "React", DepName: "react", Confidence: 0.90},
	{Name: "Express", DepName: "express", Confidence: 0.90},
	{Name: "NestJS", DepName: "@nestjs/core", Confidence: 0.90},
	{Name: "Vue", DepName: "vue", Confidence: 0.90},
	{Name: "Angular", DepName: "@angular/core", Confidence: 0.90, FileCheck: "angular.json"},
	{Name: "Svelte", DepName: "svelte", Confidence: 0.90},
	{Name: "Nuxt", DepName: "nuxt", Confidence: 0.90},
	{Name: "Fastify", DepName: "fastify", Confidence: 0.85},
	{Name: "Koa", DepName: "koa", Confidence: 0.85},
	{Name: "Hapi", DepName: "@hapi/hapi", Confidence: 0.85},
}

// detectTSFrameworksV2 scans dependencies, config files, and source patterns.
func detectTSFrameworksV2(rootDir string, deps []TSDependencyEntry, nodeMap map[string]*TSPackageNode) []TSDetectedFramework {
	depSet := make(map[string]string, len(deps))
	for _, d := range deps {
		depSet[d.Name] = d.Version
	}

	var frameworks []TSDetectedFramework
	seen := make(map[string]bool)

	for _, sig := range tsFrameworkSignals {
		if seen[sig.Name] {
			continue
		}

		confidence := 0.0
		var evidence []string

		// Check dependency presence.
		if _, ok := depSet[sig.DepName]; ok {
			confidence += 0.5
			evidence = append(evidence, sig.DepName+" in dependencies")
		}

		// Check config file.
		if sig.FileCheck != "" {
			// Check common variants (.js, .mjs, .cjs, .ts).
			base := strings.TrimSuffix(sig.FileCheck, filepath.Ext(sig.FileCheck))
			for _, ext := range []string{".js", ".mjs", ".cjs", ".ts"} {
				if fileExists(filepath.Join(rootDir, base+ext)) {
					confidence += 0.3
					evidence = append(evidence, base+ext+" config found")
					break
				}
			}
		}

		// Framework-specific additional signals.
		switch sig.Name {
		case "Next.js":
			if dirExists(filepath.Join(rootDir, "pages")) || dirExists(filepath.Join(rootDir, "app")) {
				confidence += 0.2
				evidence = append(evidence, "pages/ or app/ directory present")
			}
		case "React":
			if hasFileWithExtension(rootDir, ".tsx") || hasFileWithExtension(rootDir, ".jsx") {
				confidence += 0.2
				evidence = append(evidence, "TSX/JSX files found")
			}
		case "NestJS":
			if scanForPattern(rootDir, reNestModule) || scanForPattern(rootDir, reNestController) {
				confidence += 0.3
				evidence = append(evidence, "@Module/@Controller decorators found")
			}
		case "Express":
			if scanForPattern(rootDir, reExpressApp) {
				confidence += 0.3
				evidence = append(evidence, "app.get/post/use patterns found")
			}
		case "Svelte":
			if hasFileWithExtension(rootDir, ".svelte") {
				confidence += 0.3
				evidence = append(evidence, ".svelte files found")
			}
			if fileExists(filepath.Join(rootDir, "svelte.config.js")) {
				confidence += 0.2
				evidence = append(evidence, "svelte.config.js found")
			}
		case "Vue":
			if hasFileWithExtension(rootDir, ".vue") {
				confidence += 0.2
				evidence = append(evidence, ".vue files found")
			}
		case "Angular":
			if fileExists(filepath.Join(rootDir, "angular.json")) {
				confidence += 0.3
				evidence = append(evidence, "angular.json found")
			}
		}

		// Only report frameworks with at least some evidence.
		if confidence > 0.2 {
			frameworks = append(frameworks, TSDetectedFramework{
				Name:       sig.Name,
				Confidence: minFloat(confidence, 1.0),
				Evidence:   strings.Join(evidence, "; "),
			})
			seen[sig.Name] = true
		}
	}

	sort.Slice(frameworks, func(i, j int) bool {
		return frameworks[i].Name < frameworks[j].Name
	})
	return frameworks
}

// scanForPattern walks TS/JS files and returns true if any line matches the regex.
func scanForPattern(rootDir string, re *regexp.Regexp) bool {
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !tsExtensions[filepath.Ext(path)] {
			return nil
		}
		f, fErr := os.Open(path)
		if fErr != nil {
			return nil
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if re.MatchString(scanner.Text()) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

// Framework detection regexes (kept for source-pattern scanning).
var (
	reNestModule     = regexp.MustCompile(`@Module\s*\(`)
	reNestController = regexp.MustCompile(`@Controller\s*\(`)
	reExpressApp     = regexp.MustCompile(`(?:app|router)\.(get|post|put|delete|patch|use)\s*\(`)
)

// ---------------------------------------------------------------------------
// Cluster computation (C4 Level 3 grouping)
// ---------------------------------------------------------------------------

// computeTSCluster returns the cluster ID for a file based on its directory.
// Files in the same directory subtree share a cluster.
// Examples:
//   - "src/components/Button.tsx" -> "src/components"
//   - "pages/index.tsx"          -> "pages"
//   - "utils.ts"                 -> ""
func computeTSCluster(relPath string) string {
	dir := filepath.Dir(relPath)
	if dir == "." {
		return ""
	}
	return dir
}

// ---------------------------------------------------------------------------
// Generated file detection
// ---------------------------------------------------------------------------

// tsGeneratedSuffixes lists file suffixes that indicate generated TS/JS files.
var tsGeneratedSuffixes = []string{
	".generated.ts",
	".generated.js",
	".gen.ts",
	".gen.js",
	".pb.ts",
	".pb.js",
	".d.ts", // type declarations are typically generated
}

// tsGeneratedPaths lists path substrings that indicate generated code.
var tsGeneratedPaths = []string{
	"__generated__/",
	"generated/",
}

// isTSGenerated returns true if the file appears to be generated.
func isTSGenerated(path string) bool {
	base := filepath.Base(path)
	for _, suffix := range tsGeneratedSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	for _, substr := range tsGeneratedPaths {
		if strings.Contains(path, substr) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Conversion: TSImportGraph -> ProfileFragment
// ---------------------------------------------------------------------------

// convertTSImportGraph converts a TSImportGraph into a ProfileFragment.
func convertTSImportGraph(graph *TSImportGraph, rootDir string) *architect.ProfileFragment {
	if graph == nil {
		return &architect.ProfileFragment{}
	}

	// Determine languages.
	languages := []string{"typescript", "javascript"}
	hasJSX := false
	for _, n := range graph.Nodes {
		if strings.HasSuffix(n.RelPath, ".tsx") || strings.HasSuffix(n.RelPath, ".jsx") {
			hasJSX = true
			break
		}
	}
	if hasJSX {
		languages = []string{"typescript", "javascript", "jsx"}
	}

	frag := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{{
			Primary: "typescript",
			All:     languages,
		}},
	}

	// Build ImportGraph.
	importGraph := &architect.ImportGraph{
		ExtractionMethod: graph.ExtractionMethod,
		AccuracyEstimate: graph.AccuracyEstimate,
		Nodes:            len(graph.Nodes),
		Edges:            len(graph.Edges),
	}

	// Convert clusters.
	pkgToCluster := make(map[string]string)
	for _, cluster := range graph.Clusters {
		ic := architect.ImportCluster{ID: cluster}
		// Collect all packages in this cluster.
		for _, node := range graph.Nodes {
			if node.Cluster == cluster {
				ic.Packages = append(ic.Packages, node.RelPath)
				pkgToCluster[node.RelPath] = cluster
			}
		}
		importGraph.Clusters = append(importGraph.Clusters, ic)
	}

	// Count internal/external edges per cluster.
	for i, cluster := range importGraph.Clusters {
		pkgSet := make(map[string]bool, len(cluster.Packages))
		for _, p := range cluster.Packages {
			pkgSet[p] = true
		}
		for _, edge := range graph.Edges {
			if pkgSet[edge.From] {
				if pkgSet[edge.To] {
					importGraph.Clusters[i].InternalEdges++
				} else {
					importGraph.Clusters[i].ExternalEdges++
				}
			}
		}
	}

	// Detect circular dependencies from the edge list.
	importGraph.CircularDependencies = detectTSCircularDeps(graph.Edges)

	frag.ImportGraph = importGraph

	// Build DependencyInfo.
	if len(graph.Dependencies) > 0 {
		depInfo := architect.DependencyInfo{
			File:     "package.json",
			Language: "javascript",
			DepCount: len(graph.Dependencies),
		}
		for _, d := range graph.Dependencies {
			signal := detectTSDepSignal(d.Name)
			depInfo.NotableDeps = append(depInfo.NotableDeps, architect.NotableDep{
				Name:   d.Name,
				FoundIn: 1,
				Signal: signal,
			})
		}
		frag.Dependencies = []architect.DependencyInfo{depInfo}
	}

	// Build metrics.
	frag.Metrics = &architect.CodeMetrics{
		LanguagesCount: len(languages),
	}

	return frag
}

// detectTSCircularDeps finds circular dependencies among local files.
func detectTSCircularDeps(edges []TSImportEdge) []architect.CircularDep {
	// Build adjacency list from resolved (local) edges only.
	adj := make(map[string][]string)
	nodeSet := make(map[string]struct{})
	for _, e := range edges {
		if !e.Resolved {
			continue
		}
		nodeSet[e.From] = struct{}{}
		nodeSet[e.To] = struct{}{}
		adj[e.From] = append(adj[e.From], e.To)
	}

	// DFS-based cycle detection.
	const (
		white = 0
		gray  = 1
		black = 2
	)

	color := make(map[string]int)
	parent := make(map[string]string)
	seen := make(map[string]bool) // deduplicate cycles
	var cycles []architect.CircularDep

	var dfs func(u string)
	dfs = func(u string) {
		color[u] = gray
		for _, v := range adj[u] {
			switch color[v] {
			case white:
				parent[v] = u
				dfs(v)
			case gray:
				// Back edge: reconstruct cycle.
				a, b := u, v
				if a > b {
					a, b = b, a
				}
				key := a + "|" + b
				if !seen[key] {
					seen[key] = true
					cycles = append(cycles, architect.CircularDep{
						A:        u,
						B:        v,
						EdgeType: "ts_import",
					})
				}
			}
		}
		color[u] = black
	}

	keys := sortedKeys(nodeSet)
	for _, k := range keys {
		if color[k] == white {
			dfs(k)
		}
	}

	return cycles
}

// tsDepSignals maps dependency name patterns to architectural signals.
var tsDepSignals = map[string]string{
	"react":          "ui_framework",
	"next":           "ssr_framework",
	"vue":            "ui_framework",
	"angular":        "ui_framework",
	"svelte":         "ui_framework",
	"express":        "web_framework",
	"fastify":        "web_framework",
	"koa":            "web_framework",
	"@nestjs":        "web_framework",
	"prisma":         "orm",
	"typeorm":        "orm",
	"sequelize":      "orm",
	"mongoose":       "odm",
	"graphql":        "graphql",
	"apollo":         "graphql",
	"redis":          "cache",
	"ioredis":        "cache",
	"kafka":          "event_driven",
	"rabbitmq":       "event_driven",
	"amqplib":        "event_driven",
	"bull":           "task_queue",
	"bullmq":         "task_queue",
	"aws-sdk":        "cloud_aws",
	"@aws-sdk":       "cloud_aws",
	"terraform":      "iac",
	"docker":         "container",
	"jest":           "testing",
	"vitest":         "testing",
	"mocha":          "testing",
	"cypress":        "e2e_testing",
	"playwright":     "e2e_testing",
	"tailwindcss":    "styling",
	"styled-component": "styling",
	"emotion":        "styling",
	"webpack":        "bundler",
	"vite":           "bundler",
	"esbuild":        "bundler",
	"rollup":         "bundler",
	"turborepo":      "monorepo",
	"lerna":          "monorepo",
	"nx":             "monorepo",
	"zod":            "validation",
	"joi":            "validation",
	"yup":            "validation",
	"axios":          "http_client",
	"got":            "http_client",
	"undici":         "http_client",
	"swagger":        "api_docs",
	"openapi":        "api_docs",
	"opentelemetry":  "observability",
	"prom-client":    "observability",
	"winston":        "logging",
	"pino":           "logging",
}

// detectTSDepSignal infers an architectural signal from a dependency name.
func detectTSDepSignal(name string) string {
	lower := strings.ToLower(name)
	for prefix, signal := range tsDepSignals {
		if strings.Contains(lower, prefix) {
			return signal
		}
	}
	return "dependency"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// tsSkipDirs are directories that should never be traversed.
var tsSkipDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".nuxt":        true,
	".output":      true,
	"coverage":     true,
	".git":         true,
}

// tsExtensions lists file extensions treated as TypeScript/JavaScript source.
var tsExtensions = map[string]bool{
	".ts":     true,
	".tsx":    true,
	".js":     true,
	".jsx":    true,
	".mjs":    true,
	".cjs":    true,
	".vue":    true,
	".svelte": true,
}

// minFloat returns the smaller of two floats.
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// sortedKeys returns the keys of a string-set map, sorted.
func sortedKeys(m interface{}) []string {
	switch v := m.(type) {
	case map[string]bool:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	case map[string]struct{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys
	default:
		return nil
	}
}

// fileExists returns true if path exists and is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists returns true if path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// hasFileWithExtension returns true if any file with the given extension exists
// under rootDir (skipping tsSkipDirs).
func hasFileWithExtension(rootDir string, ext string) bool {
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if tsSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ext {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// parseWorkspaces handles both array and object forms of the "workspaces" field
// in package.json.
func parseWorkspaces(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	// Array form: ["packages/*", "apps/*"]
	if arr, ok := raw.([]interface{}); ok {
		var out []string
		for _, v := range arr {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	// Object form: {"packages": ["packages/*"]}
	if obj, ok := raw.(map[string]interface{}); ok {
		if pkgs, ok := obj["packages"]; ok {
			return parseWorkspaces(pkgs)
		}
	}
	return nil
}
