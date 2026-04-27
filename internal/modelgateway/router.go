package modelgateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

type TaskClass = kernel.TaskClass

const (
	TaskClassCode      = kernel.TaskClassCode
	TaskClassAnalysis  = kernel.TaskClassAnalysis
	TaskClassCreative  = kernel.TaskClassCreative
	TaskClassReasoning = kernel.TaskClassReasoning
	TaskClassEmbedding = kernel.TaskClassEmbedding
)

type SensitivityLevel = kernel.SensitivityLevel

const (
	SensitivityPublic       = kernel.SensitivityPublic
	SensitivityInternal     = kernel.SensitivityInternal
	SensitivityConfidential = kernel.SensitivityConfidential
	SensitivityRestricted   = kernel.SensitivityRestricted
)

type RoutingInput = kernel.RoutingInput

type RoutingDecision = kernel.RoutingDecision

type RoutingConstraints = kernel.RoutingConstraints

type RoutingEvidence struct {
	Decision  RoutingDecision `json:"decision"`
	Input     RoutingInput    `json:"input"`
	Timestamp time.Time       `json:"timestamp"`
}

func (e *RoutingEvidence) ToJSON() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

type PolicyEvaluator interface {
	Evaluate(ctx context.Context, input RoutingInput) (*RoutingDecision, error)
}

type TenantConfig struct {
	TenantID        string             `json:"tenant_id"`
	DefaultProvider ProviderID         `json:"default_provider"`
	FallbackChain   []ProviderID       `json:"fallback_chain"`
	Constraints     RoutingConstraints `json:"constraints"`
	RateLimits      *RateLimitConfig   `json:"rate_limits,omitempty"`
}

type TenantConfigStore interface {
	Get(ctx context.Context, tenantID string) (*TenantConfig, error)
}

type PolicyRouter struct {
	mu             sync.RWMutex
	registry       *ProviderRegistry
	policy         PolicyEvaluator
	tenantStore    TenantConfigStore
	simulationMode bool
	decisionLog    []RoutingEvidence
}

type PolicyRouterOption func(*PolicyRouter)

func WithSimulationMode(sim bool) PolicyRouterOption {
	return func(r *PolicyRouter) {
		r.simulationMode = sim
	}
}

func WithTenantStore(store TenantConfigStore) PolicyRouterOption {
	return func(r *PolicyRouter) {
		r.tenantStore = store
	}
}

func WithPolicyEvaluator(evaluator PolicyEvaluator) PolicyRouterOption {
	return func(r *PolicyRouter) {
		r.policy = evaluator
	}
}

