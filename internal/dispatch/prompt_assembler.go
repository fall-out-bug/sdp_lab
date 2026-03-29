package dispatch

import "strings"

// charsPerToken is the conservative heuristic for estimating token count from
// character length. 4 characters per token is a safe lower bound for English
// text and code.
const charsPerToken = 4

// ModelContextBudget defines context window allocation per model.
type ModelContextBudget struct {
	// TotalTokens is the full context window size for the model.
	TotalTokens int
	// ContextPct is the fraction of the window reserved for injected context
	// layers (default 0.3 meaning 30%).
	ContextPct float64
}

// modelBudgets maps known model identifiers to their context budgets.
// Unknown models fall back to the "default" entry.
var modelBudgets = map[string]ModelContextBudget{
	"claude-opus-4":   {TotalTokens: 200000, ContextPct: 0.3},
	"claude-sonnet-4": {TotalTokens: 200000, ContextPct: 0.3},
	"gpt-4o":          {TotalTokens: 128000, ContextPct: 0.3},
	"gemini-2.5-pro":  {TotalTokens: 1000000, ContextPct: 0.2},
	"default":         {TotalTokens: 128000, ContextPct: 0.3},
}

// PromptLayer represents an injectable context layer with a name and content.
type PromptLayer struct {
	// Name is a human-readable label for the layer (e.g. "scope", "conventions").
	Name string
	// Content is the text body of the layer.
	Content string
}

// PromptAssembler builds a model-aware prompt with injected context layers,
// respecting the target model's context window budget.
type PromptAssembler struct {
	// Model is the target model identifier used to look up the context budget.
	Model string
}

// NewPromptAssembler creates an assembler for the given model.
func NewPromptAssembler(model string) *PromptAssembler {
	return &PromptAssembler{Model: model}
}

// Budget returns the context budget in tokens available for injected context
// layers. Unknown models use the "default" budget.
func (a *PromptAssembler) Budget() int {
	b, ok := modelBudgets[a.Model]
	if !ok {
		b = modelBudgets["default"]
	}
	return int(float64(b.TotalTokens) * b.ContextPct)
}

// Assemble builds the final prompt string. The taskPrompt is always included
// (mandatory). Each PromptLayer is appended in order, but only if it fits
// within the remaining token budget. Layers that would exceed the budget are
// silently dropped.
//
// Token estimation uses a simple heuristic: 1 token equals approximately
// charsPerToken (4) characters.
func (a *PromptAssembler) Assemble(taskPrompt string, layers ...PromptLayer) string {
	budget := a.Budget()

	var b strings.Builder
	b.WriteString(taskPrompt)

	// Task prompt is always included; remaining budget is reduced by its cost.
	taskTokens := charToToken(len(taskPrompt))
	remaining := budget - taskTokens

	for _, layer := range layers {
		layerTokens := charToToken(len(layer.Content))
		if layerTokens <= remaining {
			b.WriteString(layer.Content)
			remaining -= layerTokens
		}
		// Layer does not fit; skip it silently.
	}

	return b.String()
}

// charToToken converts a character count to an estimated token count using
// the conservative charsPerToken ratio.
func charToToken(chars int) int {
	if chars <= 0 {
		return 0
	}
	return (chars + charsPerToken - 1) / charsPerToken
}
