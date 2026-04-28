package strataudit

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/strataudit/model"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float32
		b    []float32
		want float64
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0},
		{"similar", []float32{1, 1, 1}, []float32{1, 1, 0.5}, 0.96},
		{"empty", []float32{}, []float32{}, 0.0},
		{"different_len", []float32{1, 0}, []float32{1}, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EmbeddingSimilarity(tt.a, tt.b)
			switch tt.name {
			case "identical":
				if got != 1.0 {
					t.Errorf("got %f, want 1.0", got)
				}
			case "orthogonal":
				if got != 0.0 {
					t.Errorf("got %f, want 0.0", got)
				}
			case "similar":
				if math.Abs(got-tt.want) > 0.01 {
					t.Errorf("got %f, want ~%f", got, tt.want)
				}
			case "empty", "different_len":
				if got != 0.0 {
					t.Errorf("got %f, want 0.0", got)
				}
			}
		})
	}
}

func TestCosineSimilarity_Realistic(t *testing.T) {
	a := []float32{0.1, 0.3, 0.5, 0.2, 0.8, 0.1, 0.4, 0.6}
	b := []float32{0.15, 0.28, 0.52, 0.18, 0.78, 0.12, 0.42, 0.58}

	sim := EmbeddingSimilarity(a, b)
	if sim < 0.95 {
		t.Errorf("similar embeddings: got %f, want >= 0.95", sim)
	}

	c := []float32{-0.5, 0.1, -0.3, 0.9, -0.2, 0.7, -0.1, 0.4}
	sim2 := EmbeddingSimilarity(a, c)
	if sim2 > sim {
		t.Errorf("less similar pair should have lower similarity: %f > %f", sim2, sim)
	}
}

func TestTraceID_Deterministic(t *testing.T) {
	id1 := traceID("e1", "e2")
	id2 := traceID("e1", "e2")
	if id1 != id2 {
		t.Error("traceID should be deterministic")
	}
	id3 := traceID("e2", "e1")
	if id1 == id3 {
		t.Error("different inputs should produce different IDs")
	}
}

func TestCandidateID_Deterministic(t *testing.T) {
	id1 := candidateID("e1", "e2")
	id2 := candidateID("e1", "e2")
	if id1 != id2 {
		t.Error("candidateID should be deterministic")
	}
}

func TestComputeStats(t *testing.T) {
	tests := []struct {
		name               string
		scores             []float64
		threshold          float64
		wantAbove          int
		wantMin            float64
		wantMax            float64
		wantRecommendation bool
	}{
		{
			name:               "basic distribution",
			scores:             []float64{0.1, 0.3, 0.5, 0.7, 0.9},
			threshold:          0.5,
			wantAbove:          3,
			wantMin:            0.1,
			wantMax:            0.9,
			wantRecommendation: false,
		},
		{
			name:               "all below threshold",
			scores:             []float64{0.1, 0.2, 0.3},
			threshold:          0.8,
			wantAbove:          0,
			wantMin:            0.1,
			wantMax:            0.3,
			wantRecommendation: true,
		},
		{
			name:               "all above threshold",
			scores:             []float64{0.8, 0.9, 1.0},
			threshold:          0.5,
			wantAbove:          3,
			wantMin:            0.8,
			wantMax:            1.0,
			wantRecommendation: false,
		},
		{
			name:               "single score",
			scores:             []float64{0.5},
			threshold:          0.5,
			wantAbove:          1,
			wantMin:            0.5,
			wantMax:            0.5,
			wantRecommendation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := computeStats(tt.scores, tt.threshold, 5, 5)
			if stats.AboveThreshold != tt.wantAbove {
				t.Errorf("AboveThreshold = %d, want %d", stats.AboveThreshold, tt.wantAbove)
			}
			if math.Abs(stats.Min-tt.wantMin) > 0.001 {
				t.Errorf("Min = %f, want %f", stats.Min, tt.wantMin)
			}
			if math.Abs(stats.Max-tt.wantMax) > 0.001 {
				t.Errorf("Max = %f, want %f", stats.Max, tt.wantMax)
			}
			gotRec := stats.Recommendation != ""
			if gotRec != tt.wantRecommendation {
				t.Errorf("Recommendation = %q, want empty=%v", stats.Recommendation, !tt.wantRecommendation)
			}
			histTotal := 0
			for _, b := range stats.Histogram {
				histTotal += b.Count
			}
			if histTotal != len(tt.scores) {
				t.Errorf("histogram total = %d, want %d", histTotal, len(tt.scores))
			}
			if stats.TotalPairs != len(tt.scores) {
				t.Errorf("TotalPairs = %d, want %d", stats.TotalPairs, len(tt.scores))
			}
		})
	}
}

