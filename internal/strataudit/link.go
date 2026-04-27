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
	"strings"
	"sync"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/strataudit/model"
)

// LinkResult holds trace linking statistics.
type LinkResult struct {
	CandidatesGenerated int
	TracesCreated       int
	Pairs               int
	Errors              []error
}

// LinkEntities generates trace candidates using embedding similarity and optionally verifies with LLM.
func LinkEntities(ctx context.Context, cfg *Config, store *SQLiteStore, runtime ModelRuntime) (*LinkResult, error) {
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
			traces, verifiedByCandidateID, diagnosticByCandidateID := createVerifiedTraces(ctx, cfg, store, runtime, candidates, levels[i+1], levels[i])
			modelCandidates := candidatesToModel(candidates, verifiedByCandidateID, diagnosticByCandidateID)
			if err := store.SaveCandidates(ctx, modelCandidates); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("save candidates %s->%s: %w", levels[i+1].Name, levels[i].Name, err))
			}
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
	source model.Entity
	target model.Entity
	sim    float64
}

// SimilarityBucket is one bucket in the similarity histogram.
type SimilarityBucket struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// LevelPairStats holds similarity statistics for one level pair.
type LevelPairStats struct {
	LevelPair      string             `json:"level_pair"`
	TotalPairs     int                `json:"total_pairs"`
	AboveThreshold int                `json:"above_threshold"`
	Min            float64            `json:"min"`
	Max            float64            `json:"max"`
	Mean           float64            `json:"mean"`
	Median         float64            `json:"median"`
	P95            float64            `json:"p95"`
	Histogram      []SimilarityBucket `json:"histogram"`
	Recommendation string             `json:"recommendation,omitempty"`
}

// DistributionReport is the top-level similarity diagnostics file.
type DistributionReport struct {
	RunID       string           `json:"run_id"`
	GeneratedAt string           `json:"generated_at"`
	Threshold   float64          `json:"threshold"`
	LevelPairs  []LevelPairStats `json:"level_pairs"`
}

type scoredPair struct {
	src, tgt model.Entity
	sim      float64
}

