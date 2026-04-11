package extract

import (
	"context"
	"os"
	"path/filepath"
	"sdp_dev/internal/architect"
	"strings"
)

// ---------------------------------------------------------------------------
// GoAdapter — wraps GoExtractor to implement architect.Extractor
// ---------------------------------------------------------------------------

// GoAdapter wraps GoExtractor to implement the canonical Extractor interface.
type GoAdapter struct{}

// Name returns the extractor identifier.
func (GoAdapter) Name() string { return "go" }

// Extract analyzes the Go repository at repoRoot and returns a ProfileFragment.
func (GoAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	// Check if go.mod or go.work exists (go.work for monorepos)
	hasGoMod := false
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		hasGoMod = true
	}
	if !hasGoMod {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.work")); err != nil {
			return &architect.ProfileFragment{}, nil // Not a Go project
		}
	}

	e := NewGoExtractor(repoRoot)
	graph, err := e.Extract(ctx)
	if err != nil {
		return nil, err
	}

	// Convert GoImportGraph → ProfileFragment
	frag := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{{Primary: "go", All: []string{"go"}}},
	}

	if graph != nil {
		// Build ImportGraph
		importGraph := &architect.ImportGraph{
			ExtractionMethod: graph.ExtractionMethod,
			AccuracyEstimate: graph.AccuracyEstimate,
			Nodes:            len(graph.Nodes),
			Edges:            len(graph.Edges),
		}

		// Convert clusters
		for _, c := range graph.Clusters {
			importGraph.Clusters = append(importGraph.Clusters, architect.ImportCluster{
				ID: c,
			})
		}

		// Fill cluster packages
		for _, node := range graph.Nodes {
			for i := range importGraph.Clusters {
				if importGraph.Clusters[i].ID == node.Cluster {
					importGraph.Clusters[i].Packages = append(importGraph.Clusters[i].Packages, node.ImportPath)
					break
				}
			}
		}

		// Count internal/external edges per cluster
		pkgSet := make(map[string]bool)
		for i, c := range importGraph.Clusters {
			pkgSet = make(map[string]bool, len(c.Packages))
			for _, p := range c.Packages {
				pkgSet[p] = true
			}
			for _, e := range graph.Edges {
				if pkgSet[e.From] {
					if pkgSet[e.To] {
						importGraph.Clusters[i].InternalEdges++
					} else {
						importGraph.Clusters[i].ExternalEdges++
					}
				}
			}
		}

		// Convert cycles
		for _, cycle := range graph.Cycles {
			if len(cycle) >= 2 {
				importGraph.CircularDependencies = append(importGraph.CircularDependencies, architect.CircularDep{
					A:        cycle[0],
					B:        cycle[len(cycle)-1],
					EdgeType: "go_import",
				})
			}
		}

		frag.ImportGraph = importGraph

		// Surface framework detection as DependencyInfo signals.
		if len(graph.Frameworks) > 0 {
			depInfo := architect.DependencyInfo{
				Language: "go",
				File:     "go.mod",
			}
			for _, fw := range graph.Frameworks {
				depInfo.NotableDeps = append(depInfo.NotableDeps, architect.NotableDep{
					Name:    fw.Name,
					FoundIn: 1,
					Signal:  "web_framework",
				})
			}
			frag.Dependencies = append(frag.Dependencies, depInfo)
		}

		// Surface go.mod module info as DependencyInfo.
		if graph.ModuleInfo != nil && len(graph.ModuleInfo.Requires) > 0 {
			modDepInfo := architect.DependencyInfo{
				Language: "go",
				File:     "go.mod",
				DepCount: len(graph.ModuleInfo.Requires),
			}
			// Collect notable signals from go.mod requires.
			signalSet := make(map[string]bool)
			for _, req := range graph.ModuleInfo.Requires {
				for prefix, signal := range notableSignals {
					if strings.Contains(strings.ToLower(req.Path), prefix) && !signalSet[signal] {
						signalSet[signal] = true
						modDepInfo.Signals = append(modDepInfo.Signals, signal)
					}
				}
			}
			sortStrings(modDepInfo.Signals)

			// Merge with framework DependencyInfo if both exist.
			if len(frag.Dependencies) > 0 {
				// Merge notable deps from frameworks into module deps.
				for _, nd := range frag.Dependencies[0].NotableDeps {
					modDepInfo.NotableDeps = append(modDepInfo.NotableDeps, nd)
				}
				frag.Dependencies[0] = modDepInfo
			} else {
				frag.Dependencies = append(frag.Dependencies, modDepInfo)
			}
		}

		frag.Metrics = &architect.CodeMetrics{
			LanguagesCount:     1,
			ContainersDetected: len(graph.DeployUnits),
			ComponentsDetected: len(graph.Clusters),
		}
	}

	return frag, nil
}

