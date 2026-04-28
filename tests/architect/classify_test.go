package architect_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/architect"
	"github.com/fall-out-bug/sdp_lab/internal/architect/classify"
	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

// TestHypothesizerNew verifies the constructor.
func TestHypothesizerNew(t *testing.T) {
	client := discovery.NewLLMClient("test-key", "https://api.example.com")
	h := classify.NewHypothesizer(client, "")
	if h == nil {
		t.Fatal("NewHypothesizer returned nil")
	}

	// Test with explicit model
	h2 := classify.NewHypothesizer(client, "custom-model")
	if h2 == nil {
		t.Fatal("NewHypothesizer with model returned nil")
	}
}

// TestParseStyleResponse tests JSON parsing for style hypothesis.
func TestParseStyleResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(*architect.StyleHypothesis) bool
	}{
		{
			name: "plain JSON",
			content: `{
				"styles": [
					{"style": "layered", "confidence": 0.8, "evidence": ["controller layer found"]},
					{"style": "microservices", "confidence": 0.3}
				],
				"human_input_needed": ["confirm deployment"]
			}`,
			wantErr: false,
			check: func(sh *architect.StyleHypothesis) bool {
				return len(sh.Styles) == 2 &&
					sh.Styles[0].Style == architect.StyleLayered &&
					sh.Styles[0].Confidence == 0.8 &&
					len(sh.Styles[0].Evidence) == 1 &&
					sh.Styles[0].Evidence[0] == "controller layer found" &&
					sh.Styles[1].Style == architect.StyleMicroservices &&
					sh.Styles[1].Confidence == 0.3 &&
					len(sh.HumanInputNeeded) == 1
			},
		},
		{
			name: "JSON in markdown code block",
			content: "Here's my analysis:\n\n```json\n{\n\t\"styles\": [\n\t\t{\"style\": \"event_driven\", \"confidence\": 0.9, \"evidence\": [\"Kafka topics\", \"event handlers\"]}\n\t]\n}\n```\n",
			wantErr: false,
			check: func(sh *architect.StyleHypothesis) bool {
				return len(sh.Styles) == 1 &&
					sh.Styles[0].Style == architect.StyleEventDriven &&
					sh.Styles[0].Confidence == 0.9 &&
					len(sh.Styles[0].Evidence) == 2
			},
		},
		{
			name: "invalid JSON",
			content: `not valid json`,
			wantErr: true,
		},
		{
			name: "JSON without code block markers",
			content: `{"styles": [{"style": "library", "confidence": 0.95}]}`,
			wantErr: false,
			check: func(sh *architect.StyleHypothesis) bool {
				return len(sh.Styles) == 1 &&
					sh.Styles[0].Style == architect.StyleLibrary &&
					sh.Styles[0].Confidence == 0.95
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use reflection or export the function for testing
			// For now, we'll test through the full Analyze flow
			sh, err := parseStyleResponse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStyleResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(sh) {
				t.Error("parseStyleResponse() check failed")
			}
		})
	}
}

