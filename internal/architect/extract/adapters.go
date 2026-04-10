package extract

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sdp_dev/internal/architect"
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
	// Check if go.mod exists
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); os.IsNotExist(err) {
		return &architect.ProfileFragment{}, nil // Not a Go project
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
		frag.Metrics = &architect.CodeMetrics{
			LanguagesCount: 1,
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
	return convertExtractionResult(result), nil
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

// TypeScriptAdapter wraps TypeScriptExtractor to implement the canonical Extractor interface.
type TypeScriptAdapter struct{}

// Name returns the extractor identifier.
func (TypeScriptAdapter) Name() string { return "typescript" }

// Extract analyzes the TypeScript/JavaScript repository at repoRoot and returns a ProfileFragment.
func (TypeScriptAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	// Check if this is a TypeScript/JavaScript project by looking for common markers
	hasTSMarkers := false
	for _, marker := range []string{"package.json", "tsconfig.json", "jsconfig.json"} {
		if _, err := os.Stat(filepath.Join(repoRoot, marker)); err == nil {
			hasTSMarkers = true
			break
		}
	}
	// Also check for any .ts, .tsx, .js, or .jsx files
	if !hasTSMarkers {
		if err := filepath.Walk(repoRoot, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			name := fi.Name()
			if strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx") || strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".jsx") {
				hasTSMarkers = true
				return filepath.SkipDir // Found a TS/JS file, stop walking
			}
			return nil
		}); err == nil && hasTSMarkers {
			// Found TS/JS files
		}
	}

	if !hasTSMarkers {
		return &architect.ProfileFragment{}, nil // Not a TypeScript project
	}

	e := &TypeScriptExtractor{}
	result, err := e.Extract(repoRoot)
	if err != nil {
		return nil, err
	}
	return convertTSResult(result), nil
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

// depSignals maps dependency name patterns to architectural signals.
var depSignals = map[string]string{
	"kafka":            "event_driven",
	"confluent-kafka":  "event_driven",
	"fastapi":          "web_framework",
	"flask":            "web_framework",
	"django":           "web_framework",
	"grpc":             "rpc",
	"grpcio":           "rpc",
	"protobuf":         "rpc",
	"celery":           "task_queue",
	"rq":               "task_queue",
	"redis":            "cache",
	"memcached":        "cache",
	"sqlalchemy":       "orm",
	"pandas":           "data_processing",
	"numpy":            "data_processing",
	"pyarrow":          "data_processing",
	"pydantic":         "validation",
	"pytest":           "testing",
	"requests":         "http_client",
	"httpx":            "http_client",
	"aiohttp":          "async_http",
	"uvicorn":          "asgi_server",
	"gunicorn":         "wsgi_server",
	"boto3":            "cloud_aws",
	"botocore":         "cloud_aws",
	"azure":            "cloud_azure",
	"google-cloud":     "cloud_gcp",
	"tensorflow":       "ml_framework",
	"torch":            "ml_framework",
	"scikit-learn":     "ml_framework",
	"django-rest":      "api_framework",
	"djangorestframework": "api_framework",
	"flask-restful":    "api_framework",
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
	// Only include deps from manifest files (requirements.txt/pyproject.toml), not from source analysis
	if len(r.Dependencies) > 0 {
		depInfo := architect.DependencyInfo{}
		seenNotable := make(map[string]bool)
		for _, d := range r.Dependencies {
			// Only deps from manifest files, not source == "import" (which catches internal/local imports)
			if (d.Source == "requirements.txt" || d.Source == "pyproject.toml") && !seenNotable[d.Name] {
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
		frag.ImportGraph = &architect.ImportGraph{
			ExtractionMethod: r.ExtractionMethod,
			AccuracyEstimate: r.AccuracyEstimate,
			Nodes:            nodes,
			Edges:            edges,
		}
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

	return frag
}

// convertTSResult converts TSExtractionResult to ProfileFragment.
func convertTSResult(r *TSExtractionResult) *architect.ProfileFragment {
	if r == nil {
		return &architect.ProfileFragment{}
	}
	frag := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{{
			Primary: "typescript",
			All:     []string{"typescript", "javascript"},
		}},
	}

	// Convert Imports → ImportGraph
	if len(r.Imports) > 0 {
		totalEdges := 0
		for _, imports := range r.Imports {
			totalEdges += len(imports)
		}
		frag.ImportGraph = &architect.ImportGraph{
			ExtractionMethod: r.ExtractionMethod,
			AccuracyEstimate: r.AccuracyEstimate,
			Nodes:            len(r.Imports),
			Edges:            totalEdges,
		}
	}

	// Convert Dependencies → DependencyInfo
	if len(r.Dependencies) > 0 {
		depInfo := architect.DependencyInfo{}
		for _, d := range r.Dependencies {
			depInfo.NotableDeps = append(depInfo.NotableDeps, architect.NotableDep{
				Name:   d.Name,
				Signal: "ts_dependency",
			})
		}
		frag.Dependencies = []architect.DependencyInfo{depInfo}
	}

	frag.Metrics = &architect.CodeMetrics{
		LanguagesCount: 1,
	}

	return frag
}
