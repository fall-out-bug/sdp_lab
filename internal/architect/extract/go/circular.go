// Package golang implements Tarjan's algorithm for cycle detection.
package golang

import (
	"sort"
)

// DetectCyclesTarjan performs cycle detection using Tarjan's algorithm.
func DetectCyclesTarjan(nodes []PackageNode, edges []ImportEdge) []Cycle {
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

	index := 0
	stack := make([]string, 0)
	onStack := make(map[string]bool)
	indices := make(map[string]int)
	lowLink := make(map[string]int)
	var sccs [][]string

	var strongConnect func(v string)
	strongConnect = func(v string) {
		indices[v] = index
		lowLink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adj[v] {
			if _, ok := indices[w]; !ok {
				strongConnect(w)
				if lowLink[w] < lowLink[v] {
					lowLink[v] = lowLink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowLink[v] {
					lowLink[v] = indices[w]
				}
			}
		}

		if lowLink[v] == indices[v] {
			var scc []string
			var w string
			for {
				w, stack = stack[len(stack)-1], stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			if len(scc) > 1 {
				sccs = append(sccs, scc)
			}
		}
	}

	keys := make([]string, 0, len(nodeSet))
	for k := range nodeSet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, v := range keys {
		if _, ok := indices[v]; !ok {
			strongConnect(v)
		}
	}

	var cycles []Cycle
	for _, scc := range sccs {
		if len(scc) >= 2 {
			cycles = append(cycles, Cycle(scc))
		}
	}
	return cycles
}

// DetectCyclesDFS is a fallback DFS-based cycle detection.
func DetectCyclesDFS(nodes []PackageNode, edges []ImportEdge) []Cycle {
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
	var cycles []Cycle

	buildCycle := func(from, to string) Cycle {
		var c Cycle
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
			case 0:
				parent[v] = u
				dfs(v)
			case 1:
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
		if color[k] == 0 {
			dfs(k)
		}
	}

	return cycles
}
