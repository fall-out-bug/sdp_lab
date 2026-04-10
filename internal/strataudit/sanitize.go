package strataudit

import (
	"regexp"
	"strings"
	"unicode"
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

// SanitizeForPrompt normalizes content and strips injection patterns.
func SanitizeForPrompt(content string) string {
	// NFKC normalization: collapses Unicode homoglyphs into canonical forms
	content = normalizeNFKC(content)

	// Strip zero-width and control characters (except newline/tab)
	content = stripControlChars(content)

	// Apply injection pattern replacements
	for _, pat := range injectionPatterns {
		content = pat.ReplaceAllString(content, "[CONTENT REDACTED]")
	}
	content = regexp.MustCompile(`(\[CONTENT REDACTED]\s*)+`).ReplaceAllString(content, "[CONTENT REDACTED] ")
	return strings.TrimSpace(content)
}

// normalizeNFKC applies Unicode NFKC normalization to collapse homoglyphs.
func normalizeNFKC(s string) string {
	// Use unicode.IsMn check as a proxy — full NFKC requires golang.org/x/text
	// For now, replace known confusable ranges with their ASCII equivalents
	mapped := make([]rune, 0, len(s))
	for _, r := range s {
		// Map Cyrillic lookalikes to Latin
		if repl, ok := confusableMap[r]; ok {
			mapped = append(mapped, repl)
		} else {
			mapped = append(mapped, r)
		}
	}
	return string(mapped)
}

// confusableMap maps common Unicode confusables to their ASCII equivalents.
var confusableMap = map[rune]rune{
	// Cyrillic → Latin
	'\u0430': 'a', // а → a
	'\u0410': 'A', // А → A
	'\u0435': 'e', // е → e
	'\u0415': 'E', // Е → E
	'\u043E': 'o', // о → o
	'\u041E': 'O', // О → O
	'\u0440': 'p', // р → p
	'\u0420': 'P', // Р → P
	'\u0441': 'c', // с → c
	'\u0421': 'C', // С → C
	'\u0443': 'y', // у → y
	'\u0423': 'Y', // У → Y
	'\u0445': 'x', // х → x
	'\u0425': 'X', // Х → X
	// Fullwidth → ASCII
	'\uFF41': 'a', '\uFF42': 'b', '\uFF43': 'c', '\uFF44': 'd', '\uFF45': 'e',
	'\uFF46': 'f', '\uFF47': 'g', '\uFF48': 'h', '\uFF49': 'i', '\uFF4A': 'j',
	'\uFF4B': 'k', '\uFF4C': 'l', '\uFF4D': 'm', '\uFF4E': 'n', '\uFF4F': 'o',
	'\uFF50': 'p', '\uFF51': 'q', '\uFF52': 'r', '\uFF53': 's', '\uFF54': 't',
	'\uFF55': 'u', '\uFF56': 'v', '\uFF57': 'w', '\uFF58': 'x', '\uFF59': 'y',
	'\uFF5A': 'z',
	'\uFF21': 'A', '\uFF22': 'B', '\uFF23': 'C', '\uFF24': 'D', '\uFF25': 'E',
	'\uFF26': 'F', '\uFF27': 'G', '\uFF28': 'H', '\uFF29': 'I', '\uFF2A': 'J',
	'\uFF2B': 'K', '\uFF2C': 'L', '\uFF2D': 'M', '\uFF2E': 'N', '\uFF2F': 'O',
	'\uFF30': 'P', '\uFF31': 'Q', '\uFF32': 'R', '\uFF33': 'S', '\uFF34': 'T',
	'\uFF35': 'U', '\uFF36': 'V', '\uFF37': 'W', '\uFF38': 'X', '\uFF39': 'Y',
	'\uFF3A': 'Z',
}

// stripControlChars removes zero-width and non-printable characters.
func stripControlChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || r == '\r' {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Mn, r) {
			continue
		}
		// Skip zero-width characters
		if r == '\u200B' || r == '\u200C' || r == '\u200D' || r == '\uFEFF' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