func TestComputeStats_Median(t *testing.T) {
	stats := computeStats([]float64{0.1, 0.5, 0.9}, 0.5, 3, 3)
	if math.Abs(stats.Median-0.5) > 0.001 {
		t.Errorf("Median (odd) = %f, want 0.5", stats.Median)
	}

	stats = computeStats([]float64{0.1, 0.3, 0.7, 0.9}, 0.5, 4, 4)
	if math.Abs(stats.Median-0.5) > 0.001 {
		t.Errorf("Median (even) = %f, want 0.5", stats.Median)
	}
}

func TestComputeStats_Histogram(t *testing.T) {
	scores := []float64{0.05, 0.15, 0.25, 0.35, 0.45, 0.55, 0.65, 0.75, 0.85, 0.95}
	stats := computeStats(scores, 0.5, 10, 10)
	for i, b := range stats.Histogram {
		if b.Count != 1 {
			t.Errorf("histogram bucket %d count = %d, want 1", i, b.Count)
		}
	}
}

func TestComputeSimilarity_WithDistribution(t *testing.T) {
	cfg := &Config{
		Thresholds: ThresholdConfig{
			Similarity:       0.5,
			EmitDistribution: true,
		},
	}

	lower := []model.Entity{
		{ID: "e1", Embedding: []float32{1.0, 0.0, 0.0}},
		{ID: "e2", Embedding: []float32{0.0, 1.0, 0.0}},
	}
	upper := []model.Entity{
		{ID: "e3", Embedding: []float32{0.9, 0.1, 0.0}},
		{ID: "e4", Embedding: []float32{0.0, 0.0, 1.0}},
	}

	candidates, stats := computeSimilarity(context.Background(), cfg, lower, upper)

	if stats == nil {
		t.Fatal("expected stats when EmitDistribution=true")
	}
	if stats.TotalPairs != 4 {
		t.Errorf("TotalPairs = %d, want 4", stats.TotalPairs)
	}
	if stats.AboveThreshold < 1 {
		t.Errorf("AboveThreshold = %d, want >= 1", stats.AboveThreshold)
	}
	if len(candidates) < 1 {
		t.Errorf("candidates = %d, want >= 1", len(candidates))
	}
}

func TestComputeSimilarity_WithoutDistribution(t *testing.T) {
	cfg := &Config{
		Thresholds: ThresholdConfig{
			Similarity:       0.5,
			EmitDistribution: false,
		},
	}

	lower := []model.Entity{
		{ID: "e1", Embedding: []float32{1.0, 0.0}},
	}
	upper := []model.Entity{
		{ID: "e2", Embedding: []float32{0.9, 0.1}},
	}

	_, stats := computeSimilarity(context.Background(), cfg, lower, upper)

	if stats != nil {
		t.Error("expected nil stats when EmitDistribution=false")
	}
}

