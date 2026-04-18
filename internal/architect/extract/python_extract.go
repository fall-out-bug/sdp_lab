// Package extract provides language-specific code extractors for AI Architect.
// This file contains Python-specific extraction logic for analyzing Python codebases.
package extract

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"sdp_dev/internal/architect"
)

// Known Python standard library top-level modules (subset for heuristic classification).
var pythonStdlib = map[string]bool{
	"abc": true, "argparse": true, "ast": true, "asyncio": true,
	"base64": true, "binascii": true, "bisect": true,
	"calendar": true, "cgi": true, "collections": true, "colorsys": true,
	"concurrent": true, "configparser": true, "contextlib": true, "copy": true,
	"csv": true, "ctypes": true, "curses": true,
	"dataclasses": true, "datetime": true, "dbm": true, "decimal": true,
	"difflib": true, "dis": true, "distutils": true,
	"email": true, "encodings": true, "enum": true, "errno": true,
	"fcntl": true, "filecmp": true, "fileinput": true, "fnmatch": true,
	"fractions": true, "ftplib": true, "functools": true,
	"gc": true, "getopt": true, "getpass": true, "gettext": true,
	"glob": true, "grp": true, "gzip": true,
	"hashlib": true, "heapq": true, "hmac": true, "html": true, "http": true,
	"imaplib": true, "importlib": true, "inspect": true, "io": true,
	"ipaddress": true, "itertools": true,
	"json":      true,
	"keyword":   true,
	"linecache": true, "locale": true, "logging": true, "lzma": true,
	"mailbox": true, "math": true, "mimetypes": true, "mmap": true,
	"multiprocessing": true,
	"numbers":         true,
	"operator":        true, "optparse": true, "os": true,
	"pathlib": true, "pdb": true, "pickle": true, "pkgutil": true,
	"platform": true, "plistlib": true, "poplib": true, "posixpath": true,
	"pprint": true, "profile": true, "pstats": true, "pty": true,
	"pwd": true, "py_compile": true, "pydoc": true,
	"queue": true, "quopri": true,
	"random": true, "re": true, "readline": true, "reprlib": true,
	"resource": true, "rlcompleter": true, "runpy": true,
	"sched": true, "secrets": true, "select": true, "selectors": true,
	"shelve": true, "shlex": true, "shutil": true, "signal": true, "site": true,
	"smtplib": true, "socket": true, "socketserver": true, "sqlite3": true,
	"ssl": true, "stat": true, "statistics": true, "string": true,
	"struct": true, "subprocess": true, "sys": true, "sysconfig": true, "syslog": true,
	"tarfile": true, "tempfile": true, "termios": true, "test": true,
	"textwrap": true, "threading": true, "time": true, "timeit": true,
	"tkinter": true, "token": true, "tokenize": true, "tomllib": true,
	"trace": true, "traceback": true, "tracemalloc": true, "tty": true,
	"types": true, "typing": true,
	"unicodedata": true, "unittest": true, "urllib": true, "uuid": true,
	"venv":     true,
	"warnings": true, "wave": true, "weakref": true, "webbrowser": true,
	"xml": true, "xmlrpc": true,
	"zipfile": true, "zipimport": true, "zlib": true,
	"_thread": true, "__future__": true,
}

// pythonSkipDirs lists directories to skip when walking a Python project.
// Includes virtual environment directories, caches, and tool artifacts.
var pythonSkipDirs = map[string]bool{
	"venv":          true,
	".venv":         true,
	"env":           true,
	"__pycache__":   true,
	"node_modules":  true,
	".git":          true,
	".tox":          true,
	".mypy_cache":   true,
	".pytest_cache": true,
	".eggs":         true,
	"*.egg-info":    true, // handled via suffix check in walker
	"dist":          true,
	"build":         true,
	".nox":          true,
}

// pythonTestDirNames identifies directories likely to hold test files.
var pythonTestDirNames = map[string]bool{
	"tests":   true,
	"test":    true,
	"spec":    true,
	"specs":   true,
	"_tests":  true,
	"_test":   true,
	"testing": true,
}

