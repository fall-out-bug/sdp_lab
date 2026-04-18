package security

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// SecretType represents the type of secret detected.
type SecretType string

const (
	SecretTypeAWSKey                  SecretType = "aws_key"
	SecretTypeGitHubToken             SecretType = "github_token"
	SecretTypePrivateKey              SecretType = "private_key"
	SecretTypeOpenAIKey               SecretType = "openai_key"
	SecretTypePasswordAssignment      SecretType = "password_assignment"
	SecretTypeStripeLiveKey           SecretType = "stripe_live_key"
	SecretTypeJWTToken                SecretType = "jwt_token"
	SecretTypeSlackToken              SecretType = "slack_token"
	SecretTypeConnectionString        SecretType = "connection_string_credentials"
	SecretTypeHighEntropyString       SecretType = "high_entropy_string"
)

// secretPattern pairs a compiled regex with a type label.
type secretPattern struct {
	re  *regexp.Regexp
	typ SecretType
}

// DefaultSecretPatterns contains all default secret detection patterns.
var DefaultSecretPatterns = []secretPattern{
	// AWS access key ID: AKIA followed by 16 alphanumeric characters
	{re: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), typ: SecretTypeAWSKey},

	// GitHub personal access token: ghp_ followed by 36 alphanumeric characters
	{re: regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`), typ: SecretTypeGitHubToken},

	// Private key headers (RSA, EC, generic)
	{re: regexp.MustCompile(`-----BEGIN (RSA |EC )?PRIVATE KEY-----`), typ: SecretTypePrivateKey},

	// OpenAI API key: sk- followed by 48 alphanumeric characters
	{re: regexp.MustCompile(`sk-[0-9a-zA-Z]{48}`), typ: SecretTypeOpenAIKey},

	// Password assignments in config/code
	{re: regexp.MustCompile(`(?i)(password|passwd)\s*[:=]\s*"[^"]+"`), typ: SecretTypePasswordAssignment},

	// Stripe live key: sk_live_ followed by 24+ alphanumeric characters
	{re: regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`), typ: SecretTypeStripeLiveKey},

	// JWT tokens: two base64-encoded segments separated by dots
	{re: regexp.MustCompile(`eyJ[A-Za-z0-9-_]{20,}\.eyJ[A-Za-z0-9-_]{20,}`), typ: SecretTypeJWTToken},

	// Slack tokens: xox[baprs]- followed by numbers and letters
	{re: regexp.MustCompile(`xox[baprs]-[0-9]{10,}-[0-9]{10,}-[0-9a-zA-Z]{24,}`), typ: SecretTypeSlackToken},

	// Connection strings with credentials: //user:pass@host
	{re: regexp.MustCompile(`//[^/@\s]+:[^/@\s]+@`), typ: SecretTypeConnectionString},
}

// SecretMatch represents a detected secret in content.
type SecretMatch struct {
	Type     SecretType `json:"type"`
	Position int        `json:"position"`
	Length   int        `json:"length"`
}

// SecurityFilter provides secret detection and redaction.
type SecurityFilter struct {
	patterns []secretPattern
}

// NewSecurityFilter creates a new security filter with default patterns.
func NewSecurityFilter() *SecurityFilter {
	return &SecurityFilter{
		patterns: DefaultSecretPatterns,
	}
}

// AddPattern adds a custom secret detection pattern.
func (sf *SecurityFilter) AddPattern(pattern string, typ SecretType) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	sf.patterns = append(sf.patterns, secretPattern{re: re, typ: typ})
	return nil
}

// ScanForSecrets detects secret patterns in content and returns matches.
func (sf *SecurityFilter) ScanForSecrets(content string) []SecretMatch {
	var matches []SecretMatch
	for _, p := range sf.patterns {
		locs := p.re.FindAllStringIndex(content, -1)
		for _, loc := range locs {
			matches = append(matches, SecretMatch{
				Type:     p.typ,
				Position: loc[0],
				Length:   loc[1] - loc[0],
			})
		}
	}
	return matches
}

