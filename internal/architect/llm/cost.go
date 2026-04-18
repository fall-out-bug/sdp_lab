package llm

import (
	"fmt"
	"sync"
	"time"
)

// ModelPricing defines pricing for a model per 1M tokens.
type ModelPricing struct {
	InputPriceUSD  float64 // Price per 1M input tokens
	OutputPriceUSD float64 // Price per 1M output tokens
}

// Known pricing for common models (as of 2025).
// Prices are per 1M tokens in USD.
var knownPricing = map[string]ModelPricing{
	// OpenAI
	"gpt-4o-mini":           {InputPriceUSD: 0.150, OutputPriceUSD: 0.600},
	"gpt-4o":                {InputPriceUSD: 2.50, OutputPriceUSD: 10.00},
	"gpt-4-turbo":           {InputPriceUSD: 10.00, OutputPriceUSD: 30.00},
	"gpt-3.5-turbo":         {InputPriceUSD: 0.50, OutputPriceUSD: 1.50},

	// Anthropic
	"claude-3-5-sonnet":     {InputPriceUSD: 3.00, OutputPriceUSD: 15.00},
	"claude-3-5-haiku":      {InputPriceUSD: 0.80, OutputPriceUSD: 4.00},
	"claude-3-opus":         {InputPriceUSD: 15.00, OutputPriceUSD: 75.00},

	// OpenRouter (approximate aggregates)
	"openai/gpt-4o-mini":    {InputPriceUSD: 0.150, OutputPriceUSD: 0.600},
	"anthropic/claude-3.5-sonnet": {InputPriceUSD: 3.00, OutputPriceUSD: 15.00},

	// Local models (free)
	"llama3.2":              {InputPriceUSD: 0.0, OutputPriceUSD: 0.0},
	"llama3.1":              {InputPriceUSD: 0.0, OutputPriceUSD: 0.0},
	"mistral":               {InputPriceUSD: 0.0, OutputPriceUSD: 0.0},
}

// CostTracker tracks LLM API costs.
type CostTracker struct {
	mu           sync.Mutex
	perModel     map[string]*ModelCost
	totalCostUSD float64
	requests     int
}

// ModelCost tracks cost for a specific model.
type ModelCost struct {
	Model           string
	InputTokens     int
	OutputTokens    int
	TotalTokens     int
	Requests        int
	TotalCostUSD    float64
	AvgInputTokens  float64
	AvgOutputTokens float64
}

// CostRecord represents a single API call's cost.
type CostRecord struct {
	Timestamp   time.Time
	Model       string
	InputTokens int
	OutputTokens int
	CostUSD     float64
}

// NewCostTracker creates a new cost tracker.
func NewCostTracker() *CostTracker {
	return &CostTracker{
		perModel: make(map[string]*ModelCost),
	}
}

// Record records a single API call's token usage and cost.
func (ct *CostTracker) Record(model string, inputTokens, outputTokens int) float64 {
	pricing, ok := knownPricing[model]
	if !ok {
		// Default pricing for unknown models (conservative estimate)
		pricing = ModelPricing{InputPriceUSD: 1.0, OutputPriceUSD: 2.0}
	}

	inputCost := float64(inputTokens) / 1_000_000.0 * pricing.InputPriceUSD
	outputCost := float64(outputTokens) / 1_000_000.0 * pricing.OutputPriceUSD
	totalCost := inputCost + outputCost

	ct.mu.Lock()
	defer ct.mu.Unlock()

	modelCost, exists := ct.perModel[model]
	if !exists {
		modelCost = &ModelCost{Model: model}
		ct.perModel[model] = modelCost
	}

	modelCost.InputTokens += inputTokens
	modelCost.OutputTokens += outputTokens
	modelCost.TotalTokens += inputTokens + outputTokens
	modelCost.Requests++
	modelCost.TotalCostUSD += totalCost
	modelCost.AvgInputTokens = float64(modelCost.InputTokens) / float64(modelCost.Requests)
	modelCost.AvgOutputTokens = float64(modelCost.OutputTokens) / float64(modelCost.Requests)

	ct.totalCostUSD += totalCost
	ct.requests++

	return totalCost
}

// GetModelCost returns cost statistics for a specific model.
func (ct *CostTracker) GetModelCost(model string) *ModelCost {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	return ct.perModel[model]
}

// GetAllModelCosts returns cost statistics for all models.
func (ct *CostTracker) GetAllModelCosts() map[string]*ModelCost {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*ModelCost, len(ct.perModel))
	for k, v := range ct.perModel {
		copy := *v
		result[k] = &copy
	}
	return result
}

// TotalCost returns the total cost across all models.
func (ct *CostTracker) TotalCost() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	return ct.totalCostUSD
}

