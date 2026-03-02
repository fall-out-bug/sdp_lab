package modelgateway

import (
	"context"
	"testing"
	"time"
)

func TestRoutingInput(t *testing.T) {
	input := RoutingInput{
		TaskClass:      TaskClassCode,
		Sensitivity:    SensitivityInternal,
		MaxLatencyMs:   1000,
		RequiresVision: false,
		TenantID:       "tenant-1",
	}

	if input.TaskClass != TaskClassCode {
		t.Errorf("expected code task class, got %s", input.TaskClass)
	}
}

func TestRoutingDecision(t *testing.T) {
	decision := RoutingDecision{
		SelectedProvider: "openai",
		SelectedModel:    "gpt-4o",
		FallbackChain:    []ProviderID{"anthropic", "selfhosted"},
		DecisionReason:   "test decision",
		EvaluatedAt:      time.Now(),
	}

	if len(decision.FallbackChain) != 2 {
		t.Errorf("expected 2 fallback providers, got %d", len(decision.FallbackChain))
	}
}

func TestRoutingEvidenceToJSON(t *testing.T) {
	evidence := RoutingEvidence{
		Decision: RoutingDecision{
			SelectedProvider: "openai",
			SelectedModel:    "gpt-4o",
		},
		Input: RoutingInput{
			TaskClass: TaskClassCode,
		},
		Timestamp: time.Now(),
	}

	json := evidence.ToJSON()
	if json == "" {
		t.Error("expected non-empty JSON")
	}
}

func TestPolicyRouter(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{id: "openai", available: true})
	registry.Register(&mockProvider{id: "anthropic", available: true})

	router := NewPolicyRouter(registry)

	input := RoutingInput{
		TaskClass:   TaskClassCode,
		Sensitivity: SensitivityPublic,
		TenantID:    "test",
	}

	decision, evidence, err := router.Route(context.Background(), input)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if decision.SelectedProvider == "" {
		t.Error("expected selected provider")
	}
	if evidence == nil {
		t.Error("expected evidence")
	}
}

func TestPolicyRouterWithVision(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{
		id:           "openai",
		available:    true,
		capabilities: ModelCapabilities{Vision: true},
	})
	registry.Register(&mockProvider{
		id:           "anthropic",
		available:    true,
		capabilities: ModelCapabilities{Vision: false},
	})

	router := NewPolicyRouter(registry)

	input := RoutingInput{
		TaskClass:      TaskClassAnalysis,
		Sensitivity:    SensitivityPublic,
		RequiresVision: true,
	}

	decision, _, err := router.Route(context.Background(), input)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if decision.SelectedProvider != "openai" {
		t.Errorf("expected openai for vision, got %s", decision.SelectedProvider)
	}
}

func TestPolicyRouterRestrictedSensitivity(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{id: "openai", available: true})
	registry.Register(&mockProvider{id: "selfhosted", available: true})

	router := NewPolicyRouter(registry)

	input := RoutingInput{
		TaskClass:   TaskClassCode,
		Sensitivity: SensitivityRestricted,
	}

	decision, _, err := router.Route(context.Background(), input)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if decision.SelectedProvider != "selfhosted" {
		t.Errorf("expected selfhosted for restricted, got %s", decision.SelectedProvider)
	}
}

func TestPolicyRouterWithTenantConfig(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{id: "openai", available: true})
	registry.Register(&mockProvider{id: "anthropic", available: true})
	registry.Register(&mockProvider{id: "selfhosted", available: true})

	tenantStore := NewInMemoryTenantStore()
	tenantStore.Set(&TenantConfig{
		TenantID:        "tenant-1",
		DefaultProvider: "anthropic",
		FallbackChain:   []ProviderID{"selfhosted"},
		Constraints: RoutingConstraints{
			AllowedProviders: []ProviderID{"anthropic", "selfhosted"},
		},
	})

	router := NewPolicyRouter(registry, WithTenantStore(tenantStore))

	input := RoutingInput{
		TaskClass:   TaskClassCode,
		Sensitivity: SensitivityPublic,
		TenantID:    "tenant-1",
	}

	decision, _, err := router.Route(context.Background(), input)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}

	valid := false
	for _, p := range decision.Constraints.AllowedProviders {
		if p == decision.SelectedProvider {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("selected provider not in allowed list")
	}
}

func TestPolicyRouterSimulationMode(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{id: "openai", available: true})

	router := NewPolicyRouter(registry, WithSimulationMode(true))

	input := RoutingInput{
		TaskClass:   TaskClassCode,
		Sensitivity: SensitivityPublic,
	}

	_, _, err := router.Route(context.Background(), input)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}

	log := router.GetDecisionLog()
	if len(log) != 0 {
		t.Error("expected empty log in simulation mode")
	}

	router.SetSimulationMode(false)
	_, _, _ = router.Route(context.Background(), input)

	log = router.GetDecisionLog()
	if len(log) != 1 {
		t.Errorf("expected 1 logged decision, got %d", len(log))
	}
}

