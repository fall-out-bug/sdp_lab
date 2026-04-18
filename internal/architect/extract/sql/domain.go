// Package sql provides SQL schema extraction and analysis for the AI Architect module.
package sql

import (
	"sort"
	"strings"

	"sdp_dev/internal/architect"
)

// clusterDomains groups tables into connected components using FK relationships.
func clusterDomains(tables []architect.Table, fks []architect.ForeignKey) []architect.DataDomain {
	// Build adjacency list.
	adj := make(map[string]map[string]bool)
	allTables := make(map[string]bool)
	for _, t := range tables {
		allTables[t.Name] = true
		if adj[t.Name] == nil {
			adj[t.Name] = make(map[string]bool)
		}
	}
	for _, fk := range fks {
		allTables[fk.FromTable] = true
		allTables[fk.ToTable] = true
		if adj[fk.FromTable] == nil {
			adj[fk.FromTable] = make(map[string]bool)
		}
		if adj[fk.ToTable] == nil {
			adj[fk.ToTable] = make(map[string]bool)
		}
		adj[fk.FromTable][fk.ToTable] = true
		adj[fk.ToTable][fk.FromTable] = true
	}

	visited := make(map[string]bool)
	var domains []architect.DataDomain

	// Sorted iteration for deterministic output.
	sortedTables := make([]string, 0, len(allTables))
	for t := range allTables {
		sortedTables = append(sortedTables, t)
	}
	sort.Strings(sortedTables)

	for _, start := range sortedTables {
		if visited[start] {
			continue
		}
		// BFS
		var component []string
		queue := []string{start}
		visited[start] = true
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			component = append(component, cur)
			neighbors := adj[cur]
			// Sort neighbors for determinism.
			sorted := make([]string, 0, len(neighbors))
			for n := range neighbors {
				sorted = append(sorted, n)
			}
			sort.Strings(sorted)
			for _, n := range sorted {
				if !visited[n] {
					visited[n] = true
					queue = append(queue, n)
				}
			}
		}
		sort.Strings(component)
		domains = append(domains, architect.DataDomain{
			Name:   domainName(component),
			Tables: component,
		})
	}

	return domains
}

// domainName picks a representative name for a domain.
// It uses the longest common prefix of table names, falling back to the first table name.
func domainName(tables []string) string {
	if len(tables) == 0 {
		return "unknown"
	}
	if len(tables) == 1 {
		return tables[0]
	}

	// Try to find a common prefix (before the first underscore).
	prefixes := make(map[string]int)
	for _, t := range tables {
		parts := strings.SplitN(t, "_", 2)
		if len(parts) >= 1 && parts[0] != "" {
			prefixes[parts[0]]++
		}
	}
	// Pick the prefix that covers the most tables.
	bestPrefix := ""
	bestCount := 0
	for p, c := range prefixes {
		if c > bestCount || (c == bestCount && p < bestPrefix) {
			bestPrefix = p
			bestCount = c
		}
	}
	// Use prefix only if it covers a majority.
	if bestCount > 1 && bestCount >= len(tables)/2 {
		return bestPrefix
	}
	return tables[0]
}

// ClusterDomainsByTablePrefix groups tables by their name prefix (before first underscore).
func ClusterDomainsByTablePrefix(tables []architect.Table) []architect.DataDomain {
	prefixGroups := make(map[string][]string)

	for _, t := range tables {
		parts := strings.SplitN(t.Name, "_", 2)
		prefix := parts[0]
		if prefix == "" {
			prefix = "other"
		}
		prefixGroups[prefix] = append(prefixGroups[prefix], t.Name)
	}

	var domains []architect.DataDomain
	for prefix, tableNames := range prefixGroups {
		sort.Strings(tableNames)
		domains = append(domains, architect.DataDomain{
			Name:   prefix,
			Tables: tableNames,
		})
	}

	// Sort domains by name for deterministic output
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Name < domains[j].Name
	})

	return domains
}

// GetDomainStats returns statistics about data domains.
func GetDomainStats(domains []architect.DataDomain) map[string]int {
	stats := map[string]int{
		"total_domains": len(domains),
		"max_size":      0,
		"min_size":      0,
		"avg_size":      0,
	}

	if len(domains) == 0 {
		return stats
	}

	minSize := len(domains[0].Tables)
	maxSize := len(domains[0].Tables)
	totalTables := 0

	for _, d := range domains {
		size := len(d.Tables)
		totalTables += size
		if size > maxSize {
			maxSize = size
		}
		if size < minSize {
			minSize = size
		}
	}

	stats["max_size"] = maxSize
	stats["min_size"] = minSize
	stats["avg_size"] = totalTables / len(domains)

	return stats
}

