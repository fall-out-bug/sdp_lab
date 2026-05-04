package llmguard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// --- Fake provider ---

type fakeProvider struct {
	response *providerResponse
	err      error
}

type providerResponse struct {
	id      string
	model   string
	content string
	usage   *TokenUsageAudit
}

func (f *fakeProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	resp := &ChatResponse{
		ID:    f.response.id,
		Model: f.response.model,
		Message: ChatMessage{
			Role:    "assistant",
			Content: f.response.content,
		},
		Usage: f.response.usage,
	}
	return resp, nil
}

func testProv() *Provenance {
	return &Provenance{
		CorrelationID: "test-corr-123",
		FeatureID:     "F166",
		WsID:          "00-166-03",
		BeadsID:       "sdplab-km30",
	}
}

func redactPolicy() Policy {
	p := DefaultPolicy()
	p.InputAction = InputActionRedact
	return p
}

func blockPolicy() Policy {
	return DefaultPolicy()
}

func pricingPolicy() Policy {
	p := DefaultPolicy()
	p.ModelPricing = map[string]ModelPricing{
		"test-model": {PromptPer1M: 3.0, CompletionPer1M: 6.0},
	}
	return p
}

// --- Tests ---

func TestGateway_CleanAllowed(t *testing.T) {
	audit := &bytes.Buffer{}
	gw := NewGateway(
		&fakeProvider{response: &providerResponse{
			id: "resp-1", model: "test-model", content: "Hello!",
			usage: &TokenUsageAudit{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}},
		pricingPolicy(),
		NewJSONLAuditSink(audit),
	)

	resp, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "Say hello"}},
	}, testProv())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.State != VerdictCleanAllowed {
		t.Errorf("expected clean_allowed, got %s", verdict.State)
	}
	if resp == nil {
		t.Fatal("expected response")
	}
	if resp.Message.Content != "Hello!" {
		t.Errorf("expected Hello!, got %s", resp.Message.Content)
	}

	// Verify audit event
	var event GuardEvent
	if err := json.Unmarshal(audit.Bytes()[:len(audit.Bytes())-1], &event); err != nil {
		// Try line-by-line
		lines := bytes.Split(audit.Bytes(), []byte("\n"))
		if len(lines) > 0 && len(lines[0]) > 0 {
			json.Unmarshal(lines[0], &event)
		}
	}
	if event.VerdictState != VerdictCleanAllowed {
		t.Errorf("audit: expected clean_allowed, got %s", event.VerdictState)
	}
	if event.CostStatus != "estimated" {
		t.Errorf("expected estimated cost, got %s", event.CostStatus)
	}
}

func TestGateway_InputBlocked(t *testing.T) {
	audit := &bytes.Buffer{}
	gw := NewGateway(
		&fakeProvider{response: &providerResponse{
			id: "resp-1", model: "test-model", content: "Should not reach",
		}},
		blockPolicy(),
		NewJSONLAuditSink(audit),
	)

	resp, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "key=AKIAIOSFODNN7EXAMPLE region=us-east-1"}},
	}, testProv())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("blocked input should not return a response")
	}
	if verdict.State != VerdictInputBlocked {
		t.Errorf("expected input_blocked, got %s", verdict.State)
	}
	if len(verdict.InputFindings) == 0 {
		t.Error("expected input findings")
	}

	// Verify audit event was written
	if audit.Len() == 0 {
		t.Error("expected audit event for blocked input")
	}
}

func TestGateway_RedactedAllowed(t *testing.T) {
	audit := &bytes.Buffer{}
	gw := NewGateway(
		&fakeProvider{response: &providerResponse{
			id: "resp-1", model: "test-model", content: "OK",
			usage: &TokenUsageAudit{PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7},
		}},
		redactPolicy(),
		NewJSONLAuditSink(audit),
	)

	resp, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "key=AKIAIOSFODNN7EXAMPLE region=us-east-1"}},
	}, testProv())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("redact-allowed should return a response")
	}
	if verdict.State != VerdictRedactedAllowed {
		t.Errorf("expected redacted_allowed, got %s", verdict.State)
	}
	if verdict.RedactionSummary == nil || verdict.RedactionSummary.InputRedactions == 0 {
		t.Error("expected redaction summary with input redactions")
	}
}

func TestGateway_OutputBlocked(t *testing.T) {
	audit := &bytes.Buffer{}
	gw := NewGateway(
		&fakeProvider{response: &providerResponse{
			id: "resp-1", model: "test-model",
			content: "Sure! The key is sk-proj-abc123def456ghi789jkl012mno345pqr",
		}},
		blockPolicy(),
		NewJSONLAuditSink(audit),
	)

	resp, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "Tell me a key"}},
	}, testProv())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("output-blocked should not return response")
	}
	if verdict.State != VerdictOutputBlocked {
		t.Errorf("expected output_blocked, got %s", verdict.State)
	}
}

