package index

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// rrfK is the Reciprocal Rank Fusion constant.
// Higher values flatten the rank influence; 60 is the standard from the RRF paper.
const rrfK = 60

// SemanticSearch performs a hybrid search using FTS5 (BM25) with optional
// RRF fusion when vector search results are provided.
// If embedFn is nil, only FTS search is used.
func SemanticSearch(store *SQLiteStore, query string, limit int, embedFn func(string) ([]float32, error)) (*SearchResponse, error) {
	start := time.Now()
	if limit <= 0 {
		limit = 10
	}

	// Collect ranked results from each source
	ftsItems, err := ftsSearch(store, query, 100)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	var vecItems []rankedItem
	if embedFn != nil {
		vec, embedErr := embedFn(query)
		if embedErr == nil && len(vec) > 0 {
			vecItems, err = vectorSearch(store, vec, 100)
			if err != nil {
				vecItems = nil
			}
		}
	}

	// Fuse via RRF or use FTS-only
	var results []SearchResult
	if len(vecItems) > 0 {
		results = rrfFuse(store, ftsItems, vecItems, limit)
	} else {
		results = expandRankedItems(store, ftsItems, limit, "fts")
	}

	resp := &SearchResponse{
		Query:   query,
		Mode:    "semantic",
		Results: results,
		Total:   len(results),
	}
	resp.Duration = time.Since(start).Round(time.Millisecond).String()
	return resp, nil
}

// FindSearch performs an exact FTS5 keyword/identifier search.
// When regex is true, the query is treated as a LIKE pattern on symbol_name
// and content rather than FTS.
func FindSearch(store *SQLiteStore, query string, regex bool, limit int) (*SearchResponse, error) {
	start := time.Now()
	if limit <= 0 {
		limit = 20
	}

	var results []SearchResult
	var err error

	if regex {
		results, err = regexSearch(store, query, limit)
	} else {
		results, err = ftsExactSearch(store, query, limit)
	}
	if err != nil {
		return nil, err
	}

	resp := &SearchResponse{
		Query:   query,
		Mode:    "find",
		Results: results,
		Total:   len(results),
	}
	resp.Duration = time.Since(start).Round(time.Millisecond).String()
	return resp, nil
}

// DepsSearch performs forward or reverse dependency traversal for a module.
func DepsSearch(store *SQLiteStore, module string, reverse bool, maxDepth int) (*DepsResponse, error) {
	if maxDepth <= 0 {
		maxDepth = 3
	}

	// Find all chunks in the module directory
	moduleChunks, err := findModuleChunks(store, module)
	if err != nil {
		return nil, fmt.Errorf("find module chunks: %w", err)
	}
	if len(moduleChunks) == 0 {
		return &DepsResponse{Module: module, Depth: maxDepth, Results: nil}, nil
	}

	// Traverse edges
	visited := make(map[int64]bool)
	for _, chunkID := range moduleChunks {
		visited[chunkID] = true
	}

	var results []DepsResult
	for _, chunkID := range moduleChunks {
		deps := traverseEdges(store, chunkID, reverse, maxDepth, visited)
		results = append(results, deps...)
	}

	// Deduplicate by module name and enrich with metadata
	results = dedupDepsResults(results)
	enrichDepsResults(store, results)

	return &DepsResponse{
		Module:  module,
		Depth:   maxDepth,
		Results: results,
	}, nil
}

// ── Internal helpers ──────────────────────────────────────────────────

// rankedItem is an intermediate structure for rank fusion.
type rankedItem struct {
	chunkID int64
	score   float64
}

// ftsSearch runs an FTS5 MATCH query and returns ranked items (BM25 score).
// Now uses Store method instead of direct db access (fixes sdplab-9zb).
func ftsSearch(store *SQLiteStore, query string, limit int) ([]rankedItem, error) {
	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	results, err := store.FTS5Search(ftsQuery, limit)
	if err != nil {
		return nil, err
	}

	items := make([]rankedItem, len(results))
	for i, r := range results {
		items[i] = rankedItem{chunkID: r.ChunkID, score: r.Score}
	}
	return items, nil
}

