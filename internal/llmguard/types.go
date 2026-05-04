// Package llmguard provides runtime input/output guard for SDP LLM calls.
//
// It scans prompts and model responses for secrets and policy violations,
// redacts matched spans, and produces audit-ready evidence without storing
// raw secrets.
package llmguard

// VerdictState represents the outcome of a guard check.
type VerdictState string

const (
	// VerdictCleanAllowed means no findings, provider was called, response is safe.
	VerdictCleanAllowed VerdictState = "clean_allowed"
	// VerdictRedactedAllowed means findings were redacted and provider was called.
	VerdictRedactedAllowed VerdictState = "redacted_allowed"
	// VerdictInputBlocked means input findings blocked the provider call.
	VerdictInputBlocked VerdictState = "input_blocked"
	// VerdictOutputBlocked means output findings blocked the response release.
	VerdictOutputBlocked VerdictState = "output_blocked"
	// VerdictAllowedWithOutputFindings means output has advisory findings but caller may proceed.
	VerdictAllowedWithOutputFindings VerdictState = "allowed_with_output_findings"
	// VerdictProviderErrorAfterInputPass means input passed but provider failed.
	VerdictProviderErrorAfterInputPass VerdictState = "provider_error_after_input_pass"
	// VerdictAuditFailed means audit sink failed; fail-closed by default.
	VerdictAuditFailed VerdictState = "audit_failed"
	// VerdictScanBudgetExceeded means scanner budget was exhausted.
	VerdictScanBudgetExceeded VerdictState = "scan_budget_exceeded"
	// VerdictClassifierAdvisoryAllowed means classifier found advisory risk but demo mode allows continuation.
	VerdictClassifierAdvisoryAllowed VerdictState = "classifier_advisory_allowed"
	// VerdictClassifierIncomplete means classifier failed to produce a complete verdict.
	VerdictClassifierIncomplete VerdictState = "classifier_incomplete"
	// VerdictNeedsReview means classifier found uncertain risk requiring operator review.
	VerdictNeedsReview VerdictState = "needs_review"
)

// FindingSeverity classifies how serious a finding is.
type FindingSeverity string

const (
	SeverityHigh FindingSeverity = "high"
	SeverityLow  FindingSeverity = "low"
)

// FindingType classifies what was detected.
type FindingType string

const (
	FindingOpenAIKey   FindingType = "openai_key"
	FindingGitHubToken FindingType = "github_token"
	FindingAWSKey      FindingType = "aws_key"
	FindingBearerToken FindingType = "bearer_token"
	FindingEmail       FindingType = "email"
	FindingCard        FindingType = "card"
	FindingPhone       FindingType = "phone"
	FindingBase64Key   FindingType = "base64_secret"
	FindingSplitKey    FindingType = "split_secret"
	// Output-side finding types.
	FindingGeneratedSecret    FindingType = "generated_secret"
	FindingPromptDisclosure   FindingType = "prompt_disclosure"
	FindingSuspiciousURL      FindingType = "suspicious_url"
	FindingShellCommand       FindingType = "shell_command"
)

// Finding represents a single detected secret or policy violation.
type Finding struct {
	Type     FindingType    `json:"type"`
	Severity FindingSeverity `json:"severity"`
	// SpanStart and SpanEnd are byte offsets in the scanned text.
	SpanStart int `json:"span_start"`
	SpanEnd   int `json:"span_end"`
	// Redacted is the text with the secret replaced by a placeholder.
	Redacted string `json:"redacted"`
	// ScanMode indicates which scan path produced this finding.
	ScanMode ScanMode `json:"scan_mode"`
}

// ScanMode indicates the scan path that produced a finding.
type ScanMode string

const (
	ScanModeRaw          ScanMode = "raw"
	ScanModeBase64Decoded ScanMode = "base64_decoded"
	ScanModeSplitJoined  ScanMode = "split_joined"
)

// ScanTrace records the path taken for a scan, used for testing transparency.
type ScanTrace struct {
	Mode            ScanMode `json:"mode"`
	CandidatesTried int      `json:"candidates_tried"`
	Matched         bool     `json:"matched"`
	// RedactedExcerpt is a short redacted excerpt of the candidate that was tried.
	RedactedExcerpt string `json:"redacted_excerpt,omitempty"`
}

// ScanResult holds the output of a scan operation.
type ScanResult struct {
	Findings []Finding   `json:"findings"`
	Traces   []ScanTrace `json:"traces"`
	// BudgetExceeded is true when the scanner hit its budget limit.
	BudgetExceeded bool `json:"budget_exceeded"`
}

// InputAction determines what happens when input findings are detected.
type InputAction string

