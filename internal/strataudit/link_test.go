package strataudit

import (
	"math"
	"testing"
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