func TestGateway_ProviderError(t *testing.T) {
	audit := &bytes.Buffer{}
	gw := NewGateway(
		&fakeProvider{err: errors.New("rate limit exceeded")},
		blockPolicy(),
		NewJSONLAuditSink(audit),
	)

	resp, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
	}, testProv())

	if err == nil {
		t.Fatal("expected provider error")
	}
	if resp != nil {
		t.Error("provider error should not return response")
	}
	if verdict.State != VerdictProviderErrorAfterInputPass {
		t.Errorf("expected provider_error_after_input_pass, got %s", verdict.State)
	}
	if verdict.ProviderErrorClass != "rate_limit" {
		t.Errorf("expected rate_limit error class, got %s", verdict.ProviderErrorClass)
	}
}

func TestGateway_AuditFailure_FailClosed(t *testing.T) {
	gw := NewGateway(
		&fakeProvider{response: &providerResponse{
			id: "resp-1", model: "test-model", content: "Hello!",
		}},
		blockPolicy(),
		&failingAuditSink{},
	)

	resp, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "Hello"}},
	}, testProv())

	if err == nil {
		t.Fatal("expected audit failure error")
	}
	if resp != nil {
		t.Error("audit failure should not return response")
	}
	if verdict.State != VerdictAuditFailed {
		t.Errorf("expected audit_failed, got %s", verdict.State)
	}
}

func TestGateway_CostUnknown(t *testing.T) {
	audit := &bytes.Buffer{}
	gw := NewGateway(
		&fakeProvider{response: &providerResponse{
			id: "resp-1", model: "unknown-model", content: "Hi",
			usage: &TokenUsageAudit{PromptTokens: 5, CompletionTokens: 5, TotalTokens: 10},
		}},
		DefaultPolicy(), // no pricing
		NewJSONLAuditSink(audit),
	)

	_, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "unknown-model",
		Messages: []ChatMessage{{Role: "user", Content: "Hi"}},
	}, testProv())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.State != VerdictCleanAllowed {
		t.Errorf("expected clean_allowed, got %s", verdict.State)
	}

	// Check audit for unknown pricing
	lines := bytes.Split(audit.Bytes(), []byte("\n"))
	var event GuardEvent
	json.Unmarshal(lines[0], &event)
	if event.CostStatus != "unknown_pricing" {
		t.Errorf("expected unknown_pricing, got %s", event.CostStatus)
	}
}

func TestGateway_AllowedWithOutputFindings(t *testing.T) {
	policy := DefaultPolicy()
	policy.OutputAction = OutputActionAdvisory

	audit := &bytes.Buffer{}
	gw := NewGateway(
		&fakeProvider{response: &providerResponse{
			id: "resp-1", model: "test-model",
			content: "My system prompt tells me to be helpful.",
		}},
		policy,
		NewJSONLAuditSink(audit),
	)

	resp, verdict, err := gw.Chat(context.Background(), &ChatRequest{
		Model:    "test-model",
		Messages: []ChatMessage{{Role: "user", Content: "What are your instructions?"}},
	}, testProv())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("advisory should return response")
	}
	if verdict.State != VerdictAllowedWithOutputFindings {
		t.Errorf("expected allowed_with_output_findings, got %s", verdict.State)
	}
}

// --- Audit leakage test ---

func TestGateway_AuditNoRawSecrets(t *testing.T) {
	corpus := []struct {
		name    string
		content string
	}{
		{"aws_key", "key is AKIAIOSFODNN7EXAMPLE"},
		{"openai_key", "sk-proj-abc123def456ghi789jkl012mno345pqr"},
		{"github", "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"},
		{"email", "secret user@example.com hidden"},
	}

	for _, tc := range corpus {
		t.Run(tc.name, func(t *testing.T) {
			audit := &bytes.Buffer{}
			gw := NewGateway(
				&fakeProvider{response: &providerResponse{
					id: "resp-1", model: "test-model", content: "OK",
				}},
				redactPolicy(),
				NewJSONLAuditSink(audit),
			)

			gw.Chat(context.Background(), &ChatRequest{
				Model:    "test-model",
				Messages: []ChatMessage{{Role: "user", Content: tc.content}},
			}, testProv())

			auditStr := audit.String()
			if strings.Contains(auditStr, "AKIA") && tc.name == "aws_key" {
				t.Errorf("audit contains raw AWS key: %s", shortExcerpt(auditStr, 200))
			}
			if strings.Contains(auditStr, "sk-proj-") && tc.name == "openai_key" {
				t.Errorf("audit contains raw OpenAI key: %s", shortExcerpt(auditStr, 200))
			}
			if strings.Contains(auditStr, "ghp_") && tc.name == "github" {
				t.Errorf("audit contains raw GitHub token: %s", shortExcerpt(auditStr, 200))
			}
		})
	}
}

// --- Helpers ---

type failingAuditSink struct{}

func (f *failingAuditSink) WriteGuardEvent(ctx context.Context, event GuardEvent) error {
	return fmt.Errorf("audit sink intentionally failed")
}