var (
	reAbsoluteImport = regexp.MustCompile(`^import\s+(\S+)`)
	reFromImport     = regexp.MustCompile(`^from\s+(\S+)\s+import\s+(\S+)`)

	// Framework detection patterns.
	reFlaskApp       = regexp.MustCompile(`(?:app|application)\s*=\s*Flask\s*\(`)
	reFlaskRoute     = regexp.MustCompile(`@(?:\w+)\.route\s*\(`)
	reFlaskBlueprint = regexp.MustCompile(`Blueprint\s*\(`)
	reFastAPIDecor   = regexp.MustCompile(`@(?:\w+)\.(get|post|put|delete|patch|api_route)\s*\(`)
	reFastAPIApp     = regexp.MustCompile(`(?:app|application)\s*=\s*FastAPI\s*\(`)
	reFastAPIRouter  = regexp.MustCompile(`APIRouter\s*\(`)
	reDjangoApps     = regexp.MustCompile(`INSTALLED_APPS\s*=`)
	reDjangoURLs     = regexp.MustCompile(`urlpatterns\s*=`)
	reDjangoModel    = regexp.MustCompile(`class\s+\w+\s*\(\s*(?:models\.)?Model\s*\)`)
	reDjangoConfig   = regexp.MustCompile(`class\s+\w+Config\s*\(\s*(?:apps\.)?AppConfig\s*\)`)
	reCeleryApp      = regexp.MustCompile(`(?:celery|app)\s*=\s*Celery\s*\(`)
	rePy4JGateway    = regexp.MustCompile(`(?:launch_gateway|JavaGateway\s*\(|_gateway\s*=\s*JavaGateway)`)
	reSparkContext   = regexp.MustCompile(`(?:SparkContext\s*\(|SparkSession\.builder)`)
	rePy4JImport     = regexp.MustCompile(`(?:import\s+py4j|from\s+py4j\s+import)`)

	// requirements.txt line: package==version or package>=version etc.
	reRequirement = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)`)

	// pyproject.toml dependencies line inside [project] or [tool.poetry.dependencies]
	rePyprojectDep = regexp.MustCompile(`^\s*"?([A-Za-z0-9][A-Za-z0-9._-]*)"?\s*[>=<~!]`)
	// Simple key = "version" style (poetry)
	rePoetryDep = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*=`)

	// setup.py install_requires entry: "package" or 'package' with optional version
	reSetupDep = regexp.MustCompile(`['"]([A-Za-z0-9][A-Za-z0-9._-]*)['"]`)

	// Pipfile package line: name = "version"
	rePipfileDep = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9._-]*)\s*=`)
)

// ---------------------------------------------------------------------------
// PythonImportGraph — internal domain model for the Python import graph
// ---------------------------------------------------------------------------

// PythonModuleNode represents a Python module or package discovered during extraction.
type PythonModuleNode struct {
	ImportPath string `json:"import_path"`
	RelPath    string `json:"rel_path"`
	Name       string `json:"name"`
	Cluster    string `json:"cluster"`
	IsTest     bool   `json:"is_test"`
	IsInit     bool   `json:"is_init"`
}

// PythonImportEdge represents a directed dependency from one Python module to another.
type PythonImportEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// PythonImportGraph is the result of running PythonExtractor against a Python project.
type PythonImportGraph struct {
	Nodes            []PythonModuleNode `json:"nodes"`
	Edges            []PythonImportEdge `json:"edges"`
	Clusters         []string           `json:"clusters"`
	TestDirs         []string           `json:"test_dirs,omitempty"`
	Frameworks       []DetectedPythonFW `json:"frameworks,omitempty"`
	RuntimeCouplings []RuntimeCoupling  `json:"runtime_couplings,omitempty"`
	ExtractionMethod string             `json:"extraction_method"`
	AccuracyEstimate float64            `json:"accuracy_estimate"`
}

// DetectedPythonFW records a Python framework detected from source analysis.
type DetectedPythonFW struct {
	Name       string   `json:"name"`
	Confidence float64  `json:"confidence"`
	Evidence   string   `json:"evidence"`
	Files      []string `json:"files,omitempty"`
}

// RuntimeCoupling records a Python runtime bridge or RPC signal.
type RuntimeCoupling struct {
	Type     string `json:"type"`
	Target   string `json:"target"`
	File     string `json:"file"`
	Evidence string `json:"evidence"`
}

// PythonExtractor implements architect.Extractor for Python projects using regex.
type PythonExtractor struct{}

// Language returns "python".
func (p *PythonExtractor) Language() string { return "python" }

// Extract walks rootDir, parses .py files for imports, reads dependency manifests,
// and detects frameworks. For the full import graph with clustering, use
// BuildPythonImportGraph instead.
func (p *PythonExtractor) Extract(ctx context.Context, rootDir string) (*architect.ExtractionResult, error) {
	result := &architect.ExtractionResult{
		Language:         "python",
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.55,
	}

	seen := make(map[string]bool) // dedup key: "source:name"
	frameworks := make(map[string]architect.Framework)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		// Check context cancellation.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, _ := filepath.Rel(rootDir, path)

		if info.IsDir() {
			name := info.Name()
			if pythonSkipDirs[name] {
				return filepath.SkipDir
			}
			if strings.HasSuffix(name, ".egg-info") {
				return filepath.SkipDir
			}
			return nil
		}

		switch {
		case strings.HasSuffix(info.Name(), ".py"):
			_, fws, fileImports, err := parsePythonFileEnhanced(path, rel)
			if err != nil {
				return nil // skip unreadable files
			}
			result.FileCount++

			// Record imports as dependencies.
			for _, imp := range fileImports {
				key := imp.Source + ":" + imp.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, architect.Dependency{
						Name:   imp.Name,
						Source: imp.Source,
						Kind:   imp.Kind,
					})
				}
			}

			// Merge framework detections.
			for _, fw := range fws {
				if existing, ok := frameworks[fw.Name]; !ok || fw.Confidence > existing.Confidence {
					frameworks[fw.Name] = fw
				}
			}

		case info.Name() == "requirements.txt":
			deps := parseRequirementsTxt(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}

		case info.Name() == "pyproject.toml":
			deps := parsePyprojectToml(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}

		case info.Name() == "setup.py":
			deps := parseSetupPy(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}

		case info.Name() == "setup.cfg":
			deps := parseSetupCfg(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}

		case info.Name() == "Pipfile":
			deps := parsePipfile(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, d)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Assemble frameworks.
	for _, fw := range frameworks {
		result.Frameworks = append(result.Frameworks, fw)
	}

	return result, nil
}

func findPythonPackageRoot(rootDir string) string {
	var packageDirs []string
	var scan func(absDir, relDir string, depth int)
	scan = func(absDir, relDir string, depth int) {
		if depth >= 2 {
			return
		}
		entries, err := os.ReadDir(absDir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if pythonSkipDirs[name] || strings.HasSuffix(name, ".egg-info") {
				continue
			}
			rel := name
			if relDir != "" {
				rel = filepath.Join(relDir, name)
			}
			abs := filepath.Join(absDir, name)
			if info, err := os.Stat(filepath.Join(abs, "__init__.py")); err == nil && !info.IsDir() {
				packageDirs = append(packageDirs, filepath.ToSlash(rel))
			}
			scan(abs, rel, depth+1)
		}
	}
	scan(rootDir, "", 0)
	if len(packageDirs) == 0 {
		return ""
	}

	sort.Slice(packageDirs, func(i, j int) bool {
		depthI := strings.Count(packageDirs[i], "/")
		depthJ := strings.Count(packageDirs[j], "/")
		if depthI != depthJ {
			return depthI < depthJ
		}
		return packageDirs[i] < packageDirs[j]
	})

	var topLevel []string
	for _, pkg := range packageDirs {
		nested := false
		for _, parent := range topLevel {
			if strings.HasPrefix(pkg, parent+"/") {
				nested = true
				break
			}
		}
		if !nested {
			topLevel = append(topLevel, pkg)
		}
	}

	normalizeDir := func(path string) string {
		path = filepath.ToSlash(path)
		if path == "." {
			return ""
		}
		return path
	}
	common := strings.Split(normalizeDir(filepath.Dir(topLevel[0])), "/")
	if len(common) == 1 && common[0] == "" {
		common = nil
	}
	for _, pkg := range topLevel[1:] {
		parts := strings.Split(normalizeDir(filepath.Dir(pkg)), "/")
		if len(parts) == 1 && parts[0] == "" {
			parts = nil
		}
		n := 0
		for n < len(common) && n < len(parts) && common[n] == parts[n] {
			n++
		}
		common = common[:n]
	}
	return strings.Join(common, "/")
}

// BuildPythonImportGraph constructs the full import graph from extraction data.
// This is called by the PythonAdapter after Extract completes.
func (p *PythonExtractor) BuildPythonImportGraph(ctx context.Context, rootDir string) (*PythonImportGraph, error) {
	nodeMap := make(map[string]*PythonModuleNode)
	edgeSet := make(map[PythonImportEdge]bool)
	clusterSet := make(map[string]bool)
	var testDirs []string
	var runtimeCouplings []RuntimeCoupling
	fwMap := make(map[string]*DetectedPythonFW)
	pkgPrefix := findPythonPackageRoot(rootDir)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, _ := filepath.Rel(rootDir, path)

		if info.IsDir() {
			name := info.Name()
			if pythonSkipDirs[name] {
				return filepath.SkipDir
			}
			if strings.HasSuffix(name, ".egg-info") {
				return filepath.SkipDir
			}
			if pythonTestDirNames[name] {
				testDirs = append(testDirs, rel)
			}
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".py") {
			// Still do framework detection for key marker files.
			detectFrameworkFromMarker(info.Name(), rel, fwMap)
			return nil
		}

		// Parse this Python file for imports and framework signals.
		_, fws, fileImports, _ := parsePythonFileEnhanced(path, rel)
		if content, readErr := os.ReadFile(path); readErr == nil {
			for _, line := range strings.Split(string(content), "\n") {
				runtimeCouplings = append(runtimeCouplings, detectRuntimeCoupling(strings.TrimSpace(line), rel)...)
			}
		}

		moduleRel := filepath.ToSlash(rel)
		if pkgPrefix != "" {
			moduleRel = strings.TrimPrefix(moduleRel, pkgPrefix+"/")
		}
		modulePath := pyPathToModule(moduleRel)
		isTest := isPythonTestFile(rel)
		isInit := info.Name() == "__init__.py"
		cluster := pythonClusterFor(modulePath)

		nodeMap[modulePath] = &PythonModuleNode{
			ImportPath: modulePath,
			RelPath:    rel,
			Name:       strings.TrimSuffix(info.Name(), ".py"),
			Cluster:    cluster,
			IsTest:     isTest,
			IsInit:     isInit,
		}
		clusterSet[cluster] = true

		// Build edges for internal imports.
		for _, imp := range fileImports {
			resolved := imp.ResolvedModule
			if resolved == "" {
				continue
			}
			// Normalize resolved module to match the module path convention.
			// This handles both adding a missing source prefix and dropping a
			// filesystem package root prefix that was stripped from modulePath.
			resolved = normalizePythonModulePath(resolved, modulePath)
			if imp.Kind == "relative" || isLikelyLocalModule(resolved, nodeMap) {
				edge := PythonImportEdge{From: modulePath, To: resolved}
				if !edgeSet[edge] {
					edgeSet[edge] = true
					if _, exists := nodeMap[resolved]; !exists {
						nodeMap[resolved] = &PythonModuleNode{
							ImportPath: resolved,
							Name:       resolved,
							Cluster:    pythonClusterFor(resolved),
						}
						clusterSet[pythonClusterFor(resolved)] = true
					}
				}
			}
		}

		// Merge framework detections.
		for _, fw := range fws {
			if dfw, ok := fwMap[fw.Name]; ok {
				dfw.Files = append(dfw.Files, rel)
				if fw.Confidence > dfw.Confidence {
					dfw.Confidence = fw.Confidence
					dfw.Evidence = fw.Evidence
				}
			} else {
				fwMap[fw.Name] = &DetectedPythonFW{
					Name:       fw.Name,
					Confidence: fw.Confidence,
					Evidence:   fw.Evidence,
					Files:      []string{rel},
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build sorted slices.
	nodes := make([]PythonModuleNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, *n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ImportPath < nodes[j].ImportPath })

	edges := make([]PythonImportEdge, 0, len(edgeSet))
	for e := range edgeSet {
		edges = append(edges, e)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	clusters := make([]string, 0, len(clusterSet))
	for c := range clusterSet {
		if c != "" {
			clusters = append(clusters, c)
		}
	}
	sort.Strings(clusters)

	var fws []DetectedPythonFW
	for _, fw := range fwMap {
		fws = append(fws, *fw)
	}
	sort.Slice(fws, func(i, j int) bool { return fws[i].Name < fws[j].Name })

	// Validate test dirs — only keep those that contain .py files.
	validTestDirs := make([]string, 0, len(testDirs))
	for _, td := range testDirs {
		absDir := filepath.Join(rootDir, td)
		if hasPyFilesInDir(absDir) {
			validTestDirs = append(validTestDirs, td)
		}
	}
	sort.Strings(validTestDirs)

	sort.Slice(runtimeCouplings, func(i, j int) bool {
		if runtimeCouplings[i].File != runtimeCouplings[j].File {
			return runtimeCouplings[i].File < runtimeCouplings[j].File
		}
		if runtimeCouplings[i].Type != runtimeCouplings[j].Type {
			return runtimeCouplings[i].Type < runtimeCouplings[j].Type
		}
		return runtimeCouplings[i].Evidence < runtimeCouplings[j].Evidence
	})

	result := &PythonImportGraph{
		Nodes:            nodes,
		Edges:            edges,
		Clusters:         clusters,
		TestDirs:         validTestDirs,
		Frameworks:       fws,
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.55,
	}
	result.RuntimeCouplings = runtimeCouplings
	return result, nil
}

func detectRuntimeCoupling(trimmed, relPath string) []RuntimeCoupling {
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	var couplings []RuntimeCoupling
	if rePy4JGateway.MatchString(trimmed) {
		couplings = append(couplings, RuntimeCoupling{
			Type:     "py4j_gateway",
			Target:   "jvm",
			File:     relPath,
			Evidence: trimmed,
		})
	}
	if reSparkContext.MatchString(trimmed) {
		couplings = append(couplings, RuntimeCoupling{
			Type:     "spark_context",
			Target:   "jvm",
			File:     relPath,
			Evidence: trimmed,
		})
	}
	if rePy4JImport.MatchString(trimmed) {
		couplings = append(couplings, RuntimeCoupling{
			Type:     "py4j_import",
			Target:   "jvm",
			File:     relPath,
			Evidence: trimmed,
		})
	}
	return couplings
}

// ---------------------------------------------------------------------------
// Enhanced file parsing
// ---------------------------------------------------------------------------

// pythonImportRecord holds a single import extracted from a Python file.
type pythonImportRecord struct {
	Name           string // raw module name
	Source         string // "import", "from-import"
	Kind           string // "stdlib", "third-party", "relative"
	ResolvedModule string // resolved absolute module path (for relative imports)
}

// parsePythonFileEnhanced extracts imports and detects frameworks from a single .py file.
// Returns dependencies, framework detections, import records, and any read error.
func parsePythonFileEnhanced(path, relPath string) ([]architect.Dependency, []architect.Framework, []pythonImportRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()

	var deps []architect.Dependency
	var fws []architect.Framework
	var imports []pythonImportRecord

	scanner := bufio.NewScanner(f)
	inTripleQuote := false
	tripleChar := ""

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle triple-quote strings.
		if inTripleQuote {
			if strings.Contains(trimmed, tripleChar) {
				inTripleQuote = false
			}
			continue
		}

		// Check for triple-quote start (not already inside one).
		if countAndToggleTriple(trimmed, &inTripleQuote, &tripleChar) {
			continue
		}

		// Skip single-line comments.
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Framework detection on every non-comment, non-string line.
		detectFrameworksFromLine(trimmed, relPath, &fws)

		// Import extraction.
		if m := reFromImport.FindStringSubmatch(trimmed); m != nil {
			modName := m[1]
			importedName := m[2]
			rec := resolveImportEnhanced(modName, importedName, relPath)
			imports = append(imports, rec)
			deps = append(deps, architect.Dependency{
				Name:   rec.Name,
				Source: rec.Source,
				Kind:   rec.Kind,
			})
			continue
		}
		if m := reAbsoluteImport.FindStringSubmatch(trimmed); m != nil {
			raw := m[1]
			// Handle "import a, b, c" — split on commas.
			for _, part := range strings.Split(raw, ",") {
				name := strings.TrimSpace(part)
				if name == "" {
					continue
				}
				rec := classifyImportEnhanced(name)
				imports = append(imports, rec)
				deps = append(deps, architect.Dependency{
					Name:   rec.Name,
					Source: rec.Source,
					Kind:   rec.Kind,
				})
			}
		}
	}

	return deps, fws, imports, scanner.Err()
}

// detectFrameworksFromLine checks a single line for framework patterns.
func detectFrameworksFromLine(trimmed, relPath string, fws *[]architect.Framework) {
	// Flask detection.
	if reFlaskApp.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Flask",
			Confidence: 0.95,
			Evidence:   "Flask app instantiation",
		})
	}
	if reFlaskRoute.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Flask",
			Confidence: 0.9,
			Evidence:   "@app.route decorator",
		})
	}
	if reFlaskBlueprint.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Flask",
			Confidence: 0.85,
			Evidence:   "Blueprint registration",
		})
	}

	// FastAPI detection.
	if reFastAPIApp.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "FastAPI",
			Confidence: 0.95,
			Evidence:   "FastAPI app instantiation",
		})
	}
	if reFastAPIDecor.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "FastAPI",
			Confidence: 0.9,
			Evidence:   "FastAPI route decorator",
		})
	}
	if reFastAPIRouter.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "FastAPI",
			Confidence: 0.85,
			Evidence:   "APIRouter usage",
		})
	}

	// Django detection.
	if reDjangoApps.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Django",
			Confidence: 0.95,
			Evidence:   "INSTALLED_APPS",
		})
	}
	if reDjangoURLs.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Django",
			Confidence: 0.9,
			Evidence:   "urlpatterns",
		})
	}
	if reDjangoModel.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Django",
			Confidence: 0.85,
			Evidence:   "Django model class",
		})
	}
	if reDjangoConfig.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Django",
			Confidence: 0.9,
			Evidence:   "AppConfig subclass",
		})
	}

	// Celery detection.
	if reCeleryApp.MatchString(trimmed) {
		*fws = append(*fws, architect.Framework{
			Name:       "Celery",
			Confidence: 0.9,
			Evidence:   "Celery app instantiation",
		})
	}
}

// detectFrameworkFromMarker checks marker files (e.g., manage.py, settings.py)
// for Django project indicators.
func detectFrameworkFromMarker(fileName, relPath string, fwMap map[string]*DetectedPythonFW) {
	switch fileName {
	case "manage.py":
		fwMap["Django"] = &DetectedPythonFW{
			Name:       "Django",
			Confidence: 0.95,
			Evidence:   "manage.py present",
			Files:      []string{relPath},
		}
	case "settings.py":
		// settings.py alone is not enough — but if we see it in a Django-like
		// path (e.g., project/settings.py) it's a strong signal.
		// We add a lower-confidence detection here.
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) >= 2 {
			if fw, ok := fwMap["Django"]; ok {
				fw.Files = append(fw.Files, relPath)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Import resolution
// ---------------------------------------------------------------------------

// resolveImportEnhanced resolves a Python import to an absolute module path.
func resolveImportEnhanced(modName, importedName, relPath string) pythonImportRecord {
	if !strings.HasPrefix(modName, ".") {
		// Absolute import.
		rec := classifyImportEnhanced(modName)
		rec.ResolvedModule = modName
		return rec
	}

	// Relative import: count leading dots.
	dots := 0
	for _, ch := range modName {
		if ch == '.' {
			dots++
		} else {
			break
		}
	}
	suffix := modName[dots:] // e.g. "" for ".", "core" for "..core"

	// Determine the package directory from the file's relative path.
	dir := filepath.Dir(relPath)
	parts := strings.Split(filepath.ToSlash(dir), "/")

	// Go up (dots - 1) levels from the current package.
	ups := dots - 1
	if ups > len(parts) {
		ups = len(parts)
	}
	if ups > 0 {
		parts = parts[:len(parts)-ups]
	}

	// Build the resolved module path.
	if suffix == "" {
		suffix = importedName
	}

	var resolved string
	base := strings.Join(parts, ".")
	if base == "." || base == "" {
		resolved = suffix
	} else if suffix == "" {
		resolved = base
	} else {
		resolved = base + "." + suffix
	}

	if resolved == "" {
		resolved = modName
	}

	return pythonImportRecord{
		Name:           resolved,
		Source:         "from-import",
		Kind:           "relative",
		ResolvedModule: resolved,
	}
}

// classifyImportEnhanced decides if a module is stdlib, third-party, or local.
func classifyImportEnhanced(name string) pythonImportRecord {
	top := name
	if idx := strings.Index(name, "."); idx > 0 {
		top = name[:idx]
	}

	kind := "third-party"
	if pythonStdlib[top] {
		kind = "stdlib"
	}

	return pythonImportRecord{
		Name:           name,
		Source:         "import",
		Kind:           kind,
		ResolvedModule: name,
	}
}

// normalizePythonModulePath attempts to align a resolved import module path
// with the convention used by the source module path. This handles both cases
// where the source module needs a missing prefix added and where the resolved
// path still carries a stripped filesystem package root prefix.
func normalizePythonModulePath(resolved, sourceModulePath string) string {
	// Already matches nodeMap convention — no change needed.
	if resolved == sourceModulePath {
		return resolved
	}

	// Extract the first segment of the resolved path (e.g. "pyspark" from "pyspark.ml.base").
	resFirst := resolved
	if idx := strings.Index(resolved, "."); idx > 0 {
		resFirst = resolved[:idx]
	}

	// If the resolved path's first segment matches a later segment in the source path,
	// prepend the source prefix. E.g. resolved="pyspark.ml.base", source="python.pyspark.ml.base"
	// → "pyspark" matches source[1] → prepend "python.".
	srcParts := strings.Split(sourceModulePath, ".")
	for i, part := range srcParts {
		if part == resFirst && i > 0 {
			prefix := strings.Join(srcParts[:i], ".")
			return prefix + "." + resolved
		}
	}

	// If the resolved path still includes an extra leading prefix, drop it once
	// the source package root aligns with a later segment.
	sourceBase := pythonClusterFor(sourceModulePath)
	if sourceBase == "" {
		sourceBase = sourceModulePath
	}
	sourceFirst := sourceModulePath
	if idx := strings.Index(sourceModulePath, "."); idx > 0 {
		sourceFirst = sourceModulePath[:idx]
	}
	resParts := strings.Split(resolved, ".")
	for i, part := range resParts {
		if part != sourceFirst || i == 0 {
			continue
		}
		candidate := strings.Join(resParts[i:], ".")
		if candidate == sourceModulePath || candidate == sourceBase || strings.HasPrefix(candidate, sourceBase+".") {
			return candidate
		}
	}

	return resolved
}

// isLikelyLocalModule checks whether a resolved module path is likely a
// local (project-internal) module based on known nodes.
func isLikelyLocalModule(modulePath string, nodeMap map[string]*PythonModuleNode) bool {
	if _, ok := nodeMap[modulePath]; ok {
		return true
	}
	// Check if a prefix matches a known node (e.g. "pkg.sub" matches "pkg").
	top := modulePath
	if idx := strings.Index(modulePath, "."); idx > 0 {
		top = modulePath[:idx]
	}
	for k := range nodeMap {
		if k == top || strings.HasPrefix(k, top+".") {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Dependency manifest parsers
// ---------------------------------------------------------------------------

// parseRequirementsTxt reads a requirements.txt file.
func parseRequirementsTxt(path string) []architect.Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []architect.Dependency
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if m := reRequirement.FindStringSubmatch(line); m != nil {
			deps = append(deps, architect.Dependency{
				Name:   m[1],
				Source: "requirements.txt",
				Kind:   "third-party",
			})
		}
	}
	return deps
}

// parsePyprojectToml does a best-effort regex parse of pyproject.toml dependencies.
func parsePyprojectToml(path string) []architect.Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []architect.Dependency
	scanner := bufio.NewScanner(f)
	inDeps := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect dependency sections.
		if strings.HasPrefix(trimmed, "[") {
			lower := strings.ToLower(trimmed)
			// PEP 621: [project.dependencies] or [project.optional-dependencies.*]
			// Poetry: [tool.poetry.dependencies]
			inDeps = strings.Contains(lower, "[project") && strings.Contains(lower, "dependencies") ||
				strings.Contains(lower, "[tool.poetry") && strings.Contains(lower, "dependencies")
			continue
		}

		if !inDeps {
			continue
		}

		// Array-style (PEP 621): "flask>=2.0",
		if m := rePyprojectDep.FindStringSubmatch(trimmed); m != nil {
			name := strings.Trim(m[1], `"`)
			if name != "" && name != "python" {
				deps = append(deps, architect.Dependency{
					Name:   name,
					Source: "pyproject.toml",
					Kind:   "third-party",
				})
			}
			continue
		}

		// Poetry-style: flask = "^2.0"
		if m := rePoetryDep.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			if name != "" && name != "python" {
				deps = append(deps, architect.Dependency{
					Name:   name,
					Source: "pyproject.toml",
					Kind:   "third-party",
				})
			}
		}
	}
	return deps
}

