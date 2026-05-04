package llmguard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/modelgateway"
	"github.com/google/uuid"
)

// GuardEvent is the JSONL audit record for a guarded LLM call.
type GuardEvent struct {
	EventID              string            `json:"event_id"`
	CorrelationID        string            `json:"correlation_id,omitempty"`
	FeatureID            string            `json:"feature_id,omitempty"`
	WsID                 string            `json:"ws_id,omitempty"`
	BeadsID              string            `json:"beads_id,omitempty"`
	SessionID            string            `json:"session_id,omitempty"`
	EvidenceRef          string            `json:"evidence_ref,omitempty"`
	Timestamp            time.Time         `json:"timestamp"`
	ProviderID           string            `json:"provider_id,omitempty"`
	Model                string            `json:"model,omitempty"`
	VerdictState         VerdictState      `json:"verdict_state"`
	InputFindings        []Finding         `json:"input_findings,omitempty"`
	OutputFindings       []Finding         `json:"output_findings,omitempty"`
	RedactionSummary     *RedactionSummary `json:"redaction_summary,omitempty"`
	TokenUsage           *TokenUsageAudit  `json:"token_usage,omitempty"`
	CostStatus           string            `json:"cost_status,omitempty"`
	EstimatedCostUSD     *float64          `json:"estimated_cost_usd,omitempty"`
	ProviderErrorClass   string            `json:"provider_error_class,omitempty"`
	ProviderErrorExcerpt string            `json:"provider_error_excerpt,omitempty"`
	Harness         string `json:"harness,omitempty"`
	EndpointSurface string `json:"endpoint_surface,omitempty"`
	StreamRequested bool   `json:"stream_requested"`
	StreamReturned  bool   `json:"stream_returned"`
	UpstreamCalled  bool   `json:"upstream_called"`
}

type ChatRequest = modelgateway.ChatRequest
type ChatResponse = modelgateway.ChatResponse
type ChatMessage = modelgateway.Message
type TokenUsageAudit = modelgateway.TokenUsage

// RedactionSummary summarizes what was redacted.
type RedactionSummary struct {
	InputRedactions  int      `json:"input_redactions"`
	OutputRedactions int      `json:"output_redactions"`
	Types            []string `json:"types,omitempty"`
}

// AuditSink writes redacted guard events.
type AuditSink interface {
	WriteGuardEvent(ctx context.Context, event GuardEvent) error
}

// JSONLAuditSink writes guard events as JSONL to a writer.
type JSONLAuditSink struct {
	w io.Writer
}

// NewJSONLAuditSink creates a JSONL audit sink writing to w.
func NewJSONLAuditSink(w io.Writer) *JSONLAuditSink {
	return &JSONLAuditSink{w: w}
}

// WriteGuardEvent writes a single JSONL event.
func (s *JSONLAuditSink) WriteGuardEvent(ctx context.Context, event GuardEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}
	_, err = s.w.Write(data)
	if err != nil {
		return fmt.Errorf("audit write: %w", err)
	}
	_, err = s.w.Write([]byte("\n"))
	return err
}

// FileAuditSink writes JSONL guard events to a file.
type FileAuditSink struct {
	path string
}

// NewFileAuditSink creates a file-based audit sink.
func NewFileAuditSink(path string) *FileAuditSink {
	return &FileAuditSink{path: path}
}

// WriteGuardEvent appends a JSONL event to the file.
func (s *FileAuditSink) WriteGuardEvent(ctx context.Context, event GuardEvent) error {
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("audit file open: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}
	_, err = f.Write(data)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("\n"))
	return err
}

// CostEstimator computes estimated cost from token usage and model pricing.
type CostEstimator struct {
	pricing map[string]ModelPricing
}

// NewCostEstimator creates a CostEstimator from a pricing map.
func NewCostEstimator(pricing map[string]ModelPricing) *CostEstimator {
	return &CostEstimator{pricing: pricing}
}

