package bdseverity

import (
	"context"
	"fmt"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"
	"github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/embed"
	"github.com/fall-out-bug/sdp_lab/internal/inference/microfirst/knn"
)

// BdInput is the input to the severity classifier.
type BdInput struct {
	Title       string
	Description string
}

// BdSeverityMicro classifies bd issue severity (P0..P3) using embedding k-NN.
// Implements decompose.Stage[BdInput, BdSeverityResult].
type BdSeverityMicro struct {
	embedder  *embed.CachedEmbedder
	index     *knn.Index[string] // label = "P0"|"P1"|"P2"|"P3"
	threshold float64            // minimum top-1 cosine score for StatusOK
}

const defaultThreshold = 0.85
const neighborCount = 5

// New builds a BdSeverityMicro from a pre-loaded corpus.
// It embeds each corpus item and adds it to the k-NN index.
func New(ctx context.Context, embedder *embed.CachedEmbedder, corpus []BdIssue, threshold float64) (*BdSeverityMicro, error) {
	if threshold <= 0 {
		threshold = defaultThreshold
	}
	idx := knn.NewIndex[string]()
	for _, issue := range corpus {
		text := issueText(issue.Title, issue.Description)
		vec, err := embedder.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("bdseverity: embed corpus item %s: %w", issue.ID, err)
		}
		idx.Add(vec, issue.Priority, issue.ID)
	}
	return &BdSeverityMicro{
		embedder:  embedder,
		index:     idx,
		threshold: threshold,
	}, nil
}

// Name satisfies decompose.Stage.
func (c *BdSeverityMicro) Name() string { return "bd-severity-micro" }

// Run classifies the input issue's severity.
func (c *BdSeverityMicro) Run(ctx context.Context, in BdInput) (BdSeverityResult, decompose.StageTrace, error) {
	start := time.Now()

	text := issueText(in.Title, in.Description)
	vec, err := c.embedder.Embed(ctx, text)
	if err != nil {
		return BdSeverityResult{}, decompose.StageTrace{}, fmt.Errorf("bdseverity: embed input: %w", err)
	}

	matches := c.index.Query(vec, neighborCount)
	result := knn.MajorityVote(matches, c.threshold)

	latency := time.Since(start).Milliseconds()
	trace := decompose.StageTrace{
		LatencyMs: latency,
		Attempts:  1,
	}

	return BdSeverityResult{
		Priority:   result.Label,
		confidence: result.Score,
		status:     result.Status,
		Neighbors:  result.Neighbors,
	}, trace, nil
}

// issueText concatenates title and description into a single embedding text.
func issueText(title, description string) string {
	if description == "" {
		return title
	}
	return title + ". " + description
}
