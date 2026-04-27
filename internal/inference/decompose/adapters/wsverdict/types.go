// Package wsverdict provides a 3-stage decomposed pipeline for ws-verdict inference
// and a monolithic single-shot baseline for A/B benchmarking.
//
// Stage order: Extract (Haiku/JSON) → Classify (Sonnet/Enum) → Aggregate (Haiku/TOON).
//
// Use NewDecomposedRunner(client) for the 3-stage pipeline.
// Use NewMonolithicRunner(client) for the single-shot baseline.
package wsverdict

import "context"

// Diff is the input to both pipeline variants.
type Diff struct {
	// WSID is the workstream ID being evaluated.
	WSID string
	// DiffText is the raw git diff output.
	DiffText string
	// Context is additional workstream context (goals, AC, scope files).
	Context string
}

// ExtractOut is the structured output of the extract stage.
type ExtractOut struct {
	ChangedFiles []string `json:"changed_files"`
	Modules      []string `json:"modules"`
	ChangeType   string   `json:"change_type"` // "feat"|"fix"|"refactor"|"docs"|"test"|"chore"
	Summary      string   `json:"summary"`
}

// FinalVerdict is the final output of both pipeline variants.
// Semantically equivalent between decomposed and monolithic for fair A/B.
type FinalVerdict struct {
	Verdict       string   `json:"verdict"`        // "passed"|"partial"|"failed"
	Score         float64  `json:"score"`          // [0,1]
	Summary       string   `json:"summary"`
	BlockingGates []string `json:"blocking_gates,omitempty"`
}

// LLMCaller is the interface the adapter uses to call LLM providers.
// Satisfied by *llmclient.Client wrapped via LLMCallerAdapter.
type LLMCaller interface {
	Call(ctx context.Context, prompt string, opts CallOptions) (text string, tokensIn, tokensOut int, costUSD float64, err error)
}

// CallOptions controls the per-call model and sampling parameters.
type CallOptions struct {
	Model       string
	Temperature float64
	MaxTokens   int
}
