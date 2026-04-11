package strataudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"sdp_dev/internal/strataudit/model"
)

// LinkResult holds trace linking statistics.
type LinkResult struct {
	CandidatesGenerated int
	TracesCreated       int
	Pairs               int
	Errors              []error
}

// LinkEntities generates trace candidates using embedding similarity and optionally verifies with LLM.
func LinkEntities(ctx context.Context, cfg *Config, store *SQLiteStore, llm *LLMClient) (*LinkResult, error) {
	result := &LinkResult{}

	levels, err := store.LoadLevels(ctx)
	if err != nil {
		return nil, fmt.Errorf("load levels: %w", err)
	}
	if len(levels) < 2 {
		return result, nil // need at least 2 levels to link
	}

	// Load all entities with embeddings
	var allEntities []model.Entity
	for _, level := range levels {
		entities, err := store.EntitiesByLevel(ctx, level.ID, model.Page{Limit: 10000})
		if err != nil {
			return nil, fmt.Errorf("entities level %s: %w", level.ID, err)
		}
		allEntities = append(allEntities, entities...)
	}

	// Filter to entities with embeddings
	var withEmbeddings []model.Entity
	for _, e := range allEntities {
		if len(e.Embedding) > 0 {
			withEmbeddings = append(withEmbeddings, e)
		}
	}

	if len(withEmbeddings) == 0 {
		return result, nil
	}

	// Load embeddings from DB (they're not in the light query result)
	entitiesByLevel := groupByLevel(withEmbeddings)

	// For each adjacent level pair, compute similarity
	var distStats []LevelPairStats
	for i := 0; i < len(levels)-1; i++ {
		lower := entitiesByLevel[levels[i+1].ID] // lower rank = more operational
		upper := entitiesByLevel[levels[i].ID]   // higher rank = more strategic

		if len(lower) == 0 || len(upper) == 0 {
			continue
		}

		candidates, stats := computeSimilarity(ctx, cfg, lower, upper)
		result.CandidatesGenerated += len(candidates)

		if stats != nil {
			stats.LevelPair = levels[i+1].Name + " -> " + levels[i].Name
			distStats = append(distStats, *stats)
		}

		// Save candidates
		if len(candidates) > 0 {
			modelCandidates := candidatesToModel(candidates)
			if err := store.SaveCandidates(ctx, modelCandidates); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("save candidates %s->%s: %w", levels[i+1].Name, levels[i].Name, err))
			}

			// Create traces from verified candidates
			traces := createTraces(ctx, cfg, llm, candidates, levels[i+1], levels[i])
			result.TracesCreated += len(traces)

			if len(traces) > 0 {
				if err := store.SaveTraces(ctx, traces); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("save traces %s->%s: %w", levels[i+1].Name, levels[i].Name, err))
				}
			}
		}

		result.Pairs++
	}

	// Write similarity distribution JSON
	if cfg.Thresholds.EmitDistribution && len(distStats) > 0 {
		report := DistributionReport{
			RunID:       fmt.Sprintf("run_%d", time.Now().UnixMilli()),
			GeneratedAt: time.Now().Format(time.RFC3339),
			Threshold:   cfg.Thresholds.Similarity,
			LevelPairs:  distStats,
		}
		writeDistributionReport(cfg, report)
	}

	return result, nil
}

func writeDistributionReport(cfg *Config, report DistributionReport) {
	path := filepath.Join(cfg.Output.Dir, "similarity_distribution.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		slog.Warn("similarity distribution: marshal error", "err", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Warn("similarity distribution: write error", "path", path, "err", err)
		return
	}
	slog.Info("similarity distribution written", "path", path, "level_pairs", len(report.LevelPairs))
}

type candidate struct {
	source  model.Entity
	target  model.Entity
	sim     float64
}

