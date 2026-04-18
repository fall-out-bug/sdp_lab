package security

import (
	"regexp"
	"strings"
)

// PII patterns for detecting and scrubbing Personally Identifiable Information.
var (
	// Email addresses
	reEmail = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	// IPv4 addresses
	reIPv4 = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`)

	// IPv6 addresses (simplified pattern)
	reIPv6 = regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`)

	// Credit card numbers (Luhn check would be more accurate but this is a basic pattern)
	reCreditCard = regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)

	// SSN (US Social Security Numbers) - XXX-XX-XXXX format
	reSSN = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// Phone numbers (various formats)
	rePhone = regexp.MustCompile(`\b(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}\b`)

	// API keys with identifiable prefixes (beyond those in security.go)
	reAPIKey = regexp.MustCompile(`\b[A-Za-z0-9]{20,}\b`)

	// Internal package names (Java-style)
	reInternalPkg = regexp.MustCompile(`\bcom\.\w+(?:\.\w+)+`)

	// User paths (Unix-like)
	reUnixUserPath = regexp.MustCompile(`/home/[^/]+/|/Users/[^/]+/`)

	// User paths (Windows-like)
	reWindowsUserPath = regexp.MustCompile(`C:\\Users\\[^\\]+\\`)

	// URLs with potential sensitive data
	reURL = regexp.MustCompile(`https?://[^\s<>"]+`)
)

// PIIType represents the type of PII detected.
type PIIType string

const (
	PIITypeEmail      PIIType = "email"
	PIITypeIPv4       PIIType = "ipv4"
	PIITypeIPv6       PIIType = "ipv6"
	PIITypeCreditCard PIIType = "credit_card"
	PIITypeSSN        PIIType = "ssn"
	PIITypePhone      PIIType = "phone"
	PIITypeAPIKey     PIIType = "api_key"
	PIITypeInternalPkg PIIType = "internal_package"
	PIITypeUserPath   PIIType = "user_path"
	PIITypeURL        PIIType = "url"
)

// PIIMatch represents a detected PII instance.
type PIIMatch struct {
	Type     PIIType `json:"type"`
	Position int     `json:"position"`
	Length   int     `json:"length"`
	Original string  `json:"original,omitempty"` // Not included in redacted output
}

// PIIScrubber scrubs PII from text content.
type PIIScrubber struct {
	// Enabled PII types to detect and scrub
	enabledTypes map[PIIType]bool

	// Custom patterns for project-specific PII
	customPatterns []struct {
		typ   PIIType
		pattern *regexp.Regexp
	}
}

// NewPIIScrubber creates a new PII scrubber with default enabled types.
func NewPIIScrubber() *PIIScrubber {
	return &PIIScrubber{
		enabledTypes: map[PIIType]bool{
			PIITypeEmail:      true,
			PIITypeIPv4:       true,
			PIITypeIPv6:       true,
			PIITypeCreditCard: true,
			PIITypeSSN:        true,
			PIITypePhone:      true,
			PIITypeAPIKey:     false, // Disabled by default (high false positive rate)
			PIITypeInternalPkg: true,
			PIITypeUserPath:   true,
			PIITypeURL:        false, // Disabled by default (legitimate URLs are common)
		},
		customPatterns: make([]struct {
			typ   PIIType
			pattern *regexp.Regexp
		}, 0),
	}
}

// EnableType enables detection for a specific PII type.
func (ps *PIIScrubber) EnableType(typ PIIType) {
	ps.enabledTypes[typ] = true
}

// DisableType disables detection for a specific PII type.
func (ps *PIIScrubber) DisableType(typ PIIType) {
	ps.enabledTypes[typ] = false
}

// AddCustomPattern adds a custom regex pattern for PII detection.
func (ps *PIIScrubber) AddCustomPattern(typ PIIType, pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	ps.customPatterns = append(ps.customPatterns, struct {
		typ   PIIType
		pattern *regexp.Regexp
	}{
		typ:   typ,
		pattern: re,
	})
	return nil
}

