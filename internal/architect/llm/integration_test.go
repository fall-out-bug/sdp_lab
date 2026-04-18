package llm

import (
	"strings"
	"testing"
	"time"

	"sdp_dev/internal/architect"
)

// TestIntegration_HypothesizerWithRealProfile tests the ArchitectureHypothesizer
// with a realistic codebase profile.
func TestIntegration_HypothesizerWithRealProfile(t *testing.T) {
	h := NewArchitectureHypothesizer()

	// Create a realistic test profile
	profile := &architect.CodebaseProfile{
		Name:    "test-service",
		Summary: "A web service with layered architecture",
		FileTree: architect.FileTreeInfo{
			TopLevel: []string{"cmd/", "internal/", "pkg/", "api/"},
		},
		Dependencies: architect.DependencyInfo{
			Manifests: []architect.ManifestInfo{
				{Path: "go.mod", Language: "go", DepsCount: 15},
			},
		},
		ImportGraph: architect.ImportGraph{
			Clusters: []architect.ImportCluster{
				{ID: "handlers", Packages: []string{"internal/handlers"}, InternalEdges: 5, ExternalEdges: 2},
				{ID: "service", Packages: []string{"internal/service"}, InternalEdges: 3, ExternalEdges: 4},
				{ID: "repository", Packages: []string{"internal/repository"}, InternalEdges: 2, ExternalEdges: 3},
			},
		},
		Infra: architect.InfraInfo{
			Resources: []architect.ResourceInfo{
				{Type: "dockerfile", Name: "Dockerfile", Provider: "docker"},
			},
		},
		Metrics: architect.CodeMetrics{
			TotalFiles:         50,
			TotalLOC:           5000,
			ContainersDetected: 1,
		},
	}

	// Build the request
	request := h.BuildRequest(profile)

	// Verify the request contains expected content
	if request == "" {
		t.Fatal("BuildRequest returned empty string")
	}

	// Check for key sections
	expectedSections := []string{
		"Codebase Profile",
		"JSON",
		"style",
	}

	for _, section := range expectedSections {
		if !strings.Contains(request, section) {
			t.Errorf("Request missing expected section: %s", section)
		}
	}

	// Verify system prompt is set
	if h.SystemPrompt() == "" {
		t.Error("SystemPrompt returned empty string")
	}
}

// TestIntegration_PatternDetectorWithRealProfile tests the PatternDetector
// with a realistic codebase profile and style hypothesis.
func TestIntegration_PatternDetectorWithRealProfile(t *testing.T) {
	d := NewPatternDetector()

	profile := &architect.CodebaseProfile{
		Name:    "test-app",
		Summary: "An application using repository pattern",
		FileTree: architect.FileTreeInfo{
			TopLevel: []string{"internal/", "cmd/"},
		},
		ImportGraph: architect.ImportGraph{
			Clusters: []architect.ImportCluster{
				{ID: "repository", Packages: []string{"internal/repository"}, InternalEdges: 5},
			},
		},
	}

	hypothesis := &architect.StyleHypothesis{
		Styles: []architect.StyleScore{
			{Style: architect.StyleLayered, Confidence: 0.8, Evidence: []string{"Clear layer structure"}},
		},
	}

	// Build the request
	request := d.BuildRequest(profile, hypothesis)

	// Verify the request contains expected content
	if request == "" {
		t.Fatal("BuildRequest returned empty string")
	}

	// Check for key sections
	expectedSections := []string{
		"Codebase Profile",
		"Style Hypothesis",
		"JSON",
		"pattern",
	}

	for _, section := range expectedSections {
		if !strings.Contains(request, section) {
			t.Errorf("Request missing expected section: %s", section)
		}
	}
}

