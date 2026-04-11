package strataudit

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/internal/strataudit/model"
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
