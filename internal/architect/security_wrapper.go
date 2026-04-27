package architect

import (
	"github.com/fall-out-bug/sdp_lab/internal/architect/security"
)

// SanitizeField sanitizes markdown content by converting to HTML and applying
// strict security policies. This is a wrapper around security.SanitizeField.
func SanitizeField(markdown string) string {
	return security.SanitizeField(markdown)
}

// SanitizeOutput recursively sanitizes all string values in parsed JSON data.
// This is a wrapper around security.SanitizeOutput.
func SanitizeOutput(parsed interface{}) interface{} {
	return security.SanitizeOutput(parsed)
}

// ScrubPII scrubs personally identifiable information from text.
// This is a wrapper around security.ScrubPII.
func ScrubPII(text string) (string, map[security.PIIType]int) {
	return security.ScrubPII(text)
}

// SanitizeText sanitizes plain text by removing HTML tags.
// This is a wrapper around security.SanitizeText.
func SanitizeText(text string) string {
	return security.SanitizeText(text)
}

// StripMarkdown removes markdown syntax, leaving plain text.
// This is a wrapper around security.StripMarkdown.
func StripMarkdown(markdown string) string {
	return security.StripMarkdown(markdown)
}

// SanitizeMarkdownLink sanitizes a URL for use in markdown links.
// This is a wrapper around security.SanitizeMarkdownLink.
func SanitizeMarkdownLink(url string) string {
	return security.SanitizeMarkdownLink(url)
}

// ContainsUnsafeHTML checks if text contains potentially unsafe HTML.
// This is a wrapper around security.ContainsUnsafeHTML.
func ContainsUnsafeHTML(text string) bool {
	return security.ContainsUnsafeHTML(text)
}

// GetAllowedTags returns a list of allowed HTML tags.
// This is a wrapper around security.GetAllowedTags.
func GetAllowedTags() []string {
	return security.GetAllowedTags()
}

// GetAllowedURLSchemes returns a list of allowed URL schemes.
// This is a wrapper around security.GetAllowedURLSchemes.
func GetAllowedURLSchemes() []string {
	return security.GetAllowedURLSchemes()
}