// TestIntegration_RiskAssessorWithRealProfile tests the RiskAssessor
// with a realistic codebase profile and patterns.
func TestIntegration_RiskAssessorWithRealProfile(t *testing.T) {
	r := NewRiskAssessor()

	profile := &architect.CodebaseProfile{
		Name:    "risky-app",
		Summary: "An application with potential security issues",
		FileTree: architect.FileTreeInfo{
			TopLevel: []string{"internal/", "cmd/"},
		},
		ImportGraph: architect.ImportGraph{
			CircularDependencies: []architect.CircularDep{
				{A: "internal/a", B: "internal/b", EdgeType: "import"},
			},
		},
	}

	patterns := []architect.DetectedPattern{
		{
			Category:   "gof",
			Name:       "repository",
			Confidence: 0.8,
			Evidence:   []string{"repository interfaces found"},
		},
	}

	// Build the request
	request := r.BuildRequest(profile, patterns)

	// Verify the request contains expected content
	if request == "" {
		t.Fatal("BuildRequest returned empty string")
	}

	// Check for key sections
	expectedSections := []string{
		"Codebase Profile",
		"Detected Patterns",
		"JSON",
		"risk",
	}

	for _, section := range expectedSections {
		if !strings.Contains(request, section) {
			t.Errorf("Request missing expected section: %s", section)
		}
	}
}

// TestIntegration_BudgetManager tests the BudgetManager with realistic scenarios.
func TestIntegration_BudgetManager(t *testing.T) {
	bm := DefaultBudgetManager()

	// Test Tier1 budget
	tier1Input := 1000
	tier1Output := 500
	err := bm.CheckBudget(Tier1, tier1Input, tier1Output)
	if err != nil {
		t.Errorf("Tier1 check failed: %v", err)
	}

	// Test recording usage
	err = bm.RecordUsage("node1", Tier1, tier1Input, tier1Output)
	if err != nil {
		t.Errorf("RecordUsage failed: %v", err)
	}

	// Verify tracking
	tracker := bm.GetTracker("node1")
	if tracker == nil {
		t.Fatal("GetTracker returned nil")
	}

	if tracker.InputTokens != tier1Input {
		t.Errorf("Expected %d input tokens, got %d", tier1Input, tracker.InputTokens)
	}

	if tracker.OutputTokens != tier1Output {
		t.Errorf("Expected %d output tokens, got %d", tier1Output, tracker.OutputTokens)
	}

	if tracker.TotalTokens != tier1Input+tier1Output {
		t.Errorf("Expected %d total tokens, got %d", tier1Input+tier1Output, tracker.TotalTokens)
	}

	// Test Tier2 budget
	tier2Input := 10000
	tier2Output := 5000
	err = bm.CheckBudget(Tier2, tier2Input, tier2Output)
	if err != nil {
		t.Errorf("Tier2 check failed: %v", err)
	}

	err = bm.RecordUsage("node2", Tier2, tier2Input, tier2Output)
	if err != nil {
		t.Errorf("RecordUsage failed: %v", err)
	}

	// Test tier-specific usage
	input, output, total := bm.TierUsage(Tier1)
	if input != tier1Input || output != tier1Output || total != tier1Input+tier1Output {
		t.Errorf("Tier1 usage mismatch: got (%d, %d, %d)", input, output, total)
	}

	// Test total usage
	_, _, grandTotal := bm.TotalUsage()
	expectedTotal := tier1Input + tier1Output + tier2Input + tier2Output
	if grandTotal != expectedTotal {
		t.Errorf("Total usage mismatch: expected %d, got %d", expectedTotal, grandTotal)
	}
}

