package classify

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"sdp_dev/internal/architect"
	"sdp_dev/internal/discovery"
)

// AuditEntry records a single LLM call for compliance.
type AuditEntry struct {
	Timestamp   string `json:"timestamp"`
	CallType    string `json:"call_type"` // "style", "pattern", "risk"
	Model       string `json:"model"`
	InputHash   string `json:"input_hash"`
	InputTokens int    `json:"input_tokens,omitempty"`
	CostUSD     float64 `json:"cost_usd,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// HypothesisResult holds all LLM-generated architecture analysis.
type HypothesisResult struct {
	StyleHypothesis  architect.StyleHypothesis   `json:"style_hypothesis"`
	Patterns         []architect.DetectedPattern `json:"patterns,omitempty"`
	Risks            []architect.ArchRisk        `json:"risks,omitempty"`
	TotalInputTokens int                         `json:"total_input_tokens"`
	TotalCostUSD     float64                     `json:"total_cost_usd"`
	AuditLog         []AuditEntry                `json:"audit_log,omitempty"`
}

// Hypothesizer runs LLM analysis on a CodebaseProfile.
type Hypothesizer struct {
	client *discovery.LLMClient
	model  string
}

// Default model is Google Gemini 2.0 Flash.
const defaultModel = "google/gemini-2.0-flash-001"

// NewHypothesizer creates a new Hypothesizer with the given LLM client and model.
// If model is empty, defaults to "google/gemini-2.0-flash-001".
func NewHypothesizer(client *discovery.LLMClient, model string) *Hypothesizer {
	if model == "" {
		model = defaultModel
	}
	return &Hypothesizer{
		client: client,
		model:  model,
	}
}

// Analyze runs 3 parallel LLM analyses on the codebase profile:
// 1. Architecture style hypothesis
// 2. Design pattern detection
// 3. Risk assessment
func (h *Hypothesizer) Analyze(ctx context.Context, profile *architect.CodebaseProfile) (*HypothesisResult, error) {
	// Serialize profile to JSON for LLM consumption.
	profileJSON, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal profile: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	result := &HypothesisResult{}

	// Run 3 LLM calls in parallel.
	for _, fn := range []func(context.Context, string) error{
		func(ctx context.Context, profileJSON string) error {
			styleHypothesis, inputTokens, cost, err := h.analyzeStyle(ctx, profileJSON)

			// Create audit entry
			audit := AuditEntry{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				CallType:  "style",
				Model:     h.model,
				InputHash: sha256Hash(profileJSON),
				Success:   err == nil,
			}
			if err != nil {
				audit.Error = err.Error()
			} else {
				audit.InputTokens = inputTokens
				audit.CostUSD = cost
			}
			mu.Lock()
			result.AuditLog = append(result.AuditLog, audit)
			mu.Unlock()

			if err != nil {
				return fmt.Errorf("style analysis: %w", err)
			}
			mu.Lock()
			result.StyleHypothesis = *styleHypothesis
			result.TotalInputTokens += inputTokens
			result.TotalCostUSD += cost
			mu.Unlock()
			return nil
		},
		func(ctx context.Context, profileJSON string) error {
			patterns, inputTokens, cost, err := h.detectPatterns(ctx, profileJSON)

			// Create audit entry
			audit := AuditEntry{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				CallType:  "pattern",
				Model:     h.model,
				InputHash: sha256Hash(profileJSON),
				Success:   err == nil,
			}
			if err != nil {
				audit.Error = err.Error()
			} else {
				audit.InputTokens = inputTokens
				audit.CostUSD = cost
			}
			mu.Lock()
			result.AuditLog = append(result.AuditLog, audit)
			mu.Unlock()

			if err != nil {
				return fmt.Errorf("pattern detection: %w", err)
			}
			mu.Lock()
			result.Patterns = patterns
			result.TotalInputTokens += inputTokens
			result.TotalCostUSD += cost
			mu.Unlock()
			return nil
		},
		func(ctx context.Context, profileJSON string) error {
			risks, inputTokens, cost, err := h.assessRisks(ctx, profileJSON)

			// Create audit entry
			audit := AuditEntry{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				CallType:  "risk",
				Model:     h.model,
				InputHash: sha256Hash(profileJSON),
				Success:   err == nil,
			}
			if err != nil {
				audit.Error = err.Error()
			} else {
				audit.InputTokens = inputTokens
				audit.CostUSD = cost
			}
			mu.Lock()
			result.AuditLog = append(result.AuditLog, audit)
			mu.Unlock()

			if err != nil {
				return fmt.Errorf("risk assessment: %w", err)
			}
			mu.Lock()
			result.Risks = risks
			result.TotalInputTokens += inputTokens
			result.TotalCostUSD += cost
			mu.Unlock()
			return nil
		},
	} {
		wg.Add(1)
		go func(f func(context.Context, string) error) {
			defer wg.Done()
			if err := f(ctx, string(profileJSON)); err != nil {
				mu.Lock()
				defer mu.Unlock()
				errs = append(errs, err)
			}
		}(fn)
	}
	wg.Wait()

	// Return partial results even if some calls failed
	if len(errs) > 0 {
		return result, fmt.Errorf("LLM analysis completed with %d error(s): %w", len(errs), errors.Join(errs...))
	}
	return result, nil
}

// analyzeStyle sends a prompt to score architecture style hypotheses.
func (h *Hypothesizer) analyzeStyle(ctx context.Context, profileJSON string) (*architect.StyleHypothesis, int, float64, error) {
	prompt := fmt.Sprintf(`Analyze the following codebase profile and score each architecture style hypothesis.
Return JSON: {"styles": [{"style": "<name>", "confidence": 0.0-1.0, "evidence": ["..."]}], "human_input_needed": ["..."]}

Styles to evaluate: layered, modular, microservices, event_driven, serverless, monorepo_multi_service, library, infra_repo

Codebase profile:
%s`, profileJSON)

	req := discovery.ChatRequest{
		Model:       h.model,
		Messages:    []discovery.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4000,
		Temperature: 0.1,
	}

	resp, err := h.client.Chat(ctx, req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("LLM chat: %w", err)
	}

	styleHypothesis, err := parseStyleResponse(resp.Content)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("parse style response: %w", err)
	}

	return styleHypothesis, resp.InputTokens, resp.CostUSD, nil
}

// detectPatterns sends a prompt to detect design patterns.
func (h *Hypothesizer) detectPatterns(ctx context.Context, profileJSON string) ([]architect.DetectedPattern, int, float64, error) {
	prompt := fmt.Sprintf(`Analyze the following codebase profile for design patterns.
Return JSON: {"patterns": [{"category": "<gof|ddd|infrastructure>", "name": "...", "confidence": 0.0-1.0, "evidence": ["..."], "location": "..."}]}

Categories: GoF (observer, strategy, factory, etc.), DDD (aggregate_root, repository, domain_service, etc.), Infrastructure (circuit_breaker, saga, cqrs, etc.)

Codebase profile:
%s`, profileJSON)

	req := discovery.ChatRequest{
		Model:       h.model,
		Messages:    []discovery.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4000,
		Temperature: 0.1,
	}

	resp, err := h.client.Chat(ctx, req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("LLM chat: %w", err)
	}

	patterns, err := parsePatternsResponse(resp.Content)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("parse patterns response: %w", err)
	}

	return patterns, resp.InputTokens, resp.CostUSD, nil
}

// assessRisks sends a prompt to identify architectural risks.
func (h *Hypothesizer) assessRisks(ctx context.Context, profileJSON string) ([]architect.ArchRisk, int, float64, error) {
	prompt := fmt.Sprintf(`Analyze the following codebase profile for architectural risks.
Return JSON: {"risks": [{"severity": "<high|medium|low>", "category": "...", "description": "...", "affected": ["..."]}]}

Categories: missing_contract, circular_dependency, pii_exposure, god_module, test_gap, tech_debt, coupling_hotspot

Codebase profile:
%s`, profileJSON)

	req := discovery.ChatRequest{
		Model:       h.model,
		Messages:    []discovery.Message{{Role: "user", Content: prompt}},
		MaxTokens:   4000,
		Temperature: 0.1,
	}

	resp, err := h.client.Chat(ctx, req)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("LLM chat: %w", err)
	}

	risks, err := parseRisksResponse(resp.Content)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("parse risks response: %w", err)
	}

	return risks, resp.InputTokens, resp.CostUSD, nil
}

// extractJSON extracts JSON from markdown code blocks if present.
func extractJSON(content string) (string, error) {
	// Check for markdown code block: ```json ... ```
	re := regexp.MustCompile("```(?:json)?\\s*\\n([\\s\\S]*?)\\n\\s*```")
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	// No code block found, try to extract JSON directly.
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return trimmed, nil
	}

	return "", fmt.Errorf("no JSON found in response")
}

// parseStyleResponse parses a StyleHypothesis from LLM response.
func parseStyleResponse(content string) (*architect.StyleHypothesis, error) {
	jsonStr, err := extractJSON(content)
	if err != nil {
		return nil, err
	}

	var result struct {
		Styles           []architect.StyleScore `json:"styles"`
		HumanInputNeeded []string               `json:"human_input_needed,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("unmarshal style hypothesis: %w", err)
	}

	return &architect.StyleHypothesis{
		Styles:           result.Styles,
		HumanInputNeeded: result.HumanInputNeeded,
	}, nil
}

// parsePatternsResponse parses detected patterns from LLM response.
func parsePatternsResponse(content string) ([]architect.DetectedPattern, error) {
	jsonStr, err := extractJSON(content)
	if err != nil {
		return nil, err
	}

	var result struct {
		Patterns []architect.DetectedPattern `json:"patterns"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("unmarshal patterns: %w", err)
	}

	return result.Patterns, nil
}

// parseRisksResponse parses architectural risks from LLM response.
func parseRisksResponse(content string) ([]architect.ArchRisk, error) {
	jsonStr, err := extractJSON(content)
	if err != nil {
		return nil, err
	}

	var result struct {
		Risks []architect.ArchRisk `json:"risks"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("unmarshal risks: %w", err)
	}

	return result.Risks, nil
}

// sha256Hash returns the hexadecimal SHA-256 hash of a string.
func sha256Hash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
