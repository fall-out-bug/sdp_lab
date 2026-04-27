package knn

import (
	"math"
	"testing"

	"sdp_dev/internal/inference/decompose"
)

// --- cosine tests ---

func TestCosine_OrthogonalVectors(t *testing.T) {
	a := []float64{1, 0}
	b := []float64{0, 1}
	got := cosine(a, b)
	if got != 0 {
		t.Errorf("orthogonal vectors: want 0, got %f", got)
	}
}

func TestCosine_IdenticalVectors(t *testing.T) {
	a := []float64{1, 2, 3}
	got := cosine(a, a)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("identical vectors: want 1.0, got %f", got)
	}
}

func TestCosine_KnownAngle(t *testing.T) {
	// 45-degree angle → cos(45°) = √2/2 ≈ 0.7071
	a := []float64{1, 0}
	b := []float64{1, 1}
	want := 1.0 / math.Sqrt(2)
	got := cosine(a, b)
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("45-degree angle: want %f, got %f", want, got)
	}
}

func TestCosine_NegativeDotClipsToZero(t *testing.T) {
	a := []float64{1, 0}
	b := []float64{-1, 0}
	got := cosine(a, b)
	if got != 0 {
		t.Errorf("anti-parallel vectors: want 0 (clipped), got %f", got)
	}
}

func TestCosine_ZeroVector(t *testing.T) {
	a := []float64{0, 0}
	b := []float64{1, 0}
	got := cosine(a, b)
	if got != 0 {
		t.Errorf("zero vector: want 0, got %f", got)
	}
}

// --- Index Query tests ---

func TestIndex_AddLen(t *testing.T) {
	idx := NewIndex[string]()
	if idx.Len() != 0 {
		t.Errorf("empty index: want 0, got %d", idx.Len())
	}
	idx.Add([]float64{1, 0}, "a", "")
	if idx.Len() != 1 {
		t.Errorf("after add: want 1, got %d", idx.Len())
	}
}

func TestIndex_Query_TopKOrder(t *testing.T) {
	idx := NewIndex[string]()

	// Insert 10 unit vectors at known angles; query with [1,0].
	// The one closest to [1,0] should have highest cosine score.
	vecs := []struct {
		v     []float64
		label string
	}{
		{[]float64{1, 0}, "exact"},
		{[]float64{0.9, 0.1}, "close1"},
		{[]float64{0.8, 0.2}, "close2"},
		{[]float64{0.7, 0.3}, "close3"},
		{[]float64{0.6, 0.4}, "close4"},
		{[]float64{0.5, 0.5}, "mid"},
		{[]float64{0.4, 0.6}, "far1"},
		{[]float64{0.3, 0.7}, "far2"},
		{[]float64{0.2, 0.8}, "far3"},
		{[]float64{0.1, 0.9}, "far4"},
	}

	for _, v := range vecs {
		idx.Add(v.v, v.label, "")
	}

	query := []float64{1, 0}
	results := idx.Query(query, 3)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// First result should be the exact match.
	if results[0].Label != "exact" {
		t.Errorf("top-1: want 'exact', got '%s'", results[0].Label)
	}

	// Scores should be descending.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("scores not descending at index %d: %f > %f", i, results[i].Score, results[i-1].Score)
		}
	}
}

func TestIndex_Query_KGreaterThanLen(t *testing.T) {
	idx := NewIndex[string]()
	idx.Add([]float64{1, 0}, "a", "")
	idx.Add([]float64{0, 1}, "b", "")

	results := idx.Query([]float64{1, 0}, 10)
	if len(results) != 2 {
		t.Errorf("expected 2 results (all), got %d", len(results))
	}
}

// --- MajorityVote tests ---

func TestMajorityVote_Unanimous_OK(t *testing.T) {
	matches := []Match[string]{
		{Label: "bug", Score: 0.92},
		{Label: "bug", Score: 0.89},
		{Label: "bug", Score: 0.85},
	}
	result := MajorityVote(matches, 0.8)
	if result.Status != decompose.StatusOK {
		t.Errorf("unanimous high-score: want StatusOK, got %s", result.Status)
	}
	if result.Label != "bug" {
		t.Errorf("expected label 'bug', got '%s'", result.Label)
	}
}

func TestMajorityVote_Disagreement_Unsure(t *testing.T) {
	matches := []Match[string]{
		{Label: "bug", Score: 0.90},
		{Label: "feature", Score: 0.88},
		{Label: "bug", Score: 0.85},
	}
	result := MajorityVote(matches, 0.8)
	if result.Status != decompose.StatusUnsure {
		t.Errorf("disagreement: want StatusUnsure, got %s", result.Status)
	}
}

func TestMajorityVote_LowScore_Unsure(t *testing.T) {
	matches := []Match[string]{
		{Label: "bug", Score: 0.50},
		{Label: "bug", Score: 0.48},
		{Label: "bug", Score: 0.45},
	}
	result := MajorityVote(matches, 0.8)
	if result.Status != decompose.StatusUnsure {
		t.Errorf("low score: want StatusUnsure, got %s", result.Status)
	}
}

func TestMajorityVote_EmptyMatches_Unsure(t *testing.T) {
	result := MajorityVote([]Match[string]{}, 0.8)
	if result.Status != decompose.StatusUnsure {
		t.Errorf("empty matches: want StatusUnsure, got %s", result.Status)
	}
}

func TestMajorityVote_SingleMatch_OK(t *testing.T) {
	matches := []Match[string]{
		{Label: "bug", Score: 0.95},
	}
	result := MajorityVote(matches, 0.8)
	if result.Status != decompose.StatusOK {
		t.Errorf("single high-score match: want StatusOK, got %s", result.Status)
	}
}

func TestMajorityVote_TwoMatches_Disagreement(t *testing.T) {
	matches := []Match[string]{
		{Label: "bug", Score: 0.95},
		{Label: "feature", Score: 0.90},
	}
	result := MajorityVote(matches, 0.8)
	if result.Status != decompose.StatusUnsure {
		t.Errorf("two disagreeing: want StatusUnsure, got %s", result.Status)
	}
}

func TestMajorityVote_NeighborsAttached(t *testing.T) {
	matches := []Match[string]{
		{Label: "bug", Score: 0.92},
		{Label: "bug", Score: 0.89},
	}
	result := MajorityVote(matches, 0.8)
	if len(result.Neighbors) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(result.Neighbors))
	}
}
