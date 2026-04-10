package architect

import "regexp"

// SecurityFilter sanitizes a CodebaseProfile before sending to an LLM.
// It detects secrets, scrubs PII (internal package names, usernames),
// and enforces the --allow-external-llm policy.
type SecurityFilter struct {
	// SecretPatterns are compiled regexes for detecting secrets.
	SecretPatterns []*regexp.Regexp

	// AllowExternalLLM must be explicitly set to true to send data to cloud LLMs.
	AllowExternalLLM bool
}

// DefaultSecurityFilter returns a SecurityFilter with common secret patterns.
func DefaultSecurityFilter() *SecurityFilter {
	return &SecurityFilter{
		SecretPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`),                    // AWS access key
			regexp.MustCompile(`(?i)(password|passwd|secret)\s*[:=]\s*\S+`), // password assignments
			regexp.MustCompile(`(?i)api[_-]?key\s*[:=]\s*\S+`),             // API keys
			regexp.MustCompile(`-----BEGIN (RSA |EC )?PRIVATE KEY-----`),    // private keys
			regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`),                      // GitHub tokens
			regexp.MustCompile(`sk-[0-9a-zA-Z]{48}`),                       // OpenAI keys
		},
		AllowExternalLLM: false,
	}
}

// Sanitize removes secrets and PII from a CodebaseProfile.
// Returns a sanitized copy safe for LLM consumption.
func (sf *SecurityFilter) Sanitize(profile *CodebaseProfile) *CodebaseProfile {
	// TODO: implement secret scrubbing, PII hashing, internal name anonymization
	return profile
}

// SecretsFound represents secrets detected during sanitization.
type SecretsFound struct {
	Count    int      `json:"count"`
	Types    []string `json:"types,omitempty"`     // "aws_key", "private_key", etc.
	Redacted bool     `json:"redacted"`
}
