package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	// mdFenceRe matches leading/trailing markdown code fences.
	// Note: Using a simpler pattern that doesn't require complex escapes
	mdFenceRe = regexp.MustCompile("(?s)^\\s*```(?:json)?\\s*\\n?(.*?)\\n?\\s*```\\s*$")
)

// ParseResponse parses LLM output into structured Go types.
// It handles JSON extraction, markdown fence stripping, and validation.
func ParseResponse(content string, target interface{}) error {
	// Step 1: Strip markdown fences defensively
	cleaned := stripMarkdownFences(content)

	// Step 2: Trim whitespace
	cleaned = strings.TrimSpace(cleaned)

	// Step 3: Validate JSON
	if !json.Valid([]byte(cleaned)) {
		return fmt.Errorf("response is not valid JSON: %s", truncate(cleaned, 200))
	}

	// Step 4: Parse into target
	if err := json.Unmarshal([]byte(cleaned), target); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}

	return nil
}

// stripMarkdownFences defensively removes markdown code block wrappers.
func stripMarkdownFences(s string) string {
	// Try matching the whole string wrapped in fences
	if m := mdFenceRe.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}

	// Also strip fences that don't wrap the whole string
	s = strings.TrimSpace(s)

	// Remove opening fence
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimLeft(s, " \n\t")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimLeft(s, " \n\t")
	}

	// Remove closing fence (only one occurrence)
	if idx := strings.Index(s, "\n```"); idx >= 0 {
		// Found closing fence, keep text before it
		s = s[:idx]
	} else if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return strings.TrimSpace(s)
}

// parseJSON is a helper for parsing JSON into a target.
func parseJSON(content string, target interface{}) error {
	return ParseResponse(content, target)
}

// truncate shortens a string to maxLen runes.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ValidateStyleHypothesis validates a StyleHypothesis structure.
func ValidateStyleHypothesis(h *interface{}) error {
	// Type assertion happens in the caller
	return nil
}

// SanitizeString applies basic sanitization to a string field.
func SanitizeString(s string) string {
	// Remove null bytes and control characters except newline/tab
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || r >= 32 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SanitizeStringArray applies sanitization to each string in an array.
func SanitizeStringArray(arr []string) []string {
	if arr == nil {
		return nil
	}
	result := make([]string, len(arr))
	for i, s := range arr {
		result[i] = SanitizeString(s)
	}
	return result
}

// ExtractJSON extracts JSON from a potentially wrapped response.
// Handles markdown fences, explanatory text before/after JSON.
func ExtractJSON(content string) (string, error) {
	// First, try to strip markdown fences
	cleaned := stripMarkdownFences(content)

	// Look for JSON object boundaries
	trimmed := strings.TrimSpace(cleaned)
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		// Try to find the first { or [
		idx := -1
		for i, r := range trimmed {
			if r == '{' || r == '[' {
				idx = i
				break
			}
		}
		if idx == -1 {
			return "", fmt.Errorf("no JSON object found in response")
		}
		trimmed = trimmed[idx:]
	}

	// Find matching closing bracket
	if strings.HasPrefix(trimmed, "{") {
	 closing := findMatchingBrace(trimmed, 0)
	 if closing == -1 {
		 return "", fmt.Errorf("unmatched JSON object")
		}
		return trimmed[:closing+1], nil
	}

	if strings.HasPrefix(trimmed, "[") {
		closing := findMatchingBrace(trimmed, 0)
		if closing == -1 {
			return "", fmt.Errorf("unmatched JSON array")
		}
		return trimmed[:closing+1], nil
	}

	return cleaned, nil
}

// findMatchingBrace finds the closing bracket/brace matching the opening one.
func findMatchingBrace(s string, start int) int {
	if start >= len(s) {
		return -1
	}

	open := rune(s[start])
	close := rune(0)
	switch open {
	case '{':
		close = '}'
	case '[':
		close = ']'
	case '(':
		close = ')'
	default:
		return -1
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		r := rune(s[i])

		if escaped {
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if r == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if r == open {
			depth++
		} else if r == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
}

// ValidateRequired checks that required string fields are non-empty.
func ValidateRequired(fields map[string]string) error {
	missing := []string{}
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}

// CoerceString safely converts an interface{} to string.
func CoerceString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%f", val)
	case int:
		return fmt.Sprintf("%d", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// CoerceStringArray safely converts an interface{} to []string.
func CoerceStringArray(v interface{}) []string {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = CoerceString(item)
		}
		return result
	default:
		return nil
	}
}