const (
	InputActionBlock  InputAction = "block"
	InputActionRedact InputAction = "redact"
)

// OutputAction determines what happens when output findings are detected.
type OutputAction string

const (
	OutputActionBlock    OutputAction = "block"
	OutputActionAdvisory OutputAction = "advisory"
)

// Policy defines the guard behavior for a gateway instance.
// Policy is immutable after construction.
type Policy struct {
	// InputAction controls what happens when secrets are found in prompts.
	InputAction InputAction `json:"input_action"`
	// OutputAction controls what happens when suspicious output is found.
	OutputAction OutputAction `json:"output_action"`
	// MaxInputBytes limits the total bytes accepted for scanning.
	MaxInputBytes int `json:"max_input_bytes"`
	// MaxDecodedBytes limits bytes after base64 decode attempts.
	MaxDecodedBytes int `json:"max_decoded_bytes"`
	// StrictBudgetMode makes budget exhaustion fail closed.
	StrictBudgetMode bool `json:"strict_budget_mode"`
	// ModelPricing maps model IDs to per-token pricing in USD.
	ModelPricing map[string]ModelPricing `json:"model_pricing,omitempty"`
	// Classifier is optional local LLM classifier config. Nil means disabled.
	Classifier *ClassifierConfig `json:"classifier,omitempty"`
}

// ClassifierConfig configures the optional local LLM classifier layer.
type ClassifierConfig struct {
	Enabled                bool   `json:"enabled"`
	BaseURL                string `json:"base_url"`
	Model                  string `json:"model"`
	APIKeyEnv              string `json:"api_key_env,omitempty"`
	TimeoutMs              int    `json:"timeout_ms"`
	TotalTimeoutMs         int    `json:"total_timeout_ms"`
	MaxChunkBytes          int    `json:"max_chunk_bytes"`
	OverlapBytes           int    `json:"overlap_bytes"`
	MaxClassifierChunks    int    `json:"max_classifier_chunks"`
	MaxParallelChunks      int    `json:"max_parallel_chunks"`
	BlockConfidenceThreshold float64 `json:"block_confidence_threshold"`
	StrictMode             bool   `json:"strict_mode"`
}

// DefaultClassifierConfig returns a safe default classifier config.
func DefaultClassifierConfig() ClassifierConfig {
	return ClassifierConfig{
		Enabled:                false,
		BaseURL:                "http://127.0.0.1:11434/v1",
		Model:                  "qwen2.5-coder:7b",
		TimeoutMs:              3000,
		TotalTimeoutMs:         10000,
		MaxChunkBytes:          12000,
		OverlapBytes:           512,
		MaxClassifierChunks:    64,
		MaxParallelChunks:      4,
		BlockConfidenceThreshold: 0.75,
		StrictMode:             true,
	}
}

// ModelPricing holds per-token cost data for a model.
type ModelPricing struct {
	PromptPer1M     float64 `json:"prompt_per_1m"`
	CompletionPer1M float64 `json:"completion_per_1m"`
}

// DefaultPolicy returns a strict-default policy suitable for production use.
func DefaultPolicy() Policy {
	return Policy{
		InputAction:     InputActionBlock,
		OutputAction:    OutputActionBlock,
		MaxInputBytes:   1 << 20, // 1 MiB
		MaxDecodedBytes: 2 << 20, // 2 MiB
		StrictBudgetMode: true,
	}
}

// RedactedPlaceholder returns the typed placeholder for a finding type.
func RedactedPlaceholder(ft FindingType) string {
	switch ft {
	case FindingOpenAIKey, FindingBase64Key, FindingSplitKey:
		return "[REDACTED_API_KEY]"
	case FindingGitHubToken:
		return "[REDACTED_GITHUB_TOKEN]"
	case FindingAWSKey:
		return "[REDACTED_AWS_KEY]"
	case FindingBearerToken:
		return "[REDACTED_BEARER_TOKEN]"
	case FindingEmail:
		return "[REDACTED_EMAIL]"
	case FindingCard:
		return "[REDACTED_CARD]"
	case FindingPhone:
		return "[REDACTED_PHONE]"
	case FindingGeneratedSecret:
		return "[REDACTED_GENERATED_SECRET]"
	case FindingPromptDisclosure:
		return "[REDACTED_PROMPT_DISCLOSURE]"
	case FindingSuspiciousURL:
		return "[REDACTED_URL]"
	case FindingShellCommand:
		return "[REDACTED_SHELL_COMMAND]"
	default:
		return "[REDACTED]"
	}
}

// UntypedPlaceholder is used for provider-facing redacted text.
const UntypedPlaceholder = "[REDACTED]"