// ---------------------------------------------------------------------------
// PythonAdapter — wraps PythonExtractor to implement architect.Extractor
// ---------------------------------------------------------------------------

// PythonAdapter wraps PythonExtractor to implement the canonical Extractor interface.
type PythonAdapter struct{}

// Name returns the extractor identifier.
func (PythonAdapter) Name() string { return "python" }

// Extract analyzes the Python repository at repoRoot and returns a ProfileFragment.
func (PythonAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	// Check if this is a Python project by looking for common markers
	hasPythonMarkers := false
	for _, marker := range []string{"requirements.txt", "pyproject.toml", "setup.py", "setup.cfg", "Pipfile", "poetry.lock"} {
		if _, err := os.Stat(filepath.Join(repoRoot, marker)); err == nil {
			hasPythonMarkers = true
			break
		}
	}
	// Also check for any .py files
	if !hasPythonMarkers {
		if err := filepath.Walk(repoRoot, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			if strings.HasSuffix(fi.Name(), ".py") {
				hasPythonMarkers = true
				return filepath.SkipDir // Found a Python file, stop walking
			}
			return nil
		}); err == nil && hasPythonMarkers {
			// Found Python files
		}
	}

	if !hasPythonMarkers {
		return &architect.ProfileFragment{}, nil // Not a Python project
	}

	e := &PythonExtractor{}
	result, err := e.Extract(ctx, repoRoot)
	if err != nil {
		return nil, err
	}
	frag := convertExtractionResult(result)

	// Build and attach the Python import graph for C4 Level 3 clustering.
	graph, graphErr := e.BuildPythonImportGraph(ctx, repoRoot)
	if graphErr == nil && graph != nil && len(graph.Nodes) > 0 {
		ig := &architect.ImportGraph{
			ExtractionMethod: graph.ExtractionMethod,
			AccuracyEstimate: graph.AccuracyEstimate,
			Nodes:            len(graph.Nodes),
			Edges:            len(graph.Edges),
		}

		// Convert clusters.
		for _, c := range graph.Clusters {
			ig.Clusters = append(ig.Clusters, architect.ImportCluster{
				ID: c,
			})
		}

		// Fill cluster packages.
		for _, node := range graph.Nodes {
			for i := range ig.Clusters {
				if ig.Clusters[i].ID == node.Cluster {
					ig.Clusters[i].Packages = append(ig.Clusters[i].Packages, node.ImportPath)
					break
				}
			}
		}

		// Count internal/external edges per cluster.
		pkgSet := make(map[string]bool)
		for i, c := range ig.Clusters {
			pkgSet = make(map[string]bool, len(c.Packages))
			for _, p := range c.Packages {
				pkgSet[p] = true
			}
			for _, edge := range graph.Edges {
				if pkgSet[edge.From] {
					if pkgSet[edge.To] {
						ig.Clusters[i].InternalEdges++
					} else {
						ig.Clusters[i].ExternalEdges++
					}
				}
			}
		}

		// Convert cycles.
		cycles := DetectPythonCycles(graph.Nodes, graph.Edges)
		for _, cycle := range cycles {
			if len(cycle) >= 2 {
				ig.CircularDependencies = append(ig.CircularDependencies, architect.CircularDep{
					A:        cycle[0],
					B:        cycle[len(cycle)-1],
					EdgeType: "python_import",
				})
			}
		}

		frag.ImportGraph = ig

		// Surface frameworks from the graph.
		if len(graph.Frameworks) > 0 {
			fwDepInfo := architect.DependencyInfo{
				Language: "python",
			}
			for _, fw := range graph.Frameworks {
				fwDepInfo.NotableDeps = append(fwDepInfo.NotableDeps, architect.NotableDep{
					Name:    fw.Name,
					FoundIn: len(fw.Files),
					Signal:  "web_framework",
				})
			}
			frag.Dependencies = append(frag.Dependencies, fwDepInfo)
		}

		if frag.Metrics != nil {
			frag.Metrics.ComponentsDetected = len(graph.Clusters)
		}
	}

	return frag, nil
}

// ---------------------------------------------------------------------------
// JavaAdapter — wraps JavaExtractor to implement architect.Extractor
// ---------------------------------------------------------------------------

// JavaAdapter wraps JavaExtractor to implement the canonical Extractor interface.
type JavaAdapter struct{}

// Name returns the extractor identifier.
func (JavaAdapter) Name() string { return "java" }