// CostStatus is the result of cost estimation.
type CostResult struct {
	Status string
	Cost   float64
}

// Estimate computes the estimated cost for the given model and token usage.
// Returns "unknown_pricing" when model pricing is absent; never reports 0 as actual cost.
func (e *CostEstimator) Estimate(model string, usage *TokenUsageAudit) CostResult {
	if usage == nil {
		return CostResult{Status: "unknown_pricing"}
	}

	p, ok := e.pricing[model]
	if !ok {
		return CostResult{Status: "unknown_pricing"}
	}

	promptCost := float64(usage.PromptTokens) / 1_000_000 * p.PromptPer1M
	completionCost := float64(usage.CompletionTokens) / 1_000_000 * p.CompletionPer1M
	total := promptCost + completionCost

	return CostResult{
		Status: "estimated",
		Cost:   total,
	}
}

// Provenance carries SDP call-site metadata for audit records.
type Provenance struct {
	CorrelationID   string
	FeatureID       string
	WsID            string
	BeadsID         string
	SessionID       string
	EvidenceRef     string
	Harness         string
	EndpointSurface string
	StreamRequested bool
}

// Verdict is the result of a guarded Chat call.
type Verdict struct {
	State              VerdictState
	EventID            string
	InputFindings      []Finding
	OutputFindings     []Finding
	RedactionSummary   *RedactionSummary
	ProviderErrorClass string
	ProviderErrorText  string
}

// Gateway wraps a provider with input/output guard.
type Gateway struct {
	provider Provider
	policy   Policy
	scanner  *Scanner
	redactor *Redactor
	audit    AuditSink
	cost     *CostEstimator
}

// Provider is the minimal interface needed for wrapping.
// Matches modelgateway.Provider.Chat contract.
type Provider interface {
	Chat(ctx context.Context, req *modelgateway.ChatRequest) (*modelgateway.ChatResponse, error)
}

// NewGateway creates a guarded gateway wrapping a provider.
// Panics if provider or auditSink is nil — fail fast at construction time.
func NewGateway(provider Provider, policy Policy, auditSink AuditSink) *Gateway {
	if provider == nil {
		panic("llmguard: provider must not be nil")
	}
	if auditSink == nil {
		panic("llmguard: auditSink must not be nil")
	}
	return &Gateway{
		provider: provider,
		policy:   policy,
		scanner:  NewScannerFromPolicy(policy),
		redactor: NewRedactor(false), // untyped for provider-facing
		audit:    auditSink,
		cost:     NewCostEstimator(policy.ModelPricing),
	}
}

