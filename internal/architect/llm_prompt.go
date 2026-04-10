package architect

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

// generateDelimiter creates a 32-character hex string using crypto/rand.
// The delimiter is used to wrap code content in LLM prompts for
// prompt-injection defense.
func generateDelimiter() (string, error) {
	b := make([]byte, 16) // 128 bits -> 32 hex chars
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// WrapForLLM wraps code content in random delimiters for prompt injection defense.
// The random delimiter makes it computationally infeasible for injected content
// to forge the boundary markers.
// Returns the delimiter, the wrapped string, and any error from rand generation.
func WrapForLLM(code string) (delimiter string, wrapped string, err error) {
	delim, err := generateDelimiter()
	if err != nil {
		return "", "", err
	}
	prefix := "---BEGIN_CODE_CONTEXT_" + delim + "---\n"
	suffix := "\n---END_CODE_CONTEXT_" + delim + "---"
	return delim, prefix + code + suffix, nil
}

// rolePattern matches lines that look like LLM role injection attempts.
var rolePattern = regexp.MustCompile(`(?m)^\s*(?:SYSTEM|ASSISTANT|USER|INSTRUCTION|IMPORTANT|IGNORE)\s*:.*$`)

// mdRolePattern matches markdown code-fence or horizontal-rule lines that
// attempt to inject LLM roles.
var mdRolePattern = regexp.MustCompile("(?m)^\\s*(?:```|---)\\s*(?:system|assistant|user)\\s*$")

// SanitizeForLLM strips instruction-like patterns from code content.
// This is a defense-in-depth measure only -- the primary defense is delimiter
// wrapping via WrapForLLM. SanitizeForLLM should be called on the raw code
// before wrapping.
func SanitizeForLLM(code string, delimiter string) string {
	// Strip lines that look like role injection.
	code = rolePattern.ReplaceAllString(code, "[STRIPPED]")

	// Strip markdown role injection.
	code = mdRolePattern.ReplaceAllString(code, "[STRIPPED]")

	// Remove the delimiter itself if it somehow appears in the content,
	// preventing delimiter forgery.
	if delimiter != "" {
		forged := "---BEGIN_CODE_CONTEXT_" + delimiter + "---"
		code = strings.ReplaceAll(code, forged, "[STRIPPED]")
		forgedEnd := "---END_CODE_CONTEXT_" + delimiter + "---"
		code = strings.ReplaceAll(code, forgedEnd, "[STRIPPED]")
	}

	return code
}