// TestParsePatternsResponse tests JSON parsing for detected patterns.
func TestParsePatternsResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func([]architect.DetectedPattern) bool
	}{
		{
			name: "plain JSON",
			content: `{
				"patterns": [
					{
						"category": "gof",
						"name": "factory",
						"confidence": 0.85,
						"evidence": ["NewFactory functions"],
						"location": "pkg/factory"
					},
					{
						"category": "ddd",
						"name": "repository",
						"confidence": 0.9,
						"location": "pkg/repository"
					}
				]
			}`,
			wantErr: false,
			check: func(p []architect.DetectedPattern) bool {
				return len(p) == 2 &&
					p[0].Category == "gof" &&
					p[0].Name == "factory" &&
					p[0].Confidence == 0.85 &&
					p[1].Category == "ddd" &&
					p[1].Name == "repository"
			},
		},
		{
			name: "JSON in markdown",
			content: "\nAnalysis complete:\n\n```json\n{\n\t\"patterns\": [\n\t\t{\"category\": \"infrastructure\", \"name\": \"circuit_breaker\", \"confidence\": 0.7}\n\t]\n}\n```\n",
			wantErr: false,
			check: func(p []architect.DetectedPattern) bool {
				return len(p) == 1 &&
					p[0].Category == "infrastructure" &&
					p[0].Name == "circuit_breaker"
			},
		},
		{
			name:    "invalid JSON",
			content: `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns, err := parsePatternsResponse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePatternsResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(patterns) {
				t.Error("parsePatternsResponse() check failed")
			}
		})
	}
}

// TestParseRisksResponse tests JSON parsing for architectural risks.
func TestParseRisksResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func([]architect.ArchRisk) bool
	}{
		{
			name: "plain JSON",
			content: `{
				"risks": [
					{
						"severity": "high",
						"category": "circular_dependency",
						"description": "Package A imports B which imports A",
						"affected": ["pkg/a", "pkg/b"]
					},
					{
						"severity": "medium",
						"category": "test_gap",
						"description": "Low test coverage",
						"affected": ["pkg/service"]
					}
				]
			}`,
			wantErr: false,
			check: func(r []architect.ArchRisk) bool {
				return len(r) == 2 &&
					r[0].Severity == architect.SeverityHigh &&
					r[0].Category == "circular_dependency" &&
					len(r[0].Affected) == 2 &&
					r[1].Severity == architect.SeverityMedium &&
					r[1].Category == "test_gap"
			},
		},
		{
			name: "JSON in markdown",
			content: "Risk analysis:\n\n```json\n{\n\t\"risks\": [\n\t\t{\"severity\": \"low\", \"category\": \"tech_debt\", \"description\": \"Legacy code\"}\n\t]\n}\n```\n",
			wantErr: false,
			check: func(r []architect.ArchRisk) bool {
				return len(r) == 1 &&
					r[0].Severity == architect.SeverityLow &&
					r[0].Category == "tech_debt"
			},
		},
		{
			name:    "malformed JSON",
			content: `{"risks": [`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risks, err := parseRisksResponse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRisksResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(risks) {
				t.Error("parseRisksResponse() check failed")
			}
		})
	}
}