// Chat executes a guarded chat completion.
func (g *Gateway) Chat(ctx context.Context, req *ChatRequest, prov *Provenance) (*ChatResponse, *Verdict, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("llmguard: request must not be nil")
	}
	if prov == nil {
		prov = &Provenance{}
	}
	eventID := uuid.New().String()
	now := time.Now()

	// Build the audit event skeleton
	event := GuardEvent{
		EventID:         eventID,
		CorrelationID:   prov.CorrelationID,
		FeatureID:       prov.FeatureID,
		WsID:            prov.WsID,
		BeadsID:         prov.BeadsID,
		SessionID:       prov.SessionID,
		EvidenceRef:     prov.EvidenceRef,
		Timestamp:       now,
		Model:           string(req.Model),
		Harness:         prov.Harness,
		EndpointSurface: prov.EndpointSurface,
		StreamRequested: prov.StreamRequested,
	}

	// Scan input
	inputText := concatMessages(req.Messages)
	inputResult := g.scanner.Scan(inputText)

	if inputResult.BudgetExceeded {
		verdict := &Verdict{
			State:         VerdictScanBudgetExceeded,
			EventID:       eventID,
			InputFindings: inputResult.Findings,
		}
		if g.policy.StrictBudgetMode {
			event.VerdictState = VerdictScanBudgetExceeded
			event.UpstreamCalled = false
			if err := g.writeAudit(ctx, event); err != nil {
				return nil, &Verdict{State: VerdictAuditFailed, EventID: eventID}, fmt.Errorf("audit write failed: %w", err)
			}
			return nil, verdict, nil
		}
		// Non-strict: advisory, continue
	}

	// Check input findings
	inputSecrets := filterSecretFindings(inputResult.Findings)
	if len(inputSecrets) > 0 {
		switch g.policy.InputAction {
		case InputActionBlock:
			event.VerdictState = VerdictInputBlocked
			event.InputFindings = redactFindings(inputSecrets)
			event.UpstreamCalled = false
			if err := g.writeAudit(ctx, event); err != nil {
				return nil, &Verdict{State: VerdictAuditFailed, EventID: eventID}, fmt.Errorf("audit write failed: %w", err)
			}
			return nil, &Verdict{
				State:         VerdictInputBlocked,
				EventID:       eventID,
				InputFindings: inputSecrets,
			}, nil

		case InputActionRedact:
			// Redact input and continue
			redactedText := g.redactor.Redact(inputText, inputSecrets)
			req = redactRequest(req, redactedText)
			event.RedactionSummary = &RedactionSummary{
				InputRedactions: len(inputSecrets),
				Types:           findingTypes(inputSecrets),
			}
		}
	}

	// Call provider
	resp, err := g.provider.Chat(ctx, req)
	if err != nil {
		// Provider error — scan and redact error message
		errClass := classifyProviderError(err)
		errText := err.Error()

		// Scan provider error for secrets
		errScan := g.scanner.Scan(errText)
		redactedErrText := errText
		if len(errScan.Findings) > 0 {
			redactedErrText = RedactWithUntyped(errText, errScan.Findings)
		}
		errExcerpt := shortExcerpt(redactedErrText, 200)

		event.VerdictState = VerdictProviderErrorAfterInputPass
		event.ProviderErrorClass = errClass
		event.ProviderErrorExcerpt = errExcerpt
		event.InputFindings = redactFindings(inputResult.Findings)
		event.UpstreamCalled = true

		if err := g.writeAudit(ctx, event); err != nil {
			return nil, &Verdict{State: VerdictAuditFailed, EventID: eventID}, fmt.Errorf("audit write failed: %w", err)
		}
		return nil, &Verdict{
			State:              VerdictProviderErrorAfterInputPass,
			EventID:            eventID,
			ProviderErrorClass: errClass,
			ProviderErrorText:  errExcerpt,
		}, err
	}

	// Scan output
	outputText := resp.Message.Content
	outputResult := g.scanner.ScanOutput(outputText)

	// Check output findings
	outputSuspicious := filterSuspiciousOutputFindings(outputResult.Findings)

	if len(outputResult.Findings) > 0 && g.policy.OutputAction == OutputActionBlock {
		event.VerdictState = VerdictOutputBlocked
		event.OutputFindings = redactFindings(outputResult.Findings)
		event.InputFindings = redactFindings(inputResult.Findings)
		event.UpstreamCalled = true
		if event.RedactionSummary == nil {
			event.RedactionSummary = &RedactionSummary{}
		}
		if err := g.writeAudit(ctx, event); err != nil {
			return nil, &Verdict{State: VerdictAuditFailed, EventID: eventID}, fmt.Errorf("audit write failed: %w", err)
		}
		return nil, &Verdict{
			State:          VerdictOutputBlocked,
			EventID:        eventID,
			InputFindings:  inputResult.Findings,
			OutputFindings: outputResult.Findings,
		}, nil
	}

	// Determine final verdict
	state := VerdictCleanAllowed
	if len(inputSecrets) > 0 {
		state = VerdictRedactedAllowed
	}
	if len(outputSuspicious) > 0 {
		state = VerdictAllowedWithOutputFindings
	}

	// Cost estimation
	var usage *TokenUsageAudit
	if resp.Usage != nil {
		usage = &TokenUsageAudit{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	costResult := g.cost.Estimate(string(req.Model), usage)

	// Build final event
	event.VerdictState = state
	event.InputFindings = redactFindings(inputResult.Findings)
	event.OutputFindings = redactFindings(outputResult.Findings)
	event.TokenUsage = usage
	event.CostStatus = costResult.Status
	event.UpstreamCalled = true
	if costResult.Status == "estimated" {
		cost := costResult.Cost
		event.EstimatedCostUSD = &cost
	}

	// Write audit — fail-closed
	if err := g.audit.WriteGuardEvent(ctx, event); err != nil {
		return nil, &Verdict{
			State:   VerdictAuditFailed,
			EventID: eventID,
		}, fmt.Errorf("audit write failed: %w", err)
	}

	// Build response
	return resp, &Verdict{
		State:            state,
		EventID:          eventID,
		InputFindings:    inputResult.Findings,
		OutputFindings:   outputResult.Findings,
		RedactionSummary: event.RedactionSummary,
	}, nil
}

// writeAudit writes to the audit sink. Returns error if audit fails,
// so callers can return VerdictAuditFailed for fail-closed behavior.
func (g *Gateway) writeAudit(ctx context.Context, event GuardEvent) error {
	return g.audit.WriteGuardEvent(ctx, event)
}

// --- helpers ---

func concatMessages(msgs []ChatMessage) string {
	var b []byte
	for _, m := range msgs {
		b = append(b, []byte(m.Content)...)
		b = append(b, ' ')
	}
	return string(b)
}

func filterSecretFindings(findings []Finding) []Finding {
	var result []Finding
	for _, f := range findings {
		switch f.Type {
		case FindingOpenAIKey, FindingGitHubToken, FindingAWSKey, FindingBearerToken,
			FindingBase64Key, FindingSplitKey, FindingCard, FindingEmail, FindingPhone:
			result = append(result, f)
		}
	}
	return result
}

func filterSuspiciousOutputFindings(findings []Finding) []Finding {
	var result []Finding
	for _, f := range findings {
		switch f.Type {
		case FindingPromptDisclosure, FindingSuspiciousURL, FindingShellCommand,
			FindingGeneratedSecret:
			result = append(result, f)
		}
	}
	return result
}

func redactFindings(findings []Finding) []Finding {
	// Redact findings for audit — clear span info, keep type/severity
	result := make([]Finding, len(findings))
	for i, f := range findings {
		result[i] = Finding{
			Type:     f.Type,
			Severity: f.Severity,
			ScanMode: f.ScanMode,
			Redacted: RedactedPlaceholder(f.Type),
		}
	}
	return result
}

func redactRequest(req *ChatRequest, redactedText string) *ChatRequest {
	// Rebuild messages with single redacted content
	newReq := *req
	newReq.Messages = []ChatMessage{
		{Role: modelgateway.RoleUser, Content: redactedText},
	}
	return &newReq
}

func findingTypes(findings []Finding) []string {
	seen := make(map[string]bool)
	var types []string
	for _, f := range findings {
		t := string(f.Type)
		if !seen[t] {
			seen[t] = true
			types = append(types, t)
		}
	}
	return types
}

func classifyProviderError(err error) string {
	msg := err.Error()
	switch {
	case contains(msg, "rate_limit", "rate limit", "too many requests"):
		return "rate_limit"
	case contains(msg, "auth", "unauthorized", "forbidden", "invalid api key", "401", "403"):
		return "auth"
	case contains(msg, "timeout", "deadline", "context"):
		return "timeout"
	case contains(msg, "model_not_available", "model not found", "not available"):
		return "model_not_available"
	default:
		return "internal"
	}
}

func contains(s string, substrs ...string) bool {
	lower := toLower(s)
	for _, sub := range substrs {
		if containsStr(lower, toLower(sub)) {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	return strings.ToLower(s)
}

func containsStr(s, sub string) bool {
	return len(sub) <= len(s) && (s == sub || len(s) > 0 && findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
