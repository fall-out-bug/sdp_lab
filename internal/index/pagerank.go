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
	// Load all edges
	edges, err := loadAllEdges(store)
	if err != nil {
		return 0, fmt.Errorf("pagerank: load edges: %w", err)
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

	// Write back to database
	updated := 0
	for i, nodeID := range nodes {
		pr := math.Round(scores[i]*1e6) / 1e6
		_, err := store.db.Exec("UPDATE chunks SET pagerank = ? WHERE id = ?", pr, nodeID)
		if err != nil {
			continue
		}
		updated++
	}

	return updated, nil
}

// loadAllEdges loads all edges from the database.
func loadAllEdges(store *SQLiteStore) ([]Edge, error) {
	rows, err := store.db.Query("SELECT id, source_id, target_id, relation, weight FROM edges")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var e Edge
		if err := rows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.Relation, &e.Weight); err != nil {
			continue
		}
		edges = append(edges, e)
	}
	return edges, nil
}
