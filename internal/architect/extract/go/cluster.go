// Package golang implements package clustering heuristics.
package golang

import (
	"sort"
	"strings"
)

// applyClusteringHeuristics applies clustering strategies to group related packages.
func applyClusteringHeuristics(nodes []PackageNode, baseClusters []string) []string {
	if len(nodes) == 0 {
		return baseClusters
	}

	clusterSet := make(map[string]bool)
	for _, cluster := range baseClusters {
		clusterSet[cluster] = true
	}

	// Add layer-based clusters
	for _, node := range nodes {
		path := strings.ToLower(node.ImportPath)
		components := strings.Split(path, "/")

		for _, comp := range components {
			switch comp {
			case "cmd", "api", "handler", "http", "grpc", "rest":
				clusterSet["presentation"] = true
			case "service", "biz", "domain", "model":
				clusterSet["business"] = true
			case "repository", "storage", "db", "dao":
				clusterSet["data"] = true
			}
		}
	}

	var result []string
	for cluster := range clusterSet {
		result = append(result, cluster)
	}
	sort.Strings(result)
	return result
}