func TestPolicyRouterChat(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{
		id:        "openai",
		available: true,
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{
				ID:      "resp-1",
				Model:   req.Model,
				Message: Message{Role: RoleAssistant, Content: "response"},
			}, nil
		},
	})

	router := NewPolicyRouter(registry)

	input := RoutingInput{
		TaskClass:   TaskClassCode,
		Sensitivity: SensitivityPublic,
	}

	req := &ChatRequest{
		Model:    "gpt-4o",
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	}

	resp, evidence, err := router.Chat(context.Background(), input, req)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}
	if resp == nil {
		t.Error("expected response")
	}
	if evidence == nil {
		t.Error("expected evidence")
	}
}

func TestSimplePolicyEvaluator(t *testing.T) {
	rules := []RoutingRule{
		{Match: RoutingMatch{ModelPrefix: "claude"}, Target: "anthropic"},
		{Match: RoutingMatch{ModelPrefix: "gpt"}, Target: "openai"},
	}

	evaluator := NewSimplePolicyEvaluator(rules)

	input := RoutingInput{
		TaskClass: TaskClassCode,
		ModelHint: "claude-3-opus",
	}

	decision, err := evaluator.Evaluate(context.Background(), input)
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if decision.SelectedProvider != "anthropic" {
		t.Errorf("expected anthropic, got %s", decision.SelectedProvider)
	}
}

func TestInMemoryTenantStore(t *testing.T) {
	store := NewInMemoryTenantStore()

	config := &TenantConfig{
		TenantID:        "tenant-1",
		DefaultProvider: "openai",
		FallbackChain:   []ProviderID{"anthropic"},
	}

	store.Set(config)

	got, err := store.Get(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.DefaultProvider != "openai" {
		t.Errorf("expected openai, got %s", got.DefaultProvider)
	}

	_, err = store.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tenant")
	}
}

func TestPolicyRouterDecisionLog(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{id: "openai", available: true})

	router := NewPolicyRouter(registry)

	input := RoutingInput{
		TaskClass:   TaskClassCode,
		Sensitivity: SensitivityPublic,
	}

	_, _, _ = router.Route(context.Background(), input)
	_, _, _ = router.Route(context.Background(), input)

	log := router.GetDecisionLog()
	if len(log) != 2 {
		t.Errorf("expected 2 log entries, got %d", len(log))
	}

	router.ClearDecisionLog()
	log = router.GetDecisionLog()
	if len(log) != 0 {
		t.Error("expected empty log after clear")
	}
}

func TestPolicyRouterModelHint(t *testing.T) {
	registry := NewProviderRegistry()
	registry.Register(&mockProvider{
		id: "anthropic",
		capabilities: ModelCapabilities{
			SupportedModels: []ModelID{"claude-3-opus", "claude-3-sonnet"},
		},
		available: true,
	})
	registry.Register(&mockProvider{id: "openai", available: true})

	router := NewPolicyRouter(registry)

	input := RoutingInput{
		TaskClass:   TaskClassCode,
		Sensitivity: SensitivityPublic,
		ModelHint:   "claude-3-opus",
	}

	decision, _, err := router.Route(context.Background(), input)
	if err != nil {
		t.Fatalf("route failed: %v", err)
	}
	if decision.SelectedProvider != "anthropic" {
		t.Errorf("expected anthropic for claude hint, got %s", decision.SelectedProvider)
	}
	if decision.SelectedModel != "claude-3-opus" {
		t.Errorf("expected claude-3-opus, got %s", decision.SelectedModel)
	}
}

func TestSensitivityLevels(t *testing.T) {
	levels := []SensitivityLevel{
		SensitivityPublic,
		SensitivityInternal,
		SensitivityConfidential,
		SensitivityRestricted,
	}

	for _, level := range levels {
		if string(level) == "" {
			t.Errorf("unexpected empty sensitivity level")
		}
	}
}

func TestTaskClasses(t *testing.T) {
	classes := []TaskClass{
		TaskClassCode,
		TaskClassAnalysis,
		TaskClassCreative,
		TaskClassReasoning,
		TaskClassEmbedding,
	}

	for _, class := range classes {
		if string(class) == "" {
			t.Errorf("unexpected empty task class")
		}
	}
}

func TestPolicyRouterIsSimulationMode(t *testing.T) {
	registry := NewProviderRegistry()
	router := NewPolicyRouter(registry, WithSimulationMode(true))

	if !router.IsSimulationMode() {
		t.Error("expected simulation mode to be true")
	}

	router.SetSimulationMode(false)
	if router.IsSimulationMode() {
		t.Error("expected simulation mode to be false")
	}
}