// TotalTokens returns the total tokens used across all models.
func (ct *CostTracker) TotalTokens() (input, output, total int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	for _, mc := range ct.perModel {
		input += mc.InputTokens
		output += mc.OutputTokens
		total += mc.TotalTokens
	}

	return input, output, total
}

// TotalRequests returns the total number of requests.
func (ct *CostTracker) TotalRequests() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	return ct.requests
}

// EstimateCost estimates the cost of a request before making it.
func EstimateCost(model string, inputTokens, outputTokens int) (float64, error) {
	pricing, ok := knownPricing[model]
	if !ok {
		return 0, fmt.Errorf("unknown model: %s", model)
	}

	inputCost := float64(inputTokens) / 1_000_000.0 * pricing.InputPriceUSD
	outputCost := float64(outputTokens) / 1_000_000.0 * pricing.OutputPriceUSD

	return inputCost + outputCost, nil
}

// EstimateCostFromText estimates cost from prompt text.
func EstimateCostFromText(model string, systemPrompt, userPrompt string, estimatedOutputTokens int) (float64, error) {
	inputTokens := EstimateInputTokens(systemPrompt, userPrompt)
	return EstimateCost(model, inputTokens, estimatedOutputTokens)
}

// FormatCost formats a cost in USD as a string.
func FormatCost(costUSD float64) string {
	if costUSD < 0.01 {
		return fmt.Sprintf("$%.4f", costUSD)
	}
	if costUSD < 1.0 {
		return fmt.Sprintf("$%.2f", costUSD)
	}
	return fmt.Sprintf("$%.2f", costUSD)
}

// Reset clears all tracking data.
func (ct *CostTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.perModel = make(map[string]*ModelCost)
	ct.totalCostUSD = 0
	ct.requests = 0
}

// Summary returns a formatted summary of costs.
func (ct *CostTracker) Summary() string {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.requests == 0 {
		return "No requests tracked"
	}

	input, output, total := ct.TotalTokens()
	avgCost := ct.totalCostUSD / float64(ct.requests)

	var summary string
	summary += fmt.Sprintf("Total Cost: %s\n", FormatCost(ct.totalCostUSD))
	summary += fmt.Sprintf("Total Requests: %d\n", ct.requests)
	summary += fmt.Sprintf("Average Cost per Request: %s\n", FormatCost(avgCost))
	summary += fmt.Sprintf("Total Tokens: %d (%d input, %d output)\n", total, input, output)
	summary += fmt.Sprintf("\nPer-Model Breakdown:\n")

	for model, mc := range ct.perModel {
		summary += fmt.Sprintf("  %s: %s (%d requests, %d tokens)\n",
			model, FormatCost(mc.TotalCostUSD), mc.Requests, mc.TotalTokens)
	}

	return summary
}

// AuditLog represents an audit log entry for LLM usage.
type AuditLog struct {
	Timestamp     time.Time `json:"timestamp"`
	RequestID     string    `json:"request_id"`
	Model         string    `json:"model"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	TotalTokens   int       `json:"total_tokens"`
	CostUSD       float64   `json:"cost_usd"`
	InputHash     string    `json:"input_hash"`    // Sanitized input hash for auditing
	Tier          string    `json:"tier"`
	Provider      string    `json:"provider"`
	Success       bool      `json:"success"`
	ErrorMsg      string    `json:"error,omitempty"`
}

// Auditor tracks audit logs for compliance and cost analysis.
type Auditor struct {
	mu   sync.Mutex
	logs []AuditLog
}

// NewAuditor creates a new auditor.
func NewAuditor() *Auditor {
	return &Auditor{
		logs: make([]AuditLog, 0, 100),
	}
}

// Log records an audit entry.
func (a *Auditor) Log(entry AuditLog) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logs = append(a.logs, entry)
}

// GetLogs returns all audit logs.
func (a *Auditor) GetLogs() []AuditLog {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Return a copy
	logs := make([]AuditLog, len(a.logs))
	copy(logs, a.logs)
	return logs
}

// GetLogsByModel returns logs for a specific model.
func (a *Auditor) GetLogsByModel(model string) []AuditLog {
	a.mu.Lock()
	defer a.mu.Unlock()

	var filtered []AuditLog
	for _, log := range a.logs {
		if log.Model == model {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// GetLogsByTier returns logs for a specific tier.
func (a *Auditor) GetLogsByTier(tier string) []AuditLog {
	a.mu.Lock()
	defer a.mu.Unlock()

	var filtered []AuditLog
	for _, log := range a.logs {
		if log.Tier == tier {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// Clear clears all audit logs.
func (a *Auditor) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.logs = make([]AuditLog, 0, 100)
}

// Count returns the number of audit log entries.
func (a *Auditor) Count() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	return len(a.logs)
}
