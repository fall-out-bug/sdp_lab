// Package typescript provides a TypeScript/JavaScript ecosystem extractor.
// It detects imports, framework patterns (React, Next.js, NestJS, Express), and workspace dependencies.
package typescript

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
)

// TSExtractor builds a TSImportGraph from a TypeScript/JavaScript project root.
// It implements the architect.Extractor interface.
type TSExtractor struct{}

// NewTSExtractor creates a new TSExtractor.
func NewTSExtractor() *TSExtractor {
	return &TSExtractor{}
}

// Name returns the extractor identifier.
func (TSExtractor) Name() string { return "typescript" }

// Extract implements architect.Extractor. Returns an empty fragment (no error)
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
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if extensions[filepath.Ext(path)] {
			found = true
		}
		return nil
	})
	return found
}

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
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if !extensions[ext] {
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

// computeTSCluster returns the cluster ID for a file based on its directory.
func computeTSCluster(relPath string) string {
	dir := filepath.Dir(relPath)
	if dir == "." {
		return ""
	}
	return dir
}

// isTSGenerated returns true if the file appears to be generated.
func isTSGenerated(path string) bool {
	base := filepath.Base(path)
	for _, suffix := range generatedSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	for _, substr := range generatedPaths {
		if strings.Contains(path, substr) {
			return true
		}
	}
	return false
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
// under rootDir (skipping skipDirs).
func hasFileWithExtension(rootDir string, ext string) bool {
	found := false
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return filepath.SkipAll
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
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