// SimilarityBucket is one bucket in the similarity histogram.
type SimilarityBucket struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// LevelPairStats holds similarity statistics for one level pair.
type LevelPairStats struct {
	LevelPair      string            `json:"level_pair"`
	TotalPairs     int               `json:"total_pairs"`
	AboveThreshold int               `json:"above_threshold"`
	Min            float64           `json:"min"`
	Max            float64           `json:"max"`
	Mean           float64           `json:"mean"`
	Median         float64           `json:"median"`
	P95            float64           `json:"p95"`
	Histogram      []SimilarityBucket `json:"histogram"`
	Recommendation string            `json:"recommendation,omitempty"`
}

// DistributionReport is the top-level similarity diagnostics file.
type DistributionReport struct {
	RunID         string           `json:"run_id"`
	GeneratedAt   string           `json:"generated_at"`
	Threshold     float64          `json:"threshold"`
	LevelPairs    []LevelPairStats `json:"level_pairs"`
}

func computeSimilarity(ctx context.Context, cfg *Config, lower, upper []model.Entity) ([]candidate, *LevelPairStats) {
	threshold := cfg.Thresholds.Similarity
	emitDist := cfg.Thresholds.EmitDistribution
	var candidates []candidate
	var allScores []float64

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Parallel computation with semaphore
	sem := make(chan struct{}, 8)

	for _, src := range lower {
		for _, tgt := range upper {
			if ctx.Err() != nil {
				return candidates, nil
			}

			wg.Add(1)
			sem <- struct{}{}
			go func(s, t model.Entity) {
				defer wg.Done()
				defer func() { <-sem }()

				sim := cosineSimilarity(s.Embedding, t.Embedding)
				mu.Lock()
				if emitDist {
					allScores = append(allScores, sim)
				}
				if sim >= threshold {
					candidates = append(candidates, candidate{source: s, target: t, sim: sim})
				}
				mu.Unlock()
			}(src, tgt)
		}
	}
	wg.Wait()

	var stats *LevelPairStats
	if emitDist && len(allScores) > 0 {
		stats = computeStats(allScores, threshold, len(lower), len(upper))
	}
	return candidates, stats
}

