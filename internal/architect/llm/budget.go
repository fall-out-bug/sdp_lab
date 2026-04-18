package llm

import (
	"fmt"
	"sync"
)

// Tier represents an enrichment tier with different token budgets.
type Tier int

const (
	// Tier1 is for lightweight enrichment (~2K tokens per request)
	Tier1 Tier = iota

	// Tier2 is for container-level enrichment (~5-15K tokens per container)
	Tier2

	// Tier3 is for component-level enrichment (~10-30K tokens per component)
	Tier3
)

// String returns the tier name.
func (t Tier) String() string {
	switch t {
	case Tier1:
		return "tier1"
	case Tier2:
		return "tier2"
	case Tier3:
		return "tier3"
	default:
		return "unknown"
	}
}

// BudgetConfig defines token budgets per tier.
type BudgetConfig struct {
	// MaxInputTokens is the maximum input tokens per request.
	MaxInputTokens map[Tier]int

	// MaxOutputTokens is the maximum output tokens per request.
	MaxOutputTokens map[Tier]int

	// MaxTotalTokens is the maximum total tokens (input + output) per request.
	MaxTotalTokens map[Tier]int
}

// DefaultBudgetConfig returns sensible defaults for each tier.
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		MaxInputTokens: map[Tier]int{
			Tier1: 1500,  // ~2K total
			Tier2: 10000, // ~15K total
			Tier3: 20000, // ~30K total
		},
		MaxOutputTokens: map[Tier]int{
			Tier1: 500,   // ~2K total
			Tier2: 5000,  // ~15K total
			Tier3: 10000, // ~30K total
		},
		MaxTotalTokens: map[Tier]int{
			Tier1: 2000,  // Lightweight
			Tier2: 15000, // Container-level
			Tier3: 30000, // Component-level
		},
	}
}

// BudgetManager tracks token usage and enforces budgets per tier.
type BudgetManager struct {
	cfg      BudgetConfig
	mu       sync.Mutex
	tracking map[string]*TokenTracker // key: nodeID or requestID
}

// TokenTracker tracks token consumption for a single entity.
type TokenTracker struct {
	Tier             Tier
	InputTokens      int
	OutputTokens     int
	TotalTokens      int
	Requests         int
	LastRequestInput int
	LastRequestOutput int
}

// NewBudgetManager creates a new budget manager with the given config.
func NewBudgetManager(cfg BudgetConfig) *BudgetManager {
	return &BudgetManager{
		cfg:      cfg,
		tracking: make(map[string]*TokenTracker),
	}
}

// DefaultBudgetManager creates a budget manager with default config.
func DefaultBudgetManager() *BudgetManager {
	return NewBudgetManager(DefaultBudgetConfig())
}

// CheckBudget verifies if a request would exceed the budget for a tier.
// Returns an error if the budget would be exceeded.
func (bm *BudgetManager) CheckBudget(tier Tier, inputTokens, outputTokens int) error {
	maxInput := bm.cfg.MaxInputTokens[tier]
	maxOutput := bm.cfg.MaxOutputTokens[tier]
	maxTotal := bm.cfg.MaxTotalTokens[tier]

	if inputTokens > maxInput {
		return fmt.Errorf("input tokens %d exceed tier %s budget of %d", inputTokens, tier, maxInput)
	}

	if outputTokens > maxOutput {
		return fmt.Errorf("output tokens %d exceed tier %s budget of %d", outputTokens, tier, maxOutput)
	}

	total := inputTokens + outputTokens
	if total > maxTotal {
		return fmt.Errorf("total tokens %d exceed tier %s budget of %d", total, tier, maxTotal)
	}

	return nil
}