// ftsExactSearch runs an exact FTS5 keyword search for identifier/path lookup.
// Now uses Store method instead of direct db access (fixes sdplab-9zb).
func ftsExactSearch(store *SQLiteStore, query string, limit int) ([]SearchResult, error) {
	ftsQuery := sanitizeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}

	return store.FTS5ExactSearchWithChunks(ftsQuery, limit)
}

// regexSearch performs a LIKE-based search when regex mode is requested.
// Now uses Store method instead of direct db access (fixes sdplab-9zb).
func regexSearch(store *SQLiteStore, pattern string, limit int) ([]SearchResult, error) {
	likePattern := regexToLike(pattern)
	return store.RegexSearchWithChunks(likePattern, limit)
}

// vectorSearch is a placeholder for vector similarity search.
// Returns results when an embedding function provides vectors.
func vectorSearch(store *SQLiteStore, queryVec []float32, limit int) ([]rankedItem, error) {
	model, _ := store.GetMeta("embedding_model")
	if model == "" || model == "none" {
		return nil, nil
	}
	// Future: implement actual vector search via sqlite-vec or similar.
	return nil, nil
}

// rrfFuse combines results from FTS and vector searches using Reciprocal Rank Fusion.
func rrfFuse(store *SQLiteStore, fts, vec []rankedItem, limit int) []SearchResult {
	// Build rank maps: chunkID -> [fts_rank, vec_rank]
	rankMap := make(map[int64][2]int)

	for i, item := range fts {
		r := rankMap[item.chunkID]
		r[0] = i + 1 // 1-indexed rank
		rankMap[item.chunkID] = r
	}
	for i, item := range vec {
		r := rankMap[item.chunkID]
		r[1] = i + 1
		rankMap[item.chunkID] = r
	}

	// Compute RRF scores
	type scored struct {
		id    int64
		score float64
	}
	scoredItems := make([]scored, 0, len(rankMap))
	for id, ranks := range rankMap {
		score := 0.0
		if ranks[0] > 0 {
			score += 1.0 / float64(rrfK+ranks[0])
		}
		if ranks[1] > 0 {
			score += 1.0 / float64(rrfK+ranks[1])
		}
		scoredItems = append(scoredItems, scored{id: id, score: score})
	}

	// Sort by score descending using O(N log N) sort
	slices.SortFunc(scoredItems, func(a, b scored) int {
		switch {
		case a.score > b.score:
			return -1
		case a.score < b.score:
			return 1
		default:
			return 0
		}
	})

	if len(scoredItems) > limit {
		scoredItems = scoredItems[:limit]
	}

	// Batch-load chunks to avoid N+1 queries
	ids := make([]int64, len(scoredItems))
	for i, item := range scoredItems {
		ids[i] = item.id
	}
	chunkMap, err := store.LoadChunksByIDs(ids)
	if err != nil {
		return nil
	}

	results := make([]SearchResult, 0, len(scoredItems))
	for _, item := range scoredItems {
		chunk, ok := chunkMap[item.id]
		if !ok {
			continue
		}
		results = append(results, SearchResult{
			Chunk:    *chunk,
			Score:    math.Round(item.score*10000) / 10000,
			MatchSrc: "fused",
		})
	}
	return results
}

// expandRankedItems fetches full chunk data for ranked items and returns SearchResults.
// Uses a single batch query instead of N individual GetChunk calls.
func expandRankedItems(store *SQLiteStore, items []rankedItem, limit int, matchSrc string) []SearchResult {
	if len(items) > limit {
		items = items[:limit]
	}

	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.chunkID
	}

	chunkMap, err := store.LoadChunksByIDs(ids)
	if err != nil {
		return nil
	}

	results := make([]SearchResult, 0, len(items))
	for _, item := range items {
		chunk, ok := chunkMap[item.chunkID]
		if !ok {
			continue
		}
		results = append(results, SearchResult{
			Chunk:    *chunk,
			Score:    math.Round(item.score*10000) / 10000,
			MatchSrc: matchSrc,
		})
	}
	return results
}

