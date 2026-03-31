package kernel

import "time"

type ProviderID string

type ModelID string

type ModelCapabilities struct {
	Vision          bool      `json:"vision"`
	FunctionCall    bool      `json:"function_call"`
	Streaming       bool      `json:"streaming"`
	MaxContext      int       `json:"max_context"`
	SupportedModels []ModelID `json:"supported_models"`
}

type ProviderMeta struct {
	ProviderID   ProviderID        `json:"provider_id"`
	ModelName    string            `json:"model_name"`
	Capabilities ModelCapabilities `json:"capabilities"`
	Latency      time.Duration     `json:"latency"`
}

type TaskClass string

const (
	TaskClassCode      TaskClass = "code"
	TaskClassAnalysis  TaskClass = "analysis"
	TaskClassCreative  TaskClass = "creative"
	TaskClassReasoning TaskClass = "reasoning"
	TaskClassEmbedding TaskClass = "embedding"
)

type SensitivityLevel string

const (
	SensitivityPublic       SensitivityLevel = "public"
	SensitivityInternal     SensitivityLevel = "internal"
	SensitivityConfidential SensitivityLevel = "confidential"
	SensitivityRestricted   SensitivityLevel = "restricted"
)

type RoutingInput struct {
	TaskClass         TaskClass        `json:"task_class"`
	Sensitivity       SensitivityLevel `json:"sensitivity"`
	MaxLatencyMs      int              `json:"max_latency_ms,omitempty"`
	MaxCostCents      int              `json:"max_cost_cents,omitempty"`
	RequiresVision    bool             `json:"requires_vision"`
	RequiresFunctions bool             `json:"requires_functions"`
	TenantID          string           `json:"tenant_id"`
	ModelHint         ModelID          `json:"model_hint,omitempty"`
}

type RoutingConstraints struct {
	AllowedProviders     []ProviderID `json:"allowed_providers"`
	AllowedModels        []ModelID    `json:"allowed_models"`
	MaxCostPerToken      float64      `json:"max_cost_per_token"`
	RequireDataResidency string       `json:"require_data_residency,omitempty"`
}

type RoutingDecision struct {
	SelectedProvider ProviderID         `json:"selected_provider"`
	SelectedModel    ModelID            `json:"selected_model"`
	FallbackChain    []ProviderID       `json:"fallback_chain"`
	DecisionReason   string             `json:"decision_reason"`
	PolicyID         string             `json:"policy_id"`
	EvaluatedAt      time.Time          `json:"evaluated_at"`
	InputHash        string             `json:"input_hash"`
	Constraints      RoutingConstraints `json:"constraints"`
}