// RecordUsage records token usage for a node/request.
func (bm *BudgetManager) RecordUsage(key string, tier Tier, inputTokens, outputTokens int) error {
	if err := bm.CheckBudget(tier, inputTokens, outputTokens); err != nil {
		return err
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	tracker, exists := bm.tracking[key]
	if !exists {
		tracker = &TokenTracker{Tier: tier}
		bm.tracking[key] = tracker
	}

	tracker.InputTokens += inputTokens
	tracker.OutputTokens += outputTokens
	tracker.TotalTokens += inputTokens + outputTokens
	tracker.Requests++
	tracker.LastRequestInput = inputTokens
	tracker.LastRequestOutput = outputTokens

	return nil
}

// GetTracker returns the token tracker for a key, or nil if not found.
func (bm *BudgetManager) GetTracker(key string) *TokenTracker {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	return bm.tracking[key]
}

// TotalUsage returns the total token usage across all tracked entities.
func (bm *BudgetManager) TotalUsage() (input, output, total int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, tracker := range bm.tracking {
		input += tracker.InputTokens
		output += tracker.OutputTokens
		total += tracker.TotalTokens
	}

	return input, output, total
}

// TierUsage returns the total token usage for a specific tier.
func (bm *BudgetManager) TierUsage(tier Tier) (input, output, total int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, tracker := range bm.tracking {
		if tracker.Tier == tier {
			input += tracker.InputTokens
			output += tracker.OutputTokens
			total += tracker.TotalTokens
		}
	}

	return input, output, total
}

// Reset clears all tracking data.
func (bm *BudgetManager) Reset() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.tracking = make(map[string]*TokenTracker)
}

// ResetKey clears tracking for a specific key.
func (bm *BudgetManager) ResetKey(key string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.tracking, key)
}

// GetBudgetForTier returns the budget limits for a tier.
func (bm *BudgetManager) GetBudgetForTier(tier Tier) (maxInput, maxOutput, maxTotal int) {
	return bm.cfg.MaxInputTokens[tier],
		bm.cfg.MaxOutputTokens[tier],
		bm.cfg.MaxTotalTokens[tier]
}

// EstimateTokens estimates token count for text input.
// This is a rough approximation: ~4 chars per token for English text.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	// Rough estimate: 4 characters per token
	// This varies by language and content, but is a reasonable baseline
	return (len(text) + 3) / 4
}

// EstimateInputTokens estimates tokens for a prompt + input.
func EstimateInputTokens(systemPrompt, userPrompt string) int {
	return EstimateTokens(systemPrompt) + EstimateTokens(userPrompt)
}

// SelectTier chooses an appropriate tier based on content size.
func SelectTier(contentSize int) Tier {
	// Estimate tokens from content size
	estimatedTokens := EstimateTokensFromSize(contentSize)

	switch {
	case estimatedTokens <= 1500:
		return Tier1
	case estimatedTokens <= 10000:
		return Tier2
	default:
		return Tier3
	}
}

// EstimateTokensFromSize estimates tokens from content size in bytes.
func EstimateTokensFromSize(contentSize int) int {
	if contentSize == 0 {
		return 0
	}
	// Rough estimate: 4 characters per token
	return (contentSize + 3) / 4
}

// ValidateRequest validates that a request fits within a tier's budget.
func ValidateRequest(tier Tier, systemPrompt, userPrompt string, maxOutputTokens int) error {
	bm := DefaultBudgetManager()

	inputTokens := EstimateInputTokens(systemPrompt, userPrompt)
	outputTokens := maxOutputTokens

	return bm.CheckBudget(tier, inputTokens, outputTokens)
}

// BudgetExceededError is returned when a budget is exceeded.
type BudgetExceededError struct {
	Tier    Tier
	What    string
	Current int
	Max     int
}

func (e *BudgetExceededError) Error() string {
	return fmt.Sprintf("tier %s %s budget exceeded: %d/%d", e.Tier, e.What, e.Current, e.Max)
}

// IsBudgetExceeded checks if an error is a budget exceeded error.
func IsBudgetExceeded(err error) bool {
	_, ok := err.(*BudgetExceededError)
	return ok
}
