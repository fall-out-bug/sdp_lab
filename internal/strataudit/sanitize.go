package strataudit

import (
	"regexp"
	"strings"
)

var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(previous|all|above)\s+instructions?`),
	regexp.MustCompile(`(?i)system\s*:\s*override`),
	regexp.MustCompile(`(?i)act\s+as\s+if\s+you\s+are`),
	regexp.MustCompile(`(?i)disregard\s+(all|previous|above|your)`),
	regexp.MustCompile(`(?i)forget\s+(everything|all|your)\s+(previous\s+)?instructions?`),
	regexp.MustCompile(`</document_content>`),
	regexp.MustCompile(`</document>`),
	regexp.MustCompile(`</source_entity>`),
	regexp.MustCompile(`</target_entity>`),
}

func SanitizeForPrompt(content string) string {
	for _, pat := range injectionPatterns {
		content = pat.ReplaceAllString(content, "[CONTENT REDACTED]")
	}
	content = regexp.MustCompile(`(\[CONTENT REDACTED]\s*)+`).ReplaceAllString(content, "[CONTENT REDACTED] ")
	return strings.TrimSpace(content)
}
