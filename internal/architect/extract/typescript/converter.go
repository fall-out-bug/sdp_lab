package typescript

import (
	"strings"

	"sdp_dev/internal/architect"
)

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
				Name:    d.Name,
				FoundIn: 1,
				Signal:  signal,
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
	"react":            "ui_framework",
	"next":             "ssr_framework",
	"vue":              "ui_framework",
	"angular":          "ui_framework",
	"svelte":           "ui_framework",
	"express":          "web_framework",
	"fastify":          "web_framework",
	"koa":              "web_framework",
	"@nestjs":          "web_framework",
	"prisma":           "orm",
	"typeorm":          "orm",
	"sequelize":        "orm",
	"mongoose":         "odm",
	"graphql":          "graphql",
	"apollo":           "graphql",
	"redis":            "cache",
	"ioredis":          "cache",
	"kafka":            "event_driven",
	"rabbitmq":         "event_driven",
	"amqplib":          "event_driven",
	"bull":             "task_queue",
	"bullmq":           "task_queue",
	"aws-sdk":          "cloud_aws",
	"@aws-sdk":         "cloud_aws",
	"terraform":        "iac",
	"docker":           "container",
	"jest":             "testing",
	"vitest":           "testing",
	"mocha":            "testing",
	"cypress":          "e2e_testing",
	"playwright":       "e2e_testing",
	"tailwindcss":      "styling",
	"styled-component": "styling",
	"emotion":          "styling",
	"webpack":          "bundler",
	"vite":             "bundler",
	"esbuild":          "bundler",
	"rollup":           "bundler",
	"turborepo":        "monorepo",
	"lerna":            "monorepo",
	"nx":               "monorepo",
	"zod":              "validation",
	"joi":              "validation",
	"yup":              "validation",
	"axios":            "http_client",
	"got":              "http_client",
	"undici":           "http_client",
	"swagger":          "api_docs",
	"openapi":          "api_docs",
	"opentelemetry":    "observability",
	"prom-client":      "observability",
	"winston":          "logging",
	"pino":             "logging",
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