// Extract analyzes the Java/Kotlin repository at repoRoot and returns a ProfileFragment.
func (JavaAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	// Check if this is a Java/Kotlin project by looking for common markers
	hasJavaMarkers := false
	for _, marker := range []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(repoRoot, marker)); err == nil {
			hasJavaMarkers = true
			break
		}
	}
	// Also check for any .java or .kt files
	if !hasJavaMarkers {
		if err := filepath.Walk(repoRoot, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			name := fi.Name()
			if strings.HasSuffix(name, ".java") || strings.HasSuffix(name, ".kt") || strings.HasSuffix(name, ".kts") {
				hasJavaMarkers = true
				return filepath.SkipDir // Found a Java/Kotlin file, stop walking
			}
			return nil
		}); err == nil && hasJavaMarkers {
			// Found Java/Kotlin files
		}
	}

	if !hasJavaMarkers {
		return &architect.ProfileFragment{}, nil // Not a Java project
	}

	e := &JavaExtractor{}
	result, err := e.Extract(repoRoot)
	if err != nil {
		return nil, err
	}
	return convertJavaResult(result), nil
}

// ---------------------------------------------------------------------------
// TypeScriptAdapter — wraps TypeScriptExtractor to implement architect.Extractor
// ---------------------------------------------------------------------------

// TypeScriptAdapter wraps TSExtractor to implement the canonical Extractor interface.
type TypeScriptAdapter struct{}

// Name returns the extractor identifier.
func (TypeScriptAdapter) Name() string { return "typescript" }

// Extract analyzes the TypeScript/JavaScript repository at repoRoot and returns a ProfileFragment.
func (TypeScriptAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	e := NewTSExtractor()
	return e.Extract(ctx, repoRoot)
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

// depSignals maps dependency name patterns to architectural signals.
var depSignals = map[string]string{
	"kafka":               "event_driven",
	"confluent-kafka":     "event_driven",
	"fastapi":             "web_framework",
	"flask":               "web_framework",
	"django":              "web_framework",
	"grpc":                "rpc",
	"grpcio":              "rpc",
	"protobuf":            "rpc",
	"celery":              "task_queue",
	"rq":                  "task_queue",
	"redis":               "cache",
	"memcached":           "cache",
	"sqlalchemy":          "orm",
	"pandas":              "data_processing",
	"numpy":               "data_processing",
	"pyarrow":             "data_processing",
	"pydantic":            "validation",
	"pytest":              "testing",
	"requests":            "http_client",
	"httpx":               "http_client",
	"aiohttp":             "async_http",
	"uvicorn":             "asgi_server",
	"gunicorn":            "wsgi_server",
	"boto3":               "cloud_aws",
	"botocore":            "cloud_aws",
	"azure":               "cloud_azure",
	"google-cloud":        "cloud_gcp",
	"tensorflow":          "ml_framework",
	"torch":               "ml_framework",
	"scikit-learn":        "ml_framework",
	"django-rest":         "api_framework",
	"djangorestframework": "api_framework",
	"flask-restful":       "api_framework",
}

// detectDepSignal infers architectural signal from dependency name.
func detectDepSignal(name string) string {
	lower := strings.ToLower(name)
	for prefix, signal := range depSignals {
		if strings.Contains(lower, prefix) {
			return signal
		}
	}
	return "dependency"
}

// convertExtractionResult converts a Python ExtractionResult to ProfileFragment.
func convertExtractionResult(r *architect.ExtractionResult) *architect.ProfileFragment {
	if r == nil {
		return &architect.ProfileFragment{}
	}
	frag := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{{
			Primary: r.Language,
			All:     []string{r.Language},
		}},
	}

	// Convert Dependencies → DependencyInfo
	// Only include deps from manifest files, not from source import analysis.
	manifestSources := map[string]bool{
		"requirements.txt": true,
		"pyproject.toml":   true,
		"setup.py":         true,
		"setup.cfg":        true,
		"Pipfile":          true,
	}
	if len(r.Dependencies) > 0 {
		depInfo := architect.DependencyInfo{}
		seenNotable := make(map[string]bool)
		for _, d := range r.Dependencies {
			if manifestSources[d.Source] && !seenNotable[d.Name] {
				seenNotable[d.Name] = true
				depInfo.NotableDeps = append(depInfo.NotableDeps, architect.NotableDep{
					Name:    d.Name,
					FoundIn: 1,
					Signal:  detectDepSignal(d.Name),
				})
			}
		}
		if len(depInfo.NotableDeps) > 0 {
			frag.Dependencies = []architect.DependencyInfo{depInfo}
		}
	}

	// Set metrics
	frag.Metrics = &architect.CodeMetrics{
		LanguagesCount: 1,
	}

	return frag
}

