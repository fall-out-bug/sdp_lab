package modelgateway

import (
	"context"
	"time"
)

type ProviderID string

type ModelID string

type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

type ChatRequest struct {
	Model       ModelID                `json:"model"`
	Messages    []Message              `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ChatResponse struct {
	ID           string        `json:"id"`
	Model        ModelID       `json:"model"`
	Created      time.Time     `json:"created"`
	Message      Message       `json:"message"`
	Usage        *TokenUsage   `json:"usage,omitempty"`
	FinishReason string        `json:"finish_reason,omitempty"`
	ProviderMeta *ProviderMeta `json:"provider_meta,omitempty"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ProviderMeta struct {
	ProviderID   ProviderID        `json:"provider_id"`
	ModelName    string            `json:"model_name"`
	Capabilities ModelCapabilities `json:"capabilities"`
	Latency      time.Duration     `json:"latency"`
}

type ModelCapabilities struct {
	Vision          bool      `json:"vision"`
	FunctionCall    bool      `json:"function_call"`
	Streaming       bool      `json:"streaming"`
	MaxContext      int       `json:"max_context"`
	SupportedModels []ModelID `json:"supported_models"`
}

type ProviderError struct {
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	ProviderID ProviderID `json:"provider_id"`
	Retryable  bool       `json:"retryable"`
	Type       ErrorType  `json:"type"`
}

type ErrorType string

const (
	ErrorTypeRateLimit         ErrorType = "rate_limit"
	ErrorTypeAuth              ErrorType = "auth"
	ErrorTypeInvalidInput      ErrorType = "invalid_input"
	ErrorTypeModelNotAvailable ErrorType = "model_not_available"
	ErrorTypeTimeout           ErrorType = "timeout"
	ErrorTypeInternal          ErrorType = "internal"
)

func (e *ProviderError) Error() string {
	return e.Message
}

func (e *ProviderError) IsRetryable() bool {
	return e.Retryable
}

type ProviderConfig struct {
	ID           ProviderID             `json:"id"`
	APIKey       string                 `json:"api_key,omitempty"`
	BaseURL      string                 `json:"base_url,omitempty"`
	DefaultModel ModelID                `json:"default_model"`
	Timeout      time.Duration          `json:"timeout"`
	MaxRetries   int                    `json:"max_retries"`
	RateLimit    *RateLimitConfig       `json:"rate_limit,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
}

type Provider interface {
	ID() ProviderID
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	Capabilities() ModelCapabilities
	IsAvailable(ctx context.Context) bool
	ValidateRequest(req *ChatRequest) error
}

type ProviderFactory func(config ProviderConfig) (Provider, error)

type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[ProviderID]Provider
	factories map[ProviderID]ProviderFactory
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[ProviderID]Provider),
		factories: make(map[ProviderID]ProviderFactory),
	}
}

func (r *ProviderRegistry) RegisterFactory(id ProviderID, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[id] = factory
}

func (r *ProviderRegistry) CreateProvider(config ProviderConfig) (Provider, error) {
	r.mu.RLock()
	factory, exists := r.factories[config.ID]
	r.mu.RUnlock()
	if !exists {
		return nil, &ProviderError{
			Code:       "PROVIDER_NOT_FOUND",
			Message:    "provider factory not registered",
			ProviderID: config.ID,
			Type:       ErrorTypeInternal,
		}
	}
	return factory(config)
}

func (r *ProviderRegistry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
}

func (r *ProviderRegistry) Get(id ProviderID) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

func (r *ProviderRegistry) List() []ProviderID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var ids []ProviderID
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

func (r *ProviderRegistry) Capabilities(id ProviderID) (ModelCapabilities, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	if !ok {
		return ModelCapabilities{}, false
	}
	return p.Capabilities(), true
}

type RouterConfig struct {
	DefaultProvider ProviderID
	FallbackOrder   []ProviderID
	RoutingRules    []RoutingRule
}

type RoutingRule struct {
	Match  RoutingMatch `json:"match"`
	Target ProviderID   `json:"target"`
}

type RoutingMatch struct {
	ModelPrefix string `json:"model_prefix,omitempty"`
	HasVision   *bool  `json:"has_vision,omitempty"`
}

type ModelRouter struct {
	registry *ProviderRegistry
	config   RouterConfig
}

func NewModelRouter(registry *ProviderRegistry, config RouterConfig) *ModelRouter {
	return &ModelRouter{
		registry: registry,
		config:   config,
	}
}

func (r *ModelRouter) Route(req *ChatRequest) (Provider, error) {
	for _, rule := range r.config.RoutingRules {
		if r.matchesRule(req, rule.Match) {
			p, ok := r.registry.Get(rule.Target)
			if ok {
				return p, nil
			}
		}
	}

	if r.config.DefaultProvider != "" {
		p, ok := r.registry.Get(r.config.DefaultProvider)
		if ok {
			return p, nil
		}
	}

	return nil, &ProviderError{
		Code:    "NO_PROVIDER_AVAILABLE",
		Message: "no provider available for request",
		Type:    ErrorTypeInternal,
	}
}

func (r *ModelRouter) matchesRule(req *ChatRequest, match RoutingMatch) bool {
	if match.ModelPrefix != "" && !hasPrefix(string(req.Model), match.ModelPrefix) {
		return false
	}
	return true
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func (r *ModelRouter) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	provider, err := r.Route(req)
	if err != nil {
		return nil, err
	}
	return provider.Chat(ctx, req)
}

func (r *ModelRouter) ChatWithFallback(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	var lastErr error

	primary, err := r.Route(req)
	if err == nil {
		resp, err := primary.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !isRetryableError(err) {
			return nil, err
		}
		lastErr = err
	}

	for _, fallbackID := range r.config.FallbackOrder {
		p, ok := r.registry.Get(fallbackID)
		if !ok {
			continue
		}

		resp, err := p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !isRetryableError(err) {
			return nil, err
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, &ProviderError{
		Code:    "ALL_PROVIDERS_FAILED",
		Message: "all providers failed",
		Type:    ErrorTypeInternal,
	}
}

func isRetryableError(err error) bool {
	if pErr, ok := err.(*ProviderError); ok {
		return pErr.IsRetryable()
	}
	return false
}

type ProviderHealth struct {
	ProviderID ProviderID    `json:"provider_id"`
	Available  bool          `json:"available"`
	LastCheck  time.Time     `json:"last_check"`
	ErrorCount int           `json:"error_count"`
	AvgLatency time.Duration `json:"avg_latency"`
}

func (r *ProviderRegistry) HealthCheck(ctx context.Context) []ProviderHealth {
	var health []ProviderHealth
	for id, p := range r.providers {
		health = append(health, ProviderHealth{
			ProviderID: id,
			Available:  p.IsAvailable(ctx),
			LastCheck:  time.Now(),
		})
	}
	return health
}