// Scan scans text for PII and returns all matches.
func (ps *PIIScrubber) Scan(text string) []PIIMatch {
	var matches []PIIMatch

	// Check each enabled PII type
	if ps.enabledTypes[PIITypeEmail] {
		matches = append(matches, ps.scanPattern(text, reEmail, PIITypeEmail)...)
	}
	if ps.enabledTypes[PIITypeIPv4] {
		matches = append(matches, ps.scanPattern(text, reIPv4, PIITypeIPv4)...)
	}
	if ps.enabledTypes[PIITypeIPv6] {
		matches = append(matches, ps.scanPattern(text, reIPv6, PIITypeIPv6)...)
	}
	if ps.enabledTypes[PIITypeCreditCard] {
		matches = append(matches, ps.scanPattern(text, reCreditCard, PIITypeCreditCard)...)
	}
	if ps.enabledTypes[PIITypeSSN] {
		matches = append(matches, ps.scanPattern(text, reSSN, PIITypeSSN)...)
	}
	if ps.enabledTypes[PIITypePhone] {
		matches = append(matches, ps.scanPattern(text, rePhone, PIITypePhone)...)
	}
	if ps.enabledTypes[PIITypeAPIKey] {
		// Only flag high-entropy API key candidates
		matches = append(matches, ps.scanAPIKeys(text)...)
	}
	if ps.enabledTypes[PIITypeInternalPkg] {
		matches = append(matches, ps.scanPattern(text, reInternalPkg, PIITypeInternalPkg)...)
	}
	if ps.enabledTypes[PIITypeUserPath] {
		matches = append(matches, ps.scanUserPaths(text)...)
	}
	if ps.enabledTypes[PIITypeURL] {
		matches = append(matches, ps.scanPattern(text, reURL, PIITypeURL)...)
	}

	// Check custom patterns
	for _, cp := range ps.customPatterns {
		if ps.enabledTypes[cp.typ] {
			matches = append(matches, ps.scanPattern(text, cp.pattern, cp.typ)...)
		}
	}

	return matches
}

// Scrub removes detected PII from text, replacing with redaction markers.
func (ps *PIIScrubber) Scrub(text string) (scrubbed string, matches []PIIMatch) {
	matches = ps.Scan(text)

	// Sort matches by position (descending) to avoid index shifting issues
	// For now, we'll use a simpler approach with replacements
	result := text
	redactions := 0

	// Apply each pattern's replacements
	for _, m := range matches {
		redactions++
		redaction := "[REDACTED_" + string(m.Type) + "]"
		result = strings.Replace(result, m.Original, redaction, 1)
	}

	return result, matches
}

// scanPattern scans text with a regex pattern and returns matches.
func (ps *PIIScrubber) scanPattern(text string, pattern *regexp.Regexp, typ PIIType) []PIIMatch {
	var matches []PIIMatch
	locs := pattern.FindAllStringIndex(text, -1)
	for _, loc := range locs {
		matches = append(matches, PIIMatch{
			Type:     typ,
			Position: loc[0],
			Length:   loc[1] - loc[0],
			Original: text[loc[0]:loc[1]],
		})
	}
	return matches
}

// scanUserPaths scans for user paths with special handling.
func (ps *PIIScrubber) scanUserPaths(text string) []PIIMatch {
	var matches []PIIMatch

	// Unix paths
	unixLocs := reUnixUserPath.FindAllStringIndex(text, -1)
	for _, loc := range unixLocs {
		matches = append(matches, PIIMatch{
			Type:     PIITypeUserPath,
			Position: loc[0],
			Length:   loc[1] - loc[0],
			Original: text[loc[0]:loc[1]],
		})
	}

	// Windows paths
	winLocs := reWindowsUserPath.FindAllStringIndex(text, -1)
	for _, loc := range winLocs {
		matches = append(matches, PIIMatch{
			Type:     PIITypeUserPath,
			Position: loc[0],
			Length:   loc[1] - loc[0],
			Original: text[loc[0]:loc[1]],
		})
	}

	return matches
}

// scanAPIKeys scans for potential API keys with entropy filtering.
func (ps *PIIScrubber) scanAPIKeys(text string) []PIIMatch {
	var matches []PIIMatch
	locs := reAPIKey.FindAllStringIndex(text, -1)

	for _, loc := range locs {
		candidate := text[loc[0]:loc[1]]

		// Only flag if it has high entropy (likely a key, not just a word)
		if HighEntropyCheck(candidate, "api_key") {
			matches = append(matches, PIIMatch{
				Type:     PIITypeAPIKey,
				Position: loc[0],
				Length:   loc[1] - loc[0],
				Original: candidate,
			})
		}
	}

	return matches
}

// ScrubPII is a convenience function that scrubs PII using default settings.
func ScrubPII(text string) (string, map[PIIType]int) {
	scrubber := NewPIIScrubber()
	result, matches := scrubber.Scrub(text)

	// Count by type
	counts := make(map[PIIType]int)
	for _, m := range matches {
		counts[m.Type]++
	}

	return result, counts
}