// parseSetupPy extracts dependencies from setup.py install_requires list.
// Best-effort regex parsing; does not execute the file.
func parseSetupPy(path string) []architect.Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []architect.Dependency
	scanner := bufio.NewScanner(f)
	inInstallRequires := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect install_requires = [ ... ]
		if strings.Contains(trimmed, "install_requires") {
			inInstallRequires = true
			// May be on the same line: install_requires = ["flask"]
			extractSetupDepsFromLine(trimmed, &deps)
			continue
		}

		if inInstallRequires {
			// End of list.
			if strings.Contains(trimmed, "]") {
				inInstallRequires = false
				// May have content before the bracket.
				extractSetupDepsFromLine(trimmed, &deps)
				continue
			}
			extractSetupDepsFromLine(trimmed, &deps)
		}
	}
	return deps
}

// extractSetupDepsFromLine extracts quoted dependency names from a setup.py line.
func extractSetupDepsFromLine(line string, deps *[]architect.Dependency) {
	matches := reSetupDep.FindAllStringSubmatch(line, -1)
	for _, m := range matches {
		name := m[1]
		if name == "" {
			continue
		}
		*deps = append(*deps, architect.Dependency{
			Name:   name,
			Source: "setup.py",
			Kind:   "third-party",
		})
	}
}

