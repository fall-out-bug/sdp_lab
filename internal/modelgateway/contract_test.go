package modelgateway

import (
	"context"
	"testing"
	"time"
)

// Compile-time interface satisfaction checks.
// Note: RouterV1 captures ModelRouter only. PolicyRouter uses a different
// Route signature (context.Context, RoutingInput) and is out of v1 scope.
var _ ProviderV1 = (Provider)(nil)
var _ RouterV1 = (*ModelRouter)(nil)
var _ CredentialManagerV1 = (*CredentialManager)(nil)

// contractMockProvider implements ProviderV1 for contract tests.
type contractMockProvider struct{}

func (m *contractMockProvider) ID() ProviderID { return "test" }
func (m *contractMockProvider) Chat(_ context.Context, _ *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{}, nil
}
func (m *contractMockProvider) Capabilities() ModelCapabilities      { return ModelCapabilities{} }
func (m *contractMockProvider) IsAvailable(_ context.Context) bool   { return true }
func (m *contractMockProvider) ValidateRequest(_ *ChatRequest) error { return nil }

// contractMockRouter implements RouterV1 for contract tests.
type contractMockRouter struct{}

func (m *contractMockRouter) Route(_ *ChatRequest) (Provider, error) { return nil, nil }
func (m *contractMockRouter) Chat(_ context.Context, _ *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{}, nil
}
func (m *contractMockRouter) ChatWithFallback(_ context.Context, _ *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{}, nil
}

// contractMockCredManager implements CredentialManagerV1 for contract tests.
type contractMockCredManager struct{}

func (m *contractMockCredManager) GetCredential(_ context.Context, _ string, _ ProviderID) (*Credential, error) {
	return &Credential{}, nil
}
func (m *contractMockCredManager) CreateCredential(_ context.Context, _ string, _ ProviderID, _ string, _ ...CredentialOption) (*Credential, error) {
	return &Credential{}, nil
}
func (m *contractMockCredManager) RotateCredential(_ context.Context, _ string, _ ProviderID, _ string) (*Credential, error) {
	return &Credential{}, nil
}
func (m *contractMockCredManager) RevokeCredential(_ context.Context, _ string, _ ProviderID) error {
	return nil
}
func (m *contractMockCredManager) CheckExpiry(_ context.Context, _ string) ([]*Credential, error) {
	return nil, nil
}

func TestContractVersionDefined(t *testing.T) {
	if ContractVersion != "v1.0.0" {
		t.Errorf("ContractVersion = %s, want v1.0.0", ContractVersion)
	}
}

func TestProviderV1InterfaceMatch(t *testing.T) {
	var p ProviderV1 = &contractMockProvider{}
	if p.ID() != "test" {
		t.Errorf("ID() = %s, want test", p.ID())
	}
}

func TestRouterV1InterfaceMatch(t *testing.T) {
	var r RouterV1 = &contractMockRouter{}
	prov, err := r.Route(&ChatRequest{})
	if err != nil {
		t.Errorf("Route() error: %v", err)
	}
	_ = prov
}

func TestCredentialManagerV1InterfaceMatch(t *testing.T) {
	var cm CredentialManagerV1 = &contractMockCredManager{}
	_, _ = cm.GetCredential(context.Background(), "t1", "p1")
	_ = cm.RevokeCredential(context.Background(), "t1", "p1")
}

func TestCostEnvelopeFields(t *testing.T) {
	env := CostEnvelope{
		MaxTokensPerRequest: 4096,
		MaxTokensPerDay:     100000,
		CostPerToken:        0.00003,
	}
	if env.MaxTokensPerRequest != 4096 {
		t.Errorf("MaxTokensPerRequest = %d, want 4096", env.MaxTokensPerRequest)
	}
	if env.MaxTokensPerDay != 100000 {
		t.Errorf("MaxTokensPerDay = %d, want 100000", env.MaxTokensPerDay)
	}
	if env.CostPerToken != 0.00003 {
		t.Errorf("CostPerToken = %f, want 0.00003", env.CostPerToken)
	}
}

func TestFallbackContractDocumentsBehavior(t *testing.T) {
	retryableErrors := []ErrorType{ErrorTypeRateLimit, ErrorTypeTimeout}
	nonRetryableErrors := []ErrorType{ErrorTypeAuth, ErrorTypeInvalidInput, ErrorTypeModelNotAvailable}
	_, _ = retryableErrors, nonRetryableErrors
}

func TestAllowlistContractDocumentsBehavior(t *testing.T) {
	defaultAllowList := []ProviderID{}
	restrictiveList := []ProviderID{"anthropic", "openai"}
	_, _ = defaultAllowList, restrictiveList
}

func TestChatRequestAndResponseTypes(t *testing.T) {
	req := ChatRequest{
		Model:       "claude-3-opus",
		Messages:    []Message{{Role: RoleUser, Content: "test"}},
		Temperature: 0.7,
		MaxTokens:   1024,
	}
	resp := ChatResponse{
		ID:      "resp-123",
		Model:   "claude-3-opus",
		Created: time.Now(),
		Message: Message{Role: RoleAssistant, Content: "response"},
		Usage: &TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
		},
		FinishReason: "stop",
	}
	if req.Model != "claude-3-opus" {
		t.Errorf("Model = %s, want claude-3-opus", req.Model)
	}
	if resp.Usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", resp.Usage.TotalTokens)
	}
}
