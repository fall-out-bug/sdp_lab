package security

import (
	"math"
	"regexp"
)

// highEntropyAllowlist contains exact full-string regex patterns for strings
// that are high-entropy but are known non-secrets (integrity hashes, UUIDs, etc.).
// Each pattern uses full anchors (^...$).
var highEntropyAllowlist = []*regexp.Regexp{
	// UUID v4 format
	regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
	// Integrity hashes (Subresource Integrity)
	regexp.MustCompile(`^sha(256|384|512)-[A-Za-z0-9+/=]{40,100}$`),
	// SHA256 hex (64 hex chars)
	regexp.MustCompile(`^[0-9a-f]{64}$`),
	// SHA512 hex (128 hex chars)
	regexp.MustCompile(`^[0-9a-f]{128}$`),
	// SHA1 hex (40 hex chars) - deprecated but still encountered
	regexp.MustCompile(`^[0-9a-f]{40}$`),
	// MD5 hex (32 hex chars) - deprecated but still encountered
	regexp.MustCompile(`^[0-9a-f]{32}$`),
	// Base64 encoded data (common for non-secret binary data)
	// 20+ chars of valid base64
	regexp.MustCompile(`^[A-Za-z0-9+/]{20,}={0,2}$`),
}

// HighEntropyCheck flags strings with Shannon entropy > 4.5 bits/char and
// length >= 20 that don't match known non-secret patterns.
//
// The allowlist uses EXACT regex patterns only — no broad category allowances.
// This minimizes false positives while catching potential secrets.
//
// Parameters:
//   - s: The string to check
//   - context: Optional context for future enhancements (currently unused)
//
// Returns true if the string appears to be a potential secret (high entropy
// and not allowlisted), false otherwise.
func HighEntropyCheck(s string, context string) bool {
	// Short strings cannot contain meaningful secrets
	if len(s) < 20 {
		return false
	}

	// Check allowlist first - these are known non-secrets
	for _, pattern := range highEntropyAllowlist {
		if pattern.MatchString(s) {
			return false
		}
	}

	// Compute Shannon entropy
	entropy := shannonEntropy(s)

	// Threshold: 4.5 bits per character indicates high randomness
	// This catches potential keys, tokens, and other secrets
	if entropy > 4.5 {
		return true
	}

	_ = context // context reserved for future use
	return false
}

// shannonEntropy computes the Shannon entropy of a string in bits per character.
//
// Shannon entropy measures the randomness or unpredictability of data.
// Higher values indicate more random content, which is characteristic of
// cryptographic secrets.
//
// Formula: H = -Σ p(x) * log2(p(x))
// where p(x) is the probability of character x appearing in the string.
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Count character frequencies
	freq := make(map[rune]float64)
	for _, r := range s {
		freq[r]++
	}

	// Calculate entropy
	length := float64(len(s))
	var entropy float64
	for _, count := range freq {
		p := count / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// EstimateCharsetSize estimates the size of the character set used in a string.
// This is useful for adjusting entropy thresholds based on the character set.
func EstimateCharsetSize(s string) int {
	seen := make(map[rune]bool)
	for _, r := range s {
		seen[r] = true
	}
	return len(seen)
}

// MaxShannonEntropy returns the theoretical maximum entropy for a given
// character set size and string length.
func MaxShannonEntropy(charsetSize, length int) float64 {
	if charsetSize <= 0 || length <= 0 {
		return 0
	}
	return float64(length) * math.Log2(float64(charsetSize))
}

// NormalizeEntropy normalizes an entropy value to a 0-1 range based on
// the theoretical maximum for the given string.
func NormalizeEntropy(s string, entropy float64) float64 {
	if len(s) == 0 {
		return 0
	}

	charsetSize := EstimateCharsetSize(s)
	if charsetSize <= 1 {
		return 0
	}

	maxEntropy := MaxShannonEntropy(charsetSize, len(s))
	if maxEntropy == 0 {
		return 0
	}

	return entropy / maxEntropy
}