func TestWriteDistributionReport(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		Output:     OutputConfig{Dir: tmpDir},
		Thresholds: ThresholdConfig{Similarity: 0.5},
	}

	report := DistributionReport{
		RunID:       "run_test",
		GeneratedAt: "2026-04-11T12:00:00Z",
		Threshold:   0.5,
		LevelPairs: []LevelPairStats{
			{
				LevelPair:      "task -> initiative",
				TotalPairs:     100,
				AboveThreshold: 5,
				Min:            0.1,
				Max:            0.95,
				Mean:           0.45,
				Median:         0.42,
				P95:            0.88,
			},
		},
	}

	writeDistributionReport(cfg, report)

	path := filepath.Join(tmpDir, "similarity_distribution.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read distribution file: %v", err)
	}

	var parsed DistributionReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse distribution JSON: %v", err)
	}
	if parsed.RunID != "run_test" {
		t.Errorf("RunID = %q, want 'run_test'", parsed.RunID)
	}
	if len(parsed.LevelPairs) != 1 {
		t.Errorf("LevelPairs count = %d, want 1", len(parsed.LevelPairs))
	}
	if parsed.LevelPairs[0].TotalPairs != 100 {
		t.Errorf("TotalPairs = %d, want 100", parsed.LevelPairs[0].TotalPairs)
	}
}

func TestCreateVerifiedTraces_MarksVerificationUnavailableWithoutRuntime(t *testing.T) {
	cfg := &Config{
		Thresholds: ThresholdConfig{
			AutoVerifySimilarity: 0.85,
			TraceConfidence:      0.6,
			LLMVerifyBudget:      50,
		},
		LLM: LLMConfig{Temperature: 0.0, Temperatures: map[string]float64{"verify": 0.0}},
	}

	candidates := []candidate{
		{source: model.Entity{
			ID:               "e1",
			DocumentID:       "d1",
			SectionID:        "s1",
			SourceQuote:      "Source quote",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(12),
			TrustGrade:       model.TrustGradeVerified,
		}, target: model.Entity{
			ID:               "e2",
			DocumentID:       "d2",
			SectionID:        "s2",
			SourceQuote:      "Target quote",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(12),
			TrustGrade:       model.TrustGradeVerified,
		}, sim: 0.90},
	}

	traces, verifiedByCandidateID, diagnosticByCandidateID := createVerifiedTraces(context.Background(), cfg, nil, nil, candidates, model.Level{ID: "task", Name: "task"}, model.Level{ID: "strategy", Name: "strategy"})
	if len(traces) != 0 {
		t.Fatalf("traces = %d, want 0", len(traces))
	}
	modelCandidates := candidatesToModel(candidates, verifiedByCandidateID, diagnosticByCandidateID)
	if len(modelCandidates) != 1 {
		t.Fatalf("len(modelCandidates) = %d, want 1", len(modelCandidates))
	}
	if modelCandidates[0].Verified {
		t.Fatal("similarity-only candidate must not be marked verified")
	}
	if modelCandidates[0].TraceID != "" {
		t.Fatalf("TraceID = %q, want empty", modelCandidates[0].TraceID)
	}
	if modelCandidates[0].DiagnosticCode != string(model.TraceCandidateDiagnosticVerificationUnavailable) {
		t.Fatalf("DiagnosticCode = %q, want verification_unavailable", modelCandidates[0].DiagnosticCode)
	}
}

func TestCreateVerifiedTraces_BudgetExhaustion(t *testing.T) {
	cfg := &Config{
		Thresholds: ThresholdConfig{
			AutoVerifySimilarity: 0.95,
			TraceConfidence:      0.3,
			LLMVerifyBudget:      0, // budget exhausted from start
		},
		LLM: LLMConfig{Temperature: 0.0},
	}

	candidates := []candidate{
		{source: model.Entity{
			ID:               "e1",
			DocumentID:       "d1",
			SectionID:        "s1",
			SourceQuote:      "Нижняя цитата",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(12),
		}, target: model.Entity{
			ID:               "e2",
			DocumentID:       "d2",
			SectionID:        "s2",
			SourceQuote:      "Верхняя цитата",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(13),
		}, sim: 0.70},
	}

	// With budget=0, no LLM calls should happen even if runtime exists.
	llm := FunctionalRuntime{
		ChatFunc: func(ctx context.Context, req LLMRequest) (*LLMResponse, error) {
			t.Fatal("Chat must not be called when verification budget is exhausted")
			return nil, nil
		},
		EmbedFunc: func(ctx context.Context, texts []string, model string) ([][]float32, error) {
			t.Fatal("Embed must not be called in createVerifiedTraces")
			return nil, nil
		},
	}
	traces, _, diagnosticByCandidateID := createVerifiedTraces(context.Background(), cfg, nil, llm, candidates, model.Level{ID: "l1"}, model.Level{ID: "l2"})
	if len(traces) != 0 {
		t.Errorf("traces = %d, want 0 (budget exhausted)", len(traces))
	}
	if diagnosticByCandidateID[candidateID("e1", "e2")] != string(model.TraceCandidateDiagnosticVerificationBudgetExhausted) {
		t.Fatalf("diagnostic = %q, want verification_budget_exhausted", diagnosticByCandidateID[candidateID("e1", "e2")])
	}
}