// parseSetupCfg extracts dependencies from setup.cfg install_requires section.
func parseSetupCfg(path string) []architect.Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []architect.Dependency
	scanner := bufio.NewScanner(f)
	inInstallRequires := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect [options] section which contains install_requires.
		if strings.HasPrefix(trimmed, "[") {
			inInstallRequires = strings.Contains(strings.ToLower(trimmed), "options")
			continue
		}

		if !inInstallRequires {
			continue
		}

		// Look for install_requires = followed by indented lines.
		if strings.HasPrefix(trimmed, "install_requires") {
			// May be inline: install_requires = flask
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" && !strings.HasPrefix(val, "\n") {
					// Inline single dep.
					name := strings.Trim(val, `'"`)
					if name != "" {
						deps = append(deps, architect.Dependency{
							Name:   name,
							Source: "setup.cfg",
							Kind:   "third-party",
						})
					}
				}
			}
			inInstallRequires = true
			continue
		}

		// Indented continuation lines under install_requires.
		if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			// New non-indented line means end of install_requires block.
			inInstallRequires = false
			continue
		}

		if trimmed != "" {
			// Extract dep name from the line (may have version specifiers).
			name := trimmed
			if idx := strings.IndexAny(name, "><=!~;"); idx >= 0 {
				name = strings.TrimSpace(name[:idx])
			}
			name = strings.Trim(name, `'"`)
			if name != "" {
				deps = append(deps, architect.Dependency{
					Name:   name,
					Source: "setup.cfg",
					Kind:   "third-party",
				})
			}
		}
	}
	return deps
}

