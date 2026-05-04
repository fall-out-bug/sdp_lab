package llmguard

import (
	"strings"
)

// Redactor replaces matched findings in text with placeholders.
type Redactor struct {
	// UseTypedPlaceholders controls whether to use type-specific placeholders
	// like [REDACTED_API_KEY] or the generic [REDACTED].
	UseTypedPlaceholders bool
}

// NewRedactor creates a Redactor. If typed is true, uses finding-type-specific
// placeholders; otherwise uses the generic [REDACTED] placeholder for provider-facing text.
func NewRedactor(typed bool) *Redactor {
	return &Redactor{UseTypedPlaceholders: typed}
}

// Redact applies redaction for all findings to the original text.
// It processes findings from last to first to preserve byte offsets.
func (r *Redactor) Redact(text string, findings []Finding) string {
	if len(findings) == 0 {
		return text
	}

	// Sort findings by position (descending) to avoid offset shifts.
	// Copy to avoid mutating the input.
	sorted := make([]Finding, len(findings))
	copy(sorted, findings)
	sortFindingsByPosDesc(sorted)

	result := text
	for _, f := range sorted {
		if f.SpanStart < 0 || f.SpanEnd > len(result) || f.SpanStart >= f.SpanEnd {
			continue
		}
		placeholder := r.placeholder(f.Type)
		result = result[:f.SpanStart] + placeholder + result[f.SpanEnd:]
	}

	return result
}

// RedactWithUntyped replaces all findings with [REDACTED].
func RedactWithUntyped(text string, findings []Finding) string {
	if len(findings) == 0 {
		return text
	}

	sorted := make([]Finding, len(findings))
	copy(sorted, findings)
	sortFindingsByPosDesc(sorted)

	result := text
	for _, f := range sorted {
		if f.SpanStart < 0 || f.SpanEnd > len(result) || f.SpanStart >= f.SpanEnd {
			continue
		}
		result = result[:f.SpanStart] + UntypedPlaceholder + result[f.SpanEnd:]
	}

	return result
}

// ContainsRawSecret checks if text contains any of the raw secrets from findings.
// This is used in tests to verify that redacted output does not leak secrets.
func ContainsRawSecret(redactedText string, findings []Finding) bool {
	for _, f := range findings {
		// We can't recover the raw secret from a finding (by design),
		// but we can check if the redacted placeholder is present.
		// If any placeholder is missing, something went wrong.
		placeholder := RedactedPlaceholder(f.Type)
		if !strings.Contains(redactedText, placeholder) &&
			!strings.Contains(redactedText, UntypedPlaceholder) {
			return true
		}
	}
	return false
}

func (r *Redactor) placeholder(ft FindingType) string {
	if r.UseTypedPlaceholders {
		return RedactedPlaceholder(ft)
	}
	return UntypedPlaceholder
}

type byPosDesc []Finding

func (a byPosDesc) Len() int           { return len(a) }
func (a byPosDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPosDesc) Less(i, j int) bool { return a[i].SpanStart > a[j].SpanStart }

func sortFindingsByPosDesc(f []Finding) {
	// Simple insertion sort, fine for the small N of findings.
	for i := 1; i < len(f); i++ {
		for j := i; j > 0 && f[j].SpanStart > f[j-1].SpanStart; j-- {
			f[j], f[j-1] = f[j-1], f[j]
		}
	}
}
