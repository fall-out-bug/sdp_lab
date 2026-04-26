package bdtype

import (
	"context"
	"fmt"
	"math"
	"strings"

	"sdp_dev/internal/inference/decompose"
	"sdp_dev/internal/inference/microfirst/knn"
)

const (
	// defaultThreshold is the minimum top-1 cosine score to trust a classification.
	// 3-class problem is easier than binary, so 0.80 is appropriate.
	defaultThreshold = 0.80

	// topK is the number of nearest neighbours used for majority vote.
	topK = 3
)

// Embedder is the interface satisfied by embed.OllamaEmbedder and embed.CachedEmbedder.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// BdInput is the input to the classifier.
// If Text is set, it is used directly as the embedding text (Title+Description are ignored).
type BdInput struct {
	Title       string
	Description string
	Text        string // optional pre-composed text; overrides Title+Description if non-empty
}

// BdTypeMicro is an embedding-based kNN classifier for issue type.
// It implements decompose.Stage[BdInput, BdTypeResult].
type BdTypeMicro struct {
	embedder  Embedder
	index     *knn.Index[string]
	threshold float64
}

// NewBdTypeMicro constructs a classifier and builds the kNN index from corpus entries.
// Each entry is embedded using embedder; embeddings are L2-normalised internally.
func NewBdTypeMicro(ctx context.Context, embedder Embedder, corpus []CorpusEntry) (*BdTypeMicro, error) {
	return NewBdTypeMicroWithThreshold(ctx, embedder, corpus, defaultThreshold)
}

// NewBdTypeMicroWithThreshold constructs a classifier with an explicit confidence threshold.
func NewBdTypeMicroWithThreshold(ctx context.Context, embedder Embedder, corpus []CorpusEntry, threshold float64) (*BdTypeMicro, error) {
	idx := knn.NewIndex[string]()
	for _, entry := range corpus {
		vec, err := embedder.Embed(ctx, entry.Text)
		if err != nil {
			return nil, fmt.Errorf("bdtype: embed corpus entry %s: %w", entry.ID, err)
		}
		l2Normalize(vec)
		idx.Add(vec, entry.Label, entry.ID)
	}
	return &BdTypeMicro{
		embedder:  embedder,
		index:     idx,
		threshold: threshold,
	}, nil
}

// Name implements decompose.Stage.
func (b *BdTypeMicro) Name() string { return "bdtype-micro" }

// Run implements decompose.Stage[BdInput, BdTypeResult].
func (b *BdTypeMicro) Run(ctx context.Context, in BdInput) (BdTypeResult, decompose.StageTrace, error) {
	text := in.Text
	if text == "" {
		text = strings.TrimSpace(in.Title + " " + in.Description)
	}
	vec, err := b.embedder.Embed(ctx, text)
	if err != nil {
		return BdTypeResult{}, decompose.StageTrace{Attempts: 1}, fmt.Errorf("bdtype: embed input: %w", err)
	}
	l2Normalize(vec)

	matches := b.index.Query(vec, topK)
	cr := knn.MajorityVote(matches, b.threshold)

	result := BdTypeResult{
		Type:       cr.Label,
		confidence: cr.Score,
		status:     cr.Status,
		Neighbors:  cr.Neighbors,
	}
	return result, decompose.StageTrace{Attempts: 1}, nil
}

// l2Normalize normalises vec in-place to unit length.
func l2Normalize(vec []float64) {
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return
	}
	for i := range vec {
		vec[i] /= norm
	}
}
