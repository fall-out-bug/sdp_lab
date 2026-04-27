// Package routing provides embedding-based capability hint for cold-start routing.
// RoutingColdStartMicro uses kNN over a corpus of labeled examples to suggest
// a capability (e.g. "go-backend", "frontend", "docs", "infra") for a task.
package routing

import (
	"context"
	"fmt"

	"sdp_dev/internal/inference/decompose"
	"sdp_dev/internal/inference/microfirst/embed"
	"sdp_dev/internal/inference/microfirst/knn"
)

const defaultThreshold = 0.80

// RoutingInput is the description of a task needing routing.
type RoutingInput struct {
	Title       string
	Description string
}

// RoutingMicroResult is returned by RoutingColdStartMicro.Run.
type RoutingMicroResult struct {
	CapabilityHint string           // e.g. "go-backend"
	confidence     float64
	status         decompose.Status
	Neighbors      []knn.Match[string]
}

// Confidence implements decompose.Confider.
func (r RoutingMicroResult) Confidence() float64 { return r.confidence }

// ConfStatus implements decompose.Confider.
func (r RoutingMicroResult) ConfStatus() decompose.Status { return r.status }

// RoutingColdStartMicro suggests a capability hint for cold-start routing.
// It embeds the input title+description and queries a kNN index built from
// DefaultCorpus (or a custom corpus), then uses MajorityVote with the
// configured threshold.
type RoutingColdStartMicro struct {
	embedder  *embed.CachedEmbedder
	index     *knn.Index[string]
	threshold float64
}

// New constructs a RoutingColdStartMicro. It embeds each corpus example at
// construction time so Run is fast.
//
// threshold is the minimum top-1 cosine similarity required for StatusOK.
// If threshold <= 0, the default 0.80 is used.
func New(ctx context.Context, embedder *embed.CachedEmbedder, corpus []RoutingExample, threshold float64) (*RoutingColdStartMicro, error) {
	if embedder == nil {
		return nil, fmt.Errorf("routing.New: embedder must not be nil")
	}
	if threshold <= 0 {
		threshold = defaultThreshold
	}

	idx := knn.NewIndex[string]()
	for _, ex := range corpus {
		text := ex.Title + " " + ex.Description
		vec, err := embedder.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("routing.New: embed corpus %q: %w", ex.Title, err)
		}
		idx.Add(vec, ex.Capability, ex.Title)
	}

	return &RoutingColdStartMicro{
		embedder:  embedder,
		index:     idx,
		threshold: threshold,
	}, nil
}

// Name implements decompose.Stage.
func (c *RoutingColdStartMicro) Name() string { return "routing-coldstart-micro" }

// Run classifies the input and returns a RoutingMicroResult, a StageTrace, and an error.
// Status is StatusOK when top-1 score >= threshold AND top-3 neighbors agree on the same label.
// Otherwise Status is StatusUnsure.
func (c *RoutingColdStartMicro) Run(ctx context.Context, in RoutingInput) (RoutingMicroResult, decompose.StageTrace, error) {
	text := in.Title + " " + in.Description
	vec, err := c.embedder.Embed(ctx, text)
	if err != nil {
		return RoutingMicroResult{}, decompose.StageTrace{}, fmt.Errorf("routing.Run: embed input: %w", err)
	}

	matches := c.index.Query(vec, 3)
	result := knn.MajorityVote(matches, c.threshold)

	out := RoutingMicroResult{
		CapabilityHint: result.Label,
		confidence:     result.Score,
		status:         result.Status,
		Neighbors:      result.Neighbors,
	}

	return out, decompose.StageTrace{Attempts: 1}, nil
}

// SuggestCapability implements the MicroRouter interface for dispatch integration.
// Returns (hint, true) when classification is confident (StatusOK), otherwise ("", false).
func (c *RoutingColdStartMicro) SuggestCapability(ctx context.Context, title, description string) (string, bool) {
	res, _, err := c.Run(ctx, RoutingInput{Title: title, Description: description})
	if err != nil {
		return "", false
	}
	if res.ConfStatus() == decompose.StatusOK {
		return res.CapabilityHint, true
	}
	return "", false
}
