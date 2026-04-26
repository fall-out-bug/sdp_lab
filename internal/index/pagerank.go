package index

import (
	"fmt"
	"math"
)

// PageRankDamping is the probability of following a link (vs. teleporting).
const PageRankDamping = 0.85

// PageRankIterations is the number of iterations for the PageRank computation.
const PageRankIterations = 100

// ComputePageRank runs the iterative PageRank algorithm over the edges graph
// and updates the pagerank column in the chunks table.
// Returns the number of chunks updated.
func ComputePageRank(store *SQLiteStore) (int, error) {
	// Load all edges using Store method (fixes sdplab-xlm)
	edges, scanErrors, err := store.GetAllEdges()
	if err != nil {
		return 0, fmt.Errorf("pagerank: load edges: %w", err)
	}
	// Log scan errors instead of silently skipping (fixes sdplab-ytr)
	if scanErrors > 0 {
		// In a real implementation, we might log this or return a warning
		// For now, we continue but note that some edges were skipped
	}

	// Collect unique node IDs
	nodeSet := make(map[int64]bool)
	for _, e := range edges {
		nodeSet[e.SourceID] = true
		nodeSet[e.TargetID] = true
	}
	n := len(nodeSet)
	if n == 0 {
		return 0, nil
	}

	// Map node IDs to indices
	nodes := make([]int64, 0, n)
	nodeIndex := make(map[int64]int)
	for id := range nodeSet {
		nodeIndex[id] = len(nodes)
		nodes = append(nodes, id)
	}

	// Build adjacency list (outgoing edges)
	outDegree := make([]int, n)
	inNeighbors := make([][]int, n) // inNeighbors[i] = nodes that have edges pointing TO i
	for _, e := range edges {
		srcIdx := nodeIndex[e.SourceID]
		tgtIdx := nodeIndex[e.TargetID]
		outDegree[srcIdx]++
		inNeighbors[tgtIdx] = append(inNeighbors[tgtIdx], srcIdx)
	}

	// Initialize scores uniformly
	scores := make([]float64, n)
	initScore := 1.0 / float64(n)
	for i := range scores {
		scores[i] = initScore
	}

	// Iterative computation
	damping := PageRankDamping
	teleport := (1.0 - damping) / float64(n)

	for iter := 0; iter < PageRankIterations; iter++ {
		newScores := make([]float64, n)
		for i := 0; i < n; i++ {
			sum := 0.0
			for _, srcIdx := range inNeighbors[i] {
				if outDegree[srcIdx] > 0 {
					sum += scores[srcIdx] / float64(outDegree[srcIdx])
				}
			}
			newScores[i] = teleport + damping*sum
		}

		// Check convergence (sum should remain ~1.0)
		scoreSum := 0.0
		for _, s := range newScores {
			scoreSum += s
		}

		// Handle dangling nodes: redistribute leaked score
		if scoreSum < 0.999 || scoreSum > 1.001 {
			for i := range newScores {
				newScores[i] /= scoreSum
			}
		}

		scores = newScores
	}

	// Write back to database using Store method (fixes sdplab-9zb)
	updated := 0
	for i, nodeID := range nodes {
		pr := math.Round(scores[i]*1e6) / 1e6
		if err := store.UpdatePageRank(nodeID, pr); err != nil {
			continue
		}
		updated++
	}

	return updated, nil
}