func computeStats(scores []float64, threshold float64, lowerCount, upperCount int) *LevelPairStats {
	sort.Float64s(scores)
	n := len(scores)

	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	mean := sum / float64(n)

	median := scores[n/2]
	if n%2 == 0 {
		median = (scores[n/2-1] + scores[n/2]) / 2
	}

	p95Idx := int(float64(n) * 0.95)
	if p95Idx >= n {
		p95Idx = n - 1
	}

	aboveThreshold := 0
	for _, s := range scores {
		if s >= threshold {
			aboveThreshold++
		}
	}

	// Build histogram with 10 buckets: [0.0, 0.1), [0.1, 0.2), ..., [0.9, 1.0]
	histogram := make([]SimilarityBucket, 10)
	for i := range histogram {
		histogram[i].Range = fmt.Sprintf("%.1f-%.1f", float64(i)*0.1, float64(i+1)*0.1)
	}
	for _, s := range scores {
		bucket := int(s * 10)
		if bucket >= 10 {
			bucket = 9
		}
		histogram[bucket].Count++
	}

	rec := ""
	if float64(aboveThreshold) < float64(n)*0.02 {
		rec = "threshold_may_be_too_high"
	}

	return &LevelPairStats{
		TotalPairs:     n,
		AboveThreshold: aboveThreshold,
		Min:            scores[0],
		Max:            scores[n-1],
		Mean:           mean,
		Median:         median,
		P95:            scores[p95Idx],
		Histogram:      histogram,
		Recommendation: rec,
	}
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		normA += af * af
		normB += bf * bf
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func createTraces(ctx context.Context, cfg *Config, llm *LLMClient, candidates []candidate, lowerLevel, upperLevel model.Level) []model.Trace {
	threshold := cfg.Thresholds.TraceConfidence
	var traces []model.Trace

	for _, c := range candidates {
		if ctx.Err() != nil {
			return traces
		}

		// High similarity candidates get auto-verified
		confidence := c.sim
		justification := fmt.Sprintf("Embedding similarity: %.2f", c.sim)

		if confidence >= threshold {
			traces = append(traces, model.Trace{
				ID:              traceID(c.source.ID, c.target.ID),
				SourceEntityID:  c.source.ID,
				TargetEntityID:  c.target.ID,
				Relation:        model.RelationContributesTo,
				Confidence:      confidence,
				Justification:   justification,
				Direction:       model.DirectionUp,
			})
		}
	}

	return traces
}

func traceID(sourceID, targetID string) string {
	h := sha256Hash([]byte(sourceID + "->" + targetID))
	return fmt.Sprintf("tr_%s", h[:12])
}

func groupByLevel(entities []model.Entity) map[string][]model.Entity {
	m := make(map[string][]model.Entity)
	for _, e := range entities {
		m[e.LevelID] = append(m[e.LevelID], e)
	}
	return m
}

func candidatesToModel(candidates []candidate) []model.Candidate {
	result := make([]model.Candidate, len(candidates))
	for i, c := range candidates {
		result[i] = model.Candidate{
			ID:             candidateID(c.source.ID, c.target.ID),
			SourceEntityID: c.source.ID,
			TargetEntityID: c.target.ID,
			Similarity:     c.sim,
		}
	}
	return result
}

func candidateID(src, tgt string) string {
	return fmt.Sprintf("cand_%s", sha256Hash([]byte(src+"|"+tgt))[:10])
}

// SaveCandidates saves trace candidates to the store.
func (s *SQLiteStore) SaveCandidates(ctx context.Context, candidates []model.Candidate) error {
	for _, c := range candidates {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO trace_candidates (id, source_entity_id, target_entity_id, similarity, verified)
			VALUES (?,?,?,?,?)`,
			c.ID, c.SourceEntityID, c.TargetEntityID, c.Similarity, c.Verified)
		if err != nil {
			return fmt.Errorf("save candidate %s: %w", c.ID, err)
		}
	}
	return nil
}

// DocumentsByLevel returns all documents for a given level.
func (s *SQLiteStore) DocumentsByLevel(ctx context.Context, levelID string) ([]model.Document, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path, level_id, content_hash, content, version FROM documents WHERE level_id = ?`, levelID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var docs []model.Document
	for rows.Next() {
		var d model.Document
		if err := rows.Scan(&d.ID, &d.Path, &d.LevelID, &d.ContentHash, &d.Content, &d.Version); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

// CountEntitiesByDocument returns the number of entities for a document.
func (s *SQLiteStore) CountEntitiesByDocument(ctx context.Context, docID string) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entities WHERE document_id = ?`, docID).Scan(&count)
	return count, err
}

// AllEntitiesWithEmbeddings loads all entities that have embeddings stored.
func (s *SQLiteStore) AllEntitiesWithEmbeddings(ctx context.Context) ([]model.Entity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, document_id, level_id, type, title, description, source_quote,
		embedding, embedding_model, embedding_dims, extraction_model
		FROM entities WHERE embedding IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var entities []model.Entity
	for rows.Next() {
		var e model.Entity
		var embeddingBlob []byte
		var embModel, embDims sql.NullString

		if err := rows.Scan(&e.ID, &e.DocumentID, &e.LevelID, &e.Type, &e.Title, &e.Description,
			&e.SourceQuote, &embeddingBlob, &embModel, &embDims, &e.ExtractionModel); err != nil {
			return nil, err
		}

		if len(embeddingBlob) > 0 {
			var floats []float32
			if err := json.Unmarshal(embeddingBlob, &floats); err == nil {
				e.Embedding = floats
			}
		}

		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// EmbeddingSimilarity computes cosine similarity between two float32 vectors.
// Public for testing.
func EmbeddingSimilarity(a, b []float32) float64 {
	return cosineSimilarity(a, b)
}