func computeSimilarity(ctx context.Context, cfg *Config, lower, upper []model.Entity) ([]candidate, *LevelPairStats) {
	threshold := cfg.Thresholds.Similarity
	adaptive := cfg.Thresholds.AdaptiveSimilarity
	emitDist := cfg.Thresholds.EmitDistribution || adaptive

	var pairs []scoredPair

	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, 8)

	for _, src := range lower {
		for _, tgt := range upper {
			if ctx.Err() != nil {
				return nil, nil
			}

			wg.Add(1)
			sem <- struct{}{}
			go func(s, t model.Entity) {
				defer wg.Done()
				defer func() { <-sem }()

				sim := cosineSimilarity(s.Embedding, t.Embedding)
				mu.Lock()
				pairs = append(pairs, scoredPair{src: s, tgt: t, sim: sim})
				mu.Unlock()
			}(src, tgt)
		}
	}
	wg.Wait()

	if len(pairs) == 0 {
		return nil, nil
	}

	// Extract scores for stats
	allScores := make([]float64, len(pairs))
	for i, p := range pairs {
		allScores[i] = p.sim
	}

	// Compute effective threshold (adaptive)
	effectiveThreshold := threshold
	if adaptive {
		sort.Float64s(allScores)
		p95Idx := int(float64(len(allScores)) * 0.95)
		if p95Idx >= len(allScores) {
			p95Idx = len(allScores) - 1
		}
		p95 := allScores[p95Idx]
		aboveOriginal := 0
		for _, s := range allScores {
			if s >= threshold {
				aboveOriginal++
			}
		}
		ratio := float64(aboveOriginal) / float64(len(allScores))
		if ratio < 0.02 && p95 > 0.2 {
			effectiveThreshold = p95
			if effectiveThreshold > threshold {
				effectiveThreshold = threshold
			}
			slog.Info("adaptive threshold applied",
				"original", threshold,
				"effective", effectiveThreshold,
				"p95", p95,
				"above_original_pct", fmt.Sprintf("%.1f%%", ratio*100),
				"total_pairs", len(allScores))
		}
	}

	// Filter candidates by effective threshold
	var candidates []candidate
	for _, p := range pairs {
		if p.sim >= effectiveThreshold {
			candidates = append(candidates, candidate{source: p.src, target: p.tgt, sim: p.sim})
		}
	}

	var stats *LevelPairStats
	if emitDist && len(allScores) > 0 {
		stats = computeStats(allScores, effectiveThreshold, len(lower), len(upper))
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

func createVerifiedTraces(ctx context.Context, cfg *Config, store *SQLiteStore, runtime ModelRuntime, candidates []candidate, lowerLevel, upperLevel model.Level) ([]model.Trace, map[string]string, map[string]string) {
	autoThreshold := cfg.Thresholds.AutoVerifySimilarity
	traceThreshold := cfg.Thresholds.TraceConfidence
	budget := cfg.Thresholds.LLMVerifyBudget
	var traces []model.Trace
	verifiedByCandidateID := make(map[string]string)
	diagnosticByCandidateID := make(map[string]string, len(candidates))

	prioritized := append([]candidate(nil), candidates...)
	sort.SliceStable(prioritized, func(i, j int) bool {
		iAuto := prioritized[i].sim >= autoThreshold
		jAuto := prioritized[j].sim >= autoThreshold
		if iAuto != jAuto {
			return iAuto
		}
		return prioritized[i].sim > prioritized[j].sim
	})

	var llmCount int

	for _, c := range prioritized {
		if ctx.Err() != nil {
			return traces, verifiedByCandidateID, diagnosticByCandidateID
		}
		id := candidateID(c.source.ID, c.target.ID)

		if c.sim < traceThreshold {
			diagnosticByCandidateID[id] = string(model.TraceCandidateDiagnosticBelowTraceConfidence)
			continue
		}
		if !hasTraceEvidence(c.source) || !hasTraceEvidence(c.target) {
			diagnosticByCandidateID[id] = string(model.TraceCandidateDiagnosticQuoteEvidenceMissing)
			continue
		}
		if runtime == nil {
			diagnosticByCandidateID[id] = string(model.TraceCandidateDiagnosticVerificationUnavailable)
			continue
		}
		if budget <= 0 {
			diagnosticByCandidateID[id] = string(model.TraceCandidateDiagnosticVerificationBudgetExhausted)
			continue
		}

		budget--
		verified, relation, conf, justification := llmVerifyPair(ctx, store, runtime, cfg, c, lowerLevel, upperLevel)
		if !verified {
			diagnosticByCandidateID[id] = string(model.TraceCandidateDiagnosticLLMVerificationRejected)
			continue
		}

		trace := buildVerifiedTrace(c, relation, conf, justification)
		traces = append(traces, trace)
		verifiedByCandidateID[id] = trace.ID
		diagnosticByCandidateID[id] = string(model.TraceCandidateDiagnosticLLMVerified)
		llmCount++
	}

	slog.Info("trace verification", "llm", llmCount, "total", len(traces))
	return traces, verifiedByCandidateID, diagnosticByCandidateID
}

type llmVerifyResult struct {
	Related       jsonBool `json:"related"`
	Confidence    float64  `json:"confidence"`
	Relation      string   `json:"relation"`
	Justification string   `json:"justification"`
}

// jsonBool handles both bool and string ("true"/"false") JSON values.
type jsonBool bool

func (b *jsonBool) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == `"true"` || s == `true` {
		*b = true
		return nil
	}
	*b = false
	return nil
}

func llmVerifyPair(ctx context.Context, store *SQLiteStore, runtime ModelRuntime, cfg *Config, c candidate, lowerLevel, upperLevel model.Level) (bool, model.TraceRelation, float64, string) {
	req := LLMRequest{
		Model:             cfg.LLM.Model,
		System:            "You are a strategy analyst. Use only the provided evidence quotes. Respond with valid JSON only.",
		User:              buildTraceVerificationPrompt(c, lowerLevel, upperLevel),
		MaxTokens:         240,
		Temperature:       cfg.TemperatureForStage("verify"),
		JSONMode:          true,
		Stage:             "verify",
		Metadata:          verifyInvocationMetadata(c, lowerLevel, upperLevel),
		ReasoningFallback: cfg.ReasoningFallbackEnabled(),
	}

	resp, err := runtime.Chat(ctx, req)
	recordLLMInvocation(ctx, store, req, resp, err)
	if err != nil {
		slog.Warn("LLM verify error", "err", err)
		return false, model.RelationNone, 0, ""
	}

	raw := ParseLLMJSON(resp.Content)
	if raw == nil {
		slog.Warn("LLM verify: invalid JSON", "content", resp.Content)
		return false, model.RelationNone, 0, ""
	}

	var result llmVerifyResult
	if err := json.Unmarshal(raw, &result); err != nil {
		slog.Warn("LLM verify: parse error", "err", err)
		return false, model.RelationNone, 0, ""
	}

	if !bool(result.Related) || result.Relation == "none" {
		return false, model.RelationNone, 0, ""
	}
	return true, model.TraceRelation(result.Relation), result.Confidence, firstNonEmpty(strings.TrimSpace(result.Justification), "LLM verified strategic relation using source and target evidence quotes.")
}

