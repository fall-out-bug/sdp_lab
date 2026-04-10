package architect

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

// sanitizePolicy is a strict HTML sanitization policy that allows only
// safe formatting elements. It is initialized once at package load.
var sanitizePolicy *bluemonday.Policy

func init() {
	p := bluemonday.NewPolicy()
	p.AllowStandardAttributes()
	p.AllowElements("b", "i", "em", "strong", "a", "p", "br", "ul", "ol", "li", "code", "pre")
	p.AllowAttrs("href").OnElements("a")
	p.RequireNoFollowOnLinks(true)
	p.AllowURLSchemes("http", "https")
	sanitizePolicy = p
}

// SanitizeField sanitizes a single string value from parsed LLM JSON output.
// Step 1: Render Markdown to HTML (FlagsNone = no raw HTML passthrough).
// Step 2: Sanitize HTML with the custom bluemonday policy.
func SanitizeField(value string) string {
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	opts := html.RendererOptions{Flags: html.FlagsNone}
	renderer := html.NewRenderer(opts)
	htmlBytes := markdown.ToHTML([]byte(value), p, renderer)
	return string(sanitizePolicy.SanitizeBytes(htmlBytes))
}