// parsePipfile extracts dependencies from Pipfile [packages] section.
func parsePipfile(path string) []architect.Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []architect.Dependency
	scanner := bufio.NewScanner(f)
	inPackages := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect [packages] section.
		if strings.HasPrefix(trimmed, "[") {
			inPackages = trimmed == "[packages]"
			continue
		}

		if !inPackages || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Pipfile style: name = "version" or name = "*"
		if m := rePipfileDep.FindStringSubmatch(trimmed); m != nil {
			name := strings.TrimSpace(m[1])
			if name != "" {
				deps = append(deps, architect.Dependency{
					Name:   name,
					Source: "Pipfile",
					Kind:   "third-party",
				})
			}
		}
	}
	return deps
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// pyPathToModule converts a relative file path to a Python module path.
// E.g., "pkg/sub/module.py" -> "pkg.sub.module"
// E.g., "pkg/sub/__init__.py" -> "pkg.sub"
func pyPathToModule(relPath string) string {
	// Normalize separators.
	p := filepath.ToSlash(relPath)
	// Remove .py extension.
	p = strings.TrimSuffix(p, ".py")
	// __init__ means the package itself.
	p = strings.TrimSuffix(p, "/__init__")
	// Replace / with .
	return strings.ReplaceAll(p, "/", ".")
}