func verifyInvocationMetadata(c candidate, lowerLevel, upperLevel model.Level) map[string]string {
	return map[string]string{
		"lower_level_id":     lowerLevel.ID,
		"lower_level_name":   lowerLevel.Name,
		"upper_level_id":     upperLevel.ID,
		"upper_level_name":   upperLevel.Name,
		"source_entity_id":   c.source.ID,
		"source_title":       c.source.Title,
		"source_document_id": c.source.DocumentID,
		"source_section_id":  c.source.SectionID,
		"target_entity_id":   c.target.ID,
		"target_title":       c.target.Title,
		"target_document_id": c.target.DocumentID,
		"target_section_id":  c.target.SectionID,
		"similarity":         fmt.Sprintf("%.6f", c.sim),
	}
}

func buildTraceVerificationPrompt(c candidate, lowerLevel, upperLevel model.Level) string {
	return fmt.Sprintf(`Assess whether the lower-level entity is meaningfully related to the upper-level entity based only on the evidence quotes below.

Lower-level entity:
- title: %q
- type: %s
- level: %s
- evidence_quote: %q

Upper-level entity:
- title: %q
- type: %s
- level: %s
- evidence_quote: %q

Return JSON:
{"related": bool, "confidence": 0.0-1.0, "relation": "contributes_to|enables|measures|decomposes_into|depends_on|conflicts_with|none", "justification": "one short sentence grounded in the quotes"}

If the quotes only sound similar but do not prove a strategic relation, return related=false. Do not rely on title similarity alone.`,
		c.source.Title, c.source.Type, lowerLevel.Name, c.source.SourceQuote,
		c.target.Title, c.target.Type, upperLevel.Name, c.target.SourceQuote)
}

func hasTraceEvidence(entity model.Entity) bool {
	if entity.DocumentID == "" || entity.SectionID == "" {
		return false
	}
	if strings.TrimSpace(entity.SourceQuote) == "" {
		return false
	}
	if entity.QuoteStartOffset == nil || entity.QuoteEndOffset == nil {
		return false
	}
	return entity.TrustGrade == "" || entity.TrustGrade == model.TrustGradeVerified
}

func buildVerifiedTrace(c candidate, relation model.TraceRelation, confidence float64, justification string) model.Trace {
	return model.Trace{
		ID:                     traceID(c.source.ID, c.target.ID),
		SourceEntityID:         c.source.ID,
		TargetEntityID:         c.target.ID,
		Relation:               relation,
		Confidence:             confidence,
		SimilarityScore:        c.sim,
		Justification:          justification,
		Direction:              model.DirectionUp,
		VerificationMode:       model.TraceVerificationModeLLMEvidence,
		TrustGrade:             model.TrustGradeVerified,
		SourceSectionID:        c.source.SectionID,
		TargetSectionID:        c.target.SectionID,
		SourceQuoteStartOffset: cloneIntPtr(c.source.QuoteStartOffset),
		SourceQuoteEndOffset:   cloneIntPtr(c.source.QuoteEndOffset),
		TargetQuoteStartOffset: cloneIntPtr(c.target.QuoteStartOffset),
		TargetQuoteEndOffset:   cloneIntPtr(c.target.QuoteEndOffset),
	}
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	return intPtr(*value)
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

func candidatesToModel(candidates []candidate, verifiedByCandidateID map[string]string, diagnosticByCandidateID map[string]string) []model.Candidate {
	result := make([]model.Candidate, len(candidates))
	for i, c := range candidates {
		candidateID := candidateID(c.source.ID, c.target.ID)
		traceID := verifiedByCandidateID[candidateID]
		diagnosticCode := diagnosticByCandidateID[candidateID]
		if diagnosticCode == "" {
			diagnosticCode = string(model.TraceCandidateDiagnosticEmbeddingSimilarityCandidate)
		}
		result[i] = model.Candidate{
			ID:             candidateID,
			SourceEntityID: c.source.ID,
			TargetEntityID: c.target.ID,
			Similarity:     c.sim,
			Verified:       traceID != "",
			TraceID:        traceID,
			DiagnosticCode: diagnosticCode,
		}
	}
	return result
}

func candidateID(src, tgt string) string {
	return fmt.Sprintf("cand_%s", sha256Hash([]byte(src + "|" + tgt))[:10])
}

// SaveCandidates saves trace candidates to the store.
func (s *SQLiteStore) SaveCandidates(ctx context.Context, candidates []model.Candidate) error {
	for _, c := range candidates {
		_, err := s.db.ExecContext(ctx,
			`INSERT OR REPLACE INTO trace_candidates (id, source_entity_id, target_entity_id, similarity, verified, trace_id, diagnostic_code)
			VALUES (?,?,?,?,?,?,?)`,
			c.ID, c.SourceEntityID, c.TargetEntityID, c.Similarity, c.Verified, nullableString(c.TraceID), nullableString(c.DiagnosticCode))
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
