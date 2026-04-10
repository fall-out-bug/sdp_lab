package architect

import (
	"regexp"
	"strings"
)

// SecretMatch represents a detected secret in content.
type SecretMatch struct {
	Type     string `json:"type"`
	Position int    `json:"position"`
	Length   int    `json:"length"`
}

// secretPattern pairs a compiled regex with a type label and length hint.
type secretPattern struct {
	re     *regexp.Regexp
	typ    string
	length int // fixed match length (0 = use match length)
}

// SecurityFilter sanitizes a CodebaseProfile before sending to an LLM.
// It detects secrets, scrubs PII (internal package names, usernames),
// and enforces the --allow-external-llm policy.
type SecurityFilter struct {
	// patterns are compiled regexes for detecting secrets.
	patterns []secretPattern

	// AllowExternalLLM must be explicitly set to true to send data to cloud LLMs.
	AllowExternalLLM bool
}

// NewSecurityFilter returns a SecurityFilter with common secret patterns.
func NewSecurityFilter() *SecurityFilter {
	return &SecurityFilter{
		patterns: []secretPattern{
			{re: regexp.MustCompile(`AKIA[0-9A-Z]{16}`), typ: "aws_key", length: 20},
			{re: regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`), typ: "github_token", length: 40},
			{re: regexp.MustCompile(`-----BEGIN (RSA |EC )?PRIVATE KEY-----`), typ: "private_key"},
			{re: regexp.MustCompile(`sk-[0-9a-zA-Z]{48}`), typ: "openai_key"},
			{re: regexp.MustCompile(`(?i)(password|passwd)\s*[:=]\s*"[^"]+"`), typ: "password_assignment"},
		},
		AllowExternalLLM: false,
	}
}

// ExternalLLMAllowed returns whether sending data to external LLMs is permitted.
func (sf *SecurityFilter) ExternalLLMAllowed() bool {
	return sf.AllowExternalLLM
}

// ScanForSecrets detects secret patterns in content and returns matches.
func (sf *SecurityFilter) ScanForSecrets(content string) []SecretMatch {
	var matches []SecretMatch
	for _, p := range sf.patterns {
		locs := p.re.FindAllStringIndex(content, -1)
		for _, loc := range locs {
			length := loc[1] - loc[0]
			if p.length > 0 {
				length = p.length
			}
			matches = append(matches, SecretMatch{
				Type:     p.typ,
				Position: loc[0],
				Length:   length,
			})
		}
	}
	return matches
}

// reUserPath matches /Users/<username>/ paths.
var reUserPath = regexp.MustCompile(`/Users/[^/]+/`)

// reInternalPkg matches Java-style internal package names (com.xxx.yyy...).
var reInternalPkg = regexp.MustCompile(`\bcom\.\w+(?:\.\w+)+`)

// Sanitize removes secrets and PII from a CodebaseProfile.
// Returns a sanitized copy safe for LLM consumption.
func (sf *SecurityFilter) Sanitize(profile *CodebaseProfile) *CodebaseProfile {
	result := &CodebaseProfile{
		Name:    profile.Name,
		Summary: sf.sanitizeString(profile.Summary),
	}

	// Copy and sanitize files.
	if profile.Files != nil {
		result.Files = make(map[string]string, len(profile.Files))
		for path, content := range profile.Files {
			sanitizedPath := sf.sanitizeString(path)
			result.Files[sanitizedPath] = sf.sanitizeString(content)
		}
	}

	// Copy and sanitize metadata.
	if profile.Metadata != nil {
		result.Metadata = make(map[string]string, len(profile.Metadata))
		for k, v := range profile.Metadata {
			result.Metadata[k] = sf.sanitizeString(v)
		}
	}

	return result
}

// sanitizeString applies all sanitization passes to a string.
func (sf *SecurityFilter) sanitizeString(s string) string {
	// 1. Redact secrets.
	for _, p := range sf.patterns {
		s = p.re.ReplaceAllStringFunc(s, func(match string) string {
			return "[REDACTED:" + p.typ + "]"
		})
	}

	// 2. Scrub user paths.
	s = reUserPath.ReplaceAllString(s, "/Users/[REDACTED]/")

	// 3. Hash internal package names.
	s = reInternalPkg.ReplaceAllStringFunc(s, func(match string) string {
		parts := strings.Split(match, ".")
		if len(parts) < 3 {
			return match
		}
		// Replace with "pkg." + last segment
		return "pkg." + parts[len(parts)-1]
	})

	return s
}

// SecretsFound represents secrets detected during sanitization.
type SecretsFound struct {
	Count    int      `json:"count"`
	Types    []string `json:"types,omitempty"`
	Redacted bool     `json:"redacted"`
}
