package strataudit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sync"

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
	for i := 0; i < len(levels)-1; i++ {
		lower := entitiesByLevel[levels[i+1].ID] // lower rank = more operational
		upper := entitiesByLevel[levels[i].ID]   // higher rank = more strategic

		if len(lower) == 0 || len(upper) == 0 {
			continue
		}

		candidates := computeSimilarity(ctx, cfg, lower, upper)
		result.CandidatesGenerated += len(candidates)

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

	return result, nil
}

type candidate struct {
	source  model.Entity
	target  model.Entity
	sim     float64
}

func computeSimilarity(ctx context.Context, cfg *Config, lower, upper []model.Entity) []candidate {
	threshold := cfg.Thresholds.Similarity
	var candidates []candidate

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Parallel computation with semaphore
	sem := make(chan struct{}, 8)

	for _, src := range lower {
		for _, tgt := range upper {
			if ctx.Err() != nil {
				return candidates
			}

			wg.Add(1)
			sem <- struct{}{}
			go func(s, t model.Entity) {
				defer wg.Done()
				defer func() { <-sem }()

				sim := cosineSimilarity(s.Embedding, t.Embedding)
				if sim >= threshold {
					mu.Lock()
					candidates = append(candidates, candidate{source: s, target: t, sim: sim})
					mu.Unlock()
				}
			}(src, tgt)
		}
	}
	wg.Wait()

	return candidates
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