func TestCreateVerifiedTraces_RequiresEvidenceAndLLMVerification(t *testing.T) {
	cfg := &Config{
		Thresholds: ThresholdConfig{
			AutoVerifySimilarity: 0.85,
			TraceConfidence:      0.6,
			LLMVerifyBudget:      5,
		},
		LLM: LLMConfig{
			Model:        "test-model",
			Temperature:  0.0,
			Temperatures: map[string]float64{"verify": 0.0},
		},
	}

	lowerLevel := model.Level{ID: "task", Name: "task"}
	upperLevel := model.Level{ID: "strategy", Name: "strategy"}
	goodCandidate := candidate{
		source: model.Entity{
			ID:               "e1",
			Title:            "Запустить интеграцию",
			Type:             model.EntityTask,
			DocumentID:       "d1",
			SectionID:        "s1",
			SourceQuote:      "Запустить интеграцию с платёжным шлюзом.",
			QuoteStartOffset: intPtr(10),
			QuoteEndOffset:   intPtr(48),
			TrustGrade:       model.TrustGradeVerified,
		},
		target: model.Entity{
			ID:               "e2",
			Title:            "Рост цифровых платежей",
			Type:             model.EntityObjective,
			DocumentID:       "d2",
			SectionID:        "s2",
			SourceQuote:      "Наша стратегия — рост цифровых платежей.",
			QuoteStartOffset: intPtr(2),
			QuoteEndOffset:   intPtr(42),
			TrustGrade:       model.TrustGradeVerified,
		},
		sim: 0.91,
	}
	missingEvidenceCandidate := candidate{
		source: model.Entity{
			ID:         "e3",
			Title:      "Без evidence",
			Type:       model.EntityTask,
			DocumentID: "d3",
		},
		target: model.Entity{
			ID:               "e4",
			Title:            "Цель",
			Type:             model.EntityGoal,
			DocumentID:       "d4",
			SectionID:        "s4",
			SourceQuote:      "Верхняя цель.",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(12),
			TrustGrade:       model.TrustGradeVerified,
		},
		sim: 0.95,
	}

	llm := newMockLLMClient(t, `{"related": true, "confidence": 0.88, "relation": "contributes_to", "justification": "Нижняя инициатива прямо поддерживает верхнюю стратегию."}`)

	traces, verifiedByCandidateID, diagnosticByCandidateID := createVerifiedTraces(context.Background(), cfg, nil, llm, []candidate{missingEvidenceCandidate, goodCandidate}, lowerLevel, upperLevel)
	if len(traces) != 1 {
		t.Fatalf("traces = %d, want 1", len(traces))
	}
	trace := traces[0]
	if trace.VerificationMode != model.TraceVerificationModeLLMEvidence {
		t.Fatalf("VerificationMode = %q, want %q", trace.VerificationMode, model.TraceVerificationModeLLMEvidence)
	}
	if trace.SimilarityScore != 0.91 {
		t.Fatalf("SimilarityScore = %f, want 0.91", trace.SimilarityScore)
	}
	if trace.SourceSectionID != "s1" || trace.TargetSectionID != "s2" {
		t.Fatalf("unexpected section refs: %+v", trace)
	}
	if trace.SourceQuoteStartOffset == nil || *trace.SourceQuoteStartOffset != 10 {
		t.Fatalf("SourceQuoteStartOffset = %+v, want 10", trace.SourceQuoteStartOffset)
	}
	if verifiedByCandidateID[candidateID(goodCandidate.source.ID, goodCandidate.target.ID)] == "" {
		t.Fatal("expected verified candidate to map to trace id")
	}
	if diagnosticByCandidateID[candidateID(goodCandidate.source.ID, goodCandidate.target.ID)] != string(model.TraceCandidateDiagnosticLLMVerified) {
		t.Fatalf("good candidate diagnostic = %q, want llm_verified", diagnosticByCandidateID[candidateID(goodCandidate.source.ID, goodCandidate.target.ID)])
	}
	if verifiedByCandidateID[candidateID(missingEvidenceCandidate.source.ID, missingEvidenceCandidate.target.ID)] != "" {
		t.Fatal("candidate without evidence must not map to verified trace")
	}
	if diagnosticByCandidateID[candidateID(missingEvidenceCandidate.source.ID, missingEvidenceCandidate.target.ID)] != string(model.TraceCandidateDiagnosticQuoteEvidenceMissing) {
		t.Fatalf("missing evidence diagnostic = %q, want quote_evidence_missing", diagnosticByCandidateID[candidateID(missingEvidenceCandidate.source.ID, missingEvidenceCandidate.target.ID)])
	}
}