// sanitizeFTSQuery cleans a user query for safe FTS5 matching.
func sanitizeFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}

	words := strings.Fields(query)
	var safe []string
	for _, w := range words {
		w = strings.Trim(w, `"'*`)
		if len(w) < 2 {
			continue
		}
		safe = append(safe, fmt.Sprintf(`"%s"`, w))
	}

	if len(safe) == 0 {
		return ""
	}
	return strings.Join(safe, " OR ")
}

// regexToLike converts a simple regex pattern to a SQL LIKE pattern.
func regexToLike(pattern string) string {
	result := strings.ReplaceAll(pattern, ".*", "%")
	result = strings.ReplaceAll(result, ".+", "%")
	result = strings.ReplaceAll(result, ".", "_")
	result = strings.TrimPrefix(result, "^")
	result = strings.TrimSuffix(result, "$")
	if !strings.Contains(result, "%") {
		result = "%" + result + "%"
	}
	return result
}

// findModuleChunks returns all chunk IDs whose file_path starts with the module prefix.
// Now uses Store method instead of direct db access (fixes sdplab-9zb).
func findModuleChunks(store *SQLiteStore, module string) ([]int64, error) {
	prefix := strings.TrimSuffix(module, "/") + "/"
	return store.QueryChunksByFilePrefix(prefix, strings.TrimSuffix(module, "/"))
}

// traverseEdges follows forward or reverse edges from a chunk using BFS.
// Now uses Store method instead of direct db access (fixes sdplab-9zb).
func traverseEdges(store *SQLiteStore, chunkID int64, reverse bool, maxDepth int, visited map[int64]bool) []DepsResult {
	var results []DepsResult

	type frontier struct {
		id    int64
		depth int
	}
	queue := []frontier{{id: chunkID, depth: 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.depth >= maxDepth {
			continue
		}

		// Use Store method instead of direct db access (fixes sdplab-9zb)
		edges, err := store.QueryEdgesBySourceOrTarget(current.id, reverse)
		if err != nil {
			continue
		}

		var nextEntries []frontier
		for _, edge := range edges {
			nextID := edge.ID
			filePath := edge.FilePath

			if visited[nextID] {
				continue
			}
			visited[nextID] = true

			modName := extractModuleFromPath(filePath)
			relation := "forward"
			if reverse {
				relation = "reverse"
			}
			results = append(results, DepsResult{
				ModuleName: modName,
				Path:       filePath,
				Relation:   relation,
			})

			nextEntries = append(nextEntries, frontier{id: nextID, depth: current.depth + 1})
		}

		queue = append(queue, nextEntries...)
	}

	return results
}

// extractModuleFromPath derives a module name from a file path.
func extractModuleFromPath(filePath string) string {
	parts := strings.Split(filePath, "/")
	if len(parts) <= 1 {
		return filePath
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

// dedupDepsResults deduplicates dependency results by module name.
func dedupDepsResults(results []DepsResult) []DepsResult {
	seen := make(map[string]bool)
	var deduped []DepsResult
	for _, r := range results {
		if !seen[r.ModuleName] {
			seen[r.ModuleName] = true
			deduped = append(deduped, r)
		}
	}
	return deduped
}

// enrichDepsResults fills in metadata from the modules table.
// Tries lookup by module name first, then by path.
func enrichDepsResults(store *SQLiteStore, results []DepsResult) {
	for i := range results {
		mm := lookupModule(store, results[i].ModuleName)
		if mm != nil {
			results[i].LOC = mm.Loc
			results[i].IsHotspot = mm.IsHotspot
			results[i].BusFactor = mm.BusFactor
		}
	}
}

// lookupModule tries to find a module by name, then by path.
// Now uses Store method instead of direct db access (fixes sdplab-9zb).
func lookupModule(store *SQLiteStore, moduleRef string) *ModuleMeta {
	// Try by name first
	if mm, err := store.GetModuleMeta(moduleRef); err == nil && mm != nil {
		return mm
	}
	// Try by path using Store method (fixes sdplab-9zb)
	if mm, err := store.QueryModuleByPath(moduleRef); err == nil && mm != nil {
		return mm
	}
	return nil
}
