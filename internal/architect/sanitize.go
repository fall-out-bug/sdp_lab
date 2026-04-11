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

// SanitizeOutput recursively walks a parsed JSON structure (map[string]interface{},
// []interface{}, string, etc.) and applies SanitizeField to every string value.
// This is the output-side counterpart to the input-side SanitizeForLLM.
// It MUST operate on parsed JSON, never on raw JSON strings.
func SanitizeOutput(parsed interface{}) interface{} {
	switch v := parsed.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			result[key] = SanitizeOutput(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, elem := range v {
			result[i] = SanitizeOutput(elem)
		}
		return result
	case string:
		return SanitizeField(v)
	default:
		// Numbers, bools, nil — return as-is.
		return v
	}
}
