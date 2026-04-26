package memory

import (
	"regexp"
	"strings"
)

// secretPattern defines a pattern to detect secrets.
type secretPattern struct {
	name    string
	pattern *regexp.Regexp
}

var (
	// AWS Access Key ID pattern (20 characters, starts with AKIA)
	awsKeyPattern = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)

	// GitHub personal access token patterns (classic: ghp_ + 36 chars, fine-grained: ghp_ + 40+ chars)
	ghpPattern = regexp.MustCompile(`\bghp_[A-Za-z0-9]{36,}\b`)
	ghoPattern = regexp.MustCompile(`\bgho_[A-Za-z0-9]{36,}\b`)
	ghuPattern = regexp.MustCompile(`\bghu_[A-Za-z0-9]{36,}\b`)
	ghsPattern = regexp.MustCompile(`\bghs_[A-Za-z0-9]{36,}\b`)
	ghrPattern = regexp.MustCompile(`\bghr_[A-Za-z0-9]{36,}\b`)

	// Private key markers
	privateKeyPattern = regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`)

	// JWT token pattern (header.payload.signature)
	jwtPattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
)

// allSecretPatterns holds all secret detection patterns.
var allSecretPatterns = []secretPattern{
	{"AWS Access Key", awsKeyPattern},
	{"GitHub Personal Access Token (classic)", ghpPattern},
	{"GitHub OAuth Access Token", ghoPattern},
	{"GitHub User Server Token", ghuPattern},
	{"GitHub Server Token", ghsPattern},
	{"GitHub Refresh Token", ghrPattern},
	{"Private Key", privateKeyPattern},
	{"JWT Token", jwtPattern},
}

// benignPhrases contains phrases that may match secret patterns but are not actual secrets.
var benignPhrases = []string{
	"token budget",
	"password field",
	"key value",
	"access key id",
	"secret key base",
	"auth token",
	"api key",
	"session token",
	"csrf token",
	"bearer token",
}

// isBenignPhrase checks if the matched text is part of a benign phrase.
func isBenignPhrase(matchedText, fullContent string) bool {
	lowerContent := strings.ToLower(fullContent)

	for _, benign := range benignPhrases {
		if strings.Contains(lowerContent, benign) {
			// Check if the match is within or adjacent to the benign phrase
			idx := strings.Index(lowerContent, benign)
			if idx != -1 {
				// Get a window around the benign phrase
				start := idx - 50
				if start < 0 {
					start = 0
				}
				end := idx + len(benign) + 50
				if end > len(lowerContent) {
					end = len(lowerContent)
				}
				window := lowerContent[start:end]

				// If the matched text is in this window, it's likely benign
				lowerMatch := strings.ToLower(matchedText)
				if strings.Contains(window, lowerMatch) {
					return true
				}
			}
		}
	}

	return false
}

// ScanForSecrets checks content for high-confidence secret patterns.
// Returns (found bool, patternName string).
func ScanForSecrets(content string) (bool, string) {
	for _, sp := range allSecretPatterns {
		matches := sp.pattern.FindAllString(content, -1)
		for _, match := range matches {
			if !isBenignPhrase(match, content) {
				return true, sp.name
			}
		}
	}
	return false, ""
}