// convertJavaResult converts JavaExtractionResult to ProfileFragment.
func convertJavaResult(r *JavaExtractionResult) *architect.ProfileFragment {
	if r == nil {
		return &architect.ProfileFragment{}
	}
	frag := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{{
			Primary: "java",
			All:     []string{"java", "kotlin"},
		}},
	}

	// Convert ImportGraph
	if len(r.ImportGraph.PackageImports) > 0 {
		nodes := 0
		edges := 0
		for _, imports := range r.ImportGraph.PackageImports {
			nodes++
			edges += len(imports)
		}
		importGraph := &architect.ImportGraph{
			ExtractionMethod: r.ExtractionMethod,
			AccuracyEstimate: r.AccuracyEstimate,
			Nodes:            nodes,
			Edges:            edges,
		}

		packageDirToCluster := make(map[string]string, len(r.ImportGraph.PackageImports))
		packageNameToCluster := make(map[string]string, len(r.ImportGraph.PackageImports))
		clusterIndex := make(map[string]int)
		for pkgDir := range r.ImportGraph.PackageImports {
			clusterID := javaClusterID(pkgDir)
			packageDirToCluster[pkgDir] = clusterID
			if pkgName := javaPackageName(pkgDir); pkgName != "" {
				packageNameToCluster[pkgName] = clusterID
			}
			if _, ok := clusterIndex[clusterID]; !ok {
				importGraph.Clusters = append(importGraph.Clusters, architect.ImportCluster{
					ID: clusterID,
				})
				clusterIndex[clusterID] = len(importGraph.Clusters) - 1
			}
			idx := clusterIndex[clusterID]
			importGraph.Clusters[idx].Packages = append(importGraph.Clusters[idx].Packages, pkgDir)
		}

		for pkgDir, imports := range r.ImportGraph.PackageImports {
			fromCluster := packageDirToCluster[pkgDir]
			if fromCluster == "" {
				continue
			}
			idx := clusterIndex[fromCluster]
			for _, imp := range imports {
				targetCluster := javaImportTargetCluster(imp, packageNameToCluster)
				if targetCluster == "" {
					importGraph.Clusters[idx].ExternalEdges++
					continue
				}
				if _, ok := clusterIndex[targetCluster]; !ok {
					importGraph.Clusters[idx].ExternalEdges++
					continue
				}
				if targetCluster == fromCluster {
					importGraph.Clusters[idx].InternalEdges++
				} else {
					importGraph.Clusters[idx].ExternalEdges++
				}
			}
		}

		frag.ImportGraph = importGraph
	}

	// Convert BuildSystem.Dependencies → DependencyInfo
	if r.BuildSystem != nil && len(r.BuildSystem.Dependencies) > 0 {
		depInfo := architect.DependencyInfo{}
		for _, d := range r.BuildSystem.Dependencies {
			depInfo.NotableDeps = append(depInfo.NotableDeps, architect.NotableDep{
				Name:   d.Group + ":" + d.Artifact,
				Signal: "java_dependency",
			})
		}
		frag.Dependencies = []architect.DependencyInfo{depInfo}
	}

	frag.Metrics = &architect.CodeMetrics{
		LanguagesCount: 1,
	}

	// Convert Maven modules to ModuleBoundaries
	if len(r.Modules) > 0 {
		for _, mod := range r.Modules {
			frag.Boundaries = append(frag.Boundaries, architect.ModuleBoundary{
				Name:       mod,
				Pattern:    mod + "/**",
				EntryFiles: []string{mod + "/pom.xml"},
			})
		}
	}

	return frag
}

func javaClusterID(pkgDir string) string {
	pkgName := javaPackageName(pkgDir)
	if pkgName == "" {
		return "unnamed"
	}

	clusterID := javaImportPrefix(pkgName, 3)
	if clusterID == "" {
		return pkgName
	}
	return clusterID
}

func javaPackageName(pkgDir string) string {
	normalized := filepath.ToSlash(pkgDir)
	for _, marker := range []string{"src/main/java/", "src/test/java/", "src/main/kotlin/", "src/test/kotlin/"} {
		if idx := strings.Index(normalized, marker); idx >= 0 {
			normalized = normalized[idx+len(marker):]
			break
		}
	}

	normalized = strings.Trim(normalized, "/")
	if normalized == "" || normalized == "." {
		return ""
	}
	return strings.ReplaceAll(normalized, "/", ".")
}

func javaImportTargetCluster(imp string, packageNameToCluster map[string]string) string {
	imp = strings.TrimSuffix(imp, ".*")
	imp = strings.TrimSuffix(imp, "*")

	for imp != "" {
		if clusterID, ok := packageNameToCluster[imp]; ok {
			return clusterID
		}
		lastDot := strings.LastIndex(imp, ".")
		if lastDot < 0 {
			break
		}
		imp = imp[:lastDot]
	}
	return ""
}

// javaImportPrefix extracts the first n segments from a Java import path.
// For example "org.apache.spark.sql.DataFrame" with n=3 returns "org.apache.spark".
func javaImportPrefix(imp string, n int) string {
	imp = strings.TrimSuffix(imp, ".*")
	imp = strings.TrimSuffix(imp, "*")

	parts := strings.Split(imp, ".")
	if len(parts) < n {
		n = len(parts)
	}
	return strings.Join(parts[:n], ".")
}
