package knn

import (
	"math"
	"sort"
)

// Match is a nearest-neighbour result.
type Match[Label comparable] struct {
	Label    Label
	Score    float64 // cosine similarity [0, 1]
	Metadata string  // optional: source issue ID etc.
}

// Index stores labelled embedding vectors for nearest-neighbour lookup.
type Index[Label comparable] struct {
	entries []entry[Label]
}

type entry[Label comparable] struct {
	vec   []float64
	label Label
	meta  string
}

// NewIndex returns an empty Index.
func NewIndex[Label comparable]() *Index[Label] {
	return &Index[Label]{}
}

// Add inserts a labelled vector. vec should be L2-normalised for cosine accuracy.
func (idx *Index[Label]) Add(vec []float64, label Label, meta string) {
	idx.entries = append(idx.entries, entry[Label]{vec: vec, label: label, meta: meta})
}

// Len returns number of indexed entries.
func (idx *Index[Label]) Len() int {
	return len(idx.entries)
}

// Query returns the top-k closest entries by cosine similarity.
func (idx *Index[Label]) Query(vec []float64, k int) []Match[Label] {
	matches := make([]Match[Label], 0, len(idx.entries))
	for _, e := range idx.entries {
		s := cosine(vec, e.vec)
		matches = append(matches, Match[Label]{Label: e.label, Score: s, Metadata: e.meta})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if k > len(matches) {
		k = len(matches)
	}
	return matches[:k]
}

// cosine returns cosine similarity in [0, 1] (clipped).
func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	var dot, normA, normB float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}

	sim := dot / denom
	// Clip to [0, 1]
	if sim < 0 {
		return 0
	}
	if sim > 1 {
		return 1
	}
	return sim
}