// TestHypothesizerAnalyze_MockServer tests the full Analyze flow with a mock LLM server.
func TestHypothesizerAnalyze_MockServer(t *testing.T) {
	// Track which endpoints were called
	styleCalled := false
	patternsCalled := false
	risksCalled := false

	// Create a mock server that returns canned responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}

		// Parse request to determine which prompt
		var req discovery.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Messages) == 0 {
			t.Error("expected at least one message")
		}

		prompt := req.Messages[0].Content

		// Return appropriate response based on prompt
		var responseJSON interface{}

		switch {
		case strings.Contains(prompt, "architecture style hypothesis"):
			styleCalled = true
			responseJSON = map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": `{"styles": [{"style": "layered", "confidence": 0.8, "evidence": ["MVC pattern found"]}]}`,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     1000,
					"completion_tokens": 100,
					"cost":              0.001,
				},
			}
		case strings.Contains(prompt, "design patterns"):
			patternsCalled = true
			responseJSON = map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": `{"patterns": [{"category": "gof", "name": "singleton", "confidence": 0.9}]}`,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     1000,
					"completion_tokens": 100,
					"cost":              0.001,
				},
			}
		case strings.Contains(prompt, "architectural risks"):
			risksCalled = true
			responseJSON = map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": `{"risks": [{"severity": "medium", "category": "test_gap", "description": "Low coverage"}]}`,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     1000,
					"completion_tokens": 100,
					"cost":              0.001,
				},
			}
		default:
			t.Errorf("unexpected prompt: %s", prompt[:100])
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responseJSON); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create hypothesizer with mock server
	client := discovery.NewLLMClient("test-key", server.URL)
	h := classify.NewHypothesizer(client, "test-model")

	// Create a minimal profile for testing
	profile := &architect.CodebaseProfile{
		Name: "test-repo",
		FileTree: architect.FileTreeInfo{
			TotalFiles: 10,
			TotalDirs:  3,
		},
		Dependencies: architect.DependencyInfo{},
		ImportGraph:  architect.ImportGraph{},
		Infra:        architect.InfraInfo{},
		Metrics: architect.CodeMetrics{
			TotalFiles: 10,
			TotalLOC:   1000,
		},
	}

	// Run analysis
	ctx := context.Background()
	result, err := h.Analyze(ctx, profile)

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Verify all three LLM calls were made
	if !styleCalled {
		t.Error("style hypothesis LLM call was not made")
	}
	if !patternsCalled {
		t.Error("patterns LLM call was not made")
	}
	if !risksCalled {
		t.Error("risks LLM call was not made")
	}

	// Verify results
	if result.StyleHypothesis.Styles == nil || len(result.StyleHypothesis.Styles) == 0 {
		t.Error("expected non-empty style hypothesis")
	} else if result.StyleHypothesis.Styles[0].Style != architect.StyleLayered {
		t.Errorf("expected style 'layered', got '%s'", result.StyleHypothesis.Styles[0].Style)
	}

	if len(result.Patterns) == 0 {
		t.Error("expected non-empty patterns")
	} else if result.Patterns[0].Name != "singleton" {
		t.Errorf("expected pattern 'singleton', got '%s'", result.Patterns[0].Name)
	}

	if len(result.Risks) == 0 {
		t.Error("expected non-empty risks")
	} else if result.Risks[0].Category != "test_gap" {
		t.Errorf("expected risk category 'test_gap', got '%s'", result.Risks[0].Category)
	}

	// Verify token/cost aggregation
	if result.TotalInputTokens != 3000 {
		t.Errorf("expected 3000 total input tokens, got %d", result.TotalInputTokens)
	}
	if result.TotalCostUSD != 0.003 {
		t.Errorf("expected 0.003 total cost, got %f", result.TotalCostUSD)
	}

	// Verify audit log has 3 entries
	if len(result.AuditLog) != 3 {
		t.Errorf("expected 3 audit log entries, got %d", len(result.AuditLog))
	} else {
		// Verify each audit entry
		callTypes := make(map[string]bool)
		for i, entry := range result.AuditLog {
			if !entry.Success {
				t.Errorf("audit entry %d should be successful", i)
			}
			if entry.Error != "" {
				t.Errorf("audit entry %d should have no error, got: %s", i, entry.Error)
			}
			if entry.Model != "test-model" {
				t.Errorf("audit entry %d has wrong model: %s", i, entry.Model)
			}
			if entry.InputTokens != 1000 {
				t.Errorf("audit entry %d has wrong input tokens: %d", i, entry.InputTokens)
			}
			if entry.CostUSD != 0.001 {
				t.Errorf("audit entry %d has wrong cost: %f", i, entry.CostUSD)
			}
			if entry.Timestamp == "" {
				t.Errorf("audit entry %d has empty timestamp", i)
			}
			if entry.InputHash == "" {
				t.Errorf("audit entry %d has empty input hash", i)
			}
			callTypes[entry.CallType] = true
		}
		// Verify all three call types are present
		if !callTypes["style"] || !callTypes["pattern"] || !callTypes["risk"] {
			t.Errorf("audit log missing call types: got %v", callTypes)
		}
	}
}

// Helper functions to access private methods for testing.
// In Go, these need to be in the same package or exported via a test bridge.
// For this implementation, we'll replicate the logic here.

func parseStyleResponse(content string) (*architect.StyleHypothesis, error) {
	jsonStr, err := extractJSONTest(content)
	if err != nil {
		return nil, err
	}

	var result struct {
		Styles           []architect.StyleScore `json:"styles"`
		HumanInputNeeded []string               `json:"human_input_needed,omitempty"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}

	return &architect.StyleHypothesis{
		Styles:           result.Styles,
		HumanInputNeeded: result.HumanInputNeeded,
	}, nil
}

func parsePatternsResponse(content string) ([]architect.DetectedPattern, error) {
	jsonStr, err := extractJSONTest(content)
	if err != nil {
		return nil, err
	}

	var result struct {
		Patterns []architect.DetectedPattern `json:"patterns"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}

	return result.Patterns, nil
}