// pythonClusterFor returns the cluster (parent directory path) for a module.
// This is used for C4 Level 3 package clustering.
func pythonClusterFor(modulePath string) string {
	parts := strings.Split(modulePath, ".")
	if len(parts) <= 1 {
		return ""
	}
	// Use all but the last component as the cluster.
	// E.g., "pkg.sub.module" -> "pkg.sub"
	// But for "pkg.module" -> "pkg"
	// For deeper nesting: "a.b.c.d" -> "a.b.c"
	cluster := strings.Join(parts[:len(parts)-1], ".")
	// Limit to top 3 levels for C4 readability.
	if numParts := len(parts) - 1; numParts > 3 {
		cluster = strings.Join(parts[:3], ".")
	}
	return cluster
}

// isPythonTestFile determines if a file path is in a test directory.
func isPythonTestFile(relPath string) bool {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	for _, part := range parts[:len(parts)-1] { // exclude filename
		if pythonTestDirNames[part] {
			return true
		}
	}
	// Also check if filename starts with test_ or _test.
	base := parts[len(parts)-1]
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py")
}

// hasPyFilesInDir returns true if the directory contains at least one .py file.
func hasPyFilesInDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") {
			return true
		}
	}
	return false
}

// countAndToggleTriple detects triple-quote boundaries. Returns true if the
// line is consumed by (or starts) a triple-quote block.
func countAndToggleTriple(line string, inTriple *bool, tripleChar *string) bool {
	for _, tq := range []string{`"""`, `'''`} {
		count := strings.Count(line, tq)
		if count == 0 {
			continue
		}
		if count == 1 {
			*inTriple = true
			*tripleChar = tq
			return true
		}
		if count%2 == 0 {
			return true
		}
		*inTriple = true
		*tripleChar = tq
		return true
	}
	return false
}

// DetectPythonCycles performs DFS-based cycle detection on the Python import
// graph nodes/edges and returns all elementary cycles found.
func DetectPythonCycles(nodes []PythonModuleNode, edges []PythonImportEdge) [][]string {
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
		white = 0
		gray  = 1
		black = 2
	)

	color := make(map[string]int)
	parent := make(map[string]string)
	var cycles [][]string

	buildCycle := func(from, to string) []string {
		var c []string
		cur := from
		for cur != to {
			c = append(c, cur)
			cur = parent[cur]
		}
		c = append(c, to)
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
				cycles = append(cycles, buildCycle(u, v))
			}
		}
		color[u] = black
	}

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