// ScrubSecrets applies secret redaction to text content.
// Returns the scrubbed text and a count of redactions per type.
func (sf *SecurityFilter) ScrubSecrets(text string) (scrubbed string, redactionCounts map[string]int) {
	redactionCounts = make(map[string]int)
	result := text

	// Apply each pattern
	for _, p := range sf.patterns {
		matches := p.re.FindAllStringIndex(result, -1)
		if len(matches) == 0 {
			continue
		}

		// Build replacement string (reverse order to preserve indices)
		for i := len(matches) - 1; i >= 0; i-- {
			loc := matches[i]
			redaction := fmt.Sprintf("[REDACTED_%s]", p.typ)
			result = result[:loc[0]] + redaction + result[loc[1]:]
			redactionCounts[string(p.typ)]++
		}
	}

	return result, redactionCounts
}

// ScrubSecretsOutput applies JSON-aware secret scrubbing to output.
// It parses JSON, scrubs only string values (never keys), and reconstructs.
func (sf *SecurityFilter) ScrubSecretsOutput(jsonBytes []byte) ([]byte, map[string]int, error) {
	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	redactionCounts := make(map[string]int)
	scrubbed := sf.scrubJSONValue(data, redactionCounts)

	result, err := json.Marshal(scrubbed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return result, redactionCounts, nil
}

// scrubJSONValue recursively scrubs string values in JSON data.
func (sf *SecurityFilter) scrubJSONValue(value interface{}, counts map[string]int) interface{} {
	switch v := value.(type) {
	case string:
		scrubbed, typeCounts := sf.ScrubSecrets(v)
		for typ, count := range typeCounts {
			counts[typ] += count
		}
		return scrubbed
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			// Keys are never scrubbed, only values
			result[key] = sf.scrubJSONValue(val, counts)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = sf.scrubJSONValue(val, counts)
		}
		return result
	default:
		return v
	}
}

// ScrubSecrets is a convenience function using default patterns.
func ScrubSecrets(text string) (string, map[string]int, error) {
	sf := NewSecurityFilter()
	scrubbed, counts := sf.ScrubSecrets(text)
	return scrubbed, counts, nil
}

// ScrubSecretsJSON is a convenience function for JSON-aware scrubbing.
func ScrubSecretsJSON(jsonBytes []byte) ([]byte, map[string]int, error) {
	sf := NewSecurityFilter()
	return sf.ScrubSecretsOutput(jsonBytes)
}

// ContainsSecrets checks if text contains any detected secrets.
func (sf *SecurityFilter) ContainsSecrets(text string) bool {
	matches := sf.ScanForSecrets(text)
	return len(matches) > 0
}

// GetSecretTypes returns all secret types detected in text.
func (sf *SecurityFilter) GetSecretTypes(text string) []SecretType {
	matches := sf.ScanForSecrets(text)
	typeSet := make(map[SecretType]bool)
	for _, m := range matches {
		typeSet[m.Type] = true
	}

	types := make([]SecretType, 0, len(typeSet))
	for typ := range typeSet {
		types = append(types, typ)
	}
	return types
}

// RedactionFormat returns the standard redaction format for a secret type.
func RedactionFormat(typ SecretType) string {
	return fmt.Sprintf("[REDACTED_%s]", typ)
}

// IsRedaction checks if a string matches a redaction marker.
func IsRedaction(s string) bool {
	return strings.HasPrefix(s, "[REDACTED_") && strings.HasSuffix(s, "]")
}

// ExtractRedactionType extracts the secret type from a redaction marker.
func ExtractRedactionType(s string) (SecretType, bool) {
	if !IsRedaction(s) {
		return "", false
	}
	inner := s[10 : len(s)-1] // Strip [REDACTED_ and ]
	return SecretType(inner), true
}