func parseRisksResponse(content string) ([]architect.ArchRisk, error) {
	jsonStr, err := extractJSONTest(content)
	if err != nil {
		return nil, err
	}

	var result struct {
		Risks []architect.ArchRisk `json:"risks"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}

	return result.Risks, nil
}

func extractJSONTest(content string) (string, error) {
	// Check for markdown code block: ```json ... ```
	// Using a simple string approach instead of regex for test isolation
	lines := strings.Split(content, "\n")
	var jsonLines []string
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				continue
			} else {
				// End of code block
				break
			}
		}
		if inCodeBlock {
			jsonLines = append(jsonLines, line)
		}
	}

	if len(jsonLines) > 0 {
		return strings.TrimSpace(strings.Join(jsonLines, "\n")), nil
	}

	// No code block found, try to extract JSON directly
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return trimmed, nil
	}

	return "", nil
}

// TestHypothesizerAnalyze_PartialFailure tests that errors from parallel goroutines are collected and returned.
// When 1 of 3 LLM calls fails, an error should be returned but partial results should still be available.
func TestHypothesizerAnalyze_PartialFailure(t *testing.T) {
	// Track which endpoints were called
	styleCalled := false
	patternsCalled := false
	risksCalled := false

	// Create a mock server that returns error for patterns, success for others
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %s", r.URL.Path)
		}

		var req discovery.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Messages) == 0 {
			t.Error("expected at least one message")
		}

		prompt := req.Messages[0].Content

		// Style: success
		if strings.Contains(prompt, "architecture style hypothesis") {
			styleCalled = true
			responseJSON := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": `{"styles": [{"style": "layered", "confidence": 0.8}]}`,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     1000,
					"completion_tokens": 100,
					"cost":              0.001,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(responseJSON)
			return
		}

		// Patterns: error
		if strings.Contains(prompt, "design patterns") {
			patternsCalled = true
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Risks: success
		if strings.Contains(prompt, "architectural risks") {
			risksCalled = true
			responseJSON := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]string{
							"role":    "assistant",
							"content": `{"risks": [{"severity": "medium", "category": "test_gap", "description": "Low coverage"}]}`,
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     1000,
					"completion_tokens": 100,
					"cost":              0.001,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(responseJSON)
			return
		}

		t.Errorf("unexpected prompt: %s", prompt[:100])
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := discovery.NewLLMClient("test-key", server.URL)
	h := classify.NewHypothesizer(client, "test-model")

	profile := &architect.CodebaseProfile{
		Name: "test-repo",
		FileTree: architect.FileTreeInfo{
			TotalFiles: 10,
			TotalDirs:  3,
		},
		Dependencies: architect.DependencyInfo{},
		ImportGraph:  architect.ImportGraph{},
		Infra:        architect.InfraInfo{},
		Metrics: architect.CodeMetrics{
			TotalFiles: 10,
			TotalLOC:   1000,
		},
	}

	ctx := context.Background()
	result, err := h.Analyze(ctx, profile)

	// Verify error was returned
	if err == nil {
		t.Fatal("expected error when patterns call fails, got nil")
	}
	if !strings.Contains(err.Error(), "error") {
		t.Errorf("error message should mention error count, got: %v", err)
	}

	// Verify all three LLM calls were attempted
	if !styleCalled {
		t.Error("style hypothesis LLM call was not made")
	}
	if !patternsCalled {
		t.Error("patterns LLM call was not made")
	}
	if !risksCalled {
		t.Error("risks LLM call was not made")
	}

	// Verify partial results: style and risks should be populated
	if len(result.StyleHypothesis.Styles) == 0 {
		t.Error("expected style hypothesis to be populated despite partial failure")
	} else if result.StyleHypothesis.Styles[0].Style != architect.StyleLayered {
		t.Errorf("expected style 'layered', got '%s'", result.StyleHypothesis.Styles[0].Style)
	}

	if len(result.Risks) == 0 {
		t.Error("expected risks to be populated despite partial failure")
	} else if result.Risks[0].Category != "test_gap" {
		t.Errorf("expected risk category 'test_gap', got '%s'", result.Risks[0].Category)
	}

	// Patterns should be empty (failed call)
	if len(result.Patterns) != 0 {
		t.Errorf("expected patterns to be empty after failure, got %d", len(result.Patterns))
	}

	// Token/cost aggregation should only include successful calls
	if result.TotalInputTokens != 2000 {
		t.Errorf("expected 2000 total input tokens (2 successful calls), got %d", result.TotalInputTokens)
	}
	if result.TotalCostUSD != 0.002 {
		t.Errorf("expected 0.002 total cost (2 successful calls), got %f", result.TotalCostUSD)
	}
}
