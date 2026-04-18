package extract

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sdp_dev/internal/architect"
	"sdp_dev/internal/architect/extract/java"
	"sdp_dev/internal/architect/extract/python"
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

		for _, coupling := range graph.RuntimeCouplings {
			kind := architect.EdgeRuntimeBridge
			protocol := "py4j"
			target := "jvm"
			if coupling.Type == "spark_context" {
				kind = architect.EdgeRPC
				protocol = "spark"
			}
			frag.Edges = append(frag.Edges, architect.StructuralEdge{
				Source:     coupling.File,
				Target:     target,
				Kind:       kind,
				Protocol:   protocol,
				Confidence: 0.85,
			})
		}

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

// Extract analyzes the Java/Kotlin/Scala repository at repoRoot and returns a ProfileFragment.
func (JavaAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	// Check if this is a Java/Kotlin/Scala project by looking for common markers.
	hasJavaMarkers := false
	for _, marker := range []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(repoRoot, marker)); err == nil {
			hasJavaMarkers = true
			break
		}
	}
	// Also check for any JVM source files.
	if !hasJavaMarkers {
		if err := filepath.Walk(repoRoot, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			name := fi.Name()
			if strings.HasSuffix(name, ".java") || strings.HasSuffix(name, ".kt") || strings.HasSuffix(name, ".kts") || strings.HasSuffix(name, ".scala") {
				hasJavaMarkers = true
				return filepath.SkipDir // Found a Java/Kotlin file, stop walking
			}
			return nil
		}); err == nil && hasJavaMarkers {
			// Found JVM source files.
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
			All:     []string{"java", "kotlin", "scala"},
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

		// Build module directory map for directory-based clustering.
		moduleDirMap := buildModuleDirMap(r.Modules)

		packageDirToCluster := make(map[string]string, len(r.ImportGraph.PackageImports))
		packageNameToCluster := make(map[string]string, len(r.ImportGraph.PackageImports))
		clusterIndex := make(map[string]int)
		for pkgDir := range r.ImportGraph.PackageImports {
			clusterID := javaClusterID(pkgDir, moduleDirMap)
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

		// Adaptive refinement: split oversized non-module clusters (>50 packages).
		const maxClusterSize = 50
		for i := 0; i < len(importGraph.Clusters); i++ {
			if len(importGraph.Clusters[i].Packages) <= maxClusterSize {
				continue
			}
			// Don't split module-derived clusters — they represent real boundaries.
			if _, isModule := moduleDirMap[importGraph.Clusters[i].ID]; isModule {
				continue
			}
			// Split by increasing prefix depth (4 segments instead of 3).
			splitClusters := make(map[string]*architect.ImportCluster)
			for _, pkg := range importGraph.Clusters[i].Packages {
				pkgName := javaPackageName(pkg)
				newID := javaImportPrefix(pkgName, 4)
				if newID == "" || newID == importGraph.Clusters[i].ID {
					newID = pkgName // Full package as cluster
				}
				if _, ok := splitClusters[newID]; !ok {
					splitClusters[newID] = &architect.ImportCluster{ID: newID}
				}
				splitClusters[newID].Packages = append(splitClusters[newID].Packages, pkg)
				packageDirToCluster[pkg] = newID
				if pkgName != "" {
					packageNameToCluster[pkgName] = newID
				}
			}
			// Replace oversized cluster with split results.
			replacement := make([]architect.ImportCluster, 0, len(importGraph.Clusters)-1+len(splitClusters))
			for j, c := range importGraph.Clusters {
				if j == i {
					for _, sc := range splitClusters {
						replacement = append(replacement, *sc)
					}
					continue
				}
				replacement = append(replacement, c)
			}
			importGraph.Clusters = replacement
			// Rebuild clusterIndex.
			clusterIndex = make(map[string]int, len(importGraph.Clusters))
			for j, c := range importGraph.Clusters {
				clusterIndex[c.ID] = j
			}
			i-- // recheck current index
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

				// Priority 1: declared-dependency EdgeSync edges (source of truth).
				// These come from submodule pom.xml/build.gradle <dependency> declarations.
				var declaredEdges []architect.StructuralEdge
				if len(r.SubmoduleDeps) > 0 && len(r.Modules) > 0 {
					artifactToModule := buildArtifactToModuleMap(r.SubmoduleDeps, r.Modules)
					for _, sd := range r.SubmoduleDeps {
						if sd.ModuleDir == "" {
							continue // skip root pom.xml
						}
						sourceSlug := resolveModuleFromDir(sd.ModuleDir, artifactToModule)
						if sourceSlug == "" {
							continue
						}
						for _, dep := range sd.Dependencies {
							if !mavenScopeIncluded[dep.Scope] {
								continue
							}
							targetSlug := resolveModuleFromArtifact(dep.Artifact, artifactToModule)
							if targetSlug == "" || targetSlug == sourceSlug {
								continue
							}
							declaredEdges = append(declaredEdges, architect.StructuralEdge{
								Source:     sourceSlug,
								Target:     targetSlug,
								Kind:       architect.EdgeSync,
								Weight:     1,
								Confidence: 0.95,
							})
						}
					}
					declaredEdges = dedupStructuralEdges(declaredEdges)
				}

			if len(declaredEdges) > 0 {
				// Use declared dependencies as source of truth.
				frag.Edges = append(frag.Edges, declaredEdges...)
			} else if moduleDirMap != nil {
				// Fallback: import-graph-derived edges when no build descriptors found.
				clusterToModule := make(map[string]string)
				for _, c := range importGraph.Clusters {
					for _, pkg := range c.Packages {
						if slug := moduleForDir(pkg, moduleDirMap); slug != "" {
							clusterToModule[c.ID] = slug
						}
					}
				}
				moduleEdgeWeight := make(map[[2]string]int)
				for pkgDir, imports := range r.ImportGraph.PackageImports {
					fromCluster := packageDirToCluster[pkgDir]
					if fromCluster == "" {
						continue
					}
					fromMod := clusterToModule[fromCluster]
					for _, imp := range imports {
						toCluster := javaImportTargetCluster(imp, packageNameToCluster)
						if toCluster == "" || toCluster == fromCluster {
							continue
						}
						toMod := clusterToModule[toCluster]
						if toMod == "" || fromMod == "" || fromMod == toMod {
							continue
						}
						moduleEdgeWeight[[2]string{fromMod, toMod}]++
					}
				}
				// Resolve bidirectional edges: keep only the stronger direction.
				seen := make(map[string]bool)
				for pair, weight := range moduleEdgeWeight {
					reverse := [2]string{pair[1], pair[0]}
					reverseWeight := moduleEdgeWeight[reverse]
					key := pair[0] + "->" + pair[1]
					revKey := reverse[0] + "->" + reverse[1]
					if seen[key] || seen[revKey] {
						continue
					}
					if reverseWeight > weight {
						seen[revKey] = true
						continue
					}
					seen[key] = true
					frag.Edges = append(frag.Edges, architect.StructuralEdge{
						Source: pair[0],
						Target: pair[1],
						Kind:   architect.EdgeSync,
						Weight: weight,
					})
				}
			}
	}

	for _, coupling := range r.RuntimeCouplings {
		kind := architect.EdgeRPC
		protocol := "spark-rpc"
		target := "spark-runtime"
		if coupling.Type == "py4j_gateway" {
			kind = architect.EdgeRuntimeBridge
			protocol = "py4j"
		} else if coupling.Type == "grpc" {
			kind = architect.EdgeRPC
			protocol = "grpc"
			target = "spark-connect"
		}
		frag.Edges = append(frag.Edges, architect.StructuralEdge{
			Source:     coupling.File,
			Target:     target,
			Kind:       kind,
			Protocol:   protocol,
			Confidence: 0.8,
		})
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
			LanguageBreakdown: buildLangBreakdownFromMetadata(r.Metadata),
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

func javaClusterID(pkgDir string, moduleDirMap map[string]string) string {
	// Check if this package belongs to a Maven module by directory path.
	if moduleDirMap != nil {
		if slug := moduleForDir(pkgDir, moduleDirMap); slug != "" {
			return slug
		}
	}

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

// buildModuleDirMap maps normalized module directory paths to module slugs.
// Unlike the old buildModulePrefixMap (which used Java package name prefixes),
// this maps by filesystem directory path, which correctly distinguishes root-level
// modules like "core", "streaming", "mllib" that all share the org.apache.spark prefix.
func buildModuleDirMap(modules []string) map[string]string {
	if len(modules) == 0 {
		return nil
	}

	result := make(map[string]string)
	for _, mod := range modules {
		normalized := filepath.ToSlash(strings.Trim(mod, "/"))
		if normalized == "" {
			continue
		}
		slug := moduleSlug(mod)
		result[normalized] = slug
	}
	return result
}

// moduleForDir returns the module slug for a package directory by longest-prefix
// matching against the module directory map.
func moduleForDir(pkgDir string, moduleDirMap map[string]string) string {
	normalized := filepath.ToSlash(filepath.Clean(pkgDir))
	bestLen := 0
	bestSlug := ""
	for modPath, slug := range moduleDirMap {
		if strings.Contains(normalized, modPath+"/") {
			if len(modPath) > bestLen {
				bestLen = len(modPath)
				bestSlug = slug
			}
		}
	}
	return bestSlug
}

// moduleSlug converts a Maven module path like "sql/core" to a slug like "spark-sql-core".
func moduleSlug(mod string) string {
	s := filepath.ToSlash(mod)
	s = strings.TrimPrefix(s, "/")
	parts := strings.Split(s, "/")
	return "spark-" + strings.Join(parts, "-")
}

func javaPackageName(pkgDir string) string {
	normalized := filepath.ToSlash(pkgDir)
	for _, marker := range []string{"src/main/java/", "src/test/java/", "src/main/kotlin/", "src/test/kotlin/", "src/main/scala/", "src/test/scala/"} {
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

// buildLangBreakdownFromMetadata creates a language breakdown map from
// the Java extraction result metadata (java_files, kotlin_files, scala_files).
func buildLangBreakdownFromMetadata(metadata map[string]string) map[string]int {
	result := make(map[string]int)
	type mapping struct {
		key string
		ext string
	}
	for _, m := range []mapping{
		{"java_files", ".java"},
		{"kotlin_files", ".kt"},
		{"scala_files", ".scala"},
	} {
		if v, ok := metadata[m.key]; ok {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
				result[m.ext] = n
			}
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Build-system declared dependency helpers
// ---------------------------------------------------------------------------

// mavenScopeIncluded defines which Maven dependency scopes are included in
// inter-module EdgeSync edges. Compile and runtime scopes represent actual
// module coupling; test/provided/system scopes are excluded.
var mavenScopeIncluded = map[string]bool{
	"":        true, // default scope = compile
	"compile": true,
	"runtime": true,
}

// buildArtifactToModuleMap creates a mapping from Maven artifactId to module
// slug, using the SubmoduleBuildDeps entries as the authoritative source.
// The modules list (from pom.xml <modules>) is used to derive clean slugs
// that match the container IDs produced by the pipeline, even when the repo
// is nested inside a version subdirectory (e.g. spark-3.5.7/).
func buildArtifactToModuleMap(submoduleDeps []SubmoduleBuildDeps, modules []string) map[string]string {
	if len(submoduleDeps) == 0 || len(modules) == 0 {
		return nil
	}

	// Build clean module path → clean slug from the canonical modules list.
	cleanSlugs := make(map[string]string, len(modules))
	for _, mod := range modules {
		normalized := filepath.ToSlash(strings.Trim(mod, "/"))
		if normalized == "" {
			continue
		}
		cleanSlugs[normalized] = moduleSlug(normalized)
	}

	// For each submodule dep entry, resolve its ModuleDir to a clean slug
	// by suffix-matching against the canonical module paths.
	result := make(map[string]string)
	for _, sd := range submoduleDeps {
		if sd.ModuleDir == "" {
			continue
		}
		moduleDirNorm := filepath.ToSlash(sd.ModuleDir)

		// Find matching clean module path by suffix.
		slug := ""
		for cleanPath, cleanSlug := range cleanSlugs {
			if moduleDirNorm == cleanPath || strings.HasSuffix(moduleDirNorm, "/"+cleanPath) {
				slug = cleanSlug
				break
			}
		}
		if slug == "" {
			slug = moduleSlug(sd.ModuleDir) // fallback
		}

		// Map the module directory path directly.
		result[sd.ModuleDir] = slug
		// Map the artifactId if available.
		if sd.ArtifactID != "" {
			result[sd.ArtifactID] = slug
			// Also map the normalized form (without Scala version suffix).
			normalized := normalizeArtifactID(sd.ArtifactID)
			result[normalized] = slug
		}
	}
	return result
}

// normalizeArtifactID strips Scala version suffixes and common Maven classifiers
// from an artifactId. For example: "spark-core_2.13" -> "spark-core".
func normalizeArtifactID(artifactID string) string {
	// Strip Scala version suffixes like _2.13, _2.12, _2.11
	if idx := strings.LastIndex(artifactID, "_"); idx > 0 {
		suffix := artifactID[idx+1:]
		// Check if suffix looks like a Scala version (2.N)
		if len(suffix) > 0 && suffix[0] == '2' && strings.Contains(suffix, ".") {
			artifactID = artifactID[:idx]
		}
	}
	return artifactID
}

// resolveModuleFromDir resolves a filesystem module directory path to its
// clean module slug by looking it up in the artifactToModule map.
func resolveModuleFromDir(moduleDir string, artifactToModule map[string]string) string {
	if moduleDir == "" || artifactToModule == nil {
		return ""
	}
	if slug, ok := artifactToModule[moduleDir]; ok {
		return slug
	}
	// Try normalized path.
	normalized := filepath.ToSlash(moduleDir)
	if slug, ok := artifactToModule[normalized]; ok {
		return slug
	}
	return ""
}

// resolveModuleFromArtifact attempts to map a Maven artifactId to a module
// slug using the artifactToModule map, with multiple matching strategies.
func resolveModuleFromArtifact(artifact string, artifactToModule map[string]string) string {
	if artifact == "" || artifactToModule == nil {
		return ""
	}
	// Direct match.
	if slug, ok := artifactToModule[artifact]; ok {
		return slug
	}
	// Normalized match (strip Scala version).
	normalized := normalizeArtifactID(artifact)
	if slug, ok := artifactToModule[normalized]; ok {
		return slug
	}
	// Suffix-based fallback: check if any key is a suffix of the artifact.
	for key, slug := range artifactToModule {
		if len(key) > 0 && len(key) < len(normalized) {
			if strings.HasSuffix(normalized, key) {
				return slug
			}
		}
	}
	return ""
}

// dedupStructuralEdges deduplicates edges by (source, target, kind) tuple.
func dedupStructuralEdges(edges []architect.StructuralEdge) []architect.StructuralEdge {
	seen := make(map[string]bool, len(edges))
	result := make([]architect.StructuralEdge, 0, len(edges))
	for _, e := range edges {
		key := e.Source + "->" + e.Target + ":" + string(e.Kind)
		if !seen[key] {
			seen[key] = true
			result = append(result, e)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// NewPythonAdapter — wraps new python.PythonExtractor to implement architect.Extractor
// ---------------------------------------------------------------------------

// NewPythonAdapter wraps the new python.PythonExtractor from the python subpackage.
type NewPythonAdapter struct{}

// Name returns the extractor identifier.
func (NewPythonAdapter) Name() string { return "python" }

// Extract analyzes the Python repository at repoRoot and returns a ProfileFragment.
func (NewPythonAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
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

	e := &python.PythonExtractor{}
	result, err := e.Extract(ctx, repoRoot)
	if err != nil {
		return nil, err
	}

	return convertExtractionResult(result), nil
}

// ---------------------------------------------------------------------------
// NewJavaAdapter — wraps new java.Extractor to implement architect.Extractor
// ---------------------------------------------------------------------------

// NewJavaAdapter wraps the new java.Extractor from the java subpackage.
type NewJavaAdapter struct{}

// Name returns the extractor identifier.
func (NewJavaAdapter) Name() string { return "java" }

// Extract analyzes the Java/Kotlin repository at repoRoot and returns a ProfileFragment.
func (NewJavaAdapter) Extract(ctx context.Context, repoRoot string) (*architect.ProfileFragment, error) {
	// Check if this is a Java/Kotlin project by looking for common markers
	hasJavaMarkers := false
	for _, marker := range []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle"} {
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
			if strings.HasSuffix(fi.Name(), ".java") || strings.HasSuffix(fi.Name(), ".kt") {
				hasJavaMarkers = true
				return filepath.SkipDir
			}
			return nil
		}); err == nil && hasJavaMarkers {
			// Found Java/Kotlin files
		}
	}

	if !hasJavaMarkers {
		return &architect.ProfileFragment{}, nil // Not a Java/Kotlin project
	}

	e := java.NewExtractor(repoRoot)
	result, err := e.Extract()
	if err != nil {
		return nil, err
	}

	return convertJavaResultToFragment(result), nil
}

// convertJavaResultToFragment converts the new java.JavaExtractionResult to ProfileFragment.
func convertJavaResultToFragment(result *java.JavaExtractionResult) *architect.ProfileFragment {
	if result == nil {
		return &architect.ProfileFragment{}
	}

	frag := &architect.ProfileFragment{
		Languages: []architect.LanguageInfo{{
			Primary: result.Language,
			All:     []string{result.Language},
		}},
	}

	// Convert import graph
	if len(result.ImportGraph.PackageImports) > 0 {
		importGraph := &architect.ImportGraph{
			ExtractionMethod: result.ExtractionMethod,
			AccuracyEstimate: result.AccuracyEstimate,
		}

		// Build simple clusters from package directories
		clusters := make(map[string]*architect.ImportCluster)
		for pkgDir := range result.ImportGraph.PackageImports {
			clusterID := filepath.Base(pkgDir)
			if _, exists := clusters[clusterID]; !exists {
				clusters[clusterID] = &architect.ImportCluster{
					ID:      clusterID,
					Packages: []string{pkgDir},
				}
			} else {
				clusters[clusterID].Packages = append(clusters[clusterID].Packages, pkgDir)
			}
			importGraph.Nodes++
		}

		for _, c := range clusters {
			importGraph.Clusters = append(importGraph.Clusters, *c)
		}

		frag.ImportGraph = importGraph
	}

	// Convert frameworks to modules (canonical field)
	if len(result.Frameworks) > 0 {
		for _, fw := range result.Frameworks {
			frag.Modules = append(frag.Modules, architect.Module{
				ID:       fw.Name,
				Name:     fw.Name,
				Language: result.Language,
			})
		}
	}

	// Convert build system
	if result.BuildSystem != nil {
		depInfo := architect.DependencyInfo{
			Language: result.Language,
		}
		for _, dep := range result.BuildSystem.Dependencies {
			depInfo.NotableDeps = append(depInfo.NotableDeps, architect.NotableDep{
				Name:   dep.Group + ":" + dep.Artifact,
				Signal: detectDepSignal(dep.Artifact),
			})
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