// SortDomains sorts data domains by name for deterministic output.
func SortDomains(domains []architect.DataDomain) {
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Name < domains[j].Name
	})
}

// FindRelatedTables finds all tables related to a given table through FK relationships.
func FindRelatedTables(tableName string, tables []architect.Table, fks []architect.ForeignKey) []string {
	// Build adjacency list
	adj := make(map[string][]string)
	for _, fk := range fks {
		adj[fk.FromTable] = append(adj[fk.FromTable], fk.ToTable)
		adj[fk.ToTable] = append(adj[fk.ToTable], fk.FromTable)
	}

	// BFS to find all related tables
	visited := make(map[string]bool)
	queue := []string{tableName}
	visited[tableName] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range adj[current] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	// Convert visited set to sorted slice
	result := make([]string, 0, len(visited))
	for table := range visited {
		result = append(result, table)
	}
	sort.Strings(result)

	return result
}

// GetForeignKeyGraph returns a simple graph representation of foreign key relationships.
func GetForeignKeyGraph(fks []architect.ForeignKey) map[string][]string {
	graph := make(map[string][]string)

	for _, fk := range fks {
		graph[fk.FromTable] = append(graph[fk.FromTable], fk.ToTable)
	}

	// Sort adjacency lists for deterministic output
	for table := range graph {
		sort.Strings(graph[table])
	}

	return graph
}

// DetectCyclicReferences detects circular foreign key dependencies.
func DetectCyclicReferences(fks []architect.ForeignKey) [][]string {
	// Build adjacency list
	adj := make(map[string][]string)
	allTables := make(map[string]bool)

	for _, fk := range fks {
		allTables[fk.FromTable] = true
		allTables[fk.ToTable] = true
		adj[fk.FromTable] = append(adj[fk.FromTable], fk.ToTable)
	}

	var cycles [][]string

	// Use DFS with coloring to detect cycles
	const (
		WHITE = 0 // Unvisited
		GRAY  = 1 // In progress
		BLACK = 2 // Completed
	)

	color := make(map[string]int)
	path := []string{}

	var dfs func(string) bool
	dfs = func(node string) bool {
		color[node] = GRAY
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if color[neighbor] == GRAY {
				// Found a cycle - extract it
				cycleStart := -1
				for i, p := range path {
					if p == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := append([]string{}, path[cycleStart:]...)
					cycles = append(cycles, cycle)
				}
			} else if color[neighbor] == WHITE {
				if dfs(neighbor) {
					return true
				}
			}
		}

		path = path[:len(path)-1]
		color[node] = BLACK
		return false
	}

	// Run DFS from all unvisited nodes
	for table := range allTables {
		if color[table] == WHITE {
			dfs(table)
		}
	}

	return cycles
}

// AnalyzeDomainComplexity analyzes the complexity of a data domain based on:
// - Number of tables
// - Number of foreign key relationships
// - Presence of cyclic dependencies
func AnalyzeDomainComplexity(domain architect.DataDomain, tables []architect.Table, fks []architect.ForeignKey) map[string]interface{} {
	// Count tables in domain
	tableCount := 0
	tableSet := make(map[string]bool)
	for _, t := range domain.Tables {
		tableSet[t] = true
		tableCount++
	}

	// Count foreign keys within domain
	fkCount := 0
	for _, fk := range fks {
		if tableSet[fk.FromTable] && tableSet[fk.ToTable] {
			fkCount++
		}
	}

	// Calculate complexity score
	complexityScore := float64(tableCount+fkCount) / 2.0

	// Determine complexity level
	var complexityLevel string
	switch {
	case complexityScore < 3:
		complexityLevel = "low"
	case complexityScore < 7:
		complexityLevel = "medium"
	default:
		complexityLevel = "high"
	}

	return map[string]interface{}{
		"table_count":      tableCount,
		"fk_count":         fkCount,
		"complexity_score": complexityScore,
		"complexity_level": complexityLevel,
	}
}