// TestIntegration_CostTracker tests the CostTracker with realistic scenarios.
func TestIntegration_CostTracker(t *testing.T) {
	ct := NewCostTracker()

	// Record some usage
	model := "openai/gpt-4o-mini"
	inputTokens := 1000
	outputTokens := 500

	cost := ct.Record(model, inputTokens, outputTokens)
	if cost <= 0 {
		t.Errorf("Expected positive cost, got %f", cost)
	}

	// Verify model cost
	modelCost := ct.GetModelCost(model)
	if modelCost == nil {
		t.Fatal("GetModelCost returned nil")
	}

	if modelCost.InputTokens != inputTokens {
		t.Errorf("Expected %d input tokens, got %d", inputTokens, modelCost.InputTokens)
	}

	if modelCost.OutputTokens != outputTokens {
		t.Errorf("Expected %d output tokens, got %d", outputTokens, modelCost.OutputTokens)
	}

	if modelCost.Requests != 1 {
		t.Errorf("Expected 1 request, got %d", modelCost.Requests)
	}

	// Record another usage
	ct.Record(model, 500, 250)

	// Verify totals
	totalInput, totalOutput, _ := ct.TotalTokens()
	if totalInput != inputTokens+500 {
		t.Errorf("Expected %d total input tokens, got %d", inputTokens+500, totalInput)
	}

	if totalOutput != outputTokens+250 {
		t.Errorf("Expected %d total output tokens, got %d", outputTokens+250, totalOutput)
	}

	if ct.TotalRequests() != 2 {
		t.Errorf("Expected 2 total requests, got %d", ct.TotalRequests())
	}
}

// TestIntegration_EstimateCost tests cost estimation.
func TestIntegration_EstimateCost(t *testing.T) {
	model := "openai/gpt-4o-mini"
	inputTokens := 1000
	outputTokens := 500

	cost, err := EstimateCost(model, inputTokens, outputTokens)
	if err != nil {
		t.Errorf("EstimateCost failed: %v", err)
	}

	if cost <= 0 {
		t.Errorf("Expected positive cost, got %f", cost)
	}

	// Test with unknown model
	_, err = EstimateCost("unknown-model", inputTokens, outputTokens)
	if err == nil {
		t.Error("Expected error for unknown model, got nil")
	}
}

// TestIntegration_Auditor tests the Auditor with audit logging.
func TestIntegration_Auditor(t *testing.T) {
	a := NewAuditor()

	entry := AuditLog{
		Timestamp:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		RequestID:   "test-123",
		Model:       "openai/gpt-4o-mini",
		InputTokens: 1000,
		OutputTokens: 500,
		TotalTokens: 1500,
		CostUSD:     0.001,
		InputHash:   "abc123",
		Tier:        "tier1",
		Provider:    "openrouter",
		Success:     true,
	}

	a.Log(entry)

	if a.Count() != 1 {
		t.Errorf("Expected 1 audit log entry, got %d", a.Count())
	}

	logs := a.GetLogs()
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	if logs[0].RequestID != "test-123" {
		t.Errorf("Expected RequestID 'test-123', got %s", logs[0].RequestID)
	}

	// Test filtering by model
	modelLogs := a.GetLogsByModel("openai/gpt-4o-mini")
	if len(modelLogs) != 1 {
		t.Errorf("Expected 1 log entry for model, got %d", len(modelLogs))
	}

	// Test filtering by tier
	tierLogs := a.GetLogsByTier("tier1")
	if len(tierLogs) != 1 {
		t.Errorf("Expected 1 log entry for tier, got %d", len(tierLogs))
	}

	// Test clear
	a.Clear()
	if a.Count() != 0 {
		t.Errorf("Expected 0 logs after clear, got %d", a.Count())
	}
}

// TestIntegration_SelectTier tests tier selection based on content size.
func TestIntegration_SelectTier(t *testing.T) {
	tests := []struct {
		name        string
		contentSize int
		expected    Tier
	}{
		// contentSize is in bytes, SelectTier converts to tokens (4 chars per token)
		{"small content", 4000, Tier1},   // ~1000 tokens
		{"medium content", 40000, Tier2}, // ~10000 tokens
		{"large content", 80000, Tier3},  // ~20000 tokens
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SelectTier(tt.contentSize)
			if got != tt.expected {
				t.Errorf("SelectTier(%d) = %v, want %v", tt.contentSize, got, tt.expected)
			}
		})
	}
}