func NewPolicyRouter(registry *ProviderRegistry, opts ...PolicyRouterOption) *PolicyRouter {
	r := &PolicyRouter{
		registry:    registry,
		decisionLog: make([]RoutingEvidence, 0),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *PolicyRouter) Route(ctx context.Context, input RoutingInput) (*RoutingDecision, *RoutingEvidence, error) {
	// Step 1: Snapshot simulationMode under RLock
	r.mu.RLock()
	simulationMode := r.simulationMode
	r.mu.RUnlock()

	// Step 2: Release lock - do policy evaluation and tenant lookup outside lock
	var decision *RoutingDecision
	var err error

	if r.policy != nil {
		decision, err = r.policy.Evaluate(ctx, input)
		if err != nil {
			return nil, nil, fmt.Errorf("policy evaluation failed: %w", err)
		}
	} else {
		decision = r.defaultRoute(input)
	}

	if r.tenantStore != nil && input.TenantID != "" {
		tenant, err := r.tenantStore.Get(ctx, input.TenantID)
		if err == nil && tenant != nil {
			decision = r.applyTenantConfig(decision, tenant)
		}
	}

	evidence := &RoutingEvidence{
		Decision:  *decision,
		Input:     input,
		Timestamp: time.Now(),
	}

	// Step 3: Acquire Lock only to append to decisionLog
	r.mu.Lock()
	if !simulationMode {
		r.decisionLog = append(r.decisionLog, *evidence)
	}
	r.mu.Unlock()

	// Step 4: Return result
	return decision, evidence, nil
}

func (r *PolicyRouter) defaultRoute(input RoutingInput) *RoutingDecision {
	decision := &RoutingDecision{
		EvaluatedAt: time.Now(),
		InputHash:   hashInput(input),
		Constraints: RoutingConstraints{},
	}

	if input.RequiresVision {
		for _, id := range r.registry.List() {
			if caps, ok := r.registry.Capabilities(id); ok && caps.Vision {
				decision.SelectedProvider = id
				decision.DecisionReason = "selected for vision capability"
				return decision
			}
		}
	}

	if input.Sensitivity == SensitivityRestricted {
		decision.SelectedProvider = "selfhosted"
		decision.DecisionReason = "selected for data residency (restricted sensitivity)"
		decision.Constraints.RequireDataResidency = "on-premise"
		return decision
	}

	if hint := string(input.ModelHint); hint != "" {
		for _, id := range r.registry.List() {
			if caps, ok := r.registry.Capabilities(id); ok {
				for _, m := range caps.SupportedModels {
					if string(m) == hint {
						decision.SelectedProvider = id
						decision.SelectedModel = input.ModelHint
						decision.DecisionReason = "selected based on model hint"
						return decision
					}
				}
			}
		}
	}

	decision.SelectedProvider = "openai"
	decision.DecisionReason = "default provider"
	return decision
}

func (r *PolicyRouter) applyTenantConfig(decision *RoutingDecision, tenant *TenantConfig) *RoutingDecision {
	if len(tenant.Constraints.AllowedProviders) > 0 {
		allowed := false
		for _, p := range tenant.Constraints.AllowedProviders {
			if p == decision.SelectedProvider {
				allowed = true
				break
			}
		}
		if !allowed && len(tenant.Constraints.AllowedProviders) > 0 {
			decision.SelectedProvider = tenant.Constraints.AllowedProviders[0]
			decision.DecisionReason += " (overridden by tenant constraints)"
		}
	}

	if len(tenant.FallbackChain) > 0 {
		decision.FallbackChain = tenant.FallbackChain
	} else {
		decision.FallbackChain = []ProviderID{decision.SelectedProvider}
	}

	decision.Constraints = tenant.Constraints
	return decision
}

func (r *PolicyRouter) Chat(ctx context.Context, input RoutingInput, req *ChatRequest) (*ChatResponse, *RoutingEvidence, error) {
	decision, evidence, err := r.Route(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	provider, ok := r.registry.Get(decision.SelectedProvider)
	if !ok {
		return nil, evidence, &ProviderError{
			Code:       "PROVIDER_NOT_FOUND",
			Message:    fmt.Sprintf("provider %s not found", decision.SelectedProvider),
			Type:       ErrorTypeInternal,
			ProviderID: decision.SelectedProvider,
		}
	}

	if decision.SelectedModel != "" {
		req.Model = decision.SelectedModel
	}

	resp, err := provider.Chat(ctx, req)
	if err == nil {
		return resp, evidence, nil
	}

	if !isRetryableError(err) {
		return nil, evidence, err
	}

	for _, fallbackID := range decision.FallbackChain {
		if fallbackID == decision.SelectedProvider {
			continue
		}
		fallback, ok := r.registry.Get(fallbackID)
		if !ok {
			continue
		}
		resp, fallbackErr := fallback.Chat(ctx, req)
		if fallbackErr == nil {
			return resp, evidence, nil
		}
	}

	return nil, evidence, err
}

func (r *PolicyRouter) GetDecisionLog() []RoutingEvidence {
	r.mu.RLock()
	defer r.mu.RUnlock()

	log := make([]RoutingEvidence, len(r.decisionLog))
	copy(log, r.decisionLog)
	return log
}

func (r *PolicyRouter) ClearDecisionLog() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decisionLog = make([]RoutingEvidence, 0)
}

func (r *PolicyRouter) SetSimulationMode(sim bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.simulationMode = sim
}

func (r *PolicyRouter) IsSimulationMode() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.simulationMode
}

type SimplePolicyEvaluator struct {
	rules []RoutingRule
}

func NewSimplePolicyEvaluator(rules []RoutingRule) *SimplePolicyEvaluator {
	return &SimplePolicyEvaluator{rules: rules}
}

func (e *SimplePolicyEvaluator) Evaluate(ctx context.Context, input RoutingInput) (*RoutingDecision, error) {
	decision := &RoutingDecision{
		EvaluatedAt: time.Now(),
		InputHash:   hashInput(input),
		PolicyID:    "simple-policy",
	}

	for _, rule := range e.rules {
		if e.matchesRule(input, rule) {
			decision.SelectedProvider = rule.Target
			decision.DecisionReason = fmt.Sprintf("matched rule for %s", rule.Match.ModelPrefix)
			return decision, nil
		}
	}

	decision.SelectedProvider = "openai"
	decision.DecisionReason = "no rule matched, using default"
	return decision, nil
}

func (e *SimplePolicyEvaluator) matchesRule(input RoutingInput, rule RoutingRule) bool {
	if rule.Match.ModelPrefix != "" && input.ModelHint != "" {
		if !hasPrefix(string(input.ModelHint), rule.Match.ModelPrefix) {
			return false
		}
	}
	return true
}

type InMemoryTenantStore struct {
	mu      sync.RWMutex
	tenants map[string]*TenantConfig
}

func NewInMemoryTenantStore() *InMemoryTenantStore {
	return &InMemoryTenantStore{
		tenants: make(map[string]*TenantConfig),
	}
}

func (s *InMemoryTenantStore) Get(ctx context.Context, tenantID string) (*TenantConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	config, ok := s.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}
	return config, nil
}

func (s *InMemoryTenantStore) Set(config *TenantConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[config.TenantID] = config
}

func hashInput(input RoutingInput) string {
	// Hash the actual input for deterministic routing decisions
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%d|%v|%v", input.TaskClass, input.Sensitivity, input.MaxLatencyMs, input.RequiresVision, input.RequiresFunctions)
	return hex.EncodeToString(h.Sum(nil))[:16]
}