func TestCreateVerifiedTraces_RecordsLLMRejectionDiagnostic(t *testing.T) {
	cfg := &Config{
		Thresholds: ThresholdConfig{
			AutoVerifySimilarity: 0.85,
			TraceConfidence:      0.6,
			LLMVerifyBudget:      5,
		},
		LLM: LLMConfig{
			Model:        "test-model",
			Temperature:  0.0,
			Temperatures: map[string]float64{"verify": 0.0},
		},
	}

	lowerLevel := model.Level{ID: "task", Name: "task"}
	upperLevel := model.Level{ID: "strategy", Name: "strategy"}
	rejectedCandidate := candidate{
		source: model.Entity{
			ID:               "e1",
			Title:            "Локальная задача",
			Type:             model.EntityTask,
			DocumentID:       "d1",
			SectionID:        "s1",
			SourceQuote:      "Локальная задача без стратегической опоры.",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(38),
			TrustGrade:       model.TrustGradeVerified,
		},
		target: model.Entity{
			ID:               "e2",
			Title:            "Стратегическая цель",
			Type:             model.EntityGoal,
			DocumentID:       "d2",
			SectionID:        "s2",
			SourceQuote:      "Стратегическая цель роста.",
			QuoteStartOffset: intPtr(0),
			QuoteEndOffset:   intPtr(25),
			TrustGrade:       model.TrustGradeVerified,
		},
		sim: 0.84,
	}

	llm := newMockLLMClient(t, `{"related": false, "confidence": 0.31, "relation": "none", "justification": "Quotes do not prove a strategic relation."}`)

	traces, verifiedByCandidateID, diagnosticByCandidateID := createVerifiedTraces(context.Background(), cfg, nil, llm, []candidate{rejectedCandidate}, lowerLevel, upperLevel)
	if len(traces) != 0 {
		t.Fatalf("traces = %d, want 0", len(traces))
	}
	if verifiedByCandidateID[candidateID("e1", "e2")] != "" {
		t.Fatal("rejected candidate must not map to a trace id")
	}
	if diagnosticByCandidateID[candidateID("e1", "e2")] != string(model.TraceCandidateDiagnosticLLMVerificationRejected) {
		t.Fatalf("diagnostic = %q, want llm_verification_rejected", diagnosticByCandidateID[candidateID("e1", "e2")])
	}
}

func TestLLMVerifyResult_ParseJSON(t *testing.T) {
	input := `{"related": true, "confidence": 0.9, "relation": "contributes_to", "justification": "Нижняя сущность поддерживает верхнюю."}`
	var result llmVerifyResult
	if err := json.Unmarshal([]byte(input), &result); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !result.Related {
		t.Error("expected related=true")
	}
	if result.Relation != "contributes_to" {
		t.Errorf("relation = %q, want contributes_to", result.Relation)
	}
	if result.Confidence != 0.9 {
		t.Errorf("confidence = %f, want 0.9", result.Confidence)
	}
	if result.Justification == "" {
		t.Fatal("expected justification to be parsed")
	}
}
