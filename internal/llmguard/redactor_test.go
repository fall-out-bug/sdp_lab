package llmguard

import (
	"strings"
	"testing"
)

func TestRedactor_TypedPlaceholders(t *testing.T) {
	r := NewRedactor(true)
	text := "key is sk-proj-abc123def456ghi789jkl012mno345pqr end"
	findings := []Finding{
		{
			Type:      FindingOpenAIKey,
			SpanStart: 7,
			SpanEnd:   47,
		},
	}

	result := r.Redact(text, findings)
	if !strings.Contains(result, "[REDACTED_API_KEY]") {
		t.Errorf("expected typed placeholder, got: %s", result)
	}
	if strings.Contains(result, "sk-proj-") {
		t.Error("redacted text should not contain raw secret")
	}
}

func TestRedactor_UntypedPlaceholders(t *testing.T) {
	r := NewRedactor(false)
	text := "key is sk-proj-abc123def456ghi789jkl012mno345pqr end"
	findings := []Finding{
		{
			Type:      FindingOpenAIKey,
			SpanStart: 7,
			SpanEnd:   47,
		},
	}

	result := r.Redact(text, findings)
	if !strings.Contains(result, "[REDACTED]") {
		t.Errorf("expected untyped placeholder, got: %s", result)
	}
	if strings.Contains(result, "sk-proj-") {
		t.Error("redacted text should not contain raw secret")
	}
}

func TestRedactWithUntyped(t *testing.T) {
	text := "aws=AKIAIOSFODNN7EXAMPLE and key=sk-proj-abc123def456ghi789jkl012mno345pqr"
	findings := []Finding{
		{Type: FindingAWSKey, SpanStart: 4, SpanEnd: 24},
		{Type: FindingOpenAIKey, SpanStart: 33, SpanEnd: 73},
	}

	result := RedactWithUntyped(text, findings)
	// Both should be replaced with [REDACTED]
	count := strings.Count(result, "[REDACTED]")
	if count != 2 {
		t.Errorf("expected 2 [REDACTED] placeholders, got %d: %s", count, result)
	}
	if strings.Contains(result, "AKIA") {
		t.Error("redacted text should not contain AWS key")
	}
	if strings.Contains(result, "sk-proj-") {
		t.Error("redacted text should not contain OpenAI key")
	}
}

func TestRedactor_MultipleFindings(t *testing.T) {
	r := NewRedactor(true)
	text := "aws AKIAIOSFODNN7EXAMPLE email test@example.com"
	findings := []Finding{
		{Type: FindingAWSKey, SpanStart: 4, SpanEnd: 24},
		{Type: FindingEmail, SpanStart: 31, SpanEnd: 47},
	}

	result := r.Redact(text, findings)
	if !strings.Contains(result, "[REDACTED_AWS_KEY]") {
		t.Errorf("expected AWS placeholder, got: %s", result)
	}
	if !strings.Contains(result, "[REDACTED_EMAIL]") {
		t.Errorf("expected email placeholder, got: %s", result)
	}
}

func TestRedactor_EmptyFindings(t *testing.T) {
	r := NewRedactor(true)
	text := "hello world"
	result := r.Redact(text, nil)
	if result != text {
		t.Errorf("no findings should not change text, got: %s", result)
	}
}

func TestRedactor_NoRawSecretsInOutput(t *testing.T) {
	s := testScanner()
	secrets := []string{
		"AKIAIOSFODNN7EXAMPLE",
		"sk-proj-abc123def456ghi789jkl012mno345pqr",
		"ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
		"test@example.com",
	}

	for _, secret := range secrets {
		text := "found: " + secret + " end"
		result := s.Scan(text)

		if len(result.Findings) == 0 {
			t.Logf("no finding for %s (may be expected miss)", secret[:min(10, len(secret))])
			continue
		}

		r := NewRedactor(true)
		redacted := r.Redact(text, result.Findings)

		if strings.Contains(redacted, secret) {
			t.Errorf("redacted text still contains raw secret: %s", secret[:min(10, len(secret))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
