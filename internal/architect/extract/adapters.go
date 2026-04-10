package extract

import (
	"context"
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
	if len(r.Dependencies) > 0 {
		depInfo := architect.DependencyInfo{}
		notableMap := make(map[string]int)
		for _, d := range r.Dependencies {
			if d.Kind == "third-party" {
				notableMap[d.Name]++
			}
		}
		for name, count := range notableMap {
			depInfo.NotableDeps = append(depInfo.NotableDeps, architect.NotableDep{
				Name:    name,
				FoundIn: count,
				Signal:  "python_dependency",
			})
		}
		frag.Dependencies = []architect.DependencyInfo{depInfo}
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
